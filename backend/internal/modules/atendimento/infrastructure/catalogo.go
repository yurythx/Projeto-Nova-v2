package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const categoriaCols = `id, nome, descricao, icone, ordem, ativa, created_at, updated_at`

func scanCategoria(row interface{ Scan(...any) error }) (domain.Categoria, error) {
	var c domain.Categoria
	err := row.Scan(&c.ID, &c.Nome, &c.Descricao, &c.Icone, &c.Ordem, &c.Ativa, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// ListCategorias devolve as categorias em ordem e, no empate, por nome.
func (r *Repository) ListCategorias(ctx context.Context, db database.DBTX, soAtivas bool) ([]domain.Categoria, error) {
	rows, err := db.Query(ctx, `SELECT `+categoriaCols+` FROM atendimento_categorias
		WHERE ativa OR NOT $1 ORDER BY ordem, nome, id`, soAtivas)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanCategoria)
}

func (r *Repository) GetCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Categoria, error) {
	c, err := scanCategoria(db.QueryRow(ctx, `SELECT `+categoriaCols+` FROM atendimento_categorias WHERE id = $1`, id))
	return c, mapDB(err)
}

func (r *Repository) InsertCategoria(ctx context.Context, db database.DBTX, c domain.Categoria) (domain.Categoria, error) {
	out, err := scanCategoria(db.QueryRow(ctx, `INSERT INTO atendimento_categorias (`+categoriaCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+categoriaCols,
		c.ID, c.Nome, c.Descricao, c.Icone, c.Ordem, c.Ativa, c.CreatedAt, c.UpdatedAt))
	return out, mapDB(err)
}

func (r *Repository) UpdateCategoria(ctx context.Context, db database.DBTX, c domain.Categoria) (domain.Categoria, error) {
	out, err := scanCategoria(db.QueryRow(ctx, `UPDATE atendimento_categorias
		SET nome = $2, descricao = $3, icone = $4, ordem = $5, ativa = $6, updated_at = now()
		WHERE id = $1 RETURNING `+categoriaCols, c.ID, c.Nome, c.Descricao, c.Icone, c.Ordem, c.Ativa))
	return out, mapDB(err)
}

// DeleteCategoria apaga; o serviço já conferiu que ela existe.
func (r *Repository) DeleteCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM atendimento_categorias WHERE id = $1`, id)
	return mapDB(err)
}
