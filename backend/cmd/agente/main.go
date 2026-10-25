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
	api := strings.TrimRight(os.Getenv("NOVA_API"), "/")
	token := os.Getenv("NOVA_TOKEN")
	unidade := os.Getenv("NOVA_UNIDADE_ID")
	if api == "" || token == "" || unidade == "" {
		return fmt.Errorf("defina NOVA_API, NOVA_TOKEN e NOVA_UNIDADE_ID")
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/v1/ativos/inventario", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
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
