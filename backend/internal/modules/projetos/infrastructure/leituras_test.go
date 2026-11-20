package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// leitura é uma consulta com uma linha legível. O banco que falha sempre
// erra na primeira chamada e não chega no miolo do laço.
type leitura struct {
	left int
	err  error
	scan func(...any) error
}

func umaLinha(scan func(...any) error) *leitura {
	return &leitura{left: 1, scan: scan}
}

func (l *leitura) Close()                                       {}
func (l *leitura) Err() error                                   { return l.err }
func (l *leitura) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (l *leitura) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (l *leitura) Values() ([]any, error)                       { return nil, nil }
func (l *leitura) RawValues() [][]byte                          { return nil }
func (l *leitura) Conn() *pgx.Conn                              { return nil }
func (l *leitura) Next() bool {
	if l.left == 0 {
		return false
	}
	l.left--
	return true
}
func (l *leitura) Scan(dest ...any) error {
	if l.scan != nil {
		return l.scan(dest...)
	}
	return nil
}

type valor struct {
	scan func(...any) error
}

func (v valor) Scan(dest ...any) error {
	if v.scan != nil {
		return v.scan(dest...)
	}
	return nil
}

func seq(row pgx.Row, consultas ...dbtest.QueryResult) *dbtest.Seq {
	return &dbtest.Seq{Row: row, Queries: consultas}
}

func TestLeiturasNoMeio(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	id := uuid.New()
	agora := time.Now().UTC()
	linhaOk := valor{}
	pessoa := func(dest ...any) error {
		if p, ok := dest[0].(*uuid.UUID); ok {
			*p = uuid.Nil
		}
		if len(dest) > 1 {
			if p, ok := dest[1].(*uuid.UUID); ok {
				*p = id
			}
		}
		if len(dest) > 2 {
			if p, ok := dest[2].(*string); ok {
				*p = "Ana"
			}
		}
		return nil
	}

	if _, _, err := r.PosicaoUnidade(ctx, seq(linhaOk, dbtest.QueryResult{Err: dbtest.ErrInjected}), id); err == nil {
		t.Fatal("posição")
	}
	if _, _, err := r.PosicaoUnidade(ctx, seq(linhaOk, dbtest.QueryResult{Rows: dbtest.BadRows()}), id); err == nil {
		t.Fatal("posição ilegível")
	}
	if _, _, err := r.PosicaoUnidade(ctx, seq(linhaOk, dbtest.QueryResult{Rows: dbtest.ErrRows()}), id); err == nil {
		t.Fatal("posição interrompida")
	}

	estado, err := r.EstadoDaTarefa(ctx, seq(valor{scan: func(dest ...any) error {
		*dest[0].(*string) = domain.EstadoTarefaAFazer
		return nil
	}}), id)
	if err != nil || estado != domain.EstadoTarefaAFazer {
		t.Fatal(err, estado)
	}

	if _, err := r.ObterTarefa(ctx, seq(linhaOk, dbtest.QueryResult{Err: dbtest.ErrInjected}), id); err == nil {
		t.Fatal("pessoas da tarefa")
	}
	if _, err := r.ObterAnexo(ctx, soLinha{row: linha{err: pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrAnexoNaoEncontrado) {
		t.Fatal(err)
	}
	historico, err := r.ListarHistorico(ctx, seq(nil, dbtest.QueryResult{Rows: dbtest.NoRows()}), id)
	if err != nil || historico == nil {
		t.Fatal(err)
	}

	if _, err := r.ListarMinhas(ctx, seq(nil,
		dbtest.QueryResult{Rows: umaLinha(nil)},
		dbtest.QueryResult{Err: dbtest.ErrInjected},
	), id, false, nil); err == nil {
		t.Fatal("minhas")
	}
	minhas, err := r.ListarMinhas(ctx, seq(nil,
		dbtest.QueryResult{Rows: umaLinha(nil)},
		dbtest.QueryResult{Rows: umaLinha(pessoa)},
	), id, false, nil)
	if err != nil || len(minhas) != 1 || len(minhas[0].Pessoas) != 1 {
		t.Fatal(err, len(minhas))
	}

	if _, err := r.ListarPrazos(ctx, seq(nil,
		dbtest.QueryResult{Rows: umaLinha(nil)},
		dbtest.QueryResult{Err: dbtest.ErrInjected},
	), "2026-11", false, nil); err == nil {
		t.Fatal("prazos")
	}
	prazos, err := r.ListarPrazos(ctx, seq(nil,
		dbtest.QueryResult{Rows: umaLinha(nil)},
		dbtest.QueryResult{Rows: umaLinha(pessoa)},
	), "2026-11", false, nil)
	if err != nil || len(prazos) != 1 || len(prazos[0].Pessoas) != 1 {
		t.Fatal(err)
	}

	if _, _, err := r.PrazosVencidos(ctx, seq(nil,
		dbtest.QueryResult{Rows: umaLinha(nil)},
		dbtest.QueryResult{Err: dbtest.ErrInjected},
	), agora); err == nil {
		t.Fatal("vencidos")
	}
	if _, _, err := r.PrazosVencidos(ctx, seq(nil,
		dbtest.QueryResult{Rows: dbtest.NoRows()},
		dbtest.QueryResult{Rows: dbtest.BadRows()},
	), agora); err == nil {
		t.Fatal("fim ilegível")
	}
	if _, _, err := r.PrazosVencidos(ctx, seq(nil,
		dbtest.QueryResult{Rows: dbtest.NoRows()},
		dbtest.QueryResult{Rows: dbtest.ErrRows()},
	), agora); err == nil {
		t.Fatal("fim interrompido")
	}
}
