package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	atendimento "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	ativosmod "github.com/yurythx/projeto-nova-v2/internal/modules/ativos"
	cofreapp "github.com/yurythx/projeto-nova-v2/internal/modules/cofre/application"
	cofredomain "github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	contratosapp "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	contratosdomain "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	problemas "github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	projetos "github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	signumdomain "github.com/yurythx/projeto-nova-v2/internal/modules/signum/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type reauthFalso struct{ err error }

func (r reauthFalso) Reauthenticate(context.Context, uuid.UUID, string, bool, string) (string, error) {
	return "", r.err
}

func TestPortasDesligadas(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	off := func() bool { return false }

	if err := (confirmaSenha{}).Confirmar(ctx, id, "ana", false, "x"); !errors.Is(err, cofredomain.ErrSemConfirmacao) {
		t.Fatalf("sem reauth: %v", err)
	}
	if err := (confirmaSenha{reauth: reauthFalso{err: signumdomain.ErrReauth}}).Confirmar(ctx, id, "ana", false, "x"); !errors.Is(err, cofredomain.ErrSenhaRecusada) {
		t.Fatalf("senha recusada: %v", err)
	}
	if err := (confirmaSenha{reauth: reauthFalso{err: errors.New("fora")}}).Confirmar(ctx, id, "ana", false, "x"); err == nil {
		t.Fatal("outra falha da reauth")
	}
	if _, err := (portaSegredo{ativo: off}).Nome(ctx, auth.Identity{}, id); !errors.Is(err, atendimento.ErrCofreDesligado) {
		t.Fatalf("cofre: %v", err)
	}
	if _, _, err := (ativoAtendimento{ativo: off}).Consultar(ctx, nil, id); !errors.Is(err, atendimento.ErrAtivosDesligado) {
		t.Fatalf("ativo do atendimento: %v", err)
	}
	if _, _, err := (ativoProblema{ativo: off}).Consultar(ctx, nil, id); !errors.Is(err, problemas.ErrIntegracaoDesligada) {
		t.Fatalf("ativo do problema: %v", err)
	}
	if _, err := (incidenteProblema{ativo: off}).Consultar(ctx, nil, id); !errors.Is(err, problemas.ErrIntegracaoDesligada) {
		t.Fatalf("incidente: %v", err)
	}
	if _, err := (produtoUnidade{ativo: off, desligado: projetos.ErrIntegracaoDesligada}).Consultar(ctx, nil, id); !errors.Is(err, projetos.ErrIntegracaoDesligada) {
		t.Fatalf("produto: %v", err)
	}
	if _, err := (programasDoAtivo{}).Contar(ctx, nil, false, nil, "chrome"); !errors.Is(err, contratosapp.ErrProgramasIndisponiveis) {
		t.Fatalf("programas sem módulo: %v", err)
	}
	if _, err := (programasDoAtivo{mod: &ativosmod.Module{}, ativo: off}).Contar(ctx, nil, false, nil, "chrome"); err == nil {
		t.Fatal("programas desligados")
	}
	if total, err := (horasDeChamado{}).Somar(ctx, nil, nil); err != nil || total != 0 {
		t.Fatalf("sem chamados: %v %v", total, err)
	}
	if _, err := (horasDeChamado{}).Somar(ctx, dbtest.Fail{}, []uuid.UUID{id}); err == nil {
		t.Fatal("soma sem banco")
	}
	if total, err := (horasDeChamado{}).Somar(ctx, somaDB{n: 1.5}, []uuid.UUID{id}); err != nil || total != 1.5 {
		t.Fatalf("soma: %v %v", total, err)
	}
	ligado := func() bool { return true }
	if _, err := (portaSegredo{svc: cofreapp.NewService(nil, nil, nil, nil), ativo: ligado}).Nome(ctx, auth.Identity{}, id); err == nil {
		t.Fatal("cofre sem alcance")
	}
	if _, err := (portaContratos{svc: contratosapp.NewService(nil, contratosQueFalham{}, nil), ativo: ligado}).Listar(ctx, auth.Identity{}); err == nil {
		t.Fatal("contratos")
	}
	mod := &ativosmod.Module{}
	if _, err := (programasDoAtivo{mod: mod, ativo: ligado}).Contar(ctx, dbtest.Fail{}, false, nil, "chrome"); err == nil {
		t.Fatal("contagem")
	}
	uso, err := (programasDoAtivo{mod: mod, ativo: ligado}).Contar(ctx, contaDB{}, false, nil, "chrome")
	if err != nil || uso.Instalados != 2 || uso.ComInventario != 3 || !uso.Parcial {
		t.Fatalf("uso: %+v %v", uso, err)
	}
}

type contratosQueFalham struct{ contratosdomain.Repository }

func (contratosQueFalham) ListContratos(context.Context, database.DBTX, contratosdomain.Filtro) ([]contratosdomain.Contrato, error) {
	return nil, errors.New("lista")
}

type somaDB struct{ n float64 }

func (somaDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (somaDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (s somaDB) QueryRow(context.Context, string, ...any) pgx.Row      { return somaRow{s.n} }

type somaRow struct{ n float64 }

func (r somaRow) Scan(dest ...any) error {
	*dest[0].(*float64) = r.n
	return nil
}

type contaDB struct{}

func (contaDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (contaDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (contaDB) QueryRow(context.Context, string, ...any) pgx.Row        { return contaRow{} }

type contaRow struct{}

func (contaRow) Scan(dest ...any) error {
	*dest[0].(*int) = 3
	*dest[1].(*int) = 2
	*dest[2].(*bool) = true
	return nil
}
