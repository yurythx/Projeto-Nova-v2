// Package transport expõe o protocolo da Fazenda.
package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// Casos é o que a tela chama. O serviço da aplicação implementa.
type Casos interface {
	Simular(domain.Simulacao) (domain.Simulado, error)
	IniciarGovBR() (govbr.Desafio, error)
	TrocarGovBR(code, verifier string) (application.SessaoGovBR, error)
	Autenticidade(ctx context.Context, codigo string) (domain.Autenticidade, error)
	Abrir(ctx context.Context, quem auth.Identity, a domain.Abertura) (domain.Processo, error)
	Listar(ctx context.Context, quem auth.Identity) ([]domain.Processo, error)
	Painel(ctx context.Context, quem auth.Identity) ([]domain.PrazoPainel, error)
	Alertar(ctx context.Context, quem auth.Identity) (int, error)
	Buscar(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error)
	RegistrarConsulta(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error)
	Exigir(ctx context.Context, quem auth.Identity, id uuid.UUID, fatoNovo bool) (domain.Processo, error)
	Responder(ctx context.Context, quem auth.Identity, id uuid.UUID, respondeu bool) (domain.Processo, error)
	Decidir(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error)
	Recorrer(ctx context.Context, quem auth.Identity, id uuid.UUID, rito bool) (domain.Processo, error)
	GuardarReducao(ctx context.Context, quem auth.Identity, id uuid.UUID, milessimos int64, condicoes int) (domain.Processo, error)
	Parcelar(ctx context.Context, quem auth.Identity, id uuid.UUID, total int64, parcelas int) (domain.Plano, error)
	Baixar(ctx context.Context, quem auth.Identity, id uuid.UUID, primeira bool) (domain.Processo, error)
	MarcarExecucao(ctx context.Context, quem auth.Identity, id uuid.UUID, emExecucao bool) (domain.Processo, error)
	Avisar(ctx context.Context, quem auth.Identity, id uuid.UUID, fato, orientacao string) (string, error)
	ReceberDUA(linhas []domain.LinhaDUA, omitidas int) (domain.ReciboDUA, error)
	EmitirCertidao(ctx context.Context, quem auth.Identity, id uuid.UUID, dados domain.DadosCertidao) (domain.Processo, error)
	AssinarTQC(ctx context.Context, quem auth.Identity, id uuid.UUID, assinado time.Time) (domain.Processo, error)
	GuardarSelo(ctx context.Context, quem auth.Identity, id uuid.UUID, criterios int) (domain.Processo, error)
	GuardarIsencao(ctx context.Context, quem auth.Identity, id uuid.UUID, fim int) (domain.Processo, error)
	AbrirCassacao(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error)
	PedirPeca(ctx context.Context, quem auth.Identity, id uuid.UUID, tipo, nome, contentType string) (modkit.UploadTicket, error)
	GuardarPeca(ctx context.Context, quem auth.Identity, id uuid.UUID, tipo, objectKey string) (domain.Processo, error)
}

// Handlers são as rotas do protocolo.
type Handlers struct {
	casos  Casos
	logger *slog.Logger
}

func NewHandlers(casos Casos, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{casos: casos, logger: logger}
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, application.MapError(err))
}

func (h *Handlers) Simular(w http.ResponseWriter, r *http.Request) {
	var in domain.Simulacao
	if err := httputil.Bind(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Simular(in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) IniciarGovBR(w http.ResponseWriter, r *http.Request) {
	out, err := h.casos.IniciarGovBR()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type trocaRequest struct {
	Code     string `json:"code" validate:"required"`
	Verifier string `json:"verifier" validate:"required"`
}

func (h *Handlers) TrocarGovBR(w http.ResponseWriter, r *http.Request) {
	var req trocaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.TrocarGovBR(req.Code, req.Verifier)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Autenticidade(w http.ResponseWriter, r *http.Request) {
	out, err := h.casos.Autenticidade(r.Context(), chi.URLParam(r, "codigo"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Abrir(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var in domain.Abertura
	if err := httputil.Bind(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Abrir(r.Context(), quem, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, out)
}

type duaRequest struct {
	Linhas   []domain.LinhaDUA `json:"linhas"`
	Omitidas int               `json:"omitidas"`
}

func (h *Handlers) DUA(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.Require(r.Context()); err != nil {
		h.fail(w, r, err)
		return
	}
	var req duaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.ReceberDUA(req.Linhas, req.Omitidas)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, out)
}

func (h *Handlers) Painel(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Painel(r.Context(), quem)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Alertar(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	n, err := h.casos.Alertar(r.Context(), quem)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, struct {
		Publicadas int `json:"publicadas"`
	}{Publicadas: n})
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Listar(r.Context(), quem)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) id(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("ID inválido"))
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handlers) Buscar(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	out, err := h.casos.Buscar(r.Context(), quem, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type fatoRequest struct {
	FatoNovo bool `json:"fato_novo"`
}

func (h *Handlers) Consultar(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	out, err := h.casos.RegistrarConsulta(r.Context(), quem, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Exigir(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req fatoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Exigir(r.Context(), quem, id, req.FatoNovo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type respostaRequest struct {
	Respondeu bool `json:"respondeu"`
}

func (h *Handlers) Responder(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var req respostaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Responder(r.Context(), quem, id, req.Respondeu)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Decidir(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	out, err := h.casos.Decidir(r.Context(), quem, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type ritoRequest struct {
	RitoEntregue bool `json:"rito_entregue"`
}

func (h *Handlers) Recorrer(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req ritoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Recorrer(r.Context(), quem, id, req.RitoEntregue)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type reducaoRequest struct {
	Milessimos int64 `json:"milessimos" validate:"required"`
	Condicoes  int   `json:"condicoes" validate:"required"`
}

func (h *Handlers) Reducao(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req reducaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.GuardarReducao(r.Context(), quem, id, req.Milessimos, req.Condicoes)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type parcelaRequest struct {
	Total    int64 `json:"total" validate:"required"`
	Parcelas int   `json:"parcelas" validate:"required"`
}

func (h *Handlers) Parcelar(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req parcelaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Parcelar(r.Context(), quem, id, req.Total, req.Parcelas)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type baixaRequest struct {
	Primeira bool `json:"primeira"`
}

func (h *Handlers) Baixar(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req baixaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Baixar(r.Context(), quem, id, req.Primeira)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type execucaoRequest struct {
	EmExecucao bool `json:"em_execucao"`
}

func (h *Handlers) Execucao(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req execucaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.MarcarExecucao(r.Context(), quem, id, req.EmExecucao)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type avisoRequest struct {
	Fato       string `json:"fato" validate:"required"`
	Orientacao string `json:"orientacao" validate:"required"`
}

func (h *Handlers) Avisar(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req avisoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.Avisar(r.Context(), quem, id, req.Fato, req.Orientacao)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, map[string]string{"aviso": out})
}

type certidaoRequest struct {
	Adquirente      string `json:"adquirente" validate:"required"`
	Transmitente    string `json:"transmitente" validate:"required"`
	Inscricao       string `json:"inscricao" validate:"required"`
	Matricula       string `json:"matricula" validate:"required"`
	Aliquota        string `json:"aliquota" validate:"required"`
	DataInstrumento string `json:"data_instrumento" validate:"required"`
}

func (h *Handlers) Certidao(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req certidaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	quando, err := time.ParseInLocation("2006-01-02", req.DataInstrumento, domain.Cuiaba)
	if err != nil {
		h.fail(w, r, apperrors.Validation("informe a data do instrumento"))
		return
	}
	out, err := h.casos.EmitirCertidao(r.Context(), quem, id, domain.DadosCertidao{
		Adquirente: req.Adquirente, Transmitente: req.Transmitente, Inscricao: req.Inscricao,
		Matricula: req.Matricula, Aliquota: req.Aliquota, DataInstrumento: quando,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type tqcRequest struct {
	AssinadoEm string `json:"assinado_em" validate:"required"`
}

func (h *Handlers) TQC(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req tqcRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	quando, err := time.ParseInLocation("2006-01-02", req.AssinadoEm, domain.Cuiaba)
	if err != nil {
		h.fail(w, r, apperrors.Validation("informe a data da assinatura"))
		return
	}
	out, err := h.casos.AssinarTQC(r.Context(), quem, id, quando)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type seloRequest struct {
	Criterios int `json:"criterios" validate:"required"`
}

func (h *Handlers) Selo(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req seloRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.GuardarSelo(r.Context(), quem, id, req.Criterios)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type isencaoRequest struct {
	Fim int `json:"fim" validate:"required"`
}

func (h *Handlers) Isencao(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	var req isencaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.GuardarIsencao(r.Context(), quem, id, req.Fim)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) Cassacao(w http.ResponseWriter, r *http.Request) {
	quem, id, ok := h.equipe(w, r)
	if !ok {
		return
	}
	out, err := h.casos.AbrirCassacao(r.Context(), quem, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type pecaPedido struct {
	Tipo        string `json:"tipo" validate:"required"`
	Nome        string `json:"nome" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}

func (h *Handlers) PedirPeca(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var req pecaPedido
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.PedirPeca(r.Context(), quem, id, req.Tipo, req.Nome, req.ContentType)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

type pecaGuardar struct {
	Tipo      string `json:"tipo" validate:"required"`
	ObjectKey string `json:"object_key" validate:"required"`
}

func (h *Handlers) GuardarPeca(w http.ResponseWriter, r *http.Request) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var req pecaGuardar
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.casos.GuardarPeca(r.Context(), quem, id, req.Tipo, req.ObjectKey)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) equipe(w http.ResponseWriter, r *http.Request) (auth.Identity, uuid.UUID, bool) {
	quem, err := auth.Require(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return auth.Identity{}, uuid.Nil, false
	}
	id, ok := h.id(w, r)
	if !ok {
		return auth.Identity{}, uuid.Nil, false
	}
	return quem, id, true
}
