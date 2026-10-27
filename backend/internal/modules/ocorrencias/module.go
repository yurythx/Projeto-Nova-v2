// Package ocorrencias é o plug-in do incidente grave e da página de
// status pública (ADR 066). Nasce desligado e não depende do atendimento.
package ocorrencias

import (
	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/httpserver"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "ocorrencias"

// Module é o plugin.
type Module struct {
	deps     modkit.Deps
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox)
	return &Module{deps: deps, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Ocorrências",
		Description:    "Incidente grave: coordenação, linha do tempo e a página de status pública (/status).",
		DefaultEnabled: false,
		Icon:           "siren",
		Route:          "/ocorrencias",
		Permissions: []kernel.PermissionInfo{
			{Key: transport.PermissaoGerir, Description: "Declarar ocorrência, publicar atualização, coordenar e revisar"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Public.Group(func(pub chi.Router) {
		pub.Use(httpserver.RateLimit(m.deps.Logger, m.deps.PublicLimiter, httpserver.ClientIPKey))
		m.handlers.RegisterPublicRoutes(pub)
	})
	m.handlers.RegisterRoutes(r.Authed)
}
