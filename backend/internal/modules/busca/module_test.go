package busca

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
)

type indiceVazio struct{}

func (indiceVazio) Search(context.Context, string, []string, int) ([]search.Result, error) {
	return nil, nil
}

// O índice (Typesense) é opcional: ligá-lo devolve o próprio módulo, para
// a composição encadear.
func TestWithIndex(t *testing.T) {
	m := New(modkit.Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}, func() []search.Provider { return nil })
	if m.WithIndex(indiceVazio{}) != m {
		t.Fatal("WithIndex devolve o módulo")
	}
}
