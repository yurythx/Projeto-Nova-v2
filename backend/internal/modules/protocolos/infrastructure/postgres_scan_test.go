package infrastructure

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type banco struct {
	row  pgx.Row
	rows pgx.Rows
}

func (b banco) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (b banco) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if b.rows == nil {
		return dbtest.NoRows(), nil
	}
	return b.rows, nil
}
func (b banco) QueryRow(context.Context, string, ...any) pgx.Row { return b.row }

type erroLinha struct{ err error }

func (e erroLinha) Scan(...any) error { return e.err }

type valores struct{ cols []any }

func (v valores) Scan(dest ...any) error {
	if len(dest) != len(v.cols) {
		return errors.New("colunas")
	}
	for i := range dest {
		if err := copiar(dest[i], v.cols[i]); err != nil {
			return err
		}
	}
	return nil
}

func copiar(dest, src any) error {
	alvo := reflect.ValueOf(dest).Elem()
	if src == nil {
		alvo.Set(reflect.Zero(alvo.Type()))
		return nil
	}
	sv := reflect.ValueOf(src)
	if sv.Type().AssignableTo(alvo.Type()) {
		alvo.Set(sv)
		return nil
	}
	return errors.New(alvo.Type().String())
}

type grade struct {
	linhas [][]any
	i      int
	err    error
}

func (g *grade) Close()                                       {}
func (g *grade) Err() error                                   { return g.err }
func (g *grade) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (g *grade) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (g *grade) Values() ([]any, error)                       { return nil, nil }
func (g *grade) RawValues() [][]byte                          { return nil }
func (g *grade) Conn() *pgx.Conn                              { return nil }
func (g *grade) Next() bool {
	if g.i >= len(g.linhas) {
		return false
	}
	g.i++
	return true
}
func (g *grade) Scan(dest ...any) error { return valores{g.linhas[g.i-1]}.Scan(dest...) }

func TestLeituras(t *testing.T) {
	r := NewPostgresRepository()
	ctx := context.Background()
	id := uuid.New()
	agora := time.Now()
	vazio := domain.Visao{}
	if _, err := jsonBytes(nil); err != nil || string(must(jsonBytes(map[string]int{"a": 1}))) == "" {
		t.Fatal("json")
	}
	if !errors.Is(traduz(pgx.ErrNoRows), domain.ErrNaoEncontrado) {
		t.Fatal("vazio")
	}
	if traduz(&pgconn.PgError{Code: "23505", ConstraintName: "outra"}) == nil {
		t.Fatal("outra chave")
	}
	sigla, nome, err := r.Unidade(ctx, banco{row: valores{[]any{"SEMA", "Meio Ambiente"}}}, id)
	if err != nil || sigla != "SEMA" || nome == "" {
		t.Fatal(err)
	}
	if _, _, err := r.Unidade(ctx, banco{row: erroLinha{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	form := []any{id, id, "Alvará", "", "", []byte("[]"), true, agora, agora}
	if _, err := r.BuscarFormulario(ctx, banco{row: valores{form}}, id); err != nil {
		t.Fatal(err)
	}
	if _, err := r.BuscarFormulario(ctx, banco{row: valores{[]any{id, id, "Alvará", "", "", []byte("{"), true, agora, agora}}}, id); err == nil {
		t.Fatal("json")
	}
	if lido, err := r.BuscarFormulario(ctx, banco{row: valores{[]any{id, id, "Alvará", "", "", []byte(nil), true, agora, agora}}}, id); err != nil || lido.Campos == nil {
		t.Fatal(err)
	}
	if _, err := r.ListarFormularios(ctx, banco{rows: &grade{linhas: [][]any{form}}}, vazio); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ListarFormularios(ctx, banco{rows: dbtest.ErrRows()}, vazio); err == nil {
		t.Fatal("linhas")
	}
	if _, err := r.ListarFormularios(ctx, banco{rows: dbtest.BadRows()}, vazio); err == nil {
		t.Fatal("scan")
	}
	if err := r.InserirFormulario(ctx, dbtest.ScanFail{}, domain.Formulario{ID: id}); err != nil {
		t.Fatal(err)
	}
	if err := r.AtualizarFormulario(ctx, banco{}, domain.Formulario{ID: id, Campos: []domain.Campo{}}); err != nil {
		t.Fatal(err)
	}
	serv := linhaServico(id, agora, []byte(`{}`), []byte(`{}`), []byte(`[]`), []byte(`[]`))
	lidoServ, err := r.BuscarServico(ctx, banco{row: valores{serv}}, id)
	if err != nil || lidoServ.UnidadeNome != "Meio Ambiente" {
		t.Fatal(lidoServ.UnidadeNome, err)
	}
	if _, err := r.BuscarServico(ctx, banco{row: valores{linhaServico(id, agora, []byte("{"), []byte(`{}`), []byte(`[]`), []byte(`[]`))}}, id); err == nil {
		t.Fatal("workflow")
	}
	if _, err := r.BuscarServico(ctx, banco{row: valores{linhaServico(id, agora, []byte(`{}`), []byte("{"), []byte(`[]`), []byte(`[]`))}}, id); err == nil {
		t.Fatal("agenda")
	}
	if _, err := r.BuscarServico(ctx, banco{row: valores{linhaServico(id, agora, []byte(`{}`), []byte(`{}`), []byte("{"), []byte(`[]`))}}, id); err == nil {
		t.Fatal("requisitos")
	}
	if _, err := r.BuscarServico(ctx, banco{row: valores{linhaServico(id, agora, []byte(`{}`), []byte(`{}`), []byte(`[]`), []byte("{"))}}, id); err == nil {
		t.Fatal("documentos")
	}
	if lido, err := r.BuscarServico(ctx, banco{row: valores{linhaServico(id, agora, []byte(`{}`), []byte(`{}`), []byte("null"), []byte("null"))}}, id); err != nil || lido.Requisitos == nil || lido.Documentos == nil {
		t.Fatal(err)
	}
	if _, err := r.ListarServicos(ctx, banco{rows: &grade{linhas: [][]any{serv}}}, true, &id); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ListarServicosDaGestao(ctx, banco{rows: dbtest.BadRows()}, vazio); err == nil {
		t.Fatal("scan")
	}
	if _, err := r.ListarServicos(ctx, banco{}, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.InserirServico(ctx, banco{}, domain.Servico{ID: id}); err != nil {
		t.Fatal(err)
	}
	if err := r.AtualizarServico(ctx, banco{}, domain.Servico{ID: id}); err != nil {
		t.Fatal(err)
	}
	ped := linhaPedido(id, agora, []byte("null"), []byte("[]"))
	if lido, err := r.BuscarPedido(ctx, banco{row: valores{ped}}, id); err != nil || lido.Respostas == nil {
		t.Fatal(err)
	}
	if _, err := r.BuscarPorNumero(ctx, banco{row: valores{ped}}, "SEMA202600001"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.BuscarPorNumero(ctx, banco{row: valores{linhaPedido(id, agora, []byte("{"), []byte("[]"))}}, "SEMA202600001"); err == nil {
		t.Fatal("respostas")
	}
	if _, err := r.BuscarPorNumero(ctx, banco{row: valores{linhaPedido(id, agora, []byte(`{}`), []byte("{"))}}, "SEMA202600001"); err == nil {
		t.Fatal("campos")
	}
	if _, err := r.ListarPedidos(ctx, banco{rows: &grade{linhas: [][]any{ped}}}, vazio); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ListarPedidos(ctx, banco{rows: dbtest.ErrRows()}, vazio); err == nil {
		t.Fatal("linhas")
	}
	if _, err := r.ListarPedidos(ctx, banco{rows: dbtest.BadRows()}, vazio); err == nil {
		t.Fatal("scan")
	}
	canal := domain.Pedido{Respostas: map[string]any{"x": make(chan int)}}
	if err := r.InserirPedido(ctx, banco{}, canal); err == nil {
		t.Fatal("respostas")
	}
	if err := r.InserirPedido(ctx, banco{}, domain.Pedido{ID: id}); err != nil {
		t.Fatal(err)
	}
	if err := r.AtualizarPedido(ctx, banco{}, canal); err == nil {
		t.Fatal("respostas")
	}
	if err := r.AtualizarPedido(ctx, banco{}, domain.Pedido{ID: id}); err != nil {
		t.Fatal(err)
	}
	ok, err := r.NumeroExiste(ctx, banco{row: valores{[]any{true}}}, "SEMA202600001")
	if err != nil || !ok {
		t.Fatal(err)
	}
	if n, err := r.CPFEmAberto(ctx, banco{row: valores{[]any{2}}}, id, "1"); err != nil || n != 2 {
		t.Fatal(err)
	}
	hist := []any{id, id, "criado", "Pedido aberto", "novo", []byte("[]"), &id, agora}
	if _, err := r.ListarHistorico(ctx, banco{rows: &grade{linhas: [][]any{hist}}}, id); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ListarHistorico(ctx, banco{rows: dbtest.BadRows()}, id); err == nil {
		t.Fatal("histórico")
	}
	if _, err := r.ListarHistorico(ctx, banco{rows: &grade{linhas: [][]any{{id, id, "criado", "Pedido aberto", "novo", []byte("{"), &id, agora}}}}, id); err == nil {
		t.Fatal("anexo")
	}
	if err := r.InserirHistorico(ctx, banco{}, domain.Historico{ID: id}); err != nil {
		t.Fatal(err)
	}
	if lista, err := r.ListarResponsaveis(ctx, banco{rows: &grade{linhas: [][]any{{id, "12345678900", "Ana Lima", true}}}}); err != nil || len(lista) != 1 {
		t.Fatal(err)
	}
	if _, err := r.ListarResponsaveis(ctx, banco{rows: dbtest.BadRows()}); err == nil {
		t.Fatal("responsável")
	}
	if err := r.InserirResponsavel(ctx, banco{}, domain.ResponsavelTecnico{ID: id}); err != nil {
		t.Fatal(err)
	}
	if ativo, err := r.ResponsavelAtivo(ctx, banco{row: valores{[]any{true}}}, "1"); err != nil || !ativo {
		t.Fatal(err)
	}
	if err := r.InserirAgendamento(ctx, banco{}, domain.Agendamento{ID: id}); err != nil {
		t.Fatal(err)
	}
	if ag, err := r.BuscarAgendamento(ctx, banco{row: valores{[]any{id, id, id, agora, "08:00", "agendado"}}}, id); err != nil || ag.Data == "" {
		t.Fatal(err)
	}
	if _, err := r.BuscarAgendamento(ctx, banco{row: erroLinha{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if n, err := r.ContarAgendamentos(ctx, banco{row: valores{[]any{1}}}, id, "2026-11-17"); err != nil || n != 1 {
		t.Fatal(err)
	}
	if horas, err := r.ListarHoras(ctx, banco{rows: &grade{linhas: [][]any{{"08:00"}, {"08:30"}}}}, id, "2026-11-17"); err != nil || len(horas) != 2 || horas[0] != "08:00" {
		t.Fatal(horas, err)
	}
	if _, err := r.ListarHoras(ctx, banco{rows: dbtest.BadRows()}, id, "2026-11-17"); err == nil {
		t.Fatal("hora")
	}
	if err := r.AtualizarAgendamento(ctx, banco{}, domain.Agendamento{ID: id, Status: "falta"}); err != nil {
		t.Fatal(err)
	}
	if err := r.AtualizarAgendamento(ctx, dbtest.ScanFail{}, domain.Agendamento{ID: id, Status: "falta"}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if n, quando, err := r.LerFaltas(ctx, banco{row: valores{[]any{2, agora}}}, "1"); err != nil || n != 2 || quando.IsZero() {
		t.Fatal(err)
	}
	if n, _, err := r.LerFaltas(ctx, banco{row: erroLinha{pgx.ErrNoRows}}, "1"); err != nil || n != 0 {
		t.Fatal(err)
	}
	if err := r.SomarFalta(ctx, banco{}, "1", agora); err != nil {
		t.Fatal(err)
	}
	// Pedido com responsável e encaminhamento pendente.
	enc := linhaPedido(id, agora, []byte(`{}`), []byte("[]"))
	enc[25], enc[26], enc[27], enc[28], enc[29], enc[30], enc[31] = "Ana Lima", &id, "é da saúde", 5, &id, &agora, "Saúde"
	lido, err := r.BuscarPedido(ctx, banco{row: valores{enc}}, id)
	if err != nil || lido.ResponsavelNome != "Ana Lima" || lido.Encaminhamento == nil ||
		lido.Encaminhamento.UnidadeNome != "Saúde" || lido.Encaminhamento.PrazoDias != 5 || !lido.Encaminhamento.Em.Equal(agora) {
		t.Fatalf("%+v %v", lido.Encaminhamento, err)
	}
	enc[30] = (*time.Time)(nil)
	if lido, err := r.BuscarPedido(ctx, banco{row: valores{enc}}, id); err != nil || !lido.Encaminhamento.Em.IsZero() {
		t.Fatal(err)
	}
	if err := r.AtualizarPedido(ctx, banco{}, lido); err != nil {
		t.Fatal(err)
	}
	if g := encaminhamentoArgs(nil); g.para != nil || g.em != nil {
		t.Fatal(g)
	}
	if g := encaminhamentoArgs(lido.Encaminhamento); g.para == nil || *g.para != id || g.prazo != 5 {
		t.Fatal(g)
	}
	if nome, err := r.Pessoa(ctx, banco{row: valores{[]any{"Ana Lima"}}}, id); err != nil || nome != "Ana Lima" {
		t.Fatal(nome, err)
	}
	if _, err := r.Pessoa(ctx, banco{row: erroLinha{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if lista, err := r.Pessoas(ctx, banco{rows: &grade{linhas: [][]any{{id, "Ana Lima"}}}}, id); err != nil || len(lista) != 1 || lista[0].Nome != "Ana Lima" {
		t.Fatal(lista, err)
	}
	if _, err := r.Pessoas(ctx, banco{rows: dbtest.BadRows()}, id); err == nil {
		t.Fatal("pessoa")
	}
	detRuim := linhaServico(id, agora, []byte(`{}`), []byte(`{}`), []byte(`[]`), []byte(`[]`))
	detRuim[24] = []byte("{")
	if _, err := r.BuscarServico(ctx, banco{row: valores{detRuim}}, id); err == nil {
		t.Fatal("detalhes")
	}
	if lista, err := r.ListarFeriados(ctx, banco{rows: &grade{linhas: [][]any{{"2026-12-25", "Natal"}}}}); err != nil || len(lista) != 1 || lista[0].Nome != "Natal" {
		t.Fatal(lista, err)
	}
	if _, err := r.ListarFeriados(ctx, banco{rows: dbtest.BadRows()}); err == nil {
		t.Fatal("feriado")
	}
	if err := r.SalvarFeriado(ctx, banco{}, domain.Feriado{Data: "2026-12-25", Nome: "Natal"}); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoverFeriado(ctx, banco{}, "2026-12-25"); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoverFeriado(ctx, dbtest.ScanFail{}, "2026-12-25"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if lista, err := r.ComplementacoesVencidas(ctx, banco{rows: &grade{linhas: [][]any{ped}}}, agora); err != nil || len(lista) != 1 {
		t.Fatal(lista, err)
	}
	if _, err := r.PrazosVencendo(ctx, banco{rows: dbtest.BadRows()}, agora, agora); err == nil {
		t.Fatal("vencendo")
	}
	if _, err := r.PrazosVencendo(ctx, banco{rows: dbtest.ErrRows()}, agora, agora); err == nil {
		t.Fatal("linhas")
	}
	if err := r.MarcarAvisoComplemento(ctx, banco{}, id); err != nil {
		t.Fatal(err)
	}
}

func linhaServico(id uuid.UUID, agora time.Time, w, a, req, docs []byte) []any {
	return []any{id, id, id, "Alvará", "", "licenca", 10, 0, true, true, false, false, false, false, "", w, a, req, docs, agora, agora, false, false, (*uuid.UUID)(nil), []byte(`{}`), "Meio Ambiente"}
}

func linhaPedido(id uuid.UUID, agora time.Time, resp, campos []byte) []any {
	return []any{id, "SEMA202600001", id, id, id, (*uuid.UUID)(nil), "Maria", "12345678900", "", "", resp, campos,
		"novo", "", "normal", (*time.Time)(nil), "", "", "", (*uuid.UUID)(nil), (*uuid.UUID)(nil), agora, agora, "Alvará", "Meio Ambiente",
		"", (*uuid.UUID)(nil), "", 0, (*uuid.UUID)(nil), (*time.Time)(nil), "", false,
		"", (*time.Time)(nil), false, (*uuid.UUID)(nil), false}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
