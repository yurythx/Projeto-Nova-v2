// Package domain guarda as regras dos Decretos 13.653/2026 (Avança) e
// 13.654/2026 (Regulariza). Portaria não cria requisito, prazo nem benefício.
// O aviso que sai do prédio leva o número do processo e a orientação.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	ProgramaAvanca     = "avanca"
	ProgramaRegulariza = "regulariza"
	EntradaGovBR       = "govbr"
	EntradaBalcao      = "balcao"
	EntradaProcurador  = "procurador"

	SituacaoProtocolado         = "protocolado"
	SituacaoExigencia           = "em_exigencia"
	SituacaoAnalise             = "em_analise"
	SituacaoArquivado           = "arquivado"
	SituacaoAguardandoPagamento = "aguardando_pagamento"
	SituacaoParcelado           = "parcelado"
	SituacaoDecidido            = "decidido"
	SituacaoRecurso             = "recurso"

	TributoIPTU = "iptu"
	TributoTaxa = "taxa"
	TributoITBI = "itbi"
)

// Cuiaba é o fuso em que a data do recibo vale como data legal.
// Cuiabá fica em UTC−4 o ano todo.
var Cuiaba = time.FixedZone("America/Cuiaba", -4*3600)

// Recusa é uma regra do decreto que o pedido não cumpre.
type Recusa struct{ Msg string }

func (r Recusa) Error() string { return r.Msg }

// Numero é a sequência do ano, no formato NNNNNN/AAAA.
func Numero(seq, ano int) (string, error) {
	if seq < 1 || ano < 2000 {
		return "", Recusa{"o número do processo é a sequência do ano"}
	}
	return fmt.Sprintf("%06d/%d", seq, ano), nil
}

// SomarDiasUteis conta dias úteis depois da data. Sábado, domingo e a lista
// de feriados (AAAA-MM-DD) não contam. A lista vazia só pula o fim de semana:
// o feriado municipal entra quando a SEFAZ entregar o calendário.
func SomarDiasUteis(inicio time.Time, dias int, feriados map[string]struct{}) time.Time {
	atual := inicio.In(Cuiaba)
	faltam := dias
	for faltam > 0 {
		atual = atual.AddDate(0, 0, 1)
		if diaUtil(atual, feriados) {
			faltam--
		}
	}
	return atual
}

func diaUtil(t time.Time, feriados map[string]struct{}) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	_, feriado := feriados[t.In(Cuiaba).Format("2006-01-02")]
	return !feriado
}

// Ciencia é a consulta, se ela ocorrer até o 10º dia corrido, ou esse dia.
func Ciencia(enviado time.Time, consulta *time.Time) time.Time {
	limite := enviado.In(Cuiaba).AddDate(0, 0, 10)
	if consulta != nil && consulta.Before(limite) {
		return consulta.In(Cuiaba)
	}
	return limite
}

// Documento é uma peça do Anexo II. JaNaBase dispensa a tela (RC-07).
// EmitidaEm é a data da matrícula: o Anexo II pede até 30 dias.
type Documento struct {
	Tipo      string    `json:"tipo"`
	Presente  bool      `json:"presente"`
	JaNaBase  bool      `json:"ja_na_base,omitempty"`
	EmitidaEm time.Time `json:"emitida_em,omitempty"`
}

// Checklist bloqueia o envio enquanto faltar obrigatório. Tipo fora da lista
// legal não entra: portaria não cria documento.
func Checklist(programa, entrada string, itbi, moduloC, temCessao, temContrato bool, docs []Documento) error {
	obrigatorios, err := listaLegal(programa, entrada, itbi, moduloC, temCessao, temContrato)
	if err != nil {
		return err
	}
	legal := map[string]struct{}{}
	for tipo := range obrigatorios {
		legal[tipo] = struct{}{}
	}
	visto := map[string]int{}
	for _, d := range docs {
		if _, ok := legal[d.Tipo]; !ok {
			return Recusa{"o documento não está no Anexo II"}
		}
		visto[d.Tipo]++
		if visto[d.Tipo] > 1 {
			return Recusa{"cada tipo do Anexo II entra uma vez"}
		}
		if d.Presente || d.JaNaBase {
			delete(obrigatorios, d.Tipo)
		}
	}
	if len(obrigatorios) > 0 {
		return Recusa{"falta documento obrigatório do Anexo II"}
	}
	return nil
}

func listaLegal(programa, entrada string, itbi, moduloC, temCessao, temContrato bool) (map[string]struct{}, error) {
	obrigatorio := map[string]struct{}{}
	switch programa {
	case ProgramaAvanca:
		for _, t := range []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"} {
			obrigatorio[t] = struct{}{}
		}
		if itbi {
			obrigatorio["instrumento"] = struct{}{}
		}
	case ProgramaRegulariza:
		for _, t := range []string{"identificacao", "instrumento", "comprovante_endereco"} {
			obrigatorio[t] = struct{}{}
		}
		if temCessao {
			obrigatorio["cessao"] = struct{}{}
		}
		if moduloC {
			obrigatorio["prova_data"] = struct{}{}
			obrigatorio["matricula"] = struct{}{}
		}
		if !temContrato {
			obrigatorio["prova_posse"] = struct{}{}
		}
	default:
		return nil, Recusa{"o programa é avança ou regulariza"}
	}
	switch entrada {
	case EntradaGovBR, EntradaBalcao:
	case EntradaProcurador:
		obrigatorio["procuracao"] = struct{}{}
	default:
		return nil, Recusa{"a entrada é gov.br, balcão ou procurador"}
	}
	return obrigatorio, nil
}

// Exigencia é a única exigência aberta. A análise fica suspensa enquanto ela
// não for respondida. Sem resposta no prazo, o processo arquiva.
type Exigencia struct {
	Programa   string
	Aberta     time.Time
	Prazo      time.Time
	Prorrogada bool
}

// NovaExigencia abre a exigência. Outra só existe quando há fato ou documento novo.
func NovaExigencia(programa string, aberta time.Time, jaHouve, fatoNovo bool, feriados map[string]struct{}) (Exigencia, error) {
	if jaHouve && !fatoNovo {
		return Exigencia{}, Recusa{"nova exigência só por fato ou documento novo"}
	}
	e := Exigencia{Programa: programa, Aberta: aberta.In(Cuiaba)}
	switch programa {
	case ProgramaAvanca:
		e.Prazo = e.Aberta.AddDate(0, 0, 15)
	case ProgramaRegulariza:
		e.Prazo = SomarDiasUteis(e.Aberta, 5, feriados)
	default:
		return Exigencia{}, Recusa{"o programa é avança ou regulariza"}
	}
	return e, nil
}

// Prorrogar só cabe no Avança, uma vez, por mais 15 dias corridos.
func (e Exigencia) Prorrogar() (Exigencia, error) {
	if e.Programa != ProgramaAvanca {
		return e, Recusa{"o Regulariza não prorroga a exigência"}
	}
	if e.Prorrogada {
		return e, Recusa{"a exigência do Avança admite uma prorrogação"}
	}
	e.Prorrogada = true
	e.Prazo = e.Prazo.AddDate(0, 0, 15)
	return e, nil
}

// Resolver devolve o estado. Sem resposta depois do prazo, arquiva.
func (e Exigencia) Resolver(agora time.Time, respondeu bool) string {
	if agora.After(e.Prazo) && !respondeu {
		return SituacaoArquivado
	}
	if respondeu {
		return SituacaoAnalise
	}
	return SituacaoExigencia
}

// AnaliseSuspensa é o efeito da exigência aberta.
func AnaliseSuspensa(situacao string) bool { return situacao == SituacaoExigencia }

var (
	cpfNoTexto  = regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`)
	cnpjNoTexto = regexp.MustCompile(`\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}`)
)

// ModeloAviso é o texto que sai do prédio: recibo, exigência, conclusão ou alerta de prazo.
func ModeloAviso(numero, modelo string) (string, error) {
	switch modelo {
	case "recibo":
		return AvisoExterno(numero, "protocolo recebido", "acompanhe na área restrita")
	case "exigencia":
		return AvisoExterno(numero, "há exigência no processo", "responda na área restrita")
	case "conclusao":
		return AvisoExterno(numero, "o processo foi decidido", "consulte a decisão na área restrita")
	case "alerta":
		return AvisoExterno(numero, "há prazo a vencer", "acompanhe na área restrita")
	default:
		return "", Recusa{"o aviso é recibo, exigência, conclusão ou alerta de prazo"}
	}
}

// AvisoExterno leva o número, o fato e a orientação. Débito e CPF ficam na área restrita.
func AvisoExterno(numero, fato, orientacao string) (string, error) {
	if numero == "" || fato == "" || orientacao == "" {
		return "", Recusa{"o aviso leva o número do processo, o fato e a orientação"}
	}
	texto := numero + "\n" + fato + "\n" + orientacao
	baixo := strings.ToLower(texto)
	if cpfNoTexto.MatchString(texto) || cnpjNoTexto.MatchString(texto) || strings.Contains(baixo, "débito") || strings.Contains(baixo, "debito") || strings.Contains(texto, "R$") {
		return "", Recusa{"o aviso externo não leva débito nem CPF"}
	}
	return texto, nil
}

// Debito é a composição que a consulta imobiliária vai entregar.
type Debito struct {
	Ano                 int    `json:"ano"`
	Tributo             string `json:"tributo"`
	Principal           int64  `json:"principal"`
	Correcao            int64  `json:"correcao"`
	Juros               int64  `json:"juros"`
	Multa               int64  `json:"multa"`
	DoloFraudeSimulacao bool   `json:"dolo_fraude_simulacao,omitempty"`
}

// Remissao aplica a remissão integral dos juros e a anistia integral das
// multas moratórias de IPTU e taxas de 2021 a 2026. Principal e correção ficam.
func Remissao(d Debito) (devido, remitido int64, aplicada bool) {
	cheio := d.Principal + d.Correcao + d.Juros + d.Multa
	if d.DoloFraudeSimulacao || d.Ano < 2021 || d.Ano > 2026 || (d.Tributo != TributoIPTU && d.Tributo != TributoTaxa) {
		return cheio, 0, false
	}
	return d.Principal + d.Correcao, d.Juros + d.Multa, true
}

// Plano é o parcelamento calculado no domínio. O estado só vira parcelado
// depois da primeira baixa, que a Fase 4 confirma.
type Plano struct {
	Parcelas           int       `json:"parcelas"`
	Valor              int64     `json:"valor"`
	PrimeiroVencimento time.Time `json:"primeiro_vencimento"`
	Estado             string    `json:"estado"`
}

// Parcelar aceita à vista ou até 12 parcelas, cada uma de no mínimo R$ 100.
// A primeira guia vence em até 10 dias.
func Parcelar(total int64, parcelas int, hoje time.Time) (Plano, error) {
	if parcelas < 1 || parcelas > 12 {
		return Plano{}, Recusa{"o parcelamento é à vista ou em até 12 parcelas"}
	}
	if total <= 0 || total/int64(parcelas) < 10000 {
		return Plano{}, Recusa{"cada parcela é de no mínimo R$ 100"}
	}
	return Plano{
		Parcelas: parcelas, Valor: total / int64(parcelas),
		PrimeiroVencimento: hoje.In(Cuiaba).AddDate(0, 0, 10),
		Estado:             SituacaoAguardandoPagamento,
	}, nil
}

// ConfirmarPrimeiraBaixa é o único caminho para o estado parcelado.
func ConfirmarPrimeiraBaixa(p Plano, primeiraPaga bool) Plano {
	if primeiraPaga {
		p.Estado = SituacaoParcelado
		return p
	}
	p.Estado = SituacaoAguardandoPagamento
	return p
}

// SituacaoAtraso alerta aos 30 dias e rescinde acima de 60.
func SituacaoAtraso(dias int) string {
	switch {
	case dias > 60:
		return "rescisao"
	case dias >= 30:
		return "alerta"
	default:
		return "em_dia"
	}
}

// RestaurarEncargos devolve juros e multas na proporção do saldo.
func RestaurarEncargos(juros, multas, saldo, total int64) (int64, int64) {
	if total <= 0 || saldo <= 0 {
		return 0, 0
	}
	if saldo > total {
		saldo = total
	}
	return juros * saldo / total, multas * saldo / total
}

// DentroDaAdesao é até 31/12/2026, 23h59, horário de Mato Grosso.
func DentroDaAdesao(t time.Time) bool {
	limite := time.Date(2026, 12, 31, 23, 59, 59, 0, Cuiaba)
	return !t.In(Cuiaba).After(limite)
}

// DentroDoAvanca é o requerimento até 31/12/2030, 23h59, horário de Mato Grosso.
// Protocolado até essa data, o direito à análise permanece.
func DentroDoAvanca(t time.Time) bool {
	limite := time.Date(2030, 12, 31, 23, 59, 59, 0, Cuiaba)
	return !t.In(Cuiaba).After(limite)
}

// ContratoNoLimite é o instrumento do Regulariza assinado até 31/12/2025.
func ContratoNoLimite(data time.Time) bool {
	if data.IsZero() {
		return false
	}
	limite := time.Date(2025, 12, 31, 23, 59, 59, 0, Cuiaba)
	return !data.In(Cuiaba).After(limite)
}

// MatriculaRecente aceita a certidão emitida no máximo 30 dias antes do protocolo.
func MatriculaRecente(emitida, protocolo time.Time) bool {
	if emitida.IsZero() || protocolo.IsZero() {
		return false
	}
	e, p := emitida.In(Cuiaba), protocolo.In(Cuiaba)
	if p.Before(e) {
		return false
	}
	limite := time.Date(e.Year(), e.Month(), e.Day(), 23, 59, 59, 0, Cuiaba).AddDate(0, 0, 30)
	return !p.After(limite)
}

func conferirMatricula(docs []Documento, protocolo time.Time) error {
	for _, d := range docs {
		if d.Tipo != "matricula" || !d.Presente || d.JaNaBase {
			continue
		}
		if !MatriculaRecente(d.EmitidaEm, protocolo) {
			return Recusa{"a matrícula tem até 30 dias na data do protocolo"}
		}
	}
	return nil
}

// ValidarRequerimento: o módulo A é obrigatório, o B é opcional e o C só
// entra com o débito quitado ou parcelado.
func ValidarRequerimento(moduloA, moduloC, debitoAberto bool) error {
	if !moduloA {
		return Recusa{"o módulo A do requerimento é obrigatório"}
	}
	if moduloC && debitoAberto {
		return Recusa{"o módulo C só entra com o débito quitado ou parcelado"}
	}
	return nil
}

// PrazoFila são 10 dias úteis. O rito prioritário começa em 01/12/2026.
func PrazoFila(entrada time.Time, feriados map[string]struct{}) time.Time {
	return SomarDiasUteis(entrada, 10, feriados)
}

func RitoPrioritario(t time.Time) bool {
	inicio := time.Date(2026, 12, 1, 0, 0, 0, 0, Cuiaba)
	return !t.In(Cuiaba).Before(inicio)
}

// ReducaoITBI guarda 75% da alíquota quando as cinco condições se cumprem.
// A alíquota entra em centésimos de milésimo (2% = 2000; o divisor é 100000).
func ReducaoITBI(milessimos int64, condicoes int) (int64, error) {
	if condicoes < 5 || milessimos <= 0 {
		return 0, Recusa{"a redução de 75% depende das cinco condições e da alíquota ordinária"}
	}
	return milessimos * 25 / 100, nil
}

// RegistroNoPrazo é o registro do ITBI até 31/03/2027.
func RegistroNoPrazo(data time.Time) bool {
	limite := time.Date(2027, 3, 31, 23, 59, 59, 0, Cuiaba)
	return !data.In(Cuiaba).After(limite)
}

// DiferencaITBI é a cobrança da diferença quando o registro passa do prazo.
func DiferencaITBI(base, milessimosCheios int64, registro time.Time) int64 {
	if RegistroNoPrazo(registro) || base <= 0 || milessimosCheios <= 0 {
		return 0
	}
	reduzida := milessimosCheios * 25 / 100
	return base * (milessimosCheios - reduzida) / 100000
}

// PrazoDefesaRevogacao são 15 dias corridos.
func PrazoDefesaRevogacao(ciencia time.Time) time.Time {
	return ciencia.In(Cuiaba).AddDate(0, 0, 15)
}

// LinhaDUA é uma linha do Anexo IV. E-mail e telefone são opcionais e não entram aqui.
type LinhaDUA struct {
	Inscricao  string `json:"inscricao"`
	Matricula  string `json:"matricula"`
	Adquirente string `json:"adquirente"`
	Documento  string `json:"documento"`
	Endereco   string `json:"endereco"`
	Data       string `json:"data"`
	Situacao   string `json:"situacao"`
	Quitacao   string `json:"quitacao,omitempty"`
}

// FalhaDUA aponta a linha inconsistente, sem repetir o documento fiscal no motivo.
type FalhaDUA struct {
	Linha  int    `json:"linha"`
	Motivo string `json:"motivo"`
}

// ReciboDUA é a entrega. O texto do recibo não leva débito nem CPF.
// A multa fica na área restrita.
type ReciboDUA struct {
	Recibo string     `json:"recibo"`
	Falhas []FalhaDUA `json:"falhas"`
	Multa  int64      `json:"multa"`
	Linhas int        `json:"linhas"`
}

var textoDoReciboDUA = func() (string, error) {
	return AvisoExterno("DUA", "entrega recebida", "o recibo e as linhas inconsistentes ficam na área restrita")
}

// ReceberDUA valida o Anexo IV e apura a multa das linhas erradas ou omitidas.
func ReceberDUA(linhas []LinhaDUA, omitidas int) (ReciboDUA, error) {
	if len(linhas) == 0 && omitidas <= 0 {
		return ReciboDUA{}, Recusa{"a DUA precisa de ao menos uma linha ou da quantidade omitida"}
	}
	if omitidas < 0 {
		omitidas = 0
	}
	falhas := []FalhaDUA{}
	erradas := omitidas
	for i, linha := range linhas {
		if err := ValidarLinha(linha); err != nil {
			falhas = append(falhas, FalhaDUA{Linha: i + 1, Motivo: err.Error()})
			erradas++
		}
	}
	recibo, err := textoDoReciboDUA()
	if err != nil {
		return ReciboDUA{}, err
	}
	return ReciboDUA{Recibo: recibo, Falhas: falhas, Multa: MultaDUA(erradas), Linhas: len(linhas)}, nil
}

// ValidarLinha confere a linha. A situação é Quitado ou Em curso.
func ValidarLinha(l LinhaDUA) error {
	switch {
	case strings.TrimSpace(l.Inscricao) == "":
		return Recusa{"a linha da DUA precisa da inscrição"}
	case strings.TrimSpace(l.Matricula) == "":
		return Recusa{"a linha da DUA precisa da matrícula"}
	case strings.TrimSpace(l.Adquirente) == "":
		return Recusa{"a linha da DUA precisa do adquirente"}
	case strings.TrimSpace(l.Endereco) == "":
		return Recusa{"a linha da DUA precisa do endereço"}
	case !documentoFiscal(l.Documento):
		return Recusa{"a linha da DUA precisa de CPF ou CNPJ"}
	}
	if _, err := time.Parse("02/01/2006", l.Data); err != nil {
		return Recusa{"a data da DUA é DD/MM/AAAA"}
	}
	switch l.Situacao {
	case "Quitado":
		if _, err := time.Parse("02/01/2006", l.Quitacao); err != nil {
			return Recusa{"a unidade quitada informa a data da quitação"}
		}
	case "Em curso":
		if strings.TrimSpace(l.Quitacao) != "" {
			return Recusa{"a unidade em curso não traz data de quitação"}
		}
	default:
		return Recusa{"a situação da unidade é Quitado ou Em curso"}
	}
	return nil
}

func documentoFiscal(s string) bool {
	var n int
	for _, r := range s {
		if unicode.IsDigit(r) {
			n++
		}
	}
	return n == 11 || n == 14
}

// MultaDUA é R$ 200 por unidade omitida ou errada.
func MultaDUA(unidades int) int64 {
	if unidades < 0 {
		return 0
	}
	return int64(unidades) * 20000
}

// PrimeiraDUA é a entrega até 28/12/2026.
func PrimeiraDUA(entrega time.Time) bool {
	limite := time.Date(2026, 12, 28, 23, 59, 59, 0, Cuiaba)
	return !entrega.In(Cuiaba).After(limite)
}

// EntregaDUA devolve o prazo da posição. A semestral começa em 30/06/2027:
// 30/06 vence em 31/07 do mesmo ano; 31/12 vence em 31/01 do ano seguinte.
// Posição até 28/12/2026 cai na primeira entrega.
func EntregaDUA(posicao time.Time) (time.Time, error) {
	p := posicao.In(Cuiaba)
	fimDoDia := func(ano int, mes time.Month, dia int) time.Time {
		return time.Date(ano, mes, dia, 23, 59, 59, 0, Cuiaba)
	}
	switch {
	case p.Month() == time.June && p.Day() == 30 && p.Year() >= 2027:
		return fimDoDia(p.Year(), time.July, 31), nil
	case p.Month() == time.December && p.Day() == 31 && p.Year() >= 2027:
		return fimDoDia(p.Year()+1, time.January, 31), nil
	case !p.After(fimDoDia(2026, time.December, 28)):
		return fimDoDia(2026, time.December, 28), nil
	default:
		return time.Time{}, Recusa{"a posição da DUA é a primeira entrega, 30/06 ou 31/12 a partir de 2027"}
	}
}

// PrazoAdquirente são 15 dias para o adquirente responder à alteração.
func PrazoAdquirente(ciencia time.Time) time.Time {
	return ciencia.In(Cuiaba).AddDate(0, 0, 15)
}

// EfeitoNoLancamento é o exercício em que a alteração de contribuinte vale.
func EfeitoNoLancamento(ano int) bool { return ano >= 2027 }

// TitularRegistralSolidario permanece responsável com o adquirente.
func TitularRegistralSolidario() bool { return true }

// PodeComunicarAlienacao cabe à pessoa física. A pessoa jurídica entrega DUA.
func PodeComunicarAlienacao(pessoaFisica bool) error {
	if !pessoaFisica {
		return Recusa{"a comunicação de alienação é da pessoa física"}
	}
	return nil
}

// EfeitoAlienacao tira o transmitente do IPTU no exercício seguinte à validação,
// quando o preço está quitado. Os exercícios anteriores permanecem com ele.
func EfeitoAlienacao(anoValidacao int, quitado bool) (int, error) {
	if !quitado {
		return 0, Recusa{"o transmitente sai do IPTU quando o preço está quitado"}
	}
	if anoValidacao < 2000 {
		return 0, Recusa{"informe o ano da validação"}
	}
	return anoValidacao + 1, nil
}

// SinalizarExecucao aponta o débito em execução para a Procuradoria.
// A suspensão é ato da Procuradoria, não deste módulo.
func SinalizarExecucao(emExecucao bool) string {
	if emExecucao {
		return "procuradoria"
	}
	return ""
}

// InvestimentoSuficiente exige 100% do valor venal na data do protocolo.
func InvestimentoSuficiente(investido, venal int64) bool {
	return venal > 0 && investido >= venal
}

// ContaComoInvestimento aplica a lista do decreto. Compra do imóvel, tributo,
// venda, publicidade, corretagem, juros e despesa administrativa ficam de fora.
// Rubrica fora das duas listas é recusa: portaria não cria item de investimento.
func ContaComoInvestimento(rubrica string) (bool, error) {
	switch rubrica {
	case "vias", "pavimentacao", "drenagem", "agua", "esgoto", "energia", "iluminacao", "calcadas", "sinalizacao", "areas_comuns", "edificacao", "infraestrutura":
		return true, nil
	case "compra", "tributo", "venda", "publicidade", "corretagem", "juros", "administrativo":
		return false, nil
	default:
		return false, Recusa{"essa rubrica não está na lista de investimento do decreto"}
	}
}

// ITBIMatriz é 1% só no imóvel matriz, com as três condições, até 31/12/2030.
func ITBIMatriz(matriz bool, condicoes int, data time.Time) (int64, error) {
	limite := time.Date(2030, 12, 31, 23, 59, 59, 0, Cuiaba)
	switch {
	case data.In(Cuiaba).After(limite):
		return 0, Recusa{"o ITBI de 1% vale até 31/12/2030"}
	case !matriz:
		return 0, Recusa{"o ITBI de 1% é só do imóvel matriz"}
	case condicoes < 3:
		return 0, Recusa{"o ITBI de 1% depende das três condições do decreto"}
	}
	return 1000, nil
}

// IsencaoIPTU dura três exercícios e o término não se move.
type IsencaoIPTU struct {
	Inicio, Fim int
}

// NovaIsencao conta três exercícios a partir do seguinte ao despacho.
// Despacho em 2027 isenta 2028, 2029 e 2030.
func NovaIsencao(anoDespacho int) (IsencaoIPTU, error) {
	if anoDespacho < 2000 {
		return IsencaoIPTU{}, Recusa{"informe o ano do despacho"}
	}
	return IsencaoIPTU{Inicio: anoDespacho + 1, Fim: anoDespacho + 3}, nil
}

// AnotacaoIsencao é o texto que o lançamento carrega.
func AnotacaoIsencao() string {
	return "Isento – Programa AVANÇA RONDONÓPOLIS – Lei nº 15.067/2026"
}

// Heranca fica com o prazo restante. O fim original não anda.
func (i IsencaoIPTU) Heranca(exercicio int) IsencaoIPTU {
	if exercicio > i.Fim {
		return IsencaoIPTU{}
	}
	if exercicio < i.Inicio {
		exercicio = i.Inicio
	}
	return IsencaoIPTU{Inicio: exercicio, Fim: i.Fim}
}

// Individualizar é o prazo de 30 dias depois do TVEO.
func Individualizar(tveo time.Time) time.Time { return tveo.In(Cuiaba).AddDate(0, 0, 30) }

// Cronograma tem quatro marcos. O primeiro vence em 90 dias do despacho e a
// conclusão em 24 meses. Uma prorrogação de até 12 meses, pedida antes do
// vencimento, não estende o benefício.
type Cronograma struct {
	MarcoInicio      time.Time
	MarcoConclusao   time.Time
	ProrrogacaoMeses int
	BeneficioFim     time.Time
}

func NovoCronograma(despacho, beneficioFim time.Time) Cronograma {
	d := despacho.In(Cuiaba)
	return Cronograma{
		MarcoInicio:    d.AddDate(0, 0, 90),
		MarcoConclusao: d.AddDate(0, 24, 0),
		BeneficioFim:   beneficioFim.In(Cuiaba),
	}
}

func (c Cronograma) Prorrogar(meses int, pedido time.Time) (Cronograma, error) {
	if c.ProrrogacaoMeses > 0 {
		return c, Recusa{"o cronograma admite uma prorrogação"}
	}
	if meses < 1 || meses > 12 {
		return c, Recusa{"a prorrogação é de até 12 meses"}
	}
	if !pedido.In(Cuiaba).Before(c.MarcoConclusao) {
		return c, Recusa{"a prorrogação se pede antes do vencimento do marco"}
	}
	c.ProrrogacaoMeses = meses
	c.MarcoConclusao = c.MarcoConclusao.AddDate(0, meses, 0)
	return c, nil
}

// PrazoTriagem são 5 dias úteis. Parecer e decisão do Avança são 30 e 10 dias.
// A exigência aberta suspende esses prazos; quem chama confere AnaliseSuspensa.
func PrazoTriagem(inicio time.Time, feriados map[string]struct{}) time.Time {
	return SomarDiasUteis(inicio, 5, feriados)
}

func PrazoParecer(inicio time.Time) time.Time {
	return inicio.In(Cuiaba).AddDate(0, 0, 30)
}

func PrazoDecisaoAvanca(inicio time.Time) time.Time {
	return inicio.In(Cuiaba).AddDate(0, 0, 10)
}

// PrazoAssinaturaTQC são 10 dias da ciência. Sem assinatura no prazo, o despacho não produz efeito.
func PrazoAssinaturaTQC(ciencia time.Time) time.Time {
	return ciencia.In(Cuiaba).AddDate(0, 0, 10)
}

func TQCSemEfeito(assinado *time.Time, limite time.Time) bool {
	return assinado == nil || assinado.After(limite)
}

// PrazoRelatorioSemestral é 31/01 ou 31/07. O de marco vence 15 dias depois do marco.
func PrazoRelatorioSemestral(ano int, julho bool) time.Time {
	if julho {
		return time.Date(ano, time.July, 31, 23, 59, 59, 0, Cuiaba)
	}
	return time.Date(ano, time.January, 31, 23, 59, 59, 0, Cuiaba)
}

func PrazoRelatorioMarco(marco time.Time) time.Time {
	return marco.In(Cuiaba).AddDate(0, 0, 15)
}

// PrazoRelatorioAnual do Avança é 31/03 do ano seguinte ao exercício.
// O relatório final do Regulariza é 30/06/2027.
func PrazoRelatorioAnual(exercicio int) time.Time {
	return time.Date(exercicio+1, time.March, 31, 23, 59, 59, 0, Cuiaba)
}

func PrazoRelatorioFinalRegulariza() time.Time {
	return time.Date(2027, time.June, 30, 23, 59, 59, 0, Cuiaba)
}

const (
	PrazoNoPrazo  = "no prazo"
	PrazoAlerta   = "alerta"
	PrazoAtrasado = "atrasado"
)

// PrazoPainel é o que a fila da equipe mostra sobre o relatório institucional.
// O marco de cada processo continua no cronograma, quando o despacho estiver gravado.
type PrazoPainel struct {
	Nome     string    `json:"nome"`
	Limite   time.Time `json:"limite"`
	Situacao string    `json:"situacao"`
}

// PrazosDoPainel devolve o último prazo vencido e o próximo de cada relatório.
// Sem registro de entrega, o vencido fica atrasado.
func PrazosDoPainel(agora time.Time) []PrazoPainel {
	agora = agora.In(Cuiaba)
	ano := agora.Year()
	var semestrais, anuais []time.Time
	for y := ano - 1; y <= ano+1; y++ {
		semestrais = append(semestrais, PrazoRelatorioSemestral(y, false), PrazoRelatorioSemestral(y, true))
	}
	for y := ano - 2; y <= ano+1; y++ {
		anuais = append(anuais, PrazoRelatorioAnual(y))
	}
	out := serie("relatório semestral", semestrais, agora)
	out = append(out, serie("relatório anual do Avança", anuais, agora)...)
	out = append(out, serie("relatório final do Regulariza", []time.Time{PrazoRelatorioFinalRegulariza()}, agora)...)
	return out
}

// RegistrarDespacho marca a decisão do Avança. Os marcos contam a partir dela.
func (p Processo) RegistrarDespacho(agora time.Time) Processo {
	if p.Programa != ProgramaAvanca || !p.DespachoEm.IsZero() || agora.IsZero() {
		return p
	}
	p.DespachoEm = agora.In(Cuiaba)
	return p
}

// MarcosDoProcesso são o início em 90 dias, a conclusão em 24 meses e o relatório de cada um.
func MarcosDoProcesso(p Processo, agora time.Time) []PrazoPainel {
	if p.Programa != ProgramaAvanca || p.DespachoEm.IsZero() {
		return nil
	}
	c := NovoCronograma(p.DespachoEm, time.Time{})
	nome := p.Numero
	out := serie("marco de início "+nome, []time.Time{c.MarcoInicio}, agora)
	out = append(out, serie("marco de conclusão "+nome, []time.Time{c.MarcoConclusao}, agora)...)
	out = append(out, serie("relatório do marco de início "+nome, []time.Time{PrazoRelatorioMarco(c.MarcoInicio)}, agora)...)
	out = append(out, serie("relatório do marco de conclusão "+nome, []time.Time{PrazoRelatorioMarco(c.MarcoConclusao)}, agora)...)
	return out
}

// MarcoEmAlerta é o primeiro marco na janela de 30 dias que ainda não foi publicado.
func MarcoEmAlerta(p Processo, agora time.Time) (time.Time, bool) {
	for _, item := range MarcosDoProcesso(p, agora) {
		if item.Situacao == PrazoAlerta && !item.Limite.Equal(p.AlertaLimite) {
			return item.Limite, true
		}
	}
	return time.Time{}, false
}

func serie(nome string, limites []time.Time, agora time.Time) []PrazoPainel {
	var passado, futuro time.Time
	for _, limite := range limites {
		if agora.After(limite) {
			passado = limite
			continue
		}
		futuro = limite
		break
	}
	var out []PrazoPainel
	if !passado.IsZero() {
		out = append(out, PrazoPainel{Nome: nome, Limite: passado, Situacao: PrazoAtrasado})
	}
	if !futuro.IsZero() {
		situacao := PrazoNoPrazo
		if AlertarMarco(futuro, agora) {
			situacao = PrazoAlerta
		}
		out = append(out, PrazoPainel{Nome: nome, Limite: futuro, Situacao: situacao})
	}
	return out
}

// AlertarMarco é verdadeiro nos 30 dias anteriores ao marco.
func AlertarMarco(previsto, hoje time.Time) bool {
	inicio := previsto.In(Cuiaba).AddDate(0, 0, -30)
	return !hoje.In(Cuiaba).Before(inicio) && hoje.In(Cuiaba).Before(previsto.In(Cuiaba))
}

// PrazoDefesaCassacao são 20 dias corridos.
func PrazoDefesaCassacao(ciencia time.Time) time.Time {
	return ciencia.In(Cuiaba).AddDate(0, 0, 20)
}

// LancarDispensado devolve o tributo que a cassação manda lançar.
func LancarDispensado(valor int64) (int64, error) {
	if valor < 0 {
		return 0, Recusa{"o valor dispensado não é negativo"}
	}
	return valor, nil
}

// DescontoSelo é 2%, 3%, 5% ou 8% para 2, 3, 4 ou 5 critérios. O mínimo é 2.
func DescontoSelo(criterios int) (int, error) {
	switch criterios {
	case 2:
		return 2, nil
	case 3:
		return 3, nil
	case 4:
		return 5, nil
	case 5:
		return 8, nil
	default:
		return 0, Recusa{"o selo pede no mínimo dois critérios e não passa de cinco"}
	}
}

// MaisFavoravel escolhe um desconto. Os dois não se acumulam.
func MaisFavoravel(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ExerciciosDoSelo são os três lançamentos seguintes ao fim da isenção.
func ExerciciosDoSelo(fimIsencao int) (int, int) {
	return fimIsencao + 1, fimIsencao + 3
}

// AjustarSelo reduz o percentual quando um critério deixa de funcionar.
// Abaixo de dois critérios o desconto acaba.
func AjustarSelo(criterios int) (int, error) {
	if criterios < 2 {
		return 0, nil
	}
	return DescontoSelo(criterios)
}

// Reconhecer recusa benefício tácito, presumido ou automático.
func Reconhecer(pedido, decisaoExpressa bool) error {
	if !pedido || !decisaoExpressa {
		return Recusa{"o benefício exige requerimento e decisão expressa"}
	}
	return nil
}

// OficioConcedeBeneficio é sempre falso. A atualização cadastral de ofício
// não cria o benefício do programa, nem depois de 31/12/2026.
func OficioConcedeBeneficio(time.Time) bool { return false }

// ProvaDaData aceita um dos cinco meios do art. 13, com data até 31/12/2025.
// O meio "data_certa" depende de decisão fundamentada.
func ProvaDaData(tipo string, data time.Time, fundamentada bool) error {
	switch tipo {
	case "firma", "ir", "vendedor", "quitacao", "data_certa":
	default:
		return Recusa{"a prova da data é firma, declaração de IR, declaração do vendedor, quitação ou documento com data certa"}
	}
	if tipo == "data_certa" && !fundamentada {
		return Recusa{"documento com data certa depende de decisão fundamentada"}
	}
	if !ContratoNoLimite(data) {
		return Recusa{"a prova da data do contrato é de até 31/12/2025"}
	}
	return nil
}

// DivulgacaoExterna só aceita o dado agregado.
func DivulgacaoExterna(agregado bool) error {
	if !agregado {
		return Recusa{"a divulgação externa da renúncia só leva dado agregado"}
	}
	return nil
}

// PodeRecorrer não inventa etapa. O rito da Lei 1.800/1990 entra quando a SEFAZ o descrever.
func PodeRecorrer(situacao string, ritoEntregue bool) error {
	if situacao != SituacaoDecidido {
		return Recusa{"o recurso cabe na decisão"}
	}
	if !ritoEntregue {
		return Recusa{"o recurso segue a Lei 1.800/1990; as etapas entram quando a SEFAZ descrever o rito"}
	}
	return nil
}

// Simulacao não abre processo.
type Simulacao struct {
	Inscricao          string   `json:"inscricao"`
	Debitos            []Debito `json:"debitos,omitempty"`
	Parcelas           int      `json:"parcelas,omitempty"`
	BaseITBI           int64    `json:"base_itbi,omitempty"`
	AliquotaMilessimos int64    `json:"aliquota_milessimos,omitempty"`
	CondicoesITBI      int      `json:"condicoes_itbi,omitempty"`
}

// Simulado é a leitura pública: valores, parcelas e ITBI estimado.
type Simulado struct {
	Devido   int64 `json:"devido"`
	Remitido int64 `json:"remitido"`
	Plano    Plano `json:"plano"`
	ITBI     int64 `json:"itbi"`
}

func Simular(s Simulacao, hoje time.Time) (Simulado, error) {
	if strings.TrimSpace(s.Inscricao) == "" {
		return Simulado{}, Recusa{"informe a inscrição"}
	}
	var out Simulado
	for _, d := range s.Debitos {
		devido, remitido, _ := Remissao(d)
		out.Devido += devido
		out.Remitido += remitido
	}
	if s.Parcelas > 0 && out.Devido > 0 {
		plano, err := Parcelar(out.Devido, s.Parcelas, hoje)
		if err != nil {
			return Simulado{}, err
		}
		out.Plano = plano
	}
	if s.BaseITBI > 0 {
		aliq, err := ReducaoITBI(s.AliquotaMilessimos, s.CondicoesITBI)
		if err != nil {
			return Simulado{}, err
		}
		out.ITBI = s.BaseITBI * aliq / 100000
	}
	return out, nil
}

// Certidao monta o texto mínimo. O QR leva o código, não o débito.
func Certidao(numero, codigo string) (string, error) {
	if numero == "" || codigo == "" {
		return "", Recusa{"a certidão leva o número do processo e o código"}
	}
	return numero + "\n" + codigo + "\nredução de 75% da alíquota guardada no processo", nil
}

// DadosCertidao são os campos do Anexo V. A página pública de autenticidade
// continua mostrando só o código.
type DadosCertidao struct {
	Numero, Codigo, Adquirente, Transmitente, Inscricao, Matricula, Aliquota string
	DataInstrumento                                                          time.Time
}

// CertidaoAnexoV é o modelo do Anexo V, com a data-limite de registro.
func CertidaoAnexoV(d DadosCertidao) (string, error) {
	if d.Numero == "" || d.Codigo == "" || d.Adquirente == "" || d.Transmitente == "" || d.Inscricao == "" || d.Matricula == "" || d.Aliquota == "" || d.DataInstrumento.IsZero() {
		return "", Recusa{"a certidão do Anexo V leva adquirente, transmitente, inscrição, matrícula, data do instrumento e alíquota"}
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\nregistro até 31/03/2027",
		d.Numero, d.Codigo, d.Adquirente, d.Transmitente, d.Inscricao, d.Matricula,
		d.DataInstrumento.In(Cuiaba).Format("02/01/2006"), d.Aliquota), nil
}

// GuiaITBI é a guia da Fase 4. A alíquota reduzida e o número da certidão vão nela.
type GuiaITBI struct {
	Aliquota       int64
	Certidao       string
	Anotacao       string
	AlertaRegistro time.Time
}

func NovaGuiaITBI(aliquota int64, certidao string) (GuiaITBI, error) {
	if aliquota <= 0 || certidao == "" {
		return GuiaITBI{}, Recusa{"a guia de ITBI leva a alíquota reduzida e o número da certidão"}
	}
	return GuiaITBI{
		Aliquota: aliquota, Certidao: certidao,
		Anotacao:       fmt.Sprintf("Alíquota reduzida – Lei nº 15.068/2026 – Certidão nº %s – registro até 31/03/2027", certidao),
		AlertaRegistro: time.Date(2027, 3, 31, 23, 59, 59, 0, Cuiaba),
	}, nil
}

// NovaGuiaITBIAvanca anota 1% com o número da Certidão de Reconhecimento.
func NovaGuiaITBIAvanca(certidao string) (GuiaITBI, error) {
	if certidao == "" {
		return GuiaITBI{}, Recusa{"a guia de ITBI leva o número da certidão de reconhecimento"}
	}
	return GuiaITBI{
		Aliquota: 1000, Certidao: certidao,
		Anotacao:       fmt.Sprintf("Alíquota reduzida – Lei nº 15.067/2026 – Certidão de Reconhecimento nº %s", certidao),
		AlertaRegistro: time.Date(2030, 12, 31, 23, 59, 59, 0, Cuiaba),
	}, nil
}

// BaixarRetorno só chama o débito de parcelado depois da primeira parcela.
func BaixarRetorno(primeiraPaga bool) string {
	if primeiraPaga {
		return SituacaoParcelado
	}
	return SituacaoAguardandoPagamento
}

// AvisoCartorio sai pelo mesmo contrato do aviso externo.
func AvisoCartorio(numero string) (string, error) {
	return AvisoExterno(numero, "registro informado pelo cartório conveniado", "acompanhe o processo na área restrita")
}

// Abertura é o pedido que vira processo.
type Abertura struct {
	Programa     string      `json:"programa"`
	Entrada      string      `json:"entrada"`
	Titular      string      `json:"titular,omitempty"`
	Seq          int         `json:"-"`
	Ano          int         `json:"-"`
	Agora        time.Time   `json:"-"`
	ITBI         bool        `json:"itbi,omitempty"`
	ModuloC      bool        `json:"modulo_c,omitempty"`
	TemCessao    bool        `json:"tem_cessao,omitempty"`
	TemContrato  bool        `json:"tem_contrato,omitempty"`
	ModuloA      bool        `json:"modulo_a,omitempty"`
	DebitoAberto bool        `json:"debito_aberto,omitempty"`
	DataContrato time.Time   `json:"data_contrato,omitempty"`
	Documentos   []Documento `json:"documentos"`
}

// Processo é o protocolo. O recibo é a data legal, no fuso de Mato Grosso.
type Processo struct {
	ID                  uuid.UUID `json:"id"`
	Numero              string    `json:"numero"`
	Ano                 int       `json:"ano"`
	Seq                 int       `json:"seq"`
	Programa            string    `json:"programa"`
	Entrada             string    `json:"entrada"`
	Titular             string    `json:"titular"`
	Situacao            string    `json:"situacao"`
	ReciboEm            time.Time `json:"recibo_em"`
	Codigo              string    `json:"codigo"`
	Certidao            string    `json:"certidao,omitempty"`
	Aliquota            int64     `json:"aliquota,omitempty"`
	Execucao            string    `json:"execucao,omitempty"`
	Pendencia           string    `json:"pendencia,omitempty"`
	ExigenciaAberta     time.Time `json:"exigencia_aberta,omitempty"`
	ExigenciaPrazo      time.Time `json:"exigencia_prazo,omitempty"`
	ExigenciaProrrogada bool      `json:"exigencia_prorrogada,omitempty"`
	DespachoEm          time.Time `json:"despacho_em,omitempty"`
	AlertaLimite        time.Time `json:"alerta_limite,omitempty"`
	Destinatario        uuid.UUID `json:"destinatario,omitempty"`
	ConsultaEm          time.Time `json:"consulta_em,omitempty"`
	CienciaRecibo       time.Time `json:"ciencia_recibo,omitempty"`
	CienciaExigencia    time.Time `json:"ciencia_exigencia,omitempty"`
}

// CienciaDo usa a consulta deste processo só se ela ocorreu depois do envio.
func (p Processo) CienciaDo(enviado time.Time) time.Time {
	if enviado.IsZero() {
		return time.Time{}
	}
	var consulta *time.Time
	if !p.ConsultaEm.IsZero() && !p.ConsultaEm.Before(enviado) {
		c := p.ConsultaEm
		consulta = &c
	}
	return Ciencia(enviado, consulta)
}

// ComCiencia preenche as datas legais do recibo e da exigência.
func (p Processo) ComCiencia() Processo {
	p.CienciaRecibo = p.CienciaDo(p.ReciboEm)
	if !p.ExigenciaAberta.IsZero() {
		p.CienciaExigencia = p.CienciaDo(p.ExigenciaAberta)
	}
	return p
}

// Abrir protocola. No Regulariza, a adesão tem data e o requerimento tem módulo.
func Abrir(a Abertura) (Processo, error) {
	if strings.TrimSpace(a.Titular) == "" {
		return Processo{}, Recusa{"informe o titular do processo"}
	}
	if a.Programa == ProgramaRegulariza && !DentroDaAdesao(a.Agora) {
		return Processo{}, Recusa{"a adesão ao Regulariza encerra em 31/12/2026, 23h59, horário de Mato Grosso"}
	}
	if a.Programa == ProgramaAvanca && !DentroDoAvanca(a.Agora) {
		return Processo{}, Recusa{"o requerimento do Avança encerra em 31/12/2030, 23h59, horário de Mato Grosso"}
	}
	if a.Programa == ProgramaRegulariza {
		if err := ValidarRequerimento(a.ModuloA, a.ModuloC, a.DebitoAberto); err != nil {
			return Processo{}, err
		}
		if a.ModuloC && !ContratoNoLimite(a.DataContrato) {
			return Processo{}, Recusa{"o contrato do módulo C é de até 31/12/2025"}
		}
	}
	if err := Checklist(a.Programa, a.Entrada, a.ITBI, a.ModuloC, a.TemCessao, a.TemContrato, a.Documentos); err != nil {
		return Processo{}, err
	}
	if err := conferirMatricula(a.Documentos, a.Agora); err != nil {
		return Processo{}, err
	}
	numero, err := Numero(a.Seq, a.Ano)
	if err != nil {
		return Processo{}, err
	}
	recibo := a.Agora.In(Cuiaba)
	return Processo{
		ID: uuid.New(), Numero: numero, Ano: a.Ano, Seq: a.Seq,
		Programa: a.Programa, Entrada: a.Entrada, Titular: strings.TrimSpace(a.Titular),
		Situacao: SituacaoProtocolado, ReciboEm: recibo, Codigo: codigo(numero, recibo),
	}, nil
}

func codigo(numero string, recibo time.Time) string {
	return strings.ReplaceAll(numero, "/", "") + recibo.Format("20060102150405")
}

// Autenticidade é a página pública: o código confere, o débito não aparece.
type Autenticidade struct {
	Codigo   string `json:"codigo"`
	Numero   string `json:"numero"`
	Valido   bool   `json:"valido"`
	Mensagem string `json:"mensagem"`
}

func ConfirmarAutenticidade(p Processo, codigo string) Autenticidade {
	if codigo == "" || codigo != p.Codigo {
		return Autenticidade{Codigo: codigo, Valido: false, Mensagem: "código não confere"}
	}
	return Autenticidade{Codigo: codigo, Numero: p.Numero, Valido: true, Mensagem: "documento autêntico"}
}

// GuardarReducao anota a alíquota reduzida no processo. A guia que cobra é a Fase 4.
func (p Processo) GuardarReducao(milessimos int64, condicoes int) (Processo, error) {
	aliq, err := ReducaoITBI(milessimos, condicoes)
	if err != nil {
		return p, err
	}
	texto, err := Certidao(p.Numero, p.Codigo)
	if err != nil {
		return p, err
	}
	p.Aliquota, p.Certidao = aliq, texto
	return p, nil
}

// MarcarExecucao sinaliza, sem suspender.
func (p Processo) MarcarExecucao(emExecucao bool) Processo {
	p.Execucao = SinalizarExecucao(emExecucao)
	return p
}

// EmitirDecisao fecha a análise. O recurso espera o rito.
func (p Processo) EmitirDecisao() (Processo, error) {
	if p.Situacao == SituacaoExigencia || p.Situacao == SituacaoArquivado {
		return p, Recusa{"não há decisão enquanto a exigência estiver aberta ou o processo arquivado"}
	}
	p.Situacao = SituacaoDecidido
	return p, nil
}

// Recorrer registra o recurso no próprio processo quando o rito já foi entregue.
func (p Processo) Recorrer(ritoEntregue bool) (Processo, error) {
	if err := PodeRecorrer(p.Situacao, ritoEntregue); err != nil {
		return p, err
	}
	p.Situacao = SituacaoRecurso
	return p, nil
}

// AplicarExigencia suspende a análise.
func (p Processo) AplicarExigencia(e Exigencia) Processo {
	p.Situacao = e.Resolver(e.Aberta, false)
	p.ExigenciaAberta = e.Aberta
	p.ExigenciaPrazo = e.Prazo
	p.ExigenciaProrrogada = e.Prorrogada
	return p
}

// ExigenciaAbertaNoProcesso reconstrói a exigência gravada.
func (p Processo) ExigenciaAbertaNoProcesso() (Exigencia, error) {
	if p.ExigenciaPrazo.IsZero() {
		return Exigencia{}, Recusa{"não há exigência aberta neste processo"}
	}
	return Exigencia{Programa: p.Programa, Aberta: p.ExigenciaAberta, Prazo: p.ExigenciaPrazo, Prorrogada: p.ExigenciaProrrogada}, nil
}

// Pendencia é o que o painel mostra: exigência, prazo vencido ou primeira parcela.
func Pendencia(p Processo, agora time.Time) string {
	switch p.Situacao {
	case SituacaoExigencia:
		if !p.ExigenciaPrazo.IsZero() && agora.After(p.ExigenciaPrazo) {
			return "exigência vencida"
		}
		return "exigência aberta"
	case SituacaoAguardandoPagamento:
		return "aguardando a primeira parcela"
	default:
		return ""
	}
}

// ResponderExigencia volta à análise ou arquiva.
func (p Processo) ResponderExigencia(e Exigencia, agora time.Time, respondeu bool) Processo {
	p.Situacao = e.Resolver(agora, respondeu)
	if p.Situacao != SituacaoExigencia {
		p.ExigenciaAberta = time.Time{}
		p.ExigenciaPrazo = time.Time{}
		p.ExigenciaProrrogada = false
	}
	return p
}
