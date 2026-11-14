package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestRepositorioPropagaFalha(t *testing.T) {
	r := NewPostgresRepository()
	ctx := context.Background()
	id := uuid.New()
	p := domain.Processo{ID: id, ReciboEm: time.Now()}
	if err := r.Inserir(ctx, dbtest.Fail{}, p); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if err := r.Atualizar(ctx, dbtest.Fail{}, p); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.Buscar(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.PorCodigo(ctx, dbtest.Fail{}, "x"); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.Listar(ctx, dbtest.Fail{}, ""); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if _, err := r.Proximo(ctx, dbtest.Fail{}, 2026); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if err := r.Atualizar(ctx, dbtest.ScanFail{}, p); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	sem := &dbtest.Seq{Row: noRows{}}
	if _, err := r.Buscar(ctx, sem, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := r.PorCodigo(ctx, sem, "x"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	for _, rows := range []pgx.Rows{dbtest.BadRows(), dbtest.ErrRows()} {
		if _, err := r.Listar(ctx, &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: rows}}}, "ana"); err == nil {
			t.Fatal("lista")
		}
	}
	if _, err := r.Listar(ctx, &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: dbtest.NoRows()}}}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestRepositorioLeProcesso(t *testing.T) {
	r := NewPostgresRepository()
	ctx := context.Background()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	quando := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	row := processoRow{id: id, quando: quando, comPrazo: true}
	p, err := r.Buscar(ctx, &dbtest.Seq{Row: row}, id)
	if err != nil || p.Numero != "000001/2026" || p.ExigenciaPrazo.IsZero() || !p.ExigenciaProrrogada || p.DespachoEm.IsZero() || p.AlertaLimite.IsZero() || p.Destinatario != id {
		t.Fatalf("%+v %v", p, err)
	}
	p, err = r.PorCodigo(ctx, &dbtest.Seq{Row: processoRow{id: id, quando: quando}}, "cod")
	if err != nil || !p.ExigenciaAberta.IsZero() || p.ID != id {
		t.Fatalf("%+v %v", p, err)
	}
	lista, err := r.Listar(ctx, &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: &uma{row: processoRow{id: id, quando: quando}}}}}, "ana")
	if err != nil || len(lista) != 1 {
		t.Fatal(err)
	}
	n, err := r.Proximo(ctx, &dbtest.Seq{Row: intRow{n: 4}}, 2026)
	if err != nil || n != 4 {
		t.Fatal(err)
	}
	if err := r.Inserir(ctx, okExec{}, domain.Processo{ID: id, ReciboEm: quando, ExigenciaAberta: quando, ExigenciaPrazo: quando, Destinatario: id}); err != nil {
		t.Fatal(err)
	}
	if err := r.Inserir(ctx, okExec{}, domain.Processo{ID: id, ReciboEm: quando}); err != nil {
		t.Fatal(err)
	}
	if err := r.Atualizar(ctx, okExec{}, domain.Processo{ID: id, ExigenciaPrazo: quando}); err != nil {
		t.Fatal(err)
	}
	if tempo(time.Time{}) != nil || tempo(quando) == nil {
		t.Fatal("tempo")
	}
}

type noRows struct{}

func (noRows) Scan(...any) error { return pgx.ErrNoRows }

type intRow struct{ n int }

func (r intRow) Scan(dest ...any) error {
	*(dest[0].(*int)) = r.n
	return nil
}

type processoRow struct {
	id       uuid.UUID
	quando   time.Time
	comPrazo bool
}

func (r processoRow) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.id
	*(dest[1].(*string)) = "000001/2026"
	*(dest[2].(*int)) = 2026
	*(dest[3].(*int)) = 1
	*(dest[4].(*string)) = domain.ProgramaAvanca
	*(dest[5].(*string)) = domain.EntradaGovBR
	*(dest[6].(*string)) = "ana"
	*(dest[7].(*string)) = domain.SituacaoProtocolado
	*(dest[8].(*time.Time)) = r.quando
	*(dest[9].(*string)) = "cod"
	*(dest[10].(*string)) = ""
	*(dest[11].(*int64)) = 0
	*(dest[12].(*string)) = ""
	if r.comPrazo {
		quando := r.quando
		*(dest[13].(**time.Time)) = &quando
		*(dest[14].(**time.Time)) = &quando
		*(dest[16].(**time.Time)) = &quando
		*(dest[17].(**time.Time)) = &quando
		quem := r.id
		*(dest[18].(**uuid.UUID)) = &quem
	}
	*(dest[15].(*bool)) = r.comPrazo
	return nil
}

type uma struct {
	row processoRow
	viu bool
}

func (u *uma) Close()                                       {}
func (u *uma) Err() error                                   { return nil }
func (u *uma) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (u *uma) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (u *uma) Values() ([]any, error)                       { return nil, nil }
func (u *uma) RawValues() [][]byte                          { return nil }
func (u *uma) Conn() *pgx.Conn                              { return nil }
func (u *uma) Next() bool {
	if u.viu {
		return false
	}
	u.viu = true
	return true
}
func (u *uma) Scan(dest ...any) error { return u.row.Scan(dest...) }

type okExec struct{}

func (okExec) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (okExec) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, dbtest.ErrInjected
}
func (okExec) QueryRow(context.Context, string, ...any) pgx.Row { return noRows{} }
