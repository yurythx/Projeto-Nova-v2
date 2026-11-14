// Package fazenda é o protocolo da Secretaria Municipal de Fazenda para os
// Decretos 13.653/2026 e 13.654/2026. Nasce desligado. A TI não vê o menu
// nem passa na API sem a permissão da unidade.
package fazenda

import (
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
	"github.com/yurythx/projeto-nova-v2/internal/platform/httpserver"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "fazenda"

// Module é o plugin.
type Module struct {
	deps     modkit.Deps
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo. Sem credencial do gov.br, a porta informa isso
// e não inventa sessão.
func New(deps modkit.Deps) *Module {
	gov := govbr.Config{
		ClientID: os.Getenv("GOVBR_CLIENT_ID"), ClientSecret: os.Getenv("GOVBR_CLIENT_SECRET"),
		RedirectURL: os.Getenv("GOVBR_REDIRECT_URL"), Ambiente: os.Getenv("GOVBR_AMBIENTE"),
	}
	svc := application.NewService(deps.Pool, infrastructure.NewPostgresRepository(), application.Nucleo{Out: deps.Outbox}, gov, deps.Logger).ComSino(deps.Notificacoes)
	if deps.Storage != nil && deps.Config != nil {
		svc.ComArquivos(application.Arquivos{
			Store: deps.Storage, Bucket: deps.Config.MinIO.Bucket,
			MaxBytes: deps.Config.Upload.MaxFileBytes, Expiry: deps.Config.Upload.URLExpiry,
		})
	}
	return &Module{deps: deps, svc: svc, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Protocolo da Fazenda",
		Description:    "Avança e Regulariza Rondonópolis. Desligado até a SEFAZ ativar.",
		DefaultEnabled: false,
		Public:         true,
		Icon:           "landmark",
		Route:          "/fazenda",
		Permissions: []kernel.PermissionInfo{
			{Key: "fazenda:protocolar", Description: "Protocolar em nome do contribuinte"},
			{Key: "fazenda:analisar", Description: "Analisar processo da Fazenda"},
			{Key: "fazenda:decidir", Description: "Decidir processo da Fazenda"},
		},
	}
}

// Workers publica o alerta de prazo no ritmo do módulo.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{{
		Name:    "alertas-de-prazo",
		Process: kernel.ProcessWorker,
		Run:     m.svc.RunAlertas,
	}}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Public.Group(func(pub chi.Router) {
		pub.Use(httpserver.RateLimit(m.deps.Logger, m.deps.PublicLimiter, httpserver.ClientIPKey))
		transport.RegisterPublic(pub, m.handlers)
	})
	transport.RegisterAuthed(r.Authed, m.handlers, m.deps.Logger)
}
