package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func entrada() Entrada {
	u := uuid.New()
	return Entrada{Nome: "  Switch  ", Tipo: "rede", Estado: "em_uso", Identificador: " PAT-1 ", Descricao: " 24 portas ", UnidadeID: &u}
}

func TestNovoEAplicar(t *testing.T) {
	a, err := Novo(entrada())
	if err != nil || a.Nome != "Switch" || a.Identificador != "PAT-1" || a.Descricao != "24 portas" || a.ID == uuid.Nil {
		t.Fatalf("novo: %+v %v", a, err)
	}
	casos := map[string]func(*Entrada){
		"sem nome":            func(e *Entrada) { e.Nome = " " },
		"nome longo":          func(e *Entrada) { e.Nome = strings.Repeat("a", NomeMax+1) },
		"identificador longo": func(e *Entrada) { e.Identificador = strings.Repeat("a", IdentificadorMax+1) },
		"descrição longa":     func(e *Entrada) { e.Descricao = strings.Repeat("a", TextoMax+1) },
	}
	for nome, mudar := range casos {
		in := entrada()
		mudar(&in)
		if _, err := Novo(in); err != ErrEntradaInvalida {
			t.Errorf("%s: %v", nome, err)
		}
	}
	tipo := entrada()
	tipo.Tipo = "nave"
	if _, err := Novo(tipo); err != ErrTipoInvalido {
		t.Fatal(err)
	}
	estado := entrada()
	estado.Estado = "perdido"
	if _, err := a.Aplicar(estado); err != ErrEstadoInvalido {
		t.Fatal(err)
	}
	in := entrada()
	in.Nome, in.Estado, in.UnidadeID = "Switch", "manutencao", a.UnidadeID
	b, err := a.Aplicar(in)
	if err != nil || b.Estado != "manutencao" || b.Igual(a) || !a.Igual(a) {
		t.Fatalf("aplicar: %+v %v", b, err)
	}
	if Dona(nil) != (Dona(nil)) || Dona(a.UnidadeID).UnidadeID == nil {
		t.Fatal("dona")
	}
	u := uuid.New()
	if !MesmaUnidade(nil, nil) || MesmaUnidade(&u, nil) || !MesmaUnidade(&u, &u) {
		t.Fatal("mesma unidade")
	}
}
