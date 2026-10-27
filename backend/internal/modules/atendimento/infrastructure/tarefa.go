package infrastructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// ModeloDaOferta devolve o modelo de tarefas em ordem, com o nome da equipe.
func (r *Repository) ModeloDaOferta(ctx context.Context, db database.DBTX, oferta uuid.UUID) ([]domain.ModeloTarefa, error) {
	rows, err := db.Query(ctx, `SELECT m.ordem, m.titulo, m.equipe_id, e.nome
		FROM atendimento_oferta_tarefas m JOIN atendimento_equipes e ON e.id = m.equipe_id
		WHERE m.oferta_id = $1 ORDER BY m.ordem`, oferta)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.ModeloTarefa{}
	for rows.Next() {
		var m domain.ModeloTarefa
		if err := rows.Scan(&m.Ordem, &m.Titulo, &m.EquipeID, &m.EquipeNome); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, m)
	}
	return out, mapDB(rows.Err())
}

// SalvarModelo troca o modelo da oferta inteiro.
func (r *Repository) SalvarModelo(ctx context.Context, db database.DBTX, oferta uuid.UUID, modelo []domain.ModeloTarefa) error {
	if _, err := db.Exec(ctx, `DELETE FROM atendimento_oferta_tarefas WHERE oferta_id = $1`, oferta); err != nil {
		return mapDB(err)
	}
	for _, m := range modelo {
		if _, err := db.Exec(ctx, `INSERT INTO atendimento_oferta_tarefas (oferta_id, ordem, titulo, equipe_id) VALUES ($1, $2, $3, $4)`,
			oferta, m.Ordem, m.Titulo, m.EquipeID); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

// InsertTarefas grava as tarefas criadas do modelo.
func (r *Repository) InsertTarefas(ctx context.Context, db database.DBTX, tarefas []domain.Tarefa) error {
	for _, t := range tarefas {
		if _, err := db.Exec(ctx, `INSERT INTO atendimento_tarefas (id, solicitacao_id, ordem, titulo, equipe_id, equipe_nome, estado, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, t.ID, t.SolicitacaoID, t.Ordem, t.Titulo, t.EquipeID, t.EquipeNome, t.Estado, t.CreatedAt); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

var tarefaSelect = `SELECT t.id, t.solicitacao_id, t.ordem, t.titulo, t.equipe_id, t.equipe_nome, t.estado, t.observacao,
	t.feita_por, COALESCE(` + nomeDe("u") + `, ''), t.feita_em, t.created_at
	FROM atendimento_tarefas t LEFT JOIN users u ON u.id = t.feita_por`

func scanTarefa(row pgx.Row) (domain.Tarefa, error) {
	var t domain.Tarefa
	err := row.Scan(&t.ID, &t.SolicitacaoID, &t.Ordem, &t.Titulo, &t.EquipeID, &t.EquipeNome, &t.Estado, &t.Observacao,
		&t.FeitaPor, &t.FeitaPorNome, &t.FeitaEm, &t.CreatedAt)
	return t, err
}

// ListTarefas devolve as tarefas do pedido, em ordem.
func (r *Repository) ListTarefas(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]domain.Tarefa, error) {
	rows, err := db.Query(ctx, tarefaSelect+` WHERE t.solicitacao_id = $1 ORDER BY t.ordem`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Tarefa{}
	for rows.Next() {
		t, err := scanTarefa(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, t)
	}
	return out, mapDB(rows.Err())
}

// GetTarefa busca a tarefa do pedido.
func (r *Repository) GetTarefa(ctx context.Context, db database.DBTX, solicitacao, id uuid.UUID) (domain.Tarefa, error) {
	t, err := scanTarefa(db.QueryRow(ctx, tarefaSelect+` WHERE t.solicitacao_id = $1 AND t.id = $2`, solicitacao, id))
	return t, mapDB(err)
}

// UpdateTarefa grava a conclusão ou a dispensa.
func (r *Repository) UpdateTarefa(ctx context.Context, db database.DBTX, t domain.Tarefa) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_tarefas SET estado = $2, observacao = $3, feita_por = $4, feita_em = $5 WHERE id = $1`,
		t.ID, t.Estado, t.Observacao, t.FeitaPor, t.FeitaEm)
	return mapDB(err)
}

// MembroDe diz de quais das equipes a pessoa é membro.
func (r *Repository) MembroDe(ctx context.Context, db database.DBTX, user uuid.UUID, equipes []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `SELECT equipe_id FROM atendimento_equipe_membros WHERE user_id = $1 AND equipe_id = ANY($2) ORDER BY equipe_id`, user, equipes)
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
