package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// AbrirAtribuicao abre o trecho do responsável. A equipe é a dele na
// unidade do chamado; com mais de uma, a primeira pelo nome.
func (r *Repository) AbrirAtribuicao(ctx context.Context, db database.DBTX, solicitacao, responsavel uuid.UUID, unidade *uuid.UUID, em time.Time) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_atribuicoes (solicitacao_id, responsavel_id, equipe_id, equipe_nome, inicio)
		SELECT $1, $2, eq.id, COALESCE(eq.nome, ''), $4
		FROM (SELECT 1) um
		LEFT JOIN LATERAL (
			SELECT e.id, e.nome FROM atendimento_equipes e
			JOIN atendimento_equipe_membros m ON m.equipe_id = e.id
			WHERE m.user_id = $2 AND e.unidade_id IS NOT DISTINCT FROM $3
			ORDER BY e.nome, e.id LIMIT 1
		) eq ON true`, solicitacao, responsavel, unidade, em)
	return mapDB(err)
}

// FecharAtribuicao fecha o trecho aberto do chamado, se houver.
func (r *Repository) FecharAtribuicao(ctx context.Context, db database.DBTX, solicitacao uuid.UUID, em time.Time) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_atribuicoes SET fim = GREATEST($2, inicio) WHERE solicitacao_id = $1 AND fim IS NULL`, solicitacao, em)
	return mapDB(err)
}

// AtribuicoesDoPeriodo lê os trechos com equipe, com o prazo atual dela.
func (r *Repository) AtribuicoesDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]domain.Atribuicao, error) {
	rows, err := db.Query(ctx, `SELECT a.solicitacao_id, a.equipe_id, COALESCE(e.nome, a.equipe_nome), e.unidade_id, e.ola_horas, a.inicio, a.fim
		FROM atendimento_atribuicoes a
		JOIN atendimento_equipes e ON e.id = a.equipe_id
		JOIN atendimento_solicitacoes s ON s.id = a.solicitacao_id
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[])) AND (a.inicio >= $3 OR a.fim IS NULL)
		ORDER BY a.inicio, a.id`, restrito, idsOrEmpty(unidades), desde)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Atribuicao{}
	for rows.Next() {
		var a domain.Atribuicao
		if err := rows.Scan(&a.SolicitacaoID, &a.EquipeID, &a.EquipeNome, &a.UnidadeID, &a.OLAHoras, &a.Inicio, &a.Fim); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, a)
	}
	return out, mapDB(rows.Err())
}
