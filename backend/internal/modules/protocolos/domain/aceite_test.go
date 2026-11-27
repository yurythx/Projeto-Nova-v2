package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAceiteETransferencia(t *testing.T) {
	agora := time.Date(2026, 11, 27, 10, 0, 0, 0, Cuiaba)
	ana, bia := uuid.New(), uuid.New()
	novo := Pedido{Status: StatusNovo, UnidadeID: uuid.New()}

	aceito, err := novo.Aceitar(ana, agora)
	if err != nil || aceito.ResponsavelID == nil || *aceito.ResponsavelID != ana || aceito.Status != StatusAndamento {
		t.Fatalf("%+v %v", aceito, err)
	}
	// O pedido já em análise mantém o status ao ganhar dono.
	if p, err := (Pedido{Status: StatusAnalise}).Aceitar(ana, agora); err != nil || p.Status != StatusAnalise {
		t.Fatal(p.Status, err)
	}
	if _, err := aceito.Aceitar(bia, agora); !errors.Is(err, ErrInvalido) {
		t.Fatal("já tem dono", err)
	}
	if _, err := novo.Aceitar(uuid.Nil, agora); !errors.Is(err, ErrInvalido) {
		t.Fatal("sem pessoa", err)
	}
	if _, err := (Pedido{Status: StatusComplementacao}).Aceitar(ana, agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := (Pedido{Status: StatusConcluido}).Aceitar(ana, agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := (Pedido{Status: StatusNovo, Encaminhamento: &Encaminhamento{}}).Aceitar(ana, agora); !errors.Is(err, ErrEncaminhamento) {
		t.Fatal(err)
	}

	transferido, err := aceito.Transferir(bia, agora)
	if err != nil || *transferido.ResponsavelID != bia {
		t.Fatal(err)
	}
	for _, c := range []struct {
		p    Pedido
		novo uuid.UUID
		erro error
	}{
		{novo, bia, ErrInvalido},
		{aceito, uuid.Nil, ErrInvalido},
		{aceito, ana, ErrInvalido},
		{Pedido{Status: StatusCancelado}, bia, ErrEstadoFinal},
	} {
		if _, err := c.p.Transferir(c.novo, agora); !errors.Is(err, c.erro) {
			t.Fatal(err)
		}
	}
}

func TestEncaminhamento(t *testing.T) {
	agora := time.Date(2026, 11, 27, 10, 0, 0, 0, Cuiaba)
	origem, destino, ana := uuid.New(), uuid.New(), uuid.New()
	dep := uuid.New()
	p := Pedido{Status: StatusAndamento, UnidadeID: origem, DepartamentoID: &dep, ResponsavelID: &ana, ResponsavelNome: "Ana"}

	enc, err := p.Encaminhar(destino, "  é da saúde  ", 5, ana, agora)
	if err != nil || enc.Encaminhamento == nil || enc.Encaminhamento.Justificativa != "é da saúde" || *enc.Encaminhamento.Por != ana {
		t.Fatalf("%+v %v", enc.Encaminhamento, err)
	}
	if _, err := enc.Encaminhar(destino, "de novo", 0, ana, agora); !errors.Is(err, ErrEncaminhamento) {
		t.Fatal(err)
	}
	for _, c := range []struct {
		destino uuid.UUID
		texto   string
		prazo   int
	}{
		{uuid.Nil, "x", 0}, {origem, "x", 0}, {destino, "x", -1}, {destino, "x", PrazoMaximoEncaminhamento + 1}, {destino, "  ", 0},
	} {
		if _, err := p.Encaminhar(c.destino, c.texto, c.prazo, ana, agora); !errors.Is(err, ErrInvalido) {
			t.Fatal(c, err)
		}
	}
	if _, err := (Pedido{Status: StatusComplementacao, UnidadeID: origem}).Encaminhar(destino, "x", 0, ana, agora); !errors.Is(err, ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := (Pedido{Status: StatusConcluido}).Encaminhar(destino, "x", 0, ana, agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}

	enc.Encaminhamento.UnidadeNome = "Saúde"
	recebido, err := enc.ReceberEncaminhamento(agora)
	if err != nil || recebido.UnidadeID != destino || recebido.UnidadeNome != "Saúde" || recebido.ResponsavelID != nil ||
		recebido.DepartamentoID != nil || recebido.ResponsavelNome != "" || recebido.Encaminhamento != nil || recebido.PrazoEm == nil {
		t.Fatalf("%+v %v", recebido, err)
	}
	semPrazo, _ := p.Encaminhar(destino, "x", 0, ana, agora)
	if r, err := semPrazo.ReceberEncaminhamento(agora); err != nil || r.PrazoEm != nil {
		t.Fatal(err)
	}
	if _, err := p.ReceberEncaminhamento(agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := (Pedido{Status: StatusConcluido}).ReceberEncaminhamento(agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}

	recusado, err := enc.RecusarEncaminhamento(agora)
	if err != nil || recusado.Encaminhamento != nil || recusado.UnidadeID != origem || *recusado.ResponsavelID != ana {
		t.Fatal(err)
	}
	if _, err := p.RecusarEncaminhamento(agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := (Pedido{Status: StatusCancelado}).RecusarEncaminhamento(agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
}

func TestAjusteEAprovacao(t *testing.T) {
	agora := time.Date(2026, 11, 27, 10, 0, 0, 0, Cuiaba)
	p := Pedido{Status: StatusAndamento, Prioridade: "normal", CreatedAt: agora}

	ajustado, err := p.Ajustar("urgente", "2026-12-10", agora)
	if err != nil || ajustado.Prioridade != "urgente" || ajustado.PrazoEm.Format("2006-01-02 15:04") != "2026-12-10 23:59" {
		t.Fatal(ajustado.PrazoEm, err)
	}
	if mantido, err := ajustado.Ajustar("", "", agora); err != nil || mantido.Prioridade != "urgente" || mantido.PrazoEm == nil {
		t.Fatal(err)
	}
	if _, err := p.Ajustar("altissima", "", agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := p.Ajustar("", "10/12/2026", agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := p.Ajustar("", "2026-11-26", agora); !errors.Is(err, ErrInvalido) {
		t.Fatal("antes da abertura", err)
	}
	if _, err := (Pedido{Status: StatusConcluido}).Ajustar("alta", "", agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}

	aguardando, err := p.PedirAprovacao(DecisaoFavoravel, "deferido", agora)
	if err != nil || aguardando.Status != StatusAguardando || aguardando.Decisao != DecisaoFavoravel {
		t.Fatal(err)
	}
	if _, err := p.PedirAprovacao("talvez", "x", agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	// A conclusão que aguarda não aceita outra conclusão.
	if _, err := aguardando.Concluir(DecisaoDesfavoravel, "outra", agora); !errors.Is(err, ErrAprovacao) {
		t.Fatal(err)
	}
	aprovado, err := aguardando.Aprovar(agora)
	if err != nil || aprovado.Status != StatusConcluido || aprovado.Decisao != DecisaoFavoravel {
		t.Fatal(err)
	}
	devolvido, err := aguardando.Devolver(agora)
	if err != nil || devolvido.Status != StatusAndamento || devolvido.Decisao != "" || devolvido.Justificativa != "" {
		t.Fatal(err)
	}
	// Etapa do workflow em "pendente de aprovação", sem decisão: não há o que aprovar.
	etapa := Pedido{Status: StatusAguardando}
	if _, err := etapa.Aprovar(agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := p.Devolver(agora); !errors.Is(err, ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := aprovado.Aprovar(agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
}

func TestTextoPublicoDasAcoes(t *testing.T) {
	ana := uuid.New()
	hist := []Historico{
		{Acao: AcaoAceite, Descricao: "assumido por Ana Lima", UsuarioID: &ana},
		{Acao: AcaoEncaminhamento, Descricao: "encaminhado a Saúde: interno"},
		{Acao: AcaoDevolucao, Descricao: "parecer fraco"},
		{Acao: "avanco", Descricao: "pedido avançou de etapa"},
	}
	c := Publico(Pedido{Status: StatusAndamento}, "", hist, nil)
	esperado := []string{"pedido em análise", "encaminhado a outra unidade", "conclusão em revisão", "pedido avançou de etapa"}
	for i, h := range c.Historico {
		if h.Descricao != esperado[i] {
			t.Fatal(i, h.Descricao)
		}
	}
	if _, ok := AcaoPublica("criado"); ok {
		t.Fatal("criado mantém a descrição")
	}
}
