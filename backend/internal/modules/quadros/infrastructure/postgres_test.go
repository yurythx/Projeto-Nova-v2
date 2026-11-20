package infrastructure

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestBancoQueFalha(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	col := domain.Coluna{Quadro: "problemas", Chave: "c00000000", Nome: "Triagem", Cor: "#112233", Tom: "info", Ordem: 2, Ativa: true}
	passagem := []domain.Passagem{{De: "registrado", Para: col.Chave}}
	for _, db := range []database.DBTX{dbtest.Fail{}, dbtest.ScanFail{}, dbtest.RowsErr{}} {
		_, _ = r.Listar(ctx, db, "problemas")
		_, _ = r.Mapa(ctx, db, "problemas")
		_ = r.SoltarOrdens(ctx, db, "problemas")
		_ = r.Inserir(ctx, db, col)
		_ = r.SubstituirPassagens(ctx, db, "problemas", col.Chave, passagem)
		_ = r.Salvar(ctx, db, []domain.Coluna{col})
		_, _ = r.Contar(ctx, db, "problemas", "registrado")
	}
	if _, err := r.Contar(ctx, nil, "outro", "x"); err == nil {
		t.Fatal("quadro")
	}
}

func TestLeiturasEGravacoes(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	col := domain.Coluna{Quadro: "problemas", Chave: "c00000000", Nome: "Triagem", Cor: "#112233", Tom: "info", Ordem: 2, Entrada: false, Ativa: true}
	db := banco{rows: &grade{linhas: [][]any{{"problemas", "c00000000", "Triagem", "#112233", "info", 2, false, true}}}}
	cols, err := r.Listar(ctx, db, "problemas")
	if err != nil || len(cols) != 1 || cols[0].Chave != "c00000000" || !cols[0].Ativa {
		t.Fatalf("listar: %v %#v", err, cols)
	}
	db.rows = &grade{linhas: [][]any{{"registrado", "c00000000"}}}
	mapa, err := r.Mapa(ctx, db, "problemas")
	if err != nil || len(mapa["registrado"]) != 1 {
		t.Fatalf("mapa: %v %#v", err, mapa)
	}
	ok := banco{tag: pgconn.NewCommandTag("UPDATE 1")}
	if err := r.SoltarOrdens(ctx, ok, "problemas"); err != nil {
		t.Fatal(err)
	}
	if err := r.Inserir(ctx, ok, col); err != nil {
		t.Fatal(err)
	}
	if err := r.Salvar(ctx, ok, []domain.Coluna{col}); err != nil {
		t.Fatal(err)
	}
	zero := &dbtest.Seq{Execs: []dbtest.ExecResult{{Tag: pgconn.NewCommandTag("UPDATE 0")}}}
	if err := r.Salvar(ctx, zero, []domain.Coluna{col}); err == nil {
		t.Fatal("nenhuma linha")
	}
	meio := &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, {Tag: pgconn.NewCommandTag("UPDATE 0")}}}
	if err := r.Salvar(ctx, meio, []domain.Coluna{col, col}); err == nil {
		t.Fatal("segunda linha")
	}
	if err := r.SubstituirPassagens(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK}}, "problemas", col.Chave, nil); err != nil {
		t.Fatal(err)
	}
	itens := []domain.Passagem{{De: "registrado", Para: col.Chave}}
	if err := r.SubstituirPassagens(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, dbtest.OK}}, "problemas", col.Chave, itens); err != nil {
		t.Fatal(err)
	}
	if err := r.SubstituirPassagens(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, {Err: dbtest.ErrInjected}}}, "problemas", col.Chave, itens); err == nil {
		t.Fatal("insert")
	}
	contagem := banco{row: nRow{}}
	for _, caso := range []struct{ quadro, chave string }{
		{"problemas", "registrado"},
		{"mudancas", "fechada"},
		{"mudancas", "rascunho"},
		{"projetos", "planejado"},
		{"tarefas", "a_fazer"},
		{"ativos", "ativo"},
		{"fila", "concluida"},
		{"fila", "novo"},
	} {
		n, err := r.Contar(ctx, contagem, caso.quadro, caso.chave)
		if err != nil || n != 2 {
			t.Fatalf("%s/%s: %v %d", caso.quadro, caso.chave, err, n)
		}
	}
}

type banco struct {
	rows pgx.Rows
	row  pgx.Row
	tag  pgconn.CommandTag
}

func (b banco) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return b.tag, nil
}
func (b banco) Query(context.Context, string, ...any) (pgx.Rows, error) { return b.rows, nil }
func (b banco) QueryRow(context.Context, string, ...any) pgx.Row        { return b.row }

type nRow struct{}

func (nRow) Scan(dest ...any) error {
	*dest[0].(*int) = 2
	return nil
}

type grade struct {
	linhas [][]any
	i      int
}

func (g *grade) Close()                                       {}
func (g *grade) Err() error                                   { return nil }
func (g *grade) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (g *grade) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (g *grade) Values() ([]any, error)                       { return nil, nil }
func (g *grade) RawValues() [][]byte                          { return nil }
func (g *grade) Conn() *pgx.Conn                              { return nil }
func (g *grade) Next() bool {
	if g.i >= len(g.linhas) {
		return false
	}
	g.i++
	return true
}
func (g *grade) Scan(dest ...any) error {
	for i := range dest {
		if err := atribuir(dest[i], g.linhas[g.i-1][i]); err != nil {
			return err
		}
	}
	return nil
}

func atribuir(dest, src any) error {
	switch d := dest.(type) {
	case *string:
		*d = src.(string)
	case *int:
		*d = src.(int)
	case *bool:
		*d = src.(bool)
	default:
		return dbtest.ErrInjected
	}
	return nil
}
