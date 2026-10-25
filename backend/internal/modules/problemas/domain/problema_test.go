package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNovoEEdicao(t *testing.T) {
	u := uuid.New()
	p, err := Novo(Entrada{Titulo: "  Quedas  ", Descricao: " d ", UnidadeID: &u}, uuid.New())
	if err != nil || p.Titulo != "Quedas" || p.Estado != Registrado || p.CriadoPor == nil || len(p.Incidentes) != 0 {
		t.Fatalf("novo: %+v %v", p, err)
	}
	if semAutor, _ := Novo(Entrada{Titulo: "x"}, uuid.Nil); semAutor.CriadoPor != nil {
		t.Fatal("sem autor")
	}
	for nome, in := range map[string]Entrada{
		"sem título":      {Titulo: " "},
		"título longo":    {Titulo: strings.Repeat("a", TituloMax+1)},
		"descrição longa": {Titulo: "x", Descricao: strings.Repeat("a", TextoMax+1)},
		"contorno longo":  {Titulo: "x", Contorno: strings.Repeat("a", TextoMax+1)},
	} {
		if _, err := Novo(in, uuid.Nil); err != ErrEntradaInvalida {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := p.Aplicar(Entrada{}); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	editado, err := p.Aplicar(Entrada{Titulo: "Quedas", Contorno: "reiniciar", UnidadeID: &u})
	if err != nil || editado.Igual(p) || !p.Igual(p) {
		t.Fatalf("edição: %+v %v", editado, err)
	}
	conhecido := p
	conhecido.Estado, conhecido.Contorno = ErroConhecido, "reiniciar"
	if _, err := conhecido.Aplicar(Entrada{Titulo: "x"}); err != ErrSemContorno {
		t.Fatalf("não apaga o contorno do erro conhecido: %v", err)
	}
	resolvido := p
	resolvido.Estado, resolvido.CausaRaiz = Resolvido, "fonte"
	if _, err := resolvido.Aplicar(Entrada{Titulo: "x"}); err != ErrSemCausaRaiz {
		t.Fatalf("não apaga a causa do resolvido: %v", err)
	}
	encerrado := resolvido
	encerrado.Estado = Encerrado
	if _, err := encerrado.Aplicar(Entrada{Titulo: "x"}); err != ErrEncerrado {
		t.Fatal(err)
	}
	if Dona(nil).UnidadeID != nil || Dona(&u).UnidadeID == nil || !MesmaUnidade(nil, nil) || MesmaUnidade(&u, nil) || !MesmaUnidade(&u, &u) {
		t.Fatal("dona e unidade")
	}
}

func TestCiclo(t *testing.T) {
	p, _ := Novo(Entrada{Titulo: "Quedas"}, uuid.New())
	if !slices.Equal(p.PodeIrPara(), []string{EmAnalise}) {
		t.Fatalf("próximos: %v", p.PodeIrPara())
	}
	if _, _, err := p.Transicionar(Resolvido); err != ErrTransicaoInvalida {
		t.Fatal(err)
	}
	p, mudou, err := p.Transicionar(EmAnalise)
	if err != nil || !mudou {
		t.Fatal(err)
	}
	if _, mudou, _ := p.Transicionar(EmAnalise); mudou {
		t.Fatal("mesmo estado")
	}
	if _, _, err := p.Transicionar(ErroConhecido); err != ErrSemContorno {
		t.Fatal(err)
	}
	if _, _, err := p.Transicionar(Resolvido); err != ErrSemCausaRaiz {
		t.Fatal(err)
	}
	p.Contorno = "reiniciar"
	if p, _, err = p.Transicionar(ErroConhecido); err != nil {
		t.Fatal(err)
	}
	p.CausaRaiz = "fonte"
	if p, _, err = p.Transicionar(Resolvido); err != nil || p.ResolvidoEm == nil {
		t.Fatalf("resolvido: %+v %v", p, err)
	}
	reaberto, _, err := p.Transicionar(EmAnalise)
	if err != nil || reaberto.ResolvidoEm != nil {
		t.Fatalf("reaberto perde a resolução: %+v %v", reaberto, err)
	}
	if p, _, err = p.Transicionar(Encerrado); err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.Transicionar(EmAnalise); err != ErrEncerrado {
		t.Fatal(err)
	}
	if len(p.PodeIrPara()) != 0 {
		t.Fatal("encerrado não tem próximos")
	}
}

func TestNaoConcluivel(t *testing.T) {
	var vazio *NaoConcluivel
	if vazio.Error() == "" || vazio.Unwrap() != nil {
		t.Fatal(vazio)
	}
	sem := &NaoConcluivel{}
	if sem.Error() == "" || sem.Unwrap() != nil {
		t.Fatal(sem)
	}
	com := &NaoConcluivel{Causa: ErrSemCausaRaiz}
	if com.Error() != ErrSemCausaRaiz.Error() || com.Unwrap() != ErrSemCausaRaiz {
		t.Fatal(com)
	}
}
