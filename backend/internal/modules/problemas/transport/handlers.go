// Package transport expõe os problemas e a base de erros conhecidos.
package transport

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// ProblemaRequest é o corpo de criação e edição.
type ProblemaRequest struct {
	Titulo    string     `json:"titulo" validate:"required,max=160"`
	Descricao string     `json:"descricao" validate:"max=4000"`
	CausaRaiz string     `json:"causa_raiz" validate:"max=4000"`
	Contorno  string     `json:"contorno" validate:"max=4000"`
	ProdutoID *uuid.UUID `json:"produto_id"`
	UnidadeID *uuid.UUID `json:"unidade_id"`
}

func (req ProblemaRequest) entrada() domain.Entrada {
	return domain.Entrada{Titulo: req.Titulo, Descricao: req.Descricao, CausaRaiz: req.CausaRaiz, Contorno: req.Contorno,
		ProdutoID: req.ProdutoID, UnidadeID: req.UnidadeID}
}

// EstadoRequest é a passagem pedida.
type EstadoRequest struct {
	Estado string `json:"estado" validate:"required,oneof=registrado em_analise erro_conhecido resolvido encerrado"`
}

// Handlers da API de problemas.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes: a leitura serve a quem gere (problemas:manage) e a quem
// consulta os erros conhecidos (problemas:read) — o serviço decide pela
// cobertura de cada um. Alterar exige problemas:manage.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/problemas", h.Listar)
	r.Get("/problemas/{id}", h.Obter)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermProblemasManage))
		r.Get("/problemas/sugestoes", h.Sugestoes)
		r.Post("/problemas", h.Criar)
		r.Put("/problemas/{id}", h.Atualizar)
		r.Post("/problemas/{id}/estado", h.Transicionar)
		r.Put("/problemas/{id}/incidentes/{alvo}", h.VincularIncidente)
		r.Delete("/problemas/{id}/incidentes/{alvo}", h.DesvincularIncidente)
		r.Put("/problemas/{id}/ativos/{alvo}", h.VincularAtivo)
		r.Delete("/problemas/{id}/ativos/{alvo}", h.DesvincularAtivo)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// Listar aceita ?estado=, ?solicitacao_id= (problemas de um incidente) e
// ?q= (título ou contorno).
func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	solicitacao, err := httputil.OptionalUUIDQuery(r, "solicitacao_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Listar(r.Context(), identity, application.FiltroEntrada{
		Estado: httputil.Query(r, "estado", 20), SolicitacaoID: solicitacao, Busca: httputil.Query(r, "q", 100),
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
	p, err := h.svc.Obter(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) Sugestoes(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Sugestoes(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req ProblemaRequest
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

func (h *Handlers) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ProblemaRequest
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
	p, err := h.svc.Transicionar(r.Context(), identity, id, req.Estado)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

// Vínculos: PUT liga e DELETE desfaz; repetir não é erro, o estado final
// é o pedido.
func (h *Handlers) VincularIncidente(w http.ResponseWriter, r *http.Request) {
	h.vincular(w, r, h.svc.VincularIncidente, true)
}

func (h *Handlers) DesvincularIncidente(w http.ResponseWriter, r *http.Request) {
	h.vincular(w, r, h.svc.VincularIncidente, false)
}

func (h *Handlers) VincularAtivo(w http.ResponseWriter, r *http.Request) {
	h.vincular(w, r, h.svc.VincularAtivo, true)
}

func (h *Handlers) DesvincularAtivo(w http.ResponseWriter, r *http.Request) {
	h.vincular(w, r, h.svc.VincularAtivo, false)
}

type vinculo func(ctx context.Context, identity auth.Identity, id, alvo uuid.UUID, ligar bool) (domain.Problema, error)

func (h *Handlers) vincular(w http.ResponseWriter, r *http.Request, fn vinculo, ligar bool) {
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
	p, err := fn(r.Context(), identity, id, alvo, ligar)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}
