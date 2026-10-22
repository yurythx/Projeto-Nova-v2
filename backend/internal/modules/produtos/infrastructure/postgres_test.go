package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/sismarq/internal/modules/produtos/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/database/dbtest"
)

type rowErr struct{ err error }

func (r rowErr) Scan(...any) error { return r.err }

type rowOK struct{}

func (rowOK) Scan(...any) error { return nil }

type queryDB struct {
	rows pgx.Rows
	err  error
	row  pgx.Row
}

func (q queryDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, q.err
}
func (q queryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return q.rows, q.err
}
func (q queryDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if q.row != nil {
		return q.row
	}
	return rowErr{q.err}
}

type okRows struct{ left int }

func (r *okRows) Close()                                       {}
func (r *okRows) Err() error                                   { return nil }
func (r *okRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *okRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *okRows) Scan(...any) error                            { return nil }
func (r *okRows) Values() ([]any, error)                       { return nil, nil }
func (r *okRows) RawValues() [][]byte                          { return nil }
func (r *okRows) Conn() *pgx.Conn                              { return nil }
func (r *okRows) Next() bool {
	if r.left == 0 {
		return false
	}
	r.left--
	return true
}

func TestRepositorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	p := domain.Produto{ID: id, Nome: "Protocolo", Estagio: "operacao"}
	calls := map[string]func(db database.DBTX) error{
		"List":   func(db database.DBTX) error { _, err := r.List(ctx, db, domain.Filtro{}); return err },
		"Get":    func(db database.DBTX) error { _, err := r.Get(ctx, db, id); return err },
		"Insert": func(db database.DBTX) error { _, err := r.Insert(ctx, db, p); return err },
		"Update": func(db database.DBTX) error { _, err := r.Update(ctx, db, p); return err },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := calls["List"](dbtest.ScanFail{}); err == nil {
		t.Error("List com linha ilegível deveria falhar")
	}
	if err := calls["List"](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("List com erro no fim da leitura: %v", err)
	}
	if _, err := r.Get(ctx, queryDB{row: rowErr{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("Get sem linha: %v", err)
	}
	if _, err := r.Insert(ctx, queryDB{row: rowErr{&pgconn.PgError{Code: "23503"}}}, p); !errors.Is(err, domain.ErrUnidadeInexistente) {
		t.Errorf("unidade inexistente: %v", err)
	}
	if _, err := r.Update(ctx, queryDB{row: rowErr{&pgconn.PgError{Code: "23514"}}}, p); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Errorf("check: %v", err)
	}
	if _, err := r.Update(ctx, queryDB{row: rowErr{pgx.ErrNoRows}}, idProduto(p)); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("Update sem linha: %v", err)
	}
	if _, err := r.Insert(ctx, queryDB{row: rowOK{}}, p); err != nil {
		t.Errorf("Insert que lê a linha: %v", err)
	}
	lista, err := r.List(ctx, queryDB{rows: &okRows{left: 1}}, domain.Filtro{Restrito: true})
	if err != nil || len(lista) != 1 {
		t.Fatalf("List com uma linha: %v %+v", err, lista)
	}
	vazia, err := r.List(ctx, queryDB{rows: dbtest.NoRows()}, domain.Filtro{})
	if err != nil || len(vazia) != 0 {
		t.Fatalf("List vazia: %v %v", err, vazia)
	}
	if _, err := r.Get(ctx, queryDB{row: rowOK{}}, id); err != nil {
		t.Errorf("Get que lê a linha: %v", err)
	}
}

func idProduto(p domain.Produto) domain.Produto { return p }

func TestPersistenciaReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	p, err := domain.Novo("Protocolo "+uuid.NewString()[:8], "desc", "operacao", "", "", &u)
	if err != nil {
		t.Fatal(err)
	}
	salvo, err := r.Insert(ctx, pool, p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM produtos WHERE id = $1`, salvo.ID)
	})
	if salvo.Nome != p.Nome || salvo.UnidadeID == nil || *salvo.UnidadeID != u {
		t.Fatalf("gravou diferente: %+v", salvo)
	}
	lista, err := r.List(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}})
	if err != nil || len(lista) != 1 || lista[0].ID != salvo.ID {
		t.Fatalf("lista da unidade: %v %+v", err, lista)
	}
	outra, err := r.List(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{uuid.New()}})
	if err != nil || len(outra) != 0 {
		t.Fatalf("outra unidade não vê: %v %+v", err, outra)
	}
	salvo.Estagio = "suporte"
	salvo.Nome = "Protocolo web"
	atual, err := r.Update(ctx, pool, salvo)
	if err != nil || atual.Estagio != "suporte" || atual.Nome != "Protocolo web" {
		t.Fatalf("update: %+v %v", atual, err)
	}
	if _, err := r.Get(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("inexistente: %v", err)
	}
	fantasma := salvo
	fantasma.ID = uuid.New()
	if _, err := r.Update(ctx, pool, fantasma); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("update inexistente: %v", err)
	}
	ruim := salvo
	ruim.UnidadeID = &[]uuid.UUID{uuid.New()}[0]
	if _, err := r.Update(ctx, pool, ruim); !errors.Is(err, domain.ErrUnidadeInexistente) {
		t.Fatalf("unidade: %v", err)
	}
}

func TestMapDBSemErro(t *testing.T) {
	if mapDB(nil) != nil {
		t.Fatal("sem erro, nada a traduzir")
	}
}
