// Package cofre é o plugin do cofre de senhas.
// Nasce desligado. A senha não entra na lista nem na auditoria.
package cofre

import (
	"context"
	"log/slog"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/caixa"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// Key é a chave do módulo no Kernel.
const Key = "cofre"

// Module é o plugin.
type Module struct {
	svc      *application.Service
	handlers *transport.Handlers
	logger   *slog.Logger
}

// New constrói o módulo. Chave mestra inválida impede o processo de subir.
func New(deps modkit.Deps) *Module {
	cifra, err := caixa.Nova(deps.Config.Security.CofreMaster())
	if err != nil {
		panic("cofre: " + err.Error())
	}
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox, cifra)
	return &Module{svc: svc, handlers: transport.NewHandlers(svc, deps.Logger), logger: deps.Logger}
}

// Servico é a porta que o atendimento consulta sem importar este pacote.
func (m *Module) Servico() *application.Service { return m.svc }

// ComConfirmacao liga a senha de acesso e o bloqueio do login.
func (m *Module) ComConfirmacao(c application.Confirmador, f application.Freio) {
	m.svc.ComConfirmacao(c, f)
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Cofre",
		Description:    "Senhas da unidade. A lista não mostra o segredo; revelar fica na auditoria.",
		DefaultEnabled: false,
		Icon:           "lock",
		Route:          "/cofre",
		Permissions: []kernel.PermissionInfo{
			{Key: "cofre:read", Description: "Ver nome, tipo e endereço dos segredos da unidade"},
			{Key: "cofre:reveal", Description: "Revelar a senha, com registro na auditoria"},
			{Key: "cofre:manage", Description: "Guardar, trocar e excluir segredos da unidade"},
			{Key: "cofre:grant", Description: "Mover o segredo para outra unidade do seu alcance"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) { m.handlers.RegisterRoutes(r.Authed) }

// Workers implementa kernel.WorkerProvider. Só corre com o módulo ligado.
func (m *Module) Workers() []kernel.Worker {
	return []kernel.Worker{{
		Name:    "rotacao-do-cofre",
		Process: kernel.ProcessWorker,
		Run:     m.runRotacao,
	}}
}

func (m *Module) runRotacao(ctx context.Context) error {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := m.svc.RotacionarDevidas(ctx); err != nil && ctx.Err() == nil && m.logger != nil {
			m.logger.WarnContext(ctx, "cofre: rotação falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
