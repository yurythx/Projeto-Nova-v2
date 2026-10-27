package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestAtribuicoesReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	quem, tec, sem := dbtest.User(t, pool), dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Rede "+uuid.NewString()[:6], "", domain.TipoIncidente, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	ola := 4
	criar := func(nome string) domain.Equipe {
		e, _ := domain.NovaEquipe(nome+" "+uuid.NewString()[:6], &u, []uuid.UUID{tec})
		e.OLAHoras = &ola
		if _, err := r.InsertEquipe(ctx, pool, e); err != nil {
			t.Fatal(err)
		}
		if err := r.SubstituirMembros(ctx, pool, e.ID, e.Membros); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// Com duas equipes, vale a primeira pelo nome.
	b, a := criar("B"), criar("A")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = ANY($1)`, []uuid.UUID{a.ID, b.ID})
	})
	nova, _ := domain.NovaSolicitacao(o, "Sem rede", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	if err := r.AbrirAtribuicao(ctx, pool, sol.ID, tec, &u, inicio); err != nil {
		t.Fatal(err)
	}
	if err := r.AbrirAtribuicao(ctx, pool, sol.ID, sem, &u, inicio); err == nil {
		t.Fatal("dois trechos abertos no mesmo chamado")
	}
	lista, err := r.AtribuicoesDoPeriodo(ctx, pool, true, []uuid.UUID{u}, inicio.Add(time.Hour))
	if err != nil || len(lista) != 1 || lista[0].EquipeID != a.ID || lista[0].Fim != nil || *lista[0].OLAHoras != 4 {
		t.Fatalf("aberto entra mesmo antes do período: %+v %v", lista, err)
	}
	// Fechar antes do início não deixa o trecho negativo.
	if err := r.FecharAtribuicao(ctx, pool, sol.ID, inicio.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := r.FecharAtribuicao(ctx, pool, sol.ID, inicio); err != nil {
		t.Fatal("fechar sem trecho aberto não é erro")
	}
	lista, _ = r.AtribuicoesDoPeriodo(ctx, pool, true, []uuid.UUID{u}, inicio.Add(-time.Minute))
	if len(lista) != 1 || lista[0].Fim == nil || !lista[0].Fim.Equal(inicio) {
		t.Fatalf("fechado: %+v", lista)
	}
	if fora, _ := r.AtribuicoesDoPeriodo(ctx, pool, true, []uuid.UUID{u}, inicio.Add(time.Minute)); len(fora) != 0 {
		t.Fatal("fechado e de antes do período não entra")
	}
	// Quem não está em equipe da unidade fica sem equipe, e o relatório o ignora.
	if err := r.AbrirAtribuicao(ctx, pool, sol.ID, sem, &u, inicio); err != nil {
		t.Fatal(err)
	}
	var equipe *uuid.UUID
	_ = pool.QueryRow(ctx, `SELECT equipe_id FROM atendimento_atribuicoes WHERE solicitacao_id = $1 AND fim IS NULL`, sol.ID).Scan(&equipe)
	if equipe != nil {
		t.Fatal("sem equipe")
	}

	if err := r.AbrirAtribuicao(ctx, dbtest.Fail{}, sol.ID, tec, nil, inicio); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if err := r.FecharAtribuicao(ctx, dbtest.Fail{}, sol.ID, inicio); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.AtribuicoesDoPeriodo(ctx, dbtest.Fail{}, false, nil, inicio); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.AtribuicoesDoPeriodo(ctx, dbtest.ScanFail{}, false, nil, inicio); err == nil {
		t.Fatal("linha ilegível")
	}
	if _, err := r.AtribuicoesDoPeriodo(ctx, dbtest.RowsErr{}, false, nil, inicio); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
}
