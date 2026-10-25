package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestCredencialDeMaquinaNaoPassaPeloJWT(t *testing.T) {
	unidade := uuid.New()
	logger := slog.New(slog.NewTextHandler(&discard{}, nil))
	h := RequireAuthentication(nil, func(context.Context, Identity) (Identity, error) {
		t.Fatal("a credencial não passa pelo IAM")
		return Identity{}, nil
	}, logger, func(_ context.Context, token string) (Identity, bool, error) {
		if token != "nova_agt_ok" {
			return Identity{}, false, nil
		}
		u := unidade
		return Identity{
			Subject: "agt", Source: SourceAgente, Permissions: []string{"ativos:reportar"},
			Scopes: []Scope{{UnidadeID: &u, Permissions: []string{"ativos:reportar"}}},
		}, true, nil
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentityFromContext(r.Context())
		if !ok || id.Source != SourceAgente || !Can(id, PermAtivosReportar, InUnidade(unidade)) {
			t.Errorf("%v %#v", ok, id)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ativos/inventario", nil)
	req.Header.Set("Authorization", "Bearer nova_agt_ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatal(rec.Code)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
