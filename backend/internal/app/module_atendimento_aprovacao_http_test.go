package app

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Aprovação em etapas (ADR 068), de ponta a ponta: a chefia, sem nenhuma
// permissão do atendimento, vê o pedido que espera a sua etapa e decide;
// a equipe só atende depois da última aprovação.
func TestAtendimentoAprovacaoHTTP(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	o := h.orgEscopo(t)
	tecID, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	_, servidor := h.pessoaEm(t, o.a, "atendimento:abrir")
	chefeID, chefe := h.user("nova-user")
	_, curioso := h.user("nova-user")

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Admissão de servidor","tipo":"requisicao","unidade_id":"`+o.a+`"}`))
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Suporte","unidade_id":"`+o.a+`","membros":["`+tecID.String()+`"]}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID+"/etapas", admin,
		`{"etapas":[{"nome":"Chefia imediata","aprovadores":["`+chefeID.String()+`"]}]}`)
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ofertas/"+oferta.ID+"/etapas", tecnico, "").Body.String(); !strings.Contains(body, "Chefia imediata") {
		t.Fatalf("etapas: %s", body)
	}

	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Admitir Ana (matrícula 123)"}`))
	if sol.Estado != "aguardando_aprovacao" {
		t.Fatalf("abre aguardando: %+v", sol)
	}
	h.expect(http.StatusConflict, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico, `{"responsavel_id":"`+tecID.String()+`"}`)

	// O aviso leva a chefia à tela de aprovações, não ao chamado que ela não vê.
	var link string
	if err := h.d.DB.QueryRow(context.Background(), `SELECT link FROM notificacoes WHERE user_id = $1 AND modulo = 'atendimento'`, chefeID).Scan(&link); err != nil || link != "/atendimento/aprovacoes" {
		t.Fatalf("aviso à chefia: %q %v", link, err)
	}
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/aprovacoes", chefe, "").Body.String(); !strings.Contains(body, sol.ID) {
		t.Fatalf("a chefia vê o pedido: %s", body)
	}
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/aprovacoes", curioso, "").Body.String(); strings.Contains(body, sol.ID) {
		t.Fatalf("quem não aprova não vê: %s", body)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/fila", chefe, "")
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/aprovacao", curioso, `{"aprovar":true}`)

	h.expect(http.StatusOK, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/aprovacao", chefe, `{"aprovar":true}`)
	d := data[detalheResp](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, tecnico, ""))
	if d.Solicitacao.Estado != "aberto" {
		t.Fatalf("aprovado, vai à fila: %+v", d.Solicitacao)
	}
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico, `{"responsavel_id":"`+tecID.String()+`"}`)
	h.expect(http.StatusConflict, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/aprovacao", chefe, `{"aprovar":true}`)
	if outboxCount(t, h, "atendimento.solicitacao.aprovacao", sol.ID) != 1 {
		t.Fatal("evento da decisão")
	}
}
