package glpi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Categoria é uma categoria de chamado do GLPI.
type Categoria struct {
	ID   int    `json:"id"`
	Nome string `json:"completename"`
}

// Chamado é o que o importador lê de um chamado do GLPI.
type Chamado struct {
	ID          int    `json:"id"`
	Titulo      string `json:"name"`
	Conteudo    string `json:"content"`
	Abertura    string `json:"date"`
	Solucao     string `json:"solvedate"`
	Fechamento  string `json:"closedate"`
	Status      int    `json:"status"`
	Tipo        int    `json:"type"`
	Urgencia    int    `json:"urgency"`
	Impacto     int    `json:"impact"`
	CategoriaID int    `json:"itilcategories_id"`
}

// Participante liga uma pessoa ao chamado: 1 requerente, 2 técnico.
type Participante struct {
	PessoaID int `json:"users_id"`
	Papel    int `json:"type"`
}

// Acompanhamento é uma mensagem do chamado.
type Acompanhamento struct {
	ID      int    `json:"id"`
	Texto   string `json:"content"`
	Em      string `json:"date"`
	AutorID int    `json:"users_id"`
	Privado int    `json:"is_private"`
}

// Satisfacao é a pesquisa respondida.
type Satisfacao struct {
	Nota       *int   `json:"satisfaction"`
	Comentario string `json:"comment"`
	Em         string `json:"date_answered"`
}

// Pessoa é um usuário do GLPI, com o e-mail principal.
type Pessoa struct {
	ID        int    `json:"id"`
	Login     string `json:"name"`
	Nome      string `json:"firstname"`
	Sobrenome string `json:"realname"`
	Email     string `json:"-"`
}

// NomeCompleto é o nome para mostrar, ou o login.
func (p Pessoa) NomeCompleto() string {
	if n := strings.TrimSpace(p.Nome + " " + p.Sobrenome); n != "" {
		return n
	}
	return p.Login
}

// Fonte é o que o importador lê. O Cliente lê a API REST; os testes usam
// uma fonte em memória.
type Fonte interface {
	Categorias(ctx context.Context) ([]Categoria, error)
	// Chamados lê uma página (de, ate inclusive) e diz o total.
	Chamados(ctx context.Context, de, ate int) ([]Chamado, int, error)
	Participantes(ctx context.Context, chamado int) ([]Participante, error)
	Acompanhamentos(ctx context.Context, chamado int) ([]Acompanhamento, error)
	Satisfacao(ctx context.Context, chamado int) (*Satisfacao, error)
	Pessoa(ctx context.Context, id int) (Pessoa, error)
}

// ErrGLPI é a resposta inesperada da API.
var ErrGLPI = errors.New("glpi: resposta inesperada")

// Cliente lê a API REST do GLPI (apirest.php). Abrir troca o token de
// usuário por uma sessão; Fechar a encerra.
type Cliente struct {
	Base      string
	AppToken  string
	UserToken string
	HTTP      *http.Client
	sessao    string
}

// NovoCliente prepara o cliente para a URL da API (…/apirest.php).
func NovoCliente(base, appToken, userToken string) *Cliente {
	return &Cliente{Base: strings.TrimRight(base, "/"), AppToken: appToken, UserToken: userToken, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

// Abrir abre a sessão.
func (c *Cliente) Abrir(ctx context.Context) error {
	var out struct {
		Sessao string `json:"session_token"`
	}
	if _, err := c.get(ctx, "/initSession", &out); err != nil {
		return err
	}
	if out.Sessao == "" {
		return fmt.Errorf("%w: sessão vazia", ErrGLPI)
	}
	c.sessao = out.Sessao
	return nil
}

// Fechar encerra a sessão. O erro só é registrado por quem chama.
func (c *Cliente) Fechar(ctx context.Context) error {
	_, err := c.get(ctx, "/killSession", nil)
	c.sessao = ""
	return err
}

func (c *Cliente) Categorias(ctx context.Context) ([]Categoria, error) {
	var out []Categoria
	_, err := c.get(ctx, "/ITILCategory?range=0-9999", &out)
	return out, err
}

func (c *Cliente) Chamados(ctx context.Context, de, ate int) ([]Chamado, int, error) {
	var out []Chamado
	h, err := c.get(ctx, fmt.Sprintf("/Ticket?range=%d-%d&sort=1&order=ASC&expand_dropdowns=false", de, ate), &out)
	if err != nil {
		return nil, 0, err
	}
	return out, total(h.Get("Content-Range"), len(out)), nil
}

func (c *Cliente) Participantes(ctx context.Context, chamado int) ([]Participante, error) {
	var out []Participante
	_, err := c.get(ctx, fmt.Sprintf("/Ticket/%d/Ticket_User", chamado), &out)
	return out, err
}

func (c *Cliente) Acompanhamentos(ctx context.Context, chamado int) ([]Acompanhamento, error) {
	var out []Acompanhamento
	_, err := c.get(ctx, fmt.Sprintf("/Ticket/%d/ITILFollowup?range=0-999", chamado), &out)
	return out, err
}

func (c *Cliente) Satisfacao(ctx context.Context, chamado int) (*Satisfacao, error) {
	var out []Satisfacao
	if _, err := c.get(ctx, fmt.Sprintf("/Ticket/%d/TicketSatisfaction", chamado), &out); err != nil {
		return nil, err
	}
	for _, s := range out {
		if s.Nota != nil {
			return &s, nil
		}
	}
	return nil, nil
}

func (c *Cliente) Pessoa(ctx context.Context, id int) (Pessoa, error) {
	var p Pessoa
	if _, err := c.get(ctx, fmt.Sprintf("/User/%d", id), &p); err != nil {
		return Pessoa{}, err
	}
	var emails []struct {
		Email     string `json:"email"`
		Principal int    `json:"is_default"`
	}
	if _, err := c.get(ctx, fmt.Sprintf("/User/%d/UserEmail", id), &emails); err != nil {
		return Pessoa{}, err
	}
	for _, e := range emails {
		if p.Email == "" || e.Principal == 1 {
			p.Email = e.Email
		}
	}
	return p, nil
}

// get chama a API e decodifica o corpo. 404 e lista vazia (206/200 com
// []) são respostas válidas; o GLPI também responde 400 com
// ERROR_RANGE_EXCEED_TOTAL quando a página passa do fim.
func (c *Cliente) get(ctx context.Context, caminho string, out any) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+caminho, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AppToken != "" {
		req.Header.Set("App-Token", c.AppToken)
	}
	if c.sessao != "" {
		req.Header.Set("Session-Token", c.sessao)
	} else {
		req.Header.Set("Authorization", "user_token "+c.UserToken)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	switch {
	case resp.StatusCode == http.StatusNotFound,
		resp.StatusCode == http.StatusBadRequest && strings.Contains(string(corpo), "ERROR_RANGE_EXCEED_TOTAL"):
		return resp.Header, nil
	case resp.StatusCode >= 300:
		return nil, fmt.Errorf("%w: %s %d", ErrGLPI, caminho, resp.StatusCode)
	}
	if out == nil || len(corpo) == 0 {
		return resp.Header, nil
	}
	if err := json.Unmarshal(corpo, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrGLPI, caminho, err)
	}
	return resp.Header, nil
}

// total lê "0-49/1234" do Content-Range. Sem o cabeçalho, a página é tudo.
func total(contentRange string, pagina int) int {
	if i := strings.LastIndex(contentRange, "/"); i >= 0 {
		if n, err := strconv.Atoi(contentRange[i+1:]); err == nil {
			return n
		}
	}
	return pagina
}
