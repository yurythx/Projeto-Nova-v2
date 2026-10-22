package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// DadosDoTitular reúne o que o atendimento guarda sobre a pessoa.
func (r *Repository) DadosDoTitular(ctx context.Context, db database.DBTX, user uuid.UUID) (domain.DadosDoTitular, error) {
	out := domain.DadosDoTitular{}
	rows, err := db.Query(ctx, solicitacaoSelect+` WHERE s.solicitante_id = $1 ORDER BY s.created_at`, user)
	if err != nil {
		return out, mapDB(err)
	}
	if out.Solicitacoes, err = coletar(rows, scanSolicitacao); err != nil {
		return out, err
	}
	rows, err = db.Query(ctx, `SELECT c.id, c.solicitacao_id, c.autor_id, '', c.texto, c.interno, c.created_at
		FROM atendimento_comentarios c WHERE c.autor_id = $1 ORDER BY c.created_at`, user)
	if err != nil {
		return out, mapDB(err)
	}
	if out.Comentarios, err = coletar(rows, scanComentario); err != nil {
		return out, err
	}
	rows, err = db.Query(ctx, visitaSelect+` WHERE v.tecnico_id = $1 ORDER BY v.inicio_em`, user)
	if err != nil {
		return out, mapDB(err)
	}
	out.Visitas, err = coletar(rows, func(row interface{ Scan(...any) error }) (domain.Visita, error) { return scanVisita(row) })
	return out, err
}

// ApagarLocalizacoesDe anula as coordenadas das visitas feitas pela pessoa.
func (r *Repository) ApagarLocalizacoesDe(ctx context.Context, db database.DBTX, user uuid.UUID) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_visitas
		SET inicio_lat = NULL, inicio_lng = NULL, inicio_precisao_m = NULL,
			fim_lat = NULL, fim_lng = NULL, fim_precisao_m = NULL,
			destino_lat = NULL, destino_lng = NULL, destino_raio_m = NULL
		WHERE tecnico_id = $1`, user)
	return mapDB(err)
}
