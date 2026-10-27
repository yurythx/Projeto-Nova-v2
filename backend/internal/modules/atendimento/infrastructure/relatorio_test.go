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

func TestRelatorioReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	sec := dbtest.Unidade(t, pool)
	var depto uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO unidades (entidade_id, parent_id, nome, sigla, slug)
		SELECT entidade_id, id, 'Departamento', 'DEP', $2 FROM unidades WHERE id = $1 RETURNING id`, sec, "d-"+uuid.NewString()[:12]).Scan(&depto); err != nil {
		t.Fatal(err)
	}
	quem, tec := dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Admissão "+uuid.NewString()[:6], "", domain.TipoRequisicao, nil, &depto, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	ola := 24
	e, _ := domain.NovaEquipe("Rede "+uuid.NewString()[:6], &depto, []uuid.UUID{tec})
	e.OLAHoras = &ola
	if salva, err := r.InsertEquipe(ctx, pool, e); err != nil || *salva.OLAHoras != 24 {
		t.Fatalf("equipe com OLA: %+v %v", salva, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, e.ID)
	})
	sem := e
	sem.OLAHoras = nil
	if atual, err := r.UpdateEquipe(ctx, pool, sem); err != nil || atual.OLAHoras != nil {
		t.Fatalf("tirar o OLA: %+v %v", atual, err)
	}
	if _, err := r.UpdateEquipe(ctx, pool, e); err != nil {
		t.Fatal(err)
	}

	nova, _ := domain.NovaSolicitacao(o, "Admitir Ana", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	desde := time.Now().Add(-time.Hour)
	lista, err := r.ListarDoPeriodo(ctx, pool, true, []uuid.UUID{depto}, desde, 10)
	if err != nil || len(lista) != 1 || lista[0].ID != sol.ID {
		t.Fatalf("do período: %+v %v", lista, err)
	}
	if fora, _ := r.ListarDoPeriodo(ctx, pool, true, []uuid.UUID{depto}, time.Now().Add(time.Hour), 10); len(fora) != 0 {
		t.Fatal("antes do período não entra")
	}
	secs, err := r.SecretariasDe(ctx, pool, []uuid.UUID{depto, sec, uuid.New()})
	if err != nil || len(secs) != 2 || secs[depto].ID != sec || secs[sec].ID != sec || secs[depto].Nome == "" {
		t.Fatalf("secretarias: %+v %v", secs, err)
	}
	cinco := 5
	if _, err := r.InsertAvaliacao(ctx, pool, domain.Avaliacao{ID: uuid.New(), SolicitacaoID: sol.ID, AutorID: quem, Nota: 4, Facilidade: &cinco, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	avs, err := r.AvaliacoesDe(ctx, pool, []uuid.UUID{sol.ID, uuid.New()})
	if err != nil || len(avs) != 1 || avs[sol.ID].Nota != 4 || *avs[sol.ID].Facilidade != 5 {
		t.Fatalf("avaliações: %+v %v", avs, err)
	}
	modelo := []domain.ModeloTarefa{{Ordem: 0, Titulo: "Conta", EquipeID: e.ID, EquipeNome: e.Nome}}
	if err := r.InsertTarefas(ctx, pool, domain.TarefasDoModelo(sol.ID, modelo, time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	ts, err := r.TarefasDoPeriodo(ctx, pool, true, []uuid.UUID{depto}, desde)
	if err != nil || len(ts) != 1 || *ts[0].OLAHoras != 24 || ts[0].Estado != domain.TarefaPendente || ts[0].EquipeNome != e.Nome {
		t.Fatalf("tarefas do período: %+v %v", ts, err)
	}
}

func TestRelatorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	agora := time.Now()
	calls := map[string]func(database.DBTX) error{
		"ListarDoPeriodo": func(db database.DBTX) error {
			_, err := r.ListarDoPeriodo(ctx, db, false, nil, agora, 1)
			return err
		},
		"SecretariasDe": func(db database.DBTX) error { _, err := r.SecretariasDe(ctx, db, []uuid.UUID{id}); return err },
		"AvaliacoesDe":  func(db database.DBTX) error { _, err := r.AvaliacoesDe(ctx, db, []uuid.UUID{id}); return err },
		"TarefasDoPeriodo": func(db database.DBTX) error {
			_, err := r.TarefasDoPeriodo(ctx, db, false, nil, agora)
			return err
		},
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	for _, name := range []string{"SecretariasDe", "AvaliacoesDe", "TarefasDoPeriodo"} {
		if err := calls[name](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s: linha ilegível", name)
		}
		if err := calls[name](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: fim da leitura: %v", name, err)
		}
	}
}
