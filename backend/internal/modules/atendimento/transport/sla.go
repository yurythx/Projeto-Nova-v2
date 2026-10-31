package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// SLARequest é o corpo para cadastrar ou editar uma SLA.
type SLARequest struct {
	Nome                 string     `json:"nome" validate:"required,max=120"`
	Descricao            string     `json:"descricao" validate:"max=500"`
	TempoRespostaMinutos int        `json:"tempo_resposta_minutos" validate:"required,min=1"`
	TempoSolucaoMinutos  int        `json:"tempo_solucao_minutos" validate:"required,min=1"`
	PausarEmEspera       bool       `json:"pausar_em_espera"`
	CalendarioTipo       string     `json:"calendario_tipo" validate:"omitempty,oneof=expediente corrido_24x7"`
	UnidadeID            *uuid.UUID `json:"unidade_id"`
	Ativa                *bool      `json:"ativa"`
}

// VincularCategoriaSLARequest associa uma SLA a uma categoria.
type VincularCategoriaSLARequest struct {
	SLAID *uuid.UUID `json:"sla_id"`
}

// ListarSLAs lista as SLAs disponíveis.
func (h *Handlers) ListarSLAs(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	slas, err := h.svc.ListarSLAs(r.Context(), identity, unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, slas)
}

// ObterSLA busca uma SLA específica.
func (h *Handlers) ObterSLA(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	sla, err := h.svc.ObterSLA(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, sla)
}

// CriarSLA cria uma nova SLA.
func (h *Handlers) CriarSLA(w http.ResponseWriter, r *http.Request) {
	var req SLARequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	sla, err := h.svc.CriarSLA(r.Context(), identity, application.SLAEntrada{
		Nome:                 req.Nome,
		Descricao:            req.Descricao,
		TempoRespostaMinutos: req.TempoRespostaMinutos,
		TempoSolucaoMinutos:  req.TempoSolucaoMinutos,
		PausarEmEspera:       req.PausarEmEspera,
		CalendarioTipo:       req.CalendarioTipo,
		UnidadeID:            req.UnidadeID,
		Ativa:                req.Ativa,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, sla)
}

// AtualizarSLA altera os dados de uma SLA.
func (h *Handlers) AtualizarSLA(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req SLARequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	sla, err := h.svc.AtualizarSLA(r.Context(), identity, id, application.SLAEntrada{
		Nome:                 req.Nome,
		Descricao:            req.Descricao,
		TempoRespostaMinutos: req.TempoRespostaMinutos,
		TempoSolucaoMinutos:  req.TempoSolucaoMinutos,
		PausarEmEspera:       req.PausarEmEspera,
		CalendarioTipo:       req.CalendarioTipo,
		UnidadeID:            req.UnidadeID,
		Ativa:                req.Ativa,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, sla)
}

// ExcluirSLA remove uma SLA.
func (h *Handlers) ExcluirSLA(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirSLA(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

// VincularCategoriaSLA vincula uma SLA a uma categoria.
func (h *Handlers) VincularCategoriaSLA(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req VincularCategoriaSLARequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.AssociarCategoriaSLA(r.Context(), identity, id, req.SLAID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, map[string]string{"status": "ok"})
}

