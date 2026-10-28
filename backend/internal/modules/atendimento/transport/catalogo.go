package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// CategoriasDoCatalogo devolve as categorias ativas para a central do
// solicitante (ADR 078).
func (h *Handlers) CategoriasDoCatalogo(w http.ResponseWriter, r *http.Request) {
	lista, err := h.svc.CategoriasDoCatalogo(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) ListarCategorias(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarCategorias(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarCategoria(w http.ResponseWriter, r *http.Request) {
	var req CategoriaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.CriarCategoria(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, c)
}

func (h *Handlers) AtualizarCategoria(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req CategoriaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.AtualizarCategoria(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}

func (h *Handlers) ApagarCategoria(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ApagarCategoria(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
