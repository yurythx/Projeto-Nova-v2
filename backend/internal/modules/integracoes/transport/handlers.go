// Package transport é a API HTTP das integrações.
package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// ConexaoRequest é o cadastro da conexão. Tipo e modo valem só na criação.
type ConexaoRequest struct {
	Tipo   string `json:"tipo" validate:"max=20"`
	Nome   string `json:"nome" validate:"required,max=160"`
	Modo   string `json:"modo" validate:"max=20"`
	URL    string `json:"url" validate:"max=500"`
	Perfil string `json:"perfil" validate:"max=80"`
	Fuso   string `json:"fuso" validate:"max=60"`
}

// CredenciaisRequest são as credenciais da origem no modo direto.
type CredenciaisRequest struct {
	AppToken  string `json:"app_token" validate:"max=200"`
	UserToken string `json:"user_token" validate:"max=200"`
	Login     string `json:"login" validate:"max=200"`
	Senha     string `json:"senha" validate:"max=200"`
}

// TabelaRequest é a tabela de-para inteira.
type TabelaRequest struct {
	Linhas []domain.Linha `json:"linhas" validate:"max=2000"`
}

// SugestaoRequest leva cada secretaria a uma unidade daqui.
type SugestaoRequest struct {
	Unidades map[string]uuid.UUID `json:"unidades"`
}

// TarefaRequest pede uma tarefa.
type TarefaRequest struct {
	Tipo string `json:"tipo" validate:"required"`
}

// ItensRequest é um lote do conector.
type ItensRequest struct {
	Itens []domain.Item `json:"itens"`
}

// ProgressoRequest é o andamento que o conector informa.
type ProgressoRequest struct {
	Feitos int `json:"feitos" validate:"min=0"`
	Total  int `json:"total" validate:"min=0"`
}

// FimRequest fecha a extração do conector.
type FimRequest struct {
	Relatorio json.RawMessage `json:"relatorio"`
	Erro      string          `json:"erro" validate:"max=2000"`
}

// Handlers da API de integrações.
type Handlers struct {
	svc    *application.Service
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// RegisterRoutes monta a tela (integracoes:manage) e o conector
// (integracoes:conector, só do token de máquina).
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermIntegracoesManage))
		r.Get("/integracoes/conexoes", h.listar)
		r.Post("/integracoes/conexoes", h.criar)
		r.Get("/integracoes/conexoes/{id}", h.obter)
		r.Put("/integracoes/conexoes/{id}", h.atualizar)
		r.Delete("/integracoes/conexoes/{id}", h.excluir)
		r.Put("/integracoes/conexoes/{id}/credenciais", h.credenciais)
		r.Post("/integracoes/conexoes/{id}/token", h.emitirToken)
		r.Delete("/integracoes/conexoes/{id}/token", h.revogarToken)
		r.Get("/integracoes/conexoes/{id}/inventario", h.inventario)
		r.Get("/integracoes/conexoes/{id}/tabela", h.tabela)
		r.Put("/integracoes/conexoes/{id}/tabela", h.salvarTabela)
		r.Post("/integracoes/conexoes/{id}/tabela/sugestao", h.sugerir)
		r.Get("/integracoes/conexoes/{id}/tarefas", h.tarefas)
		r.Post("/integracoes/conexoes/{id}/tarefas", h.pedir)
		r.Post("/integracoes/tarefas/{id}/cancelar", h.cancelar)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(h.logger, auth.PermIntegracoesConector))
		r.Get("/integracoes/conector/versoes", h.versoes)
		r.Post("/integracoes/conector/itens", h.itens)
		r.Post("/integracoes/conector/tarefa", h.pegar)
		r.Post("/integracoes/conector/tarefas/{id}/progresso", h.progresso)
		r.Post("/integracoes/conector/tarefas/{id}/fim", h.fim)
	})
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

// comID lê o {id} e a identidade, ou responde o erro.
func (h *Handlers) comID(w http.ResponseWriter, r *http.Request) (uuid.UUID, auth.Identity, bool) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return uuid.Nil, auth.Identity{}, false
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	return id, identity, true
}

func responder[T any](h *Handlers, w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, v)
}

func (h *Handlers) listar(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.ListarConexoes(r.Context(), identity)
	responder(h, w, r, lista, err)
}

func (h *Handlers) criar(w http.ResponseWriter, r *http.Request) {
	var req ConexaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.CriarConexao(r.Context(), identity, application.ConexaoEntrada(req))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, c)
}

func (h *Handlers) obter(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		c, err := h.svc.ObterConexao(r.Context(), identity, id)
		responder(h, w, r, c, err)
	}
}

func (h *Handlers) atualizar(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req ConexaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	c, err := h.svc.AtualizarConexao(r.Context(), identity, id, application.ConexaoEntrada(req))
	responder(h, w, r, c, err)
}

func (h *Handlers) excluir(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		if err := h.svc.ExcluirConexao(r.Context(), identity, id); err != nil {
			h.fail(w, r, err)
			return
		}
		httputil.WriteNoContent(w)
	}
}

func (h *Handlers) credenciais(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req CredenciaisRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.DefinirCredenciais(r.Context(), identity, id, domain.Credenciais(req)); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

func (h *Handlers) emitirToken(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		token, err := h.svc.EmitirToken(r.Context(), identity, id)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httputil.WriteCreated(w, map[string]string{"token": token})
	}
}

func (h *Handlers) revogarToken(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		if err := h.svc.RevogarToken(r.Context(), identity, id); err != nil {
			h.fail(w, r, err)
			return
		}
		httputil.WriteNoContent(w)
	}
}

func (h *Handlers) inventario(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		inv, err := h.svc.Inventario(r.Context(), identity, id)
		responder(h, w, r, inv, err)
	}
}

func (h *Handlers) tabela(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		linhas, err := h.svc.Tabela(r.Context(), identity, id)
		responder(h, w, r, linhas, err)
	}
}

func (h *Handlers) salvarTabela(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req TabelaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	linhas, err := h.svc.SalvarTabela(r.Context(), identity, id, req.Linhas)
	responder(h, w, r, linhas, err)
}

func (h *Handlers) sugerir(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req SugestaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	linhas, err := h.svc.SugerirTabela(r.Context(), identity, id, req.Unidades)
	responder(h, w, r, linhas, err)
}

func (h *Handlers) tarefas(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		lista, err := h.svc.ListarTarefas(r.Context(), identity, id)
		responder(h, w, r, lista, err)
	}
}

func (h *Handlers) pedir(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req TarefaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	t, err := h.svc.PedirTarefa(r.Context(), identity, id, req.Tipo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, t)
}

func (h *Handlers) cancelar(w http.ResponseWriter, r *http.Request) {
	if id, identity, ok := h.comID(w, r); ok {
		if err := h.svc.CancelarTarefa(r.Context(), identity, id); err != nil {
			h.fail(w, r, err)
			return
		}
		httputil.WriteNoContent(w)
	}
}

// versoes diz ao conector o que já foi extraído.
func (h *Handlers) versoes(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	v, pessoas, err := h.svc.ConectorVersoes(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	ids := make([]int, 0, len(pessoas))
	for id := range pessoas {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	httputil.WriteOK(w, map[string]any{"versoes": v, "pessoas": ids})
}

func (h *Handlers) itens(w http.ResponseWriter, r *http.Request) {
	var req ItensRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	if err := h.svc.ConectorGravar(r.Context(), identity, req.Itens); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}

// pegar entrega a extração pendente; sem nenhuma, 204.
func (h *Handlers) pegar(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	t, ok, err := h.svc.ConectorPegar(r.Context(), identity)
	switch {
	case err != nil:
		h.fail(w, r, err)
	case !ok:
		httputil.WriteNoContent(w)
	default:
		httputil.WriteOK(w, t)
	}
}

func (h *Handlers) progresso(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req ProgressoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	estado, err := h.svc.ConectorProgresso(r.Context(), identity, id, req.Feitos, req.Total)
	responder(h, w, r, map[string]string{"estado": estado}, err)
}

func (h *Handlers) fim(w http.ResponseWriter, r *http.Request) {
	id, identity, ok := h.comID(w, r)
	if !ok {
		return
	}
	var req FimRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.ConectorTerminar(r.Context(), identity, id, req.Relatorio, req.Erro); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteNoContent(w)
}
