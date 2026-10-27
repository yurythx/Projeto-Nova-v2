// Package integracoes é o plugin de migração de outras ferramentas
// (ADR 076): conexão, área de preparo, tabela de-para e tarefas. Nasce
// desligado. Não conhece a origem: a extração e a carga chegam pelo app.
package integracoes

import (
	"context"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "integracoes"

// Module é o plugin.
type Module struct {
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox, deps.Cipher).WithLogger(deps.Logger)
	return &Module{svc: svc, handlers: transport.NewHandlers(svc, deps.Logger)}
}

// Servico liga a extração e a carga (portas no app).
func (m *Module) Servico() *application.Service { return m.svc }

// Reconhecer autentica o token do conector.
func (m *Module) Reconhecer(ctx context.Context, token string) (auth.Identity, bool, error) {
	return m.svc.Reconhecer(ctx, token)
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Integrações",
		Description:    "Migrar de outra ferramenta (GLPI): extrair, revisar a tabela de-para e carregar, com um conector na rede da origem.",
		DefaultEnabled: false,
		Icon:           "plug",
		Route:          "/integracoes",
		Permissions: []kernel.PermissionInfo{
			{Key: string(auth.PermIntegracoesManage), Description: "Cadastrar conexões, extrair, revisar a tabela de-para e carregar dados de outra ferramenta"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}

// Workers implementa kernel.WorkerProvider: as tarefas rodam no worker.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{{Name: "integracoes-tarefas", Process: kernel.ProcessWorker, Run: m.svc.RunTarefas}}
}
