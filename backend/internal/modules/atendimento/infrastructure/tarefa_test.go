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

func TestTarefasReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	quem, tec := dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Admissão "+uuid.NewString()[:6], "", domain.TipoRequisicao, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	e, _ := domain.NovaEquipe("Rede "+uuid.NewString()[:6], &u, []uuid.UUID{tec})
	if _, err := r.InsertEquipe(ctx, pool, e); err != nil {
		t.Fatal(err)
	}
	if err := r.SubstituirMembros(ctx, pool, e.ID, e.Membros); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, e.ID)
	})

	if m, err := r.ModeloDaOferta(ctx, pool, o.ID); err != nil || len(m) != 0 {
		t.Fatalf("sem modelo: %v %v", m, err)
	}
	modelo := []domain.ModeloTarefa{{Ordem: 0, Titulo: "Criar conta", EquipeID: e.ID}, {Ordem: 1, Titulo: "Liberar VPN", EquipeID: e.ID}}
	if err := r.SalvarModelo(ctx, pool, o.ID, modelo); err != nil {
		t.Fatal(err)
	}
	lido, err := r.ModeloDaOferta(ctx, pool, o.ID)
	if err != nil || len(lido) != 2 || lido[1].Titulo != "Liberar VPN" || lido[0].EquipeNome != e.Nome {
		t.Fatalf("modelo: %+v %v", lido, err)
	}

	nova, _ := domain.NovaSolicitacao(o, "Admitir Ana", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC().Truncate(time.Microsecond)
	if err := r.InsertTarefas(ctx, pool, domain.TarefasDoModelo(sol.ID, lido, agora)); err != nil {
		t.Fatal(err)
	}
	tarefas, err := r.ListTarefas(ctx, pool, sol.ID)
	if err != nil || len(tarefas) != 2 || tarefas[0].EquipeNome != e.Nome || tarefas[0].Estado != domain.TarefaPendente {
		t.Fatalf("tarefas: %+v %v", tarefas, err)
	}
	if _, err := r.GetTarefa(ctx, pool, uuid.New(), tarefas[0].ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("tarefa de outro pedido: %v", err)
	}
	feita, _ := tarefas[0].Fazer(tec, true, true, "ok", agora)
	if err := r.UpdateTarefa(ctx, pool, feita); err != nil {
		t.Fatal(err)
	}
	if lida, err := r.GetTarefa(ctx, pool, sol.ID, feita.ID); err != nil || lida.Estado != domain.TarefaConcluida || lida.FeitaPorNome == "" {
		t.Fatalf("tarefa feita: %+v %v", lida, err)
	}
	// O banco também segura a dispensa sem motivo.
	semMotivo := tarefas[1]
	semMotivo.Estado, semMotivo.FeitaEm = domain.TarefaDispensada, &agora
	if err := r.UpdateTarefa(ctx, pool, semMotivo); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("dispensa sem motivo no banco: %v", err)
	}
	if m, err := r.MembroDe(ctx, pool, tec, []uuid.UUID{e.ID, uuid.New()}); err != nil || len(m) != 1 || m[0] != e.ID {
		t.Fatalf("membro: %v %v", m, err)
	}
	if m, _ := r.MembroDe(ctx, pool, quem, []uuid.UUID{e.ID}); len(m) != 0 {
		t.Fatal("quem pediu não é da equipe")
	}
	if err := r.SalvarModelo(ctx, pool, o.ID, nil); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.ModeloDaOferta(ctx, pool, o.ID); len(m) != 0 {
		t.Fatal("tirar o modelo")
	}
	if depois, _ := r.ListTarefas(ctx, pool, sol.ID); len(depois) != 2 {
		t.Fatal("mudar o modelo não mexe nas tarefas criadas")
	}
}

func TestTarefasPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	modelo := []domain.ModeloTarefa{{Titulo: "x", EquipeID: id}}
	calls := map[string]func(database.DBTX) error{
		"ModeloDaOferta": func(db database.DBTX) error { _, err := r.ModeloDaOferta(ctx, db, id); return err },
		"SalvarModelo":   func(db database.DBTX) error { return r.SalvarModelo(ctx, db, id, modelo) },
		"InsertTarefas":  func(db database.DBTX) error { return r.InsertTarefas(ctx, db, []domain.Tarefa{{}}) },
		"ListTarefas":    func(db database.DBTX) error { _, err := r.ListTarefas(ctx, db, id); return err },
		"GetTarefa":      func(db database.DBTX) error { _, err := r.GetTarefa(ctx, db, id, id); return err },
		"UpdateTarefa":   func(db database.DBTX) error { return r.UpdateTarefa(ctx, db, domain.Tarefa{}) },
		"MembroDe":       func(db database.DBTX) error { _, err := r.MembroDe(ctx, db, id, []uuid.UUID{id}); return err },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	for _, name := range []string{"ModeloDaOferta", "ListTarefas", "MembroDe"} {
		if err := calls[name](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s: linha ilegível", name)
		}
		if err := calls[name](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: fim da leitura: %v", name, err)
		}
	}
	seq := &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, {Err: dbtest.ErrInjected}}}
	if err := r.SalvarModelo(ctx, seq, id, modelo); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("gravar a linha do modelo: %v", err)
	}
}
