package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// EtapaRequest é uma etapa de aprovação da oferta.
type EtapaRequest struct {
	Nome        string      `json:"nome" validate:"required,max=80"`
	Aprovadores []uuid.UUID `json:"aprovadores" validate:"required,min=1,max=20"`
}

// EtapasRequest troca as etapas da oferta (lista vazia tira a aprovação).
type EtapasRequest struct {
	Etapas []EtapaRequest `json:"etapas" validate:"max=5,dive"`
}

// DecisaoAprovacaoRequest é a decisão de quem aprova a etapa atual.
type DecisaoAprovacaoRequest struct {
	Aprovar *bool  `json:"aprovar" validate:"required"`
	Motivo  string `json:"motivo" validate:"max=1000"`
}

func (h *Handlers) EtapasDaOferta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	etapas, err := h.svc.EtapasDaOferta(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, etapas)
}

func (h *Handlers) SalvarEtapas(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EtapasRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	etapas := make([]domain.EtapaAprovacao, 0, len(req.Etapas))
	for _, e := range req.Etapas {
		etapas = append(etapas, domain.EtapaAprovacao{Nome: e.Nome, Aprovadores: e.Aprovadores})
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	out, err := h.svc.SalvarEtapas(r.Context(), identity, id, etapas)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) AguardandoMinhaAprovacao(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.AguardandoMinhaAprovacao(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) DecidirAprovacao(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req DecisaoAprovacaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	sol, err := h.svc.DecidirAprovacao(r.Context(), identity, id, *req.Aprovar, req.Motivo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, sol)
}
