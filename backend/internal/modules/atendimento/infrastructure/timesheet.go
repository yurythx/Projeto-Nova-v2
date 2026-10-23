package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// ListApontamentos devolve todos os apontamentos de uma solicitação em ordem cronológica reversa.
func (r *Repository) ListApontamentos(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]domain.Apontamento, error) {
	rows, err := db.Query(ctx, `SELECT id, solicitacao_id, usuario_id, usuario_nome,
		to_char(data_apontamento, 'YYYY-MM-DD'), horas, descricao, created_at
		FROM atendimento_apontamentos
		WHERE solicitacao_id = $1
		ORDER BY data_apontamento DESC, created_at DESC`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Apontamento
	for rows.Next() {
		var a domain.Apontamento
		if err := rows.Scan(
			&a.ID, &a.SolicitacaoID, &a.UsuarioID, &a.UsuarioNome,
			&a.DataApontamento, &a.Horas, &a.Descricao, &a.CreatedAt,
		); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, a)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return lista, nil
}

// InsertApontamento grava um novo registro de horas na solicitação.
func (r *Repository) InsertApontamento(ctx context.Context, db database.DBTX, a domain.Apontamento) (domain.Apontamento, error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.DataApontamento == "" {
		a.DataApontamento = time.Now().Format("2006-01-02")
	}

	var dataRet string
	err := db.QueryRow(ctx, `INSERT INTO atendimento_apontamentos
		(id, solicitacao_id, usuario_id, usuario_nome, data_apontamento, horas, descricao, created_at)
		VALUES ($1, $2, $3, $4, $5::date, $6, $7, $8)
		RETURNING to_char(data_apontamento, 'YYYY-MM-DD')`,
		a.ID, a.SolicitacaoID, a.UsuarioID, a.UsuarioNome, a.DataApontamento, a.Horas, a.Descricao, a.CreatedAt).Scan(&dataRet)
	if err != nil {
		return domain.Apontamento{}, mapDB(err)
	}
	a.DataApontamento = dataRet
	return a, nil
}

// DeleteApontamento remove um apontamento específico da solicitação.
func (r *Repository) DeleteApontamento(ctx context.Context, db database.DBTX, solicitacaoID, apontamentoID uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM atendimento_apontamentos WHERE id = $1 AND solicitacao_id = $2`, apontamentoID, solicitacaoID)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}
