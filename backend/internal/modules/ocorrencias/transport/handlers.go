// Package transport expõe as ocorrências via HTTP.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// PermissaoGerir declara, atualiza e revisa ocorrências.
const PermissaoGerir = "ocorrencias:manage"

// DeclaracaoRequest abre a ocorrência.
type DeclaracaoRequest struct {
	Titulo    string     `json:"titulo" validate:"required,max=160"`
	Servicos  []string   `json:"servicos" validate:"required,min=1,max=10"`
	Impacto   string     `json:"impacto" validate:"required,oneof=degradado parcial fora"`
	Texto     string     `json:"texto" validate:"required,max=4000"`
	ChamadoID *uuid.UUID `json:"chamado_id"`
}

// AtualizacaoRequest é a próxima notícia.
type AtualizacaoRequest struct {
	Estado  string `json:"estado" validate:"required,oneof=investigando identificado monitorando resolvido"`
	Impacto string `json:"impacto" validate:"required,oneof=degradado parcial fora"`
	Texto   string `json:"texto" validate:"required,max=4000"`
}

// CoordenadorRequest troca quem coordena.
type CoordenadorRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
}

// RevisaoRequest é a revisão pós-incidente.
type RevisaoRequest struct {
	Texto string `json:"texto" validate:"required,max=4000"`
}

// Handlers agrupa os handlers HTTP.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterPublicRoutes monta a página de status (sem login).
func (h *Handlers) RegisterPublicRoutes(r chi.Router) {
	r.Get("/status", h.Situacao)
}

// RegisterRoutes monta o painel da coordenação.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.Permission(PermissaoGerir)))
		r.Get("/ocorrencias", h.Listar)
		r.Post("/ocorrencias", h.Declarar)
		r.Get("/ocorrencias/{id}", h.Obter)
		r.Post("/ocorrencias/{id}/atualizacoes", h.Atualizar)
		r.Put("/ocorrencias/{id}/coordenador", h.TrocarCoordenador)
		r.Put("/ocorrencias/{id}/revisao", h.Revisar)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// Situacao é a página pública. Cache curto: num incidente esta é a página
// mais acessada, e a borda absorve a carga sem segurar a notícia.
func (h *Handlers) Situacao(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Situacao(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	httputil.WriteOK(w, s)
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	lista, err := h.svc.Listar(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Declarar(w http.ResponseWriter, r *http.Request) {
	var req DeclaracaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	o, err := h.svc.Declarar(r.Context(), identity, application.DeclaracaoEntrada{
		Titulo: req.Titulo, Servicos: req.Servicos, Impacto: req.Impacto, Texto: req.Texto, ChamadoID: req.ChamadoID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, o)
}

func (h *Handlers) Obter(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	o, err := h.svc.Obter(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req AtualizacaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	o, err := h.svc.Atualizar(r.Context(), identity, id, application.AtualizacaoEntrada{Estado: req.Estado, Impacto: req.Impacto, Texto: req.Texto})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}

func (h *Handlers) TrocarCoordenador(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req CoordenadorRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	o, err := h.svc.TrocarCoordenador(r.Context(), id, req.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}

func (h *Handlers) Revisar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req RevisaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	o, err := h.svc.Revisar(r.Context(), id, req.Texto)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}
