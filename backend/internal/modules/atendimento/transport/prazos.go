package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/atendimento/application"
	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// PoliticaRequest é a política de prazo de uma unidade (sem unidade:
// institucional). As regras de forma ficam no domínio, que explica o erro.
type PoliticaRequest struct {
	UnidadeID *uuid.UUID      `json:"unidade_id"`
	Fuso      string          `json:"fuso" validate:"required,max=64"`
	Turnos    []domain.Turno  `json:"turnos" validate:"max=28"`
	Matriz    []domain.Celula `json:"matriz" validate:"max=9"`
	Metas     []domain.Meta   `json:"metas" validate:"max=4"`
}

// FeriadoRequest é um feriado ou ponto facultativo.
type FeriadoRequest struct {
	UnidadeID *uuid.UUID `json:"unidade_id"`
	Data      string     `json:"data" validate:"required,max=10"`
	Nome      string     `json:"nome" validate:"required,max=160"`
	Tipo      string     `json:"tipo" validate:"required,oneof=nacional estadual municipal ponto_facultativo"`
}

// ClassificacaoRequest são o impacto e a urgência (1 baixo, 3 alto).
type ClassificacaoRequest struct {
	Impacto  int `json:"impacto" validate:"required,min=1,max=3"`
	Urgencia int `json:"urgencia" validate:"required,min=1,max=3"`
}

// ObterPolitica devolve a política que vale para a unidade (null = nenhuma).
func (h *Handlers) ObterPolitica(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.ObterPolitica(r.Context(), identity, unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) SalvarPolitica(w http.ResponseWriter, r *http.Request) {
	var req PoliticaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.SalvarPolitica(r.Context(), identity, application.PoliticaEntrada{
		UnidadeID: req.UnidadeID, Fuso: req.Fuso, Turnos: req.Turnos, Matriz: req.Matriz, Metas: req.Metas,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) ListarFeriados(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarFeriados(r.Context(), identity, unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarFeriado(w http.ResponseWriter, r *http.Request) {
	var req FeriadoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	f, err := h.svc.CriarFeriado(r.Context(), identity, application.FeriadoEntrada{
		UnidadeID: req.UnidadeID, Data: req.Data, Nome: req.Nome, Tipo: req.Tipo,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, f)
}

func (h *Handlers) ExcluirFeriado(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirFeriado(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) Classificar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ClassificacaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.Classificar(r.Context(), identity, id, req.Impacto, req.Urgencia)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}
