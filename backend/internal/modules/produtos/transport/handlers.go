// Package transport expõe o cadastro de produtos digitais.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/produtos/application"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// ProdutoRequest é o corpo de criação e de edição. Na edição o estágio
// informado não muda o ciclo: isso é o caso de uso /estagio. Na criação
// o domínio recusa estágio vazio.
type ProdutoRequest struct {
	Nome      string     `json:"nome" validate:"required,max=160"`
	Descricao string     `json:"descricao" validate:"max=4000"`
	Estagio   string     `json:"estagio" validate:"omitempty,oneof=descoberta desenho aquisicao construcao transicao operacao entrega suporte"`
	Utilidade string     `json:"utilidade" validate:"max=4000"`
	Garantia  string     `json:"garantia" validate:"max=4000"`
	UnidadeID *uuid.UUID `json:"unidade_id"`
}

// EstagioRequest muda o estágio do ciclo de vida.
type EstagioRequest struct {
	Estagio string `json:"estagio" validate:"required,oneof=descoberta desenho aquisicao construcao transicao operacao entrega suporte"`
}

// Handlers da API de produtos.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta o cadastro. Tudo exige produtos:manage; o serviço
// ainda restringe pela unidade dona.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermProdutosManage))
		r.Get("/produtos", h.Listar)
		r.Post("/produtos", h.Criar)
		r.Get("/produtos/{id}", h.Obter)
		r.Put("/produtos/{id}", h.Atualizar)
		r.Post("/produtos/{id}/estagio", h.MudarEstagio)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (req ProdutoRequest) entrada() application.Entrada {
	return application.Entrada{
		Nome: req.Nome, Descricao: req.Descricao, Estagio: req.Estagio,
		Utilidade: req.Utilidade, Garantia: req.Garantia, UnidadeID: req.UnidadeID,
	}
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Listar(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req ProdutoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.Criar(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, p)
}

func (h *Handlers) Obter(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.Obter(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ProdutoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.Atualizar(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) MudarEstagio(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EstagioRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	p, err := h.svc.MudarEstagio(r.Context(), identity, id, req.Estagio)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}
