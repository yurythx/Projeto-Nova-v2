package canais

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// Ações de auditoria (nunca com a senha).
const (
	ActionEmailAlterado = "canais.email.alterado"
	ActionEmailTestado  = "canais.email.testado"
	resourceEmail       = "canais_email"
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
	case errors.Is(err, ErrSemConta):
		return apperrors.NotFound("sua conta ainda não está cadastrada na plataforma")
	}
	return err
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, MapError(err))
}

func (h *Handlers) registrar(r *http.Request, acao string, meta map[string]any) {
	e := audit.FromRequest(r)
	e.Action, e.ResourceType, e.ResourceID, e.Metadata = acao, resourceEmail, "1", meta
	_ = h.audit.Record(r.Context(), e)
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
	id, _ := auth.IdentityFromContext(r.Context())
	por := id.Username
	if por == "" {
		por = id.Subject
	}
	e, err := h.svc.SalvarEmail(r.Context(), Email{Ativo: req.Ativo, Host: req.Host, Porta: req.Porta, Usuario: req.Usuario,
		Senha: req.Senha, Remetente: req.Remetente}, req.RemoverSenha, por)
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
