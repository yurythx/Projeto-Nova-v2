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

func TestEventosReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	quem := dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Monitoramento "+uuid.NewString()[:6], "", domain.TipoIncidente, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	nova, _ := domain.NovaSolicitacao(o, "Disco cheio", "", domain.ContaMonitoramento)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatalf("a conta de sistema existe e pode abrir: %v", err)
	}
	fp := "fp-" + uuid.NewString()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_eventos_config`)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE id = $1`, sol.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})

	if oferta, err := r.OfertaDosEventos(ctx, pool); err != nil || oferta != nil {
		t.Fatalf("sem configuração: %v %v", oferta, err)
	}
	agora := time.Now().UTC().Truncate(time.Microsecond)
	if err := r.SalvarOfertaDosEventos(ctx, pool, &o.ID, quem, agora); err != nil {
		t.Fatal(err)
	}
	if oferta, err := r.OfertaDosEventos(ctx, pool); err != nil || oferta == nil || *oferta != o.ID {
		t.Fatalf("configurada: %v %v", oferta, err)
	}
	if err := r.SalvarOfertaDosEventos(ctx, pool, nil, quem, agora); err != nil {
		t.Fatal(err)
	}
	if oferta, _ := r.OfertaDosEventos(ctx, pool); oferta != nil {
		t.Fatal("limpar a oferta")
	}

	if _, err := r.TravarEvento(ctx, pool, fp); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("alerta novo: %v", err)
	}
	e := domain.EventoVinculado{Fingerprint: fp, SolicitacaoID: sol.ID, Alerta: "NovaDiscoCritico", UltimoDisparo: agora}
	if err := r.SalvarEvento(ctx, pool, e); err != nil {
		t.Fatal(err)
	}
	e.UltimoDisparo = agora.Add(time.Hour)
	if err := r.SalvarEvento(ctx, pool, e); err != nil {
		t.Fatalf("regravar o mesmo alerta: %v", err)
	}
	lido, err := r.TravarEvento(ctx, pool, fp)
	if err != nil || lido.SolicitacaoID != sol.ID || !lido.UltimoDisparo.Equal(agora.Add(time.Hour)) {
		t.Fatalf("vínculo: %+v %v", lido, err)
	}
	if err := r.SalvarEvento(ctx, pool, domain.EventoVinculado{Fingerprint: "fp-" + uuid.NewString(), SolicitacaoID: uuid.New(), Alerta: "x", UltimoDisparo: agora}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("chamado inexistente: %v", err)
	}
}

func TestEventosPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	calls := map[string]func(database.DBTX) error{
		"OfertaDosEventos": func(db database.DBTX) error { _, err := r.OfertaDosEventos(ctx, db); return err },
		"SalvarOfertaDosEventos": func(db database.DBTX) error {
			return r.SalvarOfertaDosEventos(ctx, db, nil, uuid.New(), time.Now())
		},
		"TravarEvento": func(db database.DBTX) error { _, err := r.TravarEvento(ctx, db, "f"); return err },
		"SalvarEvento": func(db database.DBTX) error { return r.SalvarEvento(ctx, db, domain.EventoVinculado{}) },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	// A trava passa e a leitura falha.
	seq := &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK}}
	if _, err := r.TravarEvento(ctx, seq, "f"); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("leitura depois da trava: %v", err)
	}
}
