package canais

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// provedorFalso responde como a Meta e a Evolution e guarda os pedidos.
type provedorFalso struct {
	mu      sync.Mutex
	pedidos []*http.Request
	corpos  []map[string]any
	status  int
	corpo   string
}

func (p *provedorFalso) servidor(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c map[string]any
		_ = json.NewDecoder(r.Body).Decode(&c)
		p.mu.Lock()
		p.pedidos, p.corpos = append(p.pedidos, r), append(p.corpos, c)
		status, corpo := p.status, p.corpo
		p.mu.Unlock()
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, corpo)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func limparWhatsApp(t *testing.T) {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, err := pool.Exec(context.Background(), `DELETE FROM canais_whatsapp`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM canais_whatsapp`) })
}

func TestValidacaoDoWhatsApp(t *testing.T) {
	w := WhatsApp{}.Normalizar()
	if w.Provedor != ProvedorMeta || w.MetaIdioma != "pt_BR" || w.Validar() != nil {
		t.Fatalf("padrões: %+v %v", w, w.Validar())
	}
	for nome, w := range map[string]WhatsApp{
		"provedor":       {Provedor: "telegram"},
		"número da Meta": {MetaNumeroID: "abc"},
		"modelo":         {MetaModelo: "Aviso Chamado"},
		"idioma":         {MetaIdioma: "portugues"},
		"url":            {EvolutionURL: "ftp://x"},
		"url com senha":  {EvolutionURL: "http://a:b@x"},
		"instância":      {EvolutionInstancia: "a b"},
		"token longo":    {MetaToken: strings.Repeat("a", 1001)},
		"meta ligada":    {Ativo: true, MetaNumeroID: "123456"},
		"evo ligada":     {Ativo: true, Provedor: ProvedorEvolution, EvolutionURL: "http://evo:8080"},
	} {
		var inv InvalidaError
		if err := w.Normalizar().Validar(); !errors.As(err, &inv) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	ok := WhatsApp{Ativo: true, Provedor: ProvedorEvolution, EvolutionURL: "http://evo:8080/", EvolutionInstancia: "central", TemChave: true}.Normalizar()
	if err := ok.Validar(); err != nil || ok.EvolutionURL != "http://evo:8080" {
		t.Fatalf("evolution completa: %+v %v", ok, err)
	}
	if got := limitar("a\n\tb     c", 10); got != "a b c" {
		t.Fatalf("parâmetro sem quebra: %q", got)
	}
	if got := limitar("abcdef", 4); got != "abc…" {
		t.Fatalf("corte: %q", got)
	}
	if motivo([]byte(`{"error":{"message":"(#131030) número fora da lista"}}`)) != "(#131030) número fora da lista" ||
		motivo([]byte(`{"response":{"message":[{"exists":false}]}}`)) != `[{"exists":false}]` || motivo([]byte(" cru ")) != "cru" {
		t.Fatal("motivo")
	}
}

func TestProvedoresDoWhatsApp(t *testing.T) {
	ctx := context.Background()
	p := &provedorFalso{}
	srv := p.servidor(t)
	antes := MetaAPI
	MetaAPI = srv.URL + "/v21.0"
	t.Cleanup(func() { MetaAPI = antes })

	meta := WhatsApp{Provedor: ProvedorMeta, MetaNumeroID: "1234567", MetaToken: "tk", MetaModelo: "aviso_chamado", MetaIdioma: "pt_BR"}
	a := Aviso{Para: "+5566999991234", Titulo: "Recebemos o seu chamado #7", Mensagem: "Senha:\nacesso", Link: "https://x/7"}
	if err := enviarWhatsApp(ctx, meta, a); err != nil {
		t.Fatal(err)
	}
	r, c := p.pedidos[0], p.corpos[0]
	parametros := c["template"].(map[string]any)["components"].([]any)[0].(map[string]any)["parameters"].([]any)
	if r.URL.Path != "/v21.0/1234567/messages" || r.Header.Get("Authorization") != "Bearer tk" || c["to"] != "5566999991234" ||
		c["template"].(map[string]any)["name"] != "aviso_chamado" || len(parametros) != 3 ||
		parametros[1].(map[string]any)["text"] != "Senha: acesso" {
		t.Fatalf("Meta: %s %v", r.URL.Path, c)
	}

	evo := WhatsApp{Provedor: ProvedorEvolution, EvolutionURL: srv.URL, EvolutionInstancia: "central", EvolutionChave: "ch"}
	if err := enviarWhatsApp(ctx, evo, a); err != nil {
		t.Fatal(err)
	}
	r, c = p.pedidos[1], p.corpos[1]
	if r.URL.Path != "/message/sendText/central" || r.Header.Get("apikey") != "ch" || c["number"] != "5566999991234" ||
		!strings.Contains(c["text"].(string), "*Recebemos o seu chamado #7*") || !strings.Contains(c["text"].(string), "Acompanhe: https://x/7") {
		t.Fatalf("Evolution: %s %v", r.URL.Path, c)
	}
	if !strings.Contains((Aviso{Titulo: "T", Mensagem: "M"}).texto(), "M") || strings.Contains((Aviso{Titulo: "T"}).texto(), "Acompanhe") {
		t.Fatal("texto sem link")
	}

	p.status, p.corpo = http.StatusBadRequest, `{"error":{"message":"modelo não aprovado"}}`
	if err := enviarWhatsApp(ctx, meta, a); err == nil || !strings.Contains(err.Error(), "400: modelo não aprovado") {
		t.Fatalf("recusa: %v", err)
	}
	if err := enviarWhatsApp(ctx, WhatsApp{Provedor: ProvedorEvolution, EvolutionURL: "http://127.0.0.1:1"}, a); err == nil {
		t.Fatal("sem rede")
	}
	if err := enviarWhatsApp(ctx, WhatsApp{Provedor: ProvedorEvolution, EvolutionURL: "http://[::1"}, a); err == nil {
		t.Fatal("endereço inválido")
	}
}

func TestServicoDoWhatsApp(t *testing.T) {
	limparWhatsApp(t)
	pool := dbtest.Pool(t)
	ctx := context.Background()
	store := NewPostgresStore(pool, cifra(t, 'c'))
	s := NovoServico(store, Email{})
	var enviados []Aviso
	s.enviarWA = func(_ context.Context, _ WhatsApp, a Aviso) error {
		enviados = append(enviados, a)
		return nil
	}

	if w, err := s.WhatsApp(ctx); err != nil || w.Ativo || w.Provedor != ProvedorMeta {
		t.Fatalf("nunca configurado: %+v %v", w, err)
	}
	if err := s.SendWhatsApp(ctx, notificacoes.MensagemWhatsApp{Para: "+5566999991234"}); err != nil || len(enviados) != 0 {
		t.Fatalf("desligado não envia: %v", err)
	}
	if err := s.TestarWhatsApp(ctx, "66999991234"); !errors.Is(err, ErrWhatsAppIncompleto) {
		t.Fatalf("testar sem provedor: %v", err)
	}
	if _, err := s.SalvarWhatsApp(ctx, WhatsApp{Ativo: true, MetaNumeroID: "123456", MetaModelo: "aviso"}, false, false, "a"); err == nil {
		t.Fatal("ligar a Meta sem token")
	}
	w, err := s.SalvarWhatsApp(ctx, WhatsApp{Ativo: true, MetaNumeroID: "123456", MetaModelo: "aviso", MetaToken: "segredo-meta",
		Provedor: ProvedorMeta}, false, false, "admin")
	if err != nil || !w.Ativo || !w.TemToken || w.UpdatedBy != "admin" {
		t.Fatalf("salvar Meta: %+v %v", w, err)
	}
	if j, _ := json.Marshal(w); strings.Contains(string(j), "segredo-meta") {
		t.Fatalf("token no JSON: %s", j)
	}
	var cifrado string
	if err := pool.QueryRow(ctx, `SELECT meta_token_cifrado FROM canais_whatsapp`).Scan(&cifrado); err != nil || strings.Contains(cifrado, "segredo") {
		t.Fatalf("token em claro: %v", err)
	}
	// Troca para a Evolution: o token da Meta fica guardado.
	w, err = s.SalvarWhatsApp(ctx, WhatsApp{Ativo: true, Provedor: ProvedorEvolution, MetaNumeroID: "123456", MetaModelo: "aviso",
		EvolutionURL: "http://evo:8080", EvolutionInstancia: "central", EvolutionChave: "ch"}, false, false, "bia")
	if err != nil || !w.TemToken || !w.TemChave || w.Provedor != ProvedorEvolution {
		t.Fatalf("trocar de provedor: %+v %v", w, err)
	}
	if err := s.SendWhatsApp(ctx, notificacoes.MensagemWhatsApp{Para: "+5566999991234", Titulo: "T"}); err != nil || len(enviados) != 1 {
		t.Fatalf("ligado envia: %v", err)
	}
	if err := s.TestarWhatsApp(ctx, "(66) 99999-1234"); err != nil || enviados[1].Para != "+5566999991234" {
		t.Fatalf("testar: %v %+v", err, enviados)
	}
	if err := s.TestarWhatsApp(ctx, "123"); err == nil {
		t.Fatal("teste para número inválido")
	}
	if err := s.TestarWhatsApp(ctx, ""); err == nil {
		t.Fatal("teste sem número")
	}
	// Remover a chave em uso não deixa ligado.
	if _, err := s.SalvarWhatsApp(ctx, WhatsApp{Ativo: true, Provedor: ProvedorEvolution, EvolutionURL: "http://evo:8080",
		EvolutionInstancia: "central"}, false, true, "bia"); err == nil {
		t.Fatal("ligado sem a chave")
	}
	if w, err = s.SalvarWhatsApp(ctx, WhatsApp{Provedor: ProvedorEvolution}, true, true, "bia"); err != nil || w.TemToken || w.TemChave {
		t.Fatalf("remover credenciais: %+v %v", w, err)
	}

	// Outra réplica, chave errada e banco fora.
	if _, err := NovoServico(store, Email{}).WhatsApp(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SalvarWhatsApp(ctx, WhatsApp{MetaToken: "outro"}, false, false, "a"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewPostgresStore(pool, cifra(t, 'd')).WhatsApp(ctx); err == nil {
		t.Fatal("token cifrado com outra chave")
	}
	if _, err := s.SalvarWhatsApp(ctx, WhatsApp{Provedor: "x"}, false, false, "a"); err == nil {
		t.Fatal("provedor inválido")
	}
	falho := NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{})
	if _, err := falho.WhatsApp(ctx); err == nil {
		t.Fatal("ler com o banco fora")
	}
	if err := falho.SendWhatsApp(ctx, notificacoes.MensagemWhatsApp{}); err == nil {
		t.Fatal("enviar com o banco fora")
	}
	if err := falho.TestarWhatsApp(ctx, "66999991234"); err == nil {
		t.Fatal("testar com o banco fora")
	}
	if _, err := falho.SalvarWhatsApp(ctx, WhatsApp{}, false, false, "a"); err == nil {
		t.Fatal("salvar com o banco fora")
	}
	gravaFalha := NovoServico(storeQueFalhaAoGravar{store}, Email{})
	if _, err := gravaFalha.SalvarWhatsApp(ctx, WhatsApp{}, false, false, "a"); err == nil {
		t.Fatal("gravar com o banco fora")
	}
	if _, err := NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')).SalvarWhatsApp(ctx, WhatsApp{MetaToken: "t", EvolutionChave: "c"}, false, false, "a"); err == nil {
		t.Fatal("store grava com o banco fora")
	}
}

// storeQueFalhaAoGravar lê do banco e falha ao gravar.
type storeQueFalhaAoGravar struct{ *PostgresStore }

func (storeQueFalhaAoGravar) SalvarWhatsApp(context.Context, WhatsApp, bool, bool, string) (WhatsApp, error) {
	return WhatsApp{}, errors.New("banco fora")
}

func TestRotasDoWhatsApp(t *testing.T) {
	limparWhatsApp(t)
	pool := dbtest.Pool(t)
	s := NovoServico(NewPostgresStore(pool, cifra(t, 'c')), Email{})
	var falha error
	s.enviarWA = func(context.Context, WhatsApp, Aviso) error { return falha }
	r := chi.NewRouter()
	RegisterRoutes(r, NewHandlers(s, audit.NewWriter(pool), slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	a := api{t: t, r: r}
	admin := auth.Identity{Username: "admin", Permissions: []string{"canais:manage"}}
	quer := func(rec *httptest.ResponseRecorder, status int, trecho string) {
		t.Helper()
		if rec.Code != status || !strings.Contains(rec.Body.String(), trecho) {
			t.Fatalf("esperado %d com %q: %d %s", status, trecho, rec.Code, rec.Body.String())
		}
	}
	quer(a.req(http.MethodGet, "/canais/whatsapp", "", auth.Identity{}), http.StatusForbidden, "")
	quer(a.req(http.MethodGet, "/canais/whatsapp", "", admin), http.StatusOK, `"provedor":"meta"`)
	quer(a.req(http.MethodPost, "/canais/whatsapp/testar", `{"para":"66999991234"}`, admin), http.StatusUnprocessableEntity, "provedor")
	quer(a.req(http.MethodPut, "/canais/whatsapp", `{`, admin), http.StatusBadRequest, "")
	quer(a.req(http.MethodPut, "/canais/whatsapp", `{"provedor":"telegram"}`, admin), http.StatusUnprocessableEntity, "")
	quer(a.req(http.MethodPut, "/canais/whatsapp", `{"ativo":true,"provedor":"evolution"}`, admin), http.StatusUnprocessableEntity, "Evolution")
	quer(a.req(http.MethodPut, "/canais/whatsapp", `{"ativo":true,"provedor":"evolution","evolution_url":"http://evo:8080",
		"evolution_instancia":"central","evolution_chave":"ch"}`, admin), http.StatusOK, `"tem_chave":true`)
	quer(a.req(http.MethodPost, "/canais/whatsapp/testar", `{}`, admin), http.StatusUnprocessableEntity, "")
	quer(a.req(http.MethodPost, "/canais/whatsapp/testar", `{"para":"123"}`, admin), http.StatusUnprocessableEntity, "WhatsApp")
	quer(a.req(http.MethodPost, "/canais/whatsapp/testar", `{"para":"66999991234"}`, admin), http.StatusOK, `"ok":true`)
	falha = errors.New("número não existe no WhatsApp")
	quer(a.req(http.MethodPost, "/canais/whatsapp/testar", `{"para":"66999991234"}`, admin), http.StatusOK, "número não existe")
	var auditoria int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE resource_type = 'canais_whatsapp'
		AND created_at > now() - interval '1 minute'`).Scan(&auditoria); err != nil || auditoria < 3 {
		t.Fatalf("auditoria: %d %v", auditoria, err)
	}
	falho := chi.NewRouter()
	RegisterRoutes(falho, NewHandlers(NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{}), audit.NewWriter(pool),
		slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	quer(api{t: t, r: falho}.req(http.MethodGet, "/canais/whatsapp", "", admin), http.StatusInternalServerError, "")
}
