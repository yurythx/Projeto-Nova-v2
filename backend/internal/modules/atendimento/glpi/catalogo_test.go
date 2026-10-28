package glpi

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type categoriaFalha struct{ domain.Repository }

func (categoriaFalha) InsertCategoria(context.Context, database.DBTX, domain.Categoria) (domain.Categoria, error) {
	return domain.Categoria{}, errFonte
}

// A raiz da árvore do GLPI vira a categoria do catálogo (ADR 078): a oferta
// criada entra nela; a que já existia só ganha categoria se não tinha.
func TestCatalogoDoGLPI(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	u := dbtest.Unidade(t, pool)
	sfx := uuid.NewString()[:6]
	hardware := "Hardware " + sfx
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_regras WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_categorias WHERE nome LIKE '%' || $1`, sfx)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi' AND id_origem LIKE 'mapa:cat-%'`)
	})
	real := infrastructure.NewRepository()
	// Uma oferta sem categoria e outra já posta em "Rede" por alguém.
	solta, _ := domain.NovaOferta("Impressora", "", domain.TipoIncidente, nil, &u, true)
	posta, _ := domain.NovaOferta("Wi-Fi", "", domain.TipoIncidente, nil, &u, true)
	rede, err := domain.NovaCategoria("Rede "+sfx, "", "", 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if rede, err = real.InsertCategoria(ctx, pool, rede); err != nil {
		t.Fatal(err)
	}
	posta.CategoriaID = &rede.ID
	for _, o := range []domain.Oferta{solta, posta} {
		if _, err := real.InsertOferta(ctx, pool, o); err != nil {
			t.Fatal(err)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	im := &Importador{Repo: real, ofertas: map[string]uuid.UUID{}}
	d := func(oferta, catalogo string) Destino {
		return Destino{Unidade: &u, UnidadeNome: "U", Oferta: oferta, Catalogo: catalogo, Tipo: "incidente", Prioridade: "media"}
	}
	nova, err := im.ofertaDoMapa(ctx, tx, "mapa:cat-1", d("Monitor", hardware))
	if err != nil {
		t.Fatal(err)
	}
	// O mesmo nome, com outra caixa e espaços, é a mesma categoria.
	if _, err := im.ofertaDoMapa(ctx, tx, "mapa:cat-2", d("Impressora", " "+strings.ToUpper(hardware)+" ")); err != nil {
		t.Fatal(err)
	}
	if _, err := im.ofertaDoMapa(ctx, tx, "mapa:cat-3", d("Wi-Fi", hardware)); err != nil {
		t.Fatal(err)
	}
	// Outra execução acha a categoria pelo banco; sem categoria, nada muda.
	outra := &Importador{Repo: real, ofertas: map[string]uuid.UUID{}}
	if _, err := outra.ofertaDoMapa(ctx, tx, "mapa:cat-4", d("Teclado", hardware)); err != nil {
		t.Fatal(err)
	}
	if _, err := outra.ofertaDoMapa(ctx, tx, "mapa:cat-5", d("Mouse", "")); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var categorias int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_categorias WHERE lower(nome) = lower($1)`, hardware).Scan(&categorias); err != nil || categorias != 1 {
		t.Fatalf("uma categoria só: %d %v", categorias, err)
	}
	porOferta := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT o.nome, coalesce(c.nome, '') FROM atendimento_ofertas o
		LEFT JOIN atendimento_categorias c ON c.id = o.categoria_id WHERE o.unidade_id = $1`, u)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var o, c string
		if err := rows.Scan(&o, &c); err != nil {
			t.Fatal(err)
		}
		porOferta[o] = c
	}
	rows.Close()
	quer := map[string]string{"Monitor": hardware, "Impressora": hardware, "Wi-Fi": "Rede " + sfx, "Teclado": hardware, "Mouse": ""}
	for o, c := range quer {
		if porOferta[o] != c {
			t.Errorf("%s: %q, quer %q", o, porOferta[o], c)
		}
	}
	if nova == uuid.Nil {
		t.Fatal("oferta nova")
	}

	// Falhas: gravar a categoria e consultar numa transação encerrada.
	tx, _ = pool.Begin(ctx)
	if _, err := (&Importador{Repo: categoriaFalha{real}, ofertas: map[string]uuid.UUID{}}).ofertaDoMapa(ctx, tx, "mapa:cat-f1",
		d("Scanner", "Nova "+sfx)); !errors.Is(err, errFonte) {
		t.Errorf("gravar a categoria: %v", err)
	}
	_ = tx.Rollback(ctx)
	if _, err := (&Importador{Repo: real, ofertas: map[string]uuid.UUID{}}).ofertaDoMapa(ctx, tx, "mapa:cat-f2", d("Scanner", hardware)); err == nil {
		t.Error("consulta numa transação encerrada")
	}
}
