// Package canais leva os avisos da plataforma para fora do sistema (ADR
// 078). O e-mail sai pelo SMTP configurado na tela (Configurações >
// Canais), com a senha cifrada; a variável de ambiente vale só enquanto
// ninguém o configurou. Desligado, o aviso fica só no sino. Cada pessoa
// deixa o contato em que quer receber.
package canais

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Origem da configuração do e-mail.
const (
	OrigemTela     = "tela"
	OrigemAmbiente = "ambiente"
	OrigemNenhuma  = "nenhuma"
)

// ErrSemServidor: o teste pede um servidor configurado.
var ErrSemServidor = errors.New("canais: configure o servidor SMTP antes de testar")

// ErrSemConta: a pessoa ainda não tem conta na plataforma.
var ErrSemConta = errors.New("canais: conta não encontrada")

// InvalidaError é a configuração ou o contato fora das regras.
type InvalidaError struct{ Msg string }

func (e InvalidaError) Error() string { return e.Msg }

// Email é a configuração do SMTP. A senha nunca sai: TemSenha diz se há.
type Email struct {
	Ativo     bool       `json:"ativo"`
	Host      string     `json:"host"`
	Porta     int        `json:"porta"`
	Usuario   string     `json:"usuario"`
	Senha     string     `json:"-"`
	TemSenha  bool       `json:"tem_senha"`
	Remetente string     `json:"remetente"`
	Origem    string     `json:"origem"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UpdatedBy string     `json:"updated_by,omitempty"`
}

// Normalizar apara os campos e põe a porta padrão.
func (e Email) Normalizar() Email {
	e.Host, e.Usuario, e.Remetente = strings.TrimSpace(e.Host), strings.TrimSpace(e.Usuario), strings.TrimSpace(e.Remetente)
	if e.Porta == 0 {
		e.Porta = 587
	}
	return e
}

// Validar confere a configuração. Ligado, precisa de servidor e de um
// remetente válido; desligado, guarda o que houver.
func (e Email) Validar() error {
	switch {
	case utf8.RuneCountInString(e.Host) > 255 || utf8.RuneCountInString(e.Usuario) > 255 || utf8.RuneCountInString(e.Remetente) > 255:
		return InvalidaError{"servidor, usuário e remetente cabem em 255 caracteres"}
	case strings.ContainsAny(e.Host, " /:"):
		return InvalidaError{"o servidor é só o nome ou o IP (sem esquema, barra ou porta)"}
	case e.Porta < 1 || e.Porta > 65535:
		return InvalidaError{"a porta vai de 1 a 65535 (em geral 587, ou 465 para TLS direto)"}
	case e.Remetente != "" && !enderecoValido(e.Remetente):
		return InvalidaError{"o remetente precisa ser um e-mail, como \"Central de Atendimento <atendimento@prefeitura.gov.br>\""}
	case e.Ativo && (e.Host == "" || e.Remetente == ""):
		return InvalidaError{"para ligar o e-mail, informe o servidor e o remetente"}
	}
	return nil
}

func enderecoValido(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && strings.Contains(a.Address, ".")
}

// Contato é onde a pessoa quer receber os avisos.
type Contato struct {
	Email        string     `json:"email"`
	WhatsApp     string     `json:"whatsapp"`
	Respondido   bool       `json:"respondido"`
	RespondidoEm *time.Time `json:"respondido_em,omitempty"`
	// EmailDoSSO é o e-mail que veio do login, quando há: o padrão dos avisos.
	EmailDoSSO string `json:"email_do_sso,omitempty"`
}

// whatsapp é o número no formato internacional (E.164): +55 e o número.
var whatsapp = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// NormalizarContato apara o e-mail e deixa do WhatsApp só os dígitos, com o
// + do formato internacional; sem DDI, vale o do Brasil.
func NormalizarContato(c Contato) (Contato, error) {
	c.Email = strings.TrimSpace(c.Email)
	if c.Email != "" && (utf8.RuneCountInString(c.Email) > 254 || !enderecoValido(c.Email) || strings.ContainsAny(c.Email, "<>")) {
		return c, InvalidaError{"informe um e-mail válido, como nome@exemplo.gov.br"}
	}
	digitos := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, c.WhatsApp)
	switch {
	case digitos == "":
		c.WhatsApp = ""
	case strings.HasPrefix(strings.TrimSpace(c.WhatsApp), "+"):
		c.WhatsApp = "+" + digitos
	default:
		c.WhatsApp = "+55" + digitos
	}
	if c.WhatsApp != "" && !whatsapp.MatchString(c.WhatsApp) {
		return c, InvalidaError{"informe o WhatsApp com DDD, como (66) 99999-9999"}
	}
	return c, nil
}
