package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestRepositorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	item := domain.Selado{Segredo: domain.Segredo{
		ID: id, Nome: "Correio", Usuario: "ana", Versao: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, Dek: "dek", Cifra: "cifra"}
	calls := map[string]func(database.DBTX) error{
		"List":   func(db database.DBTX) error { _, err := r.List(ctx, db, domain.Filtro{}); return err },
		"Get":    func(db database.DBTX) error { _, err := r.Get(ctx, db, id, domain.Filtro{}); return err },
		"Insert": func(db database.DBTX) error { return r.Insert(ctx, db, item) },
		"Update": func(db database.DBTX) error { return r.Update(ctx, db, item, domain.Filtro{}) },
		"Delete": func(db database.DBTX) error { return r.Delete(ctx, db, id, domain.Filtro{}) },
	}
	for nome, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := r.List(ctx, dbtest.ScanFail{}, domain.Filtro{}); err == nil {
		t.Error("linha ilegível")
	}
	if _, err := r.List(ctx, dbtest.RowsErr{}, domain.Filtro{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("fim da leitura: %v", err)
	}
	if err := r.Delete(ctx, dbtest.ScanFail{}, id, domain.Filtro{}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("exclusão sem linha: %v", err)
	}
	casos := map[error]bool{
		pgx.ErrNoRows:                  true,
		&pgconn.PgError{Code: "23503"}: false,
		&pgconn.PgError{Code: "23514"}: false,
	}
	for causa, naoEncontrado := range casos {
		err := r.Insert(ctx, linha{causa}, item)
		if naoEncontrado {
			if !errors.Is(err, domain.ErrNaoEncontrado) {
				t.Errorf("%v: %v", causa, err)
			}
			continue
		}
		var invalida domain.EntradaInvalida
		if !errors.As(err, &invalida) {
			t.Errorf("%v: %v", causa, err)
		}
	}
	if mapDB(nil) != nil {
		t.Fatal("sem erro")
	}
}

type linha struct{ err error }

func (l linha) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, l.err
}
func (l linha) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, l.err }
func (l linha) QueryRow(context.Context, string, ...any) pgx.Row        { return linhaRow{l.err} }

type linhaRow struct{ err error }

func (l linhaRow) Scan(...any) error { return l.err }
