package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const colunasAtualizacao = `id, tarefa_id, texto, autor_id, autor_nome, created_at`

func (r *Repository) CriarAtualizacao(ctx context.Context, db database.DBTX, a *domain.Atualizacao) error {
	const q = `INSERT INTO projetos_tarefa_atualizacoes (
		id, tarefa_id, texto, autor_id, autor_nome, created_at
	) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(ctx, q, a.ID, a.TarefaID, a.Texto, a.AutorID, a.AutorNome, a.CreatedAt)
	return mapDB(err)
}

func (r *Repository) ListarAtualizacoes(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Atualizacao, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefa_atualizacoes WHERE tarefa_id = $1 ORDER BY created_at`, colunasAtualizacao)
	rows, err := db.Query(ctx, q, tarefaID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.Atualizacao{}
	for rows.Next() {
		var a domain.Atualizacao
		if err := rows.Scan(&a.ID, &a.TarefaID, &a.Texto, &a.AutorID, &a.AutorNome, &a.CreatedAt); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, a)
	}
	return lista, mapDB(rows.Err())
}

func (r *Repository) ListarPrazos(ctx context.Context, db database.DBTX, mes string, restrito bool, unidades []uuid.UUID) ([]domain.PrazoAgenda, error) {
	if unidades == nil {
		unidades = []uuid.UUID{}
	}
	q := `SELECT t.id, t.titulo, t.estado, t.data_limite, t.responsavel_id, t.responsavel_nome, t.pai_id,
			p.id, p.numero, p.titulo
		FROM projetos_tarefas t
		JOIN projetos p ON p.id = t.projeto_id
		WHERE t.data_limite IS NOT NULL
		  AND p.estado NOT IN ('concluido', 'cancelado')
		  AND (
		    to_char(t.data_limite AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM') = $1
		    OR (
		      t.estado <> 'concluida'
		      AND (t.data_limite AT TIME ZONE 'America/Sao_Paulo')::date < (now() AT TIME ZONE 'America/Sao_Paulo')::date
		      AND to_char(t.data_limite AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM') <> $1
		    )
		  )`
	args := []any{mes}
	if restrito {
		q += clausulaUnidade(true, 2)
		args = append(args, unidades)
	}
	q += ` ORDER BY t.data_limite, p.numero, t.ordem`
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.PrazoAgenda{}
	for rows.Next() {
		var item domain.PrazoAgenda
		if err := rows.Scan(
			&item.ID, &item.Titulo, &item.Estado, &item.DataLimite, &item.ResponsavelID, &item.ResponsavelNome, &item.PaiID,
			&item.ProjetoID, &item.ProjetoNumero, &item.ProjetoTitulo,
		); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	ids := make([]uuid.UUID, len(lista))
	for i := range lista {
		ids[i] = lista[i].ID
	}
	mapa, err := r.PessoasPorTarefa(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range lista {
		if grupo := mapa[lista[i].ID]; len(grupo) > 0 {
			lista[i].Pessoas = grupo
		}
	}
	return lista, nil
}
