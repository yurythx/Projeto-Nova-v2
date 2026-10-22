// Package produtos é o plugin do produto digital (ciclo de vida ITIL 5).
// Nasce desligado.
package produtos

import (
	"github.com/yurythx/sismarq/internal/modules/produtos/application"
	"github.com/yurythx/sismarq/internal/modules/produtos/domain"
	"github.com/yurythx/sismarq/internal/modules/produtos/infrastructure"
	"github.com/yurythx/sismarq/internal/modules/produtos/transport"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "produtos"

// Module é o plugin.
type Module struct {
	repo     domain.Repository
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox)
	return &Module{repo: repo, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Repositorio é a porta usada pelo atendimento para conferir o produto
// sem importar este módulo.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Produtos digitais",
		Description:    "Produtos que a organização mantém ao longo do ciclo de vida: da descoberta ao suporte.",
		DefaultEnabled: false,
		Icon:           "package",
		Route:          "/produtos",
		Permissions: []kernel.PermissionInfo{
			{Key: "produtos:manage", Description: "Cadastrar produtos digitais e mudar o estágio do ciclo de vida"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
