package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) UnidadesComSolicitacao(ctx context.Context, db database.DBTX) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT unidade_id FROM atendimento_solicitacoes
		WHERE unidade_id IS NOT NULL ORDER BY unidade_id`)
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

func (r *Repository) GravarRetrato(ctx context.Context, db database.DBTX, p domain.Retrato) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_indicadores_dia (
		dia, unidade_id, mttr_horas, mtbf_horas, sla_percentual, fcr_percentual, csat_media,
		soma_mttr_min, mttr_n, sla_cumprido, sla_n, fcr_n, concluidas, incidentes, csat_soma, csat_n, truncado)
		VALUES ($1::date,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT ON CONSTRAINT atendimento_indicadores_dia_uq DO UPDATE SET
			mttr_horas = EXCLUDED.mttr_horas, mtbf_horas = EXCLUDED.mtbf_horas,
			sla_percentual = EXCLUDED.sla_percentual, fcr_percentual = EXCLUDED.fcr_percentual,
			csat_media = EXCLUDED.csat_media, soma_mttr_min = EXCLUDED.soma_mttr_min,
			mttr_n = EXCLUDED.mttr_n, sla_cumprido = EXCLUDED.sla_cumprido, sla_n = EXCLUDED.sla_n,
			fcr_n = EXCLUDED.fcr_n, concluidas = EXCLUDED.concluidas, incidentes = EXCLUDED.incidentes,
			csat_soma = EXCLUDED.csat_soma, csat_n = EXCLUDED.csat_n, truncado = EXCLUDED.truncado,
			atualizado_em = now()`,
		p.Dia.UTC().Format("2006-01-02"), p.UnidadeID, p.MTTRHoras, p.MTBFHoras, p.SLAPercentual, p.FCRPercentual, p.CSATMedia,
		p.SomaMTTRMin, p.MTTRN, p.SLACumprido, p.SLAN, p.FCRN, p.Concluidas, p.Incidentes, p.CSATSoma, p.CSATN, p.Truncado)
	return mapDB(err)
}

func (r *Repository) ListarRetratos(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]domain.Retrato, error) {
	rows, err := db.Query(ctx, `SELECT dia, unidade_id, mttr_horas, mtbf_horas, sla_percentual, fcr_percentual, csat_media,
		soma_mttr_min, mttr_n, sla_cumprido, sla_n, fcr_n, concluidas, incidentes, csat_soma, csat_n, truncado
		FROM atendimento_indicadores_dia
		WHERE dia >= $1::date AND (($2 = false AND unidade_id IS NULL) OR ($2 = true AND unidade_id = ANY($3::uuid[])))
		ORDER BY dia, unidade_id`, desde.UTC().Format("2006-01-02"), restrito, idsOrEmpty(unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Retrato{}
	for rows.Next() {
		var p domain.Retrato
		if err := rows.Scan(&p.Dia, &p.UnidadeID, &p.MTTRHoras, &p.MTBFHoras, &p.SLAPercentual, &p.FCRPercentual, &p.CSATMedia,
			&p.SomaMTTRMin, &p.MTTRN, &p.SLACumprido, &p.SLAN, &p.FCRN, &p.Concluidas, &p.Incidentes, &p.CSATSoma, &p.CSATN, &p.Truncado); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, p)
	}
	return out, mapDB(rows.Err())
}
