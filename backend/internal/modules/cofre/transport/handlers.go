package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// SegredoRequest cria um segredo. A resposta não devolve a senha.
type SegredoRequest struct {
	Nome       string      `json:"nome" validate:"required,max=160"`
	Usuario    string      `json:"usuario" validate:"max=160"`
	Senha      string      `json:"senha" validate:"required,max=2000"`
	Notas      string      `json:"notas" validate:"max=4000"`
	Tipo       string      `json:"tipo" validate:"required,oneof=maquina aplicacao painel outro"`
	Endereco   string      `json:"endereco" validate:"max=500"`
	Administra []uuid.UUID `json:"administra"`
	UnidadeID  *uuid.UUID  `json:"unidade_id"`
	Rotaciona  bool        `json:"rotaciona"`
	PastaID    *uuid.UUID  `json:"pasta_id"`
}

// AtualizarRequest troca o que foi enviado. Senha e notas ausentes ficam como estão.
type AtualizarRequest struct {
	Nome         string       `json:"nome" validate:"required,max=160"`
	Usuario      string       `json:"usuario" validate:"max=160"`
	Senha        *string      `json:"senha" validate:"omitempty,max=2000"`
	Notas        *string      `json:"notas" validate:"omitempty,max=4000"`
	Tipo         *string      `json:"tipo" validate:"omitempty,oneof=maquina aplicacao painel outro"`
	Endereco     *string      `json:"endereco" validate:"omitempty,max=500"`
	Administra   *[]uuid.UUID `json:"administra"`
	UnidadeID    *uuid.UUID   `json:"unidade_id"`
	MoverUnidade bool         `json:"mover_unidade"`
	Rotaciona    *bool        `json:"rotaciona,omitempty"`
	MoverPasta   bool         `json:"mover_pasta"`
	PastaID      *uuid.UUID   `json:"pasta_id"`
}

// Handlers da API do cofre.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta o cofre. Cada ação confere a própria permissão.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/cofre/segredos", h.listar)
	r.Post("/cofre/segredos", h.criar)
	r.Put("/cofre/segredos/{id}", h.atualizar)
	r.Delete("/cofre/segredos/{id}", h.excluir)
	r.Post("/cofre/segredos/{id}/revelar", h.revelar)
	r.Post("/cofre/segredos/{id}/retirar", h.retirar)
	r.Post("/cofre/segredos/{id}/devolver", h.devolver)
	r.Post("/cofre/segredos/{id}/rotacao/aplicada", h.aplicada)
	r.Get("/cofre/pastas", h.listarPastas)
	r.Post("/cofre/pastas", h.criarPasta)
	r.Put("/cofre/pastas/{id}/membros", h.membrosPasta)
	r.Delete("/cofre/pastas/{id}", h.excluirPasta)
	r.Get("/cofre/pessoas", h.pessoas)
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (h *Handlers) exigir(w http.ResponseWriter, r *http.Request, perms ...auth.Permission) (auth.Identity, bool) {
	identity, _ := auth.IdentityFromContext(r.Context())
	for _, perm := range perms {
		if auth.HasPermission(identity, perm) {
			return identity, true
		}
	}
	httputil.WriteError(w, r, h.logger, apperrors.Forbidden("seu perfil não abre o cofre"))
	return identity, false
}

func (h *Handlers) listar(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreRead, auth.PermCofreReveal, auth.PermCofreManage, auth.PermCofreGrant)
	if !ok {
		return
	}
	lista, err := h.svc.Listar(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) criar(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	var req SegredoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.Criar(r.Context(), identity, application.Entrada{
		Nome: req.Nome, Usuario: req.Usuario, Senha: req.Senha, Notas: req.Notas,
		Tipo: req.Tipo, Endereco: req.Endereco, Unidade: req.UnidadeID, Administra: req.Administra,
		Rotaciona: req.Rotaciona, Pasta: req.PastaID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) atualizar(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage, auth.PermCofreGrant)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req AtualizarRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.Atualizar(r.Context(), identity, id, application.Mudanca{
		Nome: req.Nome, Usuario: req.Usuario, Senha: req.Senha, Notas: req.Notas,
		Tipo: req.Tipo, Endereco: req.Endereco, Administra: req.Administra,
		Unidade: req.UnidadeID, Mover: req.MoverUnidade, Rotaciona: req.Rotaciona,
		MoverPasta: req.MoverPasta, Pasta: req.PastaID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) excluir(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.Excluir(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) revelar(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreReveal, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.Revelar(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

// RetirarRequest reserva o segredo e pede a senha de acesso de novo.
type RetirarRequest struct {
	SenhaDeAcesso string `json:"senha_de_acesso" validate:"required,max=1024"`
	Minutos       int    `json:"minutos" validate:"required,min=15,max=480"`
}

func (h *Handlers) retirar(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreReveal, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req RetirarRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.Retirar(r.Context(), identity, id, req.SenhaDeAcesso, req.Minutos)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) devolver(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreReveal, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.Devolver(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) aplicada(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.MarcarAplicada(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

type pastaRequest struct {
	Nome      string          `json:"nome" validate:"required,max=160"`
	UnidadeID *uuid.UUID      `json:"unidade_id"`
	Membros   []domain.Membro `json:"membros" validate:"max=50,dive"`
}

type membrosRequest struct {
	Membros []domain.Membro `json:"membros" validate:"max=50,dive"`
}

func (h *Handlers) listarPastas(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreRead, auth.PermCofreReveal, auth.PermCofreManage, auth.PermCofreGrant)
	if !ok {
		return
	}
	lista, err := h.svc.ListarPastas(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) criarPasta(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	var req pastaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.CriarPasta(r.Context(), identity, req.Nome, req.UnidadeID, req.Membros)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, item)
}

func (h *Handlers) membrosPasta(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req membrosRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.svc.DefinirMembros(r.Context(), identity, id, req.Membros)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

func (h *Handlers) excluirPasta(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.ExcluirPasta(r.Context(), identity, id); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) pessoas(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.exigir(w, r, auth.PermCofreManage)
	if !ok {
		return
	}
	var unidade *uuid.UUID
	if bruto := r.URL.Query().Get("unidade_id"); bruto != "" {
		id, err := uuid.Parse(bruto)
		if err != nil {
			h.fail(w, r, apperrors.Validation("unidade inválida"))
			return
		}
		unidade = &id
	}
	lista, err := h.svc.Pessoas(r.Context(), identity, unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}
