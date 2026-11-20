package quadros

import (
	"context"
	"log/slog"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

func TestModulo(t *testing.T) {
	ctx := context.Background()
	var vazio *Module
	if mapa, err := vazio.Mapa(ctx, nil, "problemas"); err != nil || len(mapa) != 0 {
		t.Fatalf("nil: %v %#v", err, mapa)
	}
	if mapa, err := (&Module{}).Mapa(ctx, nil, "problemas"); err != nil || len(mapa) != 0 {
		t.Fatalf("sem repo: %v %#v", err, mapa)
	}
	mod := New(modkit.Deps{Logger: slog.Default()})
	if mod.Manifest().Key != Key || !mod.Manifest().DefaultEnabled {
		t.Fatalf("%+v", mod.Manifest())
	}
	if _, err := mod.Mapa(ctx, dbtest.Fail{}, "problemas"); err == nil {
		t.Fatal("mapa")
	}
	r := chi.NewRouter()
	mod.RegisterRoutes(kernel.Routes{Authed: r})
	if r.Routes() == nil {
		t.Fatal("rotas")
	}
}
