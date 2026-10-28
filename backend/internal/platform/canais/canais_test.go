package canais

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/secretcrypto"
)

func cifra(t *testing.T, b byte) *secretcrypto.Cipher {
	t.Helper()
	c, err := secretcrypto.NewFromBase64Key(base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(rune(b)), 32))))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// carteiro guarda o que seria enviado.
type carteiro struct {
	enviadas []mail.Mensagem
	smtp     []mail.SMTP
	err      error
}

func (c *carteiro) enviar(_ context.Context, s mail.SMTP, m mail.Mensagem) error {
	c.smtp, c.enviadas = append(c.smtp, s), append(c.enviadas, m)
	return c.err
}

// limpar zera a configuração do e-mail: cada teste começa sem ela.
func limpar(t *testing.T) {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, err := pool.Exec(context.Background(), `DELETE FROM canais_email`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM canais_email`) })
}

func TestValidacao(t *testing.T) {
	ok := Email{Ativo: true, Host: "smtp.x.gov.br", Porta: 587, Remetente: "Central <atendimento@x.gov.br>"}
	if err := ok.Validar(); err != nil {
		t.Fatal(err)
	}
	for nome, e := range map[string]Email{
		"longo":         {Host: strings.Repeat("a", 256), Porta: 587},
		"host com url":  {Host: "smtp://x", Porta: 587},
		"porta":         {Host: "x", Porta: 70000},
		"remetente":     {Host: "x", Porta: 587, Remetente: "atendimento"},
		"sem domínio":   {Host: "x", Porta: 587, Remetente: "a@b"},
		"ligado vazio":  {Ativo: true, Porta: 587},
		"ligado sem de": {Ativo: true, Host: "x", Porta: 587},
	} {
		var inv InvalidaError
		if err := e.Validar(); !errors.As(err, &inv) || inv.Error() == "" {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if e := (Email{Host: " x "}).Normalizar(); e.Host != "x" || e.Porta != 587 {
		t.Fatalf("normalizar: %+v", e)
	}

	for entrada, quer := range map[Contato]Contato{
		{Email: " ana@x.gov.br ", WhatsApp: "(66) 99999-1234"}: {Email: "ana@x.gov.br", WhatsApp: "+5566999991234"},
		{WhatsApp: "+351 912 345 678"}:                         {WhatsApp: "+351912345678"},
		{}:                                                     {},
	} {
		if c, err := NormalizarContato(entrada); err != nil || c.Email != quer.Email || c.WhatsApp != quer.WhatsApp {
			t.Errorf("%+v: %+v %v", entrada, c, err)
		}
	}
	for _, c := range []Contato{{Email: "nao-e-email"}, {Email: "Ana <ana@x.gov.br>"}, {WhatsApp: "123"}} {
		if _, err := NormalizarContato(c); err == nil {
			t.Errorf("%+v aceito", c)
		}
	}
}

func TestServicoEStore(t *testing.T) {
	limpar(t)
	pool := dbtest.Pool(t)
	ctx := context.Background()
	store := NewPostgresStore(pool, cifra(t, 'c'))
	c := &carteiro{}

	// Sem tela e sem ambiente: desligado, e o aviso fica no sino.
	vazio := NovoServico(store, Email{})
	vazio.enviar = c.enviar
	if e, err := vazio.Email(ctx); err != nil || e.Ativo || e.Origem != OrigemNenhuma {
		t.Fatalf("nenhum: %+v %v", e, err)
	}
	if err := vazio.Send(ctx, notificacoes.Mensagem{Para: "a@x.gov.br"}); err != nil || len(c.enviadas) != 0 {
		t.Fatalf("desligado não envia: %v", err)
	}
	if err := vazio.Testar(ctx, "a@x.gov.br"); !errors.Is(err, ErrSemServidor) {
		t.Fatalf("testar sem servidor: %v", err)
	}

	// O ambiente vale enquanto ninguém configurou pela tela.
	s := NovoServico(store, Email{Host: "smtp.env", Senha: "s", Remetente: "env@x.gov.br"})
	s.enviar = c.enviar
	if e, err := s.Email(ctx); err != nil || !e.Ativo || e.Origem != OrigemAmbiente || !e.TemSenha || e.Porta != 587 {
		t.Fatalf("ambiente: %+v %v", e, err)
	}
	if err := s.Send(ctx, notificacoes.Mensagem{Para: "a@x.gov.br", Assunto: "A", Corpo: "B"}); err != nil ||
		len(c.enviadas) != 1 || c.smtp[0].Host != "smtp.env" || c.enviadas[0].Corpo != "B" {
		t.Fatalf("envio pelo ambiente: %v %+v", err, c.enviadas)
	}

	// A tela: desligado, guarda; a senha fica cifrada e não volta.
	if _, err := s.SalvarEmail(ctx, Email{Ativo: true}, false, "admin"); err == nil {
		t.Fatal("ligado sem servidor")
	}
	e, err := s.SalvarEmail(ctx, Email{Host: "smtp.tela", Porta: 465, Usuario: "u", Senha: "segredo", Remetente: "at@x.gov.br"}, false, "admin")
	if err != nil || e.Ativo || !e.TemSenha || e.Origem != OrigemTela || e.UpdatedBy != "admin" {
		t.Fatalf("salvar: %+v %v", e, err)
	}
	var cifrada string
	if err := pool.QueryRow(ctx, `SELECT senha_cifrada FROM canais_email`).Scan(&cifrada); err != nil || cifrada == "" || strings.Contains(cifrada, "segredo") {
		t.Fatalf("senha em claro: %q %v", cifrada, err)
	}
	if j, _ := json.Marshal(e); strings.Contains(string(j), "segredo") {
		t.Fatalf("senha no JSON: %s", j)
	}
	if err := s.Send(ctx, notificacoes.Mensagem{Para: "a@x.gov.br"}); err != nil || len(c.enviadas) != 1 {
		t.Fatalf("desligado na tela não envia: %v", err)
	}
	if err := s.Testar(ctx, "teste@x.gov.br"); err != nil || len(c.enviadas) != 2 || c.smtp[1].Senha != "segredo" || c.smtp[1].Porta != 465 {
		t.Fatalf("testar desligado: %v", err)
	}
	if err := s.Testar(ctx, "nao-e-email"); err == nil {
		t.Fatal("teste para endereço inválido")
	}
	if err := s.Testar(ctx, ""); err == nil {
		t.Fatal("teste sem destino")
	}

	if _, _, err := NewPostgresStore(pool, cifra(t, 'd')).Email(ctx); err == nil {
		t.Fatal("senha cifrada com outra chave")
	}
	// O envio padrão é o do SMTP (sem servidor, não envia).
	if err := NovoServico(store, Email{}).enviar(ctx, mail.SMTP{}, mail.Mensagem{}); err != nil {
		t.Fatal(err)
	}

	// Ligar mantendo a senha; depois remover.
	if e, err = s.SalvarEmail(ctx, Email{Ativo: true, Host: "smtp.tela", Porta: 465, Remetente: "at@x.gov.br"}, false, "bia"); err != nil || !e.TemSenha {
		t.Fatalf("manter a senha: %+v %v", e, err)
	}
	c.err = errors.New("530 auth")
	if err := s.Send(ctx, notificacoes.Mensagem{Para: "a@x.gov.br"}); err == nil {
		t.Fatal("falha do servidor volta")
	}
	if e, err = s.SalvarEmail(ctx, Email{Ativo: true, Host: "smtp.tela", Remetente: "at@x.gov.br"}, true, "bia"); err != nil || e.TemSenha || e.Porta != 587 {
		t.Fatalf("remover a senha: %+v %v", e, err)
	}

	// Outra réplica lê do banco quando o cache vence.
	outra := NovoServico(store, Email{})
	if e, err := outra.Email(ctx); err != nil || e.Origem != OrigemTela || !e.Ativo {
		t.Fatalf("outra réplica: %+v %v", e, err)
	}
	antes := Validade
	Validade = 0
	t.Cleanup(func() { Validade = antes })
	if _, err := outra.Email(ctx); err != nil {
		t.Fatal(err)
	}

	// Falhas do banco.
	falho := NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{})
	if _, err := falho.Email(ctx); err == nil {
		t.Fatal("ler com o banco fora")
	}
	if err := falho.Send(ctx, notificacoes.Mensagem{}); err == nil {
		t.Fatal("enviar com o banco fora")
	}
	if err := falho.Testar(ctx, "a@x.gov.br"); err == nil {
		t.Fatal("testar com o banco fora")
	}
	if _, err := falho.SalvarEmail(ctx, Email{Host: "x"}, false, "a"); err == nil {
		t.Fatal("salvar com o banco fora")
	}
}

func TestContato(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	s := NovoServico(NewPostgresStore(pool, cifra(t, 'c')), Email{})
	id := dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE users SET email = 'kc-' || id || '@sem-email.invalid' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	c, err := s.Contato(ctx, id)
	if err != nil || c.Respondido || c.EmailDoSSO != "" {
		t.Fatalf("sem resposta: %+v %v", c, err)
	}
	if c, err = s.SalvarContato(ctx, id, Contato{Email: "ana@x.gov.br", WhatsApp: "66 99999-1234"}); err != nil ||
		!c.Respondido || c.WhatsApp != "+5566999991234" {
		t.Fatalf("salvar: %+v %v", c, err)
	}
	if _, err := s.SalvarContato(ctx, id, Contato{Email: "x"}); err == nil {
		t.Fatal("e-mail inválido")
	}
	if _, err := s.SalvarContato(ctx, uuid.New(), Contato{}); !errors.Is(err, ErrSemConta) {
		t.Fatalf("sem conta: %v", err)
	}
	if _, err := s.Contato(ctx, uuid.New()); !errors.Is(err, ErrSemConta) {
		t.Fatalf("sem conta: %v", err)
	}
	falho := NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{})
	if _, err := falho.Contato(ctx, id); err == nil || errors.Is(err, ErrSemConta) {
		t.Fatalf("banco fora: %v", err)
	}
	if _, err := falho.SalvarContato(ctx, id, Contato{}); err == nil {
		t.Fatal("salvar com o banco fora")
	}
}

type api struct {
	t *testing.T
	r chi.Router
}

func (a api) req(method, path, body string, quem auth.Identity) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	a.r.ServeHTTP(rec, req.WithContext(auth.WithIdentity(req.Context(), quem)))
	return rec
}

func TestRotas(t *testing.T) {
	limpar(t)
	pool := dbtest.Pool(t)
	c := &carteiro{}
	s := NovoServico(NewPostgresStore(pool, cifra(t, 'c')), Email{})
	s.enviar = c.enviar
	r := chi.NewRouter()
	RegisterRoutes(r, NewHandlers(s, audit.NewWriter(pool), slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	a := api{t: t, r: r}
	id := dbtest.User(t, pool)
	admin := auth.Identity{UserID: id, Username: "admin", Permissions: []string{"canais:manage"}}
	comum := auth.Identity{UserID: id, Subject: "kc-1"}

	quer := func(rec *httptest.ResponseRecorder, status int, trecho string) {
		t.Helper()
		if rec.Code != status || !strings.Contains(rec.Body.String(), trecho) {
			t.Fatalf("esperado %d com %q: %d %s", status, trecho, rec.Code, rec.Body.String())
		}
	}
	quer(a.req(http.MethodGet, "/canais/email", "", comum), http.StatusForbidden, "")
	quer(a.req(http.MethodGet, "/canais/email", "", admin), http.StatusOK, `"origem":"nenhuma"`)
	quer(a.req(http.MethodPost, "/canais/email/testar", `{"para":"a@x.gov.br"}`, admin), http.StatusUnprocessableEntity, "configure")
	quer(a.req(http.MethodPut, "/canais/email", `{`, admin), http.StatusBadRequest, "")
	quer(a.req(http.MethodPut, "/canais/email", `{"ativo":true}`, admin), http.StatusUnprocessableEntity, "servidor")
	quer(a.req(http.MethodPut, "/canais/email", `{"ativo":true,"host":"smtp.x","senha":"s","remetente":"at@x.gov.br"}`, admin),
		http.StatusOK, `"tem_senha":true`)
	quer(a.req(http.MethodPut, "/canais/email", `{"ativo":true,"host":"smtp.x","remetente":"at@x.gov.br"}`,
		auth.Identity{Subject: "kc-adm", Permissions: []string{"canais:manage"}}), http.StatusOK, `"updated_by":"kc-adm"`)
	quer(a.req(http.MethodPost, "/canais/email/testar", `{}`, admin), http.StatusUnprocessableEntity, "")
	quer(a.req(http.MethodPost, "/canais/email/testar", `{"para":"nada"}`, admin), http.StatusUnprocessableEntity, "e-mail")
	quer(a.req(http.MethodPost, "/canais/email/testar", `{"para":"a@x.gov.br"}`, admin), http.StatusOK, `"ok":true`)
	c.err = errors.New("535 credenciais recusadas")
	quer(a.req(http.MethodPost, "/canais/email/testar", `{"para":"a@x.gov.br"}`, admin), http.StatusOK, "535 credenciais recusadas")
	var auditoria int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action LIKE 'canais.email.%' AND created_at > now() - interval '1 minute'`).
		Scan(&auditoria); err != nil || auditoria < 4 {
		t.Fatalf("auditoria: %d %v", auditoria, err)
	}

	// O contato é de cada um, sem permissão especial.
	quer(a.req(http.MethodGet, "/me/contato", "", comum), http.StatusOK, `"respondido":false`)
	quer(a.req(http.MethodPut, "/me/contato", `{`, comum), http.StatusBadRequest, "")
	quer(a.req(http.MethodPut, "/me/contato", `{"email":"x"}`, comum), http.StatusUnprocessableEntity, "e-mail")
	quer(a.req(http.MethodPut, "/me/contato", `{"email":"ana@x.gov.br","whatsapp":"66999991234"}`, comum), http.StatusOK, `"+5566999991234"`)
	quer(a.req(http.MethodGet, "/me/contato", "", auth.Identity{UserID: uuid.New()}), http.StatusNotFound, "")

	// Com o banco fora, a leitura falha.
	falho := chi.NewRouter()
	RegisterRoutes(falho, NewHandlers(NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{}), audit.NewWriter(pool),
		slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	af := api{t: t, r: falho}
	quer(af.req(http.MethodGet, "/canais/email", "", admin), http.StatusInternalServerError, "")
	quer(af.req(http.MethodPost, "/canais/email/testar", `{"para":"a@x.gov.br"}`, admin), http.StatusOK, `"ok":false`)
}
