package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Avaliar grava a nota de quem abriu. A segunda avaliação da mesma
// solicitação é recusada. O responsável, quando houver, recebe o aviso.
func (s *Service) Avaliar(ctx context.Context, identity auth.Identity, id uuid.UUID, nota int, comentario string) (domain.Avaliacao, error) {
	return s.AvaliarCom(ctx, identity, id, nota, nil, comentario)
}

// AvaliarCom grava a nota com a resposta opcional a "foi fácil?" (ADR 069).
func (s *Service) AvaliarCom(ctx context.Context, identity auth.Identity, id uuid.UUID, nota int, facilidade *int, comentario string) (domain.Avaliacao, error) {
	var salva domain.Avaliacao
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelSolicitante) {
			return nil, domain.ErrAvaliacaoAlheia
		}
		nova, err := domain.NovaAvaliacao(sol, identity.UserID, nota, comentario)
		if err != nil {
			return nil, err
		}
		if nova, err = nova.ComFacilidade(facilidade); err != nil {
			return nil, err
		}
		if salva, err = s.repo.InsertAvaliacao(ctx, tx, nova); err != nil {
			return nil, err
		}
		payload := payloadDe(sol)
		payload["nota"] = salva.Nota
		if err := s.registrar(ctx, tx, EventoSolicitacaoAvaliada, "solicitacao", sol.ID, payload, salva, nil); err != nil {
			return nil, err
		}
		var destinos []uuid.UUID
		if sol.ResponsavelID != nil {
			destinos = []uuid.UUID{*sol.ResponsavelID}
		}
		return s.avisar(ctx, tx, sol, destinos, identity.UserID, "avaliacao:"+salva.ID.String(),
			fmt.Sprintf("Avaliação da solicitação #%d: %d de 5", sol.Numero, salva.Nota), sol.Titulo)
	})
	if err != nil {
		return domain.Avaliacao{}, err
	}
	return salva, nil
}

// ConcluirPorMudanca conclui a solicitação na transação da mudança.
// Já concluída ou encerrada não é erro. Estado que a equipe não pode
// concluir daqui devolve NaoConcluivel e a mudança inteira recua.
func (s *Service) ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, numero int64, titulo, texto string) (func(context.Context), error) {
	if err := s.repo.TravarSolicitacao(ctx, tx, id); err != nil {
		return nil, err
	}
	sol, err := s.repo.GetSolicitacao(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if sol.Estado == domain.EstadoResolvido || sol.Estado == domain.EstadoAtendido || sol.Estado == domain.EstadoEncerrado {
		return nil, nil
	}
	fim := domain.Conclusao(sol.Tipo)
	next, _, err := sol.Transicionar(fim, domain.PapelTecnico)
	if err != nil {
		return nil, &domain.NaoConcluivel{Numero: sol.Numero, Estado: sol.Estado, Causa: err}
	}
	quem := auth.Identity{UserID: autor}
	enviadas, err := s.gravarPassagem(ctx, tx, quem, sol, next, true)
	if errors.Is(err, domain.ErrTarefasPendentes) {
		return nil, &domain.NaoConcluivel{Numero: sol.Numero, Estado: sol.Estado, Causa: err}
	}
	if err != nil {
		return nil, err
	}
	comentario, err := domain.NovoComentario(next, autor, textoDaMudanca(numero, titulo, texto), false, true)
	if err != nil {
		return nil, err
	}
	salvo, err := s.repo.InsertComentario(ctx, tx, comentario)
	if err != nil {
		return nil, err
	}
	payload := payloadDe(next)
	payload["comentario_id"] = salvo.ID.String()
	payload["origem"] = "mudanca"
	if err := s.registrar(ctx, tx, EventoSolicitacaoComentada, "solicitacao", next.ID, payload, salvo, nil); err != nil {
		return nil, err
	}
	aviso, err := s.avisar(ctx, tx, next, []uuid.UUID{next.SolicitanteID}, autor, "comentario:"+salvo.ID.String(),
		fmt.Sprintf("Nova resposta na solicitação #%d", next.Numero), next.Titulo)
	if err != nil {
		return nil, err
	}
	enviadas = append(enviadas, aviso...)
	return func(ctx context.Context) { s.entregar(ctx, enviadas) }, nil
}

func textoDaMudanca(numero int64, titulo, texto string) string {
	base := fmt.Sprintf("Concluído com a mudança #%d (%s).", numero, strings.TrimSpace(titulo))
	if extra := strings.TrimSpace(texto); extra != "" {
		base += " " + extra
	}
	if utf8.RuneCountInString(base) > domain.TextoMax {
		base = string([]rune(base)[:domain.TextoMax])
	}
	return base
}
