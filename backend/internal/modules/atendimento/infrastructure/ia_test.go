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

func TestSugestoesDaIAReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	tec, quem := dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Sistema "+uuid.NewString()[:6], "", domain.TipoIncidente, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	nova, _ := domain.NovaSolicitacao(o, "Protocolo parado", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE id = $1`, sol.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})
	agora := time.Now().UTC().Truncate(time.Microsecond)

	classe := domain.Sugestao{ID: uuid.New(), SolicitacaoID: sol.ID, Tipo: domain.SugestaoClassificacao, Conexao: "IA local",
		Classificacao: &domain.ClassificacaoSugerida{Impacto: 3, Urgencia: 2, Motivo: "vários setores"}, PedidaPor: &tec, CreatedAt: agora}
	resumo := domain.Sugestao{ID: uuid.New(), SolicitacaoID: sol.ID, Tipo: domain.SugestaoResumo, Conexao: "IA local",
		Resumo: "O protocolo parou.", PedidaPor: &tec, CreatedAt: agora}
	for _, s := range []domain.Sugestao{classe, resumo} {
		if err := r.InsertSugestao(ctx, pool, s); err != nil {
			t.Fatal(err)
		}
	}
	lida, err := r.TravarSugestao(ctx, pool, classe.ID)
	if err != nil || lida.Classificacao == nil || lida.Classificacao.Impacto != 3 || lida.Decisao != nil || *lida.PedidaPor != tec || !lida.CreatedAt.Equal(agora) {
		t.Fatalf("leitura: %+v %v", lida, err)
	}
	if r2, _ := r.TravarSugestao(ctx, pool, resumo.ID); r2.Resumo != "O protocolo parou." || r2.Classificacao != nil {
		t.Fatalf("resumo: %+v", r2)
	}
	if _, err := r.TravarSugestao(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("inexistente: %v", err)
	}

	decidida, _ := lida.Decidir(true, tec, agora)
	if err := r.DecidirSugestao(ctx, pool, decidida); err != nil {
		t.Fatal(err)
	}
	// O banco também só aceita uma decisão, mesmo se a aplicação errar.
	if err := r.DecidirSugestao(ctx, pool, decidida); !errors.Is(err, domain.ErrSugestaoDecidida) {
		t.Fatalf("segunda decisão: %v", err)
	}

	ind, err := r.IndicadoresIA(ctx, pool, agora.Add(-time.Hour), true, []uuid.UUID{u})
	if err != nil || len(ind) != 2 {
		t.Fatalf("indicadores: %+v %v", ind, err)
	}
	if ind[0] != (domain.IndicadorIA{Tipo: "classificacao", Total: 1, Aceitas: 1}) || ind[1] != (domain.IndicadorIA{Tipo: "resumo", Total: 1, Pendentes: 1}) {
		t.Fatalf("contagem: %+v", ind)
	}
	// Outra unidade não vê as sugestões desta, mas as duas tarefas aparecem.
	vazio, err := r.IndicadoresIA(ctx, pool, agora.Add(-time.Hour), true, []uuid.UUID{dbtest.Unidade(t, pool)})
	if err != nil || len(vazio) != 2 || vazio[0].Total != 0 || vazio[1].Total != 0 {
		t.Fatalf("fora do escopo: %+v %v", vazio, err)
	}

	// Pessoa e chamado inexistentes são recusados pelo banco.
	semChamado := resumo
	semChamado.ID, semChamado.SolicitacaoID = uuid.New(), uuid.New()
	if err := r.InsertSugestao(ctx, pool, semChamado); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("chamado inexistente: %v", err)
	}

	// Conteúdo gravado fora do formato não vira sugestão.
	estranha := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO atendimento_ia_sugestoes (id, solicitacao_id, tipo, conexao, conteudo)
		VALUES ($1, $2, 'resumo', 'x', '"texto solto"')`, estranha, sol.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.TravarSugestao(ctx, pool, estranha); err == nil {
		t.Fatal("conteúdo ilegível")
	}
}

func TestSugestoesDaIAPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	calls := map[string]func(database.DBTX) error{
		"InsertSugestao": func(db database.DBTX) error { return r.InsertSugestao(ctx, db, domain.Sugestao{}) },
		"TravarSugestao": func(db database.DBTX) error { _, err := r.TravarSugestao(ctx, db, uuid.New()); return err },
		"DecidirSugestao": func(db database.DBTX) error {
			return r.DecidirSugestao(ctx, db, domain.Sugestao{})
		},
		"IndicadoresIA": func(db database.DBTX) error {
			_, err := r.IndicadoresIA(ctx, db, time.Now(), false, nil)
			return err
		},
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := calls["IndicadoresIA"](dbtest.ScanFail{}); err == nil {
		t.Error("linha ilegível")
	}
	if err := calls["IndicadoresIA"](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("fim da leitura: %v", err)
	}
}
