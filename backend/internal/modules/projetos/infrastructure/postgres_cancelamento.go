package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) SalvarCancelamento(ctx context.Context, db database.DBTX, c *domain.Cancelamento) error {
	c.CreatedAt = time.Now().UTC()
	const q = `INSERT INTO projetos_cancelamentos
		(id, projeto_id, motivo, aprovador_id, aprovador_nome, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(ctx, q, c.ID, c.ProjetoID, c.Motivo, c.AprovadorID, c.AprovadorNome, c.CreatedAt)
	return mapDB(err)
}

func (r *Repository) UltimoCancelamento(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Cancelamento, bool, error) {
	const q = `SELECT id, projeto_id, motivo, aprovador_id, aprovador_nome, created_at
		FROM projetos_cancelamentos WHERE projeto_id = $1
		ORDER BY created_at DESC LIMIT 1`
	var c domain.Cancelamento
	err := db.QueryRow(ctx, q, projetoID).Scan(
		&c.ID, &c.ProjetoID, &c.Motivo, &c.AprovadorID, &c.AprovadorNome, &c.CreatedAt,
	)
	if err != nil {
		if database.IsNoRows(err) {
			return domain.Cancelamento{}, false, nil
		}
		return domain.Cancelamento{}, false, mapDB(err)
	}
	return c, true, nil
}
