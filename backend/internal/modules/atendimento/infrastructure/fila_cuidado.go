package infrastructure

import (
	"context"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) ParaEncerrar(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if limite < 1 {
		limite = 1
	}
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE s.estado IN ('resolvido', 'atendido')
		  AND COALESCE(s.resolvida_em, s.updated_at) <= $1
		ORDER BY COALESCE(s.resolvida_em, s.updated_at), s.id
		LIMIT $2`, ate, limite)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}

func (r *Repository) Paradas(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if limite < 1 {
		limite = 1
	}
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE s.estado IN ('em_atendimento', 'aguardando_terceiro')
		  AND s.updated_at <= $1
		ORDER BY s.updated_at, s.id
		LIMIT $2`, ate, limite)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}
