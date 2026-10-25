package domain

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestMetadadoNaoSerializaOEnvelope(t *testing.T) {
	unidade := uuid.New()
	item := Selado{
		Segredo: Segredo{ID: uuid.New(), Nome: "Correio", Usuario: "ana", UnidadeID: &unidade, Versao: 1},
		Dek:     "dek-embrulhada",
		Cifra:   "texto-cifrado",
	}
	corpo, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(corpo), "dek-embrulhada") || strings.Contains(string(corpo), "texto-cifrado") || strings.Contains(string(corpo), "senha") {
		t.Fatalf("vazou: %s", corpo)
	}
	if _, _, _, _, err := Validar("", "ana", "x", "", true); err == nil {
		t.Fatal("aceitou nome vazio")
	}
	if bytesDe(AAD(item.ID, 1, &unidade)) == bytesDe(AAD(item.ID, 2, &unidade)) {
		t.Fatal("a versão não entrou no contexto")
	}
}

func bytesDe(b []byte) string { return string(b) }
