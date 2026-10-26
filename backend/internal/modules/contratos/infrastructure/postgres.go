package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
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
		return domain.EntradaInvalida{"o fornecedor ou a unidade não existe, ou ainda há contrato ou licença ligado a este fornecedor"}
	case database.IsCheckViolation(err):
		return domain.EntradaInvalida{"dados fora do cadastro"}
	default:
		return fmt.Errorf("contratos: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

const onde = `($1 = false OR unidade_id = ANY($2::uuid[]))`

func (r *Repository) ListFornecedores(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Fornecedor, error) {
	rows, err := db.Query(ctx, `SELECT id, nome, email, telefone, unidade_id, created_at, updated_at
		FROM contratos_fornecedores WHERE `+onde+` ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Fornecedor{}
	for rows.Next() {
		var item domain.Fornecedor
		if err := rows.Scan(&item.ID, &item.Nome, &item.Email, &item.Telefone, &item.UnidadeID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InsertFornecedor(ctx context.Context, db database.DBTX, item domain.Fornecedor) (domain.Fornecedor, error) {
	err := db.QueryRow(ctx, `INSERT INTO contratos_fornecedores (id, nome, email, telefone, unidade_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, item.ID, item.Nome, item.Email, item.Telefone, item.UnidadeID, item.CreatedAt, item.UpdatedAt).Scan(&item.ID)
	return item, mapDB(err)
}

func (r *Repository) DeleteFornecedor(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `DELETE FROM contratos_fornecedores WHERE id = $1 AND ($2 = false OR unidade_id = ANY($3::uuid[]))`, id, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ListContratos(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Contrato, error) {
	rows, err := db.Query(ctx, `SELECT c.id, c.numero, c.titulo, c.fornecedor_id, f.nome, c.inicio, c.fim, c.valor_centavos, c.unidade_id, c.created_at, c.updated_at
		FROM contratos_contratos c JOIN contratos_fornecedores f ON f.id = c.fornecedor_id
		WHERE ($1 = false OR c.unidade_id = ANY($2::uuid[])) ORDER BY c.fim, c.id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Contrato{}
	for rows.Next() {
		var item domain.Contrato
		if err := rows.Scan(&item.ID, &item.Numero, &item.Titulo, &item.FornecedorID, &item.FornecedorNome, &item.Inicio, &item.Fim, &item.ValorCentavos, &item.UnidadeID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InsertContrato(ctx context.Context, db database.DBTX, item domain.Contrato) (domain.Contrato, error) {
	err := db.QueryRow(ctx, `INSERT INTO contratos_contratos
		(id, numero, titulo, fornecedor_id, inicio, fim, valor_centavos, unidade_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		item.ID, item.Numero, item.Titulo, item.FornecedorID, item.Inicio, item.Fim, item.ValorCentavos, item.UnidadeID, item.CreatedAt, item.UpdatedAt).Scan(&item.ID)
	if err != nil {
		return domain.Contrato{}, mapDB(err)
	}
	lista, err := r.ListContratos(ctx, db, domain.Filtro{})
	if err != nil {
		return domain.Contrato{}, err
	}
	for _, c := range lista {
		if c.ID == item.ID {
			return c, nil
		}
	}
	return item, nil
}

func (r *Repository) DeleteContrato(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `DELETE FROM contratos_contratos WHERE id = $1 AND ($2 = false OR unidade_id = ANY($3::uuid[]))`, id, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ListLicencas(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Licenca, error) {
	rows, err := db.Query(ctx, `SELECT l.id, l.nome, l.fornecedor_id, f.nome, l.fim, l.quantidade, l.programa, l.unidade_id, l.created_at, l.updated_at
		FROM contratos_licencas l JOIN contratos_fornecedores f ON f.id = l.fornecedor_id
		WHERE ($1 = false OR l.unidade_id = ANY($2::uuid[])) ORDER BY l.nome, l.id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Licenca{}
	for rows.Next() {
		var item domain.Licenca
		if err := rows.Scan(&item.ID, &item.Nome, &item.FornecedorID, &item.FornecedorNome, &item.Fim, &item.Quantidade, &item.Programa, &item.UnidadeID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InsertLicenca(ctx context.Context, db database.DBTX, item domain.Licenca) (domain.Licenca, error) {
	err := db.QueryRow(ctx, `INSERT INTO contratos_licencas (id, nome, fornecedor_id, fim, quantidade, programa, unidade_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		item.ID, item.Nome, item.FornecedorID, item.Fim, item.Quantidade, item.Programa, item.UnidadeID, item.CreatedAt, item.UpdatedAt).Scan(&item.ID)
	if err != nil {
		return domain.Licenca{}, mapDB(err)
	}
	lista, err := r.ListLicencas(ctx, db, domain.Filtro{})
	if err != nil {
		return domain.Licenca{}, err
	}
	for _, l := range lista {
		if l.ID == item.ID {
			return l, nil
		}
	}
	return item, nil
}

func (r *Repository) DeleteLicenca(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	tag, err := db.Exec(ctx, `DELETE FROM contratos_licencas WHERE id = $1 AND ($2 = false OR unidade_id = ANY($3::uuid[]))`, id, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) Vencendo(ctx context.Context, db database.DBTX) ([]domain.Vencimento, error) {
	rows, err := db.Query(ctx, `SELECT id, 'contrato', titulo, fim, unidade_id FROM contratos_contratos WHERE fim <= CURRENT_DATE + 30
		UNION ALL
		SELECT id, 'licenca', nome, fim, unidade_id FROM contratos_licencas WHERE fim IS NOT NULL AND fim <= CURRENT_DATE + 30`)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Vencimento{}
	for rows.Next() {
		var v domain.Vencimento
		if err := rows.Scan(&v.ID, &v.Tipo, &v.Titulo, &v.Fim, &v.UnidadeID); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, v)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) Destinatarios(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT u.id FROM users u
		LEFT JOIN user_scopes us ON us.user_id = u.id
		LEFT JOIN perfis p ON p.id = us.perfil_id AND p.ativo
		WHERE u.active AND (
			'nova-admin' = ANY (u.roles)
			OR (
				p.permissoes && ARRAY['contratos:manage', 'contratos:*', '*']
				AND (
					(us.unidade_id IS NULL AND us.entidade_id IS NULL AND us.departamento_id IS NULL)
					OR us.unidade_id = $1
				)
			)
		)`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, id)
	}
	return out, mapDB(rows.Err())
}
