package application

import "testing"

func TestMensagemAtribuida(t *testing.T) {
	if got := mensagemAtribuida("Ana Souza", "Instalar o link"); got != "Atribuído por Ana Souza. Instalar o link" {
		t.Fatalf("com autor: %q", got)
	}
	if got := mensagemAtribuida("  ", "Portal"); got != "Atribuído por alguém. Portal" {
		t.Fatalf("sem autor: %q", got)
	}
	if got := mensagemAtribuida("Carlos Silva", "  "); got != "Atribuído por Carlos Silva." {
		t.Fatalf("sem corpo: %q", got)
	}
}
