// Package transport expõe o atendimento: cadastro, abertura e fila.
package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// OfertaRequest é o corpo de criação e edição de uma oferta.
type OfertaRequest struct {
	Nome      string          `json:"nome" validate:"required,max=160"`
	Descricao string          `json:"descricao" validate:"max=4000"`
	Tipo      string          `json:"tipo" validate:"omitempty,oneof=incidente requisicao"`
	ProdutoID *uuid.UUID      `json:"produto_id"`
	UnidadeID *uuid.UUID      `json:"unidade_id"`
	Ativa     *bool           `json:"ativa"`
	Campos    *[]domain.Campo `json:"campos"`
}

// EquipeRequest é o corpo de criação e edição de uma equipe.
type EquipeRequest struct {
	Nome      string      `json:"nome" validate:"required,max=160"`
	UnidadeID *uuid.UUID  `json:"unidade_id"`
	Membros   []uuid.UUID `json:"membros" validate:"max=100"`
}

// Handlers da API de atendimento.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta o cadastro (atendimento:manage), a abertura
// (atendimento:abrir) e a fila (atendimento:atender). O serviço ainda
// restringe pela unidade. Ver, mudar o estado e comentar servem às duas
// pontas: o serviço decide pelo papel de quem pede (quem abriu ou quem
// atende a unidade) e devolve não encontrado a quem não é nenhum dos dois.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtendimentoAbrir))
		r.Get("/atendimento/catalogo", h.Catalogo)
		r.Post("/atendimento/solicitacoes", h.Abrir)
		r.Post("/atendimento/deflexao", h.RegistrarDeflexao)
		r.Get("/atendimento/minhas", h.ListarMinhas)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtendimentoAtender))
		r.Get("/atendimento/fila", h.ListarFila)
		r.Get("/atendimento/governanca", h.Governanca)
		r.Get("/atendimento/governanca/serie", h.Serie)
		r.Get("/atendimento/fila/topicos", h.TopicosDaFila)
		r.Put("/atendimento/solicitacoes/{id}/responsavel", h.Atribuir)
		r.Get("/atendimento/solicitacoes/{id}/responsaveis", h.Responsaveis)
		r.Put("/atendimento/solicitacoes/{id}/classificacao", h.Classificar)
		// IA assistiva (ADR 065): sugere a quem atende, que decide.
		r.Get("/atendimento/ia", h.EstadoDaIA)
		r.Get("/atendimento/ia/indicadores", h.IndicadoresIA)
		r.Post("/atendimento/solicitacoes/{id}/ia/classificacao", h.SugerirClassificacao)
		r.Post("/atendimento/solicitacoes/{id}/ia/resumo", h.Resumir)
		r.Post("/atendimento/ia/sugestoes/{id}/decisao", h.DecidirSugestao)
		r.Put("/atendimento/solicitacoes/{id}/segredo", h.VincularSegredo)
		r.Post("/atendimento/solicitacoes/{id}/visitas", h.IniciarVisita)
		r.Post("/atendimento/visitas/{id}/checkout", h.ConcluirVisita)
		r.Post("/atendimento/visitas/{id}/cancelar", h.CancelarVisita)
		r.Post("/atendimento/visitas/{id}/evidencias/upload", h.UploadEvidencia)
		r.Post("/atendimento/visitas/{id}/evidencias", h.RegistrarEvidencia)
		r.Get("/atendimento/visitas/em-campo", h.VisitasEmCampo)
		r.Post("/atendimento/solicitacoes/{id}/apontamentos", h.RegistrarApontamento)
		r.Delete("/atendimento/solicitacoes/{id}/apontamentos/{apontamentoId}", h.ExcluirApontamento)
	})
	r.Get("/atendimento/solicitacoes/{id}", h.ObterSolicitacao)
	r.Get("/atendimento/ofertas/{id}/respostas", h.ObterResposta)
	r.Get("/atendimento/solicitacoes/{id}/apontamentos", h.ListarApontamentos)
	r.Post("/atendimento/solicitacoes/{id}/estado", h.Transicionar)
	r.Post("/atendimento/solicitacoes/{id}/comentarios", h.Comentar)
	r.Post("/atendimento/solicitacoes/{id}/avaliacao", h.Avaliar)
	r.Get("/atendimento/visitas/{id}/evidencias/{evidencia}", h.URLEvidencia)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermAtendimentoManage))
		r.Get("/atendimento/ofertas", h.ListarOfertas)
		r.Post("/atendimento/ofertas", h.CriarOferta)
		r.Get("/atendimento/ofertas/{id}", h.ObterOferta)
		r.Put("/atendimento/ofertas/{id}", h.AtualizarOferta)
		r.Get("/atendimento/equipes", h.ListarEquipes)
		r.Post("/atendimento/equipes", h.CriarEquipe)
		r.Get("/atendimento/equipes/{id}", h.ObterEquipe)
		r.Put("/atendimento/equipes/{id}", h.AtualizarEquipe)
		r.Get("/atendimento/candidatos", h.Candidatos)
		r.Get("/atendimento/politica", h.ObterPolitica)
		r.Put("/atendimento/politica", h.SalvarPolitica)
		r.Get("/atendimento/feriados", h.ListarFeriados)
		r.Post("/atendimento/feriados", h.CriarFeriado)
		r.Delete("/atendimento/feriados/{id}", h.ExcluirFeriado)
		r.Get("/atendimento/regras", h.ListarRegras)
		r.Post("/atendimento/regras", h.CriarRegra)
		r.Delete("/atendimento/regras/{id}", h.ExcluirRegra)
		r.Put("/atendimento/ofertas/{id}/respostas", h.SalvarResposta)
		r.Get("/atendimento/recorrencias", h.ListarRecorrencias)
		r.Post("/atendimento/recorrencias", h.CriarRecorrencia)
		r.Delete("/atendimento/recorrencias/{id}", h.ExcluirRecorrencia)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (req OfertaRequest) entrada() application.OfertaEntrada {
	return application.OfertaEntrada{Nome: req.Nome, Descricao: req.Descricao, Tipo: req.Tipo, ProdutoID: req.ProdutoID, UnidadeID: req.UnidadeID, Ativa: req.Ativa, Campos: req.Campos}
}

func (req EquipeRequest) entrada() application.EquipeEntrada {
	return application.EquipeEntrada{Nome: req.Nome, UnidadeID: req.UnidadeID, Membros: req.Membros}
}

func (h *Handlers) ListarOfertas(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarOfertas(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarOferta(w http.ResponseWriter, r *http.Request) {
	var req OfertaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	o, err := h.svc.CriarOferta(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, o)
}

func (h *Handlers) ObterOferta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	o, err := h.svc.ObterOferta(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}

func (h *Handlers) AtualizarOferta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req OfertaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	o, err := h.svc.AtualizarOferta(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, o)
}

func (h *Handlers) ListarEquipes(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarEquipes(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) CriarEquipe(w http.ResponseWriter, r *http.Request) {
	var req EquipeRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	e, err := h.svc.CriarEquipe(r.Context(), identity, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, e)
}

func (h *Handlers) ObterEquipe(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	e, err := h.svc.ObterEquipe(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, e)
}

func (h *Handlers) AtualizarEquipe(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EquipeRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	e, err := h.svc.AtualizarEquipe(r.Context(), identity, id, req.entrada())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, e)
}

func (h *Handlers) Candidatos(w http.ResponseWriter, r *http.Request) {
	unidade, err := httputil.OptionalUUIDQuery(r, "unidade_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if unidade == nil {
		h.fail(w, r, apperrors.BadRequest("informe a unidade"))
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Candidatos(r.Context(), identity, *unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}
