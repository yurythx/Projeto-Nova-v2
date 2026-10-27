package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

var avaliacaoSelect = `SELECT a.id, a.solicitacao_id, a.autor_id, ` + nomeDe("u") + `, a.nota, a.facilidade, a.comentario, a.created_at,
	s.numero, s.titulo
	FROM atendimento_avaliacoes a
	JOIN atendimento_solicitacoes s ON s.id = a.solicitacao_id
	JOIN users u ON u.id = a.autor_id`

func scanAvaliacao(row interface{ Scan(...any) error }) (domain.Avaliacao, error) {
	var a domain.Avaliacao
	err := row.Scan(&a.ID, &a.SolicitacaoID, &a.AutorID, &a.AutorNome, &a.Nota, &a.Facilidade, &a.Comentario, &a.CreatedAt, &a.Numero, &a.Titulo)
	return a, err
}

// GetAvaliacao devolve a avaliação da solicitação.
func (r *Repository) GetAvaliacao(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) (domain.Avaliacao, error) {
	a, err := scanAvaliacao(db.QueryRow(ctx, avaliacaoSelect+` WHERE a.solicitacao_id = $1`, solicitacao))
	if err != nil {
		return domain.Avaliacao{}, mapDB(err)
	}
	return a, nil
}

// InsertAvaliacao grava a nota. A segunda da mesma solicitação é recusada.
func (r *Repository) InsertAvaliacao(ctx context.Context, db database.DBTX, a domain.Avaliacao) (domain.Avaliacao, error) {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_avaliacoes
		(id, solicitacao_id, autor_id, nota, facilidade, comentario, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.SolicitacaoID, a.AutorID, a.Nota, a.Facilidade, a.Comentario, a.CreatedAt)
	if database.IsUniqueViolation(err) {
		return domain.Avaliacao{}, domain.ErrAvaliacaoExistente
	}
	if err != nil {
		return domain.Avaliacao{}, mapDB(err)
	}
	return r.GetAvaliacao(ctx, db, a.SolicitacaoID)
}

// ResumoAvaliacoes agrega as notas visíveis no escopo da fila.
func (r *Repository) ResumoAvaliacoes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, recentes int) (domain.ResumoAvaliacoes, error) {
	var out domain.ResumoAvaliacoes
	err := db.QueryRow(ctx, `SELECT count(*), COALESCE(sum(a.nota), 0), count(a.facilidade), COALESCE(sum(a.facilidade), 0)
		FROM atendimento_avaliacoes a
		JOIN atendimento_solicitacoes s ON s.id = a.solicitacao_id
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[]))`, restrito, idsOrEmpty(unidades)).Scan(&out.Total, &out.Soma, &out.TotalFacilidade, &out.SomaFacilidade)
	if err != nil {
		return domain.ResumoAvaliacoes{}, mapDB(err)
	}
	if recentes <= 0 || out.Total == 0 {
		out.Recentes = []domain.Avaliacao{}
		return out, nil
	}
	rows, err := db.Query(ctx, avaliacaoSelect+`
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[]))
		ORDER BY a.created_at DESC, a.id
		LIMIT $3`, restrito, idsOrEmpty(unidades), recentes)
	if err != nil {
		return domain.ResumoAvaliacoes{}, mapDB(err)
	}
	out.Recentes, err = coletar(rows, scanAvaliacao)
	return out, err
}

// ListarParaGovernanca lê as solicitações do escopo, da mais antiga à mais nova.
func (r *Repository) ListarParaGovernanca(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, limite int) ([]domain.Solicitacao, error) {
	rows, err := db.Query(ctx, solicitacaoSelect+`
		WHERE ($1 = false OR s.unidade_id = ANY($2::uuid[]))
		ORDER BY s.created_at ASC, s.numero ASC
		LIMIT $3`, restrito, idsOrEmpty(unidades), limite)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanSolicitacao)
}
