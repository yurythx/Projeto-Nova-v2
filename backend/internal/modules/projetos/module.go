package projetos

import (
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "projetos"

// Module é o plug-in de gestão de projetos.
type Module struct {
	deps     modkit.Deps
	handlers *transport.Handlers
	service  *application.Service
}

// ComPassagens liga os caminhos acrescentados nos quadros de projetos e de tarefas.
func (m *Module) ComPassagens(p application.Passagens) {
	if m != nil && m.service != nil {
		m.service.ComPassagens(p)
	}
}

// ComPortas liga os vínculos de mudanças, problemas, ativos, solicitações e produtos.
func (m *Module) ComPortas(portas domain.Portas, produtos domain.Produtos) {
	if m != nil && m.service != nil {
		m.service.ComPortas(portas, produtos)
	}
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox, deps.Logger)
	handlers := transport.NewHandlers(svc, deps.Logger)
	return &Module{
		deps:     deps,
		handlers: handlers,
		service:  svc,
	}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Projetos",
		Description:    "Gestão de projetos e planos de ação da organização.",
		DefaultEnabled: false,
		Icon:           "folder-kanban",
		Route:          "/projetos",
		Permissions: []kernel.PermissionInfo{
			{Key: "projetos:read", Description: "Consultar projetos e tarefas"},
			{Key: "projetos:manage", Description: "Criar e conduzir projetos e tarefas"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}

// Service retorna o serviço de aplicação.
func (m *Module) Service() *application.Service {
	return m.service
}
