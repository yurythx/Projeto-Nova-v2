package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

var _ domain.Repository = (*Repository)(nil)

const colunasProjeto = `id, numero, titulo, descricao, objetivo, estado, prioridade,
	data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real,
	orcamento, unidade_id, produto_id, gerente_id, gerente_nome, created_at, updated_at`

const colunasTarefa = `id, projeto_id, titulo, descricao, responsavel_id, responsavel_nome, estado, ordem,
	data_limite, created_at, updated_at, origem, predecessora_id, relato, pai_id`

func scanProjeto(row interface{ Scan(...any) error }) (domain.Projeto, error) {
	var p domain.Projeto
	err := row.Scan(
		&p.ID, &p.Numero, &p.Titulo, &p.Descricao, &p.Objetivo, &p.Estado, &p.Prioridade,
		&p.DataInicioPlanejada, &p.DataFimPlanejada, &p.DataInicioReal, &p.DataFimReal,
		&p.Orcamento, &p.UnidadeID, &p.ProdutoID, &p.GerenteID, &p.GerenteNome,
		&p.CreatedAt, &p.UpdatedAt,
	)
	p.Proximos = domain.ProximosEstados(p.Estado)
	p.Tarefas = []domain.Tarefa{}
	return p, err
}

func scanTarefa(row interface{ Scan(...any) error }) (domain.Tarefa, error) {
	var t domain.Tarefa
	err := row.Scan(
		&t.ID, &t.ProjetoID, &t.Titulo, &t.Descricao, &t.ResponsavelID, &t.ResponsavelNome, &t.Estado, &t.Ordem,
		&t.DataLimite, &t.CreatedAt, &t.UpdatedAt, &t.Origem, &t.PredecessoraID, &t.Relato, &t.PaiID,
	)
	return t, err
}

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrProjetoNaoEncontrado
	case database.IsForeignKeyViolation(err):
		return domain.ErrTarefaNaoEncontrada
	default:
		return fmt.Errorf("projetos: %w", err)
	}
}

func (r *Repository) Criar(ctx context.Context, db database.DBTX, p *domain.Projeto) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Estado == "" {
		p.Estado = domain.EstadoPlanejado
	}
	if p.Prioridade == "" {
		p.Prioridade = domain.PrioridadeMedia
	}

	const q = `INSERT INTO projetos (
		id, titulo, descricao, objetivo, estado, prioridade,
		data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real,
		orcamento, unidade_id, produto_id, gerente_id, gerente_nome, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	RETURNING numero, created_at, updated_at`

	err := db.QueryRow(ctx, q,
		p.ID, p.Titulo, p.Descricao, p.Objetivo, p.Estado, p.Prioridade,
		p.DataInicioPlanejada, p.DataFimPlanejada, p.DataInicioReal, p.DataFimReal,
		p.Orcamento, p.UnidadeID, p.ProdutoID, p.GerenteID, p.GerenteNome, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.Numero, &p.CreatedAt, &p.UpdatedAt)

	p.Proximos = domain.ProximosEstados(p.Estado)
	return mapDB(err)
}

func (r *Repository) Obter(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Projeto, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos WHERE id = $1`, colunasProjeto)
	p, err := scanProjeto(db.QueryRow(ctx, q, id))
	if err != nil {
		return nil, mapDB(err)
	}

	// Carrega tarefas associadas
	tarefas, err := r.ListarTarefas(ctx, db, id)
	if err == nil {
		p.Tarefas = tarefas
	}
	vinculos, err := r.ListarVinculos(ctx, db, id)
	if err == nil {
		p.Vinculos = vinculos
	}
	return &p, nil
}

func (r *Repository) Listar(ctx context.Context, db database.DBTX, f domain.FiltroProjetos) ([]domain.Projeto, error) {
	args := []any{}
	n := 1

	q := fmt.Sprintf(`SELECT %s,
		(SELECT count(*) FROM projetos_tarefas t WHERE t.projeto_id = projetos.id AND t.pai_id IS NULL),
		(SELECT count(*) FILTER (WHERE t.estado = 'concluida') FROM projetos_tarefas t WHERE t.projeto_id = projetos.id AND t.pai_id IS NULL)
		FROM projetos WHERE 1=1`, colunasProjeto)

	if f.Estado != "" {
		q += fmt.Sprintf(` AND estado = $%d`, n)
		args = append(args, f.Estado)
		n++
	}

	if f.Busca != "" {
		q += fmt.Sprintf(` AND (titulo ILIKE $%d OR descricao ILIKE $%d OR objetivo ILIKE $%d)`, n, n, n)
		args = append(args, "%"+f.Busca+"%")
		n++
	}

	if f.ProdutoID != nil {
		q += fmt.Sprintf(` AND produto_id = $%d`, n)
		args = append(args, *f.ProdutoID)
		n++
	}

	if f.AlvoID != nil {
		if f.AlvoTipo != "" {
			q += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM projetos_vinculos v WHERE v.projeto_id = projetos.id AND v.tipo = $%d AND v.alvo_id = $%d)`, n, n+1)
			args = append(args, f.AlvoTipo, *f.AlvoID)
			n += 2
		} else {
			q += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM projetos_vinculos v WHERE v.projeto_id = projetos.id AND v.alvo_id = $%d)`, n)
			args = append(args, *f.AlvoID)
			n++
		}
	}

	if f.Restrito {
		q += fmt.Sprintf(` AND (unidade_id = ANY($%d) OR unidade_id IS NULL)`, n)
		args = append(args, f.Unidades)
	}

	q += ` ORDER BY created_at DESC`

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Projeto
	for rows.Next() {
		var item domain.Projeto
		err := rows.Scan(
			&item.ID, &item.Numero, &item.Titulo, &item.Descricao, &item.Objetivo, &item.Estado, &item.Prioridade,
			&item.DataInicioPlanejada, &item.DataFimPlanejada, &item.DataInicioReal, &item.DataFimReal,
			&item.Orcamento, &item.UnidadeID, &item.ProdutoID, &item.GerenteID, &item.GerenteNome,
			&item.CreatedAt, &item.UpdatedAt, &item.TarefasTotal, &item.TarefasConcluidas,
		)
		if err != nil {
			return nil, mapDB(err)
		}
		item.Proximos = domain.ProximosEstados(item.Estado)
		item.Tarefas = []domain.Tarefa{}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	if lista == nil {
		lista = []domain.Projeto{}
	}
	return lista, nil
}

func (r *Repository) Atualizar(ctx context.Context, db database.DBTX, id uuid.UUID, e domain.EntradaProjeto) (*domain.Projeto, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`UPDATE projetos SET
		titulo = $2, descricao = $3, objetivo = $4, prioridade = $5,
		data_inicio_planejada = $6, data_fim_planejada = $7,
		data_inicio_real = $8, data_fim_real = $9, orcamento = $10,
		unidade_id = $11, produto_id = $12, gerente_id = $13, gerente_nome = $14,
		updated_at = $15
		WHERE id = $1
		RETURNING %s`, colunasProjeto)

	p, err := scanProjeto(db.QueryRow(ctx, q,
		id, e.Titulo, e.Descricao, e.Objetivo, e.Prioridade,
		e.DataInicioPlanejada, e.DataFimPlanejada,
		e.DataInicioReal, e.DataFimReal, e.Orcamento,
		e.UnidadeID, e.ProdutoID, e.GerenteID, e.GerenteNome,
		now,
	))
	if err != nil {
		return nil, mapDB(err)
	}
	return &p, nil
}

func (r *Repository) PosicaoUnidade(ctx context.Context, db database.DBTX, unidade uuid.UUID) (*uuid.UUID, []uuid.UUID, error) {
	var entidade *uuid.UUID
	err := db.QueryRow(ctx, `SELECT entidade_id FROM unidades WHERE id = $1`, unidade).Scan(&entidade)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, nil, nil
		}
		return nil, nil, mapDB(err)
	}
	rows, err := db.Query(ctx, `
		WITH RECURSIVE acima AS (
			SELECT parent_id AS id FROM unidades WHERE id = $1
			UNION ALL
			SELECT u.parent_id FROM unidades u
			JOIN acima a ON u.id = a.id
			WHERE u.parent_id IS NOT NULL
		)
		SELECT id FROM acima WHERE id IS NOT NULL`, unidade)
	if err != nil {
		return nil, nil, mapDB(err)
	}
	defer rows.Close()
	var acima []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, nil, mapDB(err)
		}
		acima = append(acima, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, mapDB(err)
	}
	return entidade, acima, nil
}

// Aprovadores são quem pode reabrir: administrador, gestão global de
// projetos, gestão da entidade da unidade ou de uma unidade acima.
func (r *Repository) Aprovadores(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error) {
	const q = `SELECT DISTINCT u.id
		FROM users u
		WHERE u.active AND (
			'nova-admin' = ANY(u.roles)
			OR EXISTS (
				SELECT 1 FROM user_scopes s
				JOIN perfis pf ON pf.id = s.perfil_id AND pf.ativo
				WHERE s.user_id = u.id
					AND pf.permissoes && ARRAY['*', 'projetos:*', 'projetos:manage']::text[]
					AND (
						(s.entidade_id IS NULL AND s.unidade_id IS NULL AND s.departamento_id IS NULL)
						OR (
							$1::uuid IS NOT NULL AND (
								s.unidade_id IN (
									WITH RECURSIVE acima AS (
										SELECT parent_id AS id FROM unidades WHERE id = $1
										UNION ALL
										SELECT un.parent_id FROM unidades un
										JOIN acima a ON un.id = a.id
										WHERE un.parent_id IS NOT NULL
									)
									SELECT id FROM acima WHERE id IS NOT NULL
								)
								OR (
									s.unidade_id IS NULL AND s.departamento_id IS NULL
									AND s.entidade_id IS NOT NULL
									AND s.entidade_id = (SELECT entidade_id FROM unidades WHERE id = $1)
								)
							)
						)
					)
			)
		)`
	rows, err := db.Query(ctx, q, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapDB(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	return ids, nil
}

func (r *Repository) SalvarReabertura(ctx context.Context, db database.DBTX, reab *domain.Reabertura) error {
	reab.CreatedAt = time.Now().UTC()
	const q = `INSERT INTO projetos_reaberturas
		(id, projeto_id, motivo, aprovador_id, aprovador_nome, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(ctx, q, reab.ID, reab.ProjetoID, reab.Motivo, reab.AprovadorID, reab.AprovadorNome, reab.CreatedAt)
	return mapDB(err)
}

func (r *Repository) UltimaReabertura(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Reabertura, bool, error) {
	const q = `SELECT id, projeto_id, motivo, aprovador_id, aprovador_nome, created_at
		FROM projetos_reaberturas WHERE projeto_id = $1
		ORDER BY created_at DESC LIMIT 1`
	var reab domain.Reabertura
	err := db.QueryRow(ctx, q, projetoID).Scan(
		&reab.ID, &reab.ProjetoID, &reab.Motivo, &reab.AprovadorID, &reab.AprovadorNome, &reab.CreatedAt,
	)
	if err != nil {
		if database.IsNoRows(err) {
			return domain.Reabertura{}, false, nil
		}
		return domain.Reabertura{}, false, mapDB(err)
	}
	return reab, true, nil
}

func (r *Repository) Transicionar(ctx context.Context, db database.DBTX, id uuid.UUID, novoEstado string) (*domain.Projeto, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`UPDATE projetos SET estado = $2, updated_at = $3 WHERE id = $1 RETURNING %s`, colunasProjeto)
	p, err := scanProjeto(db.QueryRow(ctx, q, id, novoEstado, now))
	if err != nil {
		return nil, mapDB(err)
	}
	return &p, nil
}

func (r *Repository) EstadoDaTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (string, error) {
	var estado string
	err := db.QueryRow(ctx, `SELECT estado FROM projetos_tarefas WHERE id = $1`, id).Scan(&estado)
	if err != nil {
		return "", mapDB(err)
	}
	return estado, nil
}

func (r *Repository) CriarTarefa(ctx context.Context, db database.DBTX, t *domain.Tarefa) error {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Estado == "" {
		t.Estado = domain.EstadoTarefaAFazer
	}

	const q = `INSERT INTO projetos_tarefas (
		id, projeto_id, titulo, descricao, responsavel_id, responsavel_nome, estado, ordem, data_limite, created_at, updated_at, origem, predecessora_id, pai_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	RETURNING created_at, updated_at`

	err := db.QueryRow(ctx, q,
		t.ID, t.ProjetoID, t.Titulo, t.Descricao, t.ResponsavelID, t.ResponsavelNome, t.Estado, t.Ordem, t.DataLimite, t.CreatedAt, t.UpdatedAt, t.Origem, t.PredecessoraID, t.PaiID,
	).Scan(&t.CreatedAt, &t.UpdatedAt)

	return mapDB(err)
}

func (r *Repository) ObterTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Tarefa, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefas WHERE id = $1`, colunasTarefa)
	t, err := scanTarefa(db.QueryRow(ctx, q, id))
	if err != nil {
		if database.IsNoRows(err) {
			return nil, domain.ErrTarefaNaoEncontrada
		}
		return nil, mapDB(err)
	}
	completas, err := preencherPessoas(ctx, r, db, []domain.Tarefa{t})
	if err != nil {
		return nil, err
	}
	return &completas[0], nil
}

// NomeDaPessoa devolve o nome canônico de um usuário ativo. Com unidade,
// a lotação exibida (perfil do diretório ou escopo principal) precisa ser essa.
func (r *Repository) NomeDaPessoa(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error) {
	const q = `SELECT COALESCE(NULLIF(u.display_name, ''), u.username)
		FROM users u
		LEFT JOIN directory_profiles dp ON dp.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT s.unidade_id FROM user_scopes s
			WHERE s.user_id = u.id
			ORDER BY s.principal DESC, s.created_at
			LIMIT 1
		) ls ON true
		WHERE u.id = $1 AND u.active
		  AND ($2::uuid IS NULL OR COALESCE(dp.unidade_id, ls.unidade_id) = $2)`
	var nome string
	err := db.QueryRow(ctx, q, pessoa, unidade).Scan(&nome)
	if err != nil {
		if database.IsNoRows(err) {
			return "", domain.ErrPessoaInvalida
		}
		return "", mapDB(err)
	}
	return nome, nil
}

func (r *Repository) AtualizarTarefa(ctx context.Context, db database.DBTX, id uuid.UUID, titulo, descricao, relato string, responsavelID *uuid.UUID, responsavel, estado string, ordem int, dataLimite *time.Time, predecessoraID *uuid.UUID) (*domain.Tarefa, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`UPDATE projetos_tarefas SET
		titulo = $2, descricao = $3, relato = $4, responsavel_id = $5, responsavel_nome = $6, estado = $7, ordem = $8, data_limite = COALESCE($9, data_limite), predecessora_id = $10, updated_at = $11
		WHERE id = $1
		RETURNING %s`, colunasTarefa)

	t, err := scanTarefa(db.QueryRow(ctx, q, id, titulo, descricao, relato, responsavelID, responsavel, estado, ordem, dataLimite, predecessoraID, now))
	if err != nil {
		return nil, mapDB(err)
	}
	return &t, nil
}

func (r *Repository) ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	q := fmt.Sprintf(`SELECT %s FROM projetos_tarefas WHERE projeto_id = $1 ORDER BY ordem ASC, created_at ASC`, colunasTarefa)
	rows, err := db.Query(ctx, q, projetoID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Tarefa
	for rows.Next() {
		item, err := scanTarefa(rows)
		if err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	if lista == nil {
		lista = []domain.Tarefa{}
	}
	return preencherPessoas(ctx, r, db, lista)
}

const colunasTarefaAlias = `t.id, t.projeto_id, t.titulo, t.descricao, t.responsavel_id, t.responsavel_nome, t.estado, t.ordem,
	t.data_limite, t.created_at, t.updated_at, t.origem, t.predecessora_id, t.relato, t.pai_id`

func clausulaUnidade(restrito bool, n int) string {
	if !restrito {
		return ""
	}
	return fmt.Sprintf(` AND (p.unidade_id = ANY($%d) OR p.unidade_id IS NULL)`, n)
}

func (r *Repository) ListarMinhas(ctx context.Context, db database.DBTX, pessoa uuid.UUID, restrito bool, unidades []uuid.UUID) ([]domain.TarefaMinha, error) {
	if unidades == nil {
		unidades = []uuid.UUID{}
	}
	q := `SELECT ` + colunasTarefaAlias + `, p.numero, p.titulo, p.estado
		FROM projetos_tarefas t
		JOIN projetos p ON p.id = t.projeto_id
		WHERE (
		    t.responsavel_id = $1
		    OR EXISTS (
		      SELECT 1 FROM projetos_tarefa_pessoas tp
		      WHERE tp.tarefa_id = t.id AND tp.user_id = $1
		    )
		  )
		  AND t.estado <> 'concluida'
		  AND p.estado NOT IN ('concluido', 'cancelado')` + clausulaUnidade(restrito, 2) + `
		ORDER BY t.data_limite NULLS LAST, p.numero, t.ordem`
	args := []any{pessoa}
	if restrito {
		args = append(args, unidades)
	}
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.TarefaMinha{}
	for rows.Next() {
		var item domain.TarefaMinha
		if err := rows.Scan(
			&item.ID, &item.ProjetoID, &item.Titulo, &item.Descricao, &item.ResponsavelID, &item.ResponsavelNome, &item.Estado, &item.Ordem,
			&item.DataLimite, &item.CreatedAt, &item.UpdatedAt, &item.Origem, &item.PredecessoraID, &item.Relato, &item.PaiID,
			&item.ProjetoNumero, &item.ProjetoTitulo, &item.ProjetoEstado,
		); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	base := make([]domain.Tarefa, len(lista))
	for i := range lista {
		base[i] = lista[i].Tarefa
	}
	completas, err := preencherPessoas(ctx, r, db, base)
	if err != nil {
		return nil, err
	}
	for i := range lista {
		lista[i].Tarefa = completas[i]
	}
	return lista, nil
}

func (r *Repository) ListarCarga(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID) ([]domain.CargaPessoa, error) {
	if unidades == nil {
		unidades = []uuid.UUID{}
	}
	q := `WITH pessoas AS (
			SELECT tp.user_id AS id, tp.nome, t.id AS tarefa_id, t.estado, t.data_limite
			FROM projetos_tarefa_pessoas tp
			JOIN projetos_tarefas t ON t.id = tp.tarefa_id
			JOIN projetos p ON p.id = t.projeto_id
			WHERE p.estado NOT IN ('concluido', 'cancelado')` + clausulaUnidade(restrito, 1) + `
			UNION
			SELECT t.responsavel_id, t.responsavel_nome, t.id, t.estado, t.data_limite
			FROM projetos_tarefas t
			JOIN projetos p ON p.id = t.projeto_id
			WHERE t.responsavel_id IS NOT NULL
			  AND p.estado NOT IN ('concluido', 'cancelado')
			  AND NOT EXISTS (SELECT 1 FROM projetos_tarefa_pessoas tp WHERE tp.tarefa_id = t.id)` + clausulaUnidade(restrito, 1) + `
		)
		SELECT id, max(nome),
			count(DISTINCT tarefa_id) FILTER (WHERE estado <> 'concluida'),
			count(DISTINCT tarefa_id) FILTER (WHERE estado <> 'concluida' AND data_limite IS NOT NULL AND data_limite < now())
		FROM pessoas
		GROUP BY id
		HAVING count(DISTINCT tarefa_id) FILTER (WHERE estado <> 'concluida') > 0
		ORDER BY 3 DESC, 2`
	args := []any{}
	if restrito {
		args = append(args, unidades)
	}
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.CargaPessoa{}
	for rows.Next() {
		var item domain.CargaPessoa
		if err := rows.Scan(&item.ID, &item.Nome, &item.Abertas, &item.Atrasadas); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	return lista, mapDB(rows.Err())
}

func (r *Repository) ExcluirTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	const q = `DELETE FROM projetos_tarefas WHERE id = $1`
	_, err := db.Exec(ctx, q, id)
	return mapDB(err)
}

func (r *Repository) CriarVinculo(ctx context.Context, db database.DBTX, v *domain.Vinculo) error {
	v.CreatedAt = time.Now().UTC()
	const q = `INSERT INTO projetos_vinculos (
		id, projeto_id, tipo, alvo_id, rotulo, vinculado_por, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (projeto_id, tipo, alvo_id) DO NOTHING`

	_, err := db.Exec(ctx, q, v.ID, v.ProjetoID, v.Tipo, v.AlvoID, v.Rotulo, v.VinculadoPor, v.CreatedAt)
	return mapDB(err)
}

func (r *Repository) ListarVinculos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Vinculo, error) {
	const q = `SELECT id, projeto_id, tipo, alvo_id, rotulo, vinculado_por, created_at
		FROM projetos_vinculos WHERE projeto_id = $1 ORDER BY created_at ASC`

	rows, err := db.Query(ctx, q, projetoID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Vinculo
	for rows.Next() {
		var v domain.Vinculo
		if err := rows.Scan(&v.ID, &v.ProjetoID, &v.Tipo, &v.AlvoID, &v.Rotulo, &v.VinculadoPor, &v.CreatedAt); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, v)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	if lista == nil {
		lista = []domain.Vinculo{}
	}
	return lista, nil
}

func (r *Repository) ProjetoDoVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error) {
	var projetoID uuid.UUID
	err := db.QueryRow(ctx, `SELECT projeto_id FROM projetos_vinculos WHERE id = $1`, id).Scan(&projetoID)
	if database.IsNoRows(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, mapDB(err)
	}
	return projetoID, true, nil
}

func (r *Repository) ExcluirVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	const q = `DELETE FROM projetos_vinculos WHERE id = $1`
	_, err := db.Exec(ctx, q, id)
	return mapDB(err)
}

func (r *Repository) CriarApontamento(ctx context.Context, db database.DBTX, a *domain.Apontamento) error {
	a.CreatedAt = time.Now().UTC()
	if a.DataApontamento == "" {
		a.DataApontamento = a.CreatedAt.Format("2006-01-02")
	}
	const q = `INSERT INTO projetos_apontamentos (
		id, projeto_id, tarefa_id, usuario_id, usuario_nome, data_apontamento, horas, descricao, custo_hora, created_at
	) VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9, $10)`

	_, err := db.Exec(ctx, q, a.ID, a.ProjetoID, a.TarefaID, a.UsuarioID, a.UsuarioNome, a.DataApontamento, a.Horas, a.Descricao, a.CustoHora, a.CreatedAt)
	return mapDB(err)
}

func (r *Repository) ListarApontamentos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Apontamento, error) {
	const q = `SELECT a.id, a.projeto_id, a.tarefa_id, COALESCE(t.titulo, ''), a.usuario_id, a.usuario_nome,
		to_char(a.data_apontamento, 'YYYY-MM-DD'), a.horas, a.descricao, a.custo_hora, (a.horas * a.custo_hora), a.created_at
		FROM projetos_apontamentos a
		LEFT JOIN projetos_tarefas t ON t.id = a.tarefa_id
		WHERE a.projeto_id = $1
		ORDER BY a.data_apontamento DESC, a.created_at DESC`

	rows, err := db.Query(ctx, q, projetoID)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()

	var lista []domain.Apontamento
	for rows.Next() {
		var a domain.Apontamento
		if err := rows.Scan(
			&a.ID, &a.ProjetoID, &a.TarefaID, &a.TarefaTitulo, &a.UsuarioID, &a.UsuarioNome,
			&a.DataApontamento, &a.Horas, &a.Descricao, &a.CustoHora, &a.CustoTotal, &a.CreatedAt,
		); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, a)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	if lista == nil {
		lista = []domain.Apontamento{}
	}
	return lista, nil
}

func (r *Repository) ProjetoDoApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error) {
	var projetoID uuid.UUID
	err := db.QueryRow(ctx, `SELECT projeto_id FROM projetos_apontamentos WHERE id = $1`, id).Scan(&projetoID)
	if database.IsNoRows(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, mapDB(err)
	}
	return projetoID, true, nil
}

func (r *Repository) ExcluirApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	const q = `DELETE FROM projetos_apontamentos WHERE id = $1`
	_, err := db.Exec(ctx, q, id)
	return mapDB(err)
}

func (r *Repository) ObterCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Carta, bool, error) {
	const q = `SELECT projeto_id, onde, patrocinador, metodologia, centro_custo, created_at, updated_at
		FROM projetos_carta WHERE projeto_id = $1`
	var c domain.Carta
	err := db.QueryRow(ctx, q, projetoID).Scan(
		&c.ProjetoID, &c.Onde, &c.Patrocinador, &c.Metodologia, &c.CentroCusto, &c.CreatedAt, &c.UpdatedAt,
	)
	if database.IsNoRows(err) {
		return domain.Carta{}, false, nil
	}
	if err != nil {
		return domain.Carta{}, false, mapDB(err)
	}
	return c, true, nil
}

func (r *Repository) SalvarCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID, e domain.EntradaCarta) (domain.Carta, error) {
	now := time.Now().UTC()
	const q = `INSERT INTO projetos_carta (
		projeto_id, onde, patrocinador, metodologia, centro_custo, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $6)
	ON CONFLICT (projeto_id) DO UPDATE SET
		onde = EXCLUDED.onde,
		patrocinador = EXCLUDED.patrocinador,
		metodologia = EXCLUDED.metodologia,
		centro_custo = EXCLUDED.centro_custo,
		updated_at = EXCLUDED.updated_at
	RETURNING projeto_id, onde, patrocinador, metodologia, centro_custo, created_at, updated_at`
	var c domain.Carta
	err := db.QueryRow(ctx, q, projetoID, e.Onde, e.Patrocinador, e.Metodologia, e.CentroCusto, now).Scan(
		&c.ProjetoID, &c.Onde, &c.Patrocinador, &c.Metodologia, &c.CentroCusto, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return domain.Carta{}, mapDB(err)
	}
	return c, nil
}

func (r *Repository) AtualizarPlano(ctx context.Context, db database.DBTX, id uuid.UUID, p domain.Plano5W2H) (*domain.Projeto, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`UPDATE projetos SET
		titulo = $2, descricao = $3, objetivo = $4,
		data_inicio_planejada = $5, data_fim_planejada = $6,
		orcamento = $7, gerente_id = $8, gerente_nome = $9, updated_at = $10
		WHERE id = $1
		RETURNING %s`, colunasProjeto)
	item, err := scanProjeto(db.QueryRow(ctx, q,
		id, p.Titulo, p.Descricao, p.Objetivo, p.Inicio, p.Fim, p.Orcamento, p.GerenteID, p.GerenteNome, now,
	))
	if err != nil {
		return nil, mapDB(err)
	}
	return &item, nil
}

func (r *Repository) ContarEtapasHow(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (int, int, error) {
	const q = `SELECT
		COUNT(*) FILTER (WHERE origem = 'how'),
		COUNT(*) FILTER (WHERE origem = 'how' AND estado = 'concluida')
		FROM projetos_tarefas WHERE projeto_id = $1`
	var total, concluidas int
	if err := db.QueryRow(ctx, q, projetoID).Scan(&total, &concluidas); err != nil {
		return 0, 0, mapDB(err)
	}
	return total, concluidas, nil
}

func (r *Repository) PrazosVencidos(ctx context.Context, db database.DBTX, agora time.Time) ([]domain.PrazoTarefa, []domain.PrazoProjeto, error) {
	const tarefas = `SELECT t.id, t.titulo, t.data_limite, t.responsavel_id,
		p.id, p.numero, p.titulo, p.gerente_id
		FROM projetos_tarefas t
		JOIN projetos p ON p.id = t.projeto_id
		WHERE t.data_limite IS NOT NULL AND t.data_limite < $1
		  AND t.estado <> 'concluida'
		  AND p.estado NOT IN ('concluido', 'cancelado')`
	rows, err := db.Query(ctx, tarefas, agora)
	if err != nil {
		return nil, nil, mapDB(err)
	}
	defer rows.Close()
	var lista []domain.PrazoTarefa
	for rows.Next() {
		var item domain.PrazoTarefa
		if err := rows.Scan(&item.TarefaID, &item.Titulo, &item.DataLimite, &item.ResponsavelID, &item.ProjetoID, &item.ProjetoNumero, &item.ProjetoTitulo, &item.GerenteID); err != nil {
			return nil, nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, mapDB(err)
	}
	ids := make([]uuid.UUID, len(lista))
	for i := range lista {
		ids[i] = lista[i].TarefaID
	}
	mapa, err := r.PessoasPorTarefa(ctx, db, ids)
	if err != nil {
		return nil, nil, err
	}
	for i := range lista {
		for _, pessoa := range mapa[lista[i].TarefaID] {
			lista[i].Pessoas = append(lista[i].Pessoas, pessoa.ID)
		}
	}

	const projetosQ = `SELECT id, numero, titulo, data_fim_planejada, gerente_id
		FROM projetos
		WHERE data_fim_planejada IS NOT NULL AND data_fim_planejada < $1
		  AND estado NOT IN ('concluido', 'cancelado')`
	prows, err := db.Query(ctx, projetosQ, agora)
	if err != nil {
		return nil, nil, mapDB(err)
	}
	defer prows.Close()
	var fins []domain.PrazoProjeto
	for prows.Next() {
		var item domain.PrazoProjeto
		if err := prows.Scan(&item.ID, &item.Numero, &item.Titulo, &item.Fim, &item.GerenteID); err != nil {
			return nil, nil, mapDB(err)
		}
		fins = append(fins, item)
	}
	if err := prows.Err(); err != nil {
		return nil, nil, mapDB(err)
	}
	return lista, fins, nil
}

func (r *Repository) SomarCusto(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (float64, error) {
	const q = `SELECT COALESCE(SUM(horas * custo_hora), 0) FROM projetos_apontamentos WHERE projeto_id = $1`
	var custo float64
	if err := db.QueryRow(ctx, q, projetoID).Scan(&custo); err != nil {
		return 0, mapDB(err)
	}
	return custo, nil
}
