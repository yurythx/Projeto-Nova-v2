// Package problemas é o plugin da gestão de problemas: a causa por trás de
// incidentes repetidos e a base de erros conhecidos. Nasce desligado.
// Incidentes, ativos e produtos chegam por portas ligadas em internal/app.
package problemas

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "problemas"

// Module é o plugin.
type Module struct {
	repo     domain.Repository
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps, portas application.Portas) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox, portas)
	return &Module{repo: repo, svc: svc, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// ConcluirPorMudanca resolve o problema na transação de quem chama.
func (m *Module) ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, texto string) error {
	return m.svc.ConcluirPorMudanca(ctx, tx, id, autor, texto)
}

// ComPassagens liga os caminhos acrescentados no quadro.
func (m *Module) ComPassagens(p application.Passagens) {
	if m != nil && m.svc != nil {
		m.svc.ComPassagens(p)
	}
}

// Repositorio é a porta usada pelas mudanças para conferir o problema.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Problemas",
		Description:    "A causa por trás dos incidentes que se repetem e a base de erros conhecidos com a solução de contorno.",
		DefaultEnabled: false,
		Icon:           "bug",
		Route:          "/problemas",
		Permissions: []kernel.PermissionInfo{
			{Key: "problemas:read", Description: "Consultar os problemas e os erros conhecidos da unidade"},
			{Key: "problemas:manage", Description: "Registrar e investigar problemas da unidade e ligá-los a incidentes e ativos"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
