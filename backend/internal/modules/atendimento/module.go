// Package atendimento é o plugin da central de atendimento: ofertas,
// equipes e solicitações (incidente e requisição). Nasce desligado.
// Plugins não se importam: o produto chega por uma porta ligada em
// internal/app.
package atendimento

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
	"github.com/yurythx/projeto-nova-v2/internal/platform/messaging"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
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
	correio  bool
}

// ComPassagens liga os caminhos acrescentados na fila.
func (m *Module) ComPassagens(p application.Passagens) {
	if m != nil && m.svc != nil {
		m.svc.ComPassagens(p)
	}
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
	correio := false
	if deps.Config != nil && deps.Config.Mail.IMAPHost != "" {
		correio = true
		svc.WithCaixa(mail.IMAP{
			Host: deps.Config.Mail.IMAPHost, Porta: deps.Config.Mail.IMAPPort,
			Usuario: deps.Config.Mail.IMAPUser, Senha: deps.Config.Mail.IMAPPassword,
		})
	}
	handlers := transport.NewHandlers(svc, deps.Logger)
	if deps.Config != nil {
		handlers.ComTokenDeEventos(deps.Config.Security.EventosToken)
	}
	return &Module{repo: repo, svc: svc, handlers: handlers, correio: correio}
}

// Repositorio é a porta usada por problemas e mudanças para conferir o
// incidente sem importar este módulo.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// ComSegredo liga a porta do cofre. Sem ela, o chamado não recebe segredo.
func (m *Module) ComSegredo(segredos application.Segredos) { m.svc.WithSegredo(segredos) }

// ComContratos liga os contratos de apoio (ADR 071). Sem ele, o chamado
// não aponta contrato.
func (m *Module) ComContratos(c application.ContratosDeApoio) { m.svc.WithContratos(c) }

// ComAssistente liga a IA de quem atende (ADR 065). Sem ela, não há IA.
func (m *Module) ComAssistente(a application.Assistente) { m.svc.WithAssistente(a) }

// ConcluirPorMudanca conclui o incidente na transação de quem chama.
func (m *Module) ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, numero int64, titulo, texto string) (func(context.Context), error) {
	return m.svc.ConcluirPorMudanca(ctx, tx, id, autor, numero, titulo, texto)
}

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
	// Sem limite por IP: o Alertmanager mora na rede interna e manda
	// rajadas; o token é a proteção.
	m.handlers.RegisterPublicRoutes(r.Public)
}

// SearchProviders implementa kernel.SearchProvider. Não é search.Indexer:
// solicitação tem dono e escopo, e o índice público não guarda isso.
func (m *Module) SearchProviders() []search.Provider { return []search.Provider{provider{m.svc}} }

type provider struct{ svc *application.Service }

func (provider) Module() string { return Key }

func (p provider) Search(ctx context.Context, identity auth.Identity, q string, limit int) ([]search.Result, error) {
	itens, err := p.svc.Search(ctx, identity, q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]search.Result, 0, len(itens))
	for _, sol := range itens {
		atualizado := sol.UpdatedAt
		out = append(out, search.Result{
			Module: Key, Type: "solicitacao", ID: sol.ID.String(),
			Title:     fmt.Sprintf("#%d — %s", sol.Numero, sol.Titulo),
			Snippet:   sol.OfertaNome + " · " + sol.Estado,
			URL:       "/atendimento/solicitacoes/" + sol.ID.String(),
			Score:     0.5,
			UpdatedAt: &atualizado,
		})
	}
	return out, nil
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
	out := []kernel.Worker{
		{Name: "alertas-de-prazo", Process: kernel.ProcessWorker, Run: m.svc.RunAlertas},
		{Name: "retencao-da-localizacao", Process: kernel.ProcessWorker, Run: m.svc.RunRetencao},
		{Name: "serie-de-indicadores", Process: kernel.ProcessWorker, Run: m.svc.RunSerie},
		{Name: "recorrencias", Process: kernel.ProcessWorker, Run: m.svc.RunRecorrencias},
	}
	if m.correio {
		out = append(out, kernel.Worker{Name: "correio-do-atendimento", Process: kernel.ProcessWorker, Run: m.svc.RunCorreio})
	}
	return out
}

// ExportPersonalData implementa lgpd.PersonalData.
func (m *Module) ExportPersonalData(ctx context.Context, db database.DBTX, user uuid.UUID) (any, error) {
	return m.svc.ExportPersonalData(ctx, db, user)
}

// ErasePersonalData implementa lgpd.PersonalData.
func (m *Module) ErasePersonalData(ctx context.Context, tx database.DBTX, user uuid.UUID) error {
	return m.svc.ErasePersonalData(ctx, tx, user)
}
