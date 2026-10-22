// Package busca é o plugin Busca Global: mecanismo de busca unificado que
// degrada graciosamente respeitando apenas os módulos ativos.
package busca

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/yurythx/sismarq/internal/modules/busca/application"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/kernel"
	"github.com/yurythx/sismarq/internal/platform/modkit"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// Key é a chave do módulo.
const Key = "search"

// Module é o plugin.
type Module struct {
	svc    *application.Service
	logger *slog.Logger
}

// New constrói o módulo; source é kernel.SearchProviders (só ativos).
func New(deps modkit.Deps, source application.ProviderSource) *Module {
	return &Module{svc: application.NewService(source, deps.Logger), logger: deps.Logger}
}

// WithIndex liga o índice da busca (Typesense, ADR 031).
func (m *Module) WithIndex(idx application.Index) *Module {
	m.svc.WithIndex(idx)
	return m
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Busca Global",
		Description:    "Busca unificada sobre os módulos ativos.",
		DefaultEnabled: false,
		Icon:           "search",
		Route:          "/busca",
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	r.Authed.Get("/search", m.search)
}

func (m *Module) search(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := m.svc.Search(r.Context(), identity, httputil.Query(r, "q", 200), httputil.Query(r, "module", 32), limit)
	if err != nil {
		httputil.WriteError(w, r, m.logger, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httputil.WriteOK(w, resp)
}
