package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func ambiente(v map[string]string) func(string) string { return func(k string) string { return v[k] } }

func TestLer(t *testing.T) {
	env := ambiente(map[string]string{"GLPI_URL": "https://glpi/apirest.php", "GLPI_USER_TOKEN": "u", "GLPI_APP_TOKEN": "a"})
	op, err := ler([]string{"--unidade", "6d6f6e69-746f-4072-a000-6d656e746f00", "--gravar", "--pagina", "50", "--mapa", "m.csv"}, env)
	if err != nil || !op.gravar || op.unidade == nil || op.pagina != 50 || op.fuso.String() != "America/Sao_Paulo" || op.appToken != "a" || op.mapa != "m.csv" {
		t.Fatalf("opções: %+v %v", op, err)
	}
	if op, err := ler(nil, env); err != nil || op.gravar || op.unidade != nil {
		t.Fatalf("padrão simula: %+v %v", op, err)
	}
	credencial := ambiente(map[string]string{"GLPI_URL": "https://glpi/apirest.php", "GLPI_LOGIN": "svc", "GLPI_PASSWORD": "p"})
	if op, err := ler(nil, credencial); err != nil || op.login != "svc" || op.senha != "p" {
		t.Fatalf("login e senha no lugar do token: %+v %v", op, err)
	}
	if _, err := ler(nil, ambiente(map[string]string{"GLPI_URL": "https://glpi/apirest.php", "GLPI_LOGIN": "svc"})); err == nil {
		t.Fatal("login sem senha")
	}
	for nome, k := range map[string]struct {
		args []string
		env  func(string) string
	}{
		"sem URL":         {nil, ambiente(nil)},
		"unidade errada":  {[]string{"--unidade", "x"}, env},
		"fuso errado":     {[]string{"--fuso", "Lugar/Nenhum"}, env},
		"opção que falta": {[]string{"--nada"}, env},
	} {
		if _, err := ler(k.args, k.env); err == nil {
			t.Errorf("%s: aceitou", nome)
		}
	}
}

func TestImportarSimulaERecusaSessao(t *testing.T) {
	pool := dbtest.Pool(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/initSession") && r.Header.Get("Authorization") == "user_token u":
			_, _ = w.Write([]byte(`{"session_token":"s"}`))
		case strings.HasSuffix(r.URL.Path, "/initSession"):
			w.WriteHeader(http.StatusUnauthorized)
		case strings.HasSuffix(r.URL.Path, "/Ticket"):
			w.Header().Set("Content-Range", "0-0/0")
			_, _ = w.Write([]byte(`[]`))
		case strings.HasSuffix(r.URL.Path, "/getMyProfiles"):
			_, _ = w.Write([]byte(`{"myprofiles":[]}`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(srv.Close)
	op, err := ler(nil, ambiente(map[string]string{"GLPI_URL": srv.URL, "GLPI_USER_TOKEN": "u"}))
	if err != nil {
		t.Fatal(err)
	}
	var saida bytes.Buffer
	if err := importar(context.Background(), op, pool, &saida); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saida.String(), `"lidos": 0`) || !strings.Contains(saida.String(), "Simulação") {
		t.Fatalf("relatório: %s", saida.String())
	}
	// Com a tabela: a unidade que não existe aparece no relatório.
	tabela := filepath.Join(t.TempDir(), "mapa.csv")
	if err := os.WriteFile(tabela, []byte("glpi_secretaria;glpi_categoria_id;unidade;oferta;tipo;prioridade\nSMS;*;Unidade Nenhuma;Geral;incidente;media\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	op.mapa = tabela
	saida.Reset()
	if err := importar(context.Background(), op, pool, &saida); err != nil || !strings.Contains(saida.String(), "Unidade Nenhuma") {
		t.Fatalf("mapa: %v %s", err, saida.String())
	}
	op.mapa = filepath.Join(t.TempDir(), "nao-existe.csv")
	if err := importar(context.Background(), op, pool, &saida); err == nil || !strings.Contains(err.Error(), "--mapa") {
		t.Fatalf("mapa que não existe: %v", err)
	}
	op.mapa = ""
	op.userToken = "errado"
	if err := importar(context.Background(), op, pool, &saida); err == nil || !strings.Contains(err.Error(), "sessão") {
		t.Fatalf("sessão recusada: %v", err)
	}
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_NAME", "")
	if err := rodar(context.Background(), op, &saida); err == nil {
		t.Fatal("sem banco configurado")
	}
}
