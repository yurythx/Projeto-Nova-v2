package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// OfertaDosEventos devolve a oferta escolhida (nil = nenhuma).
func (r *Repository) OfertaDosEventos(ctx context.Context, db database.DBTX) (*uuid.UUID, error) {
	var oferta *uuid.UUID
	err := db.QueryRow(ctx, `SELECT oferta_id FROM atendimento_eventos_config WHERE unico`).Scan(&oferta)
	if database.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, mapDB(err)
	}
	return oferta, nil
}

// SalvarOfertaDosEventos grava (ou limpa, com nil) a oferta dos alertas.
func (r *Repository) SalvarOfertaDosEventos(ctx context.Context, db database.DBTX, oferta *uuid.UUID, por uuid.UUID, agora time.Time) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_eventos_config (unico, oferta_id, updated_at, updated_by)
		VALUES (true, $1, $2, $3)
		ON CONFLICT (unico) DO UPDATE SET oferta_id = EXCLUDED.oferta_id, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by`,
		oferta, agora, por)
	return mapDB(err)
}

// TravarEvento serializa a fingerprint com um advisory lock da transação:
// duas entregas do mesmo alerta novo não têm linha para travar com
// FOR UPDATE, e sem a trava abririam dois chamados.
func (r *Repository) TravarEvento(ctx context.Context, db database.DBTX, fingerprint string) (domain.EventoVinculado, error) {
	if _, err := db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('atendimento.evento:' || $1, 0))`, fingerprint); err != nil {
		return domain.EventoVinculado{}, mapDB(err)
	}
	var e domain.EventoVinculado
	err := db.QueryRow(ctx, `SELECT fingerprint, solicitacao_id, alerta, ultimo_disparo
		FROM atendimento_eventos WHERE fingerprint = $1`, fingerprint).Scan(&e.Fingerprint, &e.SolicitacaoID, &e.Alerta, &e.UltimoDisparo)
	if err != nil {
		return domain.EventoVinculado{}, mapDB(err)
	}
	return e, nil
}

// SalvarEvento liga a fingerprint ao chamado (o novo substitui o antigo).
func (r *Repository) SalvarEvento(ctx context.Context, db database.DBTX, e domain.EventoVinculado) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_eventos (fingerprint, solicitacao_id, alerta, ultimo_disparo)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (fingerprint) DO UPDATE SET solicitacao_id = EXCLUDED.solicitacao_id, alerta = EXCLUDED.alerta,
			ultimo_disparo = EXCLUDED.ultimo_disparo`, e.Fingerprint, e.SolicitacaoID, e.Alerta, e.UltimoDisparo)
	return mapDB(err)
}
