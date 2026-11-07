package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestAtualizacaoEMes(t *testing.T) {
	if _, err := ValidarAtualizacao("  "); !errors.Is(err, ErrAtualizacaoInvalida) {
		t.Fatalf("vazia: %v", err)
	}
	if _, err := ValidarAtualizacao(strings.Repeat("a", AtualizacaoMax+1)); !errors.Is(err, ErrAtualizacaoInvalida) {
		t.Fatalf("longa: %v", err)
	}
	texto, err := ValidarAtualizacao("  Reunião confirmada.  ")
	if err != nil || texto != "Reunião confirmada." {
		t.Fatalf("texto: %q %v", texto, err)
	}
	if err := MesValido("2026-11"); err != nil {
		t.Fatalf("mês: %v", err)
	}
	if err := MesValido("2026-13"); !errors.Is(err, ErrMesInvalido) {
		t.Fatalf("mês inválido: %v", err)
	}
	if err := MesValido(""); !errors.Is(err, ErrMesInvalido) {
		t.Fatalf("mês vazio: %v", err)
	}
}
