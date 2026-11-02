package domain

import (
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func TestPodeReabrir(t *testing.T) {
	unidade, pai, entidade := uuid.New(), uuid.New(), uuid.New()
	manage := []string{string(auth.PermProjetosManage)}
	gestor := auth.Identity{Scopes: []auth.Scope{{UnidadeID: &unidade, Permissions: manage}}}
	superior := auth.Identity{Scopes: []auth.Scope{{UnidadeID: &pai, Permissions: manage}}}
	secretaria := auth.Identity{Scopes: []auth.Scope{{EntidadeID: &entidade, Permissions: manage}}}
	admin := auth.Identity{Roles: []string{auth.RoleAdmin}}

	if PodeReabrir(gestor, &unidade, &entidade, []uuid.UUID{pai}) {
		t.Fatal("gestor da própria unidade não reabre")
	}
	if !PodeReabrir(superior, &unidade, &entidade, []uuid.UUID{pai}) {
		t.Fatal("unidade acima reabre")
	}
	if !PodeReabrir(secretaria, &unidade, &entidade, nil) {
		t.Fatal("gestão da entidade reabre")
	}
	if !PodeReabrir(admin, nil, nil, nil) {
		t.Fatal("administrador reabre projeto institucional")
	}
	if PodeReabrir(superior, nil, nil, nil) {
		t.Fatal("superior de uma unidade não reabre o institucional")
	}
	motivo, err := NormalizarMotivo("  voltou o escopo ")
	if err != nil || motivo != "voltou o escopo" {
		t.Fatalf("motivo: %q %v", motivo, err)
	}
	if _, err := NormalizarMotivo("  "); err != ErrMotivoObrigatorio {
		t.Fatalf("motivo vazio: %v", err)
	}
}
