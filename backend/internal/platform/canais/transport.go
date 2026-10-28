package canais

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// Ações de auditoria (nunca com a senha).
const (
	ActionEmailAlterado    = "canais.email.alterado"
	ActionEmailTestado     = "canais.email.testado"
	ActionWhatsAppAlterado = "canais.whatsapp.alterado"
	ActionWhatsAppTestado  = "canais.whatsapp.testado"
	resourceEmail          = "canais_email"
	resourceWhatsApp       = "canais_whatsapp"
)

// Handlers dos canais.
type Handlers struct {
	svc    *Servico
	audit  *audit.Writer
	logger *slog.Logger
}

// NewHandlers cria os handlers.
func NewHandlers(svc *Servico, auditWriter *audit.Writer, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, audit: auditWriter, logger: logger}
}

// RegisterRoutes monta as rotas (r já autenticado): o contato é de cada
// um; a configuração, de quem tem canais:manage.
func RegisterRoutes(r chi.Router, h *Handlers, logger *slog.Logger) {
	r.Get("/me/contato", h.ObterContato)
	r.Put("/me/contato", h.SalvarContato)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequirePermission(logger, auth.PermCanaisManage))
		r.Get("/canais/email", h.ObterEmail)
		r.Put("/canais/email", h.SalvarEmail)
		r.Post("/canais/email/testar", h.TestarEmail)
		r.Get("/canais/whatsapp", h.ObterWhatsApp)
		r.Put("/canais/whatsapp", h.SalvarWhatsApp)
		r.Post("/canais/whatsapp/testar", h.TestarWhatsApp)
	})
}

// MapError traduz os erros do pacote.
func MapError(err error) error {
	var inv InvalidaError
	switch {
	case errors.As(err, &inv):
		return apperrors.Validation(inv.Msg)
	case errors.Is(err, ErrSemServidor):
		return apperrors.Validation("configure e salve o servidor SMTP e o remetente antes de testar")
	case errors.Is(err, ErrWhatsAppIncompleto):
		return apperrors.Validation("configure e salve o provedor do WhatsApp (com a credencial) antes de testar")
	case errors.Is(err, ErrSemConta):
		return apperrors.NotFound("sua conta ainda não está cadastrada na plataforma")
	}
	return err
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, MapError(err))
}

func (h *Handlers) registrar(r *http.Request, acao string, meta map[string]any) {
	recurso := resourceEmail
	if strings.HasPrefix(acao, "canais.whatsapp.") {
		recurso = resourceWhatsApp
	}
	e := audit.FromRequest(r)
	e.Action, e.ResourceType, e.ResourceID, e.Metadata = acao, recurso, "1", meta
	_ = h.audit.Record(r.Context(), e)
}

func autor(r *http.Request) string {
	id, _ := auth.IdentityFromContext(r.Context())
	if id.Username != "" {
		return id.Username
	}
	return id.Subject
}

func (h *Handlers) ObterEmail(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.Email(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, e)
}

type emailRequest struct {
	Ativo        bool   `json:"ativo"`
	Host         string `json:"host" validate:"max=255"`
	Porta        int    `json:"porta" validate:"min=0,max=65535"`
	Usuario      string `json:"usuario" validate:"max=255"`
	Senha        string `json:"senha" validate:"max=500"`
	RemoverSenha bool   `json:"remover_senha"`
	Remetente    string `json:"remetente" validate:"max=255"`
}

func (h *Handlers) SalvarEmail(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	e, err := h.svc.SalvarEmail(r.Context(), Email{Ativo: req.Ativo, Host: req.Host, Porta: req.Porta, Usuario: req.Usuario,
		Senha: req.Senha, Remetente: req.Remetente}, req.RemoverSenha, autor(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.registrar(r, ActionEmailAlterado, map[string]any{"ativo": e.Ativo, "host": e.Host, "porta": e.Porta,
		"usuario": e.Usuario, "remetente": e.Remetente, "tem_senha": e.TemSenha})
	httputil.WriteOK(w, e)
}

type testeRequest struct {
	Para string `json:"para" validate:"required,max=254"`
}

// TestarEmail manda a mensagem de teste. A falha do servidor volta como
// resultado (200, ok false), com a mensagem dele: é o que se quer ver.
func (h *Handlers) TestarEmail(w http.ResponseWriter, r *http.Request) {
	var req testeRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	err := h.svc.Testar(r.Context(), req.Para)
	var inv InvalidaError
	if errors.As(err, &inv) || errors.Is(err, ErrSemServidor) {
		h.fail(w, r, err)
		return
	}
	out := map[string]any{"ok": err == nil}
	if err != nil {
		out["erro"] = err.Error()
	}
	h.registrar(r, ActionEmailTestado, map[string]any{"ok": err == nil})
	httputil.WriteOK(w, out)
}

func (h *Handlers) ObterContato(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.Contato(r.Context(), id.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}

type contatoRequest struct {
	Email    string `json:"email" validate:"max=254"`
	WhatsApp string `json:"whatsapp" validate:"max=32"`
}

// SalvarContato grava o contato da própria pessoa. Vazio também é
// resposta: ela preferiu não deixar, e não se pergunta de novo.
func (h *Handlers) SalvarContato(w http.ResponseWriter, r *http.Request) {
	var req contatoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	id, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.SalvarContato(r.Context(), id.UserID, Contato{Email: req.Email, WhatsApp: req.WhatsApp})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}

func (h *Handlers) ObterWhatsApp(w http.ResponseWriter, r *http.Request) {
	wa, err := h.svc.WhatsApp(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, wa)
}

type whatsAppRequest struct {
	Ativo              bool   `json:"ativo"`
	Provedor           string `json:"provedor" validate:"omitempty,oneof=meta evolution"`
	MetaNumeroID       string `json:"meta_numero_id" validate:"max=32"`
	MetaToken          string `json:"meta_token" validate:"max=1000"`
	RemoverToken       bool   `json:"remover_token"`
	MetaModelo         string `json:"meta_modelo" validate:"max=512"`
	MetaIdioma         string `json:"meta_idioma" validate:"max=10"`
	EvolutionURL       string `json:"evolution_url" validate:"max=500"`
	EvolutionInstancia string `json:"evolution_instancia" validate:"max=100"`
	EvolutionChave     string `json:"evolution_chave" validate:"max=500"`
	RemoverChave       bool   `json:"remover_chave"`
	Receber            bool   `json:"receber"`
	MetaAppSecret      string `json:"meta_app_secret" validate:"max=200"`
	NovoWebhook        bool   `json:"novo_webhook"`
}

func (h *Handlers) SalvarWhatsApp(w http.ResponseWriter, r *http.Request) {
	var req whatsAppRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	wa, err := h.svc.SalvarWhatsApp(r.Context(), WhatsApp{Ativo: req.Ativo, Provedor: req.Provedor, MetaNumeroID: req.MetaNumeroID,
		MetaToken: req.MetaToken, MetaModelo: req.MetaModelo, MetaIdioma: req.MetaIdioma, EvolutionURL: req.EvolutionURL,
		EvolutionInstancia: req.EvolutionInstancia, EvolutionChave: req.EvolutionChave, Receber: req.Receber,
		MetaAppSecret: req.MetaAppSecret, NovoWebhook: req.NovoWebhook}, req.RemoverToken, req.RemoverChave, autor(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.registrar(r, ActionWhatsAppAlterado, map[string]any{"ativo": wa.Ativo, "provedor": wa.Provedor,
		"meta_numero_id": wa.MetaNumeroID, "meta_modelo": wa.MetaModelo, "evolution_url": wa.EvolutionURL,
		"evolution_instancia": wa.EvolutionInstancia, "tem_token": wa.TemToken, "tem_chave": wa.TemChave,
		"receber": wa.Receber, "tem_app_secret": wa.TemAppSecret, "novo_webhook": req.NovoWebhook})
	httputil.WriteOK(w, wa)
}

type testeWhatsAppRequest struct {
	Para string `json:"para" validate:"required,max=32"`
}

// TestarWhatsApp manda o aviso de teste. A recusa do provedor volta como
// resultado (200, ok false), com a mensagem dele.
func (h *Handlers) TestarWhatsApp(w http.ResponseWriter, r *http.Request) {
	var req testeWhatsAppRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	err := h.svc.TestarWhatsApp(r.Context(), req.Para)
	var inv InvalidaError
	if errors.As(err, &inv) || errors.Is(err, ErrWhatsAppIncompleto) {
		h.fail(w, r, err)
		return
	}
	out := map[string]any{"ok": err == nil}
	if err != nil {
		out["erro"] = err.Error()
	}
	h.registrar(r, ActionWhatsAppTestado, map[string]any{"ok": err == nil})
	httputil.WriteOK(w, out)
}

// RegisterWebhookRoutes monta o webhook do WhatsApp, público: quem o chama
// é o provedor. Só o token secreto do endereço (e, na Meta, a assinatura)
// dá acesso; o resto é 404.
func RegisterWebhookRoutes(r chi.Router, h *Handlers) {
	r.Get("/canais/whatsapp/webhook/{token}", h.VerificarWebhook)
	r.Post("/canais/whatsapp/webhook/{token}", h.ReceberWebhook)
}

// VerificarWebhook responde ao desafio da Meta ao cadastrar o endereço.
func (h *Handlers) VerificarWebhook(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	desafio, err := h.svc.Verificar(r.Context(), chi.URLParam(r, "token"), q.Get("hub.mode"), q.Get("hub.verify_token"), q.Get("hub.challenge"))
	if err != nil {
		h.falhaWebhook(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(desafio))
}

// ReceberWebhook grava as mensagens que o provedor mandou.
func (h *Handlers) ReceberWebhook(w http.ResponseWriter, r *http.Request) {
	corpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.Validation("corpo do webhook grande demais"))
		return
	}
	if err := h.svc.Webhook(r.Context(), chi.URLParam(r, "token"), r.Header.Get("X-Hub-Signature-256"), corpo); err != nil {
		h.falhaWebhook(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) falhaWebhook(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrWebhookDesconhecido):
		httputil.WriteError(w, r, h.logger, apperrors.NotFound("recurso não encontrado"))
	case errors.Is(err, ErrAssinatura):
		httputil.WriteError(w, r, h.logger, apperrors.Unauthorized("assinatura inválida"))
	default:
		h.fail(w, r, err)
	}
}
