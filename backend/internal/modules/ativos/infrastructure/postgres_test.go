package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type linha struct{ err error }

func (l linha) Scan(...any) error { return l.err }

type umaLinha struct{ err error }

func (u umaLinha) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, u.err
}
func (u umaLinha) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, u.err }
func (u umaLinha) QueryRow(context.Context, string, ...any) pgx.Row        { return linha{u.err} }

func TestRepositorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	a := domain.Ativo{ID: uuid.New()}
	calls := map[string]func(database.DBTX) error{
		"List":         func(db database.DBTX) error { _, err := r.List(ctx, db, domain.Filtro{}); return err },
		"Get":          func(db database.DBTX) error { _, err := r.Get(ctx, db, a.ID); return err },
		"Insert":       func(db database.DBTX) error { _, err := r.Insert(ctx, db, a); return err },
		"Update":       func(db database.DBTX) error { _, err := r.Update(ctx, db, a); return err },
		"ListRelacoes": func(db database.DBTX) error { _, err := r.ListRelacoes(ctx, db, a.ID); return err },
		"InsertRelacao": func(db database.DBTX) error {
			return r.InsertRelacao(ctx, db, domain.Relacao{ID: a.ID, OrigemID: a.ID, DestinoID: uuid.New(), Tipo: "conectado"})
		},
		"DeleteRelacao":    func(db database.DBTX) error { return r.DeleteRelacao(ctx, db, a.ID, a.ID) },
		"FechaCiclo":       func(db database.DBTX) error { _, err := r.FechaCiclo(ctx, db, a.ID, uuid.New()); return err },
		"PorIdentificador": func(db database.DBTX) error { _, err := r.PorIdentificador(ctx, db, a.Identificador); return err },
		"InsertCredencial": func(db database.DBTX) error {
			return r.InsertCredencial(ctx, db, domain.Credencial{ID: a.ID, UnidadeID: a.ID, TokenHash: []byte("x")})
		},
		"ListCredenciais":   func(db database.DBTX) error { _, err := r.ListCredenciais(ctx, db, domain.Filtro{}); return err },
		"RevogarCredencial": func(db database.DBTX) error { return r.RevogarCredencial(ctx, db, a.ID, domain.Filtro{}) },
		"PorHash":           func(db database.DBTX) error { _, err := r.PorHash(ctx, db, []byte("x")); return err },
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
	casos := map[error]error{
		pgx.ErrNoRows: domain.ErrNaoEncontrado,
		&pgconn.PgError{Code: "23505", ConstraintName: "uq_ativos_identificador"}: domain.ErrIdentificadorRepetido,
		&pgconn.PgError{Code: "23503", ConstraintName: "ativos_unidade_id_fkey"}:  domain.ErrUnidadeInexistente,
		&pgconn.PgError{Code: "23514", ConstraintName: "ativos_tipo_chk"}:         domain.ErrEntradaInvalida,
	}
	for causa, want := range casos {
		if _, err := r.Insert(ctx, umaLinha{causa}, a); !errors.Is(err, want) {
			t.Errorf("%v: %v", causa, err)
		}
	}
	if mapDB(nil) != nil {
		t.Fatal("sem erro")
	}
}

func TestPersistenciaReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	produto := uuid.New()
	a, _ := domain.Novo(domain.Entrada{Nome: "Servidor", Tipo: "hardware", Estado: "em_uso", Identificador: "SRV-" + uuid.NewString()[:8],
		UnidadeID: &u, ProdutoID: &produto})
	salvo, err := r.Insert(ctx, pool, a)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM ativos WHERE id = $1`, salvo.ID) })
	if _, err := r.Insert(ctx, pool, domain.Ativo{ID: uuid.New(), Nome: "Dup", Tipo: "hardware", Estado: "em_uso", Identificador: a.Identificador}); !errors.Is(err, domain.ErrIdentificadorRepetido) {
		t.Fatalf("identificador repetido: %v", err)
	}
	lista, err := r.List(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}, ProdutoID: &produto})
	if err != nil || len(lista) != 1 {
		t.Fatalf("por produto: %v %v", lista, err)
	}
	salvo.Estado = "desativado"
	if got, err := r.Update(ctx, pool, salvo); err != nil || got.Estado != "desativado" || got.Origem != domain.OrigemManual {
		t.Fatalf("update: %+v %v", got, err)
	}
	salvo.Inventario = &domain.Inventario{Sistema: "linux amd64", Processadores: 2, Discos: []domain.Disco{{Nome: "sda", Bytes: 10}}}
	if got, err := r.Update(ctx, pool, salvo); err != nil || got.Inventario == nil || got.Inventario.Processadores != 2 || len(got.Inventario.Discos) != 1 {
		t.Fatalf("inventário: %+v %v", got.Inventario, err)
	}
	if achado, err := r.PorIdentificador(ctx, pool, a.Identificador); err != nil || achado.ID != salvo.ID || achado.Inventario == nil {
		t.Fatalf("por identificador: %+v %v", achado, err)
	}
	if _, err := r.PorIdentificador(ctx, pool, "nao-existe"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
}
