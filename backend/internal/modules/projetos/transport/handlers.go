package transport

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

type ProjetoRequest struct {
	Titulo              string     `json:"titulo" validate:"required,max=160"`
	Descricao           string     `json:"descricao" validate:"max=4000"`
	Objetivo            string     `json:"objetivo" validate:"max=4000"`
	Prioridade          string     `json:"prioridade" validate:"omitempty,oneof=baixa media alta critica"`
	DataInicioPlanejada *time.Time `json:"data_inicio_planejada"`
	DataFimPlanejada    *time.Time `json:"data_fim_planejada"`
	DataInicioReal      *time.Time `json:"data_inicio_real"`
	DataFimReal         *time.Time `json:"data_fim_real"`
	Orcamento           float64    `json:"orcamento"`
	UnidadeID           *uuid.UUID `json:"unidade_id"`
	ProdutoID           *uuid.UUID `json:"produto_id"`
	GerenteID           *uuid.UUID `json:"gerente_id"`
	GerenteNome         string     `json:"gerente_nome" validate:"max=200"`
}

func (req ProjetoRequest) entrada() domain.EntradaProjeto {
	return domain.EntradaProjeto{
		Titulo:              req.Titulo,
		Descricao:           req.Descricao,
		Objetivo:            req.Objetivo,
		Prioridade:          req.Prioridade,
		DataInicioPlanejada: req.DataInicioPlanejada,
		DataFimPlanejada:    req.DataFimPlanejada,
		DataInicioReal:      req.DataInicioReal,
		DataFimReal:         req.DataFimReal,
		Orcamento:           req.Orcamento,
		UnidadeID:           req.UnidadeID,
		ProdutoID:           req.ProdutoID,
		GerenteID:           req.GerenteID,
		GerenteNome:         req.GerenteNome,
	}
}

type EstadoRequest struct {
	Estado string `json:"estado" validate:"required,min=1,max=40"`
}

type TarefaRequest struct {
	Titulo          string     `json:"titulo" validate:"required,max=200"`
	Descricao       string     `json:"descricao" validate:"max=2000"`
	ResponsavelNome string     `json:"responsavel_nome" validate:"max=200"`
	Estado          string     `json:"estado" validate:"omitempty,min=1,max=40"`
	Ordem           int        `json:"ordem"`
	DataLimite      *time.Time `json:"data_limite"`
}

type VinculoRequest struct {
	Tipo   string    `json:"tipo" validate:"required,oneof=mudanca problema ativo solicitacao"`
	AlvoID uuid.UUID `json:"alvo_id" validate:"required"`
	Rotulo string    `json:"rotulo" validate:"max=200"`
}

type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermProjetosRead))
		r.Get("/projetos", h.Listar)
		r.Get("/projetos/{id}", h.Obter)
		r.Get("/projetos/{id}/tarefas", h.ListarTarefas)
		r.Get("/projetos/{id}/vinculos", h.ListarVinculos)
		r.Get("/projetos/{id}/apontamentos", h.ListarApontamentos)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermProjetosManage))
		r.Post("/projetos", h.Criar)
		r.Put("/projetos/{id}", h.Atualizar)
		r.Post("/projetos/{id}/estado", h.Transicionar)

		r.Post("/projetos/{id}/tarefas", h.CriarTarefa)
		r.Put("/projetos/{id}/tarefas/{tarefaId}", h.AtualizarTarefa)
		r.Delete("/projetos/{id}/tarefas/{tarefaId}", h.ExcluirTarefa)

		r.Post("/projetos/{id}/vinculos", h.CriarVinculo)
		r.Delete("/projetos/{id}/vinculos/{vinculoId}", h.ExcluirVinculo)

		r.Post("/projetos/{id}/apontamentos", h.CriarApontamento)
		r.Delete("/projetos/{id}/apontamentos/{apontamentoId}", h.ExcluirApontamento)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	produtoID, err := httputil.OptionalUUIDQuery(r, "produto_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	alvoID, err := httputil.OptionalUUIDQuery(r, "alvo_id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	f := domain.FiltroProjetos{
		Estado:    httputil.Query(r, "estado", 20),
		Busca:     httputil.Query(r, "q", 100),
		ProdutoID: produtoID,
		AlvoTipo:  httputil.Query(r, "alvo_tipo", 20),
		AlvoID:    alvoID,
	}

	lista, err := h.svc.ListarPorFiltro(r.Context(), identity, f)
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

func (h *Handlers) Criar(w http.ResponseWriter, r *http.Request) {
	var req ProjetoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
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
	var req ProjetoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
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
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
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

func (h *Handlers) ListarTarefas(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	tarefas, err := h.svc.ListarTarefas(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, tarefas)
}

func (h *Handlers) CriarTarefa(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req TarefaRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	t, err := h.svc.CriarTarefa(r.Context(), identity, id, req.Titulo, req.Descricao, req.ResponsavelNome, req.Estado, req.Ordem, req.DataLimite)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, t)
}

func (h *Handlers) AtualizarTarefa(w http.ResponseWriter, r *http.Request) {
	tarefaID, err := httputil.UUIDParam(r, "tarefaId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req TarefaRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	t, err := h.svc.AtualizarTarefa(r.Context(), identity, tarefaID, req.Titulo, req.Descricao, req.ResponsavelNome, req.Estado, req.Ordem, req.DataLimite)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, t)
}

func (h *Handlers) ExcluirTarefa(w http.ResponseWriter, r *http.Request) {
	tarefaID, err := httputil.UUIDParam(r, "tarefaId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirTarefa(r.Context(), identity, tarefaID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) ListarVinculos(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	vinculos, err := h.svc.ListarVinculos(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, vinculos)
}

func (h *Handlers) CriarVinculo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req VinculoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	v, err := h.svc.Vincular(r.Context(), identity, id, req.Tipo, req.AlvoID, req.Rotulo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, v)
}

func (h *Handlers) ExcluirVinculo(w http.ResponseWriter, r *http.Request) {
	vinculoID, err := httputil.UUIDParam(r, "vinculoId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.Desvincular(r.Context(), identity, vinculoID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

type ApontamentoRequest struct {
	TarefaID        *uuid.UUID `json:"tarefa_id"`
	DataApontamento string     `json:"data_apontamento"`
	Horas           float64    `json:"horas" validate:"required,gt=0,lte=24"`
	Descricao       string     `json:"descricao" validate:"required,max=2000"`
	CustoHora       float64    `json:"custo_hora"`
}

type ListarApontamentosResponse struct {
	Items  []domain.Apontamento      `json:"items"`
	Resumo domain.ResumoHorasProjeto `json:"resumo"`
}

func (h *Handlers) ListarApontamentos(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	itens, resumo, err := h.svc.ListarApontamentos(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ListarApontamentosResponse{
		Items:  itens,
		Resumo: resumo,
	})
}

func (h *Handlers) CriarApontamento(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ApontamentoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	a, err := h.svc.RegistrarApontamento(r.Context(), identity, id, req.TarefaID, req.DataApontamento, req.Horas, req.Descricao, req.CustoHora)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, a)
}

func (h *Handlers) ExcluirApontamento(w http.ResponseWriter, r *http.Request) {
	apontamentoID, err := httputil.UUIDParam(r, "apontamentoId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ExcluirApontamento(r.Context(), identity, apontamentoID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
