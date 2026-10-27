package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// FornecedorRequest é o cadastro de quem presta.
type FornecedorRequest struct {
	Nome      string     `json:"nome" validate:"required,max=160"`
	Email     string     `json:"email" validate:"max=160"`
	Telefone  string     `json:"telefone" validate:"max=40"`
	UnidadeID *uuid.UUID `json:"unidade_id"`
}

// ContratoRequest é a vigência. O valor é em centavos.
type ContratoRequest struct {
	Numero        string     `json:"numero" validate:"max=40"`
	Titulo        string     `json:"titulo" validate:"required,max=160"`
	FornecedorID  uuid.UUID  `json:"fornecedor_id" validate:"required"`
	Inicio        time.Time  `json:"inicio" validate:"required"`
	Fim           time.Time  `json:"fim" validate:"required"`
	ValorCentavos int64      `json:"valor_centavos" validate:"min=0"`
	UnidadeID     *uuid.UUID `json:"unidade_id"`
}

// LicencaRequest é o direito de uso. Sem a chave secreta.
type LicencaRequest struct {
	Nome         string     `json:"nome" validate:"required,max=160"`
	FornecedorID uuid.UUID  `json:"fornecedor_id" validate:"required"`
	Fim          *time.Time `json:"fim"`
	Quantidade   int        `json:"quantidade" validate:"required,min=1"`
	Programa     string     `json:"programa" validate:"max=80"`
	UnidadeID    *uuid.UUID `json:"unidade_id"`
}

// Handlers da API de contratos.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta o cadastro. Tudo exige contratos:manage.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermContratosManage))
		r.Get("/contratos/fornecedores", h.listarFornecedores)
		r.Post("/contratos/fornecedores", h.criarFornecedor)
		r.Delete("/contratos/fornecedores/{id}", h.excluirFornecedor)
		r.Get("/contratos/contratos", h.listarContratos)
		r.Post("/contratos/contratos", h.criarContrato)
		r.Delete("/contratos/contratos/{id}", h.excluirContrato)
		r.Put("/contratos/contratos/{id}/prazo", h.definirPrazo)
		r.Get("/contratos/licencas", h.listarLicencas)
		r.Post("/contratos/licencas", h.criarLicenca)
		r.Delete("/contratos/licencas/{id}", h.excluirLicenca)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (h *Handlers) listarFornecedores(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarFornecedores(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) criarFornecedor(w http.ResponseWriter, r *http.Request) {
	var req FornecedorRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarFornecedor(r.Context(), identity, req.Nome, req.Email, req.Telefone, req.UnidadeID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) excluirFornecedor(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirFornecedor(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) listarContratos(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarContratos(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) criarContrato(w http.ResponseWriter, r *http.Request) {
	var req ContratoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarContrato(r.Context(), identity, req.Numero, req.Titulo, req.FornecedorID, req.Inicio, req.Fim, req.ValorCentavos, req.UnidadeID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) excluirContrato(w http.ResponseWriter, r *http.Request) {
	h.excluir(w, r, h.svc.ExcluirContrato)
}

// PrazoRequest é o prazo de atendimento do fornecedor (nulo tira).
type PrazoRequest struct {
	PrazoHoras *int `json:"prazo_horas" validate:"omitempty,min=1,max=8760"`
}

func (h *Handlers) definirPrazo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req PrazoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.DefinirPrazo(r.Context(), identity, id, req.PrazoHoras); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) listarLicencas(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarLicencas(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) criarLicenca(w http.ResponseWriter, r *http.Request) {
	var req LicencaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	item, err := h.svc.CriarLicenca(r.Context(), identity, req.Nome, req.FornecedorID, req.Fim, req.Quantidade, req.Programa, req.UnidadeID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) excluirLicenca(w http.ResponseWriter, r *http.Request) {
	h.excluir(w, r, h.svc.ExcluirLicenca)
}

func (h *Handlers) excluir(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, identity auth.Identity, id uuid.UUID) error) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := fn(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
