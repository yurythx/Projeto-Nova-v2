package transport

import (
	"crypto/subtle"
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// WebhookAlertmanager é o corpo do webhook do Alertmanager (versão 4). Todo
// campo do formato está aqui (o decoder recusa os desconhecidos); só os
// alertas são usados.
type WebhookAlertmanager struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []domain.Alerta   `json:"alerts" validate:"required,min=1,max=200"`
}

// ConfiguracaoEventosRequest escolhe a oferta dos alertas (null limpa).
type ConfiguracaoEventosRequest struct {
	OfertaID *uuid.UUID `json:"oferta_id"`
}

// ComTokenDeEventos liga a entrada de eventos (ADR 067). Sem token, ela
// responde 404.
func (h *Handlers) ComTokenDeEventos(token string) *Handlers {
	h.tokenEventos = token
	return h
}

// ReceberAlertas é o webhook do Alertmanager. Não há sessão de pessoa: o
// token da máquina é conferido em tempo constante.
func (h *Handlers) ReceberAlertas(w http.ResponseWriter, r *http.Request) {
	if h.tokenEventos == "" {
		http.NotFound(w, r)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+h.tokenEventos)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req WebhookAlertmanager
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	res, err := h.svc.ReceberAlertas(r.Context(), req.Alerts)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, res)
}

func (h *Handlers) ConfiguracaoDosEventos(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.ConfiguracaoDosEventos(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}

func (h *Handlers) ConfigurarEventos(w http.ResponseWriter, r *http.Request) {
	var req ConfiguracaoEventosRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.ConfigurarEventos(r.Context(), identity, req.OfertaID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}
