package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case database.IsForeignKeyViolation(err):
		return domain.EntradaInvalida{"a unidade não existe"}
	case database.IsCheckViolation(err):
		return domain.EntradaInvalida{"dados fora do cadastro"}
	default:
		return fmt.Errorf("cofre: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

const onde = `($1 = false OR unidade_id = ANY($2::uuid[]))`

func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Selado, error) {
	rows, err := db.Query(ctx, `SELECT id, nome, usuario, unidade_id, versao, dek_embrulhada, segredo_cifrado, created_at, updated_at
		FROM cofre_segredos WHERE `+onde+` ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Selado{}
	for rows.Next() {
		var item domain.Selado
		if err := rows.Scan(&item.ID, &item.Nome, &item.Usuario, &item.UnidadeID, &item.Versao, &item.Dek, &item.Cifra, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) (domain.Selado, error) {
	var item domain.Selado
	err := db.QueryRow(ctx, `SELECT id, nome, usuario, unidade_id, versao, dek_embrulhada, segredo_cifrado, created_at, updated_at
		FROM cofre_segredos WHERE id = $1 AND `+onde, id, f.Restrito, idsOrEmpty(f.Unidades)).
		Scan(&item.ID, &item.Nome, &item.Usuario, &item.UnidadeID, &item.Versao, &item.Dek, &item.Cifra, &item.CreatedAt, &item.UpdatedAt)
	return item, mapDB(err)
}

func (r *Repository) Insert(ctx context.Context, db database.DBTX, item domain.Selado) error {
	_, err := db.Exec(ctx, `INSERT INTO cofre_segredos
		(id, nome, usuario, unidade_id, versao, dek_embrulhada, segredo_cifrado, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		item.ID, item.Nome, item.Usuario, item.UnidadeID, item.Versao, item.Dek, item.Cifra, item.CreatedAt, item.UpdatedAt)
	return mapDB(err)
}

func (r *Repository) Update(ctx context.Context, db database.DBTX, item domain.Selado, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `UPDATE cofre_segredos
		SET nome = $2, usuario = $3, unidade_id = $4, versao = $5, dek_embrulhada = $6, segredo_cifrado = $7, updated_at = $8
		WHERE id = $1 AND ($9 = false OR unidade_id = ANY($10::uuid[]))`,
		item.ID, item.Nome, item.Usuario, item.UnidadeID, item.Versao, item.Dek, item.Cifra, item.UpdatedAt, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `DELETE FROM cofre_segredos WHERE id = $1 AND ($2 = false OR unidade_id = ANY($3::uuid[]))`, id, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}
