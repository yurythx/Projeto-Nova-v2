package problemas

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

func TestConcluirPorMudancaDelega(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	m := New(modkit.Deps{}, application.Portas{})
	if err := m.ConcluirPorMudanca(ctx, tx, uuid.New(), uuid.New(), "pela mudança"); err == nil {
		t.Fatal("problema inexistente não é concluído")
	}
}
