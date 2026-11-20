package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidarInventario(t *testing.T) {
	if got, err := ValidarInventario(nil); err != nil || got != nil {
		t.Fatal(err)
	}
	inv, err := ValidarInventario(&Inventario{
		Sistema: "linux amd64", Processadores: 2,
		Programas: []Programa{{Nome: "zsh", Versao: "1"}, {Nome: "bash", Versao: "5"}},
	})
	if err != nil || inv.Programas[0].Nome != "bash" || !strings.Contains(inv.Resumo(), "2 programas") {
		t.Fatalf("programas: %v %#v", err, inv)
	}
	inv, err = ValidarInventario(&Inventario{
		Sistema: " Windows 11 amd64 ", Processadores: 8, MemoriaBytes: 8 << 30,
		Discos: []Disco{{Nome: "D:", Bytes: 20}, {Nome: "C:", Bytes: 10}},
	})
	if err != nil || inv.Discos[0].Nome != "C:" || !strings.Contains(inv.Resumo(), "8 GB") || !strings.Contains(inv.Resumo(), "2 discos") {
		t.Fatalf("%v %#v", err, inv)
	}
	ruins := []*Inventario{
		{Sistema: "", Processadores: 1},
		{Sistema: "linux", Processadores: 0},
		{Sistema: "linux", Processadores: 1, MemoriaBytes: -1},
		{Sistema: "linux", Processadores: 1, Discos: []Disco{{Nome: "", Bytes: 1}}},
		{Sistema: "linux", Processadores: 1, Discos: []Disco{{Nome: "C:", Bytes: 1}, {Nome: "C:", Bytes: 2}}},
		{Sistema: "linux", Processadores: 1, Programas: []Programa{{Nome: `C:\Windows\notepad.exe`}}},
	}
	for _, ruim := range ruins {
		if _, err := ValidarInventario(ruim); !errors.Is(err, ErrEntradaInvalida) {
			t.Fatalf("%#v: %v", ruim, err)
		}
	}
}

func TestRelatoComInventario(t *testing.T) {
	unidade := uuid.New()
	inv := &Inventario{Sistema: "Windows 11 amd64", Processadores: 4, MemoriaBytes: 8 << 30, Discos: []Disco{{Nome: "C:", Bytes: 100}}}
	criado, mudou, err := AplicarRelato(nil, Relato{Identificador: "PC-2", Nome: "Balcão", UnidadeID: unidade, Inventario: inv})
	if err != nil || !mudou || criado.Inventario == nil || !strings.Contains(criado.Descricao, "4 processadores") {
		t.Fatalf("%v %v %#v", err, mudou, criado)
	}
	igual, mudou, err := AplicarRelato(&criado, Relato{Identificador: "PC-2", Nome: "Balcão", UnidadeID: unidade, Inventario: inv})
	if err != nil || mudou || !igual.Igual(criado) {
		t.Fatal("o mesmo inventário não regrava")
	}
	inv.MemoriaBytes = 16 << 30
	proximo, mudou, err := AplicarRelato(&criado, Relato{Identificador: "PC-2", Nome: "Balcão", UnidadeID: unidade, Inventario: inv})
	if err != nil || !mudou || !strings.Contains(proximo.Descricao, "16 GB") {
		t.Fatalf("%v %v %s", err, mudou, proximo.Descricao)
	}
	editado, err := proximo.Aplicar(Entrada{Nome: "Balcão 2", Tipo: "hardware", Estado: "em_uso", Identificador: "PC-2", UnidadeID: &unidade})
	if err != nil || editado.Inventario == nil || editado.Inventario.MemoriaBytes != 16<<30 {
		t.Fatal("a edição manual conserva a medida da máquina")
	}
	um, err := ValidarInventario(&Inventario{
		Sistema: "linux", Processadores: 1, MemoriaBytes: 500,
		Discos:    []Disco{{Nome: "sda", Bytes: 1}},
		Programas: []Programa{{Nome: "zsh", Versao: "2"}, {Nome: "zsh", Versao: "1"}},
	})
	if err != nil || !strings.Contains(um.Resumo(), "500 B") || !strings.Contains(um.Resumo(), "1 disco") || !strings.Contains(um.Resumo(), "2 programas") || um.Programas[0].Versao != "1" {
		t.Fatalf("%v %s", err, um.Resumo())
	}
	parcial := *um
	parcial.ProgramasTruncado = true
	parcial.Programas = []Programa{{Nome: "zsh", Versao: "1"}}
	if !strings.Contains(parcial.Resumo(), "1 programa") {
		t.Fatal(parcial.Resumo())
	}
	parcial.Programas = append(parcial.Programas, Programa{Nome: "bash", Versao: "1"})
	if !strings.Contains(parcial.Resumo(), "lista parcial") {
		t.Fatal(parcial.Resumo())
	}
	discos := make([]Disco, 17)
	for i := range discos {
		discos[i] = Disco{Nome: string(rune('a' + i)), Bytes: 1}
	}
	if _, err := ValidarInventario(&Inventario{Sistema: "linux", Processadores: 1, Discos: discos}); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatal("discos")
	}
	programas := make([]Programa, 101)
	for i := range programas {
		programas[i] = Programa{Nome: string(rune('a'+i/26)) + string(rune('a'+i%26))}
	}
	if _, err := ValidarInventario(&Inventario{Sistema: "linux", Processadores: 1, Programas: programas}); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatal("programas")
	}
	outroDisco := *um
	outroDisco.Discos = []Disco{{Nome: "sdb", Bytes: 1}}
	outroPrograma := *um
	outroPrograma.Programas = []Programa{{Nome: "bash", Versao: "1"}, {Nome: "zsh", Versao: "1"}}
	outroSistema := *um
	outroSistema.Sistema = "bsd"
	base := Ativo{Inventario: um}
	if base.Igual(Ativo{Inventario: &outroDisco}) || base.Igual(Ativo{Inventario: &outroPrograma}) || base.Igual(Ativo{Inventario: &outroSistema}) || base.Igual(Ativo{}) {
		t.Fatal("inventários diferentes")
	}
}
