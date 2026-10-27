package infrastructure

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// nomeDe é o nome de exibição do usuário no alias dado.
func nomeDe(alias string) string {
	return `COALESCE(NULLIF(btrim(` + alias + `.display_name), ''), ` + alias + `.username)`
}

var solicitacaoSelect = `SELECT s.id, s.numero, s.tipo, s.estado, s.titulo, s.descricao, s.oferta_id, s.produto_id,
	s.unidade_id, s.solicitante_id, s.responsavel_id, s.resolvida_em, s.encerrada_em, s.created_at, s.updated_at,
	s.impacto, s.urgencia, s.prioridade, s.meta_resposta_min, s.meta_solucao_min, s.respostas, s.cofre_segredo_id, s.etapa_atual,
	o.nome, ` + nomeDe("us") + `, COALESCE(` + nomeDe("ur") + `, '')
	FROM atendimento_solicitacoes s
	JOIN atendimento_ofertas o ON o.id = s.oferta_id
	JOIN users us ON us.id = s.solicitante_id
	LEFT JOIN users ur ON ur.id = s.responsavel_id`

// Sem solicitante ($1), é a fila: recorte de unidade, mais antigas
// primeiro. Com solicitante, é a lista de quem abriu, mais novas primeiro.
const solicitacaoWhere = `
	WHERE ($1::uuid IS NULL OR s.solicitante_id = $1)
		AND ($1::uuid IS NOT NULL OR $2 = false OR s.unidade_id = ANY($3::uuid[]))
		AND ($4 = '' OR s.tipo = $4)
		AND (CASE WHEN $5 <> '' THEN s.estado = $5
			WHEN $6 THEN true
			ELSE s.estado NOT IN ('encerrado', 'cancelado') END)
		AND ($7::uuid IS NULL OR s.responsavel_id = $7)
		AND ($8 = false OR s.responsavel_id IS NULL)
		AND ($9 = '' OR s.titulo ILIKE $9 ESCAPE E'\\'
			OR s.descricao ILIKE $9 ESCAPE E'\\'
			OR s.numero::text ILIKE $9 ESCAPE E'\\')`

// padraoLike vazio não filtra. Com texto, casa o trecho e trata % e _ como
// letra, não como curinga.
func padraoLike(texto string) string {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return ""
	}
	texto = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(texto)
	return "%" + texto + "%"
}

func scanSolicitacao(row interface{ Scan(...any) error }) (domain.Solicitacao, error) {
	var s domain.Solicitacao
	var raw []byte
	err := row.Scan(&s.ID, &s.Numero, &s.Tipo, &s.Estado, &s.Titulo, &s.Descricao, &s.OfertaID, &s.ProdutoID,
		&s.UnidadeID, &s.SolicitanteID, &s.ResponsavelID, &s.ResolvidaEm, &s.EncerradaEm, &s.CreatedAt, &s.UpdatedAt,
		&s.Impacto, &s.Urgencia, &s.Prioridade, &s.MetaRespostaMin, &s.MetaSolucaoMin, &raw, &s.CofreSegredoID, &s.EtapaAtual,
		&s.OfertaNome, &s.SolicitanteNome, &s.ResponsavelNome)
	if err != nil {
		return domain.Solicitacao{}, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &s.Respostas); err != nil {
			return domain.Solicitacao{}, err
		}
	}
	if s.Respostas == nil {
		s.Respostas = map[string]string{}
	}
	return s, nil
}

func scanPassagem(row interface{ Scan(...any) error }) (domain.Passagem, error) {
	var p domain.Passagem
	err := row.Scan(&p.ID, &p.SolicitacaoID, &p.De, &p.Para, &p.AutorID, &p.AutorNome, &p.Em)
	return p, err
}

func scanComentario(row interface{ Scan(...any) error }) (domain.Comentario, error) {
	var c domain.Comentario
	err := row.Scan(&c.ID, &c.SolicitacaoID, &c.AutorID, &c.AutorNome, &c.Texto, &c.Interno, &c.CreatedAt)
	return c, err
}

// coletar lê todas as linhas com scan.
func coletar[T any](rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}, scan func(interface{ Scan(...any) error }) (T, error)) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return out, nil
}

// ListOfertasAtivas devolve o catálogo que o servidor vê, por nome.
func (r *Repository) ListOfertasAtivas(ctx context.Context, db database.DBTX) ([]domain.Oferta, error) {
	rows, err := db.Query(ctx, `SELECT `+ofertaCols+` FROM atendimento_ofertas WHERE ativa ORDER BY nome, id`)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanOferta)
}

// ListSolicitacoes devolve a página do filtro e o total.
func (r *Repository) ListSolicitacoes(ctx context.Context, db database.DBTX, f domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	args := []any{f.Solicitante, f.Restrito, idsOrEmpty(f.Unidades), f.Tipo, f.Estado, f.Finalizadas, f.Responsavel, f.SemDono, padraoLike(f.Texto)}
	var total int64
	if err := db.QueryRow(ctx, `SELECT count(*) FROM atendimento_solicitacoes s`+solicitacaoWhere, args...).Scan(&total); err != nil {
		return nil, 0, mapDB(err)
	}
	rows, err := db.Query(ctx, solicitacaoSelect+solicitacaoWhere+`
		ORDER BY CASE WHEN $1::uuid IS NULL THEN s.created_at END ASC, s.created_at DESC, s.id
		LIMIT $10 OFFSET $11`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, mapDB(err)
	}
	lista, err := coletar(rows, scanSolicitacao)
	return lista, total, err
}

// GetSolicitacao busca a solicitação com os nomes de leitura.
func (r *Repository) GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Solicitacao, error) {
	s, err := scanSolicitacao(db.QueryRow(ctx, solicitacaoSelect+` WHERE s.id = $1`, id))
	if err != nil {
		return domain.Solicitacao{}, mapDB(err)
	}
	return s, nil
}

// TravarSolicitacao bloqueia a linha na transação do chamador.
func (r *Repository) TravarSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	var achou uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM atendimento_solicitacoes WHERE id = $1 FOR UPDATE`, id).Scan(&achou)
	return mapDB(err)
}

// InsertSolicitacao grava a solicitação e a devolve com número e nomes.
func (r *Repository) InsertSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) (domain.Solicitacao, error) {
	_, err := db.Exec(ctx, `INSERT INTO atendimento_solicitacoes
		(id, tipo, estado, titulo, descricao, oferta_id, produto_id, unidade_id, solicitante_id, responsavel_id,
		 impacto, urgencia, prioridade, meta_resposta_min, meta_solucao_min, respostas, created_at, updated_at, etapa_atual)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		s.ID, s.Tipo, s.Estado, s.Titulo, s.Descricao, s.OfertaID, s.ProdutoID, s.UnidadeID, s.SolicitanteID,
		s.ResponsavelID, s.Impacto, s.Urgencia, s.Prioridade, s.MetaRespostaMin, s.MetaSolucaoMin, respostasJSON(s.Respostas), s.CreatedAt, s.UpdatedAt,
		s.EtapaAtual)
	if err != nil {
		return domain.Solicitacao{}, mapDB(err)
	}
	return r.GetSolicitacao(ctx, db, s.ID)
}

// UpdateSolicitacao grava estado, responsável, os marcos de conclusão e a
// classificação. Título, descrição e oferta não mudam depois de abertos.
func (r *Repository) UpdateSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) error {
	tag, err := db.Exec(ctx, `UPDATE atendimento_solicitacoes
		SET estado = $2, responsavel_id = $3, resolvida_em = $4, encerrada_em = $5,
			impacto = $6, urgencia = $7, prioridade = $8, meta_resposta_min = $9, meta_solucao_min = $10, etapa_atual = $11
		WHERE id = $1`, s.ID, s.Estado, s.ResponsavelID, s.ResolvidaEm, s.EncerradaEm,
		s.Impacto, s.Urgencia, s.Prioridade, s.MetaRespostaMin, s.MetaSolucaoMin, s.EtapaAtual)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

// SetSegredo grava o id do cofre, ou limpa quando segredo é nulo.
func (r *Repository) SetSegredo(ctx context.Context, db database.DBTX, id uuid.UUID, segredo *uuid.UUID, agora time.Time) error {
	tag, err := db.Exec(ctx, `UPDATE atendimento_solicitacoes SET cofre_segredo_id = $2, updated_at = $3 WHERE id = $1`, id, segredo, agora)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

// InsertPassagem registra a mudança de estado.
func (r *Repository) InsertPassagem(ctx context.Context, db database.DBTX, p domain.Passagem) (domain.Passagem, error) {
	out, err := scanPassagem(db.QueryRow(ctx, `INSERT INTO atendimento_passagens (solicitacao_id, de, para, autor_id, em)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, solicitacao_id, de, para, autor_id,
			COALESCE((SELECT `+nomeDe("u")+` FROM users u WHERE u.id = autor_id), ''), em`,
		p.SolicitacaoID, p.De, p.Para, p.AutorID, p.Em))
	if err != nil {
		return domain.Passagem{}, mapDB(err)
	}
	return out, nil
}

// ListPassagens devolve o histórico em ordem.
func (r *Repository) ListPassagens(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]domain.Passagem, error) {
	rows, err := db.Query(ctx, `SELECT p.id, p.solicitacao_id, p.de, p.para, p.autor_id, COALESCE(`+nomeDe("u")+`, ''), p.em
		FROM atendimento_passagens p LEFT JOIN users u ON u.id = p.autor_id
		WHERE p.solicitacao_id = $1 ORDER BY p.em, p.id`, solicitacao)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanPassagem)
}

// InsertComentario grava a resposta ou a nota.
func (r *Repository) InsertComentario(ctx context.Context, db database.DBTX, c domain.Comentario) (domain.Comentario, error) {
	out, err := scanComentario(db.QueryRow(ctx, `INSERT INTO atendimento_comentarios (id, solicitacao_id, autor_id, texto, interno, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, solicitacao_id, autor_id,
			COALESCE((SELECT `+nomeDe("u")+` FROM users u WHERE u.id = autor_id), ''), texto, interno, created_at`,
		c.ID, c.SolicitacaoID, c.AutorID, c.Texto, c.Interno, c.CreatedAt))
	if err != nil {
		return domain.Comentario{}, mapDB(err)
	}
	return out, nil
}

// ListComentarios devolve a conversa; sem internos, só as respostas.
func (r *Repository) ListComentarios(ctx context.Context, db database.DBTX, solicitacao uuid.UUID, internos bool) ([]domain.Comentario, error) {
	rows, err := db.Query(ctx, `SELECT c.id, c.solicitacao_id, c.autor_id, COALESCE(`+nomeDe("u")+`, ''), c.texto, c.interno, c.created_at
		FROM atendimento_comentarios c LEFT JOIN users u ON u.id = c.autor_id
		WHERE c.solicitacao_id = $1 AND ($2 OR NOT c.interno)
		ORDER BY c.created_at, c.id`, solicitacao, internos)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanComentario)
}

const membrosDaUnidade = `FROM users u
	JOIN atendimento_equipe_membros m ON m.user_id = u.id
	JOIN atendimento_equipes e ON e.id = m.equipe_id
	WHERE u.active AND e.unidade_id IS NOT DISTINCT FROM $1`

// Responsaveis lista quem pode assumir solicitações da unidade.
func (r *Repository) Responsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Candidato, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT u.id, `+nomeDe("u")+` `+membrosDaUnidade+` ORDER BY 2, 1`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, scanCandidato)
}

// ResponsavelValido confere se user é membro ativo de equipe da unidade.
func (r *Repository) ResponsavelValido(ctx context.Context, db database.DBTX, user uuid.UUID, unidade *uuid.UUID) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 `+membrosDaUnidade+` AND u.id = $2)`, unidade, user).Scan(&ok)
	return ok, mapDB(err)
}

// IncidentesRecorrentes agrupa os incidentes recentes por oferta.
func (r *Repository) IncidentesRecorrentes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, minimo int) ([]domain.Recorrencia, error) {
	rows, err := db.Query(ctx, `SELECT s.oferta_id, o.nome, s.unidade_id, count(*), max(s.created_at),
			array_agg(s.id ORDER BY s.created_at)
		FROM atendimento_solicitacoes s JOIN atendimento_ofertas o ON o.id = s.oferta_id
		WHERE s.tipo = 'incidente' AND s.estado <> 'cancelado' AND s.created_at >= $3
			AND ($1 = false OR s.unidade_id = ANY($2::uuid[]))
		GROUP BY s.oferta_id, o.nome, s.unidade_id
		HAVING count(*) >= $4
		ORDER BY count(*) DESC, max(s.created_at) DESC
		LIMIT 50`, restrito, idsOrEmpty(unidades), desde, minimo)
	if err != nil {
		return nil, mapDB(err)
	}
	return coletar(rows, func(row interface{ Scan(...any) error }) (domain.Recorrencia, error) {
		var g domain.Recorrencia
		err := row.Scan(&g.OfertaID, &g.OfertaNome, &g.UnidadeID, &g.Total, &g.UltimoEm, &g.Solicitacoes)
		return g, err
	})
}

func respostasJSON(m map[string]string) []byte {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// SolicitacaoPorNumero devolve a solicitação do número público.
func (r *Repository) SolicitacaoPorNumero(ctx context.Context, db database.DBTX, numero int64) (domain.Solicitacao, error) {
	s, err := scanSolicitacao(db.QueryRow(ctx, solicitacaoSelect+` WHERE s.numero = $1`, numero))
	if err != nil {
		return domain.Solicitacao{}, mapDB(err)
	}
	return s, nil
}

// EmailDe devolve o e-mail da conta, ou vazio se não houver.
func (r *Repository) EmailDe(ctx context.Context, db database.DBTX, user uuid.UUID) (string, error) {
	var email string
	err := db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, user).Scan(&email)
	if database.IsNoRows(err) {
		return "", nil
	}
	return email, mapDB(err)
}

// ReservarCorreio grava a marca da mensagem. false = ela já entrou.
func (r *Repository) ReservarCorreio(ctx context.Context, db database.DBTX, messageID string, solicitacao uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `INSERT INTO correio_recebidos (message_id, solicitacao_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, messageID, solicitacao)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

// LiberarCorreio desfaz a marca quando a gravação do comentário falha.
func (r *Repository) LiberarCorreio(ctx context.Context, db database.DBTX, messageID string) error {
	_, err := db.Exec(ctx, `DELETE FROM correio_recebidos WHERE message_id = $1`, messageID)
	return mapDB(err)
}
