package application

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Eventos da aprovação em etapas (ADR 068).
const (
	EventoOfertaEtapas        = "atendimento.oferta.etapas"
	EventoAprovacaoDecidida   = "atendimento.solicitacao.aprovacao"
	LimiteAguardandoAprovacao = 100
)

// EtapasDaOferta devolve as etapas de aprovação da oferta (quem gere a unidade).
func (s *Service) EtapasDaOferta(ctx context.Context, identity auth.Identity, oferta uuid.UUID) ([]domain.EtapaAprovacao, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return nil, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return nil, MapError(domain.ErrNaoEncontrado)
	}
	etapas, err := s.repo.EtapasDaOferta(ctx, s.pool, oferta)
	return etapas, MapError(err)
}

// SalvarEtapas troca as etapas da oferta. Os aprovadores precisam existir
// e estar ativos. Vale para os pedidos novos: o que já espera aprovação
// segue a etapa em que está.
func (s *Service) SalvarEtapas(ctx context.Context, identity auth.Identity, oferta uuid.UUID, etapas []domain.EtapaAprovacao) ([]domain.EtapaAprovacao, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return nil, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return nil, MapError(domain.ErrNaoEncontrado)
	}
	validas, err := domain.ValidarEtapas(o.Tipo, etapas)
	if err != nil {
		return nil, MapError(err)
	}
	pessoas := []uuid.UUID{}
	for _, e := range validas {
		for _, a := range e.Aprovadores {
			if !slices.Contains(pessoas, a) {
				pessoas = append(pessoas, a)
			}
		}
	}
	if len(pessoas) > 0 {
		n, err := s.repo.ContarAtivos(ctx, s.pool, pessoas)
		if err != nil {
			return nil, MapError(err)
		}
		if n != len(pessoas) {
			return nil, MapError(domain.ErrEtapaInvalida)
		}
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SalvarEtapas(ctx, tx, oferta, validas); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoOfertaEtapas, "oferta", oferta, validas, nil)
	})
	if err != nil {
		return nil, MapError(err)
	}
	return validas, nil
}

// AguardandoMinhaAprovacao lista os pedidos que esperam a decisão de quem
// consulta, na etapa atual.
func (s *Service) AguardandoMinhaAprovacao(ctx context.Context, identity auth.Identity) ([]domain.Solicitacao, error) {
	lista, err := s.repo.AguardandoAprovacaoDe(ctx, s.pool, identity.UserID, LimiteAguardandoAprovacao)
	return lista, MapError(err)
}

// DecidirAprovacao grava a decisão de quem aprova a etapa atual. A última
// aprovação entrega o pedido à fila e avisa a equipe; a recusa cancela,
// com o motivo como resposta a quem pediu.
func (s *Service) DecidirAprovacao(ctx context.Context, identity auth.Identity, id uuid.UUID, aprovar bool, motivo string) (domain.Solicitacao, error) {
	var out domain.Solicitacao
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.TravarSolicitacao(ctx, tx, id); err != nil {
			return err
		}
		prev, err := s.repo.GetSolicitacao(ctx, tx, id)
		if err != nil {
			return err
		}
		etapas, err := s.repo.EtapasDaOferta(ctx, tx, prev.OfertaID)
		if err != nil {
			return err
		}
		next, a, liberada, err := prev.Decidir(etapas, identity.UserID, aprovar, motivo, s.agora().UTC())
		if err != nil {
			return err
		}
		if err := s.repo.InsertAprovacao(ctx, tx, a); err != nil {
			return err
		}
		if err := s.registrar(ctx, tx, EventoAprovacaoDecidida, "solicitacao", id,
			map[string]any{"id": id.String(), "etapa": a.Etapa, "decisao": a.Decisao}, a, nil); err != nil {
			return err
		}
		out = next
		switch {
		case !aprovar:
			// A resposta vai antes do cancelamento: depois, o pedido não
			// recebe comentário.
			resposta, err := domain.NovoComentario(prev, identity.UserID,
				fmt.Sprintf("Pedido recusado na etapa %q: %s", a.EtapaNome, a.Motivo), false, false)
			if err != nil {
				return err
			}
			if _, err := s.repo.InsertComentario(ctx, tx, resposta); err != nil {
				return err
			}
			enviadas, err = s.gravarPassagem(ctx, tx, identity, prev, next, true)
			return err
		case liberada:
			if enviadas, err = s.gravarPassagem(ctx, tx, identity, prev, next, true); err != nil {
				return err
			}
			equipe, err := s.destinosDaAbertura(ctx, tx, next, nil)
			if err != nil {
				return err
			}
			avisos, err := s.avisar(ctx, tx, next, equipe, identity.UserID, "aprovada",
				fmt.Sprintf("Nova solicitação #%d (aprovada)", next.Numero), next.OfertaNome+": "+next.Titulo)
			if err != nil {
				return err
			}
			tarefas, err := s.criarTarefas(ctx, tx, next, identity.UserID)
			enviadas = append(append(enviadas, avisos...), tarefas...)
			return err
		default:
			if err := s.repo.UpdateSolicitacao(ctx, tx, next); err != nil {
				return err
			}
			enviadas, err = s.avisarAprovadores(ctx, tx, next, etapas[*next.EtapaAtual], identity.UserID)
			return err
		}
	})
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	s.entregar(ctx, enviadas)
	return out, nil
}

// avisarAprovadores avisa quem aprova a etapa, menos quem pediu.
func (s *Service) avisarAprovadores(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, etapa domain.EtapaAprovacao, autor uuid.UUID) ([]notificacoes.Enviada, error) {
	destinos := []uuid.UUID{}
	for _, a := range etapa.Aprovadores {
		if a != sol.SolicitanteID {
			destinos = append(destinos, a)
		}
	}
	return s.avisarEm(ctx, tx, sol, destinos, autor, fmt.Sprintf("aprovacao:%d", etapa.Ordem),
		fmt.Sprintf("Pedido #%d aguardando sua aprovação (%s)", sol.Numero, etapa.Nome), sol.OfertaNome+": "+sol.Titulo,
		"/atendimento/aprovacoes")
}
