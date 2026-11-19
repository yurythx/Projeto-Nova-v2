package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	atendimento "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	ativosmod "github.com/yurythx/projeto-nova-v2/internal/modules/ativos"
	cofredomain "github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	contratosapp "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	problemas "github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	projetos "github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	signumdomain "github.com/yurythx/projeto-nova-v2/internal/modules/signum/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
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
}
