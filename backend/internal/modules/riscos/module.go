// Package riscos implementa a gestão de riscos de TI conforme a ISO/IEC 27001 (GRC).
package riscos

import (
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

const Key = "riscos"

type Module struct {
	repo     domain.Repository
	svc      *application.Service
	handlers *transport.Handlers
}

func New(deps modkit.Deps) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox)
	return &Module{
		repo:     repo,
		svc:      svc,
		handlers: transport.NewHandlers(svc, deps.Logger),
	}
}

func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Riscos (ISO 27001)",
		Description:    "Matriz de Riscos de TI, Continuidade e Segurança da Informação conforme a norma ISO/IEC 27001 e conformidade LGPD.",
		DefaultEnabled: true,
		Icon:           "shield-alert",
		Route:          "/riscos",
		Permissions: []kernel.PermissionInfo{
			{Key: "riscos:read", Description: "Consultar a matriz de riscos e o heatmap da unidade"},
			{Key: "riscos:manage", Description: "Cadastrar, editar e definir planos de mitigação de riscos"},
		},
	}
}

func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}

