package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Corpo hostil usado em toda a superfície. Não é um exploit: só prova que
// o módulo não executa o texto, não busca o endereço e não devolve 500.
const corpoHostil = `{"q":"' OR 1=1--","nome":"<script>alert(1)</script>","url":"http://169.254.169.254/latest/meta-data/"}`

// TestTodoModuloCumpreOWASP é o padrão de segurança de todo plugin, no
// mesmo espírito do módulo-modelo: a suíte percorre as rotas que o próprio
// plugin registra e confere as premissas do OWASP Top 10 que cabem na
// superfície HTTP.
//
//	A01 / A07  rota autenticada sem sessão, ou com token inválido, responde 401
//	A02        a resposta não devolve segredo nem pilha
//	A03        texto de injeção não derruba o handler (nada de 500)
//	A05        os cabeçalhos de segurança da API saem em toda resposta
//	A08        erro é envelope JSON, não uma página HTML
//	A09        toda resposta carrega X-Request-ID para correlacionar o log
//
// A04 (módulo desligado some da superfície) está em
// TestEveryPluginSurfaceFollowsActivation. A06 é o govulncheck do CI.
// A10 (SSRF) é o netguard nas saídas HTTP da plataforma; aqui o corpo só
// não pode fazer o handler buscar o endereço e estourar.
func TestTodoModuloCumpreOWASP(t *testing.T) {
	h := newHarness(t)
	for _, p := range h.d.Kernel.Plugins() {
		rotas := pluginRoutes(t, p)
		if len(rotas) == 0 {
			continue
		}
		chave := p.Manifest().Key
		t.Run(chave, func(t *testing.T) {
			sub := &apiHarness{t: t, d: h.d, router: h.router}
			var autenticada *route
			for i := range rotas {
				r := rotas[i]
				rec := sub.do(r.method, concrete(r.path), "", corpoHostil)
				conferirSuperficie(t, chave, r, rec)
				if !r.publica {
					if rec.Code != http.StatusUnauthorized {
						t.Errorf("A01/A07 %s %s sem sessão: esperado 401, veio %d", r.method, r.path, rec.Code)
					}
					autenticada = &rotas[i]
				}
			}
			if autenticada == nil {
				return
			}
			rec := sub.do(autenticada.method, concrete(autenticada.path), "nao-e-jwt", corpoHostil)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("A07 %s %s com token inválido: esperado 401, veio %d", autenticada.method, autenticada.path, rec.Code)
			}
			conferirSuperficie(t, chave, *autenticada, rec)
		})
	}
}

func conferirSuperficie(t *testing.T, chave string, r route, rec *httptest.ResponseRecorder) {
	t.Helper()
	onde := chave + " " + r.method + " " + r.path
	// 503 é dependência declarada indisponível (Gov.br sem configuração).
	// 500 é handler que estourou com a entrada.
	if rec.Code == http.StatusInternalServerError {
		t.Errorf("A03 %s: entrada hostil respondeu 500", onde)
	}
	corpo := rec.Body.String()
	for _, vazamento := range []string{"password_hash", "BEGIN PRIVATE", "BEGIN RSA", "goroutine ", "panic:"} {
		if strings.Contains(corpo, vazamento) {
			t.Errorf("A02 %s: a resposta contém %q", onde, vazamento)
		}
	}
	if rec.Code >= 400 && corpo != "" && !strings.HasPrefix(strings.TrimSpace(corpo), "{") {
		t.Errorf("A08 %s: erro não é envelope JSON", onde)
	}
	cabecalhos := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for nome, quer := range cabecalhos {
		if rec.Header().Get(nome) != quer {
			t.Errorf("A05 %s: %s = %q, esperado %q", onde, nome, rec.Header().Get(nome), quer)
		}
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Errorf("A05 %s: Content-Security-Policy sem default-src 'none'", onde)
	}
	if !strings.Contains(rec.Header().Get("Strict-Transport-Security"), "preload") {
		t.Errorf("A05 %s: HSTS sem preload", onde)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Errorf("A09 %s: resposta sem X-Request-ID", onde)
	}
}
