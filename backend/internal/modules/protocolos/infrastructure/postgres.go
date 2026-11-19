// Package infrastructure persiste a carta e o pedido em PostgreSQL.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// PostgresRepository implementa domain.Repository.
type PostgresRepository struct{}

// NewPostgresRepository cria o repositório.
func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func traduz(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNaoEncontrado
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "protocolos_pedidos_numero_key" {
		return domain.ErrNumeroRepetido
	}
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "protocolos_responsaveis_cpf_key" {
		return domain.ErrCPFCadastrado
	}
	return err
}

func jsonBytes(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return []byte("[]"), nil
	}
	return b, nil
}

func lerJSON(raw []byte, dest any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

const filtro = `($1::bool OR t.unidade_id = ANY($2::uuid[]) OR u.entidade_id = ANY($3::uuid[]))`

// Unidade devolve sigla e nome de uma unidade ativa.
func (r *PostgresRepository) Unidade(ctx context.Context, db database.DBTX, id uuid.UUID) (string, string, error) {
	var sigla, nome string
	err := db.QueryRow(ctx, `SELECT u.sigla, u.nome FROM unidades u
		JOIN entidades e ON e.id = u.entidade_id
		WHERE u.id = $1 AND u.ativo AND e.ativo`, id).Scan(&sigla, &nome)
	if err != nil {
		return "", "", fmt.Errorf("protocolos.Unidade: %w", traduz(err))
	}
	return sigla, nome, nil
}

func (r *PostgresRepository) InserirFormulario(ctx context.Context, db database.DBTX, f domain.Formulario) error {
	campos, err := jsonBytes(f.Campos)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO protocolos_formularios
		(id, unidade_id, titulo, descricao, instrucao, campos, ativo, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		f.ID, f.UnidadeID, f.Titulo, f.Descricao, f.Instrucao, campos, f.Ativo, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.InserirFormulario: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AtualizarFormulario(ctx context.Context, db database.DBTX, f domain.Formulario) error {
	campos, err := jsonBytes(f.Campos)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, `UPDATE protocolos_formularios SET titulo=$2, descricao=$3, instrucao=$4, campos=$5, ativo=$6, updated_at=$7 WHERE id=$1`,
		f.ID, f.Titulo, f.Descricao, f.Instrucao, campos, f.Ativo, f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.AtualizarFormulario: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func scanFormulario(row pgx.Row) (domain.Formulario, error) {
	var f domain.Formulario
	var campos []byte
	err := row.Scan(&f.ID, &f.UnidadeID, &f.Titulo, &f.Descricao, &f.Instrucao, &campos, &f.Ativo, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return domain.Formulario{}, traduz(err)
	}
	if err := lerJSON(campos, &f.Campos); err != nil {
		return domain.Formulario{}, err
	}
	if f.Campos == nil {
		f.Campos = []domain.Campo{}
	}
	return f, nil
}

func (r *PostgresRepository) BuscarFormulario(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Formulario, error) {
	f, err := scanFormulario(db.QueryRow(ctx, `SELECT id, unidade_id, titulo, descricao, instrucao, campos, ativo, created_at, updated_at
		FROM protocolos_formularios WHERE id=$1`, id))
	if err != nil {
		return domain.Formulario{}, fmt.Errorf("protocolos.BuscarFormulario: %w", err)
	}
	return f, nil
}

func (r *PostgresRepository) ListarFormularios(ctx context.Context, db database.DBTX, v domain.Visao) ([]domain.Formulario, error) {
	rows, err := db.Query(ctx, `SELECT t.id, t.unidade_id, t.titulo, t.descricao, t.instrucao, t.campos, t.ativo, t.created_at, t.updated_at
		FROM protocolos_formularios t JOIN unidades u ON u.id = t.unidade_id
		WHERE `+filtro+` ORDER BY t.titulo`, v.Tudo, v.Unidades, v.Entidades)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarFormularios: %w", err)
	}
	defer rows.Close()
	out := []domain.Formulario{}
	for rows.Next() {
		f, err := scanFormulario(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func servicoArgs(s domain.Servico) (workflow, agenda, req, docs []byte, err error) {
	if workflow, err = jsonBytes(s.Workflow); err != nil {
		return
	}
	if agenda, err = jsonBytes(s.Agenda); err != nil {
		return
	}
	if req, err = jsonBytes(s.Requisitos); err != nil {
		return
	}
	docs, err = jsonBytes(s.Documentos)
	return
}

func (r *PostgresRepository) InserirServico(ctx context.Context, db database.DBTX, s domain.Servico) error {
	w, a, req, docs, err := servicoArgs(s)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO protocolos_servicos (
		id, unidade_id, formulario_id, nome, descricao, categoria, prazo_dias, taxa_centavos, destaque, ativo,
		aprova_sozinho, exige_agendamento, so_responsavel_tecnico, impedir_cpf_aberto, campo_anonimo,
		workflow, agenda, requisitos, documentos, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		s.ID, s.UnidadeID, s.FormularioID, s.Nome, s.Descricao, s.Categoria, s.PrazoDias, s.TaxaCentavos, s.Destaque, s.Ativo,
		s.AprovaSozinho, s.ExigeAgendamento, s.SoResponsavelTecnico, s.ImpedirCPFAberto, s.CampoAnonimo,
		w, a, req, docs, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.InserirServico: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AtualizarServico(ctx context.Context, db database.DBTX, s domain.Servico) error {
	w, a, req, docs, err := servicoArgs(s)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, `UPDATE protocolos_servicos SET unidade_id=$2, formulario_id=$3, nome=$4, descricao=$5, categoria=$6,
		prazo_dias=$7, taxa_centavos=$8, destaque=$9, ativo=$10, aprova_sozinho=$11, exige_agendamento=$12,
		so_responsavel_tecnico=$13, impedir_cpf_aberto=$14, campo_anonimo=$15, workflow=$16, agenda=$17,
		requisitos=$18, documentos=$19, updated_at=$20 WHERE id=$1`,
		s.ID, s.UnidadeID, s.FormularioID, s.Nome, s.Descricao, s.Categoria, s.PrazoDias, s.TaxaCentavos, s.Destaque, s.Ativo,
		s.AprovaSozinho, s.ExigeAgendamento, s.SoResponsavelTecnico, s.ImpedirCPFAberto, s.CampoAnonimo, w, a, req, docs, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.AtualizarServico: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func scanServico(row pgx.Row) (domain.Servico, error) {
	var s domain.Servico
	var w, a, req, docs []byte
	err := row.Scan(&s.ID, &s.UnidadeID, &s.FormularioID, &s.Nome, &s.Descricao, &s.Categoria, &s.PrazoDias, &s.TaxaCentavos,
		&s.Destaque, &s.Ativo, &s.AprovaSozinho, &s.ExigeAgendamento, &s.SoResponsavelTecnico, &s.ImpedirCPFAberto, &s.CampoAnonimo,
		&w, &a, &req, &docs, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return domain.Servico{}, traduz(err)
	}
	if err := lerJSON(w, &s.Workflow); err != nil {
		return domain.Servico{}, err
	}
	if err := lerJSON(a, &s.Agenda); err != nil {
		return domain.Servico{}, err
	}
	if err := lerJSON(req, &s.Requisitos); err != nil {
		return domain.Servico{}, err
	}
	if err := lerJSON(docs, &s.Documentos); err != nil {
		return domain.Servico{}, err
	}
	if s.Requisitos == nil {
		s.Requisitos = []domain.Orientacao{}
	}
	if s.Documentos == nil {
		s.Documentos = []domain.Orientacao{}
	}
	return s, nil
}

const colsServico = `id, unidade_id, formulario_id, nome, descricao, categoria, prazo_dias, taxa_centavos, destaque, ativo,
	aprova_sozinho, exige_agendamento, so_responsavel_tecnico, impedir_cpf_aberto, campo_anonimo,
	workflow, agenda, requisitos, documentos, created_at, updated_at`

func (r *PostgresRepository) BuscarServico(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Servico, error) {
	s, err := scanServico(db.QueryRow(ctx, `SELECT `+colsServico+` FROM protocolos_servicos WHERE id=$1`, id))
	if err != nil {
		return domain.Servico{}, fmt.Errorf("protocolos.BuscarServico: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) ListarServicos(ctx context.Context, db database.DBTX, soAtivos bool, unidade *uuid.UUID) ([]domain.Servico, error) {
	rows, err := db.Query(ctx, `SELECT `+colsServico+` FROM protocolos_servicos
		WHERE ($1::bool = false OR ativo) AND ($2::uuid IS NULL OR unidade_id = $2)
		ORDER BY destaque DESC, nome`, soAtivos, unidade)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarServicos: %w", err)
	}
	defer rows.Close()
	return lerServicos(rows)
}

func (r *PostgresRepository) ListarServicosDaGestao(ctx context.Context, db database.DBTX, v domain.Visao) ([]domain.Servico, error) {
	rows, err := db.Query(ctx, `SELECT t.id, t.unidade_id, t.formulario_id, t.nome, t.descricao, t.categoria, t.prazo_dias, t.taxa_centavos,
		t.destaque, t.ativo, t.aprova_sozinho, t.exige_agendamento, t.so_responsavel_tecnico, t.impedir_cpf_aberto, t.campo_anonimo,
		t.workflow, t.agenda, t.requisitos, t.documentos, t.created_at, t.updated_at
		FROM protocolos_servicos t JOIN unidades u ON u.id = t.unidade_id
		WHERE `+filtro+` ORDER BY t.nome`, v.Tudo, v.Unidades, v.Entidades)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarServicosDaGestao: %w", err)
	}
	defer rows.Close()
	return lerServicos(rows)
}

func lerServicos(rows pgx.Rows) ([]domain.Servico, error) {
	out := []domain.Servico{}
	for rows.Next() {
		s, err := scanServico(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) InserirPedido(ctx context.Context, db database.DBTX, p domain.Pedido) error {
	resp, campos, err := pedidoJSON(p)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO protocolos_pedidos (
		id, numero, servico_id, formulario_id, unidade_id, departamento_id, nome, cpf, email, telefone,
		respostas, campos, status, etapa_id, prioridade, prazo_em, decisao, justificativa, complemento,
		usuario_id, responsavel_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
		p.ID, p.Numero, p.ServicoID, p.FormularioID, p.UnidadeID, p.DepartamentoID, p.Nome, p.CPF, p.Email, p.Telefone,
		resp, campos, p.Status, p.EtapaID, p.Prioridade, p.PrazoEm, p.Decisao, p.Justificativa, p.Complemento,
		p.UsuarioID, p.ResponsavelID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.InserirPedido: %w", traduz(err))
	}
	return nil
}

func pedidoJSON(p domain.Pedido) (resp, campos []byte, err error) {
	if p.Respostas == nil {
		p.Respostas = map[string]any{}
	}
	if resp, err = json.Marshal(p.Respostas); err != nil {
		return
	}
	campos, err = jsonBytes(p.Campos)
	return
}

func (r *PostgresRepository) AtualizarPedido(ctx context.Context, db database.DBTX, p domain.Pedido) error {
	resp, campos, err := pedidoJSON(p)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, `UPDATE protocolos_pedidos SET unidade_id=$2, departamento_id=$3, respostas=$4, campos=$5,
		status=$6, etapa_id=$7, prazo_em=$8, decisao=$9, justificativa=$10, complemento=$11, responsavel_id=$12, updated_at=$13
		WHERE id=$1`,
		p.ID, p.UnidadeID, p.DepartamentoID, resp, campos, p.Status, p.EtapaID, p.PrazoEm, p.Decisao, p.Justificativa, p.Complemento, p.ResponsavelID, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.AtualizarPedido: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func scanPedido(row pgx.Row) (domain.Pedido, error) {
	var p domain.Pedido
	var resp, campos []byte
	err := row.Scan(&p.ID, &p.Numero, &p.ServicoID, &p.FormularioID, &p.UnidadeID, &p.DepartamentoID, &p.Nome, &p.CPF, &p.Email, &p.Telefone,
		&resp, &campos, &p.Status, &p.EtapaID, &p.Prioridade, &p.PrazoEm, &p.Decisao, &p.Justificativa, &p.Complemento,
		&p.UsuarioID, &p.ResponsavelID, &p.CreatedAt, &p.UpdatedAt, &p.ServicoNome, &p.UnidadeNome)
	if err != nil {
		return domain.Pedido{}, traduz(err)
	}
	if err := lerJSON(resp, &p.Respostas); err != nil {
		return domain.Pedido{}, err
	}
	if err := lerJSON(campos, &p.Campos); err != nil {
		return domain.Pedido{}, err
	}
	if p.Respostas == nil {
		p.Respostas = map[string]any{}
	}
	return p, nil
}

const colsPedido = `p.id, p.numero, p.servico_id, p.formulario_id, p.unidade_id, p.departamento_id, p.nome, p.cpf, p.email, p.telefone,
	p.respostas, p.campos, p.status, p.etapa_id, p.prioridade, p.prazo_em, p.decisao, p.justificativa, p.complemento,
	p.usuario_id, p.responsavel_id, p.created_at, p.updated_at, s.nome, u.nome`

func (r *PostgresRepository) BuscarPedido(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Pedido, error) {
	p, err := scanPedido(db.QueryRow(ctx, `SELECT `+colsPedido+` FROM protocolos_pedidos p
		JOIN protocolos_servicos s ON s.id = p.servico_id JOIN unidades u ON u.id = p.unidade_id WHERE p.id=$1`, id))
	if err != nil {
		return domain.Pedido{}, fmt.Errorf("protocolos.BuscarPedido: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) BuscarPorNumero(ctx context.Context, db database.DBTX, numero string) (domain.Pedido, error) {
	p, err := scanPedido(db.QueryRow(ctx, `SELECT `+colsPedido+` FROM protocolos_pedidos p
		JOIN protocolos_servicos s ON s.id = p.servico_id JOIN unidades u ON u.id = p.unidade_id WHERE p.numero=$1`, numero))
	if err != nil {
		return domain.Pedido{}, fmt.Errorf("protocolos.BuscarPorNumero: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) ListarPedidos(ctx context.Context, db database.DBTX, v domain.Visao) ([]domain.Pedido, error) {
	rows, err := db.Query(ctx, `SELECT `+colsPedido+` FROM protocolos_pedidos p
		JOIN protocolos_servicos s ON s.id = p.servico_id
		JOIN unidades u ON u.id = p.unidade_id
		WHERE ($1::bool OR p.unidade_id = ANY($2::uuid[]) OR u.entidade_id = ANY($3::uuid[]))
		ORDER BY p.created_at DESC`, v.Tudo, v.Unidades, v.Entidades)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarPedidos: %w", err)
	}
	defer rows.Close()
	out := []domain.Pedido{}
	for rows.Next() {
		p, err := scanPedido(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) NumeroExiste(ctx context.Context, db database.DBTX, numero string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM protocolos_pedidos WHERE numero=$1)`, numero).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("protocolos.NumeroExiste: %w", err)
	}
	return ok, nil
}

func (r *PostgresRepository) CPFEmAberto(ctx context.Context, db database.DBTX, servico uuid.UUID, cpf string) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM protocolos_pedidos
		WHERE servico_id=$1 AND cpf=$2 AND status NOT IN ('concluido','aprovado','rejeitado','cancelado')`, servico, cpf).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("protocolos.CPFEmAberto: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) InserirHistorico(ctx context.Context, db database.DBTX, h domain.Historico) error {
	anexos, err := jsonBytes(h.Anexos)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO protocolos_historico (id, pedido_id, acao, descricao, status, anexos, usuario_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, h.ID, h.PedidoID, h.Acao, h.Descricao, h.Status, anexos, h.UsuarioID, h.CreatedAt)
	if err != nil {
		return fmt.Errorf("protocolos.InserirHistorico: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListarHistorico(ctx context.Context, db database.DBTX, pedido uuid.UUID) ([]domain.Historico, error) {
	rows, err := db.Query(ctx, `SELECT id, pedido_id, acao, descricao, status, anexos, usuario_id, created_at
		FROM protocolos_historico WHERE pedido_id=$1 ORDER BY created_at`, pedido)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarHistorico: %w", err)
	}
	defer rows.Close()
	out := []domain.Historico{}
	for rows.Next() {
		var h domain.Historico
		var anexos []byte
		if err := rows.Scan(&h.ID, &h.PedidoID, &h.Acao, &h.Descricao, &h.Status, &anexos, &h.UsuarioID, &h.CreatedAt); err != nil {
			return nil, err
		}
		if err := lerJSON(anexos, &h.Anexos); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListarResponsaveis(ctx context.Context, db database.DBTX) ([]domain.ResponsavelTecnico, error) {
	rows, err := db.Query(ctx, `SELECT id, cpf, nome, ativo FROM protocolos_responsaveis ORDER BY nome`)
	if err != nil {
		return nil, fmt.Errorf("protocolos.ListarResponsaveis: %w", err)
	}
	defer rows.Close()
	out := []domain.ResponsavelTecnico{}
	for rows.Next() {
		var resp domain.ResponsavelTecnico
		if err := rows.Scan(&resp.ID, &resp.CPF, &resp.Nome, &resp.Ativo); err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) InserirResponsavel(ctx context.Context, db database.DBTX, resp domain.ResponsavelTecnico) error {
	_, err := db.Exec(ctx, `INSERT INTO protocolos_responsaveis (id, cpf, nome, ativo) VALUES ($1,$2,$3,$4)`,
		resp.ID, resp.CPF, resp.Nome, resp.Ativo)
	if err != nil {
		return fmt.Errorf("protocolos.InserirResponsavel: %w", traduz(err))
	}
	return nil
}

func (r *PostgresRepository) ResponsavelAtivo(ctx context.Context, db database.DBTX, cpf string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM protocolos_responsaveis WHERE cpf=$1 AND ativo)`, cpf).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("protocolos.ResponsavelAtivo: %w", err)
	}
	return ok, nil
}

func (r *PostgresRepository) InserirAgendamento(ctx context.Context, db database.DBTX, a domain.Agendamento) error {
	_, err := db.Exec(ctx, `INSERT INTO protocolos_agendamentos (id, pedido_id, servico_id, data, hora, status) VALUES ($1,$2,$3,$4,$5,$6)`,
		a.ID, a.PedidoID, a.ServicoID, a.Data, a.Hora, a.Status)
	if err != nil {
		return fmt.Errorf("protocolos.InserirAgendamento: %w", err)
	}
	return nil
}

func (r *PostgresRepository) BuscarAgendamento(ctx context.Context, db database.DBTX, pedido uuid.UUID) (domain.Agendamento, error) {
	var a domain.Agendamento
	var data time.Time
	err := db.QueryRow(ctx, `SELECT id, pedido_id, servico_id, data, hora, status FROM protocolos_agendamentos WHERE pedido_id=$1`, pedido).
		Scan(&a.ID, &a.PedidoID, &a.ServicoID, &data, &a.Hora, &a.Status)
	if err != nil {
		return domain.Agendamento{}, fmt.Errorf("protocolos.BuscarAgendamento: %w", traduz(err))
	}
	a.Data = data.Format("2006-01-02")
	return a, nil
}

func (r *PostgresRepository) ContarAgendamentos(ctx context.Context, db database.DBTX, servico uuid.UUID, dia string) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM protocolos_agendamentos WHERE servico_id=$1 AND data=$2 AND status IN ('agendado','confirmado')`, servico, dia).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("protocolos.ContarAgendamentos: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) LerFaltas(ctx context.Context, db database.DBTX, cpf string) (int, time.Time, error) {
	var n int
	var quando time.Time
	err := db.QueryRow(ctx, `SELECT quantidade, ultima FROM protocolos_faltas WHERE cpf=$1`, cpf).Scan(&n, &quando)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("protocolos.LerFaltas: %w", err)
	}
	return n, quando, nil
}

func (r *PostgresRepository) SomarFalta(ctx context.Context, db database.DBTX, cpf string, quando time.Time) error {
	_, err := db.Exec(ctx, `INSERT INTO protocolos_faltas (cpf, quantidade, ultima) VALUES ($1, 1, $2)
		ON CONFLICT (cpf) DO UPDATE SET quantidade = protocolos_faltas.quantidade + 1, ultima = EXCLUDED.ultima`, cpf, quando)
	if err != nil {
		return fmt.Errorf("protocolos.SomarFalta: %w", err)
	}
	return nil
}
