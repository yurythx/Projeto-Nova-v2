package app

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Tarefas por equipe (ADR 068), de ponta a ponta: a admissão chega à fila
// com uma tarefa por equipe, cada equipe é avisada da sua, e o pedido só é
// dado por atendido depois de todas.
func TestAtendimentoTarefasHTTP(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	tecID, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	redeID, rede := h.pessoaEm(t, o.a, "atendimento:atender")
	_, servidor := h.pessoaEm(t, o.a, "atendimento:abrir")

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Admissão de servidor","tipo":"requisicao","unidade_id":"`+o.a+`"}`))
	suporte := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Suporte","unidade_id":"`+o.a+`","membros":["`+tecID.String()+`"]}`))
	equipeRede := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Rede","unidade_id":"`+o.a+`","membros":["`+redeID.String()+`"]}`))
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID+"/tarefas", tecnico,
		`{"tarefas":[{"titulo":"Criar conta","equipe_id":"`+suporte.ID+`"},{"titulo":"Liberar VPN","equipe_id":"`+equipeRede.ID+`"}]}`)
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ofertas/"+oferta.ID+"/tarefas", tecnico, "").Body.String(); !strings.Contains(body, "Liberar VPN") {
		t.Fatalf("modelo: %s", body)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/ofertas/"+oferta.ID+"/tarefas", rede, "")

	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Admitir Ana"}`))
	var avisos int
	if err := h.d.DB.QueryRow(context.Background(), `SELECT count(*) FROM notificacoes WHERE user_id = $1 AND titulo LIKE 'Tarefa no pedido%'`, redeID).Scan(&avisos); err != nil || avisos != 1 {
		t.Fatalf("a rede é avisada da tarefa dela: %d %v", avisos, err)
	}

	d := data[detalheTarefas](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, rede, ""))
	if len(d.Tarefas) != 2 || d.Tarefas[0].Minha || !d.Tarefas[1].Minha {
		t.Fatalf("tarefas vistas pela rede: %+v", d.Tarefas)
	}
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico, `{"responsavel_id":"`+tecID.String()+`"}`)
	h.expect(http.StatusConflict, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/estado", tecnico, `{"estado":"atendido"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/tarefas/"+d.Tarefas[1].ID, tecnico, `{"concluir":true}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/tarefas/"+d.Tarefas[0].ID, servidor, `{"concluir":true}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/tarefas/"+d.Tarefas[0].ID, tecnico, `{"concluir":true,"observacao":"ana.silva"}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/tarefas/"+d.Tarefas[1].ID, rede, `{"concluir":true}`)
	if outboxCount(t, h, "atendimento.solicitacao.tarefa", sol.ID) != 2 {
		t.Fatal("evento de cada tarefa")
	}
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/estado", tecnico, `{"estado":"atendido"}`)
}

type detalheTarefas struct {
	Tarefas []struct {
		ID    string `json:"id"`
		Minha bool   `json:"minha"`
	} `json:"tarefas"`
}
