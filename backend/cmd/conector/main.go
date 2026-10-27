// Command conector roda na rede da origem (ADR 076): lê o GLPI e envia os
// dados à área de preparo do Projeto Nova, pela API, com o token da
// conexão. As credenciais do GLPI ficam nesta máquina; o servidor nunca as
// recebe.
//
// Ele atende a extração pedida na tela das integrações:
//
//	NOVA_API=https://api.exemplo NOVA_TOKEN=nova_con_... \
//	GLPI_URL=http://helpdesk/apirest.php GLPI_APP_TOKEN=... GLPI_LOGIN=... GLPI_PASSWORD=... \
//	  conector             # atende a extração pedida, se houver, e sai
//	  conector --vigiar    # espera os pedidos, de minuto em minuto
//
// GLPI_USER_TOKEN entra no lugar do login e da senha; GLPI_PERFIL escolhe
// o perfil (padrão: Super-Admin, se a conta tiver).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/glpi"
)

// opcoes é o que o ambiente e a linha de comando dizem.
type opcoes struct {
	api, token                   string
	glpiURL, appToken, userToken string
	login, senha, perfil         string
	vigiar                       bool
	intervalo                    time.Duration
}

func main() {
	op, err := ler(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conector:", err)
		os.Exit(2)
	}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()
	if err := rodar(ctx, op, abrirGLPI, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "conector:", err)
		os.Exit(1)
	}
}

func ler(args []string, env func(string) string) (opcoes, error) {
	fs := flag.NewFlagSet("conector", flag.ContinueOnError)
	vigiar := fs.Bool("vigiar", false, "esperar os pedidos de extração, em vez de sair")
	intervalo := fs.Duration("intervalo", time.Minute, "de quanto em quanto procurar pedido, com --vigiar")
	if err := fs.Parse(args); err != nil {
		return opcoes{}, err
	}
	api, err := enderecoDaAPI(env("NOVA_API"))
	if err != nil {
		return opcoes{}, err
	}
	op := opcoes{api: api, token: strings.TrimSpace(env("NOVA_TOKEN")), glpiURL: env("GLPI_URL"), appToken: env("GLPI_APP_TOKEN"),
		userToken: env("GLPI_USER_TOKEN"), login: env("GLPI_LOGIN"), senha: env("GLPI_PASSWORD"), perfil: env("GLPI_PERFIL"),
		vigiar: *vigiar, intervalo: *intervalo}
	switch {
	case !strings.HasPrefix(op.token, "nova_con_"):
		return opcoes{}, errors.New("NOVA_TOKEN é o token do conector, emitido na tela da conexão (nova_con_...)")
	case op.glpiURL == "" || (op.userToken == "" && (op.login == "" || op.senha == "")):
		return opcoes{}, errors.New("defina GLPI_URL e GLPI_USER_TOKEN, ou GLPI_LOGIN e GLPI_PASSWORD")
	case op.intervalo < time.Second:
		return opcoes{}, errors.New("--intervalo de pelo menos 1s")
	}
	return op, nil
}

// enderecoDaAPI confere NOVA_API. O token vai no cabeçalho: fora da
// própria máquina, só por HTTPS.
func enderecoDaAPI(bruto string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("NOVA_API precisa ser um endereço como https://api.exemplo")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", fmt.Errorf("NOVA_API precisa usar https (http só para localhost)")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// abridor abre a sessão no GLPI; os testes trocam por um GLPI falso.
type abridor func(ctx context.Context, op opcoes) (glpi.Extraivel, func(), error)

func abrirGLPI(ctx context.Context, op opcoes) (glpi.Extraivel, func(), error) {
	cli := glpi.NovoCliente(op.glpiURL, op.appToken, op.userToken)
	cli.Login, cli.Senha, cli.Perfil = op.login, op.senha, op.perfil
	if err := cli.Abrir(ctx); err != nil {
		return nil, nil, fmt.Errorf("sessão no GLPI: %w", err)
	}
	return cli, func() { _ = cli.Fechar(context.WithoutCancel(ctx)) }, nil
}

// rodar atende o pedido e, com --vigiar, continua esperando.
func rodar(ctx context.Context, op opcoes, abrir abridor, saida io.Writer) error {
	api := &cliente{base: op.api + "/api/v1/integracoes/conector", token: op.token, http: &http.Client{Timeout: 2 * time.Minute}}
	for {
		atendeu, err := atender(ctx, api, op, abrir, saida)
		if err != nil || !op.vigiar {
			if err == nil && !atendeu {
				_, _ = fmt.Fprintln(saida, "Nenhuma extração pedida. Peça na tela da conexão e rode de novo.")
			}
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(op.intervalo):
		}
	}
}

// tarefa é o que a API devolve do pedido.
type tarefa struct {
	ID string `json:"id"`
}

// atender pega a extração pendente e a executa.
func atender(ctx context.Context, api *cliente, op opcoes, abrir abridor, saida io.Writer) (bool, error) {
	var t tarefa
	code, err := api.chamar(ctx, http.MethodPost, "/tarefa", nil, &t)
	if err != nil || code == http.StatusNoContent {
		return false, err
	}
	_, _ = fmt.Fprintln(saida, "Extração pedida:", t.ID)
	trabalho, parar := context.WithCancel(ctx)
	defer parar()
	resumo, falha := extrair(trabalho, api, op, abrir, t, parar)
	if ctx.Err() != nil {
		return true, nil
	}
	fim := map[string]any{"relatorio": resumo}
	if falha != nil {
		fim["erro"] = falha.Error()
		_, _ = fmt.Fprintln(saida, "Falhou:", falha)
	} else {
		_, _ = fmt.Fprintf(saida, "Concluída: %d chamados, %d novos ou alterados, %d pessoas.\n", resumo.Chamados, resumo.Novos, resumo.Pessoas)
	}
	_, err = api.chamar(context.WithoutCancel(ctx), http.MethodPost, "/tarefas/"+t.ID+"/fim", fim, nil)
	return true, err
}

func extrair(ctx context.Context, api *cliente, op opcoes, abrir abridor, t tarefa, parar func()) (glpi.ResumoExtracao, error) {
	origem, fechar, err := abrir(ctx, op)
	if err != nil {
		return glpi.ResumoExtracao{}, err
	}
	defer fechar()
	destino := &destinoAPI{api: api}
	progresso := func(feitos, total int) {
		var r struct {
			Estado string `json:"estado"`
		}
		if _, err := api.chamar(ctx, http.MethodPost, "/tarefas/"+t.ID+"/progresso", map[string]int{"feitos": feitos, "total": total}, &r); err == nil &&
			r.Estado == "cancelada" {
			parar()
		}
	}
	return glpi.Extrair(ctx, origem, destino, 0, progresso)
}

// destinoAPI é a área de preparo do outro lado da API.
type destinoAPI struct {
	api     *cliente
	versoes map[int]string
	pessoas map[int]bool
}

func (d *destinoAPI) carregar(ctx context.Context) error {
	if d.versoes != nil {
		return nil
	}
	var r struct {
		Versoes map[int]string `json:"versoes"`
		Pessoas []int          `json:"pessoas"`
	}
	if _, err := d.api.chamar(ctx, http.MethodGet, "/versoes", nil, &r); err != nil {
		return err
	}
	d.versoes, d.pessoas = r.Versoes, map[int]bool{}
	if d.versoes == nil {
		d.versoes = map[int]string{}
	}
	for _, id := range r.Pessoas {
		d.pessoas[id] = true
	}
	return nil
}

func (d *destinoAPI) Versoes(ctx context.Context) (map[int]string, error) {
	err := d.carregar(ctx)
	return d.versoes, err
}

func (d *destinoAPI) Pessoas(ctx context.Context) (map[int]bool, error) {
	err := d.carregar(ctx)
	return d.pessoas, err
}

// LoteMaxBytes cabe no limite de 1 MiB por requisição da API.
const LoteMaxBytes = 900 << 10

// Gravar envia em lotes que cabem numa requisição (até 200 itens).
func (d *destinoAPI) Gravar(ctx context.Context, itens []glpi.ItemPreparo) error {
	lote, tamanho := []glpi.ItemPreparo{}, 0
	enviar := func() error {
		if len(lote) == 0 {
			return nil
		}
		_, err := d.api.chamar(ctx, http.MethodPost, "/itens", map[string]any{"itens": lote}, nil)
		lote, tamanho = lote[:0], 0
		return err
	}
	for _, it := range itens {
		n := len(it.Dados) + 512
		if len(lote) == 200 || (len(lote) > 0 && tamanho+n > LoteMaxBytes) {
			if err := enviar(); err != nil {
				return err
			}
		}
		lote, tamanho = append(lote, it), tamanho+n
	}
	return enviar()
}

// cliente fala com a API do Projeto Nova.
type cliente struct {
	base, token string
	http        *http.Client
}

// chamar faz a requisição e decodifica o "data" da resposta.
func (c *cliente) chamar(ctx context.Context, metodo, caminho string, corpo, out any) (int, error) {
	var envio io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return 0, err
		}
		envio = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, c.base+caminho, envio)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(b, &e)
		return resp.StatusCode, fmt.Errorf("API %s %s: %d %s", metodo, caminho, resp.StatusCode, e.Error.Message)
	}
	if out != nil && len(b) > 0 {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(b, &env); err != nil {
			return resp.StatusCode, fmt.Errorf("API %s: %w", caminho, err)
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("API %s: %w", caminho, err)
		}
	}
	return resp.StatusCode, nil
}
