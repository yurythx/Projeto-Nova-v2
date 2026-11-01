package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const riscoCols = `id, codigo, titulo, descricao, categoria, ativo_id, unidade_id, probabilidade, impacto, severidade, estrategia, plano_mitigacao, mudanca_id, projeto_id, responsavel_id, responsavel_nome, status, created_at, updated_at`

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func scanRisco(row interface{ Scan(...any) error }) (domain.Risco, error) {
	var r domain.Risco
	err := row.Scan(
		&r.ID, &r.Codigo, &r.Titulo, &r.Descricao, &r.Categoria, &r.AtivoID,
		&r.UnidadeID, &r.Probabilidade, &r.Impacto, &r.Severidade, &r.Estrategia,
		&r.PlanoMitigacao, &r.MudancaID, &r.ProjetoID, &r.ResponsavelID,
		&r.ResponsavelNome, &r.Status, &r.CreatedAt, &r.UpdatedAt,
	)
	return r, mapDB(err)
}

func mapDB(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRiscoNaoEncontrado
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrRiscoInvalido
	}
	return err
}

func (r *Repository) ListRiscos(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Risco, error) {
	query := `SELECT ` + riscoCols + ` FROM riscos WHERE 1=1`
	var args []any
	idx := 1

	if len(f.Unidades) > 0 {
		query += ` AND unidade_id = ANY($` + string(rune('0'+idx)) + `)`
		args = append(args, f.Unidades)
		idx++
	}
	if f.Categoria != "" {
		query += ` AND categoria = $` + string(rune('0'+idx))
		args = append(args, f.Categoria)
		idx++
	}
	if f.Status != "" {
		query += ` AND status = $` + string(rune('0'+idx))
		args = append(args, f.Status)
		idx++
	}

	query += ` ORDER BY severidade DESC, created_at DESC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Risco
	for rows.Next() {
		item, err := scanRisco(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, rows.Err()
}

func (r *Repository) GetRisco(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Risco, error) {
	return scanRisco(db.QueryRow(ctx, `SELECT `+riscoCols+` FROM riscos WHERE id = $1`, id))
}

func (r *Repository) InsertRisco(ctx context.Context, db database.DBTX, item domain.Risco) (domain.Risco, error) {
	out, err := scanRisco(db.QueryRow(ctx, `INSERT INTO riscos
		(id, codigo, titulo, descricao, categoria, ativo_id, unidade_id, probabilidade, impacto, estrategia, plano_mitigacao, mudanca_id, projeto_id, responsavel_id, responsavel_nome, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING `+riscoCols,
		item.ID, item.Codigo, item.Titulo, item.Descricao, item.Categoria, item.AtivoID,
		item.UnidadeID, item.Probabilidade, item.Impacto, item.Estrategia,
		item.PlanoMitigacao, item.MudancaID, item.ProjetoID, item.ResponsavelID,
		item.ResponsavelNome, item.Status, item.CreatedAt, item.UpdatedAt,
	))
	return out, err
}

func (r *Repository) UpdateRisco(ctx context.Context, db database.DBTX, item domain.Risco) (domain.Risco, error) {
	out, err := scanRisco(db.QueryRow(ctx, `UPDATE riscos
		SET codigo = $2, titulo = $3, descricao = $4, categoria = $5, ativo_id = $6,
		    unidade_id = $7, probabilidade = $8, impacto = $9, estrategia = $10,
		    plano_mitigacao = $11, mudanca_id = $12, projeto_id = $13, responsavel_id = $14,
		    responsavel_nome = $15, status = $16, updated_at = now()
		WHERE id = $1
		RETURNING `+riscoCols,
		item.ID, item.Codigo, item.Titulo, item.Descricao, item.Categoria, item.AtivoID,
		item.UnidadeID, item.Probabilidade, item.Impacto, item.Estrategia,
		item.PlanoMitigacao, item.MudancaID, item.ProjetoID, item.ResponsavelID,
		item.ResponsavelNome, item.Status,
	))
	return out, err
}

func (r *Repository) DeleteRisco(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM riscos WHERE id = $1`, id)
	return mapDB(err)
}

