package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) InsertCredencial(ctx context.Context, db database.DBTX, c domain.Credencial) error {
	_, err := db.Exec(ctx, `INSERT INTO ativos_credenciais (id, nome, unidade_id, token_hash, created_at)
		VALUES ($1,$2,$3,$4,$5)`, c.ID, c.Nome, c.UnidadeID, c.TokenHash, c.CreatedAt)
	return mapDB(err)
}

func (r *Repository) ListCredenciais(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Credencial, error) {
	rows, err := db.Query(ctx, `SELECT id, nome, unidade_id, created_at, revogada_em
		FROM ativos_credenciais
		WHERE revogada_em IS NULL AND ($1 = false OR unidade_id = ANY($2::uuid[]))
		ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Credencial{}
	for rows.Next() {
		var c domain.Credencial
		if err := rows.Scan(&c.ID, &c.Nome, &c.UnidadeID, &c.CreatedAt, &c.RevogadaEm); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, c)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) RevogarCredencial(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `UPDATE ativos_credenciais SET revogada_em = now()
		WHERE id = $1 AND revogada_em IS NULL AND ($2 = false OR unidade_id = ANY($3::uuid[]))`,
		id, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) PorHash(ctx context.Context, db database.DBTX, hash []byte) (domain.Credencial, error) {
	var c domain.Credencial
	err := db.QueryRow(ctx, `SELECT id, nome, unidade_id, created_at
		FROM ativos_credenciais
		WHERE token_hash = $1 AND revogada_em IS NULL`, hash).Scan(&c.ID, &c.Nome, &c.UnidadeID, &c.CreatedAt)
	if err != nil {
		return domain.Credencial{}, mapDB(err)
	}
	return c, nil
}
