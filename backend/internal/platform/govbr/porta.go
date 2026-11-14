// Package govbr é a porta do Login Único do contribuinte (OAuth2 com PKCE).
// O servidor da prefeitura continua no Keycloak. Sem client id, a porta
// não inventa sessão: informa que a credencial ainda não foi configurada.
package govbr

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	AmbienteHomologacao = "homologacao"
	AmbienteProducao    = "producao"
)

// ErrSemCredencial: o Login Único ainda não tem client id e segredo.
var ErrSemCredencial = errors.New("govbr: credencial do Login Único ainda não configurada")

// Config são os dados do aplicativo no Login Único.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Ambiente     string
}

// Pronta diz se dá para começar o login.
func (c Config) Pronta() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != "" && strings.TrimSpace(c.RedirectURL) != ""
}

func (c Config) base() string {
	if strings.EqualFold(strings.TrimSpace(c.Ambiente), AmbienteProducao) {
		return "https://sso.acesso.gov.br"
	}
	return "https://sso.staging.acesso.gov.br"
}

// Desafio é o par PKCE e o state que o navegador precisa guardar.
type Desafio struct {
	State         string `json:"state"`
	Nonce         string `json:"nonce"`
	Verifier      string `json:"verifier"`
	CodeChallenge string `json:"code_challenge"`
	URL           string `json:"url"`
}

// Iniciar monta a URL de autorização. Sem credencial, devolve ErrSemCredencial.
func (c Config) Iniciar() (Desafio, error) {
	if !c.Pronta() {
		return Desafio{}, ErrSemCredencial
	}
	verifier, err := aleatorio(32)
	if err != nil {
		return Desafio{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	state, err := aleatorio(16)
	if err != nil {
		return Desafio{}, err
	}
	nonce, err := aleatorio(16)
	if err != nil {
		return Desafio{}, err
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURL)
	q.Set("scope", "openid email profile govbr_confiabilidades")
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return Desafio{State: state, Nonce: nonce, Verifier: verifier, CodeChallenge: challenge, URL: c.base() + "/authorize?" + q.Encode()}, nil
}

// Identidade é o que o userinfo devolve e o protocolo pode guardar.
type Identidade struct {
	Sub   string `json:"sub"`
	Nome  string `json:"name"`
	Email string `json:"email"`
	CPF   string `json:"cpf"`
}

// Trocar entrega o code no token endpoint e lê o userinfo.
// O cliente HTTP é injetado para o teste não sair da máquina.
func (c Config) Trocar(cli *http.Client, code, verifier string) (Identidade, error) {
	if !c.Pronta() {
		return Identidade{}, ErrSemCredencial
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(verifier) == "" {
		return Identidade{}, errors.New("govbr: code e verifier são obrigatórios")
	}
	if cli == nil {
		cli = &http.Client{Timeout: 10 * time.Second}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURL)
	form.Set("code_verifier", verifier)
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	req, err := http.NewRequest(http.MethodPost, enderecoToken(c), strings.NewReader(form.Encode()))
	if err != nil {
		return Identidade{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(req)
	if err != nil {
		return Identidade{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return Identidade{}, fmt.Errorf("govbr: token %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return Identidade{}, errors.New("govbr: resposta de token sem access_token")
	}
	ureq, err := http.NewRequest(http.MethodGet, enderecoUserinfo(c), nil)
	if err != nil {
		return Identidade{}, err
	}
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uresp, err := cli.Do(ureq)
	if err != nil {
		return Identidade{}, err
	}
	defer uresp.Body.Close()
	ubody, _ := io.ReadAll(io.LimitReader(uresp.Body, 1<<20))
	if uresp.StatusCode != http.StatusOK {
		return Identidade{}, fmt.Errorf("govbr: userinfo %d", uresp.StatusCode)
	}
	var id Identidade
	if err := json.Unmarshal(ubody, &id); err != nil || id.Sub == "" {
		return Identidade{}, errors.New("govbr: userinfo sem sujeito")
	}
	return id, nil
}

var (
	lerAleatorio     = rand.Read
	enderecoToken    = func(c Config) string { return c.base() + "/token" }
	enderecoUserinfo = func(c Config) string { return c.base() + "/userinfo" }
)

func aleatorio(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := lerAleatorio(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
