package mail

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

func TestEnvioPortaPadraoETls(t *testing.T) {
	ctx := context.Background()
	if err := (SMTP{Host: "127.0.0.1", De: "a@b.co"}).Send(ctx, Mensagem{Para: "c@d.co"}); err == nil {
		t.Fatal("porta 587 recusada")
	}
	if err := (SMTP{Host: "127.0.0.1", Porta: 465, De: "a@b.co"}).Send(ctx, Mensagem{Para: "c@d.co"}); err == nil {
		t.Fatal("porta 465 recusada")
	}
	if hostDe("semporta") != "semporta" {
		t.Fatal("host sem porta")
	}
}

func TestEnvioQuebraNoCaminho(t *testing.T) {
	for _, recusa := range []string{"banner", "starttls", "auth", "mail", "data", "corpo", "fecha"} {
		t.Run(recusa, func(t *testing.T) {
			ln := escuta(t)
			go smtpQueRecusa(ln, recusa)
			host, porta := hostPorta(t, ln)
			smtp := SMTP{Host: host, Porta: porta, De: "nova@org.gov.br"}
			if recusa == "auth" {
				smtp.Usuario, smtp.Senha = "nova", "segredo"
			}
			corpo := "x"
			if recusa == "corpo" {
				corpo = strings.Repeat("a", 1<<20)
			}
			if err := smtp.Send(context.Background(), Mensagem{Para: "ana@org.gov.br", Assunto: "oi", Corpo: corpo}); err == nil {
				t.Fatal(recusa)
			}
		})
	}
}

func smtpQueRecusa(ln net.Listener, recusa string) {
	c, err := ln.Accept()
	if err != nil {
		return
	}
	defer c.Close()
	if recusa == "banner" {
		return
	}
	br, bw := bufio.NewReader(c), bufio.NewWriter(c)
	responder(bw, "220 ok")
	for {
		linha, err := br.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(linha))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			extra := ""
			if recusa == "starttls" {
				extra = "250-STARTTLS\r\n"
			}
			if recusa == "auth" {
				extra = "250-AUTH PLAIN\r\n"
			}
			responder(bw, "250-localhost\r\n"+extra+"250 OK")
		case strings.HasPrefix(cmd, "STARTTLS"):
			responder(bw, "220 go")
			return
		case strings.HasPrefix(cmd, "AUTH"):
			responder(bw, "535 nao")
			return
		case strings.HasPrefix(cmd, "MAIL"):
			if recusa == "mail" {
				responder(bw, "550 nao")
				return
			}
			responder(bw, "250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			if recusa == "data" {
				responder(bw, "554 nao")
				return
			}
			responder(bw, "354 go")
			if recusa == "corpo" {
				return
			}
			for {
				l, err := br.ReadString('\n')
				if err != nil || strings.TrimSpace(l) == "." {
					break
				}
			}
			responder(bw, "550 nao")
			return
		default:
			responder(bw, "250 OK")
		}
	}
}

func TestCaixaQueFalha(t *testing.T) {
	ctx := context.Background()
	semRede := func(context.Context, string) (net.Conn, error) {
		return nil, errors.New("sem rede")
	}
	if _, err := (IMAP{Discador: semRede}).NaoLidas(ctx); err == nil {
		t.Fatal("ligar")
	}
	if err := (IMAP{Discador: semRede}).MarcarLida(ctx, "7"); err == nil {
		t.Fatal("ligar ao marcar")
	}
	if _, err := (IMAP{Host: "127.0.0.1", Porta: 1}).NaoLidas(ctx); err == nil {
		t.Fatal("tls direto")
	}

	falhaLer := []string{"saudacao", "corta", "login", "select", "busca", "fetch", "literal"}
	for _, etapa := range falhaLer {
		if _, err := (IMAP{Usuario: "nova", Senha: "s", Discador: imapAte(etapa)}).NaoLidas(ctx); err == nil {
			t.Fatal(etapa)
		}
	}
	falhaMarcar := []string{"saudacao", "corta", "login", "select", "store"}
	for _, etapa := range falhaMarcar {
		if err := (IMAP{Usuario: "nova", Senha: "s", Discador: imapAte(etapa)}).MarcarLida(ctx, "7"); err == nil {
			t.Fatal("marcar " + etapa)
		}
	}
	for _, etapa := range []string{"uid", "sem-corpo", "ruim"} {
		cartas, err := (IMAP{Discador: imapAte(etapa)}).NaoLidas(ctx)
		if err != nil || len(cartas) != 0 {
			t.Fatalf("%s: %v %#v", etapa, err, cartas)
		}
	}
	cartas, err := (IMAP{Discador: imapAte("sem-id")}).NaoLidas(ctx)
	if err != nil || len(cartas) != 1 || !strings.HasPrefix(cartas[0].MessageID, "sha256:") {
		t.Fatalf("sem id: %v %#v", err, cartas)
	}
	if _, err := (IMAP{Usuario: strings.Repeat("a", 8000), Discador: escritaQueFalha()}).NaoLidas(ctx); err == nil {
		t.Fatal("escrita longa")
	}
	if _, err := (IMAP{Usuario: "nova", Discador: escritaQueFalha()}).NaoLidas(ctx); err == nil {
		t.Fatal("flush")
	}
}

func imapAte(falha string) func(context.Context, string) (net.Conn, error) {
	return conversaIMAP(func(br *bufio.Reader, bw *bufio.Writer) {
		if falha == "saudacao" {
			return
		}
		responder(bw, "* OK")
		for {
			linha := ler(br)
			campos := strings.Fields(linha)
			if len(campos) < 2 {
				return
			}
			id := campos[0]
			upper := strings.ToUpper(linha)
			cmd := "OUTRO"
			for _, nome := range []string{"LOGIN", "SELECT", "SEARCH", "FETCH", "STORE"} {
				if strings.Contains(upper, nome) {
					cmd = nome
					break
				}
			}
			if falha == "corta" && cmd == "LOGIN" {
				return
			}
			recusa := map[string]string{"login": "LOGIN", "select": "SELECT", "busca": "SEARCH", "fetch": "FETCH", "store": "STORE"}[falha]
			if recusa != "" && recusa == cmd {
				responder(bw, id+" NO")
				return
			}
			if cmd == "SEARCH" && falha == "uid" {
				responder(bw, "* SEARCH abc")
				responder(bw, id+" OK")
				continue
			}
			if cmd == "SEARCH" {
				responder(bw, "* SEARCH 7")
				responder(bw, id+" OK")
				continue
			}
			if cmd == "FETCH" {
				switch falha {
				case "sem-corpo":
					responder(bw, "* 1 FETCH (UID 7 FLAGS)")
					responder(bw, id+" OK")
				case "literal":
					responder(bw, "* 1 FETCH (BODY[] {20}")
					return
				default:
					bruto := "nao e carta"
					if falha == "sem-id" {
						bruto = "From: ana@org.gov.br\r\nSubject: oi\r\n\r\ncorpo"
					}
					responder(bw, fmt.Sprintf("* 1 FETCH (BODY[] {%d}", len(bruto)))
					responder(bw, bruto)
					responder(bw, id+" OK")
				}
				continue
			}
			responder(bw, id+" OK")
		}
	})
}

func escritaQueFalha() func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		cliente, servidor := net.Pipe()
		go func() {
			defer servidor.Close()
			_, _ = servidor.Write([]byte("* OK\r\n"))
			_, _ = io.Copy(io.Discard, servidor)
		}()
		return falhaWrite{cliente}, nil
	}
}

type falhaWrite struct{ net.Conn }

func (falhaWrite) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
