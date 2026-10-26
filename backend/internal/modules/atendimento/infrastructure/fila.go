package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const regraCols = `id, unidade_id, ordem, nome, oferta_id, palavra, unidade_solicitante_id, prioridade, equipe_id, ativa, created_at, updated_at`

func scanRegra(scan func(dest ...any) error) (domain.Regra, error) {
	var r domain.Regra
	err := scan(&r.ID, &r.UnidadeID, &r.Ordem, &r.Nome, &r.OfertaID, &r.Palavra, &r.UnidadeSolicitanteID, &r.Prioridade, &r.EquipeID, &r.Ativa, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *Repository) ListarRegras(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Regra, error) {
	rows, err := db.Query(ctx, `SELECT `+regraCols+` FROM atendimento_regras
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
		ORDER BY ordem, nome, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Regra{}
	for rows.Next() {
		item, err := scanRegra(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ListarRegrasDaUnidade(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Regra, error) {
	rows, err := db.Query(ctx, `SELECT `+regraCols+` FROM atendimento_regras
		WHERE ativa AND unidade_id IS NOT DISTINCT FROM $1
		ORDER BY ordem, id`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Regra{}
	for rows.Next() {
		item, err := scanRegra(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ContarRegras(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM atendimento_regras WHERE unidade_id IS NOT DISTINCT FROM $1`, unidade).Scan(&n)
	return n, mapDB(err)
}

func (r *Repository) InserirRegra(ctx context.Context, db database.DBTX, item domain.Regra) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_regras
		(id, unidade_id, ordem, nome, oferta_id, palavra, unidade_solicitante_id, prioridade, equipe_id, ativa, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		item.ID, item.UnidadeID, item.Ordem, item.Nome, item.OfertaID, item.Palavra, item.UnidadeSolicitanteID,
		item.Prioridade, item.EquipeID, item.Ativa, item.CreatedAt, item.UpdatedAt)
	return mapDB(err)
}

func (r *Repository) ExcluirRegra(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM atendimento_regras WHERE id = $1`, id)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ObterRegra(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Regra, error) {
	item, err := scanRegra(db.QueryRow(ctx, `SELECT `+regraCols+` FROM atendimento_regras WHERE id = $1`, id).Scan)
	return item, mapDB(err)
}

func (r *Repository) ObterResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) (domain.RespostaPronta, error) {
	var item domain.RespostaPronta
	err := db.QueryRow(ctx, `SELECT oferta_id, publico, interna FROM atendimento_respostas WHERE oferta_id = $1`, oferta).
		Scan(&item.OfertaID, &item.Publico, &item.Interna)
	if database.IsNoRows(err) {
		return domain.RespostaPronta{OfertaID: oferta}, nil
	}
	return item, mapDB(err)
}

func (r *Repository) SalvarResposta(ctx context.Context, db database.DBTX, item domain.RespostaPronta) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_respostas (oferta_id, publico, interna, updated_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (oferta_id) DO UPDATE SET publico = EXCLUDED.publico, interna = EXCLUDED.interna, updated_at = EXCLUDED.updated_at`,
		item.OfertaID, item.Publico, item.Interna, time.Now().UTC())
	return mapDB(err)
}

func (r *Repository) ExcluirResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM atendimento_respostas WHERE oferta_id = $1`, oferta)
	return mapDB(err)
}

const recorrenciaCols = `id, unidade_id, oferta_id, titulo, descricao, dia, criado_por, ativa, created_at, updated_at`

func scanRecorrencia(scan func(dest ...any) error) (domain.ChamadoMensal, error) {
	var item domain.ChamadoMensal
	err := scan(&item.ID, &item.UnidadeID, &item.OfertaID, &item.Titulo, &item.Descricao, &item.Dia, &item.CriadoPor, &item.Ativa, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *Repository) ListarRecorrencias(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.ChamadoMensal, error) {
	rows, err := db.Query(ctx, `SELECT `+recorrenciaCols+` FROM atendimento_recorrencias
		WHERE ($1 = false OR unidade_id = ANY($2::uuid[]))
		ORDER BY titulo, id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.ChamadoMensal{}
	for rows.Next() {
		item, err := scanRecorrencia(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ListarRecorrenciasAtivas(ctx context.Context, db database.DBTX) ([]domain.ChamadoMensal, error) {
	rows, err := db.Query(ctx, `SELECT `+recorrenciaCols+` FROM atendimento_recorrencias WHERE ativa ORDER BY id`)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.ChamadoMensal{}
	for rows.Next() {
		item, err := scanRecorrencia(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) InserirRecorrencia(ctx context.Context, db database.DBTX, item domain.ChamadoMensal) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_recorrencias
		(id, unidade_id, oferta_id, titulo, descricao, dia, criado_por, ativa, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		item.ID, item.UnidadeID, item.OfertaID, item.Titulo, item.Descricao, item.Dia, item.CriadoPor, item.Ativa, item.CreatedAt, item.UpdatedAt)
	return mapDB(err)
}

func (r *Repository) ExcluirRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM atendimento_recorrencias WHERE id = $1`, id)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ObterRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.ChamadoMensal, error) {
	item, err := scanRecorrencia(db.QueryRow(ctx, `SELECT `+recorrenciaCols+` FROM atendimento_recorrencias WHERE id = $1`, id).Scan)
	return item, mapDB(err)
}

func (r *Repository) ReivindicarAbertura(ctx context.Context, db database.DBTX, id uuid.UUID, ano, mes int) (bool, error) {
	var marcado uuid.UUID
	err := db.QueryRow(ctx, `INSERT INTO atendimento_recorrencia_aberturas (recorrencia_id, ano, mes, tentado_em)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (recorrencia_id, ano, mes) DO UPDATE
		SET tentado_em = now()
		WHERE atendimento_recorrencia_aberturas.solicitacao_id IS NULL
		  AND atendimento_recorrencia_aberturas.tentado_em < now() - interval '15 minutes'
		RETURNING recorrencia_id`, id, ano, mes).Scan(&marcado)
	if database.IsNoRows(err) {
		return false, nil
	}
	return err == nil, mapDB(err)
}

func (r *Repository) LigarAbertura(ctx context.Context, db database.DBTX, id uuid.UUID, ano, mes int, solicitacao uuid.UUID) error {
	_, err := db.Exec(ctx, `UPDATE atendimento_recorrencia_aberturas
		SET solicitacao_id = $4
		WHERE recorrencia_id = $1 AND ano = $2 AND mes = $3 AND solicitacao_id IS NULL`, id, ano, mes, solicitacao)
	return mapDB(err)
}

func (r *Repository) SolicitacaoDoMes(ctx context.Context, db database.DBTX, oferta, solicitante uuid.UUID, titulo string, inicio time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM atendimento_solicitacoes
		WHERE oferta_id = $1 AND solicitante_id = $2 AND titulo = $3 AND created_at >= $4
		ORDER BY created_at DESC LIMIT 1`, oferta, solicitante, titulo, inicio).Scan(&id)
	if database.IsNoRows(err) {
		return uuid.Nil, nil
	}
	return id, mapDB(err)
}

func (r *Repository) RegistrarDeflexao(ctx context.Context, db database.DBTX, usuario uuid.UUID, artigo, dia string, unidade *uuid.UUID, resolveu bool) error {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_deflexoes (usuario_id, artigo_id, dia, unidade_id, resolveu, created_at)
		VALUES ($1, $2, $3::date, $4, $5, now())
		ON CONFLICT (usuario_id, artigo_id, dia) DO UPDATE
		SET resolveu = atendimento_deflexoes.resolveu OR EXCLUDED.resolveu`,
		usuario, artigo, dia, unidade, resolveu)
	return mapDB(err)
}

func (r *Repository) MedirDeflexao(ctx context.Context, db database.DBTX, desde string, restrito bool, unidades []uuid.UUID) (int, int, error) {
	var viram, sem int
	err := db.QueryRow(ctx, `WITH vistas AS (
			SELECT usuario_id, dia FROM atendimento_deflexoes
			WHERE dia >= $1::date AND ($2 = false OR unidade_id = ANY($3::uuid[]))
		), pessoas AS (
			SELECT usuario_id FROM vistas GROUP BY usuario_id
		), sem AS (
			SELECT p.usuario_id FROM pessoas p
			WHERE NOT EXISTS (
				SELECT 1 FROM vistas v
				JOIN atendimento_solicitacoes s ON s.solicitante_id = v.usuario_id
				WHERE v.usuario_id = p.usuario_id
				  AND (s.created_at AT TIME ZONE 'America/Sao_Paulo')::date = v.dia
			)
		)
		SELECT (SELECT count(*) FROM pessoas), (SELECT count(*) FROM sem)`, desde, restrito, idsOrEmpty(unidades)).Scan(&viram, &sem)
	return viram, sem, mapDB(err)
}
