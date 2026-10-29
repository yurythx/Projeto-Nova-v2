// Package transport expõe o cadastro de ativos.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
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

// RegisterRoutes monta o cadastro. O relato da máquina exige
// ativos:reportar; a leitura serve a quem gere ou consulta na cadeia de
// assistência/projetos; alterar exige ativos:manage.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/ativos", h.Listar)
	r.Get("/ativos/{id}", h.Obter)
	r.Get("/ativos/{id}/relacoes", h.ListarRelacoes)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtivosReportar))
		r.Post("/ativos/inventario", h.Inventariar)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtivosManage))
		r.Get("/ativos/credenciais", h.ListarCredenciais)
		r.Post("/ativos/credenciais", h.EmitirCredencial)
		r.Delete("/ativos/credenciais/{id}", h.RevogarCredencial)
		r.Post("/ativos", h.Criar)
		r.Put("/ativos/{id}", h.Atualizar)
		r.Post("/ativos/{id}/relacoes", h.Relacionar)
		r.Delete("/ativos/{id}/relacoes/{relacao}", h.Desrelacionar)
	})
}

type InventarioRequest struct {
	Identificador string             `json:"identificador" validate:"required,max=80"`
	Nome          string             `json:"nome" validate:"max=160"`
	Tipo          string             `json:"tipo" validate:"omitempty,oneof=hardware software servico rede outro"`
	Descricao     string             `json:"descricao" validate:"max=4000"`
	UnidadeID     *uuid.UUID         `json:"unidade_id"`
	Inventario    *domain.Inventario `json:"inventario"`
}

func (h *Handlers) Inventariar(w http.ResponseWriter, r *http.Request) {
	var req InventarioRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	var unidade uuid.UUID
	if req.UnidadeID != nil {
		unidade = *req.UnidadeID
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.Inventariar(r.Context(), identity, domain.Relato{
		Identificador: req.Identificador, Nome: req.Nome, Tipo: req.Tipo,
		Descricao: req.Descricao, UnidadeID: unidade, Inventario: req.Inventario,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

type CredencialRequest struct {
	Nome      string    `json:"nome" validate:"required,max=160"`
	UnidadeID uuid.UUID `json:"unidade_id" validate:"required"`
}

func (h *Handlers) EmitirCredencial(w http.ResponseWriter, r *http.Request) {
	var req CredencialRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.EmitirCredencial(r.Context(), identity, req.Nome, req.UnidadeID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) ListarCredenciais(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarCredenciais(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) RevogarCredencial(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.RevogarCredencial(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
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

type RelacaoRequest struct {
	DestinoID uuid.UUID `json:"destino_id" validate:"required"`
	Tipo      string    `json:"tipo" validate:"required,oneof=depende composto conectado"`
}

func (h *Handlers) ListarRelacoes(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarRelacoes(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Relacionar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req RelacaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.Relacionar(r.Context(), identity, id, req.DestinoID, req.Tipo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) Desrelacionar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	relacao, err := httputil.UUIDParam(r, "relacao")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.Desrelacionar(r.Context(), identity, id, relacao); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
