package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// RegraRequest é a triagem curta. A primeira que casa vale.
type RegraRequest struct {
	Nome                 string     `json:"nome" validate:"required,max=160"`
	Ordem                int        `json:"ordem" validate:"min=0,max=999"`
	OfertaID             *uuid.UUID `json:"oferta_id"`
	Palavra              string     `json:"palavra" validate:"max=80"`
	UnidadeSolicitanteID *uuid.UUID `json:"unidade_solicitante_id"`
	Prioridade           string     `json:"prioridade" validate:"omitempty,oneof=critica alta media baixa"`
	EquipeID             *uuid.UUID `json:"equipe_id"`
	UnidadeID            *uuid.UUID `json:"unidade_id"`
}

// RespostaRequest é o texto que a equipe cola. Não dispara sozinho.
type RespostaRequest struct {
	Publico string `json:"publico" validate:"max=4000"`
	Interna string `json:"interna" validate:"max=4000"`
}

// RecorrenciaRequest abre um chamado todo mês, no dia pedido.
type RecorrenciaRequest struct {
	OfertaID  uuid.UUID `json:"oferta_id" validate:"required"`
	Titulo    string    `json:"titulo" validate:"required,max=160"`
	Descricao string    `json:"descricao" validate:"max=4000"`
	Dia       int       `json:"dia" validate:"required,min=1,max=28"`
}

// DeflexaoRequest anota os artigos vistos antes de abrir o pedido.
type DeflexaoRequest struct {
	Artigos  []string `json:"artigos" validate:"required,min=1,max=4,dive,required,max=80"`
	Resolveu bool     `json:"resolveu"`
}

func (h *Handlers) ListarRegras(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarRegras(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarRegra(w http.ResponseWriter, r *http.Request) {
	var req RegraRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarRegra(r.Context(), identity, domain.Regra{
		Nome: req.Nome, Ordem: req.Ordem, OfertaID: req.OfertaID, Palavra: req.Palavra,
		UnidadeSolicitanteID: req.UnidadeSolicitanteID, Prioridade: req.Prioridade,
		EquipeID: req.EquipeID, UnidadeID: req.UnidadeID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) ExcluirRegra(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirRegra(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) ObterResposta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.ObterResposta(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) SalvarResposta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req RespostaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.SalvarResposta(r.Context(), identity, id, req.Publico, req.Interna)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) ListarRecorrencias(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarRecorrencias(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarRecorrencia(w http.ResponseWriter, r *http.Request) {
	var req RecorrenciaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarRecorrencia(r.Context(), identity, domain.ChamadoMensal{
		OfertaID: req.OfertaID, Titulo: req.Titulo, Descricao: req.Descricao, Dia: req.Dia,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) ExcluirRecorrencia(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirRecorrencia(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) RegistrarDeflexao(w http.ResponseWriter, r *http.Request) {
	var req DeflexaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.RegistrarDeflexao(r.Context(), identity, req.Artigos, req.Resolveu); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
