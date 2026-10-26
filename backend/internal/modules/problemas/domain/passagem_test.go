package domain

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestPassagemExtraDoProblema(t *testing.T) {
	p, err := Novo(Entrada{Titulo: "Quedas"}, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	const coluna = "c01234567"
	if slices.Contains(p.PodeIrPara(), coluna) {
		t.Fatal("sem passagem extra a coluna nova não aparece")
	}
	if _, _, err := p.Transicionar(coluna); err != ErrTransicaoInvalida {
		t.Fatalf("sem extra: %v", err)
	}
	if !slices.Contains(p.PodeIrPara(coluna), coluna) {
		t.Fatal("a passagem extra entra nos próximos")
	}
	foi, ok, err := p.Transicionar(coluna, coluna)
	if err != nil || !ok || foi.Estado != coluna {
		t.Fatalf("extra: %+v %v %v", foi.Estado, ok, err)
	}

	semCausa, _, err := p.Transicionar(Resolvido, Resolvido)
	if err != ErrSemCausaRaiz || semCausa.Estado != "" {
		t.Fatalf("causa: %+v %v", semCausa.Estado, err)
	}
	p.CausaRaiz = "fonte"
	resolvido, ok, err := p.Transicionar(Resolvido, Resolvido)
	if err != nil || !ok || resolvido.Estado != Resolvido || resolvido.ResolvidoEm == nil {
		t.Fatalf("resolvido: %+v %v", resolvido, err)
	}

	encerrado := resolvido
	encerrado.Estado = Encerrado
	if encerrado.PodeIrPara(coluna) != nil {
		t.Fatalf("encerrado ainda sai: %v", encerrado.PodeIrPara(coluna))
	}
	if _, _, err := encerrado.Transicionar(coluna, coluna); err != ErrEncerrado {
		t.Fatalf("encerrado: %v", err)
	}
}
