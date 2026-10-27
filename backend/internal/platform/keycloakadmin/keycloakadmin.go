// Package keycloakadmin lê as pessoas e os grupos da API de administração
// do Keycloak (ADR 077), com a conta de serviço do client do Projeto Nova
// (client_credentials). Só leitura: a conta precisa da função view-users
// do realm-management, e de nenhuma outra.
package keycloakadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNaoConfigurado: sem issuer, client ou segredo não há o que ler.
var ErrNaoConfigurado = errors.New("keycloakadmin: SSO sem client e segredo configurados")

// ErrResposta é a resposta inesperada do Keycloak.
var ErrResposta = errors.New("keycloakadmin: resposta inesperada")

// Config é o client do Projeto Nova no realm.
type Config struct {
	IssuerURL, ClientID, ClientSecret string
}

// Usuario é uma pessoa do realm, com os nomes dos grupos.
type Usuario struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Enabled   bool     `json:"enabled"`
	Grupos    []string `json:"-"`
}

// Nome é o nome para mostrar, como o token traz em "name".
func (u Usuario) Nome() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Cliente fala com o Keycloak.
type Cliente struct {
	HTTP *http.Client
	// Pagina é quantas pessoas vêm por chamada.
	Pagina int
}

// Novo cria o cliente com um tempo limite por chamada.
func Novo() *Cliente {
	return &Cliente{HTTP: &http.Client{Timeout: 30 * time.Second}, Pagina: 200}
}

// enderecos tira do issuer (https://sso/realms/x) o endpoint do token e a
// base da administração (https://sso/admin/realms/x).
func enderecos(issuer string) (token, admin string, err error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(issuer), "/"))
	if err != nil || u.Host == "" {
		return "", "", fmt.Errorf("%w: issuer %q", ErrNaoConfigurado, issuer)
	}
	i := strings.LastIndex(u.Path, "/realms/")
	if i < 0 || i+len("/realms/") >= len(u.Path) {
		return "", "", fmt.Errorf("%w: issuer sem /realms/<realm>: %q", ErrNaoConfigurado, issuer)
	}
	base := u.Scheme + "://" + u.Host + u.Path[:i]
	return u.String() + "/protocol/openid-connect/token", base + "/admin" + u.Path[i:], nil
}

// Usuarios lê todas as pessoas do realm, página a página, com os grupos.
func (c *Cliente) Usuarios(ctx context.Context, cfg Config) ([]Usuario, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, ErrNaoConfigurado
	}
	tokenURL, admin, err := enderecos(cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	token, err := c.token(ctx, tokenURL, cfg)
	if err != nil {
		return nil, err
	}
	pagina := c.Pagina
	if pagina <= 0 {
		pagina = 200
	}
	out := []Usuario{}
	for primeiro := 0; ; primeiro += pagina {
		var lote []Usuario
		if err := c.get(ctx, token, fmt.Sprintf("%s/users?first=%d&max=%d&briefRepresentation=false", admin, primeiro, pagina), &lote); err != nil {
			return nil, err
		}
		for _, u := range lote {
			var grupos []struct {
				Nome string `json:"name"`
			}
			if err := c.get(ctx, token, admin+"/users/"+url.PathEscape(u.ID)+"/groups?briefRepresentation=true", &grupos); err != nil {
				return nil, err
			}
			for _, g := range grupos {
				u.Grupos = append(u.Grupos, g.Nome)
			}
			out = append(out, u)
		}
		if len(lote) < pagina {
			return out, nil
		}
	}
}

// token troca o client e o segredo pelo token da conta de serviço.
func (c *Cliente) token(ctx context.Context, endpoint string, cfg Config) (string, error) {
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {cfg.ClientID}, "client_secret": {cfg.ClientSecret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var out struct {
		Token string `json:"access_token"`
	}
	if err := c.fazer(req, &out); err != nil {
		return "", fmt.Errorf("token da conta de serviço (habilite Service accounts no client e dê view-users): %w", err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("%w: token vazio", ErrResposta)
	}
	return out.Token, nil
}

func (c *Cliente) get(ctx context.Context, token, endereco string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return c.fazer(req, out)
}

func (c *Cliente) fazer(req *http.Request, out any) error {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s %d", ErrResposta, req.URL.Path, resp.StatusCode)
	}
	if err := json.Unmarshal(corpo, out); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrResposta, req.URL.Path, err)
	}
	return nil
}
