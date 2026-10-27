package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// ListarDoPeriodo lê as solicitações do escopo abertas desde a data.
func (r *Repository) ListarDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, limite int) ([]domain.Solicitacao, error) {
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[])) AND s.created_at >= $3
		ORDER BY s.created_at DESC, s.numero DESC
		LIMIT $4`, restrito, idsOrEmpty(unidades), desde, limite)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}

// ComContratoNoPeriodo lê os chamados do escopo com contrato de apoio.
func (r *Repository) ComContratoNoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]domain.Solicitacao, error) {
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE s.contrato_apoio_id IS NOT NULL AND ($1 = false OR s.unidade_id = ANY($2::uuid[])) AND s.created_at >= $3
		ORDER BY s.created_at`, restrito, idsOrEmpty(unidades), desde)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}

// SecretariasDe sobe a árvore de cada unidade até o topo.
func (r *Repository) SecretariasDe(ctx context.Context, db database.DBTX, unidades []uuid.UUID) (map[uuid.UUID]domain.Secretaria, error) {
	rows, err := db.Query(ctx, `WITH RECURSIVE sobe AS (
			SELECT id AS origem, id, parent_id, 0 AS nivel FROM unidades WHERE id = ANY($1::uuid[])
			UNION ALL
			SELECT s.origem, u.id, u.parent_id, s.nivel + 1 FROM sobe s JOIN unidades u ON u.id = s.parent_id WHERE s.nivel < 32
		)
		SELECT s.origem, u.id, u.nome, u.sigla FROM sobe s JOIN unidades u ON u.id = s.id WHERE s.parent_id IS NULL`, idsOrEmpty(unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]domain.Secretaria{}
	for rows.Next() {
		var origem uuid.UUID
		var s domain.Secretaria
		if err := rows.Scan(&origem, &s.ID, &s.Nome, &s.Sigla); err != nil {
			return nil, mapDB(err)
		}
		out[origem] = s
	}
	return out, mapDB(rows.Err())
}

// AvaliacoesDe devolve nota e facilidade de cada solicitação avaliada.
func (r *Repository) AvaliacoesDe(ctx context.Context, db database.DBTX, solicitacoes []uuid.UUID) (map[uuid.UUID]domain.Avaliacao, error) {
	rows, err := db.Query(ctx, `SELECT solicitacao_id, nota, facilidade FROM atendimento_avaliacoes WHERE solicitacao_id = ANY($1::uuid[])`, idsOrEmpty(solicitacoes))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]domain.Avaliacao{}
	for rows.Next() {
		var a domain.Avaliacao
		if err := rows.Scan(&a.SolicitacaoID, &a.Nota, &a.Facilidade); err != nil {
			return nil, mapDB(err)
		}
		out[a.SolicitacaoID] = a
	}
	return out, mapDB(rows.Err())
}

// TarefasDoPeriodo lê as tarefas com dona dos pedidos do escopo.
func (r *Repository) TarefasDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]domain.TarefaMedida, error) {
	rows, err := db.Query(ctx, `SELECT t.equipe_id, e.nome, e.ola_horas, t.estado, t.created_at, t.feita_em
		FROM atendimento_tarefas t
		JOIN atendimento_equipes e ON e.id = t.equipe_id
		JOIN atendimento_solicitacoes s ON s.id = t.solicitacao_id
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[])) AND s.created_at >= $3
		ORDER BY t.created_at`, restrito, idsOrEmpty(unidades), desde)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.TarefaMedida{}
	for rows.Next() {
		var t domain.TarefaMedida
		if err := rows.Scan(&t.EquipeID, &t.EquipeNome, &t.OLAHoras, &t.Estado, &t.CreatedAt, &t.FeitaEm); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, t)
	}
	return out, mapDB(rows.Err())
}
