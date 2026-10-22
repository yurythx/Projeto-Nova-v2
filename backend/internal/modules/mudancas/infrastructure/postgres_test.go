package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/sismarq/internal/modules/mudancas/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/database/dbtest"
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
	id := uuid.New()
	m := domain.Mudanca{ID: id}
	calls := map[string]func(database.DBTX) error{
		"List":            func(db database.DBTX) error { _, err := r.List(ctx, db, domain.Filtro{}); return err },
		"Get":             func(db database.DBTX) error { _, err := r.Get(ctx, db, id); return err },
		"Travar":          func(db database.DBTX) error { return r.Travar(ctx, db, id) },
		"Insert":          func(db database.DBTX) error { _, err := r.Insert(ctx, db, m); return err },
		"Update":          func(db database.DBTX) error { return r.Update(ctx, db, m) },
		"Aprovacoes":      func(db database.DBTX) error { _, err := r.Aprovacoes(ctx, db, id); return err },
		"InsertAprovacao": func(db database.DBTX) error { return r.InsertAprovacao(ctx, db, id, domain.Aprovacao{}) },
		"Vinculos":        func(db database.DBTX) error { _, err := r.Vinculos(ctx, db, id); return err },
		"Vincular":        func(db database.DBTX) error { _, err := r.Vincular(ctx, db, id, domain.Vinculo{}, id); return err },
		"Desvincular":     func(db database.DBTX) error { _, err := r.Desvincular(ctx, db, id, "ativo", id); return err },
	}
	for nome, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	for _, nome := range []string{"List", "Aprovacoes", "Vinculos"} {
		if err := calls[nome](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s com linha ilegível", nome)
		}
		if err := calls[nome](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s no fim da leitura: %v", nome, err)
		}
	}
	if err := r.Update(ctx, dbtest.ScanFail{}, m); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("update sem linha: %v", err)
	}
	for causa, want := range map[error]error{
		pgx.ErrNoRows:                  domain.ErrNaoEncontrado,
		&pgconn.PgError{Code: "23503"}: domain.ErrUnidadeInexistente,
		&pgconn.PgError{Code: "23514"}: domain.ErrEntradaInvalida,
	} {
		if _, err := r.Get(ctx, umaLinha{causa}, id); !errors.Is(err, want) {
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
	quem := dbtest.User(t, pool)
	ini := time.Now().Add(time.Hour)
	fim := ini.Add(time.Hour)
	nova, _ := domain.Nova(domain.Entrada{Titulo: "Trocar fonte", Tipo: domain.TipoNormal, Risco: "baixo", UnidadeID: &u,
		InicioPlanejado: &ini, FimPlanejado: &fim}, quem)
	m, err := r.Insert(ctx, pool, nova)
	if err != nil || m.Numero == 0 || m.SolicitanteNome == "" {
		t.Fatalf("insert: %+v %v", m, err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM mudancas WHERE id = $1`, m.ID) })
	if err := r.InsertAprovacao(ctx, pool, m.ID, domain.Aprovacao{ID: uuid.New(), AprovadorID: &quem, Decisao: domain.DecisaoAprovada, Em: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if as, err := r.Aprovacoes(ctx, pool, m.ID); err != nil || len(as) != 1 || as[0].Aprovador == "" {
		t.Fatalf("aprovações: %v %v", as, err)
	}
	alvo := uuid.New()
	if ok, err := r.Vincular(ctx, pool, m.ID, domain.Vinculo{Tipo: domain.VinculoAtivo, AlvoID: alvo, Rotulo: "Ativo · X"}, quem); err != nil || !ok {
		t.Fatal(err)
	}
	if lista, err := r.List(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}, AlvoTipo: domain.VinculoAtivo, AlvoID: &alvo}); err != nil || len(lista) != 1 {
		t.Fatalf("lista pelo alvo: %v %v", lista, err)
	}
	if vs, err := r.Vinculos(ctx, pool, m.ID); err != nil || len(vs) != 1 {
		t.Fatalf("vínculos: %v %v", vs, err)
	}
	if ok, err := r.Desvincular(ctx, pool, m.ID, domain.VinculoAtivo, alvo); err != nil || !ok {
		t.Fatal(err)
	}
	if err := r.Travar(ctx, pool, m.ID); err != nil {
		t.Fatal(err)
	}
	m.Estado = domain.EmAvaliacao
	if err := r.Update(ctx, pool, m); err != nil {
		t.Fatal(err)
	}
}
