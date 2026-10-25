package atendimento

import (
	"context"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
)

func TestBuscaNaoVaiParaOIndice(t *testing.T) {
	m := New(modkit.Deps{}, nil)
	ps := m.SearchProviders()
	if len(ps) != 1 || ps[0].Module() != Key {
		t.Fatalf("provider: %d", len(ps))
	}
	if _, publico := ps[0].(search.Indexer); publico {
		t.Fatal("solicitação não é conteúdo público")
	}
	res, err := ps[0].Search(context.Background(), auth.Identity{}, "   ", 5)
	if err != nil || len(res) != 0 {
		t.Fatalf("consulta em branco: %+v %v", res, err)
	}
}
