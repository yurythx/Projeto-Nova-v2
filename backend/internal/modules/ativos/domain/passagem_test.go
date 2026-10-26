package domain

import (
	"slices"
	"testing"
)

func TestPassagemDeAtivo(t *testing.T) {
	const coluna = "c01234567"
	if !PassagemDeAtivo("", "em_uso", nil) || !PassagemDeAtivo("planejado", "desativado", nil) {
		t.Fatal("os estados de fábrica continuam livres")
	}
	if PassagemDeAtivo("", coluna, nil) {
		t.Fatal("criar não aceita coluna nova sem passagem")
	}
	if !PassagemDeAtivo("", coluna, map[string][]string{"em_uso": {coluna}}) {
		t.Fatal("criar aceita quando alguma passagem aponta para a coluna")
	}
	if PassagemDeAtivo("em_uso", coluna, map[string][]string{"planejado": {coluna}}) {
		t.Fatal("atualizar não usa passagem de outra coluna")
	}
	if !PassagemDeAtivo("em_uso", coluna, map[string][]string{"em_uso": {coluna}}) {
		t.Fatal("atualizar segue a passagem do estado atual")
	}
	livres := EstadosLivres(map[string][]string{"em_uso": {coluna, "planejado"}})
	if !slices.Contains(livres, coluna) || slices.Contains(livres, "planejado") {
		t.Fatalf("livres = %v", livres)
	}
}
