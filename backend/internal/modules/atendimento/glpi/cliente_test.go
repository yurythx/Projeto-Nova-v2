package glpi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Sem token de usuário, a sessão abre com login e senha (Basic Auth, como
// a API do GLPI documenta); com token, as credenciais nem são enviadas.
func TestAbrirComLoginESenha(t *testing.T) {
	var recebido string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/getMyProfiles" {
			_, _ = w.Write([]byte(`{"myprofiles":[]}`))
			return
		}
		login, senha, basic := r.BasicAuth()
		switch {
		case basic && login == "svc" && senha == "p":
			recebido = "basic"
		case r.Header.Get("Authorization") == "user_token u":
			recebido = "token"
		default:
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"session_token":"s"}`))
	}))
	t.Cleanup(srv.Close)

	c := NovoCliente(srv.URL, "", "")
	c.Login, c.Senha = "svc", "p"
	if err := c.Abrir(context.Background()); err != nil || recebido != "basic" {
		t.Fatalf("credencial: %q %v", recebido, err)
	}
	c = NovoCliente(srv.URL, "", "u")
	c.Login, c.Senha = "svc", "p"
	if err := c.Abrir(context.Background()); err != nil || recebido != "token" {
		t.Fatalf("o token vale mais: %q %v", recebido, err)
	}
}

// A sessão abre no perfil padrão da conta, que pode ver menos: Abrir troca
// para o pedido ou, sem pedido, para o Super-Admin, se a conta o tiver.
func TestAbrirTrocaOPerfil(t *testing.T) {
	perfis := `[{"id":3,"name":"Admin"},{"id":4,"name":"Super-Admin"}]`
	trocou, trocaFalha := 0, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/initSession":
			_, _ = w.Write([]byte(`{"session_token":"s"}`))
		case "/getMyProfiles":
			if perfis == "" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"myprofiles":` + perfis + `}`))
		case "/changeActiveProfile":
			var corpo struct {
				ID int `json:"profiles_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&corpo)
			if trocaFalha || r.Method != http.MethodPost {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			trocou = corpo.ID
		}
	}))
	t.Cleanup(srv.Close)
	abrir := func(perfil string) (*Cliente, error) {
		c := NovoCliente(srv.URL, "", "u")
		c.Perfil = perfil
		return c, c.Abrir(context.Background())
	}

	if c, err := abrir(""); err != nil || trocou != 4 || c.PerfilAtivo != "Super-Admin" {
		t.Fatalf("padrão: %d %q %v", trocou, c.PerfilAtivo, err)
	}
	if c, err := abrir("admin"); err != nil || trocou != 3 || c.PerfilAtivo != "Admin" {
		t.Fatalf("pedido, sem caixa: %d %q %v", trocou, c.PerfilAtivo, err)
	}
	if _, err := abrir("Técnico"); !errors.Is(err, ErrGLPI) {
		t.Fatalf("perfil que a conta não tem: %v", err)
	}
	perfis, trocou = `[{"id":3,"name":"Admin"}]`, 0
	if c, err := abrir(""); err != nil || trocou != 0 || c.PerfilAtivo != "" {
		t.Fatalf("sem Super-Admin, fica no da conta: %d %q %v", trocou, c.PerfilAtivo, err)
	}
	trocaFalha = true
	if _, err := abrir("Admin"); !errors.Is(err, ErrGLPI) {
		t.Fatalf("troca recusada: %v", err)
	}
	perfis = ""
	if _, err := abrir(""); !errors.Is(err, ErrGLPI) {
		t.Fatalf("lista de perfis com falha: %v", err)
	}
}
