package domain

import (
	"errors"
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

func TestPrazoEVigencia(t *testing.T) {
	zero, oito := 0, 8
	if ValidarPrazo(nil) != nil || ValidarPrazo(&oito) != nil {
		t.Fatal("válidos")
	}
	var invalida EntradaInvalida
	if !errors.As(ValidarPrazo(&zero), &invalida) {
		t.Fatal("prazo zero")
	}
	dia := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	c := Contrato{Inicio: dia, Fim: dia.AddDate(0, 0, 2)}
	for nome, k := range map[string]struct {
		em     time.Time
		espera bool
	}{
		"antes":           {dia.Add(-time.Hour), false},
		"no primeiro dia": {dia.Add(9 * time.Hour), true},
		"no último dia":   {dia.AddDate(0, 0, 2).Add(23 * time.Hour), true},
		"depois":          {dia.AddDate(0, 0, 3), false},
	} {
		if c.Vigente(k.em) != k.espera {
			t.Errorf("%s", nome)
		}
	}
}
