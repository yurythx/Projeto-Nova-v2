package application

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// MetasXLA devolve as metas de experiência a quem acompanha os indicadores.
func (s *Service) MetasXLA(ctx context.Context, identity auth.Identity) (domain.MetasXLA, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return domain.MetasXLA{}, MapError(domain.ErrForaDoEscopo)
	}
	m, err := s.repo.MetasXLA(ctx, s.pool)
	return m, MapError(err)
}

// SalvarMetasXLA troca as metas. São da instituição: só quem gere o
// atendimento em toda ela as define, e a troca vai para a auditoria.
func (s *Service) SalvarMetasXLA(ctx context.Context, identity auth.Identity, m domain.MetasXLA) (domain.MetasXLA, error) {
	if !auth.CoverageOf(identity, auth.PermAtendimentoManage).All {
		return domain.MetasXLA{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := m.Validar(); err != nil {
		return domain.MetasXLA{}, MapError(err)
	}
	agora := s.agora().UTC()
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		antes, err := s.repo.MetasXLA(ctx, tx)
		if err != nil {
			return err
		}
		if err := s.repo.SalvarMetasXLA(ctx, tx, m, identity.UserID, agora); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "atendimento.xla.metas", "atendimento_xla", "metas", antes, m))
	})
	if err != nil {
		return domain.MetasXLA{}, MapError(err)
	}
	m.UpdatedAt = &agora
	return m, nil
}
