// Package infrastructure persiste mudanças no PostgreSQL.
package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const nome = `COALESCE(NULLIF(btrim(u.display_name), ''), u.username, '')`

const selectMudanca = `SELECT m.id, m.numero, m.titulo, m.descricao, m.justificativa, m.tipo, m.risco, m.risco_score, m.estado, m.plano_reversao,
	m.inicio_planejado, m.fim_planejado, m.resultado, m.executada_inicio, m.executada_fim, m.produto_id, m.unidade_id,
	m.solicitante_id, ` + nome + `, m.created_at, m.updated_at
	FROM mudancas m LEFT JOIN users u ON u.id = m.solicitante_id`

func scan(row interface{ Scan(...any) error }) (domain.Mudanca, error) {
	var m domain.Mudanca
	err := row.Scan(&m.ID, &m.Numero, &m.Titulo, &m.Descricao, &m.Justificativa, &m.Tipo, &m.Risco, &m.RiscoScore, &m.Estado, &m.PlanoReversao,
		&m.InicioPlanejado, &m.FimPlanejado, &m.Resultado, &m.ExecutadaInicio, &m.ExecutadaFim, &m.ProdutoID, &m.UnidadeID,
		&m.SolicitanteID, &m.SolicitanteNome, &m.CreatedAt, &m.UpdatedAt)
	m.Aprovacoes, m.Vinculos = []domain.Aprovacao{}, []domain.Vinculo{}
	return m, err
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
		return fmt.Errorf("mudancas: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

func coletar[T any](rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}, scan func(interface{ Scan(...any) error }) (T, error)) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// List devolve as mudanças do filtro; as com janela mais próxima primeiro.
func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Mudanca, error) {
	rows, err := db.Query(ctx, selectMudanca+`
		WHERE ($1 = false OR m.unidade_id = ANY($2::uuid[]))
			AND ($3 = '' OR m.estado = $3)
			AND ($5::uuid IS NULL OR EXISTS (SELECT 1 FROM mudancas_vinculos v WHERE v.mudanca_id = m.id AND v.tipo = $4 AND v.alvo_id = $5))
		ORDER BY m.inicio_planejado NULLS LAST, m.created_at DESC, m.id
		LIMIT 200`, f.Restrito, idsOrEmpty(f.Unidades), f.Estado, f.AlvoTipo, f.AlvoID)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scan)
}

// Get busca a mudança (sem as listas).
func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Mudanca, error) {
	m, err := scan(db.QueryRow(ctx, selectMudanca+` WHERE m.id = $1`, id))
	if err != nil {
		return domain.Mudanca{}, mapDB(err)
	}
	return m, nil
}

// Travar bloqueia a linha na transação do chamador.
func (r *Repository) Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	var achou uuid.UUID
	return mapDB(db.QueryRow(ctx, `SELECT id FROM mudancas WHERE id = $1 FOR UPDATE`, id).Scan(&achou))
}

// Insert grava o rascunho e devolve o número.
func (r *Repository) Insert(ctx context.Context, db database.DBTX, m domain.Mudanca) (domain.Mudanca, error) {
	if _, err := db.Exec(ctx, `INSERT INTO mudancas
		(id, titulo, descricao, justificativa, tipo, risco, risco_score, estado, plano_reversao, inicio_planejado, fim_planejado,
		 produto_id, unidade_id, solicitante_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		m.ID, m.Titulo, m.Descricao, m.Justificativa, m.Tipo, m.Risco, m.RiscoScore, m.Estado, m.PlanoReversao, m.InicioPlanejado,
		m.FimPlanejado, m.ProdutoID, m.UnidadeID, m.SolicitanteID, m.CreatedAt, m.UpdatedAt); err != nil {
		return domain.Mudanca{}, mapDB(err)
	}
	return r.Get(ctx, db, m.ID)
}

// Update grava a edição, o estado e a execução.
func (r *Repository) Update(ctx context.Context, db database.DBTX, m domain.Mudanca) error {
	tag, err := db.Exec(ctx, `UPDATE mudancas
		SET titulo = $2, descricao = $3, justificativa = $4, tipo = $5, risco = $6, risco_score = $7, estado = $8, plano_reversao = $9,
			inicio_planejado = $10, fim_planejado = $11, resultado = $12, executada_inicio = $13, executada_fim = $14,
			produto_id = $15, unidade_id = $16
		WHERE id = $1`, m.ID, m.Titulo, m.Descricao, m.Justificativa, m.Tipo, m.Risco, m.RiscoScore, m.Estado, m.PlanoReversao,
		m.InicioPlanejado, m.FimPlanejado, m.Resultado, m.ExecutadaInicio, m.ExecutadaFim, m.ProdutoID, m.UnidadeID)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

// Aprovacoes devolve as decisões em ordem.
func (r *Repository) Aprovacoes(ctx context.Context, db database.DBTX, mudanca uuid.UUID) ([]domain.Aprovacao, error) {
	rows, err := db.Query(ctx, `SELECT a.id, a.aprovador_id, `+nome+`, a.decisao, a.parecer, a.em
		FROM mudancas_aprovacoes a LEFT JOIN users u ON u.id = a.aprovador_id
		WHERE a.mudanca_id = $1 ORDER BY a.em, a.id`, mudanca)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.Aprovacao, error) {
		var a domain.Aprovacao
		err := row.Scan(&a.ID, &a.AprovadorID, &a.Aprovador, &a.Decisao, &a.Parecer, &a.Em)
		return a, err
	})
}

// InsertAprovacao registra a decisão.
func (r *Repository) InsertAprovacao(ctx context.Context, db database.DBTX, mudanca uuid.UUID, a domain.Aprovacao) error {
	_, err := db.Exec(ctx, `INSERT INTO mudancas_aprovacoes (id, mudanca_id, aprovador_id, decisao, parecer, em)
		VALUES ($1, $2, $3, $4, $5, $6)`, a.ID, mudanca, a.AprovadorID, a.Decisao, a.Parecer, a.Em)
	return mapDB(err)
}

// Vinculos devolve os itens ligados, na ordem do vínculo.
func (r *Repository) Vinculos(ctx context.Context, db database.DBTX, mudanca uuid.UUID) ([]domain.Vinculo, error) {
	rows, err := db.Query(ctx, `SELECT tipo, alvo_id, rotulo, vinculado_em FROM mudancas_vinculos
		WHERE mudanca_id = $1 ORDER BY vinculado_em, tipo, alvo_id`, mudanca)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.Vinculo, error) {
		var v domain.Vinculo
		err := row.Scan(&v.Tipo, &v.AlvoID, &v.Rotulo, &v.VinculadoEm)
		return v, err
	})
}

// Vincular liga o item; falso quando já estava ligado.
func (r *Repository) Vincular(ctx context.Context, db database.DBTX, mudanca uuid.UUID, v domain.Vinculo, autor uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `INSERT INTO mudancas_vinculos (mudanca_id, tipo, alvo_id, rotulo, vinculado_por)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, mudanca, v.Tipo, v.AlvoID, v.Rotulo, autor)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// Desvincular desfaz o vínculo; falso quando não havia.
func (r *Repository) Desvincular(ctx context.Context, db database.DBTX, mudanca uuid.UUID, tipo string, alvo uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM mudancas_vinculos WHERE mudanca_id = $1 AND tipo = $2 AND alvo_id = $3`, mudanca, tipo, alvo)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// JanelaOcupada diz se o intervalo cruza um congelamento da unidade ou
// um congelamento institucional.
func (r *Repository) JanelaOcupada(ctx context.Context, db database.DBTX, unidade *uuid.UUID, inicio, fim time.Time) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM mudanca_congelamentos
		WHERE fim > $1 AND inicio < $2
			AND (unidade_id IS NULL OR unidade_id = $3 OR $3::uuid IS NULL)
	)`, inicio, fim, unidade).Scan(&ok)
	return ok, mapDB(err)
}

func (r *Repository) ListCongelamentos(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Congelamento, error) {
	rows, err := db.Query(ctx, `SELECT id, unidade_id, inicio, fim, motivo, created_at FROM mudanca_congelamentos
		WHERE ($1 = false OR unidade_id IS NULL OR unidade_id = ANY($2::uuid[]))
		ORDER BY inicio, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Congelamento{}
	for rows.Next() {
		var c domain.Congelamento
		if err := rows.Scan(&c.ID, &c.UnidadeID, &c.Inicio, &c.Fim, &c.Motivo, &c.CreatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, c)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InsertCongelamento(ctx context.Context, db database.DBTX, c domain.Congelamento) (domain.Congelamento, error) {
	err := db.QueryRow(ctx, `INSERT INTO mudanca_congelamentos (id, unidade_id, inicio, fim, motivo, created_at)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, c.ID, c.UnidadeID, c.Inicio, c.Fim, c.Motivo, c.CreatedAt).Scan(&c.ID)
	return c, mapDB(err)
}

func (r *Repository) DeleteCongelamento(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM mudanca_congelamentos WHERE id = $1`, id)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}
