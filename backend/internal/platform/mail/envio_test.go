package mail

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestEnvioAceito(t *testing.T) {
	ln := escuta(t)
	corpo := make(chan string, 1)
	go smtpQueAceita(ln, corpo, false)
	host, porta := hostPorta(t, ln)
	err := (SMTP{Host: host, Porta: porta, De: "nova@org.gov.br"}).Send(context.Background(), Mensagem{
		Para: "ana@org.gov.br", Assunto: "Avalie\no chamado", Corpo: "linha 1\nlinha 2",
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := <-corpo
	if !strings.Contains(msg, "To: ana@org.gov.br") || !strings.Contains(msg, "Subject: Avalie o chamado") || !strings.Contains(msg, "linha 1\r\nlinha 2") {
		t.Fatalf("%s", msg)
	}
}

func TestEnvioComAutenticacao(t *testing.T) {
	ln := escuta(t)
	corpo := make(chan string, 1)
	go smtpQueAceita(ln, corpo, true)
	host, porta := hostPorta(t, ln)
	err := (SMTP{Host: host, Porta: porta, Usuario: "nova", Senha: "segredo", De: "nova@org.gov.br"}).Send(
		context.Background(), Mensagem{Para: "ana@org.gov.br", Assunto: "oi", Corpo: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(<-corpo, "Subject: oi") {
		t.Fatal("corpo")
	}
}

func TestEnvioRecusado(t *testing.T) {
	ln := escuta(t)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		br, bw := bufio.NewReader(c), bufio.NewWriter(c)
		responder(bw, "220 ok")
		for {
			linha, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(strings.ToUpper(linha), "RCPT") {
				responder(bw, "550 recusado")
				return
			}
			if strings.HasPrefix(strings.ToUpper(linha), "EHLO") || strings.HasPrefix(strings.ToUpper(linha), "HELO") {
				responder(bw, "250-localhost\r\n250 OK")
				continue
			}
			responder(bw, "250 OK")
		}
	}()
	host, porta := hostPorta(t, ln)
	if err := (SMTP{Host: host, Porta: porta, De: "nova@org.gov.br"}).Send(context.Background(), Mensagem{Para: "ana@org.gov.br", Assunto: "oi", Corpo: "x"}); err == nil {
		t.Fatal("destinatário recusado precisa falhar")
	}
}

func TestCaixaNaoLidaEMarca(t *testing.T) {
	bruto := "From: Ana <ana@org.gov.br>\r\nSubject: [#42] senha\r\nMessage-Id: <abc@org.gov.br>\r\n\r\ntexto do chamado"
	caixa := IMAP{Host: "caixa", Usuario: "nova", Senha: "s", Discador: conversaIMAP(func(br *bufio.Reader, bw *bufio.Writer) {
		responder(bw, "* OK")
		ler(br)
		responder(bw, "a1 OK")
		ler(br)
		responder(bw, "a2 OK")
		ler(br)
		responder(bw, "* SEARCH 7")
		responder(bw, "a3 OK")
		ler(br)
		responder(bw, fmt.Sprintf("* 1 FETCH (UID 7 BODY[] {%d}", len(bruto)))
		fmt.Fprintf(bw, "%s\r\n", bruto)
		bw.Flush()
		responder(bw, "a4 OK")
		ler(br)
		responder(bw, "a5 OK")
	})}
	cartas, err := caixa.NaoLidas(context.Background())
	if err != nil || len(cartas) != 1 || cartas[0].De != "ana@org.gov.br" || cartas[0].MessageID != "abc@org.gov.br" || !strings.Contains(cartas[0].Corpo, "texto do chamado") || cartas[0].UID != "7" {
		t.Fatalf("%v %#v", err, cartas)
	}
	marcada := IMAP{Host: "caixa", Usuario: "nova", Senha: "s", Discador: conversaIMAP(func(br *bufio.Reader, bw *bufio.Writer) {
		responder(bw, "* OK")
		ler(br)
		responder(bw, "a1 OK")
		ler(br)
		responder(bw, "a2 OK")
		linha := ler(br)
		if !strings.Contains(linha, "UID STORE 7") {
			t.Errorf("store: %s", linha)
		}
		responder(bw, "a3 OK")
		ler(br)
		responder(bw, "a4 OK")
	})}
	if err := marcada.MarcarLida(context.Background(), "7"); err != nil {
		t.Fatal(err)
	}
	if err := (IMAP{}).MarcarLida(context.Background(), "abc"); err == nil {
		t.Fatal("uid inválido")
	}
}

func escuta(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func hostPorta(t *testing.T, ln net.Listener) (string, int) {
	t.Helper()
	host, porta, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(porta)
	if err != nil {
		t.Fatal(err)
	}
	return host, n
}

func smtpQueAceita(ln net.Listener, corpo chan<- string, auth bool) {
	c, err := ln.Accept()
	if err != nil {
		return
	}
	defer c.Close()
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
			if auth {
				responder(bw, "250-localhost\r\n250-AUTH PLAIN\r\n250 OK")
			} else {
				responder(bw, "250-localhost\r\n250 OK")
			}
		case strings.HasPrefix(cmd, "AUTH"):
			responder(bw, "235 OK")
		case strings.HasPrefix(cmd, "DATA"):
			responder(bw, "354 go")
			var b strings.Builder
			for {
				l, err := br.ReadString('\n')
				if err != nil || strings.TrimSpace(l) == "." {
					break
				}
				b.WriteString(l)
			}
			corpo <- b.String()
			responder(bw, "250 OK")
		case strings.HasPrefix(cmd, "QUIT"):
			responder(bw, "221 bye")
			return
		default:
			responder(bw, "250 OK")
		}
	}
}

func conversaIMAP(f func(*bufio.Reader, *bufio.Writer)) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		servidor, cliente := net.Pipe()
		go func() {
			defer servidor.Close()
			f(bufio.NewReader(servidor), bufio.NewWriter(servidor))
		}()
		return cliente, nil
	}
}

func responder(w *bufio.Writer, linha string) {
	fmt.Fprintf(w, "%s\r\n", linha)
	_ = w.Flush()
}

func ler(r *bufio.Reader) string {
	linha, _ := r.ReadString('\n')
	return linha
}
