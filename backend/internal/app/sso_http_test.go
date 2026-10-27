package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/iam"
	iamapp "github.com/yurythx/projeto-nova-v2/internal/modules/iam/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/keycloakadmin"
)

// keycloakDeTeste: uma pessoa, com um grupo.
func keycloakDeTeste(t *testing.T, sub string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/x/protocol/openid-connect/token":
			_, _ = w.Write([]byte(`{"access_token":"tk"}`))
		case "/admin/realms/x/users":
			_, _ = w.Write([]byte(`[{"id":"` + sub + `","username":"` + sub + `","firstName":"Ana","enabled":true}]`))
		default:
			_, _ = w.Write([]byte(`[{"name":"Administrators"}]`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSincronizacaoSSOHTTP(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	_, comum := h.user("nova-user")
	var mod *iam.Module
	for _, p := range h.d.Kernel.Plugins() {
		if m, ok := p.(*iam.Module); ok {
			mod = m
		}
	}
	if mod == nil {
		t.Fatal("IAM sem registro")
	}
	if ws := mod.Workers(); len(ws) != 1 || ws[0].Process != kernel.ProcessWorker {
		t.Fatalf("worker: %+v", ws)
	}
	sub := "kc-" + uuid.NewString()
	srv := keycloakDeTeste(t, sub)
	porta := portaSSO{cli: keycloakadmin.Novo(), credenciais: func(context.Context) (string, string, string) {
		return srv.URL + "/realms/x", "projeto-nova", "s"
	}}
	mod.Servico().WithDiretorio(porta, slog.New(slog.DiscardHandler))

	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/iam/sso/sincronizar", comum, "")
	rel := data[iamapp.RelatorioSSO](t, h.expect(http.StatusOK, http.MethodPost, "/api/v1/iam/sso/sincronizar", admin, ""))
	if rel.Criados != 1 {
		t.Fatalf("relatório: %+v", rel)
	}
	t.Cleanup(func() { _, _ = h.d.DB.Exec(context.Background(), `DELETE FROM users WHERE keycloak_subject = $1`, sub) })

	// Sem client e segredo, a sincronização pede a configuração.
	mod.Servico().WithDiretorio(portaSSO{cli: keycloakadmin.Novo(), credenciais: func(context.Context) (string, string, string) { return "", "", "" }},
		slog.New(slog.DiscardHandler))
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/iam/sso/sincronizar", admin, "")
	if _, err := (portaSSO{cli: keycloakadmin.Novo(), credenciais: func(context.Context) (string, string, string) {
		return "http://127.0.0.1:1/realms/x", "p", "s"
	}}).Usuarios(context.Background()); err == nil || errors.Is(err, iamapp.ErrSemSSO) {
		t.Fatalf("Keycloak fora do ar: %v", err)
	}
}
