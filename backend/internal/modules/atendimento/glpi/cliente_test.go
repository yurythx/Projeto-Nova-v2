package glpi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Sem token de usuário, a sessão abre com login e senha (Basic Auth, como
// a API do GLPI documenta); com token, as credenciais nem são enviadas.
func TestAbrirComLoginESenha(t *testing.T) {
	var recebido string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
