package govbr

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDesafioJSON(t *testing.T) {
	b, err := json.Marshal(Desafio{State: "s", Nonce: "n", Verifier: "v", CodeChallenge: "c", URL: "https://sso.staging.acesso.gov.br/authorize"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["state"] != "s" || m["verifier"] != "v" || m["code_challenge"] != "c" || m["url"] == "" || m["State"] != "" {
		t.Fatalf("json: %s", b)
	}
}

func TestSemCredencial(t *testing.T) {
	c := Config{}
	if c.Pronta() {
		t.Fatal("vazia não está pronta")
	}
	if _, err := c.Iniciar(); !errors.Is(err, ErrSemCredencial) {
		t.Fatalf("iniciar: %v", err)
	}
	if _, err := c.Trocar(nil, "code", "ver"); !errors.Is(err, ErrSemCredencial) {
		t.Fatalf("trocar: %v", err)
	}
}

func TestAleatorioFalha(t *testing.T) {
	antes := lerAleatorio
	t.Cleanup(func() { lerAleatorio = antes })
	c := Config{ClientID: "app", ClientSecret: "seg", RedirectURL: "https://nova/cb"}
	for falha := 1; falha <= 3; falha++ {
		chamadas := 0
		lerAleatorio = func(b []byte) (int, error) {
			chamadas++
			if chamadas == falha {
				return 0, errors.New("sem entropia")
			}
			for i := range b {
				b[i] = 1
			}
			return len(b), nil
		}
		if _, err := c.Iniciar(); err == nil || !strings.Contains(err.Error(), "entropia") {
			t.Fatalf("leitura %d: %v", falha, err)
		}
	}
}

func TestIniciarHomologacaoEProducao(t *testing.T) {
	c := Config{ClientID: "app", ClientSecret: "seg", RedirectURL: "https://nova.exemplo/cb", Ambiente: AmbienteHomologacao}
	d, err := c.Iniciar()
	if err != nil {
		t.Fatal(err)
	}
	if d.State == "" || d.Verifier == "" || d.CodeChallenge == "" || !strings.Contains(d.URL, "sso.staging.acesso.gov.br/authorize") || !strings.Contains(d.URL, "code_challenge_method=S256") {
		t.Fatalf("desafio: %+v", d)
	}
	c.Ambiente = AmbienteProducao
	d, err = c.Iniciar()
	if err != nil || !strings.Contains(d.URL, "https://sso.acesso.gov.br/authorize") {
		t.Fatalf("produção: %+v %v", d, err)
	}
	if _, err := c.Trocar(nil, " ", "v"); err == nil {
		t.Fatal("code vazio")
	}
}

func TestTrocar(t *testing.T) {
	var tokenHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenHits++
			if tokenHits == 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if tokenHits == 2 {
				_, _ = io.WriteString(w, `{}`)
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"abc"}`)
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer abc" {
				t.Errorf("bearer: %s", r.Header.Get("Authorization"))
			}
			if tokenHits == 3 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if tokenHits == 4 {
				_, _ = io.WriteString(w, `{}`)
				return
			}
			_, _ = io.WriteString(w, `{"sub":"1","name":"Ana","email":"ana@x.gov.br","cpf":"123"}`)
		default:
			t.Errorf("caminho %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	c := Config{ClientID: "app", ClientSecret: "seg", RedirectURL: "https://nova/cb"}
	// O host do Login Único é fixo. O teste cobre o cliente com um
	// transporte que manda tudo para o servidor local.
	cli := srv.Client()
	cli.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})
	if _, err := c.Trocar(cli, "code", "ver"); err == nil || !strings.Contains(err.Error(), "token 400") {
		t.Fatalf("token recusado: %v", err)
	}
	if _, err := c.Trocar(cli, "code", "ver"); err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("token vazio: %v", err)
	}
	if _, err := c.Trocar(cli, "code", "ver"); err == nil || !strings.Contains(err.Error(), "userinfo 403") {
		t.Fatalf("userinfo recusado: %v", err)
	}
	if _, err := c.Trocar(cli, "code", "ver"); err == nil || !strings.Contains(err.Error(), "sujeito") {
		t.Fatalf("userinfo vazio: %v", err)
	}
	id, err := c.Trocar(cli, "code", "ver")
	if err != nil || id.Nome != "Ana" || id.Email != "ana@x.gov.br" {
		t.Fatalf("identidade: %+v %v", id, err)
	}
}

func TestTrocarSemRede(t *testing.T) {
	c := Config{ClientID: "app", ClientSecret: "seg", RedirectURL: "https://nova/cb"}
	antesToken, antesInfo := enderecoToken, enderecoUserinfo
	t.Cleanup(func() { enderecoToken, enderecoUserinfo = antesToken, antesInfo })
	if _, err := c.Trocar(nil, " ", "ver"); err == nil {
		t.Fatal("code vazio")
	}
	enderecoToken = func(Config) string { return "http://[::1" }
	if _, err := c.Trocar(http.DefaultClient, "code", "ver"); err == nil {
		t.Fatal("token com endereço inválido")
	}
	enderecoToken = func(Config) string { return "http://127.0.0.1:1/token" }
	if _, err := c.Trocar(nil, "code", "ver"); err == nil {
		t.Fatal("token sem rede")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"tok"}`)
	}))
	t.Cleanup(srv.Close)
	enderecoToken = func(Config) string { return srv.URL }
	enderecoUserinfo = func(Config) string { return "http://[::1" }
	if _, err := c.Trocar(srv.Client(), "code", "ver"); err == nil {
		t.Fatal("userinfo com endereço inválido")
	}
	enderecoUserinfo = func(Config) string { return "http://127.0.0.1:1/userinfo" }
	if _, err := c.Trocar(srv.Client(), "code", "ver"); err == nil {
		t.Fatal("userinfo sem rede")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
