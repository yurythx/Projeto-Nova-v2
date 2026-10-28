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
	"sync"
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
	// Paralelo é quantas pessoas têm os grupos lidos ao mesmo tempo.
	Paralelo int
}

// Padrões medidos num realm federado ao AD com 7.740 pessoas: a página
// resumida de 50 sai em meio segundo (a completa consulta o LDAP pessoa a
// pessoa e passa de 3 minutos), e os grupos de uma pessoa levam 0,3 s.
const (
	paginaPadrao   = 50
	paraleloPadrao = 8
)

// tempoChamada fica acima do aborto de transação do Keycloak (5 minutos):
// fora do cache, uma página às vezes faz o provedor LDAP varrer o AD
// inteiro e leva 4 minutos. Depois dela as seguintes saem do cache.
const tempoChamada = 10 * time.Minute

// Novo cria o cliente com um tempo limite por chamada.
func Novo() *Cliente {
	return &Cliente{HTTP: &http.Client{Timeout: tempoChamada}, Pagina: paginaPadrao, Paralelo: paraleloPadrao}
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

// Usuarios lê todas as pessoas do realm, página a página (representação
// resumida), com os grupos lidos em paralelo, e entrega cada página a lote
// assim que chega: a leitura toda pode levar horas, e quem chama grava e
// mostra o andamento no caminho. Um erro de lote para a leitura. O token é
// renovado a cada página: o da conta de serviço dura minutos.
func (c *Cliente) Usuarios(ctx context.Context, cfg Config, lote func([]Usuario) error) error {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return ErrNaoConfigurado
	}
	tokenURL, admin, err := enderecos(cfg.IssuerURL)
	if err != nil {
		return err
	}
	pagina, paralelo := c.Pagina, c.Paralelo
	if pagina <= 0 {
		pagina = paginaPadrao
	}
	if paralelo <= 0 {
		paralelo = paraleloPadrao
	}
	// Federado ao AD, o Keycloak devolve páginas curtas no meio da lista:
	// o fim é o total que ele conta (na primeira volta), não a primeira
	// página incompleta.
	total := 1
	for primeiro := 0; primeiro < total; primeiro += pagina {
		token, err := c.token(ctx, tokenURL, cfg)
		if err != nil {
			return err
		}
		if primeiro == 0 {
			if err := c.get(ctx, token, admin+"/users/count", &total); err != nil {
				return err
			}
		}
		var pessoas []Usuario
		err = c.get(ctx, token, pagina0(admin, primeiro, pagina), &pessoas)
		if errors.Is(err, errDecodificar) {
			// Federado ao AD, o Keycloak às vezes emenda o JSON de uma página
			// inteira, sempre no mesmo ponto; lida uma a uma, cada pessoa vem
			// certa.
			pessoas, err = c.umaAUma(ctx, token, admin, primeiro, pagina)
		}
		if err != nil {
			// O total também oscila, e a página além do fim real volta 400:
			// contado de novo, se a lista acabou antes, o fim é normal.
			var agora int
			if primeiro > 0 && c.get(ctx, token, admin+"/users/count", &agora) == nil && agora <= primeiro {
				return nil
			}
			return err
		}
		if err := c.grupos(ctx, token, admin, pessoas, paralelo); err != nil {
			return err
		}
		if err := lote(pessoas); err != nil {
			return err
		}
	}
	return nil
}

// Ativo lê a pessoa completa, que o Keycloak valida no AD, e diz se ela está
// ativa. A lista resumida às vezes informa estado desatualizado (cada
// réplica do Keycloak com o seu cache): a mudança de estado se confirma aqui.
func (c *Cliente) Ativo(ctx context.Context, cfg Config, id string) (bool, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return false, ErrNaoConfigurado
	}
	tokenURL, admin, err := enderecos(cfg.IssuerURL)
	if err != nil {
		return false, err
	}
	token, err := c.token(ctx, tokenURL, cfg)
	if err != nil {
		return false, err
	}
	var u struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.get(ctx, token, admin+"/users/"+url.PathEscape(id), &u); err != nil {
		return false, err
	}
	return u.Enabled, nil
}

// pagina0 é o endereço de uma página da lista resumida.
func pagina0(admin string, primeiro, max int) string {
	return fmt.Sprintf("%s/users?first=%d&max=%d&briefRepresentation=true", admin, primeiro, max)
}

// umaAUma lê a página pessoa por pessoa; o fim da lista encerra antes.
func (c *Cliente) umaAUma(ctx context.Context, token, admin string, primeiro, pagina int) ([]Usuario, error) {
	out := []Usuario{}
	for i := range pagina {
		var uma []Usuario
		if err := c.get(ctx, token, pagina0(admin, primeiro+i, 1), &uma); err != nil {
			return nil, err
		}
		if len(uma) == 0 {
			break
		}
		out = append(out, uma...)
	}
	return out, nil
}

// grupos preenche os grupos de cada pessoa do lote, paralelo de cada vez.
// O primeiro erro vence; os demais pedidos param com ele.
func (c *Cliente) grupos(ctx context.Context, token, admin string, lote []Usuario, paralelo int) error {
	ctx, parar := context.WithCancel(ctx)
	defer parar()
	vagas := make(chan struct{}, paralelo)
	erros := make(chan error, len(lote))
	var espera sync.WaitGroup
	for i := range lote {
		espera.Add(1)
		vagas <- struct{}{}
		go func(u *Usuario) {
			defer func() { <-vagas; espera.Done() }()
			var grupos []struct {
				Nome string `json:"name"`
			}
			if err := c.get(ctx, token, admin+"/users/"+url.PathEscape(u.ID)+"/groups?briefRepresentation=true", &grupos); err != nil {
				erros <- err
				parar()
				return
			}
			for _, g := range grupos {
				u.Grupos = append(u.Grupos, g.Nome)
			}
		}(&lote[i])
	}
	espera.Wait()
	close(erros)
	return <-erros
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

// Tentativas e Espera: uma leitura que falha por resposta inválida ou rede
// é repetida, com espera crescente. Federado ao AD, o Keycloak às vezes
// devolve uma página corrompida; uma só não pode derrubar a sincronização
// inteira. Recusa (401, 403, 404) não se repete.
var (
	Tentativas = 3
	Espera     = 2 * time.Second
)

// errDefinitivo marca a falha que repetir não resolve; errDecodificar, a
// resposta que chegou mas não é o JSON esperado.
var (
	errDefinitivo  = errors.New("keycloakadmin: falha definitiva")
	errDecodificar = errors.New("keycloakadmin: resposta ilegível")
)

func (c *Cliente) get(ctx context.Context, token, endereco string, out any) error {
	var err error
	for i := range Tentativas {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(Espera * time.Duration(i)):
			}
		}
		if err = c.getUmaVez(ctx, token, endereco, out); err == nil || errors.Is(err, errDefinitivo) || ctx.Err() != nil {
			return err
		}
	}
	return err
}

func (c *Cliente) getUmaVez(ctx context.Context, token, endereco string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", errDefinitivo, err)
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
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return fmt.Errorf("%w: %w: %s %d", ErrResposta, errDefinitivo, req.URL.Path, resp.StatusCode)
	default:
		return fmt.Errorf("%w: %s %d", ErrResposta, req.URL.Path, resp.StatusCode)
	}
	if err := json.Unmarshal(corpo, out); err != nil {
		return fmt.Errorf("%w: %w: %s: %v", ErrResposta, errDecodificar, req.URL.Path, err)
	}
	return nil
}
