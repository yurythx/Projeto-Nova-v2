// Package blog é o plugin Blog: comunicados e notícias
// com fluxo rascunho -> publicação; publicar emite blog.post.published.
package blog

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/sismarq/internal/modules/blog/application"
	"github.com/yurythx/sismarq/internal/modules/blog/infrastructure"
	"github.com/yurythx/sismarq/internal/modules/blog/transport"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/httpserver"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/modkit"
	"github.com/yurythx/sismarq/internal/platform/search"
)

// Key é a chave do módulo.
const Key = "blog"

// Module é o plugin.
type Module struct {
	deps     modkit.Deps
	svc      *application.Service
	handlers *transport.Handlers
}

// New constrói o módulo.
func New(deps modkit.Deps) *Module {
	svc := application.NewService(deps.Pool, infrastructure.NewRepository(), deps.Outbox, deps.Storage,
		deps.Config.MinIO.Bucket, deps.Config.Upload.URLExpiry, deps.Logger)
	return &Module{deps: deps, svc: svc, handlers: transport.NewHandlers(svc, deps.Logger, deps.Config.MaxPageSize)}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Blog",
		Description:    "Comunicados e notícias (rascunho, publicação e arquivo).",
		DefaultEnabled: false,
		Public:         true,
		Icon:           "newspaper",
		Route:          "/gestao/blog",
		Permissions:    []kernel.PermissionInfo{{Key: "blog:manage", Description: "Criar, editar, publicar e arquivar publicações"}},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Public.Group(func(pub chi.Router) {
		pub.Use(httpserver.RateLimit(m.deps.Logger, m.deps.PublicLimiter, httpserver.ClientIPKey))
		m.handlers.RegisterPublicRoutes(pub)
	})
	m.handlers.RegisterAdminRoutes(r.Authed)
}

// SearchProviders implementa kernel.SearchProvider.
func (m *Module) SearchProviders() []search.Provider { return []search.Provider{provider{m.svc}} }

type provider struct{ svc *application.Service }

func (provider) Module() string { return Key }

func (p provider) Search(ctx context.Context, identity auth.Identity, q string, limit int) ([]search.Result, error) {
	posts, ranks, err := p.svc.Search(ctx, identity, q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]search.Result, 0, len(posts))
	for i, post := range posts {
		updated := post.UpdatedAt
		out = append(out, search.Result{
			Module: Key, Type: post.Kind, ID: post.ID.String(), Title: post.Title,
			Snippet: modkit.Snippet(post.Summary, 180), URL: "/blog/" + post.Slug, Score: ranks[i], UpdatedAt: &updated,
		})
	}
	return out, nil
}
