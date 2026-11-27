package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCamposERespostas(t *testing.T) {
	agora := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	un := uuid.New()
	campos := []Campo{
		{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true},
		{ID: "idade", Tipo: "number", Rotulo: "Idade"},
		{ID: "mail", Tipo: "email", Rotulo: "E-mail"},
		{ID: "doc", Tipo: "cpf", Rotulo: "CPF"},
		{ID: "emp", Tipo: "cnpj", Rotulo: "CNPJ"},
		{ID: "ok", Tipo: "boolean", Rotulo: "Tem anexo", Revela: []string{"arq"}},
		{ID: "arq", Tipo: "file", Rotulo: "Arquivo", Obrigatorio: true},
		{ID: "tipo", Tipo: "select", Rotulo: "Tipo", Opcoes: []string{"A", "B"}},
		{ID: "mods", Tipo: "checkbox_group", Rotulo: "Mods", Opcoes: []string{"x"}},
		{ID: "pag", Tipo: "page", Rotulo: "Página 2"},
	}
	f, err := NovoFormulario(un, "Alvará", "", "", campos, agora)
	if err != nil || !f.Ativo || f.Campos[0].ID != "nome" {
		t.Fatalf("formulário: %+v %v", f, err)
	}
	if _, err := NovoFormulario(uuid.Nil, "Alvará", "", "", nil, agora); err == nil {
		t.Fatal("sem unidade")
	}
	if _, err := NovoFormulario(un, "A", "", "", nil, agora); err == nil {
		t.Fatal("título curto")
	}
	if _, err := NovoFormulario(un, "Ok", "", "", []Campo{{ID: "a", Tipo: "text", Rotulo: " "}}, agora); err == nil {
		t.Fatal("rótulo vazio")
	}
	if err := ValidarCampos([]Campo{{ID: "a b", Tipo: "text", Rotulo: "A"}}); err == nil {
		t.Fatal("id")
	}
	if err := ValidarCampos([]Campo{{ID: "a", Tipo: "x", Rotulo: "A"}}); err == nil {
		t.Fatal("tipo")
	}
	if err := ValidarCampos([]Campo{{ID: "a", Tipo: "text", Rotulo: "A"}, {ID: "a", Tipo: "text", Rotulo: "B"}}); err == nil {
		t.Fatal("id repetido")
	}
	if err := ValidarCampos([]Campo{{ID: "a", Tipo: "select", Rotulo: "A"}}); err == nil {
		t.Fatal("sem opção")
	}
	if err := ValidarCampos([]Campo{{ID: "a", Tipo: "radio", Rotulo: "A", Opcoes: []string{" "}}}); err == nil {
		t.Fatal("opção vazia")
	}
	if err := ValidarCampos([]Campo{{ID: "a", Tipo: "text", Rotulo: "A", Revela: []string{"b"}}, {ID: "b", Tipo: "text", Rotulo: "B"}}); err == nil {
		t.Fatal("revela fora do sim/não")
	}
	dados := map[string]any{"nome": "Ana", "idade": 3.0, "mail": "a@b.co", "doc": "123.456.789-00", "emp": "11222333000181", "ok": false, "tipo": "A", "mods": []any{"x"}}
	if err := RespostasValidas(campos, dados); err != nil {
		t.Fatal(err)
	}
	if err := RespostasValidas(campos, map[string]any{"ok": true}); err == nil {
		t.Fatal("arquivo obrigatório revelado")
	}
	if err := RespostasValidas(campos, nil); err == nil {
		t.Fatal("nome obrigatório")
	}
	if err := RespostasValidas([]Campo{{ID: "n", Tipo: "number", Rotulo: "N", Obrigatorio: true}}, map[string]any{"n": "no"}); err == nil {
		t.Fatal("número")
	}
	if err := RespostasValidas([]Campo{{ID: "n", Tipo: "number", Rotulo: "N"}}, map[string]any{"n": "2"}); err != nil {
		t.Fatal(err)
	}
	if err := RespostasValidas([]Campo{{ID: "e", Tipo: "email", Rotulo: "E"}}, map[string]any{"e": "sem"}); err == nil {
		t.Fatal("e-mail")
	}
	if err := RespostasValidas([]Campo{{ID: "c", Tipo: "checkbox", Rotulo: "C"}}, map[string]any{"c": "s"}); err == nil {
		t.Fatal("caixa")
	}
	if err := RespostasValidas([]Campo{{ID: "g", Tipo: "checkbox_group", Rotulo: "G", Opcoes: []string{"x"}}}, map[string]any{"g": "x"}); err == nil {
		t.Fatal("grupo")
	}
	if err := RespostasValidas([]Campo{{ID: "g", Tipo: "checkbox_group", Rotulo: "G", Opcoes: []string{"x"}}}, map[string]any{"g": []any{"z"}}); err == nil {
		t.Fatal("opção do grupo")
	}
	if err := RespostasValidas([]Campo{{ID: "s", Tipo: "select", Rotulo: "S", Opcoes: []string{"A"}}}, map[string]any{"s": "Z"}); err == nil {
		t.Fatal("seleção")
	}
	if err := RespostasValidas([]Campo{{ID: "f", Tipo: "file", Rotulo: "F"}}, map[string]any{"f": 1}); err == nil {
		t.Fatal("arquivo")
	}
	if err := RespostasValidas([]Campo{{ID: "f", Tipo: "map", Rotulo: "M"}}, map[string]any{"f": "-16,-54"}); err != nil {
		t.Fatal(err)
	}
	if err := valorAceito(Campo{Tipo: "file"}, "  "); err == nil {
		t.Fatal("arquivo em branco")
	}
	if err := valorAceito(Campo{Tipo: "phone"}, true); err == nil {
		t.Fatal("telefone")
	}
	if err := RespostasValidas([]Campo{{ID: "z", Tipo: "outro", Rotulo: "Z"}}, map[string]any{"z": true}); err == nil {
		t.Fatal("tipo solto")
	}
	if err := RespostasValidas([]Campo{{ID: "t", Tipo: "text", Rotulo: "T"}}, map[string]any{"t": ""}); err != nil {
		t.Fatal("opcional vazio")
	}
}

func TestServicoWorkflowAgenda(t *testing.T) {
	agora := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	un, form := uuid.New(), uuid.New()
	s, err := NovoServico(Servico{UnidadeID: un, FormularioID: form, Nome: "Alvará", Categoria: "Obras"}, agora)
	if err != nil || !s.Ativo || s.Requisitos == nil {
		t.Fatalf("serviço: %+v %v", s, err)
	}
	if _, err := NovoServico(Servico{Nome: "A"}, agora); err == nil {
		t.Fatal("incompleto")
	}
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: form, Nome: "Ok", PrazoDias: -1}, agora); err == nil {
		t.Fatal("prazo")
	}
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: form, Nome: "Ok", TaxaCentavos: -1}, agora); err == nil {
		t.Fatal("taxa")
	}
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: form, Nome: "Ok", Requisitos: []Orientacao{{Titulo: "x"}}}, agora); err == nil {
		t.Fatal("requisito")
	}
	w := Workflow{Ligado: true, Etapas: []Etapa{{ID: "b", Ordem: 2, Status: StatusAndamento}, {ID: "a", Ordem: 1, Status: StatusNovo, PrazoDias: 3, ResponsavelTipo: "unidade", ResponsavelID: un.String()}}}
	if err := w.Validar(); err != nil {
		t.Fatal(err)
	}
	primeira, ok := w.Primeira()
	if !ok || primeira.ID != "a" {
		t.Fatalf("primeira: %+v", primeira)
	}
	if ord := w.Ordenadas(); len(ord) != 2 || ord[0].ID != "a" || ord[1].ID != "b" || w.Etapas[0].ID != "b" {
		t.Fatalf("ordenadas sem mexer no original: %+v / %+v", ord, w.Etapas)
	}
	if ord := (Workflow{Etapas: w.Etapas}).Ordenadas(); ord != nil {
		t.Fatalf("desligado não tem fluxo: %+v", ord)
	}
	seg, err := w.Seguinte("a")
	if err != nil || seg.ID != "b" {
		t.Fatalf("seguinte: %+v %v", seg, err)
	}
	if _, err := w.Seguinte("b"); err == nil {
		t.Fatal("última")
	}
	if _, err := (Workflow{}).Seguinte("a"); err == nil {
		t.Fatal("desligado")
	}
	if _, ok := (Workflow{}).Primeira(); ok {
		t.Fatal("sem etapa")
	}
	if e, err := w.Seguinte(""); err != nil || e.ID != "a" {
		t.Fatal(e, err)
	}
	if e, err := w.Seguinte("sumiu"); err != nil || e.ID != "a" {
		t.Fatal(e, err)
	}
	ruim := Workflow{Ligado: true}
	if err := ruim.Validar(); err == nil {
		t.Fatal("sem etapas")
	}
	if err := (Workflow{Ligado: true, Etapas: []Etapa{{ID: "a a", Status: StatusNovo}}}).Validar(); err == nil {
		t.Fatal("id da etapa")
	}
	if err := (Workflow{Ligado: true, Etapas: []Etapa{{ID: "a", Status: "nope"}}}).Validar(); err == nil {
		t.Fatal("status")
	}
	if err := (Workflow{Ligado: true, Etapas: []Etapa{{ID: "a", ResponsavelTipo: "cargo"}}}).Validar(); err == nil {
		t.Fatal("responsável")
	}
	if err := (Workflow{Ligado: true, Etapas: []Etapa{{ID: "a", ResponsavelTipo: "usuario", ResponsavelID: "x"}}}).Validar(); err == nil {
		t.Fatal("uuid")
	}
	ag := Agenda{Dias: []int{1, 2, 3, 4, 5}, Inicio: "08:00", Fim: "12:00", DuracaoMin: 30, VagasDia: 4}
	if err := ag.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := (Agenda{}).Validar(); err == nil {
		t.Fatal("agenda vazia")
	}
	if err := (Agenda{Dias: []int{1}, Inicio: "08:00", Fim: "09:00", DuracaoMin: 30, VagasDia: 1}).Validar(); err != nil {
		t.Fatal(err)
	}
	if err := (Agenda{Dias: []int{8}, Inicio: "08:00", Fim: "09:00", DuracaoMin: 30, VagasDia: 1}).Validar(); err == nil {
		t.Fatal("dia")
	}
	segunda := time.Date(2026, 11, 16, 10, 0, 0, 0, Cuiaba)
	if !ag.Cabe(segunda) {
		t.Fatal("segunda")
	}
	if ag.Cabe(time.Date(2026, 11, 15, 10, 0, 0, 0, Cuiaba)) {
		t.Fatal("domingo")
	}
	if h, ok := ag.HorarioLivre(0); !ok || h != "08:00" {
		t.Fatal(h, ok)
	}
	if h, ok := ag.HorarioLivre(1); !ok || h != "08:30" {
		t.Fatal(h, ok)
	}
	if _, ok := ag.HorarioLivre(4); ok {
		t.Fatal("lotado")
	}
	if _, ok := ag.HorarioLivre(-1); ok {
		t.Fatal("negativo")
	}
	curta := Agenda{Dias: []int{1}, Inicio: "08:00", Fim: "08:20", DuracaoMin: 30, VagasDia: 2}
	if _, ok := curta.HorarioLivre(0); ok {
		t.Fatal("não cabe")
	}
	dia, hora, err := ProcurarVaga(ag, time.Date(2026, 11, 14, 18, 0, 0, 0, Cuiaba), func(string) ([]string, error) { return nil, nil })
	if err != nil || dia != "2026-11-16" || hora != "08:00" {
		t.Fatalf("vaga: %s %s %v", dia, hora, err)
	}
	dia, hora, err = ProcurarVaga(ag, time.Date(2026, 11, 14, 18, 0, 0, 0, Cuiaba), func(string) ([]string, error) {
		return []string{"08:00"}, nil
	})
	if err != nil || dia != "2026-11-16" || hora != "08:30" {
		t.Fatalf("próxima vaga: %s %s %v", dia, hora, err)
	}
	if ag.HoraNaGrade("08:30") != true || ag.HoraNaGrade("03:00") || ag.HoraNaGrade("08:15") {
		t.Fatal(ag.Grade())
	}
	if NormalizarHora("09:00:00") != "09:00" || NormalizarHora("9h") != "" {
		t.Fatal(NormalizarHora("09:00:00"))
	}
	if _, _, err := ProcurarVaga(Agenda{}, agora, nil); err == nil {
		t.Fatal("agenda inválida")
	}
	if _, _, err := ProcurarVaga(ag, agora, func(string) ([]string, error) { return nil, errors.New("falha") }); err == nil {
		t.Fatal("contagem")
	}
	if _, _, err := ProcurarVaga(ag, agora, func(string) ([]string, error) {
		return []string{"08:00", "08:30", "09:00", "09:30"}, nil
	}); !errors.Is(err, ErrSemVaga) {
		t.Fatal(err)
	}
	st, err := StatusDoHorario("2026-11-16", "2026-11-16")
	if err != nil || st != "confirmado" {
		t.Fatal(st, err)
	}
	st, err = StatusDoHorario("2026-11-17", "2026-11-16")
	if err != nil || st != "agendado" {
		t.Fatal(st, err)
	}
	if _, err := StatusDoHorario("ontem", "2026-11-16"); err == nil {
		t.Fatal("data")
	}
	if _, err := StatusDoHorario("2026-11-15", "2026-11-16"); err == nil {
		t.Fatal("passado")
	}
	ultima := agora
	if !BloqueioPorFaltas(2, ultima, ultima.Add(29*24*time.Hour)) {
		t.Fatal("bloqueio")
	}
	if BloqueioPorFaltas(1, ultima, agora) || BloqueioPorFaltas(2, time.Time{}, agora) || BloqueioPorFaltas(2, ultima, ultima.Add(31*24*time.Hour)) {
		t.Fatal("sem bloqueio")
	}
}

func TestNumeroPedidoEAviso(t *testing.T) {
	if Iniciais("sema", "") != "SEMA" || Iniciais("SECRETARIA", "") != "SECR" {
		t.Fatal(Iniciais("sema", ""), Iniciais("SECRETARIA", ""))
	}
	if Iniciais("", "Secretaria de Meio Ambiente") != "SDMA" || Iniciais("", "123") != "PROT" {
		t.Fatal(Iniciais("", "Secretaria de Meio Ambiente"), Iniciais("", "123"))
	}
	n, err := MontarNumero("SEMA", 2026, 12)
	if err != nil || n != "SEMA202600012" {
		t.Fatal(n, err)
	}
	if _, err := MontarNumero("", 2026, 1); err == nil {
		t.Fatal("sigla")
	}
	if _, err := MontarNumero("A", 1999, 1); err == nil {
		t.Fatal("ano")
	}
	if _, err := MontarNumero("A", 2026, -1); err == nil {
		t.Fatal("sufixo negativo")
	}
	if _, err := MontarNumero("A", 2026, 100000); err == nil {
		t.Fatal("sufixo")
	}
	r, err := NovoResponsavel("Ana Lima", "123.456.789-00")
	if err != nil || r.CPF != "12345678900" || !r.Ativo {
		t.Fatal(r, err)
	}
	if _, err := NovoResponsavel("A", "123"); err == nil {
		t.Fatal("responsável")
	}
	if Prioridade("alta") != "alta" || Prioridade("x") != "normal" {
		t.Fatal(Prioridade("x"))
	}
	agora := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	un := uuid.New()
	f, err := NovoFormulario(un, "Alvará", "", "", []Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}, {ID: "anon", Tipo: "boolean", Rotulo: "Anônimo"}}, agora)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NovoServico(Servico{UnidadeID: un, FormularioID: f.ID, Nome: "Alvará", CampoAnonimo: "anon"}, agora)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Abrir(s, f, "SEMA202600001", "Maria Souza", "123.456.789-00", "a@b.co", "", "urgente", map[string]any{"nome": "Maria", "anon": true}, nil, agora)
	if err != nil || p.Status != StatusNovo || p.CPF != "12345678900" || p.Prioridade != "urgente" {
		t.Fatalf("%+v %v", p, err)
	}
	s.AprovaSozinho = true
	ap, err := Abrir(s, f, "SEMA202600002", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, nil, agora)
	if err != nil || ap.Status != StatusAprovado {
		t.Fatal(ap.Status, err)
	}
	usr := uuid.New()
	s.AprovaSozinho = false
	s.Workflow = Workflow{Ligado: true, Etapas: []Etapa{{ID: "ana", Ordem: 1, Status: StatusAnalise, PrazoDias: 2, ResponsavelTipo: "usuario", ResponsavelID: usr.String()}}}
	com, err := Abrir(s, f, "SEMA202600003", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, &usr, agora)
	if err != nil || com.Status != StatusAnalise || com.ResponsavelID == nil || *com.ResponsavelID != usr || com.PrazoEm == nil {
		t.Fatalf("%+v %v", com, err)
	}
	dep := uuid.New()
	s.Workflow.Etapas = []Etapa{{ID: "dep", Ordem: 1, ResponsavelTipo: "departamento", ResponsavelID: dep.String()}}
	d, err := Abrir(s, f, "SEMA202600004", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, nil, agora)
	if err != nil || d.DepartamentoID == nil || *d.DepartamentoID != dep || d.Status != StatusNovo {
		t.Fatal(d, err)
	}
	s.Ativo = false
	if _, err := Abrir(s, f, "X", "Maria", "12345678900", "", "", "", nil, nil, agora); err == nil {
		t.Fatal("inativo")
	}
	s.Ativo = true
	if _, err := Abrir(s, f, "", "M", "1", "", "", "", nil, nil, agora); err == nil {
		t.Fatal("identidade")
	}
	s.Workflow = Workflow{Ligado: true, Etapas: []Etapa{{ID: "ruim", Ordem: 1, Status: "nope"}}}
	if _, err := Abrir(s, f, "N", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, nil, agora); err == nil {
		t.Fatal("etapa inválida")
	}
	s.Workflow = Workflow{Ligado: true, Etapas: []Etapa{{ID: "u", Ordem: 1, ResponsavelTipo: "outro", ResponsavelID: un.String()}}}
	if _, err := Abrir(s, f, "N", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, nil, agora); err == nil {
		t.Fatal("tipo de responsável")
	}

	ped := p
	comp, err := ped.Complementar("envie o comprovante", agora)
	if err != nil || comp.Status != StatusComplementacao {
		t.Fatal(comp, err)
	}
	if _, err := comp.Complementar("de novo", agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := comp.Avancar(Workflow{Ligado: true, Etapas: []Etapa{{ID: "a", Ordem: 1}}}, agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	resp, err := comp.ResponderComplemento(agora)
	if err != nil || resp.Status != StatusAndamento || resp.Complemento != "" {
		t.Fatal(resp, err)
	}
	if _, err := ped.ResponderComplemento(agora); err == nil {
		t.Fatal("sem complemento")
	}
	if _, err := ped.Complementar("  ", agora); err == nil {
		t.Fatal("texto")
	}
	wf := Workflow{Ligado: true, Etapas: []Etapa{{ID: "r", Ordem: 1, Status: StatusAndamento}, {ID: "e", Ordem: 2, Status: StatusAnalise, ResponsavelTipo: "unidade", ResponsavelID: un.String()}}}
	ped.EtapaID = "r"
	avan, err := ped.Avancar(wf, agora)
	if err != nil || avan.EtapaID != "e" || avan.UnidadeID != un || avan.Status != StatusAnalise {
		t.Fatalf("%+v %v", avan, err)
	}
	fim, err := avan.Concluir(DecisaoFavoravel, "deferido", agora)
	if err != nil || fim.Status != StatusConcluido {
		t.Fatal(fim, err)
	}
	if _, err := fim.Cancelar(agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := fim.Complementar("x", agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := ped.Concluir("talvez", "x", agora); err == nil {
		t.Fatal("decisão")
	}
	if _, err := ped.Concluir(DecisaoDesfavoravel, "", agora); err == nil {
		t.Fatal("justificativa")
	}
	canc, err := ped.Cancelar(agora)
	if err != nil || canc.Status != StatusCancelado {
		t.Fatal(canc, err)
	}
	consulta := Publico(p, "anon", nil, nil)
	if consulta.Nome != "Denúncia anônima" || consulta.Historico == nil || len(consulta.Historico) != 0 {
		t.Fatal(consulta)
	}
	aberta := comp
	c2 := Publico(aberta, "", []Historico{{Acao: "complemento"}}, &Agendamento{Hora: "08:00"})
	if c2.Nome != p.Nome || c2.Complemento == "" || c2.Agendamento == nil {
		t.Fatal(c2)
	}
	if _, err := NovoFormulario(un, "Ok", " d ", " i ", nil, agora); err != nil {
		t.Fatal(err)
	}
	if !vazio(nil) || !vazio([]any{}) || vazio(false) {
		t.Fatal("vazio")
	}
	if err := RespostasValidas([]Campo{{ID: "c", Tipo: "cpf", Rotulo: "C"}}, map[string]any{"c": "123"}); err == nil {
		t.Fatal("cpf curto")
	}
	if err := RespostasValidas([]Campo{{ID: "c", Tipo: "cnpj", Rotulo: "C"}}, map[string]any{"c": true}); err == nil {
		t.Fatal("cnpj")
	}
	if _, ok := numero(true); ok {
		t.Fatal("número solto")
	}
	longo := strings.Repeat("a", 81)
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: f.ID, Nome: "Ok", Categoria: longo}, agora); err == nil {
		t.Fatal("categoria")
	}
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: f.ID, Nome: "Ok", Workflow: Workflow{Ligado: true}}, agora); err == nil {
		t.Fatal("workflow")
	}
	if _, err := NovoServico(Servico{UnidadeID: un, FormularioID: f.ID, Nome: "Ok", ExigeAgendamento: true}, agora); err == nil {
		t.Fatal("agenda obrigatória")
	}
	if err := (Agenda{Inicio: "08:00", Fim: "09:00", DuracaoMin: 30, VagasDia: 1}).Validar(); err == nil {
		t.Fatal("sem dias")
	}
	if Iniciais("", "Um Dois Tres Quatro Cinco") != "UDTQ" {
		t.Fatal(Iniciais("", "Um Dois Tres Quatro Cinco"))
	}
	if _, err := Abrir(s, f, "N", "Maria Souza", "12345678900", "", "", "", map[string]any{}, nil, agora); err == nil {
		t.Fatal("resposta faltando")
	}
	livre := f
	livre.Campos = nil
	sLivre := s
	sLivre.Ativo = true
	sLivre.Workflow = Workflow{}
	sLivre.AprovaSozinho = false
	sLivre.FormularioID = livre.ID
	if aberto, err := Abrir(sLivre, livre, "N1", "Maria Souza", "12345678900", "", "", "", nil, nil, agora); err != nil || aberto.Respostas == nil {
		t.Fatal(aberto, err)
	}
	for _, tipo := range []string{"unidade", "departamento", "usuario"} {
		s.Workflow = Workflow{Ligado: true, Etapas: []Etapa{{ID: "z", Ordem: 1, ResponsavelTipo: tipo, ResponsavelID: "nao-e-uuid"}}}
		if _, err := Abrir(s, f, "N", "Maria Souza", "12345678900", "", "", "", map[string]any{"nome": "Maria"}, nil, agora); err == nil {
			t.Fatal(tipo)
		}
	}
	if _, err := fim.ResponderComplemento(agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := fim.Avancar(wf, agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := fim.Concluir(DecisaoFavoravel, "de novo", agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := comp.Concluir(DecisaoFavoravel, "cedo", agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := ped.Avancar(Workflow{}, agora); err == nil {
		t.Fatal("sem workflow")
	}
	txt, err := Aviso(p.Numero, "protocolo recebido", "acompanhe pelo número")
	if err != nil || txt == "" {
		t.Fatal(txt, err)
	}
	if _, err := Aviso("", "a", "b"); err == nil {
		t.Fatal("aviso")
	}
}

func TestDiaUtilMapaEConsulta(t *testing.T) {
	sexta := time.Date(2026, 11, 13, 15, 0, 0, 0, Cuiaba)
	segunda := SomarDiasUteis(sexta, 1)
	if segunda.Weekday() != time.Monday || segunda.Format("2006-01-02") != "2026-11-16" {
		t.Fatal(segunda)
	}
	if SomarDiasUteis(sexta, 0).Format("2006-01-02") != "2026-11-13" {
		t.Fatal("zero")
	}
	prazo := SomarDiasUteis(sexta, 1)
	if SituacaoPrazo(StatusNovo, sexta, &prazo, sexta) != PrazoNoPrazo {
		t.Fatal("no prazo")
	}
	if SituacaoPrazo(StatusNovo, sexta, &prazo, prazo) != PrazoAtencao {
		t.Fatal("atenção")
	}
	terca := time.Date(2026, 11, 17, 9, 0, 0, 0, Cuiaba)
	if SituacaoPrazo(StatusNovo, sexta, &prazo, terca) != PrazoAtrasado {
		t.Fatal("atrasado")
	}
	if SituacaoPrazo(StatusConcluido, sexta, &prazo, terca) != PrazoFinalizado {
		t.Fatal("finalizado")
	}
	if SituacaoPrazo(StatusNovo, sexta, nil, sexta) != PrazoNoPrazo {
		t.Fatal("sem prazo")
	}
	if !Coordenada("-16.47, -54.63") || Coordenada("91,0") || Coordenada("ponto") {
		t.Fatal("coordenada")
	}
	if err := RespostasValidas(
		[]Campo{{ID: "p", Tipo: "map", Rotulo: "P", Obrigatorio: true}, {ID: "a", Tipo: "map_area", Rotulo: "A", Obrigatorio: true}},
		map[string]any{"p": "-16.47,-54.63", "a": "-16.47,-54.63;-16.48,-54.64"},
	); err != nil {
		t.Fatal(err)
	}
	if err := RespostasValidas([]Campo{{ID: "p", Tipo: "ambulante", Rotulo: "P", Obrigatorio: true}}, map[string]any{"p": "rua 1"}); err == nil {
		t.Fatal("ambulante texto")
	}
	un := uuid.New()
	f, err := NovoFormulario(un, "Form", "", "", nil, sexta)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NovoServico(Servico{UnidadeID: un, FormularioID: f.ID, Nome: "Alvará", PrazoDias: 1, Ativo: true}, sexta)
	if err != nil {
		t.Fatal(err)
	}
	aberto, err := Abrir(s, f, "N1", "Maria Souza", "12345678900", "", "", "", nil, nil, sexta)
	if err != nil || aberto.PrazoEm == nil || aberto.PrazoEm.In(Cuiaba).Format("2006-01-02") != "2026-11-16" {
		t.Fatal(aberto.PrazoEm, err)
	}
	consulta := Publico(Pedido{Numero: "N", Status: StatusComplementacao, Nome: "Maria"}, "", []Historico{
		{Descricao: "falta", Anexos: []string{"protocolos/complemento/x/a.pdf"}, Status: StatusComplementacao},
		{Acao: AcaoNota, Descricao: "só a mesa vê"},
		{Acao: AcaoEdicao, Descricao: "corrigi o endereço interno"},
	}, nil)
	if consulta.StatusRotulo != "Em complementação" || len(consulta.Historico) != 2 || consulta.Historico[0].Anexos != nil || consulta.Historico[1].Descricao != "pedido atualizado" {
		t.Fatal(consulta)
	}
	if _, err := TextoInterno("  ", 20); err == nil {
		t.Fatal("texto vazio")
	}
	editado, err := aberto.Editar(map[string]any{}, sexta)
	if err != nil {
		t.Fatal(err)
	}
	editado.Status = StatusConcluido
	if _, err := editado.Editar(map[string]any{}, sexta); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	for status, rotulo := range map[string]string{
		StatusNovo: "Novo", StatusAndamento: "Em andamento", StatusPendente: "Pendente",
		StatusAnalise: "Em análise", StatusAguardando: "Aguardando aprovação", StatusAprovado: "Aprovado",
		StatusRejeitado: "Rejeitado", StatusConcluido: "Concluído", StatusCancelado: "Cancelado",
		"desconhecido": "desconhecido",
	} {
		if RotuloStatus(status) != rotulo {
			t.Fatalf("%s: %s", status, RotuloStatus(status))
		}
	}
	if err := RespostasValidas([]Campo{{ID: "a", Tipo: "map_area", Rotulo: "Área", Obrigatorio: true}}, map[string]any{"a": "1,2"}); err == nil {
		t.Fatal("área")
	}
	if NormalizarHora("08:00:00") != "08:00" || (Agenda{Inicio: "08:00", Fim: "08:00", DuracaoMin: 30, VagasDia: 3}).Grade() != nil && len((Agenda{Inicio: "18:00", Fim: "08:00", DuracaoMin: 30, VagasDia: 3}).Grade()) != 0 {
		t.Fatal("grade")
	}
	if _, err := aberto.Editar(nil, sexta); err != nil {
		t.Fatal(err)
	}
	comCampo := aberto
	comCampo.Campos = []Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}}
	if _, err := comCampo.Editar(map[string]any{"nome": "  "}, sexta); err == nil {
		t.Fatal("resposta")
	}
	if diasUteisFechado(sexta, sexta.AddDate(0, 0, -1), nil) != 0 {
		t.Fatal("intervalo invertido")
	}
	antes := sexta.AddDate(0, 0, -3)
	if SituacaoPrazo(StatusNovo, sexta, &antes, antes) != PrazoNoPrazo {
		t.Fatal("prazo anterior")
	}
	if Coordenada("abc,1") {
		t.Fatal("coordenada inválida")
	}
	if CPFMascarado("123.456.789-00") != "***.456.789-00" || CPFMascarado("123") != "" {
		t.Fatal(CPFMascarado("123.456.789-00"))
	}
}
