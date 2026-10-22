// Package infrastructure persiste produtos no PostgreSQL.
package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/produtos/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const cols = `id, nome, descricao, estagio, utilidade, garantia, unidade_id, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (domain.Produto, error) {
	var p domain.Produto
	err := row.Scan(&p.ID, &p.Nome, &p.Descricao, &p.Estagio, &p.Utilidade, &p.Garantia, &p.UnidadeID, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case database.IsForeignKeyViolation(err):
		return domain.ErrUnidadeInexistente
	case database.IsCheckViolation(err):
		return domain.ErrEntradaInvalida
	default:
		return fmt.Errorf("produtos: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// List devolve os produtos visíveis no filtro, por nome.
func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Produto, error) {
	rows, err := db.Query(ctx, `SELECT `+cols+` FROM produtos
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
		ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Produto{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// Get busca um produto pelo id.
func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Produto, error) {
	p, err := scan(db.QueryRow(ctx, `SELECT `+cols+` FROM produtos WHERE id = $1`, id))
	if err != nil {
		return domain.Produto{}, mapDB(err)
	}
	return p, nil
}

// Insert grava o produto e devolve a linha persistida.
func (r *Repository) Insert(ctx context.Context, db database.DBTX, p domain.Produto) (domain.Produto, error) {
	out, err := scan(db.QueryRow(ctx, `INSERT INTO produtos (`+cols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+cols, p.ID, p.Nome, p.Descricao, p.Estagio, p.Utilidade, p.Garantia, p.UnidadeID, p.CreatedAt, p.UpdatedAt))
	if err != nil {
		return domain.Produto{}, mapDB(err)
	}
	return out, nil
}

// Update grava os campos editáveis e o estágio.
func (r *Repository) Update(ctx context.Context, db database.DBTX, p domain.Produto) (domain.Produto, error) {
	out, err := scan(db.QueryRow(ctx, `UPDATE produtos
		SET nome = $2, descricao = $3, estagio = $4, utilidade = $5, garantia = $6, unidade_id = $7
		WHERE id = $1 RETURNING `+cols, p.ID, p.Nome, p.Descricao, p.Estagio, p.Utilidade, p.Garantia, p.UnidadeID))
	if err != nil {
		return domain.Produto{}, mapDB(err)
	}
	return out, nil
}
