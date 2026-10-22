package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
)

// acima é a unidade e as que estão acima dela na árvore ($1 nulo = vazio).
// O limite de níveis protege de um ciclo que o cadastro deixe passar.
const acima = `WITH RECURSIVE acima (id, nivel) AS (
	SELECT $1::uuid, 0 WHERE $1::uuid IS NOT NULL
	UNION ALL
	SELECT u.parent_id, a.nivel + 1 FROM unidades u JOIN acima a ON u.id = a.id
	WHERE u.parent_id IS NOT NULL AND a.nivel < 32
) `

const politicaCols = `p.id, p.unidade_id, p.fuso, p.turnos, p.matriz, p.metas, p.created_at, p.updated_at`

func scanPolitica(row interface{ Scan(...any) error }) (domain.Politica, error) {
	var p domain.Politica
	var turnos, matriz, metas []byte
	if err := row.Scan(&p.ID, &p.UnidadeID, &p.Fuso, &turnos, &matriz, &metas, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return domain.Politica{}, err
	}
	if err := errors.Join(json.Unmarshal(turnos, &p.Turnos), json.Unmarshal(matriz, &p.Matriz), json.Unmarshal(metas, &p.Metas)); err != nil {
		return domain.Politica{}, fmt.Errorf("atendimento: política ilegível: %w", err)
	}
	return p, nil
}

const feriadoCols = `id, unidade_id, to_char(data, 'YYYY-MM-DD'), nome, tipo, created_at`

func scanFeriado(row interface{ Scan(...any) error }) (domain.Feriado, error) {
	var f domain.Feriado
	err := row.Scan(&f.ID, &f.UnidadeID, &f.Data, &f.Nome, &f.Tipo, &f.CreatedAt)
	return f, err
}

// GetPolitica busca a política da própria unidade.
func (r *Repository) GetPolitica(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (domain.Politica, error) {
	p, err := scanPolitica(db.QueryRow(ctx, `SELECT `+politicaCols+` FROM atendimento_politicas p
		WHERE p.unidade_id IS NOT DISTINCT FROM $1`, unidade))
	if err != nil {
		return domain.Politica{}, mapDB(err)
	}
	return p, nil
}

// ResolverPolitica sobe a árvore até achar uma política; no topo, a
// institucional.
func (r *Repository) ResolverPolitica(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (domain.Politica, error) {
	p, err := scanPolitica(db.QueryRow(ctx, acima+`SELECT `+politicaCols+` FROM atendimento_politicas p
		LEFT JOIN acima a ON a.id = p.unidade_id
		WHERE p.unidade_id IS NULL OR a.id IS NOT NULL
		ORDER BY p.unidade_id IS NULL, a.nivel
		LIMIT 1`, unidade))
	if err != nil {
		return domain.Politica{}, mapDB(err)
	}
	p.Herdada = !domain.MesmaUnidade(p.UnidadeID, unidade)
	return p, nil
}

// SalvarPolitica grava a política da unidade, criando ou substituindo.
func (r *Repository) SalvarPolitica(ctx context.Context, db database.DBTX, p domain.Politica) (domain.Politica, error) {
	turnos, _ := json.Marshal(p.Turnos)
	matriz, _ := json.Marshal(p.Matriz)
	metas, _ := json.Marshal(p.Metas)
	out, err := scanPolitica(db.QueryRow(ctx, `INSERT INTO atendimento_politicas AS p (id, unidade_id, fuso, turnos, matriz, metas)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT ON CONSTRAINT atendimento_politicas_unidade_uq
		DO UPDATE SET fuso = EXCLUDED.fuso, turnos = EXCLUDED.turnos, matriz = EXCLUDED.matriz, metas = EXCLUDED.metas
		RETURNING `+politicaCols, p.ID, p.UnidadeID, p.Fuso, turnos, matriz, metas))
	if err != nil {
		return domain.Politica{}, mapDB(err)
	}
	return out, nil
}

// ListFeriados devolve os feriados cadastrados na própria unidade, por data.
func (r *Repository) ListFeriados(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Feriado, error) {
	rows, err := db.Query(ctx, `SELECT `+feriadoCols+` FROM atendimento_feriados
		WHERE unidade_id IS NOT DISTINCT FROM $1 ORDER BY data`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanFeriado)
}

// FeriadosVigentes devolve os feriados que valem para a unidade no período.
func (r *Repository) FeriadosVigentes(ctx context.Context, db database.DBTX, unidade *uuid.UUID, de, ate string) ([]domain.Feriado, error) {
	rows, err := db.Query(ctx, acima+`SELECT `+feriadoCols+` FROM atendimento_feriados
		WHERE (unidade_id IS NULL OR unidade_id IN (SELECT id FROM acima))
			AND data BETWEEN $2::date AND $3::date
		ORDER BY data`, unidade, de, ate)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanFeriado)
}

// GetFeriado busca um feriado.
func (r *Repository) GetFeriado(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Feriado, error) {
	f, err := scanFeriado(db.QueryRow(ctx, `SELECT `+feriadoCols+` FROM atendimento_feriados WHERE id = $1`, id))
	if err != nil {
		return domain.Feriado{}, mapDB(err)
	}
	return f, nil
}

// InsertFeriado grava o feriado; a mesma data na mesma unidade é recusada.
func (r *Repository) InsertFeriado(ctx context.Context, db database.DBTX, f domain.Feriado) (domain.Feriado, error) {
	out, err := scanFeriado(db.QueryRow(ctx, `INSERT INTO atendimento_feriados (id, unidade_id, data, nome, tipo, created_at)
		VALUES ($1, $2, $3::date, $4, $5, $6) RETURNING `+feriadoCols,
		f.ID, f.UnidadeID, f.Data, f.Nome, f.Tipo, f.CreatedAt))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.Contains(pgErr.ConstraintName, "dia_uq") {
		return domain.Feriado{}, domain.ErrFeriadoRepetido
	}
	if err != nil {
		return domain.Feriado{}, mapDB(err)
	}
	return out, nil
}

// DeleteFeriado remove o feriado.
func (r *Repository) DeleteFeriado(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM atendimento_feriados WHERE id = $1`, id)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

// PassagensDe agrupa as passagens das solicitações dadas.
func (r *Repository) PassagensDe(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID][]domain.Passagem, error) {
	rows, err := db.Query(ctx, `SELECT p.id, p.solicitacao_id, p.de, p.para, p.autor_id, '', p.em
		FROM atendimento_passagens p WHERE p.solicitacao_id = ANY($1::uuid[]) ORDER BY p.em, p.id`, idsOrEmpty(ids))
	if err != nil {
		return nil, mapDB(err)
	}
	lista, err := coletar(rows, scanPassagem)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID][]domain.Passagem{}
	for _, p := range lista {
		out[p.SolicitacaoID] = append(out[p.SolicitacaoID], p)
	}
	return out, nil
}

// ComPrazoCorrendo devolve as solicitações classificadas com o relógio vivo.
func (r *Repository) ComPrazoCorrendo(ctx context.Context, db database.DBTX) ([]domain.Solicitacao, error) {
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE s.prioridade IS NOT NULL AND s.estado IN ('aberto', 'em_atendimento')
		ORDER BY s.created_at`)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}
