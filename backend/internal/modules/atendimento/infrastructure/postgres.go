// Package infrastructure persiste o atendimento no PostgreSQL.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const ofertaCols = `id, nome, descricao, tipo, produto_id, unidade_id, ativa, created_at, updated_at, campos, categoria_id, icone, ordem`

const equipeCols = `e.id, e.nome, e.unidade_id, e.ola_horas, e.created_at, e.updated_at,
	COALESCE((SELECT array_agg(m.user_id ORDER BY m.user_id) FROM atendimento_equipe_membros m WHERE m.equipe_id = e.id), '{}')`

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case database.IsForeignKeyViolation(err):
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch {
			case strings.Contains(pgErr.ConstraintName, "categoria_id"):
				return domain.ErrCategoriaInexistente
			case strings.Contains(pgErr.ConstraintName, "oferta_id"):
				return domain.ErrOfertaInexistente
			case strings.Contains(pgErr.ConstraintName, "solicitacao_id"):
				return domain.ErrNaoEncontrado
			case strings.Contains(pgErr.ConstraintName, "responsavel"):
				return domain.ErrResponsavelInvalido
			case strings.Contains(pgErr.ConstraintName, "solicitante"), strings.Contains(pgErr.ConstraintName, "autor"):
				return domain.ErrSolicitanteInvalido
			case strings.Contains(pgErr.ConstraintName, "user_id"):
				return domain.ErrMembroInvalido
			}
		}
		return domain.ErrUnidadeInexistente
	case database.IsCheckViolation(err):
		return domain.ErrEntradaInvalida
	case database.IsUniqueViolation(err) && strings.Contains(err.Error(), "atendimento_categorias_nome"):
		return domain.ErrCategoriaRepetida
	default:
		return fmt.Errorf("atendimento: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

func scanOferta(row interface{ Scan(...any) error }) (domain.Oferta, error) {
	var o domain.Oferta
	var raw []byte
	err := row.Scan(&o.ID, &o.Nome, &o.Descricao, &o.Tipo, &o.ProdutoID, &o.UnidadeID, &o.Ativa, &o.CreatedAt, &o.UpdatedAt, &raw, &o.CategoriaID, &o.Icone, &o.Ordem)
	if err != nil {
		return domain.Oferta{}, err
	}
	o.Campos, err = camposDe(raw)
	return o, err
}

func camposDe(raw []byte) ([]domain.Campo, error) {
	if len(raw) == 0 {
		return []domain.Campo{}, nil
	}
	var out []domain.Campo
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []domain.Campo{}
	}
	return out, nil
}

func camposJSON(campos []domain.Campo) ([]byte, error) {
	if campos == nil {
		campos = []domain.Campo{}
	}
	return json.Marshal(campos)
}

func scanEquipe(row interface{ Scan(...any) error }) (domain.Equipe, error) {
	var e domain.Equipe
	err := row.Scan(&e.ID, &e.Nome, &e.UnidadeID, &e.OLAHoras, &e.CreatedAt, &e.UpdatedAt, &e.Membros)
	if e.Membros == nil {
		e.Membros = []uuid.UUID{}
	}
	return e, err
}

func scanCandidato(row interface{ Scan(...any) error }) (domain.Candidato, error) {
	var c domain.Candidato
	err := row.Scan(&c.ID, &c.Nome)
	return c, err
}

// ListOfertas devolve as ofertas do filtro, por nome.
func (r *Repository) ListOfertas(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Oferta, error) {
	rows, err := db.Query(ctx, `SELECT `+ofertaCols+` FROM atendimento_ofertas
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
		ORDER BY nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Oferta{}
	for rows.Next() {
		o, err := scanOferta(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// GetOferta busca uma oferta.
func (r *Repository) GetOferta(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Oferta, error) {
	o, err := scanOferta(db.QueryRow(ctx, `SELECT `+ofertaCols+` FROM atendimento_ofertas WHERE id = $1`, id))
	if err != nil {
		return domain.Oferta{}, mapDB(err)
	}
	return o, nil
}

// InsertOferta grava a oferta.
func (r *Repository) InsertOferta(ctx context.Context, db database.DBTX, o domain.Oferta) (domain.Oferta, error) {
	campos, err := camposJSON(o.Campos)
	if err != nil {
		return domain.Oferta{}, err
	}
	out, err := scanOferta(db.QueryRow(ctx, `INSERT INTO atendimento_ofertas (`+ofertaCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+ofertaCols,
		o.ID, o.Nome, o.Descricao, o.Tipo, o.ProdutoID, o.UnidadeID, o.Ativa, o.CreatedAt, o.UpdatedAt, campos, o.CategoriaID, o.Icone, o.Ordem))
	if err != nil {
		return domain.Oferta{}, mapDB(err)
	}
	return out, nil
}

// UpdateOferta grava a edição.
func (r *Repository) UpdateOferta(ctx context.Context, db database.DBTX, o domain.Oferta) (domain.Oferta, error) {
	campos, err := camposJSON(o.Campos)
	if err != nil {
		return domain.Oferta{}, err
	}
	out, err := scanOferta(db.QueryRow(ctx, `UPDATE atendimento_ofertas
		SET nome = $2, descricao = $3, tipo = $4, produto_id = $5, unidade_id = $6, ativa = $7, campos = $8,
			categoria_id = $9, icone = $10, ordem = $11
		WHERE id = $1 RETURNING `+ofertaCols,
		o.ID, o.Nome, o.Descricao, o.Tipo, o.ProdutoID, o.UnidadeID, o.Ativa, campos, o.CategoriaID, o.Icone, o.Ordem))
	if err != nil {
		return domain.Oferta{}, mapDB(err)
	}
	return out, nil
}

// ListEquipes devolve as equipes com os membros.
func (r *Repository) ListEquipes(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Equipe, error) {
	rows, err := db.Query(ctx, `SELECT `+equipeCols+` FROM atendimento_equipes e
		WHERE ($1 = false OR e.unidade_id = ANY($2::uuid[]))
		ORDER BY e.nome, e.id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Equipe{}
	for rows.Next() {
		e, err := scanEquipe(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// GetEquipe busca uma equipe com os membros.
func (r *Repository) GetEquipe(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Equipe, error) {
	e, err := scanEquipe(db.QueryRow(ctx, `SELECT `+equipeCols+` FROM atendimento_equipes e WHERE e.id = $1`, id))
	if err != nil {
		return domain.Equipe{}, mapDB(err)
	}
	return e, nil
}

// InsertEquipe grava a equipe. Os membros entram em SubstituirMembros.
func (r *Repository) InsertEquipe(ctx context.Context, db database.DBTX, e domain.Equipe) (domain.Equipe, error) {
	out, err := scanEquipe(db.QueryRow(ctx, `INSERT INTO atendimento_equipes (id, nome, unidade_id, ola_horas, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, nome, unidade_id, ola_horas, created_at, updated_at, '{}'::uuid[]`,
		e.ID, e.Nome, e.UnidadeID, e.OLAHoras, e.CreatedAt, e.UpdatedAt))
	if err != nil {
		return domain.Equipe{}, mapDB(err)
	}
	out.Membros = idsOrEmpty(e.Membros)
	return out, nil
}

// UpdateEquipe grava nome e unidade.
func (r *Repository) UpdateEquipe(ctx context.Context, db database.DBTX, e domain.Equipe) (domain.Equipe, error) {
	out, err := scanEquipe(db.QueryRow(ctx, `UPDATE atendimento_equipes SET nome = $2, unidade_id = $3, ola_horas = $4 WHERE id = $1
		RETURNING id, nome, unidade_id, ola_horas, created_at, updated_at, '{}'::uuid[]`, e.ID, e.Nome, e.UnidadeID, e.OLAHoras))
	if err != nil {
		return domain.Equipe{}, mapDB(err)
	}
	out.Membros = idsOrEmpty(e.Membros)
	return out, nil
}

// SubstituirMembros troca o conjunto de membros da equipe.
func (r *Repository) SubstituirMembros(ctx context.Context, db database.DBTX, equipe uuid.UUID, membros []uuid.UUID) error {
	if _, err := db.Exec(ctx, `DELETE FROM atendimento_equipe_membros WHERE equipe_id = $1`, equipe); err != nil {
		return mapDB(err)
	}
	if len(membros) == 0 {
		return nil
	}
	_, err := db.Exec(ctx, `INSERT INTO atendimento_equipe_membros (equipe_id, user_id)
		SELECT $1, unnest($2::uuid[])`, equipe, membros)
	return mapDB(err)
}

// ContarAtivos conta usuários ativos dentre os ids.
func (r *Repository) ContarAtivos(ctx context.Context, db database.DBTX, ids []uuid.UUID) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM users WHERE active AND id = ANY($1::uuid[])`, idsOrEmpty(ids)).Scan(&n)
	return n, mapDB(err)
}

// Candidatos lista usuários ativos lotados na unidade.
func (r *Repository) Candidatos(ctx context.Context, db database.DBTX, unidade uuid.UUID) ([]domain.Candidato, error) {
	rows, err := db.Query(ctx, `SELECT u.id, COALESCE(NULLIF(btrim(u.display_name), ''), u.username)
		FROM users u
		WHERE u.active AND EXISTS (
			SELECT 1 FROM user_scopes s WHERE s.user_id = u.id AND s.unidade_id = $1
		)
		ORDER BY 2, u.id`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Candidato{}
	for rows.Next() {
		c, err := scanCandidato(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}
