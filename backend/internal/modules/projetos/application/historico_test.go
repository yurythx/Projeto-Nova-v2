package application

import (
	"encoding/json"
	"testing"
)

func TestTextoDoHistorico(t *testing.T) {
	depois, _ := json.Marshal(map[string]any{
		"estado":       "em_andamento",
		"gerente_nome": "Ana Souza",
		"reabertura":   map[string]string{"motivo": "o escopo voltou"},
		"cancelamento": map[string]string{"motivo": "a secretaria desistiu"},
	})
	antes, _ := json.Marshal(map[string]any{"gerente_nome": "Carlos Silva", "estado": "planejado"})
	if got := textoDoHistorico(EventProjetoTransitioned, nil, depois); got != "Passou para em andamento" {
		t.Fatalf("transição: %s", got)
	}
	if got := textoDoHistorico("projetos.projeto.reopened", nil, depois); got != "Reaberto. Motivo: o escopo voltou" {
		t.Fatalf("reabertura: %s", got)
	}
	if got := textoDoHistorico("projetos.projeto.cancelled", nil, depois); got != "Cancelado. Motivo: a secretaria desistiu" {
		t.Fatalf("cancelamento: %s", got)
	}
	if got := textoDoHistorico(EventProjetoUpdated, antes, depois); got != "Responsável passou a ser Ana Souza" {
		t.Fatalf("responsável: %s", got)
	}
	if got := textoDoHistorico(EventProjetoCreated, nil, nil); got != "Projeto aberto" {
		t.Fatalf("criação: %s", got)
	}
}
