package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

const (
	// esperaConfirmacao é o silêncio depois da conclusão antes do
	// encerramento automático. Quem abriu ainda pode confirmar ou reabrir.
	esperaConfirmacao = 5 * 24 * time.Hour
	// silencioAntigo não avisa: o chamado já passou da janela em que a
	// pessoa ainda acompanha, e um lote antigo não pode encher o sino.
	silencioAntigo = 30 * 24 * time.Hour
	// paradoHa avisa uma vez quem está com o chamado sem movimento.
	paradoHa = 7 * 24 * time.Hour
	loteFila = 40
)

const notaDeEncerramento = "Encerrado automaticamente depois de 5 dias sem confirmação de quem abriu."

// CuidarDaFila encerra o que já foi cumprido e ficou sem confirmação, e
// avisa uma vez o que está parado em atendimento ou com terceiro.
func (s *Service) CuidarDaFila(ctx context.Context) error {
	if err := s.encerrarConfirmadas(ctx); err != nil {
		return err
	}
	return s.avisarParadas(ctx)
}

func (s *Service) encerrarConfirmadas(ctx context.Context) error {
	ate := s.agora().Add(-esperaConfirmacao)
	lista, err := s.repo.ParaEncerrar(ctx, s.pool, ate, loteFila)
	if err != nil {
		return err
	}
	var enviadas []notificacoes.Enviada
	for _, sol := range lista {
		parte, err := s.encerrarUma(ctx, sol)
		if err != nil {
			if s.logger != nil {
				s.logger.WarnContext(ctx, "atendimento: encerrar chamado cumprido falhou",
					slog.Int64("numero", sol.Numero), slog.Any("error", err))
			}
			continue
		}
		enviadas = append(enviadas, parte...)
	}
	s.entregar(ctx, enviadas)
	return nil
}

func (s *Service) encerrarUma(ctx context.Context, sol domain.Solicitacao) ([]notificacoes.Enviada, error) {
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.TravarSolicitacao(ctx, tx, sol.ID); err != nil {
			return err
		}
		atual, err := s.repo.GetSolicitacao(ctx, tx, sol.ID)
		if err != nil {
			return err
		}
		if atual.Estado != domain.EstadoResolvido && atual.Estado != domain.EstadoAtendido {
			return nil
		}
		if err := s.recusarEspelho(ctx, tx, atual.ID); errors.Is(err, domain.ErrEspelho) {
			return nil
		} else if err != nil {
			return err
		}
		next, mudou, err := atual.Transicionar(domain.EstadoEncerrado, domain.PapelTecnico)
		if err != nil || !mudou {
			return err
		}
		if err := s.repo.UpdateSolicitacao(ctx, tx, next); err != nil {
			return err
		}
		if err := s.acompanharAtribuicao(ctx, tx, atual, next); err != nil {
			return err
		}
		if _, err := s.repo.InsertPassagem(ctx, tx, domain.Passagem{
			SolicitacaoID: next.ID, De: &atual.Estado, Para: next.Estado, Em: next.UpdatedAt,
		}); err != nil {
			return err
		}
		if _, err := s.repo.InsertComentario(ctx, tx, domain.Comentario{
			ID: uuid.New(), SolicitacaoID: next.ID, Texto: notaDeEncerramento, Interno: true, CreatedAt: next.UpdatedAt,
		}); err != nil {
			return err
		}
		payload := payloadDe(next)
		payload["de"] = atual.Estado
		payload["automatico"] = true
		if err := s.registrar(ctx, tx, EventoSolicitacaoEstado, "solicitacao", next.ID, payload, next, atual); err != nil {
			return err
		}
		if s.agora().Sub(quandoConcluiu(atual)) >= silencioAntigo || s.avisos == nil {
			return nil
		}
		enviadas, err = s.avisar(ctx, tx, next, []uuid.UUID{next.SolicitanteID}, uuid.Nil, "encerrada-automatica",
			fmt.Sprintf("Solicitação #%d encerrada", next.Numero), notaDeEncerramento)
		return err
	})
	return enviadas, err
}

func quandoConcluiu(s domain.Solicitacao) time.Time {
	if s.ResolvidaEm != nil {
		return *s.ResolvidaEm
	}
	return s.UpdatedAt
}

func (s *Service) avisarParadas(ctx context.Context) error {
	if s.avisos == nil {
		return nil
	}
	lista, err := s.repo.Paradas(ctx, s.pool, s.agora().Add(-paradoHa), loteFila)
	if err != nil {
		return err
	}
	var enviadas []notificacoes.Enviada
	for _, sol := range lista {
		destinos, err := s.destinosDaParada(ctx, sol)
		if err != nil {
			return err
		}
		parte, err := notificacoes.Registrar(ctx, s.pool, destinos, notificacoes.Nova{
			Modulo:   "atendimento",
			Titulo:   fmt.Sprintf("Solicitação #%d parada", sol.Numero),
			Mensagem: sol.OfertaNome + ": " + sol.Titulo + ". " + textoDaParada(sol.Estado),
			Link:     "/atendimento/solicitacoes/" + sol.ID.String(),
			Chave:    "atendimento:" + sol.ID.String() + ":parado",
		})
		if err != nil {
			return err
		}
		enviadas = append(enviadas, parte...)
	}
	s.entregar(ctx, enviadas)
	return nil
}

func (s *Service) destinosDaParada(ctx context.Context, sol domain.Solicitacao) ([]uuid.UUID, error) {
	if sol.ResponsavelID != nil {
		return []uuid.UUID{*sol.ResponsavelID}, nil
	}
	membros, err := s.repo.Responsaveis(ctx, s.pool, sol.UnidadeID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(membros))
	for _, m := range membros {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

func textoDaParada(estado string) string {
	if estado == domain.EstadoAguardandoTerceiro {
		return "Está aguardando terceiro sem movimento há mais de 7 dias."
	}
	return "Está em atendimento sem movimento há mais de 7 dias."
}
