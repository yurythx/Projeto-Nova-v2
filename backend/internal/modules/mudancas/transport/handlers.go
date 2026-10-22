// Package transport expõe as mudanças, a aprovação e os vínculos.
package transport

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/mudancas/application"
	"github.com/yurythx/sismarq/internal/modules/mudancas/domain"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// MudancaRequest é o corpo de criação e edição.
type MudancaRequest struct {
	Titulo          string     `json:"titulo" validate:"required,max=160"`
	Descricao       string     `json:"descricao" validate:"max=4000"`
	Justificativa   string     `json:"justificativa" validate:"max=4000"`
	Tipo            string     `json:"tipo" validate:"required,oneof=padrao normal emergencial"`
	Risco           string     `json:"risco" validate:"required,oneof=baixo medio alto"`
	PlanoReversao   string     `json:"plano_reversao" validate:"max=4000"`
	InicioPlanejado *time.Time `json:"inicio_planejado"`
	FimPlanejado    *time.Time `json:"fim_planejado"`
	ProdutoID       *uuid.UUID `json:"produto_id"`
	UnidadeID       *uuid.UUID `json:"unidade_id"`
}

func (req MudancaRequest) entrada() domain.Entrada {
	return domain.Entrada{Titulo: req.Titulo, Descricao: req.Descricao, Justificativa: req.Justificativa, Tipo: req.Tipo,
		Risco: req.Risco, PlanoReversao: req.PlanoReversao, InicioPlanejado: req.InicioPlanejado, FimPlanejado: req.FimPlanejado,
		ProdutoID: req.ProdutoID, UnidadeID: req.UnidadeID}
}

// EstadoRequest é a passagem pedida. Texto é o parecer (aprovar ou
// rejeitar) ou o resultado (concluir ou registrar a falha).
type EstadoRequest struct {
	Estado string `json:"estado" validate:"required,oneof=em_avaliacao aprovada rejeitada em_execucao concluida falhou cancelada"`
	Texto  string `json:"texto" validate:"max=4000"`
}

// Handlers da API de mudanças.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes: quem gere (mudancas:manage) ou aprova (mudancas:aprovar)
// vê e muda o estado conforme o papel — o serviço decide. Criar, editar e
// vincular são do gestor.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/mudancas", h.Listar)
	r.Get("/mudancas/{id}", h.Obter)
	r.Post("/mudancas/{id}/estado", h.Transicionar)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermMudancasManage))
		r.Post("/mudancas", h.Criar)
		r.Put("/mudancas/{id}", h.Atualizar)
		r.Put("/mudancas/{id}/vinculos/{tipo}/{alvo}", h.Vincular)
		r.Delete("/mudancas/{id}/vinculos/{tipo}/{alvo}", h.Desvincular)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// Listar aceita ?estado= e ?alvo_tipo=&alvo_id= (as mudanças ligadas a um
// problema, incidente ou ativo).
func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	alvo, err := httputil.OptionalUUIDQuery(r, "alvo_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Listar(r.Context(), identity, application.FiltroEntrada{
		Estado: httputil.Query(r, "estado", 20), AlvoTipo: httputil.Query(r, "alvo_tipo", 20), AlvoID: alvo,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Obter(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.Obter(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req MudancaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.Criar(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, m)
}

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req MudancaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.Atualizar(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}

func (h *Handlers) Transicionar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EstadoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.Transicionar(r.Context(), identity, id, req.Estado, req.Texto)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}

func (h *Handlers) Vincular(w http.ResponseWriter, r *http.Request)    { h.vincular(w, r, true) }
func (h *Handlers) Desvincular(w http.ResponseWriter, r *http.Request) { h.vincular(w, r, false) }

// vincular: PUT liga e DELETE desfaz; repetir não é erro.
func (h *Handlers) vincular(w http.ResponseWriter, r *http.Request, ligar bool) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	alvo, err := httputil.UUIDParam(r, "alvo")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.Vincular(r.Context(), identity, id, chi.URLParam(r, "tipo"), alvo, ligar)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}
