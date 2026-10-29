// Package ativos é o plugin dos itens de configuração: o que sustenta os
// produtos digitais. Nasce desligado. Plugins não se importam: o produto
// chega por uma porta ligada em internal/app.
package ativos

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "ativos"

// Module é o plugin.
type Module struct {
	repo     domain.Repository
	handlers *transport.Handlers
	svc      *application.Service
}

// New constrói o módulo. produtos confere o produto que o ativo sustenta;
// nil deixa o ativo sem esse vínculo.
func New(deps modkit.Deps, produtos domain.Produtos) *Module {
	repo := infrastructure.NewRepository()
	svc := application.NewService(deps.Pool, repo, deps.Outbox, produtos)
	return &Module{repo: repo, handlers: transport.NewHandlers(svc, deps.Logger), svc: svc}
}

// ComPassagens liga os caminhos acrescentados no quadro.
func (m *Module) ComPassagens(p application.Passagens) {
	if m != nil && m.svc != nil {
		m.svc.ComPassagens(p)
	}
}

// Reconhecer autentica a credencial da máquina no middleware. Não é uma conta.
func (m *Module) Reconhecer(ctx context.Context, token string) (auth.Identity, bool, error) {
	if m == nil || m.svc == nil {
		return auth.Identity{}, false, nil
	}
	return m.svc.Reconhecer(ctx, token)
}

// Repositorio é a porta usada por problemas e mudanças para conferir o
// ativo sem importar este módulo.
func (m *Module) Repositorio() domain.Repository { return m.repo }

// ContagemPrograma é a conta do nome que o agente viu.
type ContagemPrograma struct {
	Instalados    int
	ComInventario int
	Parcial       bool
}

// ContarPrograma lê o inventário. O módulo de contratos não importa este pacote.
func (m *Module) ContarPrograma(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, nome string) (ContagemPrograma, error) {
	c, err := infrastructure.ContarPrograma(ctx, db, restrito, unidades, nome)
	if err != nil {
		return ContagemPrograma{}, err
	}
	return ContagemPrograma{Instalados: c.Instalados, ComInventario: c.ComInventario, Parcial: c.Parcial}, nil
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Ativos",
		Description:    "Itens de configuração que sustentam os produtos digitais: equipamentos, sistemas, licenças e links.",
		DefaultEnabled: false,
		Icon:           "server",
		Route:          "/ativos",
		Permissions: []kernel.PermissionInfo{
			{Key: "ativos:read", Description: "Consultar os ativos da unidade e as relações de dependência"},
			{Key: "ativos:manage", Description: "Cadastrar os ativos da unidade e ligá-los aos produtos"},
			{Key: "ativos:reportar", Description: "A própria máquina informa patrimônio e capacidade, sem alterar o cadastro feito à mão"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	m.handlers.RegisterRoutes(r.Authed)
}
