// Package infrastructure persiste problemas no PostgreSQL.
package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

const cols = `id, numero, titulo, descricao, estado, causa_raiz, contorno, produto_id, unidade_id, criado_por, resolvido_em, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (domain.Problema, error) {
	var p domain.Problema
	err := row.Scan(&p.ID, &p.Numero, &p.Titulo, &p.Descricao, &p.Estado, &p.CausaRaiz, &p.Contorno, &p.ProdutoID,
		&p.UnidadeID, &p.CriadoPor, &p.ResolvidoEm, &p.CreatedAt, &p.UpdatedAt)
	p.Incidentes, p.Ativos = []domain.IncidenteVinculado{}, []domain.AtivoVinculado{}
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
		return fmt.Errorf("problemas: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// coletar lê todas as linhas.
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

// List devolve os problemas do filtro, os mais novos primeiro.
func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Problema, error) {
	rows, err := db.Query(ctx, `SELECT `+cols+` FROM problemas p
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
			AND ($3 = '' OR estado = $3)
			AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM problemas_incidentes i WHERE i.problema_id = p.id AND i.solicitacao_id = $4))
			AND ($5 = '' OR titulo ILIKE '%' || $5 || '%' OR contorno ILIKE '%' || $5 || '%')
		ORDER BY created_at DESC, id
		LIMIT 200`, f.Restrito, idsOrEmpty(f.Unidades), f.Estado, f.SolicitacaoID, f.Busca)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scan)
}

// Get busca um problema (sem os vínculos).
func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Problema, error) {
	p, err := scan(db.QueryRow(ctx, `SELECT `+cols+` FROM problemas WHERE id = $1`, id))
	if err != nil {
		return domain.Problema{}, mapDB(err)
	}
	return p, nil
}

// Travar bloqueia a linha na transação do chamador.
func (r *Repository) Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	var achou uuid.UUID
	return mapDB(db.QueryRow(ctx, `SELECT id FROM problemas WHERE id = $1 FOR UPDATE`, id).Scan(&achou))
}

// Insert grava o problema e devolve o número.
func (r *Repository) Insert(ctx context.Context, db database.DBTX, p domain.Problema) (domain.Problema, error) {
	out, err := scan(db.QueryRow(ctx, `INSERT INTO problemas
		(id, titulo, descricao, estado, causa_raiz, contorno, produto_id, unidade_id, criado_por, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+cols,
		p.ID, p.Titulo, p.Descricao, p.Estado, p.CausaRaiz, p.Contorno, p.ProdutoID, p.UnidadeID, p.CriadoPor, p.CreatedAt, p.UpdatedAt))
	if err != nil {
		return domain.Problema{}, mapDB(err)
	}
	return out, nil
}

// Update grava a edição e o estado.
func (r *Repository) Update(ctx context.Context, db database.DBTX, p domain.Problema) error {
	tag, err := db.Exec(ctx, `UPDATE problemas
		SET titulo = $2, descricao = $3, estado = $4, causa_raiz = $5, contorno = $6, produto_id = $7, unidade_id = $8, resolvido_em = $9
		WHERE id = $1`, p.ID, p.Titulo, p.Descricao, p.Estado, p.CausaRaiz, p.Contorno, p.ProdutoID, p.UnidadeID, p.ResolvidoEm)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

// Incidentes devolve os incidentes ligados, na ordem do vínculo.
func (r *Repository) Incidentes(ctx context.Context, db database.DBTX, problema uuid.UUID) ([]domain.IncidenteVinculado, error) {
	rows, err := db.Query(ctx, `SELECT solicitacao_id, numero, titulo, vinculado_em FROM problemas_incidentes
		WHERE problema_id = $1 ORDER BY vinculado_em, solicitacao_id`, problema)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.IncidenteVinculado, error) {
		var i domain.IncidenteVinculado
		err := row.Scan(&i.SolicitacaoID, &i.Numero, &i.Titulo, &i.VinculadoEm)
		return i, err
	})
}

// VincularIncidente liga o incidente; falso quando já estava ligado.
func (r *Repository) VincularIncidente(ctx context.Context, db database.DBTX, problema uuid.UUID, i domain.IncidenteVinculado, autor uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `INSERT INTO problemas_incidentes (problema_id, solicitacao_id, numero, titulo, vinculado_por)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, problema, i.SolicitacaoID, i.Numero, i.Titulo, autor)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// DesvincularIncidente desfaz o vínculo; falso quando não havia.
func (r *Repository) DesvincularIncidente(ctx context.Context, db database.DBTX, problema, solicitacao uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM problemas_incidentes WHERE problema_id = $1 AND solicitacao_id = $2`, problema, solicitacao)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// AtivosDe devolve os ativos ligados (o nome vem da porta, na leitura).
func (r *Repository) AtivosDe(ctx context.Context, db database.DBTX, problema uuid.UUID) ([]domain.AtivoVinculado, error) {
	rows, err := db.Query(ctx, `SELECT ativo_id, vinculado_em FROM problemas_ativos
		WHERE problema_id = $1 ORDER BY vinculado_em, ativo_id`, problema)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.AtivoVinculado, error) {
		var a domain.AtivoVinculado
		err := row.Scan(&a.AtivoID, &a.VinculadoEm)
		return a, err
	})
}

// VincularAtivo liga o ativo; falso quando já estava ligado.
func (r *Repository) VincularAtivo(ctx context.Context, db database.DBTX, problema, ativo, autor uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `INSERT INTO problemas_ativos (problema_id, ativo_id, vinculado_por)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, problema, ativo, autor)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// DesvincularAtivo desfaz o vínculo; falso quando não havia.
func (r *Repository) DesvincularAtivo(ctx context.Context, db database.DBTX, problema, ativo uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM problemas_ativos WHERE problema_id = $1 AND ativo_id = $2`, problema, ativo)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// JaVinculados marca as solicitações que já estão em algum problema.
func (r *Repository) JaVinculados(ctx context.Context, db database.DBTX, solicitacoes []uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT solicitacao_id FROM problemas_incidentes WHERE solicitacao_id = ANY($1::uuid[])`,
		idsOrEmpty(solicitacoes))
	if err != nil {
		return nil, mapDB(err)
	}
	ids, err := coletar(rows, func(row interface{ Scan(...any) error }) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
