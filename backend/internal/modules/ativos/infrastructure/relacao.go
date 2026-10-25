package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) ListRelacoes(ctx context.Context, db database.DBTX, origem uuid.UUID) ([]domain.Relacao, error) {
	rows, err := db.Query(ctx, `SELECT r.id, r.origem_id, r.destino_id, a.nome, r.tipo, r.created_at
		FROM ativo_relacoes r JOIN ativos a ON a.id = r.destino_id
		WHERE r.origem_id = $1 ORDER BY r.tipo, a.nome, r.id`, origem)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Relacao{}
	for rows.Next() {
		var item domain.Relacao
		if err := rows.Scan(&item.ID, &item.OrigemID, &item.DestinoID, &item.DestinoNome, &item.Tipo, &item.CriadaEm); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InsertRelacao(ctx context.Context, db database.DBTX, item domain.Relacao) error {
	_, err := db.Exec(ctx, `INSERT INTO ativo_relacoes (id, origem_id, destino_id, tipo) VALUES ($1,$2,$3,$4)`,
		item.ID, item.OrigemID, item.DestinoID, item.Tipo)
	if database.IsUniqueViolation(err) {
		return domain.RelacaoInvalida{"essa relação já existe"}
	}
	return mapDB(err)
}

func (r *Repository) DeleteRelacao(ctx context.Context, db database.DBTX, id, origem uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM ativo_relacoes WHERE id = $1 AND origem_id = $2`, id, origem)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) FechaCiclo(ctx context.Context, db database.DBTX, origem, destino uuid.UUID) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `WITH RECURSIVE cadeia AS (
			SELECT destino_id FROM ativo_relacoes WHERE origem_id = $1 AND tipo = 'depende'
			UNION
			SELECT r.destino_id FROM ativo_relacoes r JOIN cadeia c ON r.origem_id = c.destino_id WHERE r.tipo = 'depende'
		)
		SELECT EXISTS (SELECT 1 FROM cadeia WHERE destino_id = $2)`, destino, origem).Scan(&ok)
	return ok, mapDB(err)
}
