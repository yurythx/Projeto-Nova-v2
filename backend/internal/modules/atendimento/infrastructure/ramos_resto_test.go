package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestRamosDeInfraDaEntrada(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	id := uuid.New()
	agora := time.Now().UTC()
	unidade := id

	if _, err := r.ParaEncerrar(ctx, dbtest.Fail{}, agora, 0); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("para encerrar: %v", err)
	}
	if _, err := r.Paradas(ctx, dbtest.Fail{}, agora, 0); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("paradas: %v", err)
	}
	if _, err := r.ParaEncerrar(ctx, dbtest.ScanFail{}, agora, 1); err == nil {
		t.Fatal("leitura de quem encerrar")
	}
	if _, err := r.UnidadesComSolicitacao(ctx, dbtest.ScanFail{}); err == nil {
		t.Fatal("unidades")
	}
	if _, err := r.ListarRetratos(ctx, dbtest.ScanFail{}, agora, false, nil); err == nil {
		t.Fatal("retratos")
	}
	if _, err := r.ListSLAs(ctx, dbtest.Fail{}, &unidade); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("slas: %v", err)
	}

	if err := r.SetSegredo(ctx, dbtest.Fail{}, id, nil, agora); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("segredo: %v", err)
	}
	if err := r.SetSegredo(ctx, dbtest.ScanFail{}, id, nil, agora); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("segredo ausente: %v", err)
	}
	if err := r.SetAtivo(ctx, dbtest.Fail{}, id, nil, "nome", agora); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("ativo: %v", err)
	}
	if err := r.SetAtivo(ctx, dbtest.ScanFail{}, id, nil, "nome", agora); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("ativo ausente: %v", err)
	}
	if err := r.SetIncidentePai(ctx, dbtest.Fail{}, id, nil, agora); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("pai: %v", err)
	}
	if err := r.SetIncidentePai(ctx, dbtest.ScanFail{}, id, &id, agora); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("pai ausente: %v", err)
	}
	if _, err := r.ListarFilhosDoIncidente(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatalf("filhos: %v", err)
	}
	if _, err := r.ListarFilhosDoIncidente(ctx, dbtest.ScanFail{}, id); err == nil {
		t.Fatal("leitura dos filhos")
	}

	email, err := r.EmailDe(ctx, linhaVazia{}, id)
	if err != nil || email != "" {
		t.Fatalf("e-mail: %q %v", email, err)
	}
}

// linhaVazia responde a consulta de uma linha com "não há linha".
type linhaVazia struct{}

func (linhaVazia) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, dbtest.ErrInjected
}

func (linhaVazia) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, dbtest.ErrInjected
}

func (linhaVazia) QueryRow(context.Context, string, ...any) pgx.Row { return semLinha{} }
