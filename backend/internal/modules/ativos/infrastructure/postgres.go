// Package infrastructure persiste ativos no PostgreSQL.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const cols = `id, nome, tipo, estado, identificador, descricao, produto_id, unidade_id, origem, inventario, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (domain.Ativo, error) {
	var a domain.Ativo
	var raw []byte
	err := row.Scan(&a.ID, &a.Nome, &a.Tipo, &a.Estado, &a.Identificador, &a.Descricao, &a.ProdutoID, &a.UnidadeID, &a.Origem, &raw, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return domain.Ativo{}, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a.Inventario); err != nil {
			return domain.Ativo{}, fmt.Errorf("ativos: inventário ilegível: %w", err)
		}
	}
	return a, nil
}

func inventarioJSON(inv *domain.Inventario) ([]byte, error) {
	if inv == nil {
		return nil, nil
	}
	return json.Marshal(inv)
}

func mapDB(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case errors.As(err, &pgErr) && strings.Contains(pgErr.ConstraintName, "uq_ativos_identificador"):
		return domain.ErrIdentificadorRepetido
	case database.IsForeignKeyViolation(err):
		return domain.ErrUnidadeInexistente
	case database.IsCheckViolation(err):
		return domain.ErrEntradaInvalida
	default:
		return fmt.Errorf("ativos: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// List devolve os ativos do filtro, por nome.
func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Ativo, error) {
	rows, err := db.Query(ctx, `SELECT `+cols+` FROM ativos
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
			AND ($3::uuid IS NULL OR produto_id = $3)
		ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades), f.ProdutoID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Ativo{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// Get busca um ativo.
func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Ativo, error) {
	a, err := scan(db.QueryRow(ctx, `SELECT `+cols+` FROM ativos WHERE id = $1`, id))
	if err != nil {
		return domain.Ativo{}, mapDB(err)
	}
	return a, nil
}

// Insert grava o ativo.
func (r *Repository) Insert(ctx context.Context, db database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	if a.Origem == "" {
		a.Origem = domain.OrigemManual
	}
	inv, err := inventarioJSON(a.Inventario)
	if err != nil {
		return domain.Ativo{}, err
	}
	out, err := scan(db.QueryRow(ctx, `INSERT INTO ativos (`+cols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+cols,
		a.ID, a.Nome, a.Tipo, a.Estado, a.Identificador, a.Descricao, a.ProdutoID, a.UnidadeID, a.Origem, inv, a.CreatedAt, a.UpdatedAt))
	if err != nil {
		return domain.Ativo{}, mapDB(err)
	}
	return out, nil
}

// Update grava a edição.
func (r *Repository) Update(ctx context.Context, db database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	inv, err := inventarioJSON(a.Inventario)
	if err != nil {
		return domain.Ativo{}, err
	}
	out, err := scan(db.QueryRow(ctx, `UPDATE ativos
		SET nome = $2, tipo = $3, estado = $4, identificador = $5, descricao = $6, produto_id = $7, unidade_id = $8, inventario = $9
		WHERE id = $1 RETURNING `+cols,
		a.ID, a.Nome, a.Tipo, a.Estado, a.Identificador, a.Descricao, a.ProdutoID, a.UnidadeID, inv))
	if err != nil {
		return domain.Ativo{}, mapDB(err)
	}
	return out, nil
}

func (r *Repository) PorIdentificador(ctx context.Context, db database.DBTX, identificador string) (domain.Ativo, error) {
	a, err := scan(db.QueryRow(ctx, `SELECT `+cols+` FROM ativos WHERE identificador = $1`, identificador))
	if err != nil {
		return domain.Ativo{}, mapDB(err)
	}
	return a, nil
}
