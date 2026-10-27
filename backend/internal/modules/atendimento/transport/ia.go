package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// DecisaoRequest é a decisão de quem atende sobre a sugestão da IA.
type DecisaoRequest struct {
	Aceita *bool `json:"aceita" validate:"required"`
}

// EstadoDaIA diz se os botões da IA aparecem.
func (h *Handlers) EstadoDaIA(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	e, err := h.svc.EstadoDaIA(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, e)
}

func (h *Handlers) SugerirClassificacao(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	s, err := h.svc.SugerirClassificacao(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, s)
}

func (h *Handlers) Resumir(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	s, err := h.svc.Resumir(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, s)
}

func (h *Handlers) DecidirSugestao(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req DecisaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	s, err := h.svc.DecidirSugestao(r.Context(), identity, id, *req.Aceita)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, s)
}

func (h *Handlers) IndicadoresIA(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	out, err := h.svc.IndicadoresIA(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}
