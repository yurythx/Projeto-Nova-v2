package domain

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEmitirCredencial(t *testing.T) {
	unidade := uuid.New()
	c, token, err := EmitirCredencial(" Balcão ", unidade)
	if err != nil || !strings.HasPrefix(token, PrefixoCredencial) || bytes.Equal(c.TokenHash, []byte(token)) {
		t.Fatalf("%v %q %#v", err, token, c)
	}
	if !bytes.Equal(c.TokenHash, HashCredencial(token)) || c.Nome != "Balcão" {
		t.Fatal("o hash não guarda o segredo")
	}
	if _, _, err := EmitirCredencial("", unidade); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, _, err := EmitirCredencial("pc", uuid.Nil); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
}
