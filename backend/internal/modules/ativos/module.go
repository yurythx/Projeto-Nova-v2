// Package ativos é o plugin dos itens de configuração: o que sustenta os
// produtos digitais. Nasce desligado. Plugins não se importam: o produto
// chega por uma porta ligada em internal/app.
package ativos

import (
	"github.com/yurythx/sismarq/internal/modules/ativos/application"
	"github.com/yurythx/sismarq/internal/modules/ativos/domain"
	"github.com/yurythx/sismarq/internal/modules/ativos/infrastructure"
	"github.com/yurythx/sismarq/internal/modules/ativos/transport"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "ativos"

// Module é o plugin.
type Module struct {
	repo     domain.Repository
	handlers *transport.Handlers
}

// New constrói o módulo. produtos confere o produto que o ativo sustenta;
// nil deixa o ativo sem esse vínculo.
func New(deps modkit.Deps, produtos domain.Produtos) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox, produtos)
	return &Module{repo: repo, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Repositorio é a porta usada por problemas e mudanças para conferir o
// ativo sem importar este módulo.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Ativos",
		Description:    "Itens de configuração que sustentam os produtos digitais: equipamentos, sistemas, licenças e links.",
		DefaultEnabled: false,
		Icon:           "server",
		Route:          "/ativos",
		Permissions: []kernel.PermissionInfo{
			{Key: "ativos:manage", Description: "Cadastrar os ativos da unidade e ligá-los aos produtos"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
