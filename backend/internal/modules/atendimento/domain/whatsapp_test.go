package domain

import "testing"

func TestRoteiroDoWhatsApp(t *testing.T) {
	ofertas := make([]Oferta, 0, 12)
	for i := 0; i < 12; i++ {
		ofertas = append(ofertas, Oferta{Nome: "s"})
	}
	pagina, mais, usada := PaginaDeOfertas(ofertas, 0)
	if len(pagina) != OfertasPorPagina || !mais || usada != 0 {
		t.Fatalf("primeira: %d %v %d", len(pagina), mais, usada)
	}
	pagina, mais, usada = PaginaDeOfertas(ofertas, 1)
	if len(pagina) != 3 || mais || usada != 1 {
		t.Fatalf("segunda: %d %v %d", len(pagina), mais, usada)
	}
	if pagina, mais, usada = PaginaDeOfertas(ofertas, 9); len(pagina) != OfertasPorPagina || !mais || usada != 0 {
		t.Fatalf("página alta volta ao início: %d %v %d", len(pagina), mais, usada)
	}

	opcoes := []OpcaoWhatsApp{{ID: "mais", Titulo: "Ver mais"}, {ID: "abc", Titulo: "E-mail"}}
	if o, ok := EscolherOpcao("2", opcoes); !ok || o.ID != "abc" {
		t.Fatalf("número: %+v %v", o, ok)
	}
	if o, ok := EscolherOpcao("ABC", opcoes); !ok || o.Titulo != "E-mail" {
		t.Fatalf("id: %+v %v", o, ok)
	}
	if o, ok := EscolherOpcao("ver mais", opcoes); !ok || o.ID != "mais" {
		t.Fatalf("título: %+v %v", o, ok)
	}
	if _, ok := EscolherOpcao("9", opcoes); ok {
		t.Fatal("fora da lista")
	}

	ramal := Campo{Chave: "ramal", Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true}
	andar := Campo{Chave: "andar", Rotulo: "Andar", Tipo: "lista", Obrigatorio: true, Opcoes: []string{"Térreo", "1"}}
	if v, ok := RespostaDeCampo(ramal, "3450"); !ok || v != "3450" {
		t.Fatalf("número: %q %v", v, ok)
	}
	if _, ok := RespostaDeCampo(ramal, "34-50"); ok {
		t.Fatal("número com traço")
	}
	if v, ok := RespostaDeCampo(andar, "1"); !ok || v != "1" {
		t.Fatalf("opção com o mesmo texto do número: %q %v", v, ok)
	}
	if v, ok := RespostaDeCampo(andar, "térreo"); !ok || v != "Térreo" {
		t.Fatalf("texto da lista: %q %v", v, ok)
	}
	if obr := CamposObrigatorios([]Campo{ramal, {Rotulo: "Obs"}}); len(obr) != 1 || obr[0].Chave != "ramal" {
		t.Fatalf("obrigatórios: %+v", obr)
	}
	if !ComandoCancelar(" Sair ") || !ComandoNovo("NOVO") || ComandoNovo("novo chamado") {
		t.Fatal("comandos")
	}
}
