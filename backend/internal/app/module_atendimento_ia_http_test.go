package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// IA assistiva (ADR 065), de ponta a ponta: configurada pela tela de IA na
// função atendimento.assistente, ela sugere a quem atende, que decide; o
// chamado não muda, e pedido e decisão ficam na auditoria.
func TestAtendimentoIAHTTP(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	admin := h.admin()
	o := h.orgEscopo(t)
	_, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	_, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	t.Cleanup(func() {
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM ia_uso WHERE funcao = 'atendimento.assistente'`)
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM ia_conexoes WHERE nome = 'IA do atendimento'`)
	})

	var mu sync.Mutex
	var perguntas []string
	fornecedor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Content string } `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sistema, pergunta := body.Messages[0].Content, body.Messages[len(body.Messages)-1].Content
		mu.Lock()
		perguntas = append(perguntas, pergunta)
		mu.Unlock()
		resposta := "ok"
		switch {
		case strings.Contains(sistema, "classificar"):
			resposta = `{\"impacto\": 3, \"urgencia\": 2, \"motivo\": \"o setor inteiro sem e-mail\"}`
		case strings.Contains(sistema, "resume"):
			resposta = "Sem e-mail desde cedo; ninguém atuou ainda."
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + resposta + `"}}]}`))
	}))
	defer fornecedor.Close()

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"E-mail fora","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Sem e-mail","descricao":"meu CPF é 123.456.789-09 e o setor todo está sem e-mail"}`))
	classificacao := "/api/v1/atendimento/solicitacoes/" + sol.ID + "/ia/classificacao"

	// Sem conexão: a tela esconde os botões e o pedido é recusado.
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ia", tecnico, "").Body.String(); !strings.Contains(body, `"disponivel":false`) {
		t.Fatalf("sem IA: %s", body)
	}
	h.expect(http.StatusServiceUnavailable, http.MethodPost, classificacao, tecnico, "")

	// Configurada pela tela de IA, como fornecedor externo com mascaramento.
	conexao := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/ia/conexoes", admin,
		`{"nome":"IA do atendimento","provedor":"compativel","endpoint":"`+fornecedor.URL+`","modelo":"m","externo":true}`)).ID
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/ia/uso/atendimento.assistente", admin,
		`{"principal_id":"`+conexao+`","mascarar_dados_pessoais":true,"autorizo_envio_externo":true}`)

	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ia", tecnico, "").Body.String(); !strings.Contains(body, `"disponivel":true`) {
		t.Fatalf("com IA: %s", body)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/ia", servidor, "")
	h.expect(http.StatusForbidden, http.MethodPost, classificacao, servidor, "")

	sug := data[struct {
		ID            string `json:"id"`
		Classificacao struct {
			Impacto  int    `json:"impacto"`
			Urgencia int    `json:"urgencia"`
			Motivo   string `json:"motivo"`
		} `json:"classificacao"`
	}](t, h.expect(http.StatusCreated, http.MethodPost, classificacao, tecnico, ""))
	if sug.Classificacao.Impacto != 3 || sug.Classificacao.Urgencia != 2 || sug.Classificacao.Motivo == "" {
		t.Fatalf("sugestão: %+v", sug)
	}
	mu.Lock()
	ultima := perguntas[len(perguntas)-1]
	mu.Unlock()
	if strings.Contains(ultima, "123.456.789-09") || !strings.Contains(ultima, "[CPF]") {
		t.Fatalf("o CPF saiu da rede: %q", ultima)
	}
	if body := h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/ia/resumo", tecnico, "").Body.String(); !strings.Contains(body, "Sem e-mail desde cedo") {
		t.Fatalf("resumo: %s", body)
	}

	// A IA não muda o chamado: a classificação continua a da abertura.
	var impacto int
	if err := h.d.DB.QueryRow(ctx, `SELECT impacto FROM atendimento_solicitacoes WHERE id = $1`, sol.ID).Scan(&impacto); err != nil || impacto == 3 {
		t.Fatalf("a IA classificou sozinha: %d %v", impacto, err)
	}

	decisao := "/api/v1/atendimento/ia/sugestoes/" + sug.ID + "/decisao"
	h.expect(http.StatusOK, http.MethodPost, decisao, tecnico, `{"aceita":true}`)
	h.expect(http.StatusConflict, http.MethodPost, decisao, tecnico, `{"aceita":false}`)
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ia/indicadores", tecnico, "").Body.String(); !strings.Contains(body, `"aceitas":1`) {
		t.Fatalf("indicadores: %s", body)
	}

	var sugeridas, decididas int
	_ = h.d.DB.QueryRow(ctx, `SELECT count(*) FILTER (WHERE action = 'atendimento.ia.sugerida'), count(*) FILTER (WHERE action = 'atendimento.ia.decidida')
		FROM audit_logs WHERE resource_id = $1`, sol.ID).Scan(&sugeridas, &decididas)
	if sugeridas != 2 || decididas != 1 {
		t.Fatalf("auditoria: %d sugeridas, %d decididas", sugeridas, decididas)
	}

	// Fornecedor fora do ar: 503, e quem atende segue sem a IA.
	fornecedor.Close()
	h.expect(http.StatusServiceUnavailable, http.MethodPost, classificacao, tecnico, "")
}
