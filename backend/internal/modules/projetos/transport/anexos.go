package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

type anexoUploadRequest struct {
	Nome        string `json:"nome"`
	ContentType string `json:"content_type"`
}

type anexoRequest struct {
	Nome      string `json:"nome"`
	ObjectKey string `json:"object_key"`
}

func (h *Handlers) PedirAnexo(w http.ResponseWriter, r *http.Request) {
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
	var req anexoUploadRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	ticket, err := h.svc.PedirAnexo(r.Context(), identity, projetoID, tarefaID, req.Nome, req.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, ticket)
}

func (h *Handlers) RegistrarAnexo(w http.ResponseWriter, r *http.Request) {
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
	var req anexoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	anexo, err := h.svc.RegistrarAnexo(r.Context(), identity, projetoID, tarefaID, req.Nome, req.ObjectKey)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, anexo)
}

func (h *Handlers) ListarAnexos(w http.ResponseWriter, r *http.Request) {
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
	lista, err := h.svc.ListarAnexos(r.Context(), identity, projetoID, tarefaID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) ListarMovimentos(w http.ResponseWriter, r *http.Request) {
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
	lista, err := h.svc.ListarMovimentos(r.Context(), identity, projetoID, tarefaID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) URLAnexo(w http.ResponseWriter, r *http.Request) {
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
	anexoID, err := httputil.UUIDParam(r, "anexoId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	url, err := h.svc.URLAnexo(r.Context(), identity, projetoID, tarefaID, anexoID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, map[string]string{"url": url})
}
