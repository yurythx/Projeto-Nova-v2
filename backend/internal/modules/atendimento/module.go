// Package atendimento é o plugin da central de atendimento: ofertas,
// equipes e solicitações (incidente e requisição). Nasce desligado.
// Plugins não se importam: o produto chega por uma porta ligada em
// internal/app.
package atendimento

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/messaging"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/ws"
)

// Key é a chave do módulo no Kernel.
const Key = "atendimento"

// QueueTempoReal recebe as mudanças das solicitações para avisar as telas
// abertas (fila, Kanban, detalhe e a lista de quem abriu).
var QueueTempoReal = messaging.QueueSpec{
	Name:    "nova.atendimento.tempo_real",
	DLQName: "nova.atendimento.tempo_real.dlq",
	RoutingKeys: []string{
		application.EventoSolicitacaoAberta, application.EventoSolicitacaoEstado,
		application.EventoSolicitacaoAtribuida, application.EventoSolicitacaoComentada,
		application.EventoSolicitacaoClassificada,
	},
}

// Module é o plugin.
type Module struct {
	repo     domain.Repository
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo. produtos confere a oferta que aponta para um
// produto digital; nil deixa a oferta sem esse vínculo.
func New(deps modkit.Deps, produtos domain.Produtos) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox, produtos).
		WithNotificacoes(deps.Notificacoes).WithLogger(deps.Logger)
	if deps.Storage != nil && deps.Config != nil {
		svc.WithArmazenamento(application.Armazenamento{
			Store: deps.Storage, Bucket: deps.Config.MinIO.Bucket,
			MaxBytes: deps.Config.Upload.MaxFileBytes, Expiry: deps.Config.Upload.URLExpiry,
		})
	}
	if deps.Hub != nil {
		svc.WithTempoReal(deps.Hub)
	}
	return &Module{repo: repo, svc: svc, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Repositorio é a porta usada por problemas e mudanças para conferir o
// incidente sem importar este módulo.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Atendimento",
		Description:    "O servidor abre incidente ou requisição e acompanha os seus. A equipe da unidade atende a fila.",
		DefaultEnabled: false,
		Icon:           "clipboard-list",
		Route:          "/atendimento",
		Permissions: []kernel.PermissionInfo{
			{Key: "atendimento:abrir", Description: "Abrir solicitação pela oferta e acompanhar as próprias"},
			{Key: "atendimento:atender", Description: "Atender a fila da unidade: assumir, responder e mudar o estado"},
			{Key: "atendimento:manage", Description: "Cadastrar ofertas e equipes da unidade"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}

// Consumers implementa kernel.ConsumerProvider.
func (m *Module) Consumers() []kernel.Consumer {
	return []kernel.Consumer{{Queue: QueueTempoReal, Handler: m.svc.HandleEvento}}
}

// WSAuthorizer implementa kernel.WSProvider.
func (m *Module) WSAuthorizer() ws.Authorizer { return m.svc.Authorize }

// WSInbound implementa kernel.WSProvider: o atendimento não recebe frames.
func (m *Module) WSInbound() ws.InboundHandler { return nil }

// Workers implementa kernel.WorkerProvider: o alerta de prazo e a
// retenção da localização das visitas.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{
		{Name: "alertas-de-prazo", Process: kernel.ProcessWorker, Run: m.svc.RunAlertas},
		{Name: "retencao-da-localizacao", Process: kernel.ProcessWorker, Run: m.svc.RunRetencao},
	}
}

// ExportPersonalData implementa lgpd.PersonalData.
func (m *Module) ExportPersonalData(ctx context.Context, db database.DBTX, user uuid.UUID) (any, error) {
	return m.svc.ExportPersonalData(ctx, db, user)
}

// ErasePersonalData implementa lgpd.PersonalData.
func (m *Module) ErasePersonalData(ctx context.Context, tx database.DBTX, user uuid.UUID) error {
	return m.svc.ErasePersonalData(ctx, tx, user)
}
