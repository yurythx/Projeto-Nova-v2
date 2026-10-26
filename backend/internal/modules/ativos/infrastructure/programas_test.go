package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestContarProgramaMarcaListaCortada(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	repo := NewRepository()
	com, err := domain.Novo(domain.Entrada{Nome: "Balcão", Tipo: "hardware", Estado: "em_uso", UnidadeID: &unidade})
	if err != nil {
		t.Fatal(err)
	}
	com.Inventario = &domain.Inventario{
		Sistema: "linux", Processadores: 2,
		Programas: []domain.Programa{{Nome: "Editor"}}, ProgramasTruncado: true,
	}
	com.CreatedAt, com.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	if _, err := repo.Insert(ctx, pool, com); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM ativos WHERE id = $1`, com.ID) })

	uso, err := ContarPrograma(ctx, pool, true, []uuid.UUID{unidade}, "editor")
	if err != nil {
		t.Fatal(err)
	}
	if uso.Instalados != 1 || uso.ComInventario != 1 || !uso.Parcial {
		t.Fatalf("conta: %+v", uso)
	}
	vazio, err := ContarPrograma(ctx, pool, true, []uuid.UUID{unidade}, "outro-programa")
	if err != nil || vazio.Instalados != 0 || !vazio.Parcial || vazio.ComInventario != 1 {
		t.Fatalf("sem o nome: %+v %v", vazio, err)
	}
}
