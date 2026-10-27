package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// MetasXLARequest troca as metas de experiência. Campo nulo = sem meta.
type MetasXLARequest struct {
	CSATMin        *float64 `json:"csat_min" validate:"omitempty,min=1,max=5"`
	FacilidadeMin  *float64 `json:"facilidade_min" validate:"omitempty,min=1,max=5"`
	ReaberturaMax  *float64 `json:"reabertura_max" validate:"omitempty,min=0,max=100"`
	IdasMax        *float64 `json:"idas_max" validate:"omitempty,min=0,max=20"`
	EsperaMaxHoras *int     `json:"espera_max_horas" validate:"omitempty,min=1,max=8760"`
}

func (h *Handlers) MetasXLA(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.MetasXLA(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}

func (h *Handlers) SalvarMetasXLA(w http.ResponseWriter, r *http.Request) {
	var req MetasXLARequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.SalvarMetasXLA(r.Context(), identity, domain.MetasXLA{
		CSATMin: req.CSATMin, FacilidadeMin: req.FacilidadeMin, ReaberturaMax: req.ReaberturaMax,
		IdasMax: req.IdasMax, EsperaMaxHoras: req.EsperaMaxHoras,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}
