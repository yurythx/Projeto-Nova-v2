package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Ocorrências (ADR 066): só ocorrencias:manage declara e atualiza; a
// página de status responde sem login, com cache curto e só o que é público.
func TestOcorrenciasHTTP(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	admin := h.admin()
	_, comum := h.user("nova-user")
	// A página fala pela instituição inteira: a permissão só vale com
	// concessão global (ADR 013). Lotada numa unidade, não declara.
	_, daUnidade := h.pessoaEm(t, o.a, "ocorrencias:manage")
	perfil := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/iam/perfis", admin,
		`{"nome":"Coordenação de incidentes `+uuid.NewString()[:8]+`","permissoes":["ocorrencias:manage"]}`)).ID
	coordID, coordena := h.user("nova-user")
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/users/"+coordID.String()+"/lotacoes", admin, `{"perfil_id":"`+perfil+`"}`)

	h.expect(http.StatusUnauthorized, http.MethodGet, "/api/v1/ocorrencias", "", "")
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/ocorrencias", comum, `{}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/ocorrencias", daUnidade, `{}`)

	criada := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/ocorrencias", coordena,
		`{"titulo":"Protocolo eletrônico fora do ar","servicos":["Protocolo"],"impacto":"fora","texto":"Estamos investigando."}`))
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/ocorrencias/"+criada.ID+"/atualizacoes", coordena,
		`{"estado":"identificado","impacto":"parcial","texto":"O certificado venceu; renovando."}`)

	// Sem login: a situação geral e a linha do tempo, sem quem atuou.
	rec := h.expect(http.StatusOK, http.MethodGet, "/api/v1/status", "", "")
	corpo := rec.Body.String()
	if !strings.Contains(corpo, criada.ID) || !strings.Contains(corpo, "O certificado venceu") || !strings.Contains(corpo, `"geral":"parcial"`) {
		t.Fatalf("página: %s", corpo)
	}
	if strings.Contains(corpo, "coordenador") || strings.Contains(corpo, "autor") || strings.Contains(corpo, "nova-user") {
		t.Fatalf("a página leva quem atuou: %s", corpo)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=30" {
		t.Fatalf("cache: %v", rec.Header())
	}

	h.expect(http.StatusOK, http.MethodPost, "/api/v1/ocorrencias/"+criada.ID+"/atualizacoes", coordena,
		`{"estado":"resolvido","impacto":"parcial","texto":"Voltou ao normal."}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/ocorrencias/"+criada.ID+"/revisao", coordena, `{"texto":"Renovar com antecedência."}`)
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/status", "", "").Body.String(); strings.Contains(body, "Renovar com antecedência") {
		t.Fatalf("a revisão é interna: %s", body)
	}
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/ocorrencias/"+criada.ID, coordena, "").Body.String(); !strings.Contains(body, "Renovar com antecedência") {
		t.Fatalf("o painel mostra a revisão: %s", body)
	}
	if outboxCount(t, h, "ocorrencias.resolvida", criada.ID) != 1 {
		t.Fatal("evento de resolução")
	}
}
