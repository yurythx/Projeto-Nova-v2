package application

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type mem struct {
	sigla, nome string
	forms       map[uuid.UUID]domain.Formulario
	servs       map[uuid.UUID]domain.Servico
	pedidos     map[uuid.UUID]domain.Pedido
	hist        []domain.Historico
	resp        map[string]bool
	tecnicos    []domain.ResponsavelTecnico
	agenda      []domain.Agendamento
	faltas      int
	ultima      time.Time
	falha       string
	numeros     map[string]bool
}

func novoMem() *mem {
	return &mem{
		sigla: "SEMA", nome: "Meio Ambiente",
		forms: map[uuid.UUID]domain.Formulario{}, servs: map[uuid.UUID]domain.Servico{},
		pedidos: map[uuid.UUID]domain.Pedido{}, resp: map[string]bool{}, numeros: map[string]bool{},
	}
}

func (m *mem) erra(nome string) error {
	if m.falha == nome {
		return domain.ErrInvalido
	}
	return nil
}

func (m *mem) Unidade(context.Context, database.DBTX, uuid.UUID) (string, string, error) {
	if err := m.erra("unidade"); err != nil {
		return "", "", err
	}
	if m.nome == "" {
		return "", "", domain.ErrNaoEncontrado
	}
	return m.sigla, m.nome, nil
}
func (m *mem) InserirFormulario(_ context.Context, _ database.DBTX, f domain.Formulario) error {
	if err := m.erra("inserir-form"); err != nil {
		return err
	}
	m.forms[f.ID] = f
	return nil
}
func (m *mem) AtualizarFormulario(_ context.Context, _ database.DBTX, f domain.Formulario) error {
	if err := m.erra("atualizar-form"); err != nil {
		return err
	}
	if _, ok := m.forms[f.ID]; !ok {
		return domain.ErrNaoEncontrado
	}
	m.forms[f.ID] = f
	return nil
}
func (m *mem) BuscarFormulario(_ context.Context, _ database.DBTX, id uuid.UUID) (domain.Formulario, error) {
	if err := m.erra("buscar-form"); err != nil {
		return domain.Formulario{}, err
	}
	f, ok := m.forms[id]
	if !ok {
		return domain.Formulario{}, domain.ErrNaoEncontrado
	}
	return f, nil
}
func (m *mem) ListarFormularios(context.Context, database.DBTX, domain.Visao) ([]domain.Formulario, error) {
	if m.falha == "nulo" {
		return nil, nil
	}
	if err := m.erra("listar-form"); err != nil {
		return nil, err
	}
	out := []domain.Formulario{}
	for _, f := range m.forms {
		out = append(out, f)
	}
	return out, nil
}
func (m *mem) InserirServico(_ context.Context, _ database.DBTX, s domain.Servico) error {
	if err := m.erra("inserir-serv"); err != nil {
		return err
	}
	m.servs[s.ID] = s
	return nil
}
func (m *mem) AtualizarServico(_ context.Context, _ database.DBTX, s domain.Servico) error {
	if err := m.erra("atualizar-serv"); err != nil {
		return err
	}
	if _, ok := m.servs[s.ID]; !ok {
		return domain.ErrNaoEncontrado
	}
	m.servs[s.ID] = s
	return nil
}
func (m *mem) BuscarServico(_ context.Context, _ database.DBTX, id uuid.UUID) (domain.Servico, error) {
	if err := m.erra("buscar-serv"); err != nil {
		return domain.Servico{}, err
	}
	s, ok := m.servs[id]
	if !ok {
		return domain.Servico{}, domain.ErrNaoEncontrado
	}
	return s, nil
}
func (m *mem) ListarServicos(_ context.Context, _ database.DBTX, soAtivos bool, _ *uuid.UUID) ([]domain.Servico, error) {
	if m.falha == "nulo" {
		return nil, nil
	}
	if err := m.erra("listar-serv"); err != nil {
		return nil, err
	}
	out := []domain.Servico{}
	for _, s := range m.servs {
		if soAtivos && !s.Ativo {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
func (m *mem) ListarServicosDaGestao(context.Context, database.DBTX, domain.Visao) ([]domain.Servico, error) {
	if m.falha == "nulo" {
		return nil, nil
	}
	if err := m.erra("listar-gestao"); err != nil {
		return nil, err
	}
	out := []domain.Servico{}
	for _, s := range m.servs {
		out = append(out, s)
	}
	return out, nil
}
func (m *mem) InserirPedido(_ context.Context, _ database.DBTX, p domain.Pedido) error {
	if err := m.erra("inserir-pedido"); err != nil {
		return err
	}
	if m.numeros[p.Numero] {
		return domain.ErrNumeroRepetido
	}
	m.numeros[p.Numero] = true
	m.pedidos[p.ID] = p
	return nil
}
func (m *mem) AtualizarPedido(_ context.Context, _ database.DBTX, p domain.Pedido) error {
	if err := m.erra("atualizar-pedido"); err != nil {
		return err
	}
	if _, ok := m.pedidos[p.ID]; !ok {
		return domain.ErrNaoEncontrado
	}
	m.pedidos[p.ID] = p
	return nil
}
func (m *mem) BuscarPedido(_ context.Context, _ database.DBTX, id uuid.UUID) (domain.Pedido, error) {
	if err := m.erra("buscar-pedido"); err != nil {
		return domain.Pedido{}, err
	}
	p, ok := m.pedidos[id]
	if !ok {
		return domain.Pedido{}, domain.ErrNaoEncontrado
	}
	return p, nil
}
func (m *mem) BuscarPorNumero(_ context.Context, _ database.DBTX, numero string) (domain.Pedido, error) {
	if err := m.erra("buscar-numero"); err != nil {
		return domain.Pedido{}, err
	}
	for _, p := range m.pedidos {
		if p.Numero == numero {
			return p, nil
		}
	}
	return domain.Pedido{}, domain.ErrNaoEncontrado
}
func (m *mem) ListarPedidos(context.Context, database.DBTX, domain.Visao) ([]domain.Pedido, error) {
	if m.falha == "nulo" {
		return nil, nil
	}
	if err := m.erra("listar-pedidos"); err != nil {
		return nil, err
	}
	out := []domain.Pedido{}
	for _, p := range m.pedidos {
		out = append(out, p)
	}
	return out, nil
}
func (m *mem) NumeroExiste(_ context.Context, _ database.DBTX, numero string) (bool, error) {
	if err := m.erra("numero"); err != nil {
		return false, err
	}
	return m.numeros[numero], nil
}
func (m *mem) CPFEmAberto(_ context.Context, _ database.DBTX, servico uuid.UUID, cpf string) (int, error) {
	if err := m.erra("cpf"); err != nil {
		return 0, err
	}
	n := 0
	for _, p := range m.pedidos {
		if p.ServicoID == servico && p.CPF == cpf && !domain.EstadoFinal(p.Status) {
			n++
		}
	}
	return n, nil
}
func (m *mem) InserirHistorico(_ context.Context, _ database.DBTX, h domain.Historico) error {
	if err := m.erra("hist"); err != nil {
		return err
	}
	m.hist = append(m.hist, h)
	return nil
}
func (m *mem) ListarHistorico(_ context.Context, _ database.DBTX, pedido uuid.UUID) ([]domain.Historico, error) {
	if m.falha == "nulo" {
		return nil, nil
	}
	if err := m.erra("listar-hist"); err != nil {
		return nil, err
	}
	out := []domain.Historico{}
	for _, h := range m.hist {
		if h.PedidoID == pedido {
			out = append(out, h)
		}
	}
	return out, nil
}
func (m *mem) InserirResponsavel(_ context.Context, _ database.DBTX, r domain.ResponsavelTecnico) error {
	if err := m.erra("resp"); err != nil {
		return err
	}
	if _, ok := m.resp[r.CPF]; ok {
		return domain.ErrCPFCadastrado
	}
	m.resp[r.CPF] = r.Ativo
	m.tecnicos = append(m.tecnicos, r)
	return nil
}
func (m *mem) ListarResponsaveis(context.Context, database.DBTX) ([]domain.ResponsavelTecnico, error) {
	if err := m.erra("listar-resp"); err != nil {
		return nil, err
	}
	if m.tecnicos == nil {
		return []domain.ResponsavelTecnico{}, nil
	}
	return append([]domain.ResponsavelTecnico{}, m.tecnicos...), nil
}
func (m *mem) ResponsavelAtivo(_ context.Context, _ database.DBTX, cpf string) (bool, error) {
	if err := m.erra("resp-ativo"); err != nil {
		return false, err
	}
	return m.resp[cpf], nil
}
func (m *mem) InserirAgendamento(_ context.Context, _ database.DBTX, a domain.Agendamento) error {
	if err := m.erra("agenda"); err != nil {
		return err
	}
	m.agenda = append(m.agenda, a)
	return nil
}
func (m *mem) AtualizarAgendamento(_ context.Context, _ database.DBTX, a domain.Agendamento) error {
	if err := m.erra("atualizar-agenda"); err != nil {
		return err
	}
	for i := range m.agenda {
		if m.agenda[i].ID == a.ID {
			m.agenda[i] = a
			return nil
		}
	}
	return domain.ErrNaoEncontrado
}
func (m *mem) ListarHoras(_ context.Context, _ database.DBTX, servico uuid.UUID, dia string) ([]string, error) {
	if err := m.erra("horas"); err != nil {
		return nil, err
	}
	out := []string{}
	for _, a := range m.agenda {
		if a.ServicoID == servico && a.Data == dia && (a.Status == domain.HorarioAgendado || a.Status == domain.HorarioConfirmado || a.Status == domain.HorarioFalta) {
			out = append(out, a.Hora)
		}
	}
	return out, nil
}
func (m *mem) BuscarAgendamento(_ context.Context, _ database.DBTX, pedido uuid.UUID) (domain.Agendamento, error) {
	if err := m.erra("buscar-agenda"); err != nil {
		return domain.Agendamento{}, err
	}
	for _, a := range m.agenda {
		if a.PedidoID == pedido {
			return a, nil
		}
	}
	return domain.Agendamento{}, domain.ErrNaoEncontrado
}
func (m *mem) ContarAgendamentos(context.Context, database.DBTX, uuid.UUID, string) (int, error) {
	if err := m.erra("contar"); err != nil {
		return 0, err
	}
	return len(m.agenda), nil
}
func (m *mem) LerFaltas(context.Context, database.DBTX, string) (int, time.Time, error) {
	if err := m.erra("faltas"); err != nil {
		return 0, time.Time{}, err
	}
	return m.faltas, m.ultima, nil
}
func (m *mem) SomarFalta(_ context.Context, _ database.DBTX, _ string, quando time.Time) error {
	if err := m.erra("somar"); err != nil {
		return err
	}
	m.faltas++
	m.ultima = quando
	return nil
}

type stubTx struct {
	falha int
	n     int
}

func (stubTx) Begin(context.Context) (pgx.Tx, error) { return &stubTx{}, nil }
func (stubTx) Commit(context.Context) error          { return nil }
func (stubTx) Rollback(context.Context) error        { return nil }
func (stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (stubTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (s *stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	s.n++
	if s.falha > 0 && s.n == s.falha {
		return pgconn.CommandTag{}, domain.ErrInvalido
	}
	return pgconn.NewCommandTag("INSERT 1"), nil
}
func (stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (stubTx) QueryRow(context.Context, string, ...any) pgx.Row        { return stubRow{} }
func (stubTx) Conn() *pgx.Conn                                         { return nil }

type stubRow struct{}

func (stubRow) Scan(...any) error { return nil }

type sinoFalso struct{ erro error }

func (s sinoFalso) Enviar(context.Context, []uuid.UUID, notificacoes.Nova) (int, error) {
	if s.erro != nil {
		return 0, s.erro
	}
	return 1, nil
}

func servicoTeste(m *mem) *Service {
	s := NewService(nil, m, outbox.NewWriter("nova.protocolos"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.run = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, &stubTx{})
	}
	s.agora = func() time.Time { return time.Date(2026, 11, 16, 15, 0, 0, 0, domain.Cuiaba) }
	n := 0
	s.sorteio = func() int { n++; return n }
	return s
}
