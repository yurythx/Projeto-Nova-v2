// Package transport expõe o cadastro de ativos.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/ativos/application"
	"github.com/yurythx/sismarq/internal/modules/ativos/domain"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// AtivoRequest é o corpo de criação e edição de um ativo.
type AtivoRequest struct {
	Nome          string     `json:"nome" validate:"required,max=160"`
	Tipo          string     `json:"tipo" validate:"required,oneof=hardware software servico rede outro"`
	Estado        string     `json:"estado" validate:"required,oneof=planejado em_uso manutencao desativado"`
	Identificador string     `json:"identificador" validate:"max=80"`
	Descricao     string     `json:"descricao" validate:"max=4000"`
	ProdutoID     *uuid.UUID `json:"produto_id"`
	UnidadeID     *uuid.UUID `json:"unidade_id"`
}

func (req AtivoRequest) entrada() domain.Entrada {
	return domain.Entrada{Nome: req.Nome, Tipo: req.Tipo, Estado: req.Estado, Identificador: req.Identificador,
		Descricao: req.Descricao, ProdutoID: req.ProdutoID, UnidadeID: req.UnidadeID}
}

// Handlers da API de ativos.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta o cadastro. Tudo exige ativos:manage; o serviço
// ainda restringe pela unidade dona.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtivosManage))
		r.Get("/ativos", h.Listar)
		r.Post("/ativos", h.Criar)
		r.Get("/ativos/{id}", h.Obter)
		r.Put("/ativos/{id}", h.Atualizar)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// Listar aceita ?produto_id= para os ativos que sustentam um produto.
func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	produto, err := httputil.OptionalUUIDQuery(r, "produto_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Listar(r.Context(), identity, produto)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req AtivoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	a, err := h.svc.Criar(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, a)
}

func (h *Handlers) Obter(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	a, err := h.svc.Obter(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, a)
}

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req AtivoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	a, err := h.svc.Atualizar(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, a)
}
