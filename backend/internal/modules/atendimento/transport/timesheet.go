package transport

import (
	"net/http"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// ApontamentoRequest reúne os campos para registrar horas em uma solicitação.
type ApontamentoRequest struct {
	DataApontamento string  `json:"data_apontamento"`
	Horas           float64 `json:"horas" validate:"required,gt=0,lte=24"`
	Descricao       string  `json:"descricao" validate:"required"`
}

// ListarApontamentos devolve os registros de horas da solicitação.
func (h *Handlers) ListarApontamentos(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	resumo, err := h.svc.ListarApontamentos(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, resumo)
}

// RegistrarApontamento adiciona um novo registro de horas trabalhadas pelo técnico.
func (h *Handlers) RegistrarApontamento(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ApontamentoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	criado, err := h.svc.RegistrarApontamento(r.Context(), identity, id, domain.EntradaApontamento{
		DataApontamento: req.DataApontamento,
		Horas:           req.Horas,
		Descricao:       req.Descricao,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, criado)
}

// ExcluirApontamento remove um apontamento de horas da solicitação.
func (h *Handlers) ExcluirApontamento(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	apontamentoID, err := httputil.UUIDParam(r, "apontamentoId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirApontamento(r.Context(), identity, id, apontamentoID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
