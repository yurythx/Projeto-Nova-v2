package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const colunasMovimento = `id, tarefa_id, estado_antes, estado_depois,
	responsavel_antes_id, responsavel_antes_nome, responsavel_depois_id, responsavel_depois_nome,
	autor_id, autor_nome, created_at`

func (r *Repository) RegistrarMovimento(ctx context.Context, db database.DBTX, m *domain.Movimento) error {
	const q = `INSERT INTO projetos_tarefa_movimentos (
		id, tarefa_id, estado_antes, estado_depois,
		responsavel_antes_id, responsavel_antes_nome, responsavel_depois_id, responsavel_depois_nome,
		autor_id, autor_nome, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := db.Exec(ctx, q,
		m.ID, m.TarefaID, m.EstadoAntes, m.EstadoDepois,
		m.ResponsavelAntesID, m.ResponsavelAntesNome, m.ResponsavelDepoisID, m.ResponsavelDepoisNome,
		m.AutorID, m.AutorNome, m.CreatedAt,
	)
	return mapDB(err)
}

func (r *Repository) ListarMovimentos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Movimento, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefa_movimentos WHERE tarefa_id = $1 ORDER BY created_at DESC`, colunasMovimento)
	rows, err := db.Query(ctx, q, tarefaID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.Movimento{}
	for rows.Next() {
		var m domain.Movimento
		if err := rows.Scan(
			&m.ID, &m.TarefaID, &m.EstadoAntes, &m.EstadoDepois,
			&m.ResponsavelAntesID, &m.ResponsavelAntesNome, &m.ResponsavelDepoisID, &m.ResponsavelDepoisNome,
			&m.AutorID, &m.AutorNome, &m.CreatedAt,
		); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, m)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return lista, nil
}
