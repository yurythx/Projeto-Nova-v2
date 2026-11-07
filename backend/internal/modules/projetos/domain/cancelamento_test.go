package domain

import (
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func TestPodeCancelar(t *testing.T) {
	gestor := auth.Identity{Permissions: []string{"projetos:manage"}}
	if PodeCancelar(gestor) {
		t.Fatal("quem gerencia o projeto não cancela")
	}
	admin := auth.Identity{Roles: []string{string(auth.RoleAdmin)}}
	if !PodeCancelar(admin) {
		t.Fatal("administrador cancela")
	}
}
