package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestAprovacaoReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	quem, chefe, ti := dbtest.User(t, pool), dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Admissão "+uuid.NewString()[:6], "", domain.TipoRequisicao, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})

	if etapas, err := r.EtapasDaOferta(ctx, pool, o.ID); err != nil || len(etapas) != 0 {
		t.Fatalf("sem etapas: %v %v", etapas, err)
	}
	etapas := []domain.EtapaAprovacao{{Ordem: 0, Nome: "Chefia", Aprovadores: []uuid.UUID{chefe}}, {Ordem: 1, Nome: "TI", Aprovadores: []uuid.UUID{ti, chefe}}}
	if err := r.SalvarEtapas(ctx, pool, o.ID, etapas); err != nil {
		t.Fatal(err)
	}
	if lidas, err := r.EtapasDaOferta(ctx, pool, o.ID); err != nil || len(lidas) != 2 || lidas[1].Nome != "TI" || len(lidas[1].Aprovadores) != 2 {
		t.Fatalf("etapas: %+v %v", lidas, err)
	}

	nova, _ := domain.NovaSolicitacao(o, "Admitir Ana", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova.AguardarAprovacao())
	if err != nil || sol.EtapaAtual == nil || *sol.EtapaAtual != 0 || sol.Estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("pedido aguardando: %+v %v", sol, err)
	}
	pendentes, err := r.AguardandoAprovacaoDe(ctx, pool, chefe, 10)
	if err != nil || len(pendentes) != 1 || pendentes[0].ID != sol.ID {
		t.Fatalf("a chefia vê o pedido: %+v %v", pendentes, err)
	}
	if outros, _ := r.AguardandoAprovacaoDe(ctx, pool, ti, 10); len(outros) != 0 {
		t.Fatal("a TI só vê na etapa dela")
	}

	agora := time.Now().UTC().Truncate(time.Microsecond)
	depois, a, _, _ := sol.Decidir(etapas, chefe, true, "", agora)
	if err := r.InsertAprovacao(ctx, pool, a); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateSolicitacao(ctx, pool, depois); err != nil {
		t.Fatal(err)
	}
	if da, _ := r.AguardandoAprovacaoDe(ctx, pool, ti, 10); len(da) != 1 {
		t.Fatal("na segunda etapa, a TI vê")
	}
	lista, err := r.ListAprovacoes(ctx, pool, sol.ID)
	if err != nil || len(lista) != 1 || lista[0].AprovadorNome == "" || lista[0].Decisao != domain.DecisaoAprovada {
		t.Fatalf("decisões: %+v %v", lista, err)
	}

	// O banco também segura as regras.
	recusaSemMotivo := domain.Aprovacao{ID: uuid.New(), SolicitacaoID: sol.ID, EtapaNome: "TI", Decisao: domain.DecisaoRecusa, CreatedAt: agora}
	if err := r.InsertAprovacao(ctx, pool, recusaSemMotivo); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("recusa sem motivo no banco: %v", err)
	}
	semEtapa := depois
	semEtapa.EtapaAtual = nil
	if err := r.UpdateSolicitacao(ctx, pool, semEtapa); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("aguardando sem etapa: %v", err)
	}
	if err := r.SalvarEtapas(ctx, pool, o.ID, nil); err != nil {
		t.Fatal(err)
	}
	if lidas, _ := r.EtapasDaOferta(ctx, pool, o.ID); len(lidas) != 0 {
		t.Fatal("tirar a aprovação")
	}
}

func TestAprovacaoPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	etapas := []domain.EtapaAprovacao{{Nome: "x", Aprovadores: []uuid.UUID{id}}}
	calls := map[string]func(database.DBTX) error{
		"EtapasDaOferta":  func(db database.DBTX) error { _, err := r.EtapasDaOferta(ctx, db, id); return err },
		"SalvarEtapas":    func(db database.DBTX) error { return r.SalvarEtapas(ctx, db, id, etapas) },
		"InsertAprovacao": func(db database.DBTX) error { return r.InsertAprovacao(ctx, db, domain.Aprovacao{}) },
		"ListAprovacoes":  func(db database.DBTX) error { _, err := r.ListAprovacoes(ctx, db, id); return err },
		"AguardandoAprovacaoDe": func(db database.DBTX) error {
			_, err := r.AguardandoAprovacaoDe(ctx, db, id, 10)
			return err
		},
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	for _, name := range []string{"EtapasDaOferta", "ListAprovacoes", "AguardandoAprovacaoDe"} {
		if err := calls[name](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s: linha ilegível", name)
		}
		if err := calls[name](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: fim da leitura: %v", name, err)
		}
	}
	// O DELETE passa e o INSERT da etapa falha.
	seq := &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, {Err: dbtest.ErrInjected}}}
	if err := r.SalvarEtapas(ctx, seq, id, etapas); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("gravar a etapa: %v", err)
	}
}
