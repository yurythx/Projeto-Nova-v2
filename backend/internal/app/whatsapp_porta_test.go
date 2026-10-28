package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento"
)

// A resposta pelo WhatsApp chega ao atendimento pela porta, e só com o
// plug-in ligado; o webhook é público e, sem o token, é 404 (ADR 078).
func TestPortaWhatsApp(t *testing.T) {
	h := newHarness(t)
	var mod *atendimento.Module
	for _, p := range h.d.Kernel.Plugins() {
		if m, ok := p.(*atendimento.Module); ok {
			mod = m
		}
	}
	if mod == nil {
		t.Fatal("atendimento sem registro")
	}
	ctx := context.Background()
	if r, err := (portaWhatsApp{mod: mod, ativo: func() bool { return false }}).ReceberWhatsApp(ctx, uuid.New(), "oi", "m1"); err != nil || r != "" {
		t.Fatalf("desligado: %q %v", r, err)
	}
	if r, err := (portaWhatsApp{mod: mod, ativo: func() bool { return true }}).ReceberWhatsApp(ctx, uuid.New(), "oi", "m2"); err != nil ||
		!strings.Contains(r, "não tem chamado") {
		t.Fatalf("ligado: %q %v", r, err)
	}
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/canais/whatsapp/webhook/"+uuid.NewString(), "", `{}`)

	// De ponta a ponta, pela montagem real: a administração liga a entrada
	// (Evolution), e a mensagem de quem deixou o número chega ao atendimento.
	t.Cleanup(func() { _, _ = h.d.DB.Exec(context.Background(), `DELETE FROM canais_whatsapp`) })
	cfg := data[struct {
		WebhookURL string `json:"webhook_url"`
	}](t, h.expect(http.StatusOK, http.MethodPut, "/api/v1/canais/whatsapp", h.admin(),
		`{"ativo":true,"receber":true,"provedor":"evolution","evolution_url":"http://127.0.0.1:1","evolution_instancia":"central","evolution_chave":"ch"}`))
	caminho := strings.TrimPrefix(cfg.WebhookURL, "http://api.test")
	if !strings.HasPrefix(caminho, "/api/v1/canais/whatsapp/webhook/") {
		t.Fatalf("endereço do webhook: %q", cfg.WebhookURL)
	}
	quem, _ := h.user("nova-user")
	numero := "+55669" + strings.Repeat("7", 8)
	if _, err := h.d.DB.Exec(ctx, `UPDATE users SET contato_whatsapp = CASE WHEN id = $1 THEN $2 ELSE '' END
		WHERE id = $1 OR contato_whatsapp = $2`, quem, numero); err != nil {
		t.Fatal(err)
	}
	h.expect(http.StatusOK, http.MethodPost, caminho, "", `{"event":"messages.upsert","data":{"key":{"remoteJid":"`+
		strings.TrimPrefix(numero, "+")+`@s.whatsapp.net","id":"E-`+uuid.NewString()+`"},"message":{"conversation":"oi"}}}`)
}
