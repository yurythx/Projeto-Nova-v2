package fazenda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

type libera struct{}

func (libera) Allow(context.Context, string) (bool, error) { return true, nil }

func TestModulo(t *testing.T) {
	t.Setenv("GOVBR_CLIENT_ID", "")
	m := New(modkit.Deps{PublicLimiter: libera{}})
	man := m.Manifest()
	if man.Key != Key || man.DefaultEnabled || !man.Public || man.Route != "/fazenda" || len(man.Permissions) != 3 {
		t.Fatalf("%+v", man)
	}
	pub := chi.NewRouter()
	authed := chi.NewRouter()
	m.RegisterRoutes(kernel.Routes{Public: pub, Authed: authed})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fazenda/govbr", nil)
	req.RemoteAddr = "127.0.0.1:9"
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusOK {
		t.Fatalf("gov.br: %d %s", rec.Code, rec.Body.String())
	}
	if workers := m.Workers(); len(workers) != 1 || workers[0].Name != "alertas-de-prazo" {
		t.Fatalf("%+v", workers)
	}
}
