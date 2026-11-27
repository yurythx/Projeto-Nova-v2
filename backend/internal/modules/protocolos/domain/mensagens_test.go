package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSeguidasDoCidadao(t *testing.T) {
	h := []Historico{
		{Acao: AcaoMensagemCidadao}, {Acao: AcaoMensagemMesa},
		{Acao: AcaoMensagemCidadao}, {Acao: AcaoNota}, {Acao: AcaoMensagemCidadao},
	}
	if n := SeguidasDoCidadao(h); n != 2 {
		t.Fatal(n)
	}
	if n := SeguidasDoCidadao([]Historico{{Acao: AcaoMensagemCidadao}}); n != 1 {
		t.Fatal(n)
	}
	if n := SeguidasDoCidadao(nil); n != 0 {
		t.Fatal(n)
	}
}

func TestMensagens(t *testing.T) {
	agora := time.Date(2026, 11, 27, 10, 0, 0, 0, Cuiaba)
	p := Pedido{Status: StatusAndamento}
	prox, texto, err := p.MensagemDoCidadao("  e aí?  ", 0, agora)
	if err != nil || !prox.MensagemNova || texto != "e aí?" || !prox.UpdatedAt.Equal(agora) {
		t.Fatal(prox, texto, err)
	}
	if _, _, err := p.MensagemDoCidadao("oi", MensagensSemResposta, agora); !errors.Is(err, ErrMuitasMensagens) {
		t.Fatal(err)
	}
	if _, _, err := p.MensagemDoCidadao(" ", 0, agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, _, err := p.MensagemDoCidadao(strings.Repeat("a", 1001), 0, agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, _, err := (Pedido{Status: StatusComplementacao}).MensagemDoCidadao("oi", 0, agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	if _, _, err := (Pedido{Status: StatusCancelado}).MensagemDoCidadao("oi", 0, agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	resp, texto, err := prox.MensagemDaMesa(" em análise ", agora)
	if err != nil || resp.MensagemNova || texto != "em análise" {
		t.Fatal(resp, texto, err)
	}
	if _, _, err := prox.MensagemDaMesa("", agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, _, err := (Pedido{Status: StatusConcluido}).MensagemDaMesa("oi", agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	// A consulta pública mostra a conversa com o texto.
	c := Publico(Pedido{Status: StatusAndamento}, "", []Historico{
		{Acao: AcaoMensagemCidadao, Descricao: "e aí?"}, {Acao: AcaoMensagemMesa, Descricao: "em análise"},
	}, nil)
	if len(c.Historico) != 2 || c.Historico[1].Descricao != "em análise" {
		t.Fatal(c.Historico)
	}
}
