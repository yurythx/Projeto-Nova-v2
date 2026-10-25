// Package mail envia aviso por SMTP e lê a caixa IMAP. Os dois ficam
// desligados quando o host não está configurado: o sino continua valendo.
package mail

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Mensagem é um aviso em texto puro.
type Mensagem struct {
	Para, Assunto, Corpo string
}

// Remetente envia uma mensagem. Host vazio não envia e não é erro.
type Remetente interface {
	Send(ctx context.Context, m Mensagem) error
}

// SMTP fala com um relay. Porta 465 abre TLS direto; as outras usam o
// STARTTLS que o servidor anunciar.
type SMTP struct {
	Host, Usuario, Senha, De string
	Porta                    int
}

// Send entrega a mensagem. Sem host ou sem destinatário, não faz nada.
func (s SMTP) Send(ctx context.Context, m Mensagem) error {
	if s.Host == "" || strings.TrimSpace(m.Para) == "" {
		return nil
	}
	porta := s.Porta
	if porta == 0 {
		porta = 587
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(porta))
	body := strings.ReplaceAll(m.Corpo, "\n", "\r\n")
	raw := []byte("From: " + s.De + "\r\nTo: " + m.Para + "\r\nSubject: " + cabecalho(m.Assunto) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n")
	var auth smtp.Auth
	if s.Usuario != "" {
		auth = smtp.PlainAuth("", s.Usuario, s.Senha, s.Host)
	}
	dial := func(network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 10 * time.Second}
		if porta == 465 {
			return tls.DialWithDialer(&d, network, address, &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12})
		}
		return d.DialContext(ctx, network, address)
	}
	return send(addr, auth, s.De, []string{m.Para}, raw, dial)
}

// send é smtp.SendMail com o discador injetável no teste.
var send = func(addr string, auth smtp.Auth, from string, to []string, msg []byte, dial func(string, string) (net.Conn, error)) error {
	c, err := dial("tcp", addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(c, hostDe(addr))
	if err != nil {
		_ = c.Close()
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: hostDe(addr), MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rec := range to {
		if err := client.Rcpt(rec); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func hostDe(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func cabecalho(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

// Carta é uma mensagem ainda não lida da caixa.
type Carta struct {
	UID, De, Assunto, Corpo, MessageID string
}

// Caixa é a entrada de e-mail. MarcarLida só depois de a carta ter sido
// aceita ou recusada de forma definitiva.
type Caixa interface {
	NaoLidas(ctx context.Context) ([]Carta, error)
	MarcarLida(ctx context.Context, uid string) error
}

// IMAP lê a caixa. O teste troca Discador por um servidor de mentira.
type IMAP struct {
	Host, Usuario, Senha string
	Porta                int
	Discador             func(ctx context.Context, addr string) (net.Conn, error)
}

// NaoLidas devolve as cartas UNSEEN, sem marcá-las.
func (c IMAP) NaoLidas(ctx context.Context) ([]Carta, error) {
	conn, err := c.ligar(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	if _, err := readIMAP(r); err != nil {
		return nil, err
	}
	if _, err := comando(r, w, "a1", "LOGIN "+quoted(c.Usuario)+" "+quoted(c.Senha)); err != nil {
		return nil, err
	}
	if _, err := comando(r, w, "a2", "SELECT INBOX"); err != nil {
		return nil, err
	}
	linhas, err := comando(r, w, "a3", "UID SEARCH UNSEEN")
	if err != nil {
		return nil, err
	}
	var uids []string
	for _, linha := range linhas {
		if strings.HasPrefix(linha, "* SEARCH") {
			uids = strings.Fields(strings.TrimPrefix(linha, "* SEARCH"))
		}
	}
	var out []Carta
	for _, uid := range uids {
		if _, err := strconv.Atoi(uid); err != nil {
			continue
		}
		fetched, err := comando(r, w, "a4", "UID FETCH "+uid+" (BODY.PEEK[])")
		if err != nil {
			return nil, err
		}
		carta, ok := cartaDe(uid, fetched)
		if ok {
			out = append(out, carta)
		}
	}
	_, _ = comando(r, w, "a5", "LOGOUT")
	return out, nil
}

// MarcarLida grava a flag Seen no UID.
func (c IMAP) MarcarLida(ctx context.Context, uid string) error {
	if _, err := strconv.Atoi(uid); err != nil {
		return fmt.Errorf("mail: uid inválido")
	}
	conn, err := c.ligar(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	if _, err := readIMAP(r); err != nil {
		return err
	}
	if _, err := comando(r, w, "a1", "LOGIN "+quoted(c.Usuario)+" "+quoted(c.Senha)); err != nil {
		return err
	}
	if _, err := comando(r, w, "a2", "SELECT INBOX"); err != nil {
		return err
	}
	if _, err := comando(r, w, "a3", "UID STORE "+uid+" +FLAGS (\\Seen)"); err != nil {
		return err
	}
	_, _ = comando(r, w, "a4", "LOGOUT")
	return nil
}

func (c IMAP) ligar(ctx context.Context) (net.Conn, error) {
	porta := c.Porta
	if porta == 0 {
		porta = 993
	}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(porta))
	if c.Discador != nil {
		return c.Discador(ctx, addr)
	}
	d := tls.Dialer{Config: &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}}
	return d.DialContext(ctx, "tcp", addr)
}

func comando(r *bufio.Reader, w *bufio.Writer, tag, cmd string) ([]string, error) {
	if _, err := fmt.Fprintf(w, "%s %s\r\n", tag, cmd); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}
	var linhas []string
	for {
		linha, err := readIMAP(r)
		if err != nil {
			return nil, err
		}
		linhas = append(linhas, linha)
		if strings.HasPrefix(linha, tag+" ") {
			if strings.Contains(linha, " OK") {
				return linhas, nil
			}
			return nil, fmt.Errorf("mail: %s", linha)
		}
	}
}

func readIMAP(r *bufio.Reader) (string, error) {
	linha, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	linha = strings.TrimRight(linha, "\r\n")
	if i := strings.LastIndex(linha, "{"); i >= 0 && strings.HasSuffix(linha, "}") {
		n, conv := strconv.Atoi(linha[i+1 : len(linha)-1])
		if conv == nil && n >= 0 && n <= 1<<20 {
			buf := make([]byte, n)
			if _, err := io.ReadFull(r, buf); err != nil {
				return "", err
			}
			_, _ = r.ReadString('\n')
			return linha + "\n" + string(buf), nil
		}
	}
	return linha, nil
}

func cartaDe(uid string, linhas []string) (Carta, bool) {
	var bruto string
	for _, linha := range linhas {
		if i := strings.Index(linha, "\n"); i >= 0 && strings.Contains(linha[:i], "BODY[]") {
			bruto = linha[i+1:]
		}
	}
	if bruto == "" {
		return Carta{}, false
	}
	msg, err := mail.ReadMessage(strings.NewReader(bruto))
	if err != nil {
		return Carta{}, false
	}
	corpo, _ := io.ReadAll(io.LimitReader(msg.Body, 4000))
	de := msg.Header.Get("From")
	if end, err := mail.ParseAddress(de); err == nil {
		de = end.Address
	}
	id := strings.Trim(msg.Header.Get("Message-Id"), "<> ")
	assunto := msg.Header.Get("Subject")
	if id == "" {
		sum := sha256.Sum256([]byte(de + "\n" + assunto + "\n" + string(corpo)))
		id = "sha256:" + hex.EncodeToString(sum[:])
	}
	return Carta{UID: uid, De: de, Assunto: assunto, Corpo: string(corpo), MessageID: id}, true
}

func quoted(s string) string {
	return "\"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(s) + "\""
}
