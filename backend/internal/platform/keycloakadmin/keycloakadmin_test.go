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
		case r.URL.Path == "/sso/admin/realms/x/users":
			var primeiro, max int
			_, _ = fmt.Sscan(r.URL.Query().Get("first"), &primeiro)
			_, _ = fmt.Sscan(r.URL.Query().Get("max"), &max)
			fim := min(len(pessoas), primeiro+max)
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

func TestUsuarios(t *testing.T) {
	ctx := context.Background()
	srv := keycloak(t, "")
	cfg := Config{IssuerURL: srv.URL + "/sso/realms/x/", ClientID: "projeto-nova", ClientSecret: "s"}
	c := Novo()
	c.Pagina = 2
	us, err := c.Usuarios(ctx, cfg)
	if err != nil || len(us) != 3 || us[0].Nome() != "Ana Souza" || us[1].Enabled || len(us[2].Grupos) != 2 || us[2].Grupos[0] != "Administrators" {
		t.Fatalf("usuários: %+v %v", us, err)
	}
	c.Pagina = 0
	if us, err := c.Usuarios(ctx, cfg); err != nil || len(us) != 3 {
		t.Fatalf("página padrão: %d %v", len(us), err)
	}

	for nome, cfgRuim := range map[string]Config{
		"sem segredo": {IssuerURL: cfg.IssuerURL, ClientID: "p"},
		"issuer":      {IssuerURL: "::", ClientID: "p", ClientSecret: "s"},
		"sem realm":   {IssuerURL: srv.URL + "/sso", ClientID: "p", ClientSecret: "s"},
		"realm vazio": {IssuerURL: srv.URL + "/sso/realms/", ClientID: "p", ClientSecret: "s"},
	} {
		if _, err := c.Usuarios(ctx, cfgRuim); !errors.Is(err, ErrNaoConfigurado) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := c.Usuarios(ctx, Config{IssuerURL: cfg.IssuerURL, ClientID: "p", ClientSecret: "errado"}); !errors.Is(err, ErrResposta) {
		t.Fatalf("segredo errado: %v", err)
	}
	for _, quebrar := range []string{"vazio", "/users", "groups", "json"} {
		if _, err := Novo().Usuarios(ctx, Config{IssuerURL: keycloak(t, quebrar).URL + "/sso/realms/x", ClientID: "p", ClientSecret: "s"}); !errors.Is(err, ErrResposta) {
			t.Errorf("%s: %v", quebrar, err)
		}
	}
	// Sem rede, e com o contexto cancelado.
	if _, err := Novo().Usuarios(ctx, Config{IssuerURL: "http://127.0.0.1:1/realms/x", ClientID: "p", ClientSecret: "s"}); err == nil {
		t.Fatal("sem rede")
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := Novo().Usuarios(cancelado, cfg); err == nil {
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
	if _, err := c.Usuarios(ctx, Config{IssuerURL: "http://sso/realms/x", ClientID: "p", ClientSecret: "s"}); err == nil {
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
