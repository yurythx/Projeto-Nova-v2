package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/yurythx/sismarq/internal/platform/audit"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/search"
)

// storeQueRecusa guarda o estado em memória e recusa a gravação.
type storeQueRecusa struct{}

func (storeQueRecusa) Ensure(context.Context, string, bool) error { return nil }
func (storeQueRecusa) LoadAll(context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (storeQueRecusa) Set(context.Context, string, bool, string, audit.Entry) error {
	return errors.New("banco fora")
}
func (storeQueRecusa) Listen(ctx context.Context, _ func()) error {
	<-ctx.Done()
	return nil
}

// indexador é um provedor de busca que também alimenta o índice.
type indexador struct{}

func (indexador) Module() string { return "falso" }
func (indexador) Search(context.Context, auth.Identity, string, int) ([]search.Result, error) {
	return nil, nil
}
func (indexador) Documents(context.Context, func(search.Document) error) error { return nil }

type pluginFalso struct{ key string }

func (p pluginFalso) Manifest() kernel.Manifest {
	return kernel.Manifest{Key: p.key, Name: p.key}
}
func (p pluginFalso) SearchProviders() []search.Provider { return []search.Provider{indexador{}} }

func kernelFalso(t *testing.T, plugins ...kernel.Plugin) *kernel.Kernel {
	t.Helper()
	k := kernel.New(storeQueRecusa{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, p := range plugins {
		if err := k.RegisterModule(p); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := k.Start(ctx); err != nil {
		t.Fatal(err)
	}
	return k
}

func TestPluginsOnInTestsRepassaAFalhaDoStore(t *testing.T) {
	k := kernelFalso(t, pluginFalso{key: "falso"})
	if err := pluginsOnInTests(context.Background(), k); err == nil || err.Error() != "banco fora" {
		t.Fatalf("err = %v", err)
	}
}

func TestIndexedModulesListaQuemAlimentaOIndice(t *testing.T) {
	d := &Dependencies{Kernel: kernelFalso(t, pluginFalso{key: "falso"})}
	if got := d.indexedModules(); len(got) != 1 || got[0] != "falso" {
		t.Fatalf("indexedModules = %v", got)
	}
}
