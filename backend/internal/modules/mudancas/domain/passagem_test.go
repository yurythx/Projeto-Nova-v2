package domain

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestPassagemExtraDaMudanca(t *testing.T) {
	solicitante, gestor := uuid.New(), uuid.New()
	const coluna = "c01234567"
	m := Mudanca{Estado: Rascunho, Tipo: TipoNormal, SolicitanteID: solicitante}.ComPassagens([]string{coluna})

	if !slices.Contains(m.PodeIrPara(gestor, PapelGestor), coluna) {
		t.Fatal("o gestor conduz a coluna nova")
	}
	if slices.Contains(m.PodeIrPara(gestor, PapelAprovador), coluna) {
		t.Fatal("quem aprova não ganha a coluna nova")
	}
	foi, decisao, ok, err := m.Transicionar(Transicao{Para: coluna, Autor: gestor}, PapelGestor)
	if err != nil || !ok || foi.Estado != coluna || decisao != nil {
		t.Fatalf("coluna nova: %s %v %v", foi.Estado, decisao, err)
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: coluna, Autor: gestor}, PapelAprovador); err != ErrTransicaoInvalida {
		t.Fatalf("aprovador: %v", err)
	}

	semJanela := Mudanca{Estado: Rascunho, Tipo: TipoNormal, SolicitanteID: solicitante}
	if _, _, _, err := semJanela.Transicionar(Transicao{Para: EmAvaliacao, Autor: gestor}, PapelGestor); err != ErrSemJanela {
		t.Fatalf("janela: %v", err)
	}

	for _, fim := range []string{Concluida, Falhou, Cancelada} {
		encerrada := m
		encerrada.Estado = fim
		if _, _, _, err := encerrada.ComPassagens([]string{coluna}).Transicionar(Transicao{Para: coluna, Autor: gestor}, PapelGestor); err != ErrMudancaEncerrada {
			t.Fatalf("%s: %v", fim, err)
		}
	}
}
