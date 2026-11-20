package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestRepositorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	agora := time.Now().UTC()
	fornecedor, _ := domain.NovoFornecedor("Casa", "", "", &id)
	contrato, _ := domain.NovoContrato("1", "Suporte", id, agora, agora.Add(time.Hour), 10, &id)
	licenca, _ := domain.NovaLicenca("Office", id, nil, 1, &id)
	calls := map[string]func(database.DBTX) error{
		"ListFornecedores": func(db database.DBTX) error { _, err := r.ListFornecedores(ctx, db, domain.Filtro{}); return err },
		"InsertFornecedor": func(db database.DBTX) error { _, err := r.InsertFornecedor(ctx, db, fornecedor); return err },
		"DeleteFornecedor": func(db database.DBTX) error { return r.DeleteFornecedor(ctx, db, id, domain.Filtro{}) },
		"ListContratos":    func(db database.DBTX) error { _, err := r.ListContratos(ctx, db, domain.Filtro{}); return err },
		"InsertContrato":   func(db database.DBTX) error { _, err := r.InsertContrato(ctx, db, contrato); return err },
		"DeleteContrato":   func(db database.DBTX) error { return r.DeleteContrato(ctx, db, id, domain.Filtro{}) },
		"UpdatePrazo":      func(db database.DBTX) error { return r.UpdatePrazo(ctx, db, id, nil, domain.Filtro{}) },
		"ListLicencas":     func(db database.DBTX) error { _, err := r.ListLicencas(ctx, db, domain.Filtro{}); return err },
		"InsertLicenca":    func(db database.DBTX) error { _, err := r.InsertLicenca(ctx, db, licenca); return err },
		"DeleteLicenca":    func(db database.DBTX) error { return r.DeleteLicenca(ctx, db, id, domain.Filtro{}) },
		"Vencendo":         func(db database.DBTX) error { _, err := r.Vencendo(ctx, db); return err },
		"Destinatarios":    func(db database.DBTX) error { _, err := r.Destinatarios(ctx, db, &id); return err },
	}
	for nome, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := r.ListFornecedores(ctx, dbtest.ScanFail{}, domain.Filtro{}); err == nil {
		t.Error("linha ilegível")
	}
	if _, err := r.ListFornecedores(ctx, dbtest.RowsErr{}, domain.Filtro{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("fim da leitura: %v", err)
	}
	casos := map[error]error{
		pgx.ErrNoRows:                  nil,
		&pgconn.PgError{Code: "23503"}: domain.EntradaInvalida{},
		&pgconn.PgError{Code: "23514"}: domain.EntradaInvalida{},
	}
	for causa, want := range casos {
		_, err := r.InsertFornecedor(ctx, linha{causa}, fornecedor)
		if want == nil {
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
func (l linha) QueryRow(context.Context, string, ...any) pgx.Row        { return linhaRow(l) }

type linhaRow struct{ err error }

func (l linhaRow) Scan(...any) error { return l.err }

func TestPersistenciaReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	fornecedor, err := domain.NovoFornecedor("Casa "+uuid.NewString()[:8], "casa@org.gov.br", "", &u)
	if err != nil {
		t.Fatal(err)
	}
	salvo, err := r.InsertFornecedor(ctx, pool, fornecedor)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = r.DeleteFornecedor(context.Background(), pool, salvo.ID, domain.Filtro{})
	})
	inicio := time.Now().UTC().Truncate(time.Second)
	contrato, err := domain.NovoContrato("12", "Suporte", salvo.ID, inicio, inicio.Add(24*time.Hour), 100, &u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.InsertContrato(ctx, pool, contrato); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.DeleteContrato(context.Background(), pool, contrato.ID, domain.Filtro{}) })
	licenca, err := domain.NovaLicenca("Office", salvo.ID, nil, 2, &u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.InsertLicenca(ctx, pool, licenca); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.DeleteLicenca(context.Background(), pool, licenca.ID, domain.Filtro{}) })
	lista, err := r.ListContratos(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}})
	if err != nil || len(lista) != 1 || lista[0].FornecedorNome != fornecedor.Nome {
		t.Fatalf("%v %#v", err, lista)
	}
	if _, err := r.InsertContrato(ctx, pool, domain.Contrato{ID: uuid.New(), FornecedorID: uuid.New(), Inicio: inicio, Fim: inicio, CreatedAt: inicio, UpdatedAt: inicio}); err == nil {
		t.Fatal("fornecedor inexistente")
	}
	if _, err := r.Vencendo(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Destinatarios(ctx, pool, &u); err != nil {
		t.Fatal(err)
	}
}

func TestLeiturasQuebradas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	agora := time.Now().UTC()
	contrato, _ := domain.NovoContrato("1", "Suporte", id, agora, agora, 1, nil)
	licenca, _ := domain.NovaLicenca("Office", id, nil, 1, nil)
	for nome, call := range map[string]func(database.DBTX) error{
		"contratos":     func(db database.DBTX) error { _, err := r.ListContratos(ctx, db, domain.Filtro{}); return err },
		"licenças":      func(db database.DBTX) error { _, err := r.ListLicencas(ctx, db, domain.Filtro{}); return err },
		"vencendo":      func(db database.DBTX) error { _, err := r.Vencendo(ctx, db); return err },
		"destinatários": func(db database.DBTX) error { _, err := r.Destinatarios(ctx, db, nil); return err },
	} {
		if err := call(dbtest.ScanFail{}); err == nil {
			t.Errorf("%s ilegível", nome)
		}
		if err := call(dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s no fim: %v", nome, err)
		}
	}
	idOk := idRow{}
	falhaLista := &dbtest.Seq{Row: idOk, Queries: []dbtest.QueryResult{{Err: dbtest.ErrInjected}}}
	if _, err := r.InsertContrato(ctx, falhaLista, contrato); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("releitura do contrato: %v", err)
	}
	if _, err := r.InsertLicenca(ctx, &dbtest.Seq{Row: idOk, Queries: []dbtest.QueryResult{{Err: dbtest.ErrInjected}}}, licenca); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("releitura da licença: %v", err)
	}
	sumiu := &dbtest.Seq{Row: idOk, Queries: []dbtest.QueryResult{{Rows: dbtest.NoRows()}}}
	if got, err := r.InsertContrato(ctx, sumiu, contrato); err != nil || got.Titulo != contrato.Titulo {
		t.Fatalf("contrato sem a lista: %+v %v", got, err)
	}
	if got, err := r.InsertLicenca(ctx, &dbtest.Seq{Row: idOk, Queries: []dbtest.QueryResult{{Rows: dbtest.NoRows()}}}, licenca); err != nil || got.Nome != licenca.Nome {
		t.Fatalf("licença sem a lista: %+v %v", got, err)
	}
}

type idRow struct{}

func (idRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		if id, ok := dest[0].(*uuid.UUID); ok {
			*id = uuid.New()
		}
	}
	return nil
}
