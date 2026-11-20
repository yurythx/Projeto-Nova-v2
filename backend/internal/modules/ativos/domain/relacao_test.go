package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidarRelacao(t *testing.T) {
	if err := ValidarRelacao(uuid.Nil, uuid.New(), "depende"); err == nil || err.Error() == "" {
		t.Fatal(err)
	}
	id := uuid.New()
	if err := ValidarRelacao(id, id, "depende"); err == nil {
		t.Fatal("o mesmo ativo")
	}
	if err := ValidarRelacao(uuid.New(), uuid.New(), "amigo"); err == nil {
		t.Fatal("tipo")
	}
	if err := ValidarRelacao(uuid.New(), uuid.New(), "depende"); err != nil {
		t.Fatal(err)
	}
}
