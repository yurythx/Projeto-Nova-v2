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

const colunasTarefa = `id, projeto_id, titulo, descricao, responsavel_nome, estado, ordem,
	data_limite, created_at, updated_at`

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
		&t.ID, &t.ProjetoID, &t.Titulo, &t.Descricao, &t.ResponsavelNome, &t.Estado, &t.Ordem,
		&t.DataLimite, &t.CreatedAt, &t.UpdatedAt,
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

	q := fmt.Sprintf(`SELECT %s FROM projetos WHERE 1=1`, colunasProjeto)

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
		item, err := scanProjeto(rows)
		if err != nil {
			return nil, mapDB(err)
		}
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
		id, projeto_id, titulo, descricao, responsavel_nome, estado, ordem, data_limite, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING created_at, updated_at`

	err := db.QueryRow(ctx, q,
		t.ID, t.ProjetoID, t.Titulo, t.Descricao, t.ResponsavelNome, t.Estado, t.Ordem, t.DataLimite, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.CreatedAt, &t.UpdatedAt)

	return mapDB(err)
}

func (r *Repository) AtualizarTarefa(ctx context.Context, db database.DBTX, id uuid.UUID, titulo, descricao, responsavel, estado string, ordem int, dataLimite *time.Time) (*domain.Tarefa, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`UPDATE projetos_tarefas SET
		titulo = $2, descricao = $3, responsavel_nome = $4, estado = $5, ordem = $6, data_limite = COALESCE($7, data_limite), updated_at = $8
		WHERE id = $1
		RETURNING %s`, colunasTarefa)

	t, err := scanTarefa(db.QueryRow(ctx, q, id, titulo, descricao, responsavel, estado, ordem, dataLimite, now))
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
	return lista, nil
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

func (r *Repository) ExcluirApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	const q = `DELETE FROM projetos_apontamentos WHERE id = $1`
	_, err := db.Exec(ctx, q, id)
	return mapDB(err)
}
