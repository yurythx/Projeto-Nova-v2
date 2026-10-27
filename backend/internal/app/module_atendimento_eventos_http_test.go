package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Entrada de eventos (ADR 067), de ponta a ponta: o Alertmanager manda o
// webhook com o token da máquina, sem sessão de pessoa; o alerta vira um
// incidente uma vez e o reenvio vira nota.
func TestAtendimentoEventosHTTP(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	admin := h.admin()
	o := h.orgEscopo(t)
	_, gestorDaUnidade := h.pessoaEm(t, o.a, "atendimento:manage")
	t.Cleanup(func() { _, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_eventos_config`) })

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", admin,
		`{"nome":"Alertas da infraestrutura","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	fp := "fp-" + uuid.NewString()
	corpo := `{"version":"4","status":"firing","receiver":"atendimento","alerts":[{"status":"firing","fingerprint":"` + fp +
		`","labels":{"alertname":"NovaDiscoCritico","severity":"critical"},"annotations":{"summary":"Disco da VM acima de 95%"},"startsAt":"2026-10-27T14:00:00Z"}]}`
	webhook := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/atendimento/eventos/alertmanager", strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.router.ServeHTTP(rec, req)
		return rec
	}

	if rec := webhook(""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sem token: %d", rec.Code)
	}
	if rec := webhook(tokenEventosTeste); rec.Code != http.StatusConflict {
		t.Fatalf("sem oferta escolhida, o Alertmanager tenta de novo: %d %s", rec.Code, rec.Body.String())
	}
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/atendimento/eventos/configuracao", gestorDaUnidade, `{"oferta_id":"`+oferta.ID+`"}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/eventos/configuracao", admin, `{"oferta_id":"`+oferta.ID+`"}`)

	if rec := webhook(tokenEventosTeste); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"abertos":1`) {
		t.Fatalf("abre: %d %s", rec.Code, rec.Body.String())
	}
	if rec := webhook(tokenEventosTeste); !strings.Contains(rec.Body.String(), `"anotados":1`) {
		t.Fatalf("reenvio vira nota: %s", rec.Body.String())
	}
	var titulo string
	var impacto int
	if err := h.d.DB.QueryRow(ctx, `SELECT s.titulo, s.impacto FROM atendimento_eventos e JOIN atendimento_solicitacoes s ON s.id = e.solicitacao_id
		WHERE e.fingerprint = $1`, fp).Scan(&titulo, &impacto); err != nil || titulo != "Disco da VM acima de 95%" || impacto != 3 {
		t.Fatalf("chamado do alerta: %q %d %v", titulo, impacto, err)
	}
}
