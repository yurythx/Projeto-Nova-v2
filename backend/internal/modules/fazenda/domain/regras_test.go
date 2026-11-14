package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPrazosDoPainel(t *testing.T) {
	novembro := PrazosDoPainel(em(2026, 11, 13, 20, 0))
	if !temPrazo(novembro, "relatório semestral", PrazoAtrasado) || !temPrazo(novembro, "relatório semestral", PrazoNoPrazo) {
		t.Fatalf("semestre: %+v", novembro)
	}
	if !temPrazo(novembro, "relatório anual do Avança", PrazoAtrasado) || !temPrazo(novembro, "relatório final do Regulariza", PrazoNoPrazo) {
		t.Fatalf("anual e final: %+v", novembro)
	}
	alerta := PrazosDoPainel(em(2027, 1, 15, 12, 0))
	if !temPrazo(alerta, "relatório semestral", PrazoAlerta) {
		t.Fatalf("alerta: %+v", alerta)
	}
	vencido := PrazosDoPainel(em(2027, 2, 1, 12, 0))
	if !temPrazo(vencido, "relatório semestral", PrazoAtrasado) {
		t.Fatalf("janeiro vencido: %+v", vencido)
	}
	final := PrazosDoPainel(em(2027, 7, 1, 12, 0))
	if !temPrazo(final, "relatório final do Regulariza", PrazoAtrasado) || temPrazo(final, "relatório final do Regulariza", PrazoNoPrazo) {
		t.Fatalf("final vencido: %+v", final)
	}
	perto := PrazosDoPainel(em(2027, 6, 15, 12, 0))
	if !temPrazo(perto, "relatório final do Regulariza", PrazoAlerta) {
		t.Fatalf("final em alerta: %+v", perto)
	}
}

func TestDespachoEMarcos(t *testing.T) {
	agora := em(2026, 6, 10, 12, 0)
	regulariza := (Processo{Programa: ProgramaRegulariza}).RegistrarDespacho(agora)
	if !regulariza.DespachoEm.IsZero() {
		t.Fatal("regulariza")
	}
	if !(Processo{Programa: ProgramaAvanca}).RegistrarDespacho(time.Time{}).DespachoEm.IsZero() {
		t.Fatal("sem data")
	}
	avanca := (Processo{Programa: ProgramaAvanca, Numero: "000001/2026"}).RegistrarDespacho(agora)
	deNovo := avanca.RegistrarDespacho(agora.AddDate(0, 0, 1))
	if !deNovo.DespachoEm.Equal(avanca.DespachoEm) {
		t.Fatal("despacho único")
	}
	if MarcosDoProcesso(Processo{Programa: ProgramaAvanca}, agora) != nil || MarcosDoProcesso(regulariza, agora) != nil {
		t.Fatal("sem despacho")
	}
	avanca.DespachoEm = em(2026, 4, 1, 12, 0)
	marcos := MarcosDoProcesso(avanca, agora)
	if !temPrazo(marcos, "marco de início 000001/2026", PrazoAlerta) {
		t.Fatalf("%+v", marcos)
	}
	limite, ok := MarcoEmAlerta(avanca, agora)
	if !ok || limite.IsZero() {
		t.Fatal("alerta")
	}
	avanca.AlertaLimite = limite
	if _, ok := MarcoEmAlerta(avanca, agora); ok {
		t.Fatal("alerta repetido")
	}
}

func temPrazo(lista []PrazoPainel, nome, situacao string) bool {
	for _, item := range lista {
		if item.Nome == nome && item.Situacao == situacao {
			return true
		}
	}
	return false
}

func em(ano int, mes time.Month, dia, hora, min int) time.Time {
	return time.Date(ano, mes, dia, hora, min, 0, 0, Cuiaba)
}

func recusa(t *testing.T, err error, trecho string) {
	t.Helper()
	var r Recusa
	if !errors.As(err, &r) || !strings.Contains(r.Msg, trecho) {
		t.Fatalf("recusa %q: %v", trecho, err)
	}
}

func docs(tipos ...string) []Documento {
	out := make([]Documento, len(tipos))
	for i, tipo := range tipos {
		out[i] = Documento{Tipo: tipo, Presente: true}
	}
	return out
}

func docsNaData(quando time.Time, tipos ...string) []Documento {
	out := docs(tipos...)
	for i := range out {
		if out[i].Tipo == "matricula" {
			out[i].EmitidaEm = quando
		}
	}
	return out
}

func TestNumeroCienciaEDiasUteis(t *testing.T) {
	if _, err := Numero(0, 2026); err == nil {
		t.Fatal("sequência")
	}
	n, err := Numero(1, 2026)
	if err != nil || n != "000001/2026" {
		t.Fatalf("número: %s %v", n, err)
	}
	sexta := em(2026, 11, 13, 10, 0)
	if got := SomarDiasUteis(sexta, 0, nil); !got.Equal(sexta) {
		t.Fatalf("zero dias: %s", got)
	}
	feriados := map[string]struct{}{"2026-11-16": {}}
	prazo := SomarDiasUteis(sexta, 1, feriados)
	if prazo.Format("2006-01-02") != "2026-11-17" {
		t.Fatalf("pula fim de semana e feriado: %s", prazo)
	}
	enviado := em(2026, 1, 1, 8, 0)
	consulta := em(2026, 1, 5, 9, 0)
	if !Ciencia(enviado, &consulta).Equal(consulta) {
		t.Fatal("ciência na consulta")
	}
	tarde := em(2026, 1, 20, 9, 0)
	if !Ciencia(enviado, &tarde).Equal(enviado.AddDate(0, 0, 10)) {
		t.Fatal("ciência no 10º dia")
	}
	if !Ciencia(enviado, nil).Equal(enviado.AddDate(0, 0, 10)) {
		t.Fatal("ciência sem consulta")
	}
	antes := Processo{ReciboEm: enviado, ConsultaEm: em(2025, 12, 1, 0, 0)}.ComCiencia()
	if !antes.CienciaRecibo.Equal(enviado.AddDate(0, 0, 10)) {
		t.Fatal("consulta anterior ao envio não antecipa a ciência")
	}
	depois := Processo{ReciboEm: enviado, ConsultaEm: consulta, ExigenciaAberta: em(2026, 1, 6, 0, 0)}.ComCiencia()
	if !depois.CienciaRecibo.Equal(consulta) || !depois.CienciaExigencia.Equal(em(2026, 1, 16, 0, 0)) {
		t.Fatal("ciência do recibo na consulta e da exigência no 10º dia")
	}
}

func TestChecklistEExigencia(t *testing.T) {
	avanca := []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"}
	if err := Checklist(ProgramaAvanca, EntradaGovBR, false, false, false, true, docs(avanca...)); err != nil {
		t.Fatal(err)
	}
	comITBI := append(append([]string{}, avanca...), "instrumento")
	if err := Checklist(ProgramaAvanca, EntradaProcurador, true, false, false, true, docs(append(comITBI, "procuracao")...)); err != nil {
		t.Fatal(err)
	}
	base := docs(avanca...)
	base[1].Presente, base[1].JaNaBase = false, true
	if err := Checklist(ProgramaAvanca, EntradaBalcao, false, false, false, true, base); err != nil {
		t.Fatal(err)
	}
	if err := Checklist(ProgramaAvanca, EntradaGovBR, false, false, false, true, append(docs(avanca...), Documento{Tipo: "extra", Presente: true})); err == nil {
		t.Fatal("fora da lista")
	}
	dup := docs(avanca...)
	dup = append(dup, dup[0])
	if err := Checklist(ProgramaAvanca, EntradaGovBR, false, false, false, true, dup); err == nil || !strings.Contains(err.Error(), "uma vez") {
		t.Fatal(err)
	}
	faltando := docs(avanca...)
	faltando[0].Presente = false
	if err := Checklist(ProgramaAvanca, EntradaGovBR, false, false, false, true, faltando); err == nil || !strings.Contains(err.Error(), "obrigatório") {
		t.Fatal(err)
	}
	if err := Checklist(ProgramaAvanca, EntradaGovBR, false, false, false, true, nil); err == nil {
		t.Fatal("faltando")
	}
	reg := []string{"identificacao", "instrumento", "comprovante_endereco", "cessao", "prova_data", "matricula", "procuracao"}
	if err := Checklist(ProgramaRegulariza, EntradaProcurador, false, true, true, true, docs(reg...)); err != nil {
		t.Fatal(err)
	}
	semContrato := []string{"identificacao", "instrumento", "comprovante_endereco", "prova_posse"}
	if err := Checklist(ProgramaRegulariza, EntradaGovBR, false, false, false, false, docs(semContrato...)); err != nil {
		t.Fatal(err)
	}
	if err := Checklist("outro", EntradaGovBR, false, false, false, true, nil); err == nil {
		t.Fatal("programa")
	}
	if err := Checklist(ProgramaAvanca, "carta", false, false, false, true, docs(avanca...)); err == nil {
		t.Fatal("entrada")
	}

	aberta := em(2026, 3, 2, 9, 0)
	e, err := NovaExigencia(ProgramaAvanca, aberta, false, false, nil)
	if err != nil || !e.Prazo.Equal(aberta.AddDate(0, 0, 15)) {
		t.Fatal(err)
	}
	e, err = e.Prorrogar()
	if err != nil || !e.Prorrogada || !e.Prazo.Equal(aberta.AddDate(0, 0, 30)) {
		t.Fatal(err)
	}
	if _, err := e.Prorrogar(); err == nil {
		t.Fatal("segunda prorrogação")
	}
	regEx, err := NovaExigencia(ProgramaRegulariza, aberta, true, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := regEx.Prorrogar(); err == nil {
		t.Fatal("regulariza prorroga")
	}
	if _, err := NovaExigencia(ProgramaAvanca, aberta, true, false, nil); err == nil {
		t.Fatal("sem fato novo")
	}
	if _, err := NovaExigencia("x", aberta, false, false, nil); err == nil {
		t.Fatal("programa da exigência")
	}
	if e.Resolver(e.Aberta, false) != SituacaoExigencia || !AnaliseSuspensa(SituacaoExigencia) || AnaliseSuspensa(SituacaoAnalise) {
		t.Fatal("análise suspensa")
	}
	if e.Resolver(e.Aberta, true) != SituacaoAnalise {
		t.Fatal("resposta")
	}
	if e.Resolver(e.Prazo.Add(time.Minute), false) != SituacaoArquivado {
		t.Fatal("arquiva")
	}
}

func TestAvisoERemissao(t *testing.T) {
	texto, err := AvisoExterno("000001/2026", "protocolo recebido", "acompanhe na área restrita")
	if err != nil || !strings.Contains(texto, "000001/2026") {
		t.Fatal(err)
	}
	if _, err := AvisoExterno("", "a", "b"); err == nil {
		t.Fatal("aviso incompleto")
	}
	for _, ruim := range []string{"529.982.247-25", "12.345.678/0001-90", "há débito", "debito em aberto", "R$ 10"} {
		if _, err := AvisoExterno("000001/2026", ruim, "acompanhe"); err == nil {
			t.Fatalf("vazou: %s", ruim)
		}
	}
	devido, remitido, ok := Remissao(Debito{Ano: 2024, Tributo: TributoIPTU, Principal: 100, Correcao: 10, Juros: 5, Multa: 7})
	if !ok || devido != 110 || remitido != 12 {
		t.Fatalf("remissão: %d %d %v", devido, remitido, ok)
	}
	if _, _, ok := Remissao(Debito{Ano: 2024, Tributo: TributoTaxa, DoloFraudeSimulacao: true}); ok {
		t.Fatal("dolo")
	}
	if _, _, ok := Remissao(Debito{Ano: 2020, Tributo: TributoIPTU}); ok {
		t.Fatal("ano")
	}
	if _, _, ok := Remissao(Debito{Ano: 2027, Tributo: TributoIPTU}); ok {
		t.Fatal("ano futuro")
	}
	if _, _, ok := Remissao(Debito{Ano: 2024, Tributo: TributoITBI, Juros: 5}); ok {
		t.Fatal("tributo")
	}
}

func TestParcelamentoFilaERequerimento(t *testing.T) {
	hoje := em(2026, 6, 1, 12, 0)
	p, err := Parcelar(120000, 12, hoje)
	if err != nil || p.Valor != 10000 || p.Estado != SituacaoAguardandoPagamento || !p.PrimeiroVencimento.Equal(hoje.AddDate(0, 0, 10)) {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := Parcelar(120000, 0, hoje); err == nil {
		t.Fatal("zero parcelas")
	}
	if _, err := Parcelar(10000, 13, hoje); err == nil {
		t.Fatal("13")
	}
	if _, err := Parcelar(0, 1, hoje); err == nil {
		t.Fatal("total")
	}
	if _, err := Parcelar(15000, 2, hoje); err == nil {
		t.Fatal("mínimo")
	}
	if ConfirmarPrimeiraBaixa(p, false).Estado != SituacaoAguardandoPagamento || ConfirmarPrimeiraBaixa(p, true).Estado != SituacaoParcelado {
		t.Fatal("baixa")
	}
	if SituacaoAtraso(0) != "em_dia" || SituacaoAtraso(30) != "alerta" || SituacaoAtraso(61) != "rescisao" {
		t.Fatal("atraso")
	}
	j, m := RestaurarEncargos(0, 0, 0, 0)
	if j != 0 || m != 0 {
		t.Fatal("sem saldo")
	}
	j, m = RestaurarEncargos(100, 50, 200, 100)
	if j != 100 || m != 50 {
		t.Fatalf("teto do saldo: %d %d", j, m)
	}
	j, m = RestaurarEncargos(100, 40, 25, 100)
	if j != 25 || m != 10 {
		t.Fatalf("proporção: %d %d", j, m)
	}
	if !DentroDaAdesao(em(2026, 12, 31, 23, 59)) || DentroDaAdesao(em(2027, 1, 1, 0, 0)) {
		t.Fatal("adesão")
	}
	if err := ValidarRequerimento(true, true, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidarRequerimento(false, false, false); err == nil {
		t.Fatal("módulo A")
	}
	if err := ValidarRequerimento(true, true, true); err == nil {
		t.Fatal("módulo C")
	}
	fila := PrazoFila(em(2026, 12, 1, 8, 0), nil)
	if fila.Before(em(2026, 12, 1, 8, 0)) || !RitoPrioritario(em(2026, 12, 1, 0, 0)) || RitoPrioritario(em(2026, 11, 30, 23, 0)) {
		t.Fatal("fila")
	}
}

func TestITBIDUAEconribuinte(t *testing.T) {
	aliq, err := ReducaoITBI(2000, 5)
	if err != nil || aliq != 500 {
		t.Fatalf("75%%: %d %v", aliq, err)
	}
	if _, err := ReducaoITBI(0, 5); err == nil {
		t.Fatal("alíquota")
	}
	if _, err := ReducaoITBI(2000, 4); err == nil {
		t.Fatal("condições")
	}
	noPrazo := em(2027, 3, 31, 23, 59)
	if !RegistroNoPrazo(noPrazo) || RegistroNoPrazo(em(2027, 4, 1, 0, 0)) {
		t.Fatal("registro")
	}
	if DiferencaITBI(100000, 2000, noPrazo) != 0 || DiferencaITBI(0, 2000, em(2027, 4, 2, 0, 0)) != 0 || DiferencaITBI(100000, 0, em(2027, 4, 2, 0, 0)) != 0 {
		t.Fatal("diferença zerada")
	}
	if DiferencaITBI(100000, 2000, em(2027, 4, 2, 0, 0)) != 1500 {
		t.Fatal("diferença")
	}
	if !PrazoDefesaRevogacao(em(2026, 1, 1, 0, 0)).Equal(em(2026, 1, 16, 0, 0)) {
		t.Fatal("revogação")
	}
	boa := LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Documento: "123.456.789-09", Endereco: "Rua 1", Data: "01/02/2024", Situacao: "Quitado", Quitacao: "02/02/2024"}
	if err := ValidarLinha(boa); err != nil {
		t.Fatal(err)
	}
	if err := ValidarLinha(LinhaDUA{Documento: "12345678901234", Data: "01/02/2024", Situacao: "Em curso"}); err == nil {
		t.Fatal("inscrição")
	}
	semMatricula := boa
	semMatricula.Matricula = ""
	if err := ValidarLinha(semMatricula); err == nil {
		t.Fatal("matrícula")
	}
	semNome := boa
	semNome.Adquirente = ""
	if err := ValidarLinha(semNome); err == nil {
		t.Fatal("adquirente")
	}
	semEndereco := boa
	semEndereco.Endereco = ""
	if err := ValidarLinha(semEndereco); err == nil {
		t.Fatal("endereço")
	}
	if err := ValidarLinha(LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Endereco: "Rua 1", Documento: "123", Data: "01/02/2024", Situacao: "Em curso"}); err == nil {
		t.Fatal("documento")
	}
	if err := ValidarLinha(LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Endereco: "Rua 1", Documento: "12345678901", Data: "32/02/2024", Situacao: "Em curso"}); err == nil {
		t.Fatal("data")
	}
	if err := ValidarLinha(LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Endereco: "Rua 1", Documento: "12345678901", Data: "01/02/2024", Situacao: "Quitado"}); err == nil {
		t.Fatal("quitação")
	}
	if err := ValidarLinha(LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Endereco: "Rua 1", Documento: "12345678901", Data: "01/02/2024", Situacao: "Em curso", Quitacao: "01/02/2024"}); err == nil {
		t.Fatal("em curso")
	}
	if err := ValidarLinha(LinhaDUA{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Endereco: "Rua 1", Documento: "12345678901", Data: "01/02/2024", Situacao: "aberto"}); err == nil {
		t.Fatal("situação")
	}
	if MultaDUA(-1) != 0 || MultaDUA(2) != 40000 {
		t.Fatal("multa")
	}
	if !PrimeiraDUA(em(2026, 12, 28, 23, 59)) || PrimeiraDUA(em(2026, 12, 29, 0, 0)) {
		t.Fatal("primeira DUA")
	}
	if !PrazoAdquirente(em(2026, 5, 1, 0, 0)).Equal(em(2026, 5, 16, 0, 0)) || !EfeitoNoLancamento(2027) || EfeitoNoLancamento(2026) || !TitularRegistralSolidario() {
		t.Fatal("contribuinte")
	}
	if err := PodeComunicarAlienacao(true); err != nil {
		t.Fatal(err)
	}
	if err := PodeComunicarAlienacao(false); err == nil {
		t.Fatal("pessoa jurídica")
	}
	if SinalizarExecucao(true) != "procuradoria" || SinalizarExecucao(false) != "" {
		t.Fatal("execução")
	}
}

func TestAvancaSeloESimulador(t *testing.T) {
	if !InvestimentoSuficiente(100, 100) || InvestimentoSuficiente(99, 100) || InvestimentoSuficiente(100, 0) {
		t.Fatal("investimento")
	}
	aliq, err := ITBIMatriz(true, 3, em(2030, 12, 31, 23, 59))
	if err != nil || aliq != 1000 {
		t.Fatal(err)
	}
	if _, err := ITBIMatriz(true, 3, em(2031, 1, 1, 0, 0)); err == nil {
		t.Fatal("prazo")
	}
	if _, err := ITBIMatriz(false, 3, em(2030, 1, 1, 0, 0)); err == nil {
		t.Fatal("matriz")
	}
	if _, err := ITBIMatriz(true, 2, em(2030, 1, 1, 0, 0)); err == nil {
		t.Fatal("condições")
	}
	isencao, err := NovaIsencao(2027)
	if err != nil || isencao.Inicio != 2028 || isencao.Fim != 2030 || AnotacaoIsencao() == "" {
		t.Fatal(err)
	}
	if _, err := NovaIsencao(0); err == nil {
		t.Fatal("exercício")
	}
	if isencao.Heranca(2031) != (IsencaoIPTU{}) || isencao.Heranca(2026).Inicio != 2028 || isencao.Heranca(2029).Fim != 2030 {
		t.Fatal("herança")
	}
	if !Individualizar(em(2027, 1, 1, 0, 0)).Equal(em(2027, 1, 31, 0, 0)) {
		t.Fatal("TVEO")
	}
	despacho := em(2027, 1, 1, 0, 0)
	fim := em(2030, 12, 31, 0, 0)
	c := NovoCronograma(despacho, fim)
	if !c.MarcoInicio.Equal(em(2027, 4, 1, 0, 0)) || !c.MarcoConclusao.Equal(despacho.AddDate(0, 24, 0)) {
		t.Fatalf("marcos: %s %s", c.MarcoInicio, c.MarcoConclusao)
	}
	c, err = c.Prorrogar(12, em(2028, 1, 1, 0, 0))
	if err != nil || !c.BeneficioFim.Equal(fim) || c.ProrrogacaoMeses != 12 {
		t.Fatal(err)
	}
	if _, err := c.Prorrogar(1, em(2028, 1, 1, 0, 0)); err == nil {
		t.Fatal("segunda")
	}
	if _, err := NovoCronograma(despacho, fim).Prorrogar(0, despacho); err == nil {
		t.Fatal("zero meses")
	}
	if _, err := NovoCronograma(despacho, fim).Prorrogar(1, despacho.AddDate(0, 24, 0)); err == nil {
		t.Fatal("pedido no vencimento")
	}
	previsto := em(2027, 6, 1, 0, 0)
	if !AlertarMarco(previsto, em(2027, 5, 2, 0, 0)) || AlertarMarco(previsto, em(2027, 4, 1, 0, 0)) || AlertarMarco(previsto, previsto) {
		t.Fatal("alerta")
	}
	if !PrazoDefesaCassacao(em(2027, 1, 1, 0, 0)).Equal(em(2027, 1, 21, 0, 0)) {
		t.Fatal("cassação")
	}
	if v, err := LancarDispensado(10); err != nil || v != 10 {
		t.Fatal(err)
	}
	if _, err := LancarDispensado(-1); err == nil {
		t.Fatal("negativo")
	}
	for criterios, quer := range map[int]int{2: 2, 3: 3, 4: 5, 5: 8} {
		got, err := DescontoSelo(criterios)
		if err != nil || got != quer {
			t.Fatalf("selo %d: %d %v", criterios, got, err)
		}
	}
	if _, err := DescontoSelo(1); err == nil {
		t.Fatal("selo mínimo")
	}
	if MaisFavoravel(8, 2) != 8 || MaisFavoravel(2, 5) != 5 {
		t.Fatal("sem cumulação")
	}
	if err := DivulgacaoExterna(true); err != nil {
		t.Fatal(err)
	}
	if err := DivulgacaoExterna(false); err == nil {
		t.Fatal("detalhe")
	}
	if err := PodeRecorrer(SituacaoAnalise, true); err == nil || PodeRecorrer(SituacaoDecidido, false) == nil {
		t.Fatal("recurso")
	}
	if err := PodeRecorrer(SituacaoDecidido, true); err != nil {
		t.Fatal(err)
	}
	hoje := em(2026, 6, 1, 0, 0)
	sim, err := Simular(Simulacao{
		Inscricao: "10", Debitos: []Debito{{Ano: 2024, Tributo: TributoIPTU, Principal: 120000, Juros: 10}},
		Parcelas: 2, BaseITBI: 100000, AliquotaMilessimos: 2000, CondicoesITBI: 5,
	}, hoje)
	if err != nil || sim.Devido != 120000 || sim.Remitido != 10 || sim.Plano.Parcelas != 2 || sim.ITBI != 500 {
		t.Fatalf("%+v %v", sim, err)
	}
	if _, err := Simular(Simulacao{}, hoje); err == nil {
		t.Fatal("inscrição")
	}
	if _, err := Simular(Simulacao{Inscricao: "1", Debitos: []Debito{{Ano: 2024, Tributo: TributoIPTU, Principal: 100}}, Parcelas: 2}, hoje); err == nil {
		t.Fatal("parcela mínima na simulação")
	}
	if _, err := Simular(Simulacao{Inscricao: "1", BaseITBI: 10, AliquotaMilessimos: 1, CondicoesITBI: 1}, hoje); err == nil {
		t.Fatal("itbi da simulação")
	}
	semParcela, err := Simular(Simulacao{Inscricao: "1"}, hoje)
	if err != nil || semParcela.Plano.Parcelas != 0 {
		t.Fatal(err)
	}
	if _, err := Certidao("", "x"); err == nil {
		t.Fatal("certidão")
	}
	if texto, err := Certidao("000001/2026", "abc"); err != nil || !strings.Contains(texto, "abc") {
		t.Fatal(err)
	}
	if _, err := NovaGuiaITBI(0, "abc"); err == nil {
		t.Fatal("guia")
	}
	guia, err := NovaGuiaITBI(500, "abc")
	if err != nil || guia.Certidao != "abc" || !RegistroNoPrazo(guia.AlertaRegistro) || !strings.Contains(guia.Anotacao, "15.068/2026") || !strings.Contains(guia.Anotacao, "Certidão nº abc") {
		t.Fatal(err)
	}
	avancaGuia, err := NovaGuiaITBIAvanca("CR-1")
	if err != nil || avancaGuia.Aliquota != 1000 || !strings.Contains(avancaGuia.Anotacao, "15.067/2026") || !strings.Contains(avancaGuia.Anotacao, "CR-1") {
		t.Fatal(err)
	}
	if _, err := NovaGuiaITBIAvanca(""); err == nil {
		t.Fatal("guia do avança")
	}
	if BaixarRetorno(false) != SituacaoAguardandoPagamento || BaixarRetorno(true) != SituacaoParcelado {
		t.Fatal("retorno")
	}
	if _, err := AvisoCartorio(""); err == nil {
		t.Fatal("cartório")
	}
	if texto, err := AvisoCartorio("000001/2026"); err != nil || strings.Contains(strings.ToLower(texto), "debito") {
		t.Fatal(err)
	}
}

func TestProcesso(t *testing.T) {
	agora := em(2026, 6, 1, 15, 4)
	avanca := []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"}
	p, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 1, Ano: 2026, Agora: agora, TemContrato: true, Documentos: docsNaData(agora, avanca...)})
	if err != nil || p.Numero != "000001/2026" || p.Situacao != SituacaoProtocolado || p.ReciboEm.Location() != Cuiaba {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := Abrir(Abertura{Titular: " "}); err == nil {
		t.Fatal("titular")
	}
	if _, err := Abrir(Abertura{Programa: ProgramaRegulariza, Entrada: EntradaGovBR, Titular: "ana", Agora: em(2027, 1, 1, 0, 0), ModuloA: true}); err == nil {
		t.Fatal("adesão encerrada")
	}
	if _, err := Abrir(Abertura{Programa: ProgramaRegulariza, Entrada: EntradaGovBR, Titular: "ana", Agora: agora}); err == nil {
		t.Fatal("módulo A")
	}
	if _, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 1, Ano: 2026, Agora: agora}); err == nil {
		t.Fatal("checklist")
	}
	if _, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 0, Ano: 2026, Agora: agora, TemContrato: true, Documentos: docsNaData(agora, avanca...)}); err == nil {
		t.Fatal("número")
	}
	reg := []string{"identificacao", "instrumento", "comprovante_endereco"}
	r, err := Abrir(Abertura{Programa: ProgramaRegulariza, Entrada: EntradaBalcao, Titular: "bia", Seq: 2, Ano: 2026, Agora: agora, ModuloA: true, TemContrato: true, Documentos: docs(reg...)})
	if err != nil || r.Entrada != EntradaBalcao {
		t.Fatal(err)
	}
	if ConfirmarAutenticidade(p, "errado").Valido || !ConfirmarAutenticidade(p, p.Codigo).Valido {
		t.Fatal("autenticidade")
	}
	if _, err := p.GuardarReducao(0, 5); err == nil {
		t.Fatal("redução")
	}
	guardado, err := p.GuardarReducao(2000, 5)
	if err != nil || guardado.Aliquota != 500 || !strings.Contains(guardado.Certidao, p.Codigo) {
		t.Fatal(err)
	}
	if _, err := (Processo{Aliquota: 1}).GuardarReducao(2000, 5); err == nil {
		t.Fatal("certidão sem número")
	}
	if p.MarcarExecucao(true).Execucao != "procuradoria" || p.MarcarExecucao(false).Execucao != "" {
		t.Fatal("execução no processo")
	}
	if _, err := (Processo{Situacao: SituacaoExigencia}).EmitirDecisao(); err == nil {
		t.Fatal("decisão na exigência")
	}
	if _, err := (Processo{Situacao: SituacaoArquivado}).EmitirDecisao(); err == nil {
		t.Fatal("decisão arquivada")
	}
	decidido, err := p.EmitirDecisao()
	if err != nil || decidido.Situacao != SituacaoDecidido {
		t.Fatal(err)
	}
	if _, err := decidido.Recorrer(false); err == nil {
		t.Fatal("rito")
	}
	recurso, err := decidido.Recorrer(true)
	if err != nil || recurso.Situacao != SituacaoRecurso {
		t.Fatal(err)
	}
	ex, err := NovaExigencia(ProgramaAvanca, agora, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	aplicada := p.AplicarExigencia(ex)
	if aplicada.Situacao != SituacaoExigencia || aplicada.ExigenciaPrazo.IsZero() {
		t.Fatal("aplicar")
	}
	if _, err := p.ExigenciaAbertaNoProcesso(); err == nil {
		t.Fatal("sem exigência")
	}
	if got, err := aplicada.ExigenciaAbertaNoProcesso(); err != nil || !got.Prazo.Equal(ex.Prazo) {
		t.Fatal(err)
	}
	if p.ResponderExigencia(ex, agora, true).Situacao != SituacaoAnalise {
		t.Fatal("responder")
	}
}

func TestRegrasDosDocumentos(t *testing.T) {
	agora := em(2026, 6, 1, 15, 4)
	if !DentroDoAvanca(em(2030, 12, 31, 23, 59)) || DentroDoAvanca(em(2031, 1, 1, 0, 0)) {
		t.Fatal("janela do avança")
	}
	if !ContratoNoLimite(em(2025, 12, 31, 23, 59)) || ContratoNoLimite(em(2026, 1, 1, 0, 0)) || ContratoNoLimite(time.Time{}) {
		t.Fatal("contrato")
	}
	if !MatriculaRecente(em(2026, 5, 2, 0, 0), agora) || MatriculaRecente(em(2026, 4, 1, 0, 0), agora) || MatriculaRecente(agora, em(2026, 5, 1, 0, 0)) || MatriculaRecente(time.Time{}, agora) {
		t.Fatal("matrícula")
	}
	avanca := []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"}
	if _, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 1, Ano: 2031, Agora: em(2031, 1, 1, 0, 0), Documentos: docsNaData(em(2031, 1, 1, 0, 0), avanca...)}); err == nil {
		t.Fatal("avança fora do prazo")
	}
	velha := docsNaData(em(2026, 1, 1, 0, 0), avanca...)
	if _, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 1, Ano: 2026, Agora: agora, Documentos: velha}); err == nil {
		t.Fatal("matrícula antiga")
	}
	naBase := docs(avanca...)
	naBase[1].Presente, naBase[1].JaNaBase = false, true
	if _, err := Abrir(Abertura{Programa: ProgramaAvanca, Entrada: EntradaGovBR, Titular: "ana", Seq: 1, Ano: 2026, Agora: agora, Documentos: naBase}); err != nil {
		t.Fatal(err)
	}
	regC := []string{"identificacao", "instrumento", "comprovante_endereco", "prova_data", "matricula"}
	pedidoC := Abertura{Programa: ProgramaRegulariza, Entrada: EntradaGovBR, Titular: "bia", Seq: 3, Ano: 2026, Agora: agora, ModuloA: true, ModuloC: true, TemContrato: true, Documentos: docsNaData(agora, regC...)}
	if _, err := Abrir(pedidoC); err == nil {
		t.Fatal("módulo C sem data")
	}
	pedidoC.DataContrato = em(2026, 1, 2, 0, 0)
	if _, err := Abrir(pedidoC); err == nil {
		t.Fatal("contrato depois de 2025")
	}
	pedidoC.DataContrato = em(2025, 12, 31, 10, 0)
	if _, err := Abrir(pedidoC); err != nil {
		t.Fatal(err)
	}
	primeira, err := EntregaDUA(em(2026, 6, 1, 0, 0))
	if err != nil || primeira.Format("2006-01-02 15:04:05") != "2026-12-28 23:59:59" {
		t.Fatalf("%v %s", err, primeira)
	}
	semestral, err := EntregaDUA(em(2027, 6, 30, 0, 0))
	if err != nil || semestral.Format("2006-01-02") != "2027-07-31" {
		t.Fatalf("%v %s", err, semestral)
	}
	virada, err := EntregaDUA(em(2027, 12, 31, 0, 0))
	if err != nil || virada.Format("2006-01-02") != "2028-01-31" {
		t.Fatalf("%v %s", err, virada)
	}
	if _, err := EntregaDUA(em(2028, 3, 1, 0, 0)); err == nil {
		t.Fatal("posição")
	}
	ano, err := EfeitoAlienacao(2027, true)
	if err != nil || ano != 2028 {
		t.Fatal(err)
	}
	if _, err := EfeitoAlienacao(2027, false); err == nil {
		t.Fatal("sem quitação")
	}
	if _, err := EfeitoAlienacao(0, true); err == nil {
		t.Fatal("ano da alienação")
	}
	conta, err := ContaComoInvestimento("drenagem")
	if err != nil || !conta {
		t.Fatal(err)
	}
	fora, err := ContaComoInvestimento("juros")
	if err != nil || fora {
		t.Fatal("juros")
	}
	if _, err := ContaComoInvestimento("portaria"); err == nil {
		t.Fatal("rubrica")
	}
	if err := Reconhecer(true, true); err != nil || Reconhecer(true, false) == nil || OficioConcedeBeneficio(agora) {
		t.Fatal("reconhecimento")
	}
	if err := ProvaDaData("firma", em(2025, 6, 1, 0, 0), false); err != nil {
		t.Fatal(err)
	}
	if err := ProvaDaData("data_certa", em(2025, 6, 1, 0, 0), false); err == nil {
		t.Fatal("data certa")
	}
	if err := ProvaDaData("ir", em(2026, 1, 1, 0, 0), false); err == nil || ProvaDaData("outro", agora, true) == nil {
		t.Fatal("prova")
	}
	if !PrazoTriagem(em(2026, 6, 1, 0, 0), nil).Equal(SomarDiasUteis(em(2026, 6, 1, 0, 0), 5, nil)) || !PrazoParecer(agora).Equal(agora.AddDate(0, 0, 30)) || !PrazoDecisaoAvanca(agora).Equal(agora.AddDate(0, 0, 10)) {
		t.Fatal("prazos do avança")
	}
	limiteTQC := PrazoAssinaturaTQC(agora)
	if !limiteTQC.Equal(agora.AddDate(0, 0, 10)) || !TQCSemEfeito(nil, limiteTQC) {
		t.Fatal("TQC")
	}
	noPrazo := agora
	if TQCSemEfeito(&noPrazo, limiteTQC) {
		t.Fatal("TQC assinado")
	}
	tarde := limiteTQC.AddDate(0, 0, 1)
	if !TQCSemEfeito(&tarde, limiteTQC) {
		t.Fatal("TQC fora do prazo")
	}
	if PrazoRelatorioSemestral(2027, false).Format("2006-01-02") != "2027-01-31" || PrazoRelatorioSemestral(2027, true).Format("2006-01-02") != "2027-07-31" {
		t.Fatal("semestral")
	}
	if !PrazoRelatorioMarco(agora).Equal(agora.AddDate(0, 0, 15)) || PrazoRelatorioAnual(2027).Format("2006-01-02") != "2028-03-31" || PrazoRelatorioFinalRegulariza().Format("2006-01-02") != "2027-06-30" {
		t.Fatal("relatórios")
	}
	texto, err := CertidaoAnexoV(DadosCertidao{Numero: "000001/2026", Codigo: "abc", Adquirente: "Ana", Transmitente: "Bia", Inscricao: "1", Matricula: "10", Aliquota: "0,5%", DataInstrumento: em(2025, 6, 1, 0, 0)})
	if err != nil || !strings.Contains(texto, "31/03/2027") || !strings.Contains(texto, "Ana") {
		t.Fatal(err)
	}
	if _, err := CertidaoAnexoV(DadosCertidao{Numero: "1"}); err == nil {
		t.Fatal("anexo V")
	}
	inicio, fim := ExerciciosDoSelo(2030)
	if inicio != 2031 || fim != 2033 {
		t.Fatal("selo")
	}
	if got, err := AjustarSelo(1); err != nil || got != 0 {
		t.Fatal(err)
	}
	if got, err := AjustarSelo(4); err != nil || got != 5 {
		t.Fatal(err)
	}
	if _, err := AjustarSelo(6); err == nil {
		t.Fatal("critério")
	}
	for _, modelo := range []string{"recibo", "exigencia", "conclusao", "alerta"} {
		texto, err := ModeloAviso("000001/2026", modelo)
		if err != nil || strings.Contains(strings.ToLower(texto), "debito") || strings.Contains(texto, "R$") {
			t.Fatalf("%s: %v %s", modelo, err, texto)
		}
	}
	if _, err := ModeloAviso("000001/2026", "outro"); err == nil {
		t.Fatal("modelo")
	}
	recibo, err := ReceberDUA([]LinhaDUA{{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Documento: "12345678901", Endereco: "Rua 1", Data: "01/02/2024", Situacao: "Em curso"}}, 1)
	if err != nil || recibo.Linhas != 1 || recibo.Multa != 20000 || len(recibo.Falhas) != 0 || strings.Contains(recibo.Recibo, "12345678901") {
		t.Fatalf("%+v %v", recibo, err)
	}
	ruim, err := ReceberDUA([]LinhaDUA{{Inscricao: "1"}}, 0)
	if err != nil || len(ruim.Falhas) != 1 || ruim.Falhas[0].Linha != 1 || ruim.Multa != 20000 {
		t.Fatalf("%+v %v", ruim, err)
	}
	if _, err := ReceberDUA(nil, 0); err == nil {
		t.Fatal("dua vazia")
	}
	if _, err := ReceberDUA(nil, -1); err == nil {
		t.Fatal("omitida negativa vira vazio")
	}
	if _, err := ReceberDUA([]LinhaDUA{{Inscricao: "1", Matricula: "10", Adquirente: "Ana", Documento: "12345678901", Endereco: "Rua 1", Data: "01/02/2024", Situacao: "Em curso"}}, -1); err != nil {
		t.Fatal(err)
	}
	anterior := textoDoReciboDUA
	textoDoReciboDUA = func() (string, error) { return "", Recusa{"aviso"} }
	t.Cleanup(func() { textoDoReciboDUA = anterior })
	if _, err := ReceberDUA(nil, 1); err == nil {
		t.Fatal("recibo")
	}
	aberto := Processo{Situacao: SituacaoExigencia, ExigenciaPrazo: em(2026, 6, 10, 0, 0)}
	if Pendencia(aberto, em(2026, 6, 1, 0, 0)) != "exigência aberta" || Pendencia(aberto, em(2026, 6, 11, 0, 0)) != "exigência vencida" {
		t.Fatal("pendência da exigência")
	}
	if Pendencia(Processo{Situacao: SituacaoAguardandoPagamento}, agora) != "aguardando a primeira parcela" || Pendencia(Processo{Situacao: SituacaoProtocolado}, agora) != "" {
		t.Fatal("pendência")
	}
}
