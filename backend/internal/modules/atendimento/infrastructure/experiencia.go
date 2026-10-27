package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// MetasXLA devolve as metas de experiência; sem linha, todas nulas.
func (r *Repository) MetasXLA(ctx context.Context, db database.DBTX) (domain.MetasXLA, error) {
	var m domain.MetasXLA
	err := db.QueryRow(ctx, `SELECT csat_min::float8, facilidade_min::float8, reabertura_max::float8, idas_max::float8, espera_max_horas, updated_at
		FROM atendimento_xla WHERE id`).Scan(&m.CSATMin, &m.FacilidadeMin, &m.ReaberturaMax, &m.IdasMax, &m.EsperaMaxHoras, &m.UpdatedAt)
	if database.IsNoRows(err) {
		return domain.MetasXLA{}, nil
	}
	return m, mapDB(err)
}

// SalvarMetasXLA grava a linha única das metas.
func (r *Repository) SalvarMetasXLA(ctx context.Context, db database.DBTX, m domain.MetasXLA, por uuid.UUID, agora time.Time) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_xla (id, csat_min, facilidade_min, reabertura_max, idas_max, espera_max_horas, updated_by, updated_at)
		VALUES (true, $1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET csat_min = $1, facilidade_min = $2, reabertura_max = $3, idas_max = $4,
			espera_max_horas = $5, updated_by = $6, updated_at = $7`,
		m.CSATMin, m.FacilidadeMin, m.ReaberturaMax, m.IdasMax, m.EsperaMaxHoras, por, agora)
	return mapDB(err)
}
