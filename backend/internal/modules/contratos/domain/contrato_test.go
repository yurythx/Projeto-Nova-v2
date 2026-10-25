package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCadastro(t *testing.T) {
	if _, err := NovoFornecedor("  ", "", "", nil); err == nil {
		t.Fatal("nome vazio")
	}
	if _, err := NovoFornecedor("Casa", "sem-arroba", "", nil); err == nil {
		t.Fatal("e-mail")
	}
	f, err := NovoFornecedor("Casa", "a@b.co", "11", nil)
	if err != nil || f.Nome != "Casa" {
		t.Fatal(err)
	}
	inicio := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	fim := inicio.Add(-time.Hour)
	if _, err := NovoContrato("1", "Link", uuid.New(), inicio, fim, 0, nil); err == nil {
		t.Fatal("vigência")
	}
	if _, err := NovoContrato("1", "Link", uuid.New(), inicio, inicio, -1, nil); err == nil {
		t.Fatal("valor")
	}
	c, err := NovoContrato("1", "Link", f.ID, inicio, inicio, 10, nil)
	if err != nil || c.ValorCentavos != 10 {
		t.Fatal(err)
	}
	if _, err := NovaLicenca("Editor", uuid.New(), nil, 0, nil); err == nil {
		t.Fatal("quantidade")
	}
	l, err := NovaLicenca("Editor", f.ID, &inicio, 3, nil)
	if err != nil || l.Quantidade != 3 {
		t.Fatal(err)
	}
	if _, err := NovoFornecedor(strings.Repeat("a", 161), "", "", nil); err == nil {
		t.Fatal("nome longo")
	}
}
