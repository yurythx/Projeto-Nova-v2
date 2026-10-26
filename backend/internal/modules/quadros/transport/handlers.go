// Package transport expõe o catálogo das colunas.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// CriarRequest abre uma coluna só neste quadro. de é de onde o card entra.
type CriarRequest struct {
	Nome  string   `json:"nome" validate:"required,min=1,max=80"`
	Cor   string   `json:"cor" validate:"required,max=7"`
	Ordem int      `json:"ordem" validate:"min=0,max=99"`
	De    []string `json:"de" validate:"required,min=1,dive,min=1,max=40"`
	Para  []string `json:"para" validate:"omitempty,dive,min=1,max=40"`
}

// AtualizarRequest altera o rótulo. A chave da URL é o estado e não muda.
type AtualizarRequest struct {
	Nome  *string `json:"nome" validate:"omitempty,min=1,max=80"`
	Cor   *string `json:"cor" validate:"omitempty,max=7"`
	Ordem *int    `json:"ordem" validate:"omitempty,min=0,max=99"`
	Ativa *bool   `json:"ativa"`
}

// Handlers agrupa os handlers HTTP.
type Handlers struct {
	service *application.Service
	logger  *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(service *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{service: service, logger: logger}
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// Listar devolve as colunas do quadro, ativas primeiro.
func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.IdentityFromContext(r.Context())
	cols, err := h.service.Listar(r.Context(), id, chi.URLParam(r, "quadro"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, cols)
}

// Criar grava a coluna nova e as passagens deste quadro.
func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req CriarRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	id, _ := auth.IdentityFromContext(r.Context())
	col, err := h.service.Criar(r.Context(), id, chi.URLParam(r, "quadro"), domain.Nova{
		Nome: req.Nome, Cor: req.Cor, Ordem: req.Ordem, De: req.De, Para: req.Para,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, col)
}

// Atualizar grava nome, cor, ordem ou se a coluna continua no quadro.
func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	var req AtualizarRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	id, _ := auth.IdentityFromContext(r.Context())
	col, err := h.service.Atualizar(r.Context(), id, chi.URLParam(r, "quadro"), chi.URLParam(r, "chave"), domain.Patch{
		Nome: req.Nome, Cor: req.Cor, Ordem: req.Ordem, Ativa: req.Ativa,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, col)
}

// RegisterRoutes monta as rotas. A permissão é a do módulo daquele quadro,
// conferida no serviço — não há uma permissão nova.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/quadros", func(r chi.Router) {
		r.Get("/{quadro}", h.Listar)
		r.Post("/{quadro}", h.Criar)
		r.Put("/{quadro}/{chave}", h.Atualizar)
	})
}
