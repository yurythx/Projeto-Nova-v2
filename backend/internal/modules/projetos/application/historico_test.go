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
	if got := textoDoHistorico("projetos.projeto.reopened", nil, json.RawMessage(`{`)); got != "Reaberto" {
		t.Fatalf("reaberto sem motivo: %s", got)
	}
	if got := textoDoHistorico("projetos.projeto.reopened", nil, json.RawMessage(`{"reabertura":1}`)); got != "Reaberto" {
		t.Fatalf("reabertura ilegível: %s", got)
	}
	if got := textoDoHistorico("projetos.projeto.cancelled", nil, json.RawMessage(`{}`)); got != "Cancelado" {
		t.Fatalf("cancelado sem motivo: %s", got)
	}
	if got := textoDoHistorico(EventProjetoTransitioned, nil, json.RawMessage(`{"estado":1}`)); got != "Estado alterado" {
		t.Fatalf("estado ilegível: %s", got)
	}
	if got := textoDoHistorico(EventProjetoTransitioned, nil, json.RawMessage(`{`)); got != "Estado alterado" {
		t.Fatalf("transição ilegível: %s", got)
	}
	if got := textoDoHistorico(EventProjetoUpdated, antes, antes); got != "Ficha alterada" {
		t.Fatalf("ficha: %s", got)
	}
	if got := textoDoHistorico("outra", nil, nil); got != "Registro do projeto" {
		t.Fatalf("outro: %s", got)
	}
	for _, estado := range []string{"planejado", "suspenso", "cancelado", "coluna"} {
		if rotuloEstado(estado) == "" {
			t.Fatal(estado)
		}
	}
}
