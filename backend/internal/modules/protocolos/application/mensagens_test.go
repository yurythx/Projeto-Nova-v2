package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func TestConversaComOCidadao(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, false)
	s.sino = sinoFalso{}
	cidadao := context.Background()
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "00000000000", "oi"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	c, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "123.456.789-00", "  e aí?  ")
	if err != nil || !m.pedidos[ped.ID].MensagemNova {
		t.Fatal(err)
	}
	if ultimo := c.Historico[len(c.Historico)-1]; ultimo.Acao != domain.AcaoMensagemCidadao || ultimo.Descricao != "e aí?" {
		t.Fatal(ultimo)
	}
	// Com responsável, o aviso vai ao sino dele.
	p := m.pedidos[ped.ID]
	dono := uuid.New()
	p.ResponsavelID = &dono
	m.pedidos[ped.ID] = p
	for range domain.MensagensSemResposta - 1 {
		if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "alô"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "alô?"); !errors.Is(err, domain.ErrMuitasMensagens) {
		t.Fatal(err)
	}
	if MapError(domain.ErrMuitasMensagens) == domain.ErrMuitasMensagens {
		t.Fatal("traduz o limite")
	}

	// A mesa lê sem responder: a novidade apaga e a trilha não muda.
	mesa := ctxGestao(un)
	linhas := len(m.hist)
	lido, err := s.MarcarMensagensLidas(mesa, ped.ID)
	if err != nil || lido.MensagemNova || m.pedidos[ped.ID].MensagemNova || len(m.hist) != linhas {
		t.Fatal(err)
	}
	if _, err := s.MarcarMensagensLidas(mesa, ped.ID); err != nil {
		t.Fatal("já lida", err)
	}

	// A resposta da mesa libera o cidadão de novo.
	if _, err := s.EnviarMensagem(mesa, ped.ID, " "); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.EnviarMensagem(mesa, ped.ID, "em análise"); err != nil {
		t.Fatal(err)
	}
	if ultimaDescricao(m, domain.AcaoMensagemMesa) != "em análise" {
		t.Fatal("trilha")
	}
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "obrigado"); err != nil {
		t.Fatal(err)
	}
}

func TestConversaFalhas(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, false)
	cidadao := context.Background()
	m.falha = "listar-hist"
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "oi"); err == nil {
		t.Fatal("histórico")
	}
	m.falha = "atualizar-pedido"
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "oi"); err == nil {
		t.Fatal("gravação")
	}
	m.falha = "hist"
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "oi"); err == nil {
		t.Fatal("trilha")
	}
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", " "); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	m.falha = ""
	// O cidadão escreve; a mesa sem permissão na unidade não lê.
	if _, err := s.EnviarMensagemPublica(cidadao, ped.Numero, "12345678900", "oi"); err != nil {
		t.Fatal(err)
	}
	estranho := ctxDe(uuid.New(), uuid.New(), auth.PermProtocolosTratar)
	if _, err := s.MarcarMensagensLidas(estranho, ped.ID); err == nil {
		t.Fatal("permissão")
	}
	m.falha = "atualizar-pedido"
	if _, err := s.MarcarMensagensLidas(ctxGestao(un), ped.ID); err == nil {
		t.Fatal("gravação")
	}
	m.falha = "buscar-pedido"
	if _, err := s.MarcarMensagensLidas(ctxGestao(un), ped.ID); err == nil {
		t.Fatal("busca")
	}
}
