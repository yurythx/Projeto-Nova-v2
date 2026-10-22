// Package calendar é o plugin Agenda: eventos e reserva
// de salas (calendar:manage para curadoria e moderação). Nasce desligado.
package calendar

import (
	"github.com/go-chi/chi/v5"

	"github.com/yurythx/sismarq/internal/modules/calendar/application"
	"github.com/yurythx/sismarq/internal/modules/calendar/infrastructure"
	"github.com/yurythx/sismarq/internal/modules/calendar/transport"
	"github.com/yurythx/sismarq/internal/platform/httpserver"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/modkit"
)

// Key é a chave do módulo.
const Key = "calendar"

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
		Name:           "Agenda",
		Description:    "Eventos e reserva de salas sem conflito de horário; agenda pública em /eventos.",
		DefaultEnabled: false,
		Public:         true,
		Icon:           "calendar",
		Route:          "/agenda",
		Permissions:    []kernel.PermissionInfo{{Key: "calendar:manage", Description: "Gerenciar salas e moderar eventos de terceiros"}},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Public.Group(func(pub chi.Router) {
		pub.Use(httpserver.RateLimit(m.deps.Logger, m.deps.PublicLimiter, httpserver.ClientIPKey))
		m.handlers.RegisterPublicRoutes(pub)
	})
	m.handlers.RegisterAuthedRoutes(r.Authed)
}
