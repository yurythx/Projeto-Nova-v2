package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const colunasAnexo = `id, tarefa_id, nome, object_key, content_type, tamanho, autor_id, autor_nome, created_at`

func scanAnexo(row interface{ Scan(...any) error }) (domain.Anexo, error) {
	var a domain.Anexo
	err := row.Scan(&a.ID, &a.TarefaID, &a.Nome, &a.ObjectKey, &a.ContentType, &a.Tamanho, &a.AutorID, &a.AutorNome, &a.CreatedAt)
	return a, err
}

func (r *Repository) ListarAnexos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Anexo, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefa_anexos WHERE tarefa_id = $1 ORDER BY created_at`, colunasAnexo)
	rows, err := db.Query(ctx, q, tarefaID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.Anexo{}
	for rows.Next() {
		item, err := scanAnexo(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return lista, nil
}

func (r *Repository) ObterAnexo(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Anexo, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefa_anexos WHERE id = $1`, colunasAnexo)
	a, err := scanAnexo(db.QueryRow(ctx, q, id))
	if err != nil {
		if database.IsNoRows(err) {
			return nil, domain.ErrAnexoNaoEncontrado
		}
		return nil, mapDB(err)
	}
	return &a, nil
}

func (r *Repository) CriarAnexo(ctx context.Context, db database.DBTX, a *domain.Anexo) (*domain.Anexo, error) {
	q := fmt.Sprintf(`WITH novo AS (
		INSERT INTO projetos_tarefa_anexos (id, tarefa_id, nome, object_key, content_type, tamanho, autor_id, autor_nome, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (object_key) DO NOTHING
		RETURNING %s)
		SELECT %s FROM novo
		UNION ALL
		SELECT %s FROM projetos_tarefa_anexos WHERE object_key = $4 AND NOT EXISTS (SELECT 1 FROM novo)`, colunasAnexo, colunasAnexo, colunasAnexo)
	gravado, err := scanAnexo(db.QueryRow(ctx, q, a.ID, a.TarefaID, a.Nome, a.ObjectKey, a.ContentType, a.Tamanho, a.AutorID, a.AutorNome, a.CreatedAt))
	if err != nil {
		return nil, mapDB(err)
	}
	return &gravado, nil
}
