// Package transport expõe a carta e o pedido.
package transport

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// Casos é o que o HTTP precisa do caso de uso.
type Casos interface {
	SalvarFormulario(ctx context.Context, f domain.Formulario) (domain.Formulario, error)
	ListarFormularios(ctx context.Context) ([]domain.Formulario, error)
	SalvarServico(ctx context.Context, s domain.Servico) (domain.Servico, error)
	ListarServicosGestao(ctx context.Context) ([]domain.Servico, error)
	Carta(ctx context.Context, unidade *uuid.UUID) ([]domain.Servico, error)
	ServicoPublico(ctx context.Context, id uuid.UUID) (domain.Servico, domain.Formulario, error)
	PedirArquivo(ctx context.Context, servico uuid.UUID, nome, contentType string) (modkit.UploadTicket, error)
	PedirArquivoComplemento(ctx context.Context, numero, cpf, nome, contentType string) (modkit.UploadTicket, error)
	LinkArquivo(ctx context.Context, pedido uuid.UUID, campo string) (string, error)
	CadastrarResponsavel(ctx context.Context, nome, cpf string, unidade uuid.UUID) (domain.ResponsavelTecnico, error)
	ListarResponsaveis(ctx context.Context) ([]domain.ResponsavelTecnico, error)
	RegistrarFalta(ctx context.Context, pedido uuid.UUID) error
	Abrir(ctx context.Context, in application.Abertura, balcao bool) (domain.Pedido, error)
	Consultar(ctx context.Context, numero string) (domain.Consulta, error)
	ResponderPublico(ctx context.Context, numero, cpf, mensagem string, arquivos []string) (domain.Consulta, error)
	ListarPedidos(ctx context.Context) ([]domain.Pedido, error)
	BuscarPedido(ctx context.Context, id uuid.UUID) (application.Ficha, error)
	ComplementarCom(ctx context.Context, id uuid.UUID, in application.Complementacao) (domain.Pedido, error)
	Responder(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error)
	AnotarCom(ctx context.Context, id uuid.UUID, mensagem string, anexos []string) (domain.Pedido, error)
	Editar(ctx context.Context, id uuid.UUID, in application.Edicao) (domain.Pedido, error)
	AvancarCom(ctx context.Context, id uuid.UUID, anexos []string) (domain.Pedido, error)
	ConcluirCom(ctx context.Context, id uuid.UUID, in application.Conclusao) (domain.Pedido, error)
	Cancelar(ctx context.Context, id uuid.UUID) (domain.Pedido, error)
	Pessoas(ctx context.Context, id uuid.UUID) ([]domain.Pessoa, error)
	Aceitar(ctx context.Context, id uuid.UUID, responsavel *uuid.UUID) (domain.Pedido, error)
	Transferir(ctx context.Context, id, novo uuid.UUID) (domain.Pedido, error)
	Encaminhar(ctx context.Context, id, destino uuid.UUID, justificativa string, prazoDias int) (domain.Pedido, error)
	DecidirEncaminhamento(ctx context.Context, id uuid.UUID, aceitar bool, motivo string) (domain.Pedido, error)
	Aprovar(ctx context.Context, id uuid.UUID) (domain.Pedido, error)
	Devolver(ctx context.Context, id uuid.UUID, motivo string) (domain.Pedido, error)
	PedirAnexoMesa(ctx context.Context, pedido uuid.UUID, nome, contentType string) (modkit.UploadTicket, error)
	ListarFeriados(ctx context.Context) ([]domain.Feriado, error)
	SalvarFeriado(ctx context.Context, data, nome string) (domain.Feriado, error)
	RemoverFeriado(ctx context.Context, data string) error
	HorariosLivres(ctx context.Context, servico uuid.UUID, dia string) ([]string, error)
	PessoasDaUnidade(ctx context.Context, unidade uuid.UUID) ([]domain.Pessoa, error)
}

// Handlers agrupa as rotas.
type Handlers struct {
	casos  Casos
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(casos Casos, logger *slog.Logger) *Handlers {
	return &Handlers{casos: casos, logger: logger}
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

type formularioBody struct {
	ID        uuid.UUID      `json:"id"`
	UnidadeID uuid.UUID      `json:"unidade_id"`
	Titulo    string         `json:"titulo"`
	Descricao string         `json:"descricao"`
	Instrucao string         `json:"instrucao"`
	Campos    []domain.Campo `json:"campos"`
	Ativo     bool           `json:"ativo"`
}

func (h *Handlers) SalvarFormulario(w http.ResponseWriter, r *http.Request) {
	var body formularioBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	f, err := h.casos.SalvarFormulario(r.Context(), domain.Formulario{
		ID: body.ID, UnidadeID: body.UnidadeID, Titulo: body.Titulo, Descricao: body.Descricao,
		Instrucao: body.Instrucao, Campos: body.Campos, Ativo: body.Ativo,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, f)
}

func (h *Handlers) ListarFormularios(w http.ResponseWriter, r *http.Request) {
	lista, err := h.casos.ListarFormularios(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) SalvarServico(w http.ResponseWriter, r *http.Request) {
	var body domain.Servico
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	s, err := h.casos.SalvarServico(r.Context(), body)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, s)
}

func (h *Handlers) ListarServicos(w http.ResponseWriter, r *http.Request) {
	lista, err := h.casos.ListarServicosGestao(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

type responsavelBody struct {
	Nome      string    `json:"nome"`
	CPF       string    `json:"cpf"`
	UnidadeID uuid.UUID `json:"unidade_id"`
}

func (h *Handlers) Responsavel(w http.ResponseWriter, r *http.Request) {
	var body responsavelBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	resp, err := h.casos.CadastrarResponsavel(r.Context(), body.Nome, body.CPF, body.UnidadeID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp.CPF = domain.CPFMascarado(resp.CPF)
	httputil.WriteCreated(w, resp)
}

func (h *Handlers) ListarResponsaveis(w http.ResponseWriter, r *http.Request) {
	lista, err := h.casos.ListarResponsaveis(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

type faltaBody struct {
	PedidoID uuid.UUID `json:"pedido_id"`
}

func (h *Handlers) Falta(w http.ResponseWriter, r *http.Request) {
	var body faltaBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.PedidoID == uuid.Nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	if err := h.casos.RegistrarFalta(r.Context(), body.PedidoID); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, map[string]bool{"ok": true})
}

type abrirBody struct {
	ServicoID  uuid.UUID      `json:"servico_id"`
	Nome       string         `json:"nome"`
	CPF        string         `json:"cpf"`
	Email      string         `json:"email"`
	Telefone   string         `json:"telefone"`
	Prioridade string         `json:"prioridade"`
	Data       string         `json:"data"`
	Hora       string         `json:"hora"`
	Respostas  map[string]any `json:"respostas"`
}

func (h *Handlers) abrir(w http.ResponseWriter, r *http.Request, balcao bool) {
	var body abrirBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	servicoID := body.ServicoID
	if !balcao {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			h.fail(w, r, apperrors.BadRequest("serviço inválido"))
			return
		}
		servicoID = id
	}
	p, err := h.casos.Abrir(r.Context(), application.Abertura{
		ServicoID: servicoID, Nome: body.Nome, CPF: body.CPF, Email: body.Email, Telefone: body.Telefone,
		Prioridade: body.Prioridade, Data: body.Data, Hora: body.Hora, Respostas: body.Respostas,
	}, balcao)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, p)
}

func (h *Handlers) AbrirPublico(w http.ResponseWriter, r *http.Request) { h.abrir(w, r, false) }
func (h *Handlers) AbrirBalcao(w http.ResponseWriter, r *http.Request)  { h.abrir(w, r, true) }

func (h *Handlers) Carta(w http.ResponseWriter, r *http.Request) {
	var unidade *uuid.UUID
	if raw := r.URL.Query().Get("unidade"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			h.fail(w, r, apperrors.BadRequest("unidade inválida"))
			return
		}
		unidade = &id
	}
	lista, err := h.casos.Carta(r.Context(), unidade)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

// CartaItem é o serviço ativo com o formulário para preencher.
type CartaItem struct {
	Servico    domain.Servico    `json:"servico"`
	Formulario domain.Formulario `json:"formulario"`
}

type arquivoBody struct {
	Nome        string `json:"nome"`
	ContentType string `json:"content_type"`
}

func (h *Handlers) PedirArquivo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("serviço inválido"))
		return
	}
	var body arquivoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	ticket, err := h.casos.PedirArquivo(r.Context(), id, body.Nome, body.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ticket)
}

func (h *Handlers) LinkArquivo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	url, err := h.casos.LinkArquivo(r.Context(), id, r.URL.Query().Get("campo"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, map[string]string{"url": url})
}

func (h *Handlers) CartaItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("serviço inválido"))
		return
	}
	serv, form, err := h.casos.ServicoPublico(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, CartaItem{Servico: serv, Formulario: form})
}

type arquivoConsulta struct {
	CPF         string `json:"cpf"`
	Nome        string `json:"nome"`
	ContentType string `json:"content_type"`
}

func (h *Handlers) PedirArquivoComplemento(w http.ResponseWriter, r *http.Request) {
	var body arquivoConsulta
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	ticket, err := h.casos.PedirArquivoComplemento(r.Context(), chi.URLParam(r, "numero"), body.CPF, body.Nome, body.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ticket)
}

type respostaPublica struct {
	CPF      string   `json:"cpf"`
	Mensagem string   `json:"mensagem"`
	Arquivos []string `json:"arquivos"`
}

func (h *Handlers) ResponderPublico(w http.ResponseWriter, r *http.Request) {
	var body respostaPublica
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	consulta, err := h.casos.ResponderPublico(r.Context(), chi.URLParam(r, "numero"), body.CPF, body.Mensagem, body.Arquivos)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, consulta)
}

func (h *Handlers) Consultar(w http.ResponseWriter, r *http.Request) {
	c, err := h.casos.Consultar(r.Context(), chi.URLParam(r, "numero"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}

func (h *Handlers) ListarPedidos(w http.ResponseWriter, r *http.Request) {
	lista, err := h.casos.ListarPedidos(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) BuscarPedido(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	ficha, err := h.casos.BuscarPedido(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ficha)
}

type textoBody struct {
	Mensagem string `json:"mensagem"`
	// PrazoDias vale na complementação: dias úteis para o cidadão responder.
	PrazoDias int      `json:"prazo_dias"`
	Anexos    []string `json:"anexos"`
}

type anexosBody struct {
	Anexos []string `json:"anexos"`
}

func (h *Handlers) Complementar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	var body textoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.ComplementarCom(r.Context(), id, application.Complementacao{
		Mensagem: body.Mensagem, PrazoDias: body.PrazoDias, Anexos: body.Anexos,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) Responder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	var body textoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.Responder(r.Context(), id, body.Mensagem)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func (h *Handlers) Anotar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	var body textoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.AnotarCom(r.Context(), id, body.Mensagem, body.Anexos)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

type editarBody struct {
	Justificativa string         `json:"justificativa"`
	Respostas     map[string]any `json:"respostas"`
	// ProcessoSEI ausente mantém; vazio desfaz o vínculo.
	ProcessoSEI *string `json:"processo_sei"`
	// Prioridade e prazo (AAAA-MM-DD) vazios mantêm o valor atual.
	Prioridade string `json:"prioridade"`
	Prazo      string `json:"prazo"`
}

func (h *Handlers) Editar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	var body editarBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.Editar(r.Context(), id, application.Edicao{
		Justificativa: body.Justificativa, Respostas: body.Respostas, Prioridade: body.Prioridade, Prazo: body.Prazo,
		ProcessoSEI: body.ProcessoSEI,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

// Avancar passa à etapa seguinte. O corpo, opcional, traz os anexos da mesa.
func (h *Handlers) Avancar(w http.ResponseWriter, r *http.Request) {
	id, err := idDoPedido(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var body anexosBody
	if r.ContentLength != 0 {
		if err := httputil.Bind(w, r, &body); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	p, err := h.casos.AvancarCom(r.Context(), id, body.Anexos)
	h.responder(w, r, p, err)
}
func (h *Handlers) Cancelar(w http.ResponseWriter, r *http.Request) {
	h.simples(w, r, h.casos.Cancelar)
}

func (h *Handlers) simples(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID) (domain.Pedido, error)) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	p, err := fn(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

type conclusaoBody struct {
	Decisao       string   `json:"decisao"`
	Justificativa string   `json:"justificativa"`
	ProcessoSEI   *string  `json:"processo_sei"`
	Anexos        []string `json:"anexos"`
}

func (h *Handlers) Concluir(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("pedido inválido"))
		return
	}
	var body conclusaoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.ConcluirCom(r.Context(), id, application.Conclusao{
		Decisao: body.Decisao, Justificativa: body.Justificativa, ProcessoSEI: body.ProcessoSEI, Anexos: body.Anexos,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

func idDoPedido(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperrors.BadRequest("pedido inválido")
	}
	return id, nil
}

// corpo lê o id do pedido e o corpo JSON.
func corpo[T any](h *Handlers, w http.ResponseWriter, r *http.Request) (uuid.UUID, T, bool) {
	var body T
	id, err := idDoPedido(r)
	if err != nil {
		h.fail(w, r, err)
		return uuid.Nil, body, false
	}
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return uuid.Nil, body, false
	}
	return id, body, true
}

func (h *Handlers) responder(w http.ResponseWriter, r *http.Request, p domain.Pedido, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, p)
}

// Pessoas lista quem pode assumir o pedido.
func (h *Handlers) Pessoas(w http.ResponseWriter, r *http.Request) {
	id, err := idDoPedido(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	lista, err := h.casos.Pessoas(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

type responsavelPedidoBody struct {
	ResponsavelID *uuid.UUID `json:"responsavel_id"`
}

// Aceitar dá dono ao pedido; sem responsavel_id, quem chama assume.
func (h *Handlers) Aceitar(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[responsavelPedidoBody](h, w, r)
	if !ok {
		return
	}
	p, err := h.casos.Aceitar(r.Context(), id, body.ResponsavelID)
	h.responder(w, r, p, err)
}

// Transferir troca o responsável.
func (h *Handlers) Transferir(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[responsavelPedidoBody](h, w, r)
	if !ok {
		return
	}
	if body.ResponsavelID == nil {
		h.fail(w, r, apperrors.Validation("indique o novo responsável"))
		return
	}
	p, err := h.casos.Transferir(r.Context(), id, *body.ResponsavelID)
	h.responder(w, r, p, err)
}

type encaminharBody struct {
	UnidadeID     uuid.UUID `json:"unidade_id"`
	Justificativa string    `json:"justificativa"`
	PrazoDias     int       `json:"prazo_dias"`
}

// Encaminhar pede a mudança de unidade.
func (h *Handlers) Encaminhar(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[encaminharBody](h, w, r)
	if !ok {
		return
	}
	p, err := h.casos.Encaminhar(r.Context(), id, body.UnidadeID, body.Justificativa, body.PrazoDias)
	h.responder(w, r, p, err)
}

// AceitarEncaminhamento recebe o pedido na unidade de destino.
func (h *Handlers) AceitarEncaminhamento(w http.ResponseWriter, r *http.Request) {
	id, err := idDoPedido(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	p, err := h.casos.DecidirEncaminhamento(r.Context(), id, true, "")
	h.responder(w, r, p, err)
}

// RecusarEncaminhamento devolve o pedido à unidade de origem.
func (h *Handlers) RecusarEncaminhamento(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[textoBody](h, w, r)
	if !ok {
		return
	}
	p, err := h.casos.DecidirEncaminhamento(r.Context(), id, false, body.Mensagem)
	h.responder(w, r, p, err)
}

// Aprovar confirma a conclusão.
func (h *Handlers) Aprovar(w http.ResponseWriter, r *http.Request) { h.simples(w, r, h.casos.Aprovar) }

// Devolver recusa a conclusão.
func (h *Handlers) Devolver(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[textoBody](h, w, r)
	if !ok {
		return
	}
	p, err := h.casos.Devolver(r.Context(), id, body.Mensagem)
	h.responder(w, r, p, err)
}

// PedirAnexoMesa entrega o ticket do anexo da mesa.
func (h *Handlers) PedirAnexoMesa(w http.ResponseWriter, r *http.Request) {
	id, body, ok := corpo[arquivoBody](h, w, r)
	if !ok {
		return
	}
	ticket, err := h.casos.PedirAnexoMesa(r.Context(), id, body.Nome, body.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, ticket)
}

// ListarFeriados devolve o calendário municipal.
func (h *Handlers) ListarFeriados(w http.ResponseWriter, r *http.Request) {
	lista, err := h.casos.ListarFeriados(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

type feriadoBody struct {
	Data string `json:"data"`
	Nome string `json:"nome"`
}

// SalvarFeriado grava o feriado da data.
func (h *Handlers) SalvarFeriado(w http.ResponseWriter, r *http.Request) {
	var body feriadoBody
	if err := httputil.Bind(w, r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	f, err := h.casos.SalvarFeriado(r.Context(), body.Data, body.Nome)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, f)
}

// RemoverFeriado tira a data do calendário.
func (h *Handlers) RemoverFeriado(w http.ResponseWriter, r *http.Request) {
	if err := h.casos.RemoverFeriado(r.Context(), chi.URLParam(r, "data")); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HorariosLivres lista os horários livres do serviço no dia (?data=AAAA-MM-DD).
func (h *Handlers) HorariosLivres(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("serviço inválido"))
		return
	}
	lista, err := h.casos.HorariosLivres(r.Context(), id, r.URL.Query().Get("data"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

// PessoasDaUnidade lista quem pode aprovar a conclusão do serviço da unidade (?unidade_id=).
func (h *Handlers) PessoasDaUnidade(w http.ResponseWriter, r *http.Request) {
	un, err := uuid.Parse(r.URL.Query().Get("unidade_id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("unidade inválida"))
		return
	}
	lista, err := h.casos.PessoasDaUnidade(r.Context(), un)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}
