package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
)

var visitaSelect = `SELECT v.id, v.solicitacao_id, v.tecnico_id, COALESCE(` + nomeDe("u") + `, ''), v.estado, v.local, v.base_legal,
	v.destino_lat, v.destino_lng, v.destino_raio_m, v.inicio_em, v.inicio_lat, v.inicio_lng, v.inicio_precisao_m,
	v.distancia_m, v.fora_da_area, v.fim_em, v.fim_lat, v.fim_lng, v.fim_precisao_m,
	v.assinante_nome, v.observacao, v.localizacao_expira_em, v.created_at, v.updated_at
	FROM atendimento_visitas v LEFT JOIN users u ON u.id = v.tecnico_id`

// ponto monta o ponto das colunas anuláveis (nulo depois da retenção).
func ponto(lat, lng, precisao *float64) *domain.Ponto {
	if lat == nil || lng == nil {
		return nil
	}
	p := domain.Ponto{Lat: *lat, Lng: *lng}
	if precisao != nil {
		p.PrecisaoM = *precisao
	}
	return &p
}

func scanVisita(row interface{ Scan(...any) error }, extra ...any) (domain.Visita, error) {
	var v domain.Visita
	var dLat, dLng, iLat, iLng, iPrec, fLat, fLng, fPrec *float64
	var raio *int
	dest := append([]any{&v.ID, &v.SolicitacaoID, &v.TecnicoID, &v.TecnicoNome, &v.Estado, &v.Local, &v.BaseLegal,
		&dLat, &dLng, &raio, &v.InicioEm, &iLat, &iLng, &iPrec,
		&v.DistanciaM, &v.ForaDaArea, &v.FimEm, &fLat, &fLng, &fPrec,
		&v.AssinanteNome, &v.Observacao, &v.LocalizacaoExpiraEm, &v.CreatedAt, &v.UpdatedAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		return domain.Visita{}, err
	}
	if dLat != nil && dLng != nil && raio != nil {
		v.Destino = &domain.Destino{Lat: *dLat, Lng: *dLng, RaioM: *raio}
	}
	v.Inicio, v.Fim = ponto(iLat, iLng, iPrec), ponto(fLat, fLng, fPrec)
	v.Evidencias = []domain.Evidencia{}
	return v, nil
}

func coordenadas(p *domain.Ponto) (lat, lng, precisao *float64) {
	if p == nil {
		return nil, nil, nil
	}
	return &p.Lat, &p.Lng, &p.PrecisaoM
}

func destino(d *domain.Destino) (lat, lng *float64, raio *int) {
	if d == nil {
		return nil, nil, nil
	}
	return &d.Lat, &d.Lng, &d.RaioM
}

// GetVisita busca a visita (sem as evidências).
func (r *Repository) GetVisita(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Visita, error) {
	v, err := scanVisita(db.QueryRow(ctx, visitaSelect+` WHERE v.id = $1`, id))
	if err != nil {
		return domain.Visita{}, mapDB(err)
	}
	return v, nil
}

// TravarVisita bloqueia a linha na transação do chamador.
func (r *Repository) TravarVisita(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	var achou uuid.UUID
	return mapDB(db.QueryRow(ctx, `SELECT id FROM atendimento_visitas WHERE id = $1 FOR UPDATE`, id).Scan(&achou))
}

// InsertVisita grava o check-in.
func (r *Repository) InsertVisita(ctx context.Context, db database.DBTX, v domain.Visita) error {
	dLat, dLng, raio := destino(v.Destino)
	iLat, iLng, iPrec := coordenadas(v.Inicio)
	_, err := db.Exec(ctx, `INSERT INTO atendimento_visitas
		(id, solicitacao_id, tecnico_id, estado, local, base_legal, destino_lat, destino_lng, destino_raio_m,
		 inicio_em, inicio_lat, inicio_lng, inicio_precisao_m, distancia_m, fora_da_area, localizacao_expira_em, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		v.ID, v.SolicitacaoID, v.TecnicoID, v.Estado, v.Local, v.BaseLegal, dLat, dLng, raio,
		v.InicioEm, iLat, iLng, iPrec, v.DistanciaM, v.ForaDaArea, v.LocalizacaoExpiraEm, v.CreatedAt, v.UpdatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.Contains(pgErr.ConstraintName, "uq_atendimento_visitas_aberta") {
		return domain.ErrVisitaAberta
	}
	return mapDB(err)
}

// UpdateVisita grava o check-out ou o cancelamento.
func (r *Repository) UpdateVisita(ctx context.Context, db database.DBTX, v domain.Visita) error {
	fLat, fLng, fPrec := coordenadas(v.Fim)
	tag, err := db.Exec(ctx, `UPDATE atendimento_visitas
		SET estado = $2, fim_em = $3, fim_lat = $4, fim_lng = $5, fim_precisao_m = $6,
			assinante_nome = $7, observacao = $8, localizacao_expira_em = $9
		WHERE id = $1`, v.ID, v.Estado, v.FimEm, fLat, fLng, fPrec, v.AssinanteNome, v.Observacao, v.LocalizacaoExpiraEm)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

const evidenciaCols = `id, visita_id, tipo, object_key, content_type, tamanho, autor_id, created_at`

func scanEvidencia(row interface{ Scan(...any) error }) (domain.Evidencia, error) {
	var e domain.Evidencia
	err := row.Scan(&e.ID, &e.VisitaID, &e.Tipo, &e.ObjectKey, &e.ContentType, &e.Tamanho, &e.AutorID, &e.CreatedAt)
	return e, err
}

// ListVisitas devolve as visitas da solicitação, com as evidências.
func (r *Repository) ListVisitas(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]domain.Visita, error) {
	rows, err := db.Query(ctx, visitaSelect+` WHERE v.solicitacao_id = $1 ORDER BY v.inicio_em, v.id`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	visitas, err := coletar(rows, func(row interface{ Scan(...any) error }) (domain.Visita, error) { return scanVisita(row) })
	if err != nil || len(visitas) == 0 {
		return visitas, err
	}
	rows, err = db.Query(ctx, `SELECT `+evidenciaCols+` FROM atendimento_evidencias
		WHERE visita_id IN (SELECT id FROM atendimento_visitas WHERE solicitacao_id = $1)
		ORDER BY created_at, id`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	evidencias, err := coletar(rows, scanEvidencia)
	if err != nil {
		return nil, err
	}
	pos := map[uuid.UUID]int{}
	for i, v := range visitas {
		pos[v.ID] = i
	}
	for _, e := range evidencias {
		i := pos[e.VisitaID]
		visitas[i].Evidencias = append(visitas[i].Evidencias, e)
	}
	return visitas, nil
}

// VisitasEmAndamento devolve as visitas abertas das unidades do filtro.
func (r *Repository) VisitasEmAndamento(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.VisitaEmCampo, error) {
	rows, err := db.Query(ctx, strings.Replace(visitaSelect, "FROM atendimento_visitas v",
		", s.numero, s.titulo, s.unidade_id FROM atendimento_visitas v JOIN atendimento_solicitacoes s ON s.id = v.solicitacao_id", 1)+`
		WHERE v.estado = 'em_andamento' AND ($1 = false OR s.unidade_id = ANY($2::uuid[]))
		ORDER BY v.inicio_em, v.id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.VisitaEmCampo, error) {
		var c domain.VisitaEmCampo
		v, err := scanVisita(row, &c.Numero, &c.Titulo, &c.UnidadeID)
		c.Visita = v
		return c, err
	})
}

// InsertEvidencia grava a evidência; a mesma chave reenviada devolve a já
// gravada.
func (r *Repository) InsertEvidencia(ctx context.Context, db database.DBTX, e domain.Evidencia) (domain.Evidencia, error) {
	out, err := scanEvidencia(db.QueryRow(ctx, `WITH novo AS (
			INSERT INTO atendimento_evidencias (id, visita_id, tipo, object_key, content_type, tamanho, autor_id, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (object_key) DO NOTHING
			RETURNING `+evidenciaCols+`)
		SELECT `+evidenciaCols+` FROM novo
		UNION ALL
		SELECT `+evidenciaCols+` FROM atendimento_evidencias WHERE object_key = $4 AND NOT EXISTS (SELECT 1 FROM novo)`,
		e.ID, e.VisitaID, e.Tipo, e.ObjectKey, e.ContentType, e.Tamanho, e.AutorID, e.CreatedAt))
	if err != nil {
		return domain.Evidencia{}, mapDB(err)
	}
	return out, nil
}

// GetEvidencia busca uma evidência.
func (r *Repository) GetEvidencia(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Evidencia, error) {
	e, err := scanEvidencia(db.QueryRow(ctx, `SELECT `+evidenciaCols+` FROM atendimento_evidencias WHERE id = $1`, id))
	if err != nil {
		return domain.Evidencia{}, mapDB(err)
	}
	return e, nil
}

// ApagarLocalizacoes anula as coordenadas das visitas com a retenção
// vencida. Distância e "fora da área" ficam: não dizem onde a pessoa estava.
func (r *Repository) ApagarLocalizacoes(ctx context.Context, db database.DBTX, agora time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `UPDATE atendimento_visitas
		SET inicio_lat = NULL, inicio_lng = NULL, inicio_precisao_m = NULL,
			fim_lat = NULL, fim_lng = NULL, fim_precisao_m = NULL,
			destino_lat = NULL, destino_lng = NULL, destino_raio_m = NULL
		WHERE localizacao_expira_em < $1
			AND (inicio_lat IS NOT NULL OR fim_lat IS NOT NULL OR destino_lat IS NOT NULL)`, agora)
	if err != nil {
		return 0, mapDB(err)
	}
	return tag.RowsAffected(), nil
}
