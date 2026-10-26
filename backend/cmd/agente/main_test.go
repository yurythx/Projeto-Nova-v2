package main

import "testing"

func TestEnderecoDaAPISoComHTTPSForaDaMaquina(t *testing.T) {
	aceitos := map[string]string{
		"https://api.projetonova.cloud/": "https://api.projetonova.cloud",
		" https://api.exemplo:8443":      "https://api.exemplo:8443",
		"http://localhost:8012":          "http://localhost:8012",
		"http://127.0.0.1:8010/":         "http://127.0.0.1:8010",
	}
	for bruto, quer := range aceitos {
		if got, err := enderecoDaAPI(bruto); err != nil || got != quer {
			t.Errorf("%q: %q, %v", bruto, got, err)
		}
	}
	for _, bruto := range []string{
		"http://192.168.1.70:8010", // o token passaria aberto pela rede
		"ftp://api.exemplo",
		"api.exemplo",
		"https://",
		"https://u:s@api.exemplo",
		"https://api.exemplo/?x=1",
	} {
		if _, err := enderecoDaAPI(bruto); err == nil {
			t.Errorf("%q deveria ser recusado", bruto)
		}
	}
}
