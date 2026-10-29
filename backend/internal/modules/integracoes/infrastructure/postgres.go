// Package infrastructure grava o módulo de integrações no Postgres.
package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case database.IsUniqueViolation(err):
		return domain.ErrTarefaAtiva
	case database.IsForeignKeyViolation(err):
		return domain.Recusa{Msg: "a unidade escolhida não existe"}
	case database.IsCheckViolation(err):
		return domain.Recusa{Msg: "dados fora do cadastro"}
	default:
		return fmt.Errorf("integracoes: %w", err)
	}
}

const colunasConexao = `id, tipo, nome, modo, url, perfil, fuso, credenciais <> '', token_hash IS NOT NULL, token_em, extraido_em,
	created_at, updated_at, created_by, credenciais, sincronizar`

func scanConexao(row pgx.Row) (domain.Conexao, error) {
	var c domain.Conexao
	err := row.Scan(&c.ID, &c.Tipo, &c.Nome, &c.Modo, &c.URL, &c.Perfil, &c.Fuso, &c.TemCredenciais, &c.TemToken, &c.TokenEm, &c.ExtraidoEm,
		&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.Cifradas, &c.Sincronizar)
	return c, mapDB(err)
}

func (r *Repository) InsertConexao(ctx context.Context, db database.DBTX, c domain.Conexao) (domain.Conexao, error) {
	return scanConexao(db.QueryRow(ctx, `INSERT INTO integracoes_conexoes (id, tipo, nome, modo, url, perfil, fuso, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9) RETURNING `+colunasConexao,
		c.ID, c.Tipo, c.Nome, c.Modo, c.URL, c.Perfil, c.Fuso, c.CreatedAt, c.CreatedBy))
}

func (r *Repository) UpdateConexao(ctx context.Context, db database.DBTX, c domain.Conexao) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_conexoes SET nome = $2, url = $3, perfil = $4, fuso = $5, updated_at = now() WHERE id = $1`,
		c.ID, c.Nome, c.URL, c.Perfil, c.Fuso)
	return afetou(tag.RowsAffected(), err)
}

func afetou(n int64, err error) error {
	if err == nil && n == 0 {
		return domain.ErrNaoEncontrado
	}
	return mapDB(err)
}

func (r *Repository) GetConexao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Conexao, error) {
	return scanConexao(db.QueryRow(ctx, `SELECT `+colunasConexao+` FROM integracoes_conexoes WHERE id = $1`, id))
}

func (r *Repository) ListConexoes(ctx context.Context, db database.DBTX) ([]domain.Conexao, error) {
	rows, err := db.Query(ctx, `SELECT `+colunasConexao+` FROM integracoes_conexoes ORDER BY created_at`)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Conexao{}
	for rows.Next() {
		c, err := scanConexao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) DeleteConexao(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM integracoes_conexoes WHERE id = $1`, id)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) SetCredenciais(ctx context.Context, db database.DBTX, id uuid.UUID, cifradas string) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_conexoes SET credenciais = $2, updated_at = now() WHERE id = $1`, id, cifradas)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) SetToken(ctx context.Context, db database.DBTX, id uuid.UUID, hash *string) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_conexoes SET token_hash = $2, token_em = CASE WHEN $2::text IS NULL THEN NULL ELSE now() END,
		updated_at = now() WHERE id = $1`, id, hash)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) PorToken(ctx context.Context, db database.DBTX, hash string) (domain.Conexao, error) {
	return scanConexao(db.QueryRow(ctx, `SELECT `+colunasConexao+` FROM integracoes_conexoes WHERE token_hash = $1`, hash))
}

func (r *Repository) MarcarExtracao(ctx context.Context, db database.DBTX, id uuid.UUID, quando time.Time) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_conexoes SET extraido_em = $2 WHERE id = $1`, id, quando)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) SetSincronizar(ctx context.Context, db database.DBTX, id uuid.UUID, ligar bool) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_conexoes SET sincronizar = $2, updated_at = now() WHERE id = $1`, id, ligar)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) Sincronizando(ctx context.Context, db database.DBTX) (bool, error) {
	var sim bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM integracoes_conexoes WHERE sincronizar)`).Scan(&sim)
	return sim, mapDB(err)
}

// ParaSincronizar olha a extração pedida, não a concluída: com o conector
// fora do ar, a sincronização espera o intervalo em vez de repetir.
func (r *Repository) ParaSincronizar(ctx context.Context, db database.DBTX, antes time.Time) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `SELECT c.id FROM integracoes_conexoes c WHERE c.sincronizar
		AND NOT EXISTS (SELECT 1 FROM integracoes_tarefas t WHERE t.conexao_id = c.id AND t.estado IN ('pendente', 'em_andamento'))
		AND NOT EXISTS (SELECT 1 FROM integracoes_tarefas t WHERE t.conexao_id = c.id AND t.tipo = 'extracao' AND t.criada_em > $1)
		ORDER BY c.created_at`, antes)
	if err != nil {
		return nil, mapDB(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	return ids, mapDB(err)
}

func (r *Repository) GravarItens(ctx context.Context, db database.DBTX, conexao uuid.UUID, itens []domain.Item) error {
	for _, it := range itens {
		if _, err := db.Exec(ctx, `INSERT INTO integracoes_itens
			(conexao_id, tipo, id_origem, versao, nome, entidade, categoria, sla, minutos, classe, data, dados, atualizado_em)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
			ON CONFLICT (conexao_id, tipo, id_origem) DO UPDATE SET versao = EXCLUDED.versao, nome = EXCLUDED.nome,
			entidade = EXCLUDED.entidade, categoria = EXCLUDED.categoria, sla = EXCLUDED.sla, minutos = EXCLUDED.minutos,
			classe = EXCLUDED.classe, data = EXCLUDED.data, dados = EXCLUDED.dados, atualizado_em = now()`,
			conexao, it.Tipo, it.ID, it.Versao, it.Nome, it.Entidade, it.Categoria, it.SLA, it.Minutos, it.Classe, it.Data, []byte(it.Dados)); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

func (r *Repository) Versoes(ctx context.Context, db database.DBTX, conexao uuid.UUID) (map[int]string, error) {
	rows, err := db.Query(ctx, `SELECT id_origem, versao FROM integracoes_itens WHERE conexao_id = $1 AND tipo = 'chamado'`, conexao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var id int
		var v string
		if err := rows.Scan(&id, &v); err != nil {
			return nil, mapDB(err)
		}
		out[id] = v
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) IDs(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string) (map[int]bool, error) {
	rows, err := db.Query(ctx, `SELECT id_origem FROM integracoes_itens WHERE conexao_id = $1 AND tipo = $2`, conexao, tipo)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, mapDB(err)
		}
		out[id] = true
	}
	return out, mapDB(rows.Err())
}

const colunasItem = `tipo, id_origem, versao, nome, entidade, categoria, sla, minutos, classe, data, dados`

func scanItem(row pgx.Row) (domain.Item, error) {
	var it domain.Item
	var dados []byte
	err := row.Scan(&it.Tipo, &it.ID, &it.Versao, &it.Nome, &it.Entidade, &it.Categoria, &it.SLA, &it.Minutos, &it.Classe, &it.Data, &dados)
	it.Dados = dados
	return it, err
}

func (r *Repository) Itens(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string, de, limite int) ([]domain.Item, int, error) {
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM integracoes_itens WHERE conexao_id = $1 AND tipo = $2`, conexao, tipo).Scan(&total); err != nil {
		return nil, 0, mapDB(err)
	}
	rows, err := db.Query(ctx, `SELECT `+colunasItem+` FROM integracoes_itens WHERE conexao_id = $1 AND tipo = $2
		ORDER BY id_origem OFFSET $3 LIMIT $4`, conexao, tipo, de, limite)
	if err != nil {
		return nil, 0, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, 0, mapDB(err)
		}
		out = append(out, it)
	}
	return out, total, mapDB(rows.Err())
}

func (r *Repository) Item(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string, id int) (domain.Item, bool, error) {
	it, err := scanItem(db.QueryRow(ctx, `SELECT `+colunasItem+` FROM integracoes_itens WHERE conexao_id = $1 AND tipo = $2 AND id_origem = $3`,
		conexao, tipo, id))
	if database.IsNoRows(err) {
		return domain.Item{}, false, nil
	}
	return it, err == nil, mapDB(err)
}

func (r *Repository) Contagens(ctx context.Context, db database.DBTX, conexao uuid.UUID) (map[string]int, error) {
	rows, err := db.Query(ctx, `SELECT tipo, count(*) FROM integracoes_itens WHERE conexao_id = $1 GROUP BY tipo`, conexao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var tipo string
		var n int
		if err := rows.Scan(&tipo, &n); err != nil {
			return nil, mapDB(err)
		}
		out[tipo] = n
	}
	return out, mapDB(rows.Err())
}

// Grupos agrupa os chamados por entidade e categoria sem abrir o JSON.
func (r *Repository) Grupos(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]domain.Grupo, error) {
	rows, err := db.Query(ctx, `SELECT coalesce(entidade, 0), coalesce(categoria, 0), count(*), count(sla),
			mode() WITHIN GROUP (ORDER BY sla), coalesce(mode() WITHIN GROUP (ORDER BY nullif(classe, '')), ''),
			coalesce(min(left(nullif(data, ''), 10)), ''), coalesce(max(left(data, 10)), '')
		FROM integracoes_itens WHERE conexao_id = $1 AND tipo = 'chamado' GROUP BY 1, 2`, conexao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Grupo{}
	for rows.Next() {
		var g domain.Grupo
		if err := rows.Scan(&g.Entidade, &g.Categoria, &g.Chamados, &g.ComPrazo, &g.SLA, &g.Classe, &g.Primeiro, &g.Ultimo); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, g)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) Nomes(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string) (map[int]string, error) {
	rows, err := db.Query(ctx, `SELECT id_origem, nome FROM integracoes_itens WHERE conexao_id = $1 AND tipo = $2`, conexao, tipo)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var id int
		var nome string
		if err := rows.Scan(&id, &nome); err != nil {
			return nil, mapDB(err)
		}
		out[id] = nome
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) SLAs(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]domain.NivelSLA, error) {
	rows, err := db.Query(ctx, `SELECT id_origem, nome, coalesce(minutos, 0) FROM integracoes_itens
		WHERE conexao_id = $1 AND tipo = 'sla' ORDER BY id_origem`, conexao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.NivelSLA{}
	for rows.Next() {
		var n domain.NivelSLA
		if err := rows.Scan(&n.ID, &n.Nome, &n.Minutos); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, n)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ListarTabela(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]domain.Linha, error) {
	rows, err := db.Query(ctx, `SELECT d.secretaria, d.categoria, d.categoria_nome, d.chamados, d.com_prazo, d.nivel, d.unidade_id,
			coalesce(u.nome, ''), d.oferta, d.catalogo, d.tipo, d.prioridade
		FROM integracoes_depara d LEFT JOIN unidades u ON u.id = d.unidade_id
		WHERE d.conexao_id = $1 ORDER BY d.secretaria, d.categoria = -1, d.chamados DESC, d.categoria`, conexao)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Linha{}
	for rows.Next() {
		var l domain.Linha
		if err := rows.Scan(&l.Secretaria, &l.Categoria, &l.CategoriaNome, &l.Chamados, &l.ComPrazo, &l.Nivel, &l.UnidadeID,
			&l.UnidadeNome, &l.Oferta, &l.Catalogo, &l.Tipo, &l.Prioridade); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, l)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) TrocarTabela(ctx context.Context, db database.DBTX, conexao uuid.UUID, linhas []domain.Linha) error {
	if _, err := db.Exec(ctx, `DELETE FROM integracoes_depara WHERE conexao_id = $1`, conexao); err != nil {
		return mapDB(err)
	}
	for _, l := range linhas {
		if _, err := db.Exec(ctx, `INSERT INTO integracoes_depara
			(conexao_id, secretaria, categoria, categoria_nome, chamados, com_prazo, nivel, unidade_id, oferta, catalogo, tipo, prioridade)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			conexao, l.Secretaria, l.Categoria, l.CategoriaNome, l.Chamados, l.ComPrazo, l.Nivel, l.UnidadeID, l.Oferta, l.Catalogo,
			l.Tipo, l.Prioridade); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

const colunasTarefa = `id, conexao_id, tipo, estado, feitos, total, relatorio, erro, pedida_por, criada_em, iniciada_em, batida_em, terminada_em`

func scanTarefa(row pgx.Row) (domain.Tarefa, error) {
	var t domain.Tarefa
	var rel []byte
	err := row.Scan(&t.ID, &t.ConexaoID, &t.Tipo, &t.Estado, &t.Feitos, &t.Total, &rel, &t.Erro, &t.PedidaPor,
		&t.CriadaEm, &t.IniciadaEm, &t.BatidaEm, &t.TerminadaEm)
	t.Relatorio = rel
	return t, err
}

func (r *Repository) InserirTarefa(ctx context.Context, db database.DBTX, t domain.Tarefa) error {
	_, err := db.Exec(ctx, `INSERT INTO integracoes_tarefas (id, conexao_id, tipo, estado, pedida_por, criada_em) VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.ConexaoID, t.Tipo, t.Estado, t.PedidaPor, t.CriadaEm)
	return mapDB(err)
}

func (r *Repository) GetTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Tarefa, error) {
	t, err := scanTarefa(db.QueryRow(ctx, `SELECT `+colunasTarefa+` FROM integracoes_tarefas WHERE id = $1`, id))
	return t, mapDB(err)
}

func (r *Repository) ListarTarefas(ctx context.Context, db database.DBTX, conexao uuid.UUID, limite int) ([]domain.Tarefa, error) {
	rows, err := db.Query(ctx, `SELECT `+colunasTarefa+` FROM integracoes_tarefas WHERE conexao_id = $1 ORDER BY criada_em DESC LIMIT $2`,
		conexao, limite)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Tarefa{}
	for rows.Next() {
		t, err := scanTarefa(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, t)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) Pegar(ctx context.Context, db database.DBTX, conexao *uuid.UUID, tipos []string, modo string) (domain.Tarefa, bool, error) {
	t, err := scanTarefa(db.QueryRow(ctx, `UPDATE integracoes_tarefas SET estado = 'em_andamento',
			iniciada_em = coalesce(iniciada_em, now()), batida_em = now()
		WHERE id = (SELECT t.id FROM integracoes_tarefas t JOIN integracoes_conexoes c ON c.id = t.conexao_id
			WHERE t.estado = 'pendente' AND t.tipo = ANY($2) AND ($1::uuid IS NULL OR t.conexao_id = $1) AND ($3 = '' OR c.modo = $3)
			ORDER BY t.criada_em LIMIT 1 FOR UPDATE OF t SKIP LOCKED)
		RETURNING `+colunasTarefa, conexao, tipos, modo))
	if database.IsNoRows(err) {
		return domain.Tarefa{}, false, nil
	}
	return t, err == nil, mapDB(err)
}

func (r *Repository) Progresso(ctx context.Context, db database.DBTX, id uuid.UUID, feitos, total int) (string, error) {
	var estado string
	err := db.QueryRow(ctx, `UPDATE integracoes_tarefas SET feitos = $2, total = $3, batida_em = now() WHERE id = $1 RETURNING estado`,
		id, feitos, total).Scan(&estado)
	return estado, mapDB(err)
}

func (r *Repository) Terminar(ctx context.Context, db database.DBTX, id uuid.UUID, estado string, relatorio []byte, erro string) error {
	_, err := db.Exec(ctx, `UPDATE integracoes_tarefas SET estado = $2, relatorio = $3, erro = $4, terminada_em = now(), batida_em = now()
		WHERE id = $1 AND estado IN ('pendente', 'em_andamento')`, id, estado, relatorio, erro)
	return mapDB(err)
}

func (r *Repository) Cancelar(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `UPDATE integracoes_tarefas SET estado = 'cancelada', terminada_em = now()
		WHERE id = $1 AND estado IN ('pendente', 'em_andamento')`, id)
	return afetou(tag.RowsAffected(), err)
}

func (r *Repository) Retomar(ctx context.Context, db database.DBTX, antes time.Time) (int, error) {
	tag, err := db.Exec(ctx, `UPDATE integracoes_tarefas SET estado = 'pendente' WHERE estado = 'em_andamento' AND batida_em < $1`, antes)
	return int(tag.RowsAffected()), mapDB(err)
}
