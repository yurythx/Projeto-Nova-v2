package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/sismarq/internal/modules/problemas/domain"
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
	p := domain.Problema{ID: id}
	calls := map[string]func(database.DBTX) error{
		"List":   func(db database.DBTX) error { _, err := r.List(ctx, db, domain.Filtro{}); return err },
		"Get":    func(db database.DBTX) error { _, err := r.Get(ctx, db, id); return err },
		"Travar": func(db database.DBTX) error { return r.Travar(ctx, db, id) },
		"Insert": func(db database.DBTX) error { _, err := r.Insert(ctx, db, p); return err },
		"Update": func(db database.DBTX) error { return r.Update(ctx, db, p) },
		"Incidentes": func(db database.DBTX) error {
			_, err := r.Incidentes(ctx, db, id)
			return err
		},
		"VincularIncidente": func(db database.DBTX) error {
			_, err := r.VincularIncidente(ctx, db, id, domain.IncidenteVinculado{}, id)
			return err
		},
		"DesvincularIncidente": func(db database.DBTX) error { _, err := r.DesvincularIncidente(ctx, db, id, id); return err },
		"AtivosDe":             func(db database.DBTX) error { _, err := r.AtivosDe(ctx, db, id); return err },
		"VincularAtivo":        func(db database.DBTX) error { _, err := r.VincularAtivo(ctx, db, id, id, id); return err },
		"DesvincularAtivo":     func(db database.DBTX) error { _, err := r.DesvincularAtivo(ctx, db, id, id); return err },
		"JaVinculados":         func(db database.DBTX) error { _, err := r.JaVinculados(ctx, db, []uuid.UUID{id}); return err },
	}
	for nome, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	for _, nome := range []string{"List", "Incidentes", "AtivosDe", "JaVinculados"} {
		if err := calls[nome](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s com linha ilegível", nome)
		}
		if err := calls[nome](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s no fim da leitura: %v", nome, err)
		}
	}
	if err := r.Update(ctx, dbtest.ScanFail{}, p); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("update sem linha: %v", err)
	}
	casos := map[error]error{
		pgx.ErrNoRows:                  domain.ErrNaoEncontrado,
		&pgconn.PgError{Code: "23503"}: domain.ErrUnidadeInexistente,
		&pgconn.PgError{Code: "23514"}: domain.ErrEntradaInvalida,
	}
	for causa, want := range casos {
		if _, err := r.Insert(ctx, umaLinha{causa}, p); !errors.Is(err, want) {
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
	autor := dbtest.User(t, pool)
	novo, _ := domain.Novo(domain.Entrada{Titulo: "Quedas", UnidadeID: &u}, autor)
	p, err := r.Insert(ctx, pool, novo)
	if err != nil || p.Numero == 0 {
		t.Fatalf("insert: %+v %v", p, err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM problemas WHERE id = $1`, p.ID) })
	inc, ativo := uuid.New(), uuid.New()
	if ok, err := r.VincularIncidente(ctx, pool, p.ID, domain.IncidenteVinculado{SolicitacaoID: inc, Numero: 7, Titulo: "Caiu"}, autor); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, _ := r.VincularIncidente(ctx, pool, p.ID, domain.IncidenteVinculado{SolicitacaoID: inc, Numero: 7, Titulo: "Caiu"}, autor); ok {
		t.Fatal("ligar de novo")
	}
	if ok, err := r.VincularAtivo(ctx, pool, p.ID, ativo, autor); err != nil || !ok {
		t.Fatal(err)
	}
	if lista, err := r.List(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}, SolicitacaoID: &inc, Busca: "qued"}); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %v %v", lista, err)
	}
	if is, err := r.Incidentes(ctx, pool, p.ID); err != nil || len(is) != 1 || is[0].Numero != 7 {
		t.Fatalf("incidentes: %v %v", is, err)
	}
	if as, err := r.AtivosDe(ctx, pool, p.ID); err != nil || len(as) != 1 {
		t.Fatalf("ativos: %v %v", as, err)
	}
	if ligados, err := r.JaVinculados(ctx, pool, []uuid.UUID{inc, uuid.New()}); err != nil || !ligados[inc] || len(ligados) != 1 {
		t.Fatalf("já vinculados: %v %v", ligados, err)
	}
	if ok, err := r.DesvincularIncidente(ctx, pool, p.ID, inc); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, err := r.DesvincularAtivo(ctx, pool, p.ID, ativo); err != nil || !ok {
		t.Fatal(err)
	}
	if err := r.Travar(ctx, pool, p.ID); err != nil {
		t.Fatal(err)
	}
	p.Estado = domain.EmAnalise
	if err := r.Update(ctx, pool, p); err != nil {
		t.Fatal(err)
	}
}
