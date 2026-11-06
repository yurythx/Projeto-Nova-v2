package transport

import (
	"io"
	"net/http"

	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

type pedidoAvatar struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

type confirmarAvatar struct {
	ObjectKey string `json:"object_key"`
}

func (h *Handlers) PedirAvatar(w http.ResponseWriter, r *http.Request) {
	var req pedidoAvatar
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	ticket, err := h.svc.PedirAvatar(r.Context(), authIdentity(r), req.Filename, req.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ticket)
}

func (h *Handlers) SalvarAvatar(w http.ResponseWriter, r *http.Request) {
	var req confirmarAvatar
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.SalvarAvatar(r.Context(), authIdentity(r), req.ObjectKey); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) RemoverAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoverAvatar(r.Context(), authIdentity(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) AbrirAvatar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	rc, tipo, err := h.svc.AbrirAvatar(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "private, max-age=120")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, io.LimitReader(rc, 2<<20+1))
}
