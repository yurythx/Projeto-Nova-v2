// Package mudancas é o plugin da gestão de mudanças: nada que está em
// operação muda sem registro de quem pediu, quem aprovou e como desfazer.
// Nasce desligado. Problemas, incidentes, ativos e produtos chegam por
// portas ligadas em internal/app.
package mudancas

import (
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "mudancas"

// Module é o plugin.
type Module struct {
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps, portas domain.Portas, produtos domain.Produtos) *Module {
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox, portas, produtos)
	return &Module{handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Mudanças",
		Description:    "Alterações no que está em operação, com janela planejada, plano de reversão e aprovação registrada.",
		DefaultEnabled: false,
		Icon:           "git-pull-request",
		Route:          "/mudancas",
		Permissions: []kernel.PermissionInfo{
			{Key: "mudancas:manage", Description: "Registrar, conduzir e executar as mudanças da unidade"},
			{Key: "mudancas:aprovar", Description: "Aprovar ou rejeitar as mudanças da unidade (não as que pediu)"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
