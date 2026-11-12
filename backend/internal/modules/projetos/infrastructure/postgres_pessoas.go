package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) IncluirParticipante(ctx context.Context, db database.DBTX, projetoID uuid.UUID, pessoa domain.Pessoa) error {
	const q = `INSERT INTO projetos_participantes (projeto_id, user_id, nome)
		VALUES ($1, $2, $3)
		ON CONFLICT (projeto_id, user_id) DO UPDATE SET nome = EXCLUDED.nome`
	_, err := db.Exec(ctx, q, projetoID, pessoa.ID, pessoa.Nome)
	return mapDB(err)
}

func (r *Repository) ExcluirParticipante(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) error {
	const q = `DELETE FROM projetos_participantes WHERE projeto_id = $1 AND user_id = $2`
	_, err := db.Exec(ctx, q, projetoID, pessoa)
	return mapDB(err)
}

func (r *Repository) ListarParticipantes(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Pessoa, error) {
	const q = `SELECT user_id, nome FROM projetos_participantes
		WHERE projeto_id = $1
		ORDER BY lower(nome), user_id`
	rows, err := db.Query(ctx, q, projetoID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.Pessoa{}
	for rows.Next() {
		var item domain.Pessoa
		if err := rows.Scan(&item.ID, &item.Nome); err != nil {
			return nil, mapDB(err)
		}
		item.Papel = domain.PapelParticipante
		lista = append(lista, item)
	}
	return lista, mapDB(rows.Err())
}

func (r *Repository) EhParticipante(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM projetos_participantes WHERE projeto_id = $1 AND user_id = $2)`
	var ok bool
	if err := db.QueryRow(ctx, q, projetoID, pessoa).Scan(&ok); err != nil {
		return false, mapDB(err)
	}
	return ok, nil
}

func (r *Repository) SubstituirPessoasDaTarefa(ctx context.Context, db database.DBTX, tarefaID uuid.UUID, pessoas []domain.Pessoa) error {
	if _, err := db.Exec(ctx, `DELETE FROM projetos_tarefa_pessoas WHERE tarefa_id = $1`, tarefaID); err != nil {
		return mapDB(err)
	}
	for i, pessoa := range pessoas {
		const q = `INSERT INTO projetos_tarefa_pessoas (tarefa_id, user_id, nome, ordem) VALUES ($1, $2, $3, $4)`
		if _, err := db.Exec(ctx, q, tarefaID, pessoa.ID, pessoa.Nome, i); err != nil {
			return mapDB(err)
		}
	}
	var id *uuid.UUID
	nome := ""
	if len(pessoas) > 0 {
		primeiro := pessoas[0].ID
		id = &primeiro
		nome = pessoas[0].Nome
	}
	_, err := db.Exec(ctx, `UPDATE projetos_tarefas SET responsavel_id = $2, responsavel_nome = $3, updated_at = now() WHERE id = $1`, tarefaID, id, nome)
	return mapDB(err)
}

func (r *Repository) PessoasPorTarefa(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID][]domain.Pessoa, error) {
	mapa := map[uuid.UUID][]domain.Pessoa{}
	if len(ids) == 0 {
		return mapa, nil
	}
	const q = `SELECT tarefa_id, user_id, nome FROM projetos_tarefa_pessoas
		WHERE tarefa_id = ANY($1)
		ORDER BY ordem, lower(nome)`
	rows, err := db.Query(ctx, q, ids)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	for rows.Next() {
		var tarefaID uuid.UUID
		var pessoa domain.Pessoa
		if err := rows.Scan(&tarefaID, &pessoa.ID, &pessoa.Nome); err != nil {
			return nil, mapDB(err)
		}
		mapa[tarefaID] = append(mapa[tarefaID], pessoa)
	}
	return mapa, mapDB(rows.Err())
}

func (r *Repository) RetirarPessoaDasTarefas(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) error {
	const apaga = `DELETE FROM projetos_tarefa_pessoas tp
		USING projetos_tarefas t
		WHERE tp.tarefa_id = t.id AND t.projeto_id = $1 AND tp.user_id = $2`
	if _, err := db.Exec(ctx, apaga, projetoID, pessoa); err != nil {
		return mapDB(err)
	}
	const primeiro = `UPDATE projetos_tarefas t
		SET responsavel_id = sub.user_id, responsavel_nome = sub.nome, updated_at = now()
		FROM (
			SELECT DISTINCT ON (tarefa_id) tarefa_id, user_id, nome
			FROM projetos_tarefa_pessoas
			WHERE tarefa_id IN (SELECT id FROM projetos_tarefas WHERE projeto_id = $1)
			ORDER BY tarefa_id, ordem, lower(nome)
		) sub
		WHERE t.id = sub.tarefa_id AND t.projeto_id = $1`
	if _, err := db.Exec(ctx, primeiro, projetoID); err != nil {
		return mapDB(err)
	}
	const vazias = `UPDATE projetos_tarefas t
		SET responsavel_id = NULL, responsavel_nome = '', updated_at = now()
		WHERE t.projeto_id = $1
		  AND t.responsavel_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM projetos_tarefa_pessoas tp WHERE tp.tarefa_id = t.id)`
	_, err := db.Exec(ctx, vazias, projetoID)
	return mapDB(err)
}

func preencherPessoas(ctx context.Context, r *Repository, db database.DBTX, tarefas []domain.Tarefa) ([]domain.Tarefa, error) {
	if len(tarefas) == 0 {
		return tarefas, nil
	}
	ids := make([]uuid.UUID, len(tarefas))
	for i := range tarefas {
		ids[i] = tarefas[i].ID
	}
	mapa, err := r.PessoasPorTarefa(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range tarefas {
		if grupo := mapa[tarefas[i].ID]; len(grupo) > 0 {
			tarefas[i].Pessoas = grupo
		}
	}
	return tarefas, nil
}
