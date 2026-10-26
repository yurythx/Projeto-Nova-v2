// Package quadros é o catálogo das colunas dos quadros kanban. Não tem
// tela própria: quadro e lista leem daqui o nome, a cor e a ordem.
package quadros

import (
	"context"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "quadros"

// Module é o plugin.
type Module struct {
	handlers *transport.Handlers
	repo     *infrastructure.Repository
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox)
	return &Module{handlers: transport.NewHandlers(svc, deps.Logger), repo: repo}
}

// Mapa devolve as passagens acrescentadas neste quadro. Os outros módulos
// recebem este valor sem importar o pacote.
func (m *Module) Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error) {
	if m == nil || m.repo == nil {
		return map[string][]string{}, nil
	}
	return m.repo.Mapa(ctx, db, quadro)
}

// Manifest implementa kernel.Plugin. Sem rota: o catálogo aparece nos
// quadros que já existem. Sem permissão nova: vale a do módulo de cada quadro.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Colunas dos quadros",
		Description:    "Nome, cor e ordem das colunas dos quadros kanban. A chave continua sendo o estado do registro.",
		DefaultEnabled: true,
		Icon:           "folder-kanban",
		Route:          "",
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
