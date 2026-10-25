package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAplicarRelato(t *testing.T) {
	unidade := uuid.New()
	relato := Relato{Identificador: " PC-1 ", Nome: "Balcão", Tipo: "hardware", Descricao: "linux amd64", UnidadeID: unidade}
	criado, mudou, err := AplicarRelato(nil, relato)
	if err != nil || !mudou || criado.Origem != OrigemAgente || criado.Estado != "em_uso" || criado.Identificador != "PC-1" {
		t.Fatalf("%v %v %#v", err, mudou, criado)
	}
	manual := criado
	manual.Origem = OrigemManual
	if _, _, err := AplicarRelato(&manual, relato); !errors.Is(err, ErrCadastroManual) {
		t.Fatal(err)
	}
	outra := uuid.New()
	relato.UnidadeID = outra
	if _, _, err := AplicarRelato(&criado, relato); !errors.Is(err, ErrForaDoEscopo) {
		t.Fatal(err)
	}
	relato.UnidadeID = unidade
	criado.Estado = "desativado"
	relato.Descricao = "linux amd64 · disco"
	proximo, mudou, err := AplicarRelato(&criado, relato)
	if err != nil || proximo.Estado != "desativado" || !mudou {
		t.Fatalf("%v %v %s", err, mudou, proximo.Estado)
	}
	igual, mudou, err := AplicarRelato(&proximo, relato)
	if err != nil || mudou || !igual.Igual(proximo) {
		t.Fatal("repetir o mesmo relato não regrava")
	}
	if _, _, err := AplicarRelato(nil, Relato{Identificador: "x", Tipo: "nave", UnidadeID: unidade}); !errors.Is(err, ErrTipoInvalido) {
		t.Fatal(err)
	}
}
