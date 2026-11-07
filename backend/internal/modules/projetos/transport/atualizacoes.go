package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

func (h *Handlers) ListarPrazos(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarPrazos(r.Context(), identity, httputil.Query(r, "mes", 7))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) ListarAtualizacoes(w http.ResponseWriter, r *http.Request) {
	projetoID, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	tarefaID, err := httputil.UUIDParam(r, "tarefaId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarAtualizacoes(r.Context(), identity, projetoID, tarefaID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarAtualizacao(w http.ResponseWriter, r *http.Request) {
	projetoID, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	tarefaID, err := httputil.UUIDParam(r, "tarefaId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req struct {
		Texto string `json:"texto"`
	}
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	criada, err := h.svc.CriarAtualizacao(r.Context(), identity, projetoID, tarefaID, req.Texto)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, criada)
}
