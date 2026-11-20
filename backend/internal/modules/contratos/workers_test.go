package contratos

import (
	"context"
	"log/slog"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

func TestWorkersDoVencimento(t *testing.T) {
	m := New(modkit.Deps{Logger: slog.New(slog.DiscardHandler)}, nil)
	lista := m.Workers()
	if len(lista) != 1 || lista[0].Name != "vencimento-de-contratos" || lista[0].Process != kernel.ProcessWorker {
		t.Fatalf("%+v", lista)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := lista[0].Run(ctx); err != nil {
		t.Fatal(err)
	}
}
