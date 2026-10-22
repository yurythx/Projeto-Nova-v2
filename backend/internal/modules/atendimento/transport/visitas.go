package transport

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/atendimento/application"
	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// CheckinRequest é o check-in registrado no aparelho. O id é gerado lá:
// reenviar (fila sem rede) não duplica a visita.
type CheckinRequest struct {
	ID          uuid.UUID       `json:"id" validate:"required"`
	Local       string          `json:"local" validate:"max=300"`
	Destino     *domain.Destino `json:"destino"`
	Ponto       domain.Ponto    `json:"ponto"`
	CapturadaEm time.Time       `json:"capturada_em" validate:"required"`
}

// CheckoutRequest é o check-out registrado no aparelho.
type CheckoutRequest struct {
	Ponto         domain.Ponto `json:"ponto"`
	CapturadaEm   time.Time    `json:"capturada_em" validate:"required"`
	AssinanteNome string       `json:"assinante_nome" validate:"max=160"`
	Observacao    string       `json:"observacao" validate:"max=4000"`
}

// EvidenciaUploadRequest pede a URL de envio de uma foto ou assinatura.
type EvidenciaUploadRequest struct {
	Nome        string `json:"nome" validate:"required,max=255"`
	ContentType string `json:"content_type" validate:"required,max=100"`
}

// EvidenciaRequest junta à visita o arquivo já enviado.
type EvidenciaRequest struct {
	Tipo      string `json:"tipo" validate:"required,oneof=foto assinatura"`
	ObjectKey string `json:"object_key" validate:"required,max=500"`
}

// EvidenciaURLResponse é o endereço temporário da evidência.
type EvidenciaURLResponse struct {
	URL string `json:"url"`
}

func (h *Handlers) IniciarVisita(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req CheckinRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	v, err := h.svc.IniciarVisita(r.Context(), identity, id, application.CheckinEntrada{
		ID: req.ID, Local: req.Local, Destino: req.Destino, Ponto: req.Ponto, CapturadaEm: req.CapturadaEm,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, v)
}

func (h *Handlers) ConcluirVisita(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req CheckoutRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	v, err := h.svc.ConcluirVisita(r.Context(), identity, id, application.CheckoutEntrada{
		Ponto: req.Ponto, CapturadaEm: req.CapturadaEm, AssinanteNome: req.AssinanteNome, Observacao: req.Observacao,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, v)
}

func (h *Handlers) CancelarVisita(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	v, err := h.svc.CancelarVisita(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, v)
}

func (h *Handlers) UploadEvidencia(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EvidenciaUploadRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	t, err := h.svc.UploadEvidencia(r.Context(), identity, id, req.Nome, req.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, t)
}

func (h *Handlers) RegistrarEvidencia(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EvidenciaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	e, err := h.svc.RegistrarEvidencia(r.Context(), identity, id, req.Tipo, req.ObjectKey)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, e)
}

func (h *Handlers) URLEvidencia(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	evidencia, err := httputil.UUIDParam(r, "evidencia")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	url, err := h.svc.URLEvidencia(r.Context(), identity, id, evidencia)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, EvidenciaURLResponse{URL: url})
}

// VisitasEmCampo é a supervisão: só visitas em andamento, com a posição.
func (h *Handlers) VisitasEmCampo(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.VisitasEmCampo(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}
