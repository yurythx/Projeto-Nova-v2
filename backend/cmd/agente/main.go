// Command agente informa o patrimônio desta máquina: nome, sistema,
// processadores, memória e discos locais. Não varre a rede, não lê
// compartilhamento e não lê arquivo de outra pessoa. O token é o mesmo
// Bearer da API, de uma conta com ativos:reportar na unidade.
//
//	NOVA_API=https://exemplo NOVA_TOKEN=... NOVA_UNIDADE_ID=... agente
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "agente:", err)
		os.Exit(1)
	}
}

func run() error {
	token := os.Getenv("NOVA_TOKEN")
	unidade := os.Getenv("NOVA_UNIDADE_ID")
	if os.Getenv("NOVA_API") == "" || token == "" || unidade == "" {
		return fmt.Errorf("defina NOVA_API, NOVA_TOKEN e NOVA_UNIDADE_ID")
	}
	api, err := enderecoDaAPI(os.Getenv("NOVA_API"))
	if err != nil {
		return err
	}
	nome, err := os.Hostname()
	if err != nil || strings.TrimSpace(nome) == "" {
		return fmt.Errorf("sem nome de máquina")
	}
	id := os.Getenv("NOVA_AGENTE_ID")
	if id == "" {
		id = nome
	}
	maq := coletar()
	if maq.Processadores < 1 {
		maq.Processadores = 1
	}
	inventario := map[string]any{
		"sistema":       maq.Sistema,
		"processadores": maq.Processadores,
		"memoria_bytes": maq.MemoriaBytes,
		"discos":        maq.Discos,
	}
	if len(maq.Programas) > 0 {
		inventario["programas"] = maq.Programas
		if maq.ProgramasTruncado {
			inventario["programas_truncado"] = true
		}
	}
	corpo, err := json.Marshal(map[string]any{
		"identificador": id,
		"nome":          nome,
		"tipo":          "hardware",
		"unidade_id":    unidade,
		"inventario":    inventario,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// #nosec G704 -- o destino é a API que quem instala o agente configura,
	// conferida por enderecoDaAPI.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/v1/ativos/inventario", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req) // #nosec G704 -- idem
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("a API respondeu %s", res.Status)
	}
	fmt.Println("patrimônio informado:", id)
	return nil
}

// enderecoDaAPI confere NOVA_API. O token vai no cabeçalho: fora da
// própria máquina, só por HTTPS.
func enderecoDaAPI(bruto string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("NOVA_API precisa ser um endereço como https://api.exemplo")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", fmt.Errorf("NOVA_API precisa usar https (http só para localhost)")
	}
	return strings.TrimRight(u.String(), "/"), nil
}
