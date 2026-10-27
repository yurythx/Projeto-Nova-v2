package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	atendimentodomain "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	contratosapp "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	contratosdomain "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	contratosinfra "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type outboxMudo struct{}

func (outboxMudo) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return nil
}

func TestPortaContratos(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	svc := contratosapp.NewService(pool, contratosinfra.NewRepository(), outboxMudo{})
	quem := auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: &u, Permissions: []string{"atendimento:atender"}, Unidades: []uuid.UUID{u}}}}

	if _, err := (portaContratos{svc: svc, ativo: func() bool { return false }}).Listar(ctx, quem); !errors.Is(err, atendimentodomain.ErrContratosDesligado) {
		t.Fatalf("desligado: %v", err)
	}
	if _, err := (portaContratos{}).Obter(ctx, quem, uuid.New()); !errors.Is(err, atendimentodomain.ErrContratosDesligado) {
		t.Fatalf("sem serviço: %v", err)
	}

	admin := auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()}
	f, err := svc.CriarFornecedor(ctx, admin, "Provedor "+uuid.NewString()[:6], "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	c, err := svc.CriarContrato(ctx, admin, "12/2026", "Link", f.ID, hoje.AddDate(0, -1, 0), hoje.AddDate(1, 0, 0), 0, &u)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM contratos_contratos WHERE fornecedor_id = $1`, f.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM contratos_fornecedores WHERE id = $1`, f.ID)
	})
	oito := 8
	if err := svc.DefinirPrazo(ctx, admin, c.ID, &oito); err != nil {
		t.Fatal(err)
	}

	p := portaContratos{svc: svc, ativo: func() bool { return true }}
	achado, err := p.Obter(ctx, quem, c.ID)
	if err != nil || achado.Rotulo != "12/2026 — Link ("+f.Nome+")" || *achado.PrazoHoras != 8 || achado.Fornecedor != f.Nome {
		t.Fatalf("obter: %+v %v", achado, err)
	}
	outra := auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: ptrUUID(uuid.New()), Permissions: []string{"atendimento:atender"}}}}
	if _, err := p.Obter(ctx, outra, c.ID); !errors.Is(err, atendimentodomain.ErrContratoAusente) {
		t.Fatalf("contrato de outra unidade: %v", err)
	}

	semNumero := rotuloContrato(contratosdomain.Contrato{Titulo: "Energia", FornecedorNome: "Concessionária"})
	if semNumero != "Energia (Concessionária)" {
		t.Fatalf("sem número: %q", semNumero)
	}
	longo := rotuloContrato(contratosdomain.Contrato{Numero: "1", Titulo: strings.Repeat("á", 300), FornecedorNome: strings.Repeat("é", 300)})
	if utf8.RuneCountInString(longo) != maxRotuloContrato {
		t.Fatalf("rótulo cortado: %d", utf8.RuneCountInString(longo))
	}
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
