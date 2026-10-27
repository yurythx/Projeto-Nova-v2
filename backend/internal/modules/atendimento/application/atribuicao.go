package application

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

// acompanharAtribuicao fecha o trecho de quem estava com o chamado quando
// ele termina e abre de novo quando é reaberto (ADR 073).
func (s *Service) acompanharAtribuicao(ctx context.Context, tx pgx.Tx, prev, next domain.Solicitacao) error {
	fim := func(e string) bool {
		switch e {
		case domain.EstadoResolvido, domain.EstadoAtendido, domain.EstadoEncerrado, domain.EstadoCancelado:
			return true
		}
		return false
	}
	switch {
	case fim(next.Estado) && !fim(prev.Estado):
		return s.repo.FecharAtribuicao(ctx, tx, next.ID, next.UpdatedAt)
	case fim(prev.Estado) && !fim(next.Estado) && next.ResponsavelID != nil:
		return s.repo.AbrirAtribuicao(ctx, tx, next.ID, *next.ResponsavelID, next.UnidadeID, next.UpdatedAt)
	}
	return nil
}
