package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type numeroResp struct {
	ID       string   `json:"id"`
	Numero   int64    `json:"numero"`
	Estado   string   `json:"estado"`
	Proximos []string `json:"proximos"`
}

// Fase 6: ativos sustentam produtos; incidentes repetidos sugerem um
// problema; o problema vira erro conhecido (com contorno) e é resolvido
// (com causa raiz). A equipe de atendimento consulta os erros conhecidos.
func TestAtivosEProblemas(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	gestorID, gestor := h.pessoaEm(t, o.a, "ativos:manage", "problemas:manage", "produtos:manage", "atendimento:manage", "atendimento:atender")
	_, leitor := h.pessoaEm(t, o.a, "problemas:read")
	_, gestorB := h.pessoaEm(t, o.b, "ativos:manage", "problemas:manage")
	_, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	_ = gestorID

	produto := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", gestor,
		`{"nome":"Rede da escola `+uuid.NewString()[:6]+`","estagio":"operacao","unidade_id":"`+o.a+`"}`))
	deB := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", h.admin(),
		`{"nome":"Rede de B","estagio":"operacao","unidade_id":"`+o.b+`"}`))

	// Ativos.
	serie := "PAT-" + uuid.NewString()[:8]
	ativo := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/ativos", gestor,
		`{"nome":"Switch do 2º andar","tipo":"rede","estado":"em_uso","identificador":"`+serie+`","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`))
	h.expect(http.StatusConflict, http.MethodPost, "/api/v1/ativos", gestor,
		`{"nome":"Outro","tipo":"rede","estado":"em_uso","identificador":"`+serie+`","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/ativos", gestor,
		`{"nome":"Errado","tipo":"rede","estado":"em_uso","unidade_id":"`+o.a+`","produto_id":"`+deB.ID+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/ativos", gestor, `{"nome":"x","tipo":"nave","estado":"em_uso"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/ativos", gestor, `{"nome":"Institucional","tipo":"rede","estado":"em_uso"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/ativos", gestor, `{`)
	lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/ativos?produto_id="+produto.ID, gestor, "").Body.String()
	if !strings.Contains(lista, ativo.ID) {
		t.Fatalf("ativos do produto: %s", lista)
	}
	if outra := h.expect(http.StatusOK, http.MethodGet, "/api/v1/ativos", gestorB, "").Body.String(); strings.Contains(outra, ativo.ID) {
		t.Fatal("B não vê os ativos de A")
	}
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/ativos?produto_id=x", gestor, "")
	h.expect(http.StatusOK, http.MethodGet, "/api/v1/ativos/"+ativo.ID, gestor, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/ativos/"+ativo.ID, gestorB, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/ativos/x", gestor, "")
	edicao := `{"nome":"Switch do 2º andar","tipo":"rede","estado":"manutencao","identificador":"` + serie + `","unidade_id":"` + o.a + `","produto_id":"` + produto.ID + `"}`
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/ativos/"+ativo.ID, gestor, edicao)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/ativos/"+ativo.ID, gestor, edicao)
	if outboxCount(t, h, "ativos.ativo.atualizado", ativo.ID) != 1 {
		t.Fatal("edição vazia não emite evento")
	}
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/ativos/"+ativo.ID, gestor, `{"nome":"x","tipo":"rede","estado":"em_uso","unidade_id":"`+o.b+`"}`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/ativos/"+uuid.NewString(), gestor, edicao)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/ativos/x", gestor, edicao)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/ativos/"+ativo.ID, gestor, `{`)

	// Três incidentes da mesma oferta: sugestão de problema.
	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Sem rede `+uuid.NewString()[:6]+`","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	pedido := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Pedir cabo","tipo":"requisicao","unidade_id":"`+o.a+`"}`))
	var incidentes []string
	for i := range 3 {
		s := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
			fmt.Sprintf(`{"oferta_id":"%s","titulo":"Caiu de novo %d"}`, oferta.ID, i)))
		incidentes = append(incidentes, s.ID)
	}
	requisicao := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+pedido.ID+`","titulo":"Um cabo"}`))
	sugestoes := h.expect(http.StatusOK, http.MethodGet, "/api/v1/problemas/sugestoes", gestor, "").Body.String()
	if !strings.Contains(sugestoes, oferta.ID) || !strings.Contains(sugestoes, `"total":3`) {
		t.Fatalf("sugestão de problema: %s", sugestoes)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/problemas/sugestoes", leitor, "")

	// Problema.
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/problemas", gestor, `{"titulo":" ","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/problemas", gestor, `{"titulo":"De B","unidade_id":"`+o.b+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/problemas", gestor,
		`{"titulo":"x","unidade_id":"`+o.a+`","produto_id":"`+deB.ID+`"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/problemas", gestor, `{`)
	prob := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/problemas", gestor,
		`{"titulo":"Quedas de rede no 2º andar","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`))
	if prob.Numero <= 0 || prob.Estado != "registrado" {
		t.Fatalf("problema: %+v", prob)
	}
	pbase := "/api/v1/problemas/" + prob.ID
	h.expect(http.StatusOK, http.MethodPut, pbase+"/incidentes/"+incidentes[0], gestor, "")
	h.expect(http.StatusOK, http.MethodPut, pbase+"/incidentes/"+incidentes[0], gestor, "")
	if outboxCount(t, h, "problemas.problema.vinculo_alterado", prob.ID) != 1 {
		t.Fatal("ligar de novo não emite evento")
	}
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, pbase+"/incidentes/"+requisicao.ID, gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, pbase+"/incidentes/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusOK, http.MethodPut, pbase+"/ativos/"+ativo.ID, gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, pbase+"/ativos/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, pbase+"/ativos/x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/problemas/x/ativos/"+ativo.ID, gestor, "")
	if s := h.expect(http.StatusOK, http.MethodGet, "/api/v1/problemas/sugestoes", gestor, "").Body.String(); strings.Contains(s, oferta.ID) {
		t.Fatalf("com um incidente já no problema, sobram dois: sem sugestão: %s", s)
	}
	detalhe := h.expect(http.StatusOK, http.MethodGet, pbase, gestor, "").Body.String()
	if !strings.Contains(detalhe, "Caiu de novo 0") || !strings.Contains(detalhe, "Switch do 2º andar") || !strings.Contains(detalhe, `"proximos":["em_analise"]`) {
		t.Fatalf("detalhe com vínculos: %s", detalhe)
	}
	doIncidente := h.expect(http.StatusOK, http.MethodGet, "/api/v1/problemas?solicitacao_id="+incidentes[0], leitor, "").Body.String()
	if !strings.Contains(doIncidente, prob.ID) {
		t.Fatalf("o incidente aponta para o problema: %s", doIncidente)
	}

	estado := func(want int, para string) {
		t.Helper()
		h.expect(want, http.MethodPost, pbase+"/estado", gestor, `{"estado":"`+para+`"}`)
	}
	estado(http.StatusConflict, "resolvido")
	estado(http.StatusOK, "em_analise")
	estado(http.StatusOK, "em_analise")
	estado(http.StatusUnprocessableEntity, "erro_conhecido")
	h.expect(http.StatusOK, http.MethodPut, pbase, gestor,
		`{"titulo":"Quedas de rede no 2º andar","contorno":"Reiniciar o switch","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`)
	estado(http.StatusOK, "erro_conhecido")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, pbase, gestor, `{"titulo":"Sem contorno","unidade_id":"`+o.a+`"}`)
	kedb := h.expect(http.StatusOK, http.MethodGet, "/api/v1/problemas?estado=erro_conhecido&q=switch", leitor, "").Body.String()
	if !strings.Contains(kedb, prob.ID) {
		t.Fatalf("a base de erros conhecidos: %s", kedb)
	}
	h.expect(http.StatusForbidden, http.MethodPost, pbase+"/estado", leitor, `{"estado":"resolvido"}`)
	if d := h.expect(http.StatusOK, http.MethodGet, pbase, leitor, "").Body.String(); !strings.Contains(d, `"proximos":[]`) {
		t.Fatalf("quem só consulta não vê passagens: %s", d)
	}
	h.expect(http.StatusNotFound, http.MethodGet, pbase, gestorB, "")
	estado(http.StatusUnprocessableEntity, "resolvido")
	h.expect(http.StatusOK, http.MethodPut, pbase, gestor,
		`{"titulo":"Quedas de rede no 2º andar","contorno":"Reiniciar o switch","causa_raiz":"Fonte com defeito","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`)
	estado(http.StatusOK, "resolvido")
	estado(http.StatusOK, "encerrado")
	h.expect(http.StatusConflict, http.MethodPut, pbase, gestor, `{"titulo":"x","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusConflict, http.MethodDelete, pbase+"/incidentes/"+incidentes[0], gestor, "")
	h.expect(http.StatusConflict, http.MethodDelete, pbase+"/ativos/"+ativo.ID, gestor, "")
	estado(http.StatusConflict, "em_analise")

	// Desvincular num problema aberto; tudo inválido de forma.
	outro := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/problemas", gestor,
		`{"titulo":"Outro","unidade_id":"`+o.a+`"}`))
	obase := "/api/v1/problemas/" + outro.ID
	h.expect(http.StatusOK, http.MethodPut, obase+"/incidentes/"+incidentes[1], gestor, "")
	h.expect(http.StatusOK, http.MethodDelete, obase+"/incidentes/"+incidentes[1], gestor, "")
	h.expect(http.StatusOK, http.MethodDelete, obase+"/incidentes/"+incidentes[1], gestor, "")
	h.expect(http.StatusOK, http.MethodPut, obase+"/ativos/"+ativo.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodDelete, obase+"/ativos/"+ativo.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodDelete, obase+"/ativos/"+ativo.ID, gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPost, obase+"/estado", gestor, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/problemas/x/estado", gestor, `{"estado":"em_analise"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/problemas/x", gestor, `{"titulo":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, obase, gestor, `{`)
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/problemas/x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/problemas?solicitacao_id=x", gestor, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/problemas/"+uuid.NewString(), gestor, "")

	// Ativos desligado: o vínculo com ativo fica indisponível.
	h.setModule(h.admin(), "ativos", false)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, obase+"/ativos/"+ativo.ID, gestor, "")
	h.setModule(h.admin(), "ativos", true)
	h.setModule(h.admin(), "atendimento", false)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, obase+"/incidentes/"+incidentes[1], gestor, "")
	h.setModule(h.admin(), "atendimento", true)
}

// Fase 7: a mudança normal passa por aprovação de outra pessoa; a padrão
// é pré-aprovada; rejeitar exige parecer e falhar exige resultado.
func TestMudancas(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	_, gestor := h.pessoaEm(t, o.a, "mudancas:manage", "problemas:manage", "ativos:manage", "atendimento:manage")
	_, aprovador := h.pessoaEm(t, o.a, "mudancas:aprovar")
	_, gestorAprovador := h.pessoaEm(t, o.a, "mudancas:manage", "mudancas:aprovar")
	_, deB := h.pessoaEm(t, o.b, "mudancas:manage", "mudancas:aprovar")

	inicio := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	fim := time.Now().Add(26 * time.Hour).UTC().Format(time.RFC3339)
	corpo := func(tipo, extra string) string {
		return `{"titulo":"Trocar a fonte do switch","tipo":"` + tipo + `","risco":"medio","unidade_id":"` + o.a + `"` + extra + `}`
	}
	janela := `,"inicio_planejado":"` + inicio + `","fim_planejado":"` + fim + `"`

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/mudancas", gestor, `{"titulo":"x","tipo":"magica","risco":"baixo"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/mudancas", aprovador, corpo("normal", ""))
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/mudancas", deB, corpo("normal", ""))
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/mudancas", gestor, `{`)
	m := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/mudancas", gestor, corpo("normal", "")))
	mbase := "/api/v1/mudancas/" + m.ID
	estado := func(want int, tok, para, texto string) numeroResp {
		t.Helper()
		rec := h.expect(want, http.MethodPost, mbase+"/estado", tok, `{"estado":"`+para+`","texto":"`+texto+`"}`)
		if want != http.StatusOK {
			return numeroResp{}
		}
		return data[numeroResp](t, rec)
	}
	estado(http.StatusUnprocessableEntity, gestor, "em_avaliacao", "")
	h.expect(http.StatusOK, http.MethodPut, mbase, gestor, corpo("normal", janela))
	estado(http.StatusUnprocessableEntity, gestor, "em_avaliacao", "")
	h.expect(http.StatusOK, http.MethodPut, mbase, gestor, corpo("normal", janela+`,"plano_reversao":"Recolocar a fonte antiga"`))
	h.expect(http.StatusOK, http.MethodPut, mbase, gestor, corpo("normal", janela+`,"plano_reversao":"Recolocar a fonte antiga"`))
	if outboxCount(t, h, "mudancas.mudanca.atualizada", m.ID) != 2 {
		t.Fatal("edição vazia não emite evento")
	}
	if e := estado(http.StatusOK, gestor, "em_avaliacao", ""); e.Estado != "em_avaliacao" {
		t.Fatalf("submetida: %+v", e)
	}
	h.expect(http.StatusConflict, http.MethodPut, mbase, gestor, corpo("normal", janela))
	estado(http.StatusConflict, gestor, "aprovada", "")
	if d := data[numeroResp](t, h.expect(http.StatusOK, http.MethodGet, mbase, aprovador, "")); strings.Join(d.Proximos, ",") != "aprovada,rejeitada" {
		t.Fatalf("o aprovador vê a decisão: %+v", d)
	}
	estado(http.StatusUnprocessableEntity, aprovador, "rejeitada", "")
	estado(http.StatusOK, aprovador, "rejeitada", "Janela coincide com a matrícula")
	h.expect(http.StatusOK, http.MethodPut, mbase, gestor, corpo("normal", janela+`,"plano_reversao":"Recolocar a fonte antiga","justificativa":"Fonte queimada"`))
	estado(http.StatusOK, gestor, "em_avaliacao", "")
	estado(http.StatusOK, aprovador, "aprovada", "De acordo")
	h.expect(http.StatusNotFound, http.MethodGet, mbase, deB, "")
	estado(http.StatusNotFound, deB, "em_execucao", "")
	estado(http.StatusOK, gestor, "em_execucao", "")
	estado(http.StatusUnprocessableEntity, gestor, "falhou", "")
	estado(http.StatusOK, gestor, "concluida", "Fonte trocada")
	estado(http.StatusOK, gestor, "concluida", "")
	estado(http.StatusConflict, gestor, "cancelada", "")
	d := h.expect(http.StatusOK, http.MethodGet, mbase, gestor, "").Body.String()
	if !strings.Contains(d, `"decisao":"rejeitada"`) || !strings.Contains(d, `"decisao":"aprovada"`) || !strings.Contains(d, "Janela coincide") {
		t.Fatalf("as decisões ficam registradas: %s", d)
	}
	var auditoria int
	if err := h.d.DB.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE resource_type = 'mudanca' AND resource_id = $1`, m.ID).Scan(&auditoria); err != nil || auditoria < 8 {
		t.Fatalf("auditoria: %d %v", auditoria, err)
	}

	// Segregação: quem pede não aprova, mesmo tendo a permissão.
	propria := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/mudancas", gestorAprovador,
		corpo("emergencial", janela+`,"plano_reversao":"Desfazer"`)))
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+propria.ID+"/estado", gestorAprovador, `{"estado":"em_avaliacao"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/mudancas/"+propria.ID+"/estado", gestorAprovador, `{"estado":"aprovada"}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+propria.ID+"/estado", gestorAprovador, `{"estado":"cancelada"}`)

	// Padrão: pré-aprovada ao submeter; falha registrada com resultado.
	padrao := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/mudancas", gestor, corpo("padrao", janela)))
	sub := data[numeroResp](t, h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+padrao.ID+"/estado", gestor, `{"estado":"em_avaliacao"}`))
	if sub.Estado != "aprovada" {
		t.Fatalf("padrão é pré-aprovada: %+v", sub)
	}
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+padrao.ID+"/estado", gestor, `{"estado":"em_execucao"}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+padrao.ID+"/estado", gestor, `{"estado":"falhou","texto":"Revertido para a versão anterior"}`)

	// Vínculos com problema, incidente e ativo.
	prob := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/problemas", gestor, `{"titulo":"Fonte","unidade_id":"`+o.a+`"}`))
	ativo := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/ativos", gestor,
		`{"nome":"Switch","tipo":"rede","estado":"em_uso","unidade_id":"`+o.a+`"}`))
	_, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Rede","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	pedido := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Cabo","tipo":"requisicao","unidade_id":"`+o.a+`"}`))
	inc := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Sem rede"}`))
	req := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+pedido.ID+`","titulo":"Cabo"}`))
	nova := data[numeroResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/mudancas", gestor, corpo("normal", "")))
	vbase := "/api/v1/mudancas/" + nova.ID + "/vinculos/"
	h.expect(http.StatusOK, http.MethodPut, vbase+"problema/"+prob.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodPut, vbase+"problema/"+prob.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodPut, vbase+"incidente/"+inc.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodPut, vbase+"ativo/"+ativo.ID, gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"incidente/"+req.ID, gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"cometa/"+ativo.ID, gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"ativo/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"problema/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"incidente/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, vbase+"ativo/x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/mudancas/x/vinculos/ativo/"+ativo.ID, gestor, "")
	ligadas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/mudancas?alvo_tipo=problema&alvo_id="+prob.ID, aprovador, "").Body.String()
	if !strings.Contains(ligadas, nova.ID) || strings.Contains(ligadas, m.ID) {
		t.Fatalf("mudanças do problema: %s", ligadas)
	}
	if d := h.expect(http.StatusOK, http.MethodGet, "/api/v1/mudancas/"+nova.ID, gestor, "").Body.String(); !strings.Contains(d, "Problema #") || !strings.Contains(d, "Ativo · Switch") {
		t.Fatalf("vínculos com rótulo: %s", d)
	}
	h.expect(http.StatusOK, http.MethodDelete, vbase+"ativo/"+ativo.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodDelete, vbase+"ativo/"+ativo.ID, gestor, "")
	h.expect(http.StatusForbidden, http.MethodPut, vbase+"ativo/"+ativo.ID, aprovador, "")
	h.setModule(h.admin(), "problemas", false)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, vbase+"problema/"+prob.ID, gestor, "")
	h.setModule(h.admin(), "problemas", true)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/mudancas/"+nova.ID+"/estado", gestor, `{"estado":"cancelada"}`)
	h.expect(http.StatusConflict, http.MethodPut, vbase+"ativo/"+ativo.ID, gestor, "")

	if lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/mudancas?estado=concluida", aprovador, "").Body.String(); !strings.Contains(lista, m.ID) {
		t.Fatalf("lista por estado: %s", lista)
	}
	if vazia := h.expect(http.StatusOK, http.MethodGet, "/api/v1/mudancas", h.mustUser(t), "").Body.String(); !strings.Contains(vazia, "[]") {
		t.Fatalf("sem permissão, lista vazia: %s", vazia)
	}
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/mudancas?alvo_id=x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/mudancas/x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/mudancas/x", gestor, corpo("normal", ""))
	h.expect(http.StatusBadRequest, http.MethodPut, mbase, gestor, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/mudancas/x/estado", gestor, `{"estado":"cancelada"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, mbase+"/estado", gestor, `{`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/mudancas/"+uuid.NewString(), gestor, corpo("normal", ""))
}

// mustUser cria um usuário sem permissões e devolve o token.
func (h *apiHarness) mustUser(t *testing.T) string {
	t.Helper()
	_, tok := h.user("nova-user")
	return tok
}
