package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// conteudoIA é o que a IA sugeriu, como fica na coluna JSONB.
type conteudoIA struct {
	Classificacao *domain.ClassificacaoSugerida `json:"classificacao,omitempty"`
	Resumo        string                        `json:"resumo,omitempty"`
}

const sugestaoCols = `id, solicitacao_id, tipo, conexao, conteudo, pedida_por, decisao, decidida_por, decidida_em, created_at`

func scanSugestao(row interface{ Scan(...any) error }) (domain.Sugestao, error) {
	var s domain.Sugestao
	var raw []byte
	if err := row.Scan(&s.ID, &s.SolicitacaoID, &s.Tipo, &s.Conexao, &raw, &s.PedidaPor, &s.Decisao, &s.DecididaPor, &s.DecididaEm, &s.CreatedAt); err != nil {
		return domain.Sugestao{}, err
	}
	var c conteudoIA
	if err := json.Unmarshal(raw, &c); err != nil {
		return domain.Sugestao{}, err
	}
	s.Classificacao, s.Resumo = c.Classificacao, c.Resumo
	return s, nil
}

// InsertSugestao grava a sugestão ainda sem decisão.
func (r *Repository) InsertSugestao(ctx context.Context, db database.DBTX, s domain.Sugestao) error {
	// Só inteiros e textos: o Marshal não tem como falhar.
	conteudo, _ := json.Marshal(conteudoIA{Classificacao: s.Classificacao, Resumo: s.Resumo})
	_, err := db.Exec(ctx, `INSERT INTO atendimento_ia_sugestoes
		(id, solicitacao_id, tipo, conexao, conteudo, pedida_por, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.SolicitacaoID, s.Tipo, s.Conexao, conteudo, s.PedidaPor, s.CreatedAt)
	return mapDB(err)
}

// TravarSugestao lê a sugestão e bloqueia a linha até o fim da transação.
func (r *Repository) TravarSugestao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Sugestao, error) {
	s, err := scanSugestao(db.QueryRow(ctx, `SELECT `+sugestaoCols+` FROM atendimento_ia_sugestoes WHERE id = $1 FOR UPDATE`, id))
	if err != nil {
		return domain.Sugestao{}, mapDB(err)
	}
	return s, nil
}

// DecidirSugestao grava a decisão. A que já foi decidida não muda.
func (r *Repository) DecidirSugestao(ctx context.Context, db database.DBTX, s domain.Sugestao) error {
	tag, err := db.Exec(ctx, `UPDATE atendimento_ia_sugestoes
		SET decisao = $2, decidida_por = $3, decidida_em = $4
		WHERE id = $1 AND decisao IS NULL`, s.ID, s.Decisao, s.DecididaPor, s.DecididaEm)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSugestaoDecidida
	}
	return nil
}

// IndicadoresIA conta as sugestões de cada tarefa desde a data, no
// escopo das unidades dos chamados. As duas tarefas sempre aparecem.
func (r *Repository) IndicadoresIA(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]domain.IndicadorIA, error) {
	rows, err := db.Query(ctx, `SELECT t.tipo,
			count(g.id),
			count(g.id) FILTER (WHERE g.decisao = 'aceita'),
			count(g.id) FILTER (WHERE g.decisao = 'recusada'),
			count(g.id) FILTER (WHERE g.id IS NOT NULL AND g.decisao IS NULL)
		FROM (VALUES ('classificacao'), ('resumo')) AS t(tipo)
		LEFT JOIN (
			SELECT g.id, g.tipo, g.decisao
			FROM atendimento_ia_sugestoes g
			JOIN atendimento_solicitacoes s ON s.id = g.solicitacao_id
			WHERE g.created_at >= $1 AND ($2 = false OR s.unidade_id = ANY($3::uuid[]))
		) g ON g.tipo = t.tipo
		GROUP BY t.tipo
		ORDER BY t.tipo`, desde, restrito, idsOrEmpty(unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.IndicadorIA{}
	for rows.Next() {
		var i domain.IndicadorIA
		if err := rows.Scan(&i.Tipo, &i.Total, &i.Aceitas, &i.Recusadas, &i.Pendentes); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, i)
	}
	return out, mapDB(rows.Err())
}
