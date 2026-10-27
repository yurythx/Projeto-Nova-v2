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

func TestMetasXLAReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	quem := dbtest.User(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_xla`) })
	if _, err := pool.Exec(ctx, `DELETE FROM atendimento_xla`); err != nil {
		t.Fatal(err)
	}
	if m, err := r.MetasXLA(ctx, pool); err != nil || m.CSATMin != nil || m.UpdatedAt != nil {
		t.Fatalf("sem metas: %+v %v", m, err)
	}
	nota, horas := 4.2, 48
	agora := time.Now().UTC().Truncate(time.Microsecond)
	if err := r.SalvarMetasXLA(ctx, pool, domain.MetasXLA{CSATMin: &nota, EsperaMaxHoras: &horas}, quem, agora); err != nil {
		t.Fatal(err)
	}
	m, err := r.MetasXLA(ctx, pool)
	if err != nil || *m.CSATMin != 4.2 || *m.EsperaMaxHoras != 48 || m.FacilidadeMin != nil || !m.UpdatedAt.Equal(agora) {
		t.Fatalf("metas: %+v %v", m, err)
	}
	idas := 1.5
	if err := r.SalvarMetasXLA(ctx, pool, domain.MetasXLA{IdasMax: &idas}, quem, agora); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.MetasXLA(ctx, pool); m.CSATMin != nil || *m.IdasMax != 1.5 {
		t.Fatalf("troca a linha inteira: %+v", m)
	}
	fora := 7.0
	if err := r.SalvarMetasXLA(ctx, pool, domain.MetasXLA{CSATMin: &fora}, quem, agora); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("o banco segura a faixa: %v", err)
	}
	if _, err := r.MetasXLA(ctx, dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("banco fora: %v", err)
	}
	if err := r.SalvarMetasXLA(ctx, dbtest.Fail{}, domain.MetasXLA{}, uuid.New(), agora); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("banco fora: %v", err)
	}
}

func TestFacilidadeNaAvaliacao(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	quem := dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Senha "+uuid.NewString()[:6], "", domain.TipoIncidente, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})
	nova, _ := domain.NovaSolicitacao(o, "Senha", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	quatro := 4
	a := domain.Avaliacao{ID: uuid.New(), SolicitacaoID: sol.ID, AutorID: quem, Nota: 5, Facilidade: &quatro, CreatedAt: time.Now().UTC()}
	salva, err := r.InsertAvaliacao(ctx, pool, a)
	if err != nil || salva.Facilidade == nil || *salva.Facilidade != 4 {
		t.Fatalf("avaliação com facilidade: %+v %v", salva, err)
	}
	res, err := r.ResumoAvaliacoes(ctx, pool, true, []uuid.UUID{u}, 1)
	if err != nil || res.Total != 1 || res.TotalFacilidade != 1 || res.SomaFacilidade != 4 {
		t.Fatalf("resumo: %+v %v", res, err)
	}
}
