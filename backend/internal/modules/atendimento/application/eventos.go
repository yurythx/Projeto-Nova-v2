package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// ResultadoEventos conta o que cada alerta do webhook virou.
type ResultadoEventos struct {
	Abertos   int `json:"abertos"`
	Anotados  int `json:"anotados"`
	Ignorados int `json:"ignorados"`
}

// ConfiguracaoEventos é a oferta em que os alertas abrem incidente.
type ConfiguracaoEventos struct {
	OfertaID *uuid.UUID `json:"oferta_id"`
}

const horaDoAlerta = "02/01 15:04 MST"

// ReceberAlertas abre ou anota um chamado por alerta (ADR 067). Sem oferta
// escolhida, recusa tudo: o Alertmanager tenta de novo e nada se perde.
func (s *Service) ReceberAlertas(ctx context.Context, alertas []domain.Alerta) (ResultadoEventos, error) {
	for _, a := range alertas {
		if err := a.Validar(); err != nil {
			return ResultadoEventos{}, MapError(err)
		}
	}
	ofertaID, err := s.repo.OfertaDosEventos(ctx, s.pool)
	if err != nil {
		return ResultadoEventos{}, MapError(err)
	}
	if ofertaID == nil {
		return ResultadoEventos{}, MapError(domain.ErrEventosSemOferta)
	}
	o, err := s.repo.GetOferta(ctx, s.pool, *ofertaID)
	if err != nil {
		return ResultadoEventos{}, MapError(err)
	}
	var r ResultadoEventos
	// Um alerta por transação: um que falha não desfaz os outros, e o
	// reenvio do Alertmanager encontra os já tratados pela fingerprint.
	for _, a := range alertas {
		efeito, err := s.receberAlerta(ctx, o, a)
		if err != nil {
			return r, MapError(err)
		}
		switch efeito {
		case "aberto":
			r.Abertos++
		case "anotado":
			r.Anotados++
		default:
			r.Ignorados++
		}
	}
	return r, nil
}

func (s *Service) receberAlerta(ctx context.Context, o domain.Oferta, a domain.Alerta) (string, error) {
	efeito := "ignorado"
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		ev, err := s.repo.TravarEvento(ctx, tx, a.Fingerprint)
		if err != nil && !errors.Is(err, domain.ErrNaoEncontrado) {
			return err
		}
		if err == nil {
			sol, err := s.repo.GetSolicitacao(ctx, tx, ev.SolicitacaoID)
			if err != nil {
				return err
			}
			if !sol.Finalizada() {
				fuso := s.fusoDa(ctx, sol.UnidadeID)
				texto := fmt.Sprintf("O alerta %s disparou de novo às %s.", a.Nome(), a.StartsAt.In(fuso).Format(horaDoAlerta))
				if a.Status == domain.AlertaNormalizou {
					texto = fmt.Sprintf("O alerta %s normalizou às %s. Confira a causa antes de concluir: o sintoma sumir não prova que ela foi tratada.",
						a.Nome(), a.EndsAt.In(fuso).Format(horaDoAlerta))
				}
				efeito = "anotado"
				enviadas, err = s.notaDoMonitoramento(ctx, tx, sol, texto)
				return err
			}
		}
		if a.Status != domain.AlertaDisparando {
			return nil
		}
		nova, err := domain.NovaSolicitacao(o, a.Titulo(), a.Descricao(), domain.ContaMonitoramento)
		if err != nil {
			return err
		}
		severidade := func(sol *domain.Solicitacao) {
			sol.Impacto, sol.Urgencia = 2, 2
			if a.Critico() {
				sol.Impacto, sol.Urgencia = 3, 3
			}
		}
		aberta, avisos, err := s.abrirNa(ctx, tx, nova, auth.Identity{UserID: domain.ContaMonitoramento}, severidade)
		if err != nil {
			return err
		}
		efeito, enviadas = "aberto", avisos
		return s.repo.SalvarEvento(ctx, tx, domain.EventoVinculado{
			Fingerprint: a.Fingerprint, SolicitacaoID: aberta.ID, Alerta: a.Nome(), UltimoDisparo: a.StartsAt.UTC(),
		})
	})
	if err != nil {
		return "", err
	}
	s.entregar(ctx, enviadas)
	return efeito, nil
}

// notaDoMonitoramento registra a nota interna do alerta e avisa quem é
// responsável, como uma nota da equipe.
func (s *Service) notaDoMonitoramento(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, texto string) ([]notificacoes.Enviada, error) {
	nota, err := domain.NovoComentario(sol, domain.ContaMonitoramento, texto, true, true)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.InsertComentario(ctx, tx, nota)
	if err != nil {
		return nil, err
	}
	payload := payloadDe(sol)
	payload["comentario_id"], payload["interno"] = c.ID.String(), true
	if err := s.registrar(ctx, tx, EventoSolicitacaoComentada, "solicitacao", sol.ID, payload, c, nil); err != nil {
		return nil, err
	}
	if sol.ResponsavelID == nil {
		return nil, nil
	}
	return s.avisar(ctx, tx, sol, []uuid.UUID{*sol.ResponsavelID}, domain.ContaMonitoramento, "comentario:"+c.ID.String(),
		fmt.Sprintf("Alerta na solicitação #%d", sol.Numero), texto)
}

// ConfiguracaoDosEventos devolve a oferta dos alertas (gestão do atendimento).
func (s *Service) ConfiguracaoDosEventos(ctx context.Context, identity auth.Identity) (ConfiguracaoEventos, error) {
	if !auth.CoverageOf(identity, auth.PermAtendimentoManage).All {
		return ConfiguracaoEventos{}, MapError(domain.ErrForaDoEscopo)
	}
	o, err := s.repo.OfertaDosEventos(ctx, s.pool)
	return ConfiguracaoEventos{OfertaID: o}, MapError(err)
}

// ConfigurarEventos escolhe (ou limpa) a oferta dos alertas. Os alertas são
// da instituição: só quem gere o atendimento em toda ela escolhe. A
// oferta precisa ser de incidente e estar ativa.
func (s *Service) ConfigurarEventos(ctx context.Context, identity auth.Identity, oferta *uuid.UUID) (ConfiguracaoEventos, error) {
	if !auth.CoverageOf(identity, auth.PermAtendimentoManage).All {
		return ConfiguracaoEventos{}, MapError(domain.ErrForaDoEscopo)
	}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if oferta != nil {
			o, err := s.repo.GetOferta(ctx, tx, *oferta)
			if errors.Is(err, domain.ErrNaoEncontrado) {
				return domain.ErrOfertaInexistente
			}
			if err != nil {
				return err
			}
			if o.Tipo != domain.TipoIncidente || !o.Ativa {
				return domain.ErrEntradaInvalida
			}
		}
		if err := s.repo.SalvarOfertaDosEventos(ctx, tx, oferta, identity.UserID, s.agora().UTC()); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "atendimento.eventos.configurados", "atendimento_eventos", "config", nil, ConfiguracaoEventos{OfertaID: oferta}))
	})
	if err != nil {
		return ConfiguracaoEventos{}, MapError(err)
	}
	return ConfiguracaoEventos{OfertaID: oferta}, nil
}
