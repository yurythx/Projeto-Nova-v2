package notificacoes

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// carteiro guarda o que sairia por e-mail e por WhatsApp.
type carteiro struct {
	emails   []Mensagem
	whatsapp []MensagemWhatsApp
	err      error
}

func (c *carteiro) Send(_ context.Context, m Mensagem) error {
	c.emails = append(c.emails, m)
	return c.err
}

func (c *carteiro) SendWhatsApp(_ context.Context, m MensagemWhatsApp) error {
	c.whatsapp = append(c.whatsapp, m)
	return c.err
}

// Os avisos do atendimento saem pelo e-mail e pelo WhatsApp de cada um; os
// dos outros módulos ficam só no sino (ADR 049, ADR 078).
func TestCanaisDeSaida(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	comContato, soLogin, semEmail := dbtest.User(t, pool), dbtest.User(t, pool), dbtest.User(t, pool)
	for _, q := range []struct {
		sql string
		id  uuid.UUID
	}{
		{`UPDATE users SET contato_email = 'contato.' || id || '@x.gov.br', contato_whatsapp = '+5566999991234' WHERE id = $1`, comContato},
		{`UPDATE users SET email = 'kc-' || id || '@sem-email.invalid' WHERE id = $1`, semEmail},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.id); err != nil {
			t.Fatal(err)
		}
	}
	c := &carteiro{}
	var registro bytes.Buffer
	s := NewService(pool, nil, slog.New(slog.NewTextHandler(&registro, nil))).WithCorreio(c, "https://nova.gov.br/").WithWhatsApp(c)
	aviso := Nova{Modulo: "atendimento", Titulo: "Recebemos o seu chamado #7", Mensagem: "Senha: acesso", Link: "/atendimento/solicitacoes/x",
		Chave: "teste:" + uuid.NewString()}
	if _, err := s.Enviar(ctx, []uuid.UUID{comContato, soLogin, semEmail}, aviso); err != nil {
		t.Fatal(err)
	}
	if len(c.emails) != 2 {
		t.Fatalf("e-mail para o contato e para o login real, nunca para o marcador: %+v", c.emails)
	}
	for _, m := range c.emails {
		if strings.HasSuffix(m.Para, ".invalid") || !strings.Contains(m.Corpo, "https://nova.gov.br/atendimento/solicitacoes/x") ||
			!strings.Contains(m.Corpo, "https://nova.gov.br/notificacoes") || m.Assunto != aviso.Titulo {
			t.Fatalf("e-mail: %+v", m)
		}
	}
	if len(c.whatsapp) != 1 || c.whatsapp[0].Para != "+5566999991234" || c.whatsapp[0].Link != "https://nova.gov.br/atendimento/solicitacoes/x" ||
		c.whatsapp[0].Titulo != aviso.Titulo || c.whatsapp[0].Mensagem != aviso.Mensagem {
		t.Fatalf("WhatsApp só para quem deixou o número: %+v", c.whatsapp)
	}

	// Outro módulo não sai do sino; sem link, o WhatsApp vai sem ele.
	c.emails, c.whatsapp = nil, nil
	if _, err := s.Enviar(ctx, []uuid.UUID{comContato}, nova("teste:"+uuid.NewString())); err != nil || len(c.emails)+len(c.whatsapp) != 0 {
		t.Fatalf("wiki saiu do sino: %v %+v %+v", err, c.emails, c.whatsapp)
	}
	semLink := aviso
	semLink.Link, semLink.Chave = "", "teste:"+uuid.NewString()
	if _, err := s.Enviar(ctx, []uuid.UUID{comContato}, semLink); err != nil || len(c.whatsapp) != 1 || c.whatsapp[0].Link != "" {
		t.Fatalf("sem link: %v %+v", err, c.whatsapp)
	}

	// A falha do canal fica no registro e não desfaz o sino.
	c.err = errors.New("provedor fora")
	aviso.Chave = "teste:" + uuid.NewString()
	if n, err := s.Enviar(ctx, []uuid.UUID{comContato}, aviso); err != nil || n != 1 {
		t.Fatalf("o sino vale com o canal fora: %d %v", n, err)
	}
	if !strings.Contains(registro.String(), "e-mail do aviso falhou") || !strings.Contains(registro.String(), "WhatsApp do aviso falhou") {
		t.Fatalf("registro: %s", registro.String())
	}

	// Sem conseguir ler o contato, nada sai, e o motivo fica no registro.
	registro.Reset()
	enviadas := []Enviada{{Usuario: comContato, Item: Notificacao{Modulo: "atendimento", Titulo: "x"}}}
	s.db = dbtest.Fail{}
	s.Entregar(ctx, enviadas)
	if !strings.Contains(registro.String(), "não leu o e-mail") || !strings.Contains(registro.String(), "não leu o WhatsApp") {
		t.Fatalf("leitura falhou: %s", registro.String())
	}
	s.db = dbtest.ScanFail{}
	s.Entregar(ctx, enviadas)
	sem := NewService(pool, nil, nil).WithCorreio(c, "").WithWhatsApp(c)
	sem.db = dbtest.Fail{}
	sem.Entregar(ctx, enviadas)
}
