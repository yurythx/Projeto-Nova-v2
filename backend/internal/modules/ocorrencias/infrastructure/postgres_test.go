package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestOcorrenciasReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	quem := dbtest.User(t, pool)
	agora := time.Now().UTC().Truncate(time.Microsecond)
	var criadas []uuid.UUID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ocorrencias WHERE id = ANY($1)`, criadas)
	})
	grava := func(o domain.Ocorrencia, a domain.Atualizacao) {
		t.Helper()
		if err := r.Insert(ctx, pool, o); err != nil {
			t.Fatal(err)
		}
		if err := r.InsertAtualizacao(ctx, pool, a); err != nil {
			t.Fatal(err)
		}
		criadas = append(criadas, o.ID)
	}

	o, a, _ := domain.Declarar("Protocolo fora", []string{"Protocolo"}, domain.ImpactoFora, "Investigando.", nil, quem, agora)
	grava(o, a)
	lida, err := r.Get(ctx, pool, o.ID)
	if err != nil || lida.Titulo != "Protocolo fora" || lida.CoordenadorNome == "" || len(lida.Atualizacoes) != 1 ||
		lida.Atualizacoes[0].AutorNome == "" || !lida.IniciadaEm.Equal(agora) {
		t.Fatalf("leitura: %+v %v", lida, err)
	}

	resolvida, _, _ := lida.Atualizar(domain.EstadoResolvido, domain.ImpactoFora, "Voltou.", quem, agora.Add(time.Minute))
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Travar(ctx, tx, o.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.Update(ctx, tx, resolvida); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	velha, va, _ := domain.Declarar("Antiga", []string{"Rede"}, domain.ImpactoParcial, "Fora.", nil, quem, agora.Add(-30*24*time.Hour))
	velha, _, _ = velha.Atualizar(domain.EstadoResolvido, domain.ImpactoParcial, "Voltou.", quem, agora.Add(-29*24*time.Hour))
	grava(velha, va)
	aberta, aa, _ := domain.Declarar("E-mail lento", []string{"E-mail"}, domain.ImpactoDegradado, "Lento.", nil, uuid.Nil, agora)
	grava(aberta, aa)

	pagina, err := r.ParaPagina(ctx, pool, agora.Add(-domain.JanelaRecentes))
	if err != nil {
		t.Fatal(err)
	}
	vistos := map[uuid.UUID]bool{}
	for _, p := range pagina {
		vistos[p.ID] = true
	}
	if !vistos[o.ID] || !vistos[aberta.ID] || vistos[velha.ID] {
		t.Fatalf("página: a resolvida há 30 dias não aparece: %+v", vistos)
	}
	painel, err := r.List(ctx, pool, 1000)
	if err != nil || len(painel) < 3 || painel[0].Resolvida() {
		t.Fatalf("painel: as abertas vêm primeiro: %v", err)
	}

	// O banco também segura as regras.
	if _, err := r.Get(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrada) {
		t.Fatalf("inexistente: %v", err)
	}
	if _, err := r.Travar(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrada) {
		t.Fatalf("travar inexistente: %v", err)
	}
	if err := r.Update(ctx, pool, domain.Ocorrencia{ID: uuid.New(), Impacto: domain.ImpactoFora, Estado: domain.EstadoInvestigando}); !errors.Is(err, domain.ErrNaoEncontrada) {
		t.Fatalf("atualizar inexistente: %v", err)
	}
	semChamado, _, _ := domain.Declarar("X", []string{"Y"}, domain.ImpactoFora, "z", ptr(uuid.New()), quem, agora)
	if err := r.Insert(ctx, pool, semChamado); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("chamado inexistente: %v", err)
	}
	semPessoa, _, _ := domain.Declarar("X", []string{"Y"}, domain.ImpactoFora, "z", nil, uuid.New(), agora)
	if err := r.Insert(ctx, pool, semPessoa); !errors.Is(err, domain.ErrCoordenadorInvalido) {
		t.Fatalf("coordenador inexistente: %v", err)
	}
	invalida := aberta
	invalida.ID, invalida.Impacto = uuid.New(), "catastrofe"
	if err := r.Insert(ctx, pool, invalida); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("impacto fora da regra: %v", err)
	}
}

func ptr(id uuid.UUID) *uuid.UUID { return &id }

func TestOcorrenciasPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	calls := map[string]func(database.DBTX) error{
		"Insert":            func(db database.DBTX) error { return r.Insert(ctx, db, domain.Ocorrencia{}) },
		"Update":            func(db database.DBTX) error { return r.Update(ctx, db, domain.Ocorrencia{}) },
		"InsertAtualizacao": func(db database.DBTX) error { return r.InsertAtualizacao(ctx, db, domain.Atualizacao{}) },
		"Travar":            func(db database.DBTX) error { _, err := r.Travar(ctx, db, id); return err },
		"Get":               func(db database.DBTX) error { _, err := r.Get(ctx, db, id); return err },
		"List":              func(db database.DBTX) error { _, err := r.List(ctx, db, 10); return err },
		"ParaPagina":        func(db database.DBTX) error { _, err := r.ParaPagina(ctx, db, time.Now()); return err },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := calls["List"](dbtest.ScanFail{}); err == nil {
		t.Error("linha ilegível")
	}
	if err := calls["List"](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("fim da leitura: %v", err)
	}
	// Ocorrência lida, linha do tempo falha (consulta, leitura e fim).
	umaLinha := dbtest.QueryResult{Rows: &linhas{n: 1}}
	for nome, segunda := range map[string]dbtest.QueryResult{
		"consulta":       {Err: dbtest.ErrInjected},
		"linha ilegível": {Rows: dbtest.BadRows()},
		"fim da leitura": {Rows: dbtest.ErrRows()},
	} {
		seq := &dbtest.Seq{Queries: []dbtest.QueryResult{umaLinha, segunda}}
		if _, err := r.List(ctx, seq, 10); err == nil {
			t.Errorf("linha do tempo, %s", nome)
		}
		umaLinha = dbtest.QueryResult{Rows: &linhas{n: 1}}
	}
	vazio := &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: dbtest.NoRows()}}}
	if lista, err := r.List(ctx, vazio, 10); err != nil || len(lista) != 0 {
		t.Errorf("painel vazio: %v %v", lista, err)
	}
}

// linhas entrega n ocorrências legíveis (Scan sem erro), para a falha
// cair na consulta seguinte.
type linhas struct{ n int }

func (r *linhas) Close()                                       {}
func (r *linhas) Err() error                                   { return nil }
func (r *linhas) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *linhas) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *linhas) Scan(dest ...any) error {
	if id, ok := dest[0].(*uuid.UUID); ok {
		*id = uuid.New()
	}
	return nil
}
func (r *linhas) Values() ([]any, error) { return nil, nil }
func (r *linhas) RawValues() [][]byte    { return nil }
func (r *linhas) Conn() *pgx.Conn        { return nil }
func (r *linhas) Next() bool {
	if r.n == 0 {
		return false
	}
	r.n--
	return true
}

type linhaLegivel struct{}

func (linhaLegivel) Scan(...any) error { return nil }

func TestGetComLinhaDoTempoFalhando(t *testing.T) {
	seq := &dbtest.Seq{Row: linhaLegivel{}, Queries: []dbtest.QueryResult{{Err: dbtest.ErrInjected}}}
	if _, err := NewRepository().Get(context.Background(), seq, uuid.New()); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("linha do tempo: %v", err)
	}
}
