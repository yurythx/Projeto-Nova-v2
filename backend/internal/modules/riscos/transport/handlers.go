package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, "riscos:read"))
		r.Get("/riscos", h.Listar)
		r.Get("/riscos/heatmap", h.Heatmap)
		r.Get("/riscos/{id}", h.Obter)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, "riscos:manage"))
		r.Post("/riscos", h.Criar)
		r.Put("/riscos/{id}", h.Atualizar)
		r.Delete("/riscos/{id}", h.Excluir)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	cat := r.URL.Query().Get("categoria")
	status := r.URL.Query().Get("status")

	identity, _ := auth.IdentityFromContext(r.Context())
	itens, err := h.svc.ListarRiscos(r.Context(), identity, unidade, cat, status)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, itens)
}

func (h *Handlers) Heatmap(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	res, err := h.svc.Heatmap(r.Context(), identity, unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, res)
}

func (h *Handlers) Obter(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.ObterRisco(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var in application.RiscoEntrada
	if err := httputil.Bind(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarRisco(r.Context(), identity, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var in application.RiscoEntrada
	if err := httputil.Bind(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.AtualizarRisco(r.Context(), identity, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) Excluir(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirRisco(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

