package domain

import "testing"

func TestCamposERespostas(t *testing.T) {
	campos, err := ValidarCampos([]Campo{{Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true}, {Rotulo: "Andar", Tipo: "lista", Opcoes: []string{"1", "2"}}})
	if err != nil || campos[0].Chave != "ramal" || campos[1].Chave != "andar" {
		t.Fatalf("%+v %v", campos, err)
	}
	if _, err := ValidarCampos([]Campo{{Rotulo: "Andar", Tipo: "lista"}}); err == nil {
		t.Fatal("lista sem opção")
	}
	if _, err := ValidarRespostas(campos, map[string]string{"ramal": "3450", "andar": "9"}); err == nil {
		t.Fatal("opção inexistente")
	}
	if _, err := ValidarRespostas(campos, map[string]string{"andar": "1"}); err == nil {
		t.Fatal("obrigatório")
	}
	ok, err := ValidarRespostas(campos, map[string]string{"ramal": "3450", "andar": "1"})
	if err != nil || ok["ramal"] != "3450" {
		t.Fatalf("%+v %v", ok, err)
	}
	if _, err := ValidarRespostas(campos, map[string]string{"ramal": "1", "extra": "x"}); err == nil {
		t.Fatal("campo a mais")
	}
}
