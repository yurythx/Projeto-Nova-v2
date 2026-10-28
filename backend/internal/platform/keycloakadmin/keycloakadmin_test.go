package keycloakadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// keycloak responde como a API de administração: 3 pessoas em páginas de 2.
func keycloak(t *testing.T, quebrar string) *httptest.Server {
	t.Helper()
	pessoas := []map[string]any{
		{"id": "a", "username": "ana", "email": "ana@x.test", "firstName": "Ana", "lastName": "Souza", "enabled": true},
		{"id": "b", "username": "beto", "enabled": false},
		{"id": "c", "username": "caio", "firstName": "Caio", "enabled": true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if quebrar != "" && strings.Contains(r.URL.Path, quebrar) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch {
		case r.URL.Path == "/sso/realms/x/protocol/openid-connect/token":
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_secret") != "s" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if quebrar == "vazio" {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"tk"}`))
		case r.Header.Get("Authorization") != "Bearer tk":
			w.WriteHeader(http.StatusUnauthorized)
		case r.URL.Path == "/sso/admin/realms/x/users/count":
			if quebrar == "count" {
				_, _ = w.Write([]byte(`"três"`))
				return
			}
			_, _ = fmt.Fprint(w, len(pessoas))
		case r.URL.Path == "/sso/admin/realms/x/users" && quebrar == "lista":
			w.WriteHeader(http.StatusForbidden)
		case r.URL.Path == "/sso/admin/realms/x/users":
			var primeiro, max int
			_, _ = fmt.Sscan(r.URL.Query().Get("first"), &primeiro)
			_, _ = fmt.Sscan(r.URL.Query().Get("max"), &max)
			fim := min(len(pessoas), primeiro+max)
			if quebrar == "curta" && primeiro == 0 {
				fim = 1 // como o Keycloak federado: página curta no meio
			}
			_ = json.NewEncoder(w).Encode(pessoas[min(primeiro, fim):fim])
		case strings.HasSuffix(r.URL.Path, "/groups"):
			if quebrar == "json" {
				_, _ = w.Write([]byte(`{`))
				return
			}
			_, _ = w.Write([]byte(`[{"name":"Administrators"},{"name":"Domain Users"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// todos junta as páginas e conta quantas vieram.
func todos(ctx context.Context, c *Cliente, cfg Config) ([]Usuario, int, error) {
	var us []Usuario
	paginas := 0
	err := c.Usuarios(ctx, cfg, func(lote []Usuario) error {
		paginas++
		us = append(us, lote...)
		return nil
	})
	return us, paginas, err
}

func TestUsuarios(t *testing.T) {
	ctx := context.Background()
	srv := keycloak(t, "")
	cfg := Config{IssuerURL: srv.URL + "/sso/realms/x/", ClientID: "projeto-nova", ClientSecret: "s"}
	c := Novo()
	c.Pagina = 2
	us, paginas, err := todos(ctx, c, cfg)
	if err != nil || paginas != 2 || len(us) != 3 || us[0].Nome() != "Ana Souza" || us[1].Enabled || len(us[2].Grupos) != 2 || us[2].Grupos[0] != "Administrators" {
		t.Fatalf("usuários: %+v %d %v", us, paginas, err)
	}
	c.Pagina, c.Paralelo = 0, 0
	if us, paginas, err := todos(ctx, c, cfg); err != nil || paginas != 1 || len(us) != 3 {
		t.Fatalf("página padrão: %d %d %v", len(us), paginas, err)
	}
	// Página curta no meio não é o fim: segue até o total contado.
	if us, paginas, err := todos(ctx, &Cliente{HTTP: http.DefaultClient, Pagina: 2}, Config{IssuerURL: keycloak(t, "curta").URL + "/sso/realms/x", ClientID: "p", ClientSecret: "s"}); err != nil || paginas != 2 || len(us) != 2 || us[1].ID != "c" {
		t.Fatalf("página curta: %+v %d %v", us, paginas, err)
	}
	// O erro de quem recebe a página para a leitura.
	parar := errors.New("parar")
	if err := c.Usuarios(ctx, cfg, func([]Usuario) error { return parar }); !errors.Is(err, parar) {
		t.Fatalf("lote com erro: %v", err)
	}

	for nome, cfgRuim := range map[string]Config{
		"sem segredo": {IssuerURL: cfg.IssuerURL, ClientID: "p"},
		"issuer":      {IssuerURL: "::", ClientID: "p", ClientSecret: "s"},
		"sem realm":   {IssuerURL: srv.URL + "/sso", ClientID: "p", ClientSecret: "s"},
		"realm vazio": {IssuerURL: srv.URL + "/sso/realms/", ClientID: "p", ClientSecret: "s"},
	} {
		if _, _, err := todos(ctx, c, cfgRuim); !errors.Is(err, ErrNaoConfigurado) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, _, err := todos(ctx, c, Config{IssuerURL: cfg.IssuerURL, ClientID: "p", ClientSecret: "errado"}); !errors.Is(err, ErrResposta) {
		t.Fatalf("segredo errado: %v", err)
	}
	for _, quebrar := range []string{"vazio", "/users", "count", "lista", "groups", "json"} {
		if _, _, err := todos(ctx, Novo(), Config{IssuerURL: keycloak(t, quebrar).URL + "/sso/realms/x", ClientID: "p", ClientSecret: "s"}); !errors.Is(err, ErrResposta) {
			t.Errorf("%s: %v", quebrar, err)
		}
	}
	// Sem rede, e com o contexto cancelado.
	if _, _, err := todos(ctx, Novo(), Config{IssuerURL: "http://127.0.0.1:1/realms/x", ClientID: "p", ClientSecret: "s"}); err == nil {
		t.Fatal("sem rede")
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, _, err := todos(cancelado, Novo(), cfg); err == nil {
		t.Fatal("contexto cancelado")
	}
}

// corpoQuebrado falha no meio da leitura da resposta.
type corpoQuebrado struct{}

func (corpoQuebrado) Read([]byte) (int, error) { return 0, errors.New("conexão caiu") }
func (corpoQuebrado) Close() error             { return nil }

type transporte func(*http.Request) (*http.Response, error)

func (f transporte) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFalhasDeTransporte(t *testing.T) {
	ctx := context.Background()
	c := &Cliente{HTTP: &http.Client{Transport: transporte(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: corpoQuebrado{}}, nil
	})}}
	if _, _, err := todos(ctx, c, Config{IssuerURL: "http://sso/realms/x", ClientID: "p", ClientSecret: "s"}); err == nil {
		t.Fatal("corpo que falha")
	}
	// O endereço que nem vira requisição.
	if err := c.get(ctx, "tk", "http://[::1", nil); err == nil {
		t.Fatal("endereço inválido")
	}
	if _, err := c.token(ctx, "http://[::1", Config{}); err == nil {
		t.Fatal("token com endereço inválido")
	}
}
