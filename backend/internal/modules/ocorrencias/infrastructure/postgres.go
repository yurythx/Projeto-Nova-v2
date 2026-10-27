// Package infrastructure persiste as ocorrências no PostgreSQL.
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const nome = `COALESCE(NULLIF(btrim(%[1]s.display_name), ''), %[1]s.username, '')`

var ocorrenciaSelect = `SELECT o.id, o.titulo, o.impacto, o.estado, o.servicos, o.chamado_id, o.coordenador_id, ` +
	fmt.Sprintf(nome, "u") + `, o.iniciada_em, o.resolvida_em, o.revisao, o.revisada_em, o.updated_at
	FROM ocorrencias o
	LEFT JOIN users u ON u.id = o.coordenador_id`

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrada
	case database.IsForeignKeyViolation(err):
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && strings.Contains(pgErr.ConstraintName, "coordenador") {
			return domain.ErrCoordenadorInvalido
		}
		return domain.ErrEntradaInvalida
	case database.IsCheckViolation(err):
		return domain.ErrEntradaInvalida
	default:
		return fmt.Errorf("ocorrencias: %w", err)
	}
}

func scanOcorrencia(row interface{ Scan(...any) error }) (domain.Ocorrencia, error) {
	var o domain.Ocorrencia
	err := row.Scan(&o.ID, &o.Titulo, &o.Impacto, &o.Estado, &o.Servicos, &o.ChamadoID, &o.CoordenadorID, &o.CoordenadorNome,
		&o.IniciadaEm, &o.ResolvidaEm, &o.Revisao, &o.RevisadaEm, &o.UpdatedAt)
	o.Atualizacoes = []domain.Atualizacao{}
	return o, err
}

// Insert grava a ocorrência sem a linha do tempo (InsertAtualizacao).
func (r *Repository) Insert(ctx context.Context, db database.DBTX, o domain.Ocorrencia) error {
	_, err := db.Exec(ctx, `INSERT INTO ocorrencias
		(id, titulo, impacto, estado, servicos, chamado_id, coordenador_id, iniciada_em, resolvida_em, revisao, revisada_em, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		o.ID, o.Titulo, o.Impacto, o.Estado, o.Servicos, o.ChamadoID, o.CoordenadorID, o.IniciadaEm, o.ResolvidaEm,
		o.Revisao, o.RevisadaEm, o.UpdatedAt)
	return mapDB(err)
}

// Update regrava o que muda depois da declaração.
func (r *Repository) Update(ctx context.Context, db database.DBTX, o domain.Ocorrencia) error {
	tag, err := db.Exec(ctx, `UPDATE ocorrencias SET impacto = $2, estado = $3, coordenador_id = $4, resolvida_em = $5,
		revisao = $6, revisada_em = $7, updated_at = $8 WHERE id = $1`,
		o.ID, o.Impacto, o.Estado, o.CoordenadorID, o.ResolvidaEm, o.Revisao, o.RevisadaEm, o.UpdatedAt)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrada
	}
	return nil
}

// InsertAtualizacao acrescenta uma entrada à linha do tempo.
func (r *Repository) InsertAtualizacao(ctx context.Context, db database.DBTX, a domain.Atualizacao) error {
	_, err := db.Exec(ctx, `INSERT INTO ocorrencias_atualizacoes (id, ocorrencia_id, estado, impacto, texto, autor_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, a.ID, a.OcorrenciaID, a.Estado, a.Impacto, a.Texto, a.AutorID, a.CreatedAt)
	return mapDB(err)
}

// Travar lê a ocorrência e bloqueia a linha até o fim da transação.
func (r *Repository) Travar(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Ocorrencia, error) {
	o, err := scanOcorrencia(db.QueryRow(ctx, ocorrenciaSelect+` WHERE o.id = $1 FOR UPDATE OF o`, id))
	if err != nil {
		return domain.Ocorrencia{}, mapDB(err)
	}
	return o, nil
}

// Get devolve a ocorrência com a linha do tempo.
func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Ocorrencia, error) {
	o, err := scanOcorrencia(db.QueryRow(ctx, ocorrenciaSelect+` WHERE o.id = $1`, id))
	if err != nil {
		return domain.Ocorrencia{}, mapDB(err)
	}
	lista, err := r.comAtualizacoes(ctx, db, []domain.Ocorrencia{o})
	if err != nil {
		return domain.Ocorrencia{}, err
	}
	return lista[0], nil
}

// List devolve as abertas primeiro (a mais antiga sem notícia no topo) e
// depois as resolvidas mais recentes.
func (r *Repository) List(ctx context.Context, db database.DBTX, limite int) ([]domain.Ocorrencia, error) {
	return r.listar(ctx, db, ocorrenciaSelect+`
		ORDER BY (o.estado = 'resolvido'), CASE WHEN o.estado = 'resolvido' THEN o.resolvida_em END DESC, o.updated_at
		LIMIT $1`, limite)
}

// ParaPagina devolve as abertas e as resolvidas desde a data.
func (r *Repository) ParaPagina(ctx context.Context, db database.DBTX, desde time.Time) ([]domain.Ocorrencia, error) {
	return r.listar(ctx, db, ocorrenciaSelect+`
		WHERE o.estado <> 'resolvido' OR o.resolvida_em >= $1
		ORDER BY (o.estado = 'resolvido'), o.iniciada_em DESC`, desde)
}

func (r *Repository) listar(ctx context.Context, db database.DBTX, sql string, arg any) ([]domain.Ocorrencia, error) {
	rows, err := db.Query(ctx, sql, arg)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Ocorrencia{}
	for rows.Next() {
		o, err := scanOcorrencia(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return r.comAtualizacoes(ctx, db, out)
}

// comAtualizacoes carrega a linha do tempo de todas de uma vez.
func (r *Repository) comAtualizacoes(ctx context.Context, db database.DBTX, lista []domain.Ocorrencia) ([]domain.Ocorrencia, error) {
	if len(lista) == 0 {
		return lista, nil
	}
	ids := make([]uuid.UUID, len(lista))
	pos := map[uuid.UUID]int{}
	for i, o := range lista {
		ids[i], pos[o.ID] = o.ID, i
	}
	rows, err := db.Query(ctx, `SELECT a.id, a.ocorrencia_id, a.estado, a.impacto, a.texto, a.autor_id, `+fmt.Sprintf(nome, "u")+`, a.created_at
		FROM ocorrencias_atualizacoes a
		LEFT JOIN users u ON u.id = a.autor_id
		WHERE a.ocorrencia_id = ANY($1::uuid[])
		ORDER BY a.created_at, a.id`, ids)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a domain.Atualizacao
		if err := rows.Scan(&a.ID, &a.OcorrenciaID, &a.Estado, &a.Impacto, &a.Texto, &a.AutorID, &a.AutorNome, &a.CreatedAt); err != nil {
			return nil, mapDB(err)
		}
		i := pos[a.OcorrenciaID]
		lista[i].Atualizacoes = append(lista[i].Atualizacoes, a)
	}
	return lista, mapDB(rows.Err())
}
