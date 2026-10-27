package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// EtapasDaOferta devolve as etapas em ordem (vazio = sem aprovação).
func (r *Repository) EtapasDaOferta(ctx context.Context, db database.DBTX, oferta uuid.UUID) ([]domain.EtapaAprovacao, error) {
	rows, err := db.Query(ctx, `SELECT ordem, nome, aprovadores FROM atendimento_oferta_etapas WHERE oferta_id = $1 ORDER BY ordem`, oferta)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.EtapaAprovacao{}
	for rows.Next() {
		var e domain.EtapaAprovacao
		if err := rows.Scan(&e.Ordem, &e.Nome, &e.Aprovadores); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, e)
	}
	return out, mapDB(rows.Err())
}

// SalvarEtapas troca as etapas da oferta inteiras.
func (r *Repository) SalvarEtapas(ctx context.Context, db database.DBTX, oferta uuid.UUID, etapas []domain.EtapaAprovacao) error {
	if _, err := db.Exec(ctx, `DELETE FROM atendimento_oferta_etapas WHERE oferta_id = $1`, oferta); err != nil {
		return mapDB(err)
	}
	for _, e := range etapas {
		if _, err := db.Exec(ctx, `INSERT INTO atendimento_oferta_etapas (oferta_id, ordem, nome, aprovadores) VALUES ($1, $2, $3, $4)`,
			oferta, e.Ordem, e.Nome, e.Aprovadores); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

// InsertAprovacao grava a decisão.
func (r *Repository) InsertAprovacao(ctx context.Context, db database.DBTX, a domain.Aprovacao) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_aprovacoes (id, solicitacao_id, etapa, etapa_nome, aprovador_id, decisao, motivo, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, a.ID, a.SolicitacaoID, a.Etapa, a.EtapaNome, a.AprovadorID, a.Decisao, a.Motivo, a.CreatedAt)
	return mapDB(err)
}

// ListAprovacoes devolve as decisões do pedido, em ordem.
func (r *Repository) ListAprovacoes(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]domain.Aprovacao, error) {
	rows, err := db.Query(ctx, `SELECT a.id, a.solicitacao_id, a.etapa, a.etapa_nome, a.aprovador_id, COALESCE(`+nomeDe("u")+`, ''),
			a.decisao, a.motivo, a.created_at
		FROM atendimento_aprovacoes a LEFT JOIN users u ON u.id = a.aprovador_id
		WHERE a.solicitacao_id = $1 ORDER BY a.created_at, a.id`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Aprovacao{}
	for rows.Next() {
		var a domain.Aprovacao
		if err := rows.Scan(&a.ID, &a.SolicitacaoID, &a.Etapa, &a.EtapaNome, &a.AprovadorID, &a.AprovadorNome, &a.Decisao, &a.Motivo, &a.CreatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, a)
	}
	return out, mapDB(rows.Err())
}

// AguardandoAprovacaoDe lista os pedidos cuja etapa atual a pessoa aprova.
func (r *Repository) AguardandoAprovacaoDe(ctx context.Context, db database.DBTX, aprovador uuid.UUID, limite int) ([]domain.Solicitacao, error) {
	rows, err := db.Query(ctx, solicitacaoSelect+`
		JOIN atendimento_oferta_etapas e ON e.oferta_id = s.oferta_id AND e.ordem = s.etapa_atual
		WHERE s.estado = 'aguardando_aprovacao' AND $1 = ANY(e.aprovadores) AND s.solicitante_id <> $1
		ORDER BY s.created_at LIMIT $2`, aprovador, limite)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Solicitacao{}
	for rows.Next() {
		s, err := scanSolicitacao(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, s)
	}
	return out, mapDB(rows.Err())
}
