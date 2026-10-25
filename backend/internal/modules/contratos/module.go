// Package contratos é o plugin de fornecedor, contrato e licença.
// Nasce desligado. O aviso de vencimento sai no sino de quem gerencia.
package contratos

import (
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "contratos"

// Module é o plugin.
type Module struct {
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox).WithNotificacoes(deps.Notificacoes)
	return &Module{svc: svc, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Contratos",
		Description:    "Fornecedores, contratos e licenças, com aviso quando a vigência se aproxima.",
		DefaultEnabled: false,
		Icon:           "file-text",
		Route:          "/contratos",
		Permissions: []kernel.PermissionInfo{
			{Key: "contratos:manage", Description: "Cadastrar fornecedor, contrato e licença da unidade"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) { m.handlers.RegisterRoutes(r.Authed) }

// Workers implementa kernel.WorkerProvider.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{{Name: "vencimento-de-contratos", Process: kernel.ProcessWorker, Run: m.svc.RunVencimentos}}
}
