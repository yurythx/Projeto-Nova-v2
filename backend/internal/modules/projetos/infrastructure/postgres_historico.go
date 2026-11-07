package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) ListarHistorico(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Marco, error) {
	rows, err := db.Query(ctx, `
		SELECT a.id, a.action,
			COALESCE(NULLIF(btrim(u.display_name), ''), u.username, ''),
			a.created_at, a.diff_before, a.diff_after
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.resource_type = 'projeto' AND a.resource_id = $1
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT 80`, projetoID.String())
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	var lista []domain.Marco
	for rows.Next() {
		var m domain.Marco
		if err := rows.Scan(&m.ID, &m.Acao, &m.AutorNome, &m.Quando, &m.Antes, &m.Depois); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, m)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	if lista == nil {
		lista = []domain.Marco{}
	}
	return lista, nil
}
