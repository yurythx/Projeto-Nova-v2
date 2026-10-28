package glpi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// poolDaCarga é o banco de teste com a vez no registro da importação, que
// é global: os testes de outros pacotes esperam.
func poolDaCarga(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t)
	dbtest.Exclusivo(t, pool, "glpi")
	return pool
}

func TestDesfazerCarga(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	repo := infrastructure.NewRepository()
	u := dbtest.Unidade(t, pool)
	ana := dbtest.User(t, pool)
	var login string
	if err := pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, ana).Scan(&login); err != nil {
		t.Fatal(err)
	}
	sfx := uuid.NewString()[:6]
	feriado := "Feriado do desfazer " + sfx
	c7, c8 := "Rede "+sfx, "Sistemas "+sfx
	limpar := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_categorias WHERE nome IN ($1, $2)`, c7, c8)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1`, feriado)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	}
	limpar()
	t.Cleanup(limpar)

	// A oferta "Geral" já existia: a tabela a reaproveita.
	geral, err := domain.NovaOferta("Geral", "Nossa.", domain.TipoIncidente, nil, &u, true)
	if err != nil {
		t.Fatal(err)
	}
	if geral, err = repo.InsertOferta(ctx, pool, geral); err != nil {
		t.Fatal(err)
	}
	destino := func(oferta, catalogo string) Destino {
		return Destino{Unidade: &u, UnidadeNome: "U", Oferta: oferta, Catalogo: catalogo, Tipo: "incidente", Prioridade: "media"}
	}
	mapa := NovoMapa([]LinhaMapa{
		{Secretaria: "SMS", Categoria: -1, Destino: destino("Geral", "")},
		{Secretaria: "SMS", Categoria: 7, Destino: destino("Rede "+sfx, c7)},
		{Secretaria: "SMS", Categoria: 8, Destino: destino("Sistemas "+sfx, c8)},
	})
	origem := expedienteDeTeste(t, login, feriado)
	for _, ch := range []Chamado{{ID: 201, CategoriaID: 7}, {ID: 202, CategoriaID: 8}, {ID: 203}} {
		ch.Titulo, ch.Tipo, ch.EntidadeID, ch.Abertura = "Chamado", 1, 1, "2025-03-10 09:00:00"
		origem.guardar(ItemPreparo{Tipo: ItemChamado, ID: ch.ID, Dados: dados(t, Pacote{Chamado: ch})})
	}
	fuso, _ := time.LoadLocation("America/Cuiaba")
	im := &Importador{Fonte: &FontePreparo{Leitor: origem}, Pool: pool, Repo: repo, Gravar: true, Fuso: fuso, Mapa: mapa}
	if rel, err := im.Executar(ctx); err != nil || rel.Importados != 3 || rel.Expediente.Equipes != 1 || rel.Expediente.Feriados != 2 {
		t.Fatalf("carga: %+v %v", rel, err)
	}

	// Alguém respondeu o 202 aqui depois da carga: ele fica, e com ele a
	// oferta e a categoria dele.
	var mexido uuid.UUID
	var numero int64
	if err := pool.QueryRow(ctx, `SELECT s.id, s.numero FROM atendimento_importacao i JOIN atendimento_solicitacoes s ON s.id = i.destino
		WHERE i.origem = 'glpi' AND i.tipo = 'solicitacao' AND i.id_origem = '202'`).Scan(&mexido, &numero); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.InsertComentario(ctx, pool, domain.Comentario{ID: uuid.New(), SolicitacaoID: mexido, AutorID: &ana,
		Texto: "Resolvido aqui.", CreatedAt: time.Now().UTC().Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	contar := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	chamados := func() int { return contar(`SELECT count(*) FROM atendimento_solicitacoes WHERE unidade_id = $1`, u) }
	esperado := Desfeito{Chamados: 2, Ofertas: 1, Categorias: 1, Equipes: 1, Politicas: 1, Feriados: 2,
		Mantidos: 1, Exemplos: []int64{numero}, OfertasMantidas: 1, CadastrosMantido: 1}
	conferir := func(r Desfeito, gravou bool) {
		t.Helper()
		esperado.Gravou = gravou
		if r.Gravou != esperado.Gravou || r.Chamados != esperado.Chamados || r.Ofertas != esperado.Ofertas || r.Categorias != esperado.Categorias ||
			r.Equipes != esperado.Equipes || r.Politicas != esperado.Politicas || r.Feriados != esperado.Feriados || r.Mantidos != esperado.Mantidos ||
			len(r.Exemplos) != 1 || r.Exemplos[0] != numero || r.OfertasMantidas != esperado.OfertasMantidas || r.CadastrosMantido != esperado.CadastrosMantido {
			t.Fatalf("relatório: %+v, esperado %+v", r, esperado)
		}
	}

	// A simulação dá os números da execução e não remove nada.
	passos := 0
	sim, err := Desfazer(ctx, pool, false, func(feitos, total int) { passos = feitos })
	if err != nil {
		t.Fatal(err)
	}
	conferir(sim, false)
	if chamados() != 3 || passos != 2+len(cadastros) {
		t.Fatalf("a simulação não remove: %d chamados, %d passos", chamados(), passos)
	}

	r, err := Desfazer(ctx, pool, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	conferir(r, true)
	if chamados() != 1 || contar(`SELECT count(*) FROM atendimento_ofertas WHERE id = $1`, geral.ID) != 1 ||
		contar(`SELECT count(*) FROM atendimento_categorias WHERE nome IN ($1, $2)`, c7, c8) != 1 ||
		contar(`SELECT count(*) FROM atendimento_equipes WHERE unidade_id = $1`, u) != 0 ||
		contar(`SELECT count(*) FROM atendimento_politicas WHERE unidade_id IS NULL`) != 0 ||
		contar(`SELECT count(*) FROM atendimento_feriados WHERE nome = $1`, feriado) != 0 {
		t.Fatal("o que ficou")
	}
	// No registro fica só o que ainda existe: uma carga nova traz o resto.
	if n := contar(`SELECT count(*) FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao'`); n != 1 {
		t.Fatalf("registro: %d", n)
	}

	// De novo: só o que ficou, nada a remover.
	esperado = Desfeito{Mantidos: 1, OfertasMantidas: 1, CadastrosMantido: 1}
	if r, err := Desfazer(ctx, pool, true, nil); err != nil {
		t.Fatal(err)
	} else {
		conferir(r, true)
	}

	// Cancelado no meio, a transação volta inteira.
	for _, passo := range []int{1, 2} {
		cctx, cancelar := context.WithCancel(ctx)
		_, err := Desfazer(cctx, pool, true, func(feitos, total int) {
			if feitos == passo {
				cancelar()
			}
		})
		cancelar()
		if err == nil {
			t.Fatalf("cancelado no passo %d", passo)
		}
	}
	if _, err := Desfazer(canceladoJa(ctx), pool, false, nil); err == nil {
		t.Fatal("sem começar")
	}
	if chamados() != 1 {
		t.Fatal("o cancelado não remove")
	}
}

func canceladoJa(ctx context.Context) context.Context {
	c, cancelar := context.WithCancel(ctx)
	cancelar()
	return c
}
