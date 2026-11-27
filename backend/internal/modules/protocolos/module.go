// Package protocolos é a carta de serviços e o pedido do cidadão.
// Nasce desligado. Trâmite, Atendimento e Fazenda não são importados.
package protocolos

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/httpserver"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "protocolos"

// Module é o plugin.
type Module struct {
	deps     modkit.Deps
	service  *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	arq := application.Arquivos{Store: deps.Storage}
	if deps.Config != nil {
		arq.Bucket = deps.Config.MinIO.Bucket
		arq.MaxBytes = deps.Config.Upload.MaxFileBytes
		arq.Expiry = deps.Config.Upload.URLExpiry
	}
	svc := application.NewService(deps.Pool, infrastructure.NewPostgresRepository(), deps.Outbox, logger).
		ComSino(deps.Notificacoes).ComArquivos(arq)
	return &Module{deps: deps, service: svc, handlers: transport.NewHandlers(svc, logger)}
}

// Workers cancela a complementação vencida e avisa os prazos que vencem em 24 horas.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{{Name: "prazos-de-protocolos", Process: kernel.ProcessWorker, Run: m.service.RunPrazos}}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Protocolos",
		Description:    "Carta de serviços, formulário e pedido do cidadão.",
		DefaultEnabled: false,
		Public:         true,
		Icon:           "clipboard-list",
		Route:          "/protocolos",
		Permissions: []kernel.PermissionInfo{
			{Key: "protocolos:gerir", Description: "Desenhar formulário e serviço da carta"},
			{Key: "protocolos:tratar", Description: "Tratar pedido na unidade atual"},
			{Key: "protocolos:abrir", Description: "Abrir pedido no balcão"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Public.Group(func(pub chi.Router) {
		pub.Use(httpserver.RateLimit(m.deps.Logger, m.deps.PublicLimiter, httpserver.ClientIPKey))
		transport.RegisterPublic(pub, m.handlers)
	})
	transport.RegisterAuthed(r.Authed, m.handlers, m.deps.Logger)
}
