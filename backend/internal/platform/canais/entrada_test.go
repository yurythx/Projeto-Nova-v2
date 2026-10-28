package canais

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// recebedorFalso faz o papel do atendimento.
type recebedorFalso struct {
	chamadas []string
	resposta string
	err      error
}

func (r *recebedorFalso) ReceberWhatsApp(_ context.Context, usuario uuid.UUID, texto, id string) (string, error) {
	r.chamadas = append(r.chamadas, usuario.String()+"|"+texto+"|"+id)
	return r.resposta, r.err
}

func assinar(segredo string, corpo []byte) string {
	m := hmac.New(sha256.New, []byte(segredo))
	m.Write(corpo)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

const corpoMeta = `{"object":"whatsapp_business_account","entry":[{"changes":[{"value":{
	"messages":[{"from":"556699991234","id":"wamid.1","type":"text","text":{"body":"#7 voltou a funcionar"}},
	{"from":"556699991234","id":"wamid.2","type":"image"}],
	"statuses":[{"id":"wamid.0","status":"read"}]}}]}]}`

func TestFormatosDaEntrada(t *testing.T) {
	if m := mensagensDaMeta([]byte(corpoMeta)); len(m) != 1 || m[0] != (Recebida{De: "556699991234", ID: "wamid.1", Texto: "#7 voltou a funcionar"}) {
		t.Fatalf("Meta: %+v", m)
	}
	if m := mensagensDaMeta([]byte(`{`)); len(m) != 0 {
		t.Fatalf("Meta ilegível: %+v", m)
	}
	casos := map[string]int{
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"A1"},"message":{"conversation":"oi"}}}`:                 1,
		`{"event":"MESSAGES_UPSERT","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"A2"},"message":{"extendedTextMessage":{"text":"ok"}}}}`: 1,
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","fromMe":true,"id":"A3"}}}`:                                   0,
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"1203630@g.us","id":"A4"},"message":{"conversation":"grupo"}}}`:                              0,
		`{"event":"connection.update","data":{}}`: 0,
		`{`: 0,
	}
	for corpo, quer := range casos {
		if m := mensagensDaEvolution([]byte(corpo)); len(m) != quer {
			t.Errorf("%s: %+v", corpo, m)
		}
	}
	if m := mensagensDaEvolution([]byte(`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"A2"},
		"message":{"extendedTextMessage":{"text":"ok"}}}}`)); m[0].Texto != "ok" || m[0].De != "5566999991234" {
		t.Fatalf("texto estendido: %+v", m)
	}
	for de, quer := range map[string][]string{
		"556699991234":   {"+556699991234", "+5566999991234"},
		"+5566999991234": {"+5566999991234", "+556699991234"},
		"5566899991234":  {"+5566899991234"},
		"351912345678":   {"+351912345678"},
	} {
		if got := variantes(de); strings.Join(got, ",") != strings.Join(quer, ",") {
			t.Errorf("%s: %v", de, got)
		}
	}
	corpo := []byte("x")
	if !assinaturaValida("s", assinar("s", corpo), corpo) || assinaturaValida("s", assinar("t", corpo), corpo) ||
		assinaturaValida("s", "sha256=zz", corpo) || assinaturaValida("s", "md5=1", corpo) || assinaturaValida("", assinar("", corpo), corpo) {
		t.Fatal("assinatura")
	}
	if len(novoToken()) != 48 || novoToken() == novoToken() {
		t.Fatal("token")
	}
}

func TestEntradaPeloWhatsApp(t *testing.T) {
	limparWhatsApp(t)
	pool := dbtest.Pool(t)
	ctx := context.Background()
	store := NewPostgresStore(pool, cifra(t, 'c'))
	rec := &recebedorFalso{resposta: "Recebemos a sua mensagem no chamado #7."}
	s := NovoServico(store, Email{}).WithAPIPublica("https://api.nova.gov.br/").WithRecebedor(rec)
	var respostas []string
	var falhaResposta error
	s.responderWA = func(_ context.Context, _ WhatsApp, para, texto string) error {
		respostas = append(respostas, para+"|"+texto)
		return falhaResposta
	}
	ana := dbtest.User(t, pool)
	// Outros testes usam o mesmo número; aqui ele é só da Ana.
	if _, err := pool.Exec(ctx, `UPDATE users SET contato_whatsapp = CASE WHEN id = $1 THEN '+5566999991234' ELSE '' END
		WHERE id = $1 OR contato_whatsapp IN ('+5566999991234', '+556699991234')`, ana); err != nil {
		t.Fatal(err)
	}

	// Ligar a entrada: pede o envio ligado e, na Meta, o App Secret.
	base := WhatsApp{Ativo: true, Provedor: ProvedorMeta, MetaNumeroID: "123456", MetaModelo: "aviso", MetaToken: "tk"}
	semEnvio := base
	semEnvio.Ativo, semEnvio.Receber = false, true
	if _, err := s.SalvarWhatsApp(ctx, semEnvio, false, false, "a"); err == nil {
		t.Fatal("receber sem enviar")
	}
	semSegredo := base
	semSegredo.Receber = true
	if _, err := s.SalvarWhatsApp(ctx, semSegredo, false, false, "a"); err == nil {
		t.Fatal("Meta sem App Secret")
	}
	ligado := semSegredo
	ligado.MetaAppSecret = "app-secret"
	w, err := s.SalvarWhatsApp(ctx, ligado, false, false, "a")
	if err != nil || !w.Receber || !w.TemAppSecret || !strings.HasPrefix(w.WebhookURL, "https://api.nova.gov.br/api/v1/canais/whatsapp/webhook/") {
		t.Fatalf("ligar a entrada: %+v %v", w, err)
	}
	token := w.WebhookToken
	// Salvar de novo mantém o endereço; pedir outro troca.
	semSegredo.Receber = true
	if w, err = s.SalvarWhatsApp(ctx, semSegredo, false, false, "a"); err != nil || w.WebhookToken != token || !w.TemAppSecret {
		t.Fatalf("manter o endereço: %+v %v", w, err)
	}
	if lido, err := s.WhatsApp(ctx); err != nil || lido.WebhookURL != w.WebhookURL {
		t.Fatalf("leitura com o endereço: %+v %v", lido, err)
	}

	// Desafio da Meta ao cadastrar.
	if d, err := s.Verificar(ctx, token, "subscribe", token, "1158201444"); err != nil || d != "1158201444" {
		t.Fatalf("desafio: %q %v", d, err)
	}
	if _, err := s.Verificar(ctx, token, "subscribe", "outro", "1"); !errors.Is(err, ErrAssinatura) {
		t.Fatalf("verificação errada: %v", err)
	}
	if _, err := s.Verificar(ctx, "errado", "subscribe", "errado", "1"); !errors.Is(err, ErrWebhookDesconhecido) {
		t.Fatalf("token errado: %v", err)
	}

	// Mensagem da Meta, sem o nono dígito: acha a Ana e confirma.
	corpo := []byte(corpoMeta)
	if err := s.Webhook(ctx, token, assinar("app-secret", corpo), corpo); err != nil {
		t.Fatal(err)
	}
	if len(rec.chamadas) != 1 || rec.chamadas[0] != ana.String()+"|#7 voltou a funcionar|wamid.1" ||
		len(respostas) != 1 || respostas[0] != "556699991234|Recebemos a sua mensagem no chamado #7." {
		t.Fatalf("entrada: %v %v", rec.chamadas, respostas)
	}
	if err := s.Webhook(ctx, token, assinar("outro", corpo), corpo); !errors.Is(err, ErrAssinatura) {
		t.Fatalf("assinatura errada: %v", err)
	}
	if err := s.Webhook(ctx, "errado", "", corpo); !errors.Is(err, ErrWebhookDesconhecido) {
		t.Fatalf("token errado: %v", err)
	}

	// A confirmação que falha não desfaz a resposta gravada; sem
	// confirmação, nada volta; o erro do atendimento volta para o provedor.
	falhaResposta = errors.New("fora")
	if err := s.Webhook(ctx, token, assinar("app-secret", corpo), corpo); err != nil {
		t.Fatal(err)
	}
	falhaResposta, rec.resposta = nil, ""
	respostas = nil
	if err := s.Webhook(ctx, token, assinar("app-secret", corpo), corpo); err != nil || len(respostas) != 0 {
		t.Fatalf("sem confirmação: %v %v", err, respostas)
	}
	rec.err = errors.New("banco fora")
	if err := s.Webhook(ctx, token, assinar("app-secret", corpo), corpo); err == nil {
		t.Fatal("o erro do atendimento volta")
	}
	rec.err = nil

	// Evolution: só o token do endereço; número desconhecido e vazio não são erro.
	evo := WhatsApp{Ativo: true, Receber: true, Provedor: ProvedorEvolution, EvolutionURL: "http://evo:8080", EvolutionInstancia: "central", EvolutionChave: "ch"}
	if w, err = s.SalvarWhatsApp(ctx, evo, false, false, "a"); err != nil || w.WebhookToken != token {
		t.Fatalf("Evolution: %+v %v", w, err)
	}
	antes := len(rec.chamadas)
	for _, c := range []string{
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"E1"},"message":{"conversation":"oi"}}}`,
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5511888887777@s.whatsapp.net","id":"E2"},"message":{"conversation":"oi"}}}`,
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"E3"},"message":{"conversation":" "}}}`,
		`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net"},"message":{"conversation":"sem id"}}}`,
	} {
		if err := s.Webhook(ctx, token, "", []byte(c)); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.chamadas) != antes+1 {
		t.Fatalf("Evolution grava só o que é de alguém: %v", rec.chamadas[antes:])
	}
	evo.NovoWebhook = true
	if w, err = s.SalvarWhatsApp(ctx, evo, false, false, "a"); err != nil || w.WebhookToken == token {
		t.Fatalf("novo endereço: %+v %v", w, err)
	}
	if err := s.Webhook(ctx, token, "", nil); !errors.Is(err, ErrWebhookDesconhecido) {
		t.Fatalf("o endereço antigo deixa de valer: %v", err)
	}
	// Sem recebedor (atendimento fora), nada entra.
	if err := NovoServico(store, Email{}).Webhook(ctx, w.WebhookToken, "", []byte(`{"event":"messages.upsert","data":{"key":{"remoteJid":"5566999991234@s.whatsapp.net","id":"E9"},"message":{"conversation":"oi"}}}`)); err != nil {
		t.Fatal(err)
	}
	// Desligada a entrada, o endereço some.
	evo.Receber, evo.NovoWebhook = false, false
	if w, err = s.SalvarWhatsApp(ctx, evo, false, false, "a"); err != nil || w.WebhookURL != "" {
		t.Fatalf("entrada desligada: %+v %v", w, err)
	}
	if err := s.Webhook(ctx, token, "", nil); !errors.Is(err, ErrWebhookDesconhecido) {
		t.Fatalf("desligada: %v", err)
	}
	if err := NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{}).Webhook(ctx, "x", "", nil); err == nil || errors.Is(err, ErrWebhookDesconhecido) {
		t.Fatalf("banco fora: %v", err)
	}
}

func TestDonoDoNumero(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	store := NewPostgresStore(pool, cifra(t, 'c'))
	numero := "+55669" + uuid.NewString()[:8]
	numero = "+5566" + strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return '1'
	}, numero[6:]) + "1"
	um, dois := dbtest.User(t, pool), dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE users SET contato_whatsapp = $2 WHERE id = $1`, um, numero); err != nil {
		t.Fatal(err)
	}
	if id, ok, err := store.UsuarioPorWhatsApp(ctx, []string{numero}); err != nil || !ok || id != um {
		t.Fatalf("dono: %v %v %v", id, ok, err)
	}
	if _, ok, err := store.UsuarioPorWhatsApp(ctx, []string{"+5500000000000"}); err != nil || ok {
		t.Fatalf("ninguém: %v %v", ok, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET contato_whatsapp = $2 WHERE id = $1`, dois, numero); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.UsuarioPorWhatsApp(ctx, []string{numero}); err != nil || ok {
		t.Fatalf("dois donos é ambíguo: %v %v", ok, err)
	}
	for _, db := range []any{dbtest.Fail{}, dbtest.ScanFail{}, dbtest.RowsErr{}} {
		var s *PostgresStore
		switch d := db.(type) {
		case dbtest.Fail:
			s = NewPostgresStore(d, cifra(t, 'c'))
		case dbtest.ScanFail:
			s = NewPostgresStore(d, cifra(t, 'c'))
		case dbtest.RowsErr:
			s = NewPostgresStore(d, cifra(t, 'c'))
		}
		if _, _, err := s.UsuarioPorWhatsApp(ctx, []string{numero}); err == nil {
			t.Errorf("%T: sem erro", db)
		}
	}
}

func TestRespostaEmTextoLivre(t *testing.T) {
	ctx := context.Background()
	p := &provedorFalso{}
	srv := p.servidor(t)
	antes := MetaAPI
	MetaAPI = srv.URL
	t.Cleanup(func() { MetaAPI = antes })
	if err := responderTexto(ctx, WhatsApp{Provedor: ProvedorMeta, MetaNumeroID: "1", MetaToken: "tk"}, "+556699991234", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := responderTexto(ctx, WhatsApp{Provedor: ProvedorEvolution, EvolutionURL: srv.URL, EvolutionInstancia: "c", EvolutionChave: "ch"}, "556699991234", "ok"); err != nil {
		t.Fatal(err)
	}
	if p.corpos[0]["type"] != "text" || p.corpos[0]["to"] != "556699991234" || p.corpos[1]["text"] != "ok" || p.pedidos[1].Header.Get("apikey") != "ch" {
		t.Fatalf("texto livre: %v", p.corpos)
	}
}

func TestRotasDoWebhook(t *testing.T) {
	limparWhatsApp(t)
	pool := dbtest.Pool(t)
	ctx := context.Background()
	rec := &recebedorFalso{}
	s := NovoServico(NewPostgresStore(pool, cifra(t, 'c')), Email{}).WithRecebedor(rec)
	s.responderWA = func(context.Context, WhatsApp, string, string) error { return nil }
	if _, err := pool.Exec(ctx, `UPDATE users SET contato_whatsapp = '' WHERE contato_whatsapp IN ('+5566999991234', '+556699991234')`); err != nil {
		t.Fatal(err)
	}
	w, err := s.SalvarWhatsApp(ctx, WhatsApp{Ativo: true, Receber: true, Provedor: ProvedorMeta, MetaNumeroID: "123456",
		MetaModelo: "aviso", MetaToken: "tk", MetaAppSecret: "app-secret"}, false, false, "a")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	RegisterWebhookRoutes(r, NewHandlers(s, audit.NewWriter(pool), slog.New(slog.DiscardHandler)))
	chamar := func(method, path, corpo string, cabecalho map[string]string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(corpo))
		for k, v := range cabecalho {
			req.Header.Set(k, v)
		}
		r.ServeHTTP(rec, req)
		return rec
	}
	caminho := "/canais/whatsapp/webhook/" + w.WebhookToken
	if rr := chamar(http.MethodGet, caminho+"?hub.mode=subscribe&hub.verify_token="+w.WebhookToken+"&hub.challenge=42", "", nil); rr.Code != http.StatusOK ||
		rr.Body.String() != "42" || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("desafio: %d %s", rr.Code, rr.Body.String())
	}
	if rr := chamar(http.MethodGet, caminho+"?hub.mode=subscribe&hub.verify_token=x", "", nil); rr.Code != http.StatusUnauthorized {
		t.Fatalf("verificação errada: %d", rr.Code)
	}
	if rr := chamar(http.MethodGet, "/canais/whatsapp/webhook/errado", "", nil); rr.Code != http.StatusNotFound {
		t.Fatalf("token errado: %d", rr.Code)
	}
	if rr := chamar(http.MethodPost, caminho, corpoMeta, map[string]string{"X-Hub-Signature-256": assinar("app-secret", []byte(corpoMeta))}); rr.Code != http.StatusOK || len(rec.chamadas) != 0 {
		// O número do corpo não é de ninguém aqui: entra 200 e nada grava.
		t.Fatalf("mensagem: %d %s", rr.Code, rr.Body.String())
	}
	if rr := chamar(http.MethodPost, caminho, corpoMeta, map[string]string{"X-Hub-Signature-256": "sha256=00"}); rr.Code != http.StatusUnauthorized {
		t.Fatalf("assinatura errada: %d", rr.Code)
	}
	if rr := chamar(http.MethodPost, caminho, strings.Repeat("a", 1<<20+1), nil); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("corpo grande: %d", rr.Code)
	}
	falho := chi.NewRouter()
	RegisterWebhookRoutes(falho, NewHandlers(NovoServico(NewPostgresStore(dbtest.Fail{}, cifra(t, 'c')), Email{}), audit.NewWriter(pool), slog.New(slog.DiscardHandler)))
	rr := httptest.NewRecorder()
	falho.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, caminho, strings.NewReader("{}")))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("banco fora: %d", rr.Code)
	}
}
