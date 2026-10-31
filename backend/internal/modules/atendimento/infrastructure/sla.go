package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const slaCols = `id, nome, descricao, tempo_resposta_minutos, tempo_solucao_minutos, pausar_em_espera, calendario_tipo, unidade_id, ativa, created_at, updated_at`

func scanSLA(row interface{ Scan(...any) error }) (domain.SLA, error) {
	var s domain.SLA
	err := row.Scan(
		&s.ID, &s.Nome, &s.Descricao, &s.TempoRespostaMinutos, &s.TempoSolucaoMinutos,
		&s.PausarEmEspera, &s.CalendarioTipo, &s.UnidadeID, &s.Ativa, &s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

// ListSLAs lista as SLAs ativas e disponíveis para a unidade (ou institucionais quando unidadeID é nil).
func (r *Repository) ListSLAs(ctx context.Context, db database.DBTX, unidadeID *uuid.UUID) ([]domain.SLA, error) {
	var query string
	var args []any

	if unidadeID == nil {
		query = `SELECT ` + slaCols + ` FROM atendimento_slas WHERE ativa ORDER BY tempo_solucao_minutos ASC, nome ASC`
	} else {
		query = `SELECT ` + slaCols + ` FROM atendimento_slas WHERE ativa AND (unidade_id IS NULL OR unidade_id = $1) ORDER BY tempo_solucao_minutos ASC, nome ASC`
		args = append(args, *unidadeID)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSLA)
}

// GetSLA busca uma SLA por ID.
func (r *Repository) GetSLA(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.SLA, error) {
	s, err := scanSLA(db.QueryRow(ctx, `SELECT `+slaCols+` FROM atendimento_slas WHERE id = $1`, id))
	return s, mapDB(err)
}

// InsertSLA cria uma nova SLA.
func (r *Repository) InsertSLA(ctx context.Context, db database.DBTX, s domain.SLA) (domain.SLA, error) {
	out, err := scanSLA(db.QueryRow(ctx, `INSERT INTO atendimento_slas (`+slaCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+slaCols,
		s.ID, s.Nome, s.Descricao, s.TempoRespostaMinutos, s.TempoSolucaoMinutos,
		s.PausarEmEspera, s.CalendarioTipo, s.UnidadeID, s.Ativa, s.CreatedAt, s.UpdatedAt,
	))
	return out, mapDB(err)
}

// UpdateSLA atualiza uma SLA existente.
func (r *Repository) UpdateSLA(ctx context.Context, db database.DBTX, s domain.SLA) (domain.SLA, error) {
	out, err := scanSLA(db.QueryRow(ctx, `UPDATE atendimento_slas
		SET nome = $2, descricao = $3, tempo_resposta_minutos = $4, tempo_solucao_minutos = $5,
		    pausar_em_espera = $6, calendario_tipo = $7, unidade_id = $8, ativa = $9, updated_at = now()
		WHERE id = $1
		RETURNING `+slaCols,
		s.ID, s.Nome, s.Descricao, s.TempoRespostaMinutos, s.TempoSolucaoMinutos,
		s.PausarEmEspera, s.CalendarioTipo, s.UnidadeID, s.Ativa,
	))
	return out, mapDB(err)
}

// DeleteSLA desativa ou remove a SLA.
func (r *Repository) DeleteSLA(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_slas SET ativa = false, updated_at = now() WHERE id = $1`, id)
	return mapDB(err)
}

// VincularCategoriaSLA altera a SLA vinculada a uma categoria (ou desvincula se slaID for nil).
func (r *Repository) VincularCategoriaSLA(ctx context.Context, db database.DBTX, categoriaID uuid.UUID, slaID *uuid.UUID) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_categorias SET sla_id = $2, updated_at = now() WHERE id = $1`, categoriaID, slaID)
	return mapDB(err)
}

