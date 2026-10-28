package keycloakadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// keycloak responde como a API de administração: 3 pessoas em páginas de 2.
func keycloak(t *testing.T, quebrar string) *httptest.Server {
	t.Helper()
	var contou atomic.Int32
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
			if quebrar == "emendada-e-curta" && contou.Add(1) > 1 {
				_, _ = w.Write([]byte(`2`))
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
			// Como o Keycloak federado: a página inteira vem com o JSON
			// emendado, e uma pessoa por vez vem certa.
			// A lista emendada que, uma a uma, passa do fim real: 400 depois
			// da última, e a contagem nova é 2.
			if quebrar == "emendada-e-curta" && max == 1 && primeiro >= 2 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if quebrar == "emendada-e-negada" && max == 1 && primeiro == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if strings.HasPrefix(quebrar, "emendada") && max > 1 {
				_, _ = w.Write([]byte(`[{"id":"a","username":"ana"{"id":"b"}]`))
				return
			}
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

// rapido encurta a espera entre as tentativas.
func rapido(t *testing.T) {
	t.Helper()
	antes := Espera
	Espera = time.Millisecond
	t.Cleanup(func() { Espera = antes })
}

func TestUsuarios(t *testing.T) {
	rapido(t)
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
	// A página emendada é lida pessoa por pessoa, até o fim da lista.
	if us, _, err := todos(ctx, &Cliente{HTTP: http.DefaultClient, Pagina: 5}, Config{IssuerURL: keycloak(t, "emendada").URL + "/sso/realms/x",
		ClientID: "p", ClientSecret: "s"}); err != nil || len(us) != 3 || us[2].ID != "c" {
		t.Fatalf("página emendada: %+v %v", us, err)
	}
	if us, _, err := todos(ctx, &Cliente{HTTP: http.DefaultClient, Pagina: 5}, Config{IssuerURL: keycloak(t, "emendada-e-curta").URL + "/sso/realms/x",
		ClientID: "p", ClientSecret: "s"}); err != nil || len(us) != 2 {
		t.Fatalf("uma a uma até o fim real: %+v %v", us, err)
	}
	if _, _, err := todos(ctx, &Cliente{HTTP: http.DefaultClient, Pagina: 5}, Config{IssuerURL: keycloak(t, "emendada-e-negada").URL + "/sso/realms/x",
		ClientID: "p", ClientSecret: "s"}); !errors.Is(err, ErrResposta) {
		t.Fatalf("pessoa negada na leitura uma a uma: %v", err)
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

// Uma página corrompida ou um 500 passageiro se repetem; recusa, não.
func TestTentativas(t *testing.T) {
	rapido(t)
	ctx := context.Background()
	var chamadas, falhas atomic.Int32
	falhas.Store(2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas.Add(1)
		switch {
		case r.URL.Path == "/negado":
			w.WriteHeader(http.StatusForbidden)
		case falhas.Add(-1) >= 0:
			if falhas.Load()%2 == 0 {
				_, _ = w.Write([]byte(`{"a":1{`))
				return
			}
			w.WriteHeader(http.StatusBadGateway)
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(srv.Close)
	c := Novo()
	var out []Usuario
	if err := c.get(ctx, "tk", srv.URL+"/users", &out); err != nil || chamadas.Load() != 3 {
		t.Fatalf("duas falhas e o acerto: %v %d", err, chamadas.Load())
	}
	chamadas.Store(0)
	falhas.Store(5)
	if err := c.get(ctx, "tk", srv.URL+"/users", &out); !errors.Is(err, ErrResposta) || chamadas.Load() != 3 {
		t.Fatalf("desiste na terceira: %v %d", err, chamadas.Load())
	}
	chamadas.Store(0)
	if err := c.get(ctx, "tk", srv.URL+"/negado", &out); !errors.Is(err, ErrResposta) || chamadas.Load() != 1 {
		t.Fatalf("recusa não se repete: %v %d", err, chamadas.Load())
	}
	// Cancelado na espera, para na hora.
	Espera = time.Hour
	falhas.Store(5)
	cancelado, cancelar := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelar()
	if err := c.get(cancelado, "tk", srv.URL+"/users", &out); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelado: %v", err)
	}
}

// O fim que a contagem nova confirma é normal; se ela ainda diz que há
// mais, o erro volta.
func TestFimDaListaQueOscila(t *testing.T) {
	rapido(t)
	ctx := context.Background()
	var contagens atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			_, _ = w.Write([]byte(`{"access_token":"tk"}`))
		case strings.HasSuffix(r.URL.Path, "/users/count"):
			// 4 na primeira contagem; depois, 2 ou 9 conforme o teste.
			if contagens.Add(1) == 1 {
				_, _ = w.Write([]byte(`4`))
				return
			}
			_, _ = w.Write([]byte(r.Header.Get("X-Depois")))
		case r.URL.Query().Get("first") == "0":
			_, _ = w.Write([]byte(`[{"id":"a"},{"id":"b"}]`))
		case strings.HasSuffix(r.URL.Path, "/groups"):
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := Config{IssuerURL: srv.URL + "/realms/x", ClientID: "p", ClientSecret: "s"}
	for depois, querErro := range map[string]bool{"2": false, "9": true, "x": true} {
		contagens.Store(0)
		c := &Cliente{HTTP: &http.Client{Transport: comCabecalho{"X-Depois", depois}}, Pagina: 2}
		us, _, err := todos(ctx, c, cfg)
		if (err != nil) != querErro || len(us) != 2 {
			t.Errorf("contagem depois %s: %d %v", depois, len(us), err)
		}
	}
}

// comCabecalho põe um cabeçalho em todo pedido (o servidor de teste o lê).
type comCabecalho struct{ nome, valor string }

func (c comCabecalho) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(c.nome, c.valor)
	return http.DefaultTransport.RoundTrip(r)
}

func TestAtivoConfirmaNaFonte(t *testing.T) {
	rapido(t)
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			_, _ = w.Write([]byte(`{"access_token":"tk"}`))
		case r.URL.Path == "/admin/realms/x/users/ana":
			_, _ = w.Write([]byte(`{"id":"ana","enabled":true}`))
		case r.URL.Path == "/admin/realms/x/users/beto":
			_, _ = w.Write([]byte(`{"id":"beto","enabled":false}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := Config{IssuerURL: srv.URL + "/realms/x", ClientID: "p", ClientSecret: "s"}
	c := Novo()
	if ok, err := c.Ativo(ctx, cfg, "ana"); err != nil || !ok {
		t.Fatalf("ana: %v %v", ok, err)
	}
	if ok, err := c.Ativo(ctx, cfg, "beto"); err != nil || ok {
		t.Fatalf("beto: %v %v", ok, err)
	}
	if _, err := c.Ativo(ctx, cfg, "sumiu"); !errors.Is(err, ErrResposta) {
		t.Fatalf("inexistente: %v", err)
	}
	if _, err := c.Ativo(ctx, Config{}, "ana"); !errors.Is(err, ErrNaoConfigurado) {
		t.Fatalf("sem configuração: %v", err)
	}
	if _, err := c.Ativo(ctx, Config{IssuerURL: srv.URL + "/sem-realm", ClientID: "p", ClientSecret: "s"}, "ana"); !errors.Is(err, ErrNaoConfigurado) {
		t.Fatalf("issuer sem realm: %v", err)
	}
	if _, err := c.Ativo(ctx, Config{IssuerURL: "http://127.0.0.1:1/realms/x", ClientID: "p", ClientSecret: "s"}, "ana"); err == nil {
		t.Fatal("sem rede")
	}
}
