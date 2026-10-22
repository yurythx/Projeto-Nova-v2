package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNovoECiclo(t *testing.T) {
	u := uuid.New()
	p, err := Novo("  Protocolo  ", " sistema ", "operacao", " registrar ", " no horário ", &u)
	if err != nil {
		t.Fatal(err)
	}
	if p.Nome != "Protocolo" || p.Descricao != "sistema" || p.Utilidade != "registrar" || p.Garantia != "no horário" {
		t.Fatalf("aparou mal: %+v", p)
	}
	if p.ID == uuid.Nil || p.CreatedAt.IsZero() {
		t.Fatal("id e horário precisam nascer preenchidos")
	}

	for _, e := range Estagios {
		if !EstagioValido(e) {
			t.Errorf("estágio do ciclo recusado: %s", e)
		}
	}
	if EstagioValido("legado") {
		t.Fatal("estágio fora do ciclo foi aceito")
	}
	if _, err := Novo("", "", "operacao", "", "", nil); err != ErrEntradaInvalida {
		t.Fatalf("nome vazio: %v", err)
	}
	if _, err := Novo(strings.Repeat("a", NomeMax+1), "", "operacao", "", "", nil); err != ErrEntradaInvalida {
		t.Fatalf("nome longo: %v", err)
	}
	if _, err := Novo("Ok", strings.Repeat("á", TextoMax+1), "operacao", "", "", nil); err != ErrEntradaInvalida {
		t.Fatalf("descrição longa: %v", err)
	}
	if _, err := Novo("Ok", "", "operacao", strings.Repeat("b", TextoMax+1), "", nil); err != ErrEntradaInvalida {
		t.Fatalf("utilidade longa: %v", err)
	}
	if _, err := Novo("Ok", "", "operacao", "", strings.Repeat("c", TextoMax+1), nil); err != ErrEntradaInvalida {
		t.Fatalf("garantia longa: %v", err)
	}
	if _, err := Novo("Ok", "", "legado", "", "", nil); err != ErrEstagioInvalido {
		t.Fatalf("estágio: %v", err)
	}

	igual, mudou, err := p.ComEstagio("operacao")
	if err != nil || mudou || !igual.Igual(p) {
		t.Fatalf("repetir o estágio: mudou=%v err=%v", mudou, err)
	}
	prox, mudou, err := p.ComEstagio("suporte")
	if err != nil || !mudou || prox.Estagio != "suporte" || prox.Igual(p) {
		t.Fatalf("trocar o estágio: %+v %v %v", prox, mudou, err)
	}
	if _, _, err := p.ComEstagio("legado"); err != ErrEstagioInvalido {
		t.Fatalf("estágio na troca: %v", err)
	}

	edit, err := p.Aplicar(" Protocolo web ", "", "", "", nil)
	if err != nil || edit.Nome != "Protocolo web" || edit.Estagio != p.Estagio || edit.UnidadeID != nil {
		t.Fatalf("aplicar: %+v %v", edit, err)
	}
	if _, err := p.Aplicar(" ", "", "", "", nil); err != ErrEntradaInvalida {
		t.Fatalf("aplicar nome vazio: %v", err)
	}
	if _, err := p.Aplicar("Ok", strings.Repeat("x", TextoMax+1), "", "", nil); err != ErrEntradaInvalida {
		t.Fatalf("aplicar descrição: %v", err)
	}
	if edit.Igual(p) {
		t.Fatal("edição com nome novo não pode ser igual")
	}

	inst := Dona(nil)
	if inst.UnidadeID != nil || inst.EntidadeID != nil {
		t.Fatal("institucional não tem dono")
	}
	if Dona(&u).UnidadeID == nil || *Dona(&u).UnidadeID != u {
		t.Fatal("dona da unidade")
	}
}

func TestMesmaUnidade(t *testing.T) {
	u, v := uuid.New(), uuid.New()
	if !mesmaUnidade(nil, nil) || mesmaUnidade(&u, nil) || mesmaUnidade(nil, &u) || !mesmaUnidade(&u, &u) || mesmaUnidade(&u, &v) {
		t.Fatal("comparação de unidade dona")
	}
}
