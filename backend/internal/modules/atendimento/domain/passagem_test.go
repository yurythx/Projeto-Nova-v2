package domain

import (
	"slices"
	"testing"
)

func TestPassagemExtraDaFila(t *testing.T) {
	const coluna = "c01234567"
	s := Solicitacao{Tipo: TipoIncidente, Estado: EstadoAberto}.ComPassagens([]string{coluna})
	if !slices.Contains(s.Proximos(PapelTecnico), coluna) {
		t.Fatal("quem atende conduz a coluna nova")
	}
	if slices.Contains(s.Proximos(PapelSolicitante), coluna) {
		t.Fatal("quem abriu o pedido não ganha caminho novo")
	}
	if _, _, err := s.Transicionar(coluna, PapelSolicitante); err != ErrTransicaoInvalida {
		t.Fatalf("solicitante: %v", err)
	}
	foi, ok, err := s.Transicionar(coluna, PapelTecnico)
	if err != nil || !ok || foi.Estado != coluna {
		t.Fatalf("tecnico: %s %v %v", foi.Estado, ok, err)
	}

	for _, fim := range []string{EstadoEncerrado, EstadoCancelado} {
		encerrada := s
		encerrada.Estado = fim
		if got := encerrada.ComPassagens([]string{coluna}).Proximos(PapelTecnico); len(got) != 0 {
			t.Fatalf("%s ainda oferece passagem: %v", fim, got)
		}
		if _, _, err := encerrada.ComPassagens([]string{coluna}).Transicionar(coluna, PapelTecnico); err != ErrSolicitacaoFinalizada {
			t.Fatalf("%s: %v", fim, err)
		}
	}
}
