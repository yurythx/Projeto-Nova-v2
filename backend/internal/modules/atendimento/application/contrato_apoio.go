package application

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// EventoSolicitacaoContrato registra o vínculo com o contrato de apoio.
const EventoSolicitacaoContrato = "atendimento.solicitacao.contrato"

// ContratosDeApoio lista os contratos vigentes que quem atende pode apontar.
func (s *Service) ContratosDeApoio(ctx context.Context, identity auth.Identity) ([]domain.ContratoDeApoio, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	if s.contratos == nil {
		return nil, MapError(domain.ErrContratosDesligado)
	}
	lista, err := s.contratos.Listar(ctx, identity)
	return lista, MapError(err)
}

// VincularContrato aponta o chamado para o contrato do fornecedor que ele
// espera, ou limpa (nil). O rótulo e o prazo são copiados agora: o que
// vale para a medição é o combinado quando o chamado foi ao fornecedor.
func (s *Service) VincularContrato(ctx context.Context, identity auth.Identity, id uuid.UUID, contrato *uuid.UUID) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrForaDoEscopo
		}
		if sol.Finalizada() {
			return nil, domain.ErrSolicitacaoFinalizada
		}
		var c *domain.ContratoDeApoio
		if contrato != nil {
			if s.contratos == nil {
				return nil, domain.ErrContratosDesligado
			}
			achado, err := s.contratos.Obter(ctx, identity, *contrato)
			if errors.Is(err, domain.ErrContratosDesligado) {
				return nil, err
			}
			if err != nil {
				return nil, domain.ErrContratoAusente
			}
			c = &achado
		}
		if err := s.repo.SetContratoApoio(ctx, tx, sol.ID, c, s.agora().UTC()); err != nil {
			return nil, err
		}
		payload := payloadDe(sol)
		if c != nil {
			payload["contrato_apoio_id"] = c.ID.String()
		}
		return nil, s.registrar(ctx, tx, EventoSolicitacaoContrato, "solicitacao", sol.ID, payload, c, nil)
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}
