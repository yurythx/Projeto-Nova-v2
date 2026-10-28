// Package domain contém as regras do módulo de integrações (ADR 076):
// a conexão com a origem, a área de preparo, a tabela de-para e as tarefas.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	TipoGLPI = "glpi"

	// ModoDireto: o servidor lê a origem. ModoConector: um programa na
	// rede do cliente lê e envia.
	ModoDireto   = "direto"
	ModoConector = "conector"

	TarefaExtracao  = "extracao"
	TarefaSimulacao = "simulacao"
	TarefaCarga     = "carga"

	EstadoPendente    = "pendente"
	EstadoEmAndamento = "em_andamento"
	EstadoConcluida   = "concluida"
	EstadoFalhou      = "falhou"
	EstadoCancelada   = "cancelada"

	ItemChamado   = "chamado"
	ItemPessoa    = "pessoa"
	ItemCategoria = "categoria"
	ItemEntidade  = "entidade"
	ItemSLA       = "sla"

	NomeMax   = 160
	URLMax    = 500
	PerfilMax = 80
	// DadosMax limita um item da área de preparo (um chamado e as suas
	// mensagens): o lote do conector cabe em 1 MiB.
	DadosMax = 900 << 10
	// LoteMax é quantos itens o conector envia de uma vez.
	LoteMax = 200
	// CategoriaQualquer é o "*" da tabela: o destino padrão da secretaria.
	CategoriaQualquer = -1

	// PrefixoToken marca o bearer do conector. Não é um JWT de pessoa.
	PrefixoToken = "nova_con_"
	// OfertaGeral é a sugestão para o chamado sem categoria.
	OfertaGeral = "Atendimento geral"
)

// Prioridades e tipos aceitos na tabela, como no atendimento.
var (
	Prioridades = []string{"critica", "alta", "media", "baixa"}
	Tipos       = []string{"incidente", "requisicao"}
)

var (
	ErrNaoEncontrado = errors.New("integracoes: não encontrado")
	ErrForaDoEscopo  = errors.New("integracoes: fora do escopo")
	// ErrTarefaAtiva: uma tarefa por conexão de cada vez.
	ErrTarefaAtiva = errors.New("integracoes: já há uma tarefa em andamento nesta conexão")
)

// Recusa é a regra de negócio que barra a operação, com a explicação.
type Recusa struct{ Msg string }

func (r Recusa) Error() string { return r.Msg }

// Credenciais da origem no modo direto. Cifradas no banco; a API nunca
// as devolve.
type Credenciais struct {
	AppToken  string `json:"app_token,omitempty"`
	UserToken string `json:"user_token,omitempty"`
	Login     string `json:"login,omitempty"`
	Senha     string `json:"senha,omitempty"`
}

// Vazia diz se não há como entrar na origem.
func (c Credenciais) Vazia() bool {
	return c.UserToken == "" && (c.Login == "" || c.Senha == "")
}

// Conexao é a ligação com uma origem.
type Conexao struct {
	ID             uuid.UUID  `json:"id"`
	Tipo           string     `json:"tipo"`
	Nome           string     `json:"nome"`
	Modo           string     `json:"modo"`
	URL            string     `json:"url"`
	Perfil         string     `json:"perfil"`
	Fuso           string     `json:"fuso"`
	TemCredenciais bool       `json:"tem_credenciais"`
	TemToken       bool       `json:"tem_token"`
	TokenEm        *time.Time `json:"token_em,omitempty"`
	ExtraidoEm     *time.Time `json:"extraido_em,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedBy      string     `json:"created_by"`
	// Cifradas são as credenciais como estão no banco.
	Cifradas string `json:"-"`
}

// NovaConexao valida o cadastro. No modo direto, a URL é obrigatória: é o
// servidor que vai até ela.
func NovaConexao(tipo, nome, modo, endereco, perfil, fuso string) (Conexao, error) {
	c := Conexao{Tipo: strings.TrimSpace(tipo), Nome: strings.TrimSpace(nome), Modo: strings.TrimSpace(modo),
		URL: strings.TrimRight(strings.TrimSpace(endereco), "/"), Perfil: strings.TrimSpace(perfil), Fuso: strings.TrimSpace(fuso)}
	if c.Fuso == "" {
		c.Fuso = "America/Sao_Paulo"
	}
	switch {
	case c.Tipo != TipoGLPI:
		return Conexao{}, Recusa{"a origem suportada é o glpi"}
	case c.Nome == "" || utf8.RuneCountInString(c.Nome) > NomeMax:
		return Conexao{}, Recusa{"informe o nome da conexão, com até 160 caracteres"}
	case c.Modo != ModoDireto && c.Modo != ModoConector:
		return Conexao{}, Recusa{"o modo é direto ou conector"}
	case utf8.RuneCountInString(c.Perfil) > PerfilMax:
		return Conexao{}, Recusa{"o perfil cabe em 80 caracteres"}
	case len(c.URL) > URLMax:
		return Conexao{}, Recusa{"o endereço cabe em 500 caracteres"}
	}
	if c.URL != "" || c.Modo == ModoDireto {
		u, err := url.Parse(c.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return Conexao{}, Recusa{"o endereço da API é http(s)://servidor/apirest.php"}
		}
	}
	if _, err := time.LoadLocation(c.Fuso); err != nil {
		return Conexao{}, Recusa{"fuso desconhecido: " + c.Fuso}
	}
	agora := time.Now().UTC()
	c.ID, c.CreatedAt, c.UpdatedAt = uuid.New(), agora, agora
	return c, nil
}

// EmitirToken gera o segredo do conector. Só o hash fica guardado.
func EmitirToken() (token, hash string) {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf) // não falha (Go 1.24+: aborta o processo sem entropia)
	token = PrefixoToken + base64.RawURLEncoding.EncodeToString(buf)
	return token, HashToken(token)
}

// HashToken é o que se compara no banco.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Item é um registro da área de preparo: o dado bruto da origem.
type Item struct {
	Tipo      string          `json:"tipo"`
	ID        int             `json:"id"`
	Versao    string          `json:"versao,omitempty"`
	Nome      string          `json:"nome,omitempty"`
	Entidade  *int            `json:"entidade,omitempty"`
	Categoria *int            `json:"categoria,omitempty"`
	SLA       *int            `json:"sla,omitempty"`
	Minutos   *int            `json:"minutos,omitempty"`
	Classe    string          `json:"classe,omitempty"`
	Data      string          `json:"data,omitempty"`
	Dados     json.RawMessage `json:"dados"`
}

// ValidarItens confere o lote que chega do conector.
func ValidarItens(itens []Item) error {
	if len(itens) == 0 || len(itens) > LoteMax {
		return Recusa{"o lote tem de 1 a 200 itens"}
	}
	for _, it := range itens {
		switch {
		case !slices.Contains([]string{ItemChamado, ItemPessoa, ItemCategoria, ItemEntidade, ItemSLA}, it.Tipo):
			return Recusa{"tipo de item desconhecido: " + it.Tipo}
		case it.ID < 0:
			return Recusa{"o id do item não é negativo"}
		case len(it.Dados) == 0 || len(it.Dados) > DadosMax || !json.Valid(it.Dados):
			return Recusa{"os dados do item são um JSON de até 900 KB"}
		case it.Classe != "" && !slices.Contains(Tipos, it.Classe):
			return Recusa{"a classe é incidente ou requisicao"}
		case utf8.RuneCountInString(it.Nome) > 500 || len(it.Versao) > 40 || len(it.Data) > 40:
			return Recusa{"nome, versão ou data longos demais"}
		}
	}
	return nil
}

// Linha é uma linha da tabela de-para (ADR 075).
type Linha struct {
	Secretaria    string     `json:"secretaria"`
	Categoria     int        `json:"categoria"`
	CategoriaNome string     `json:"categoria_nome"`
	Chamados      int        `json:"chamados"`
	ComPrazo      int        `json:"com_prazo"`
	Nivel         string     `json:"nivel"`
	UnidadeID     *uuid.UUID `json:"unidade_id"`
	UnidadeNome   string     `json:"unidade_nome,omitempty"`
	Oferta        string     `json:"oferta"`
	// Catalogo é a categoria do catálogo em blocos em que a oferta entra
	// (ADR 078). Vazio: "Outros".
	Catalogo   string `json:"catalogo"`
	Tipo       string `json:"tipo"`
	Prioridade string `json:"prioridade"`
}

// CatalogoMax é o tamanho do nome da categoria do catálogo.
const CatalogoMax = 80

// ValidarTabela confere a tabela inteira: uma linha por secretaria e
// categoria.
func ValidarTabela(linhas []Linha) error {
	vistas := map[string]bool{}
	for i := range linhas {
		l := &linhas[i]
		l.Secretaria, l.Oferta, l.Catalogo = strings.TrimSpace(l.Secretaria), strings.TrimSpace(l.Oferta), strings.TrimSpace(l.Catalogo)
		switch {
		case l.Secretaria == "" || utf8.RuneCountInString(l.Secretaria) > NomeMax:
			return Recusa{"cada linha tem a secretaria, com até 160 caracteres"}
		case l.Categoria < CategoriaQualquer:
			return Recusa{"a categoria é o id da origem, ou -1 para todas"}
		case l.Oferta == "" || utf8.RuneCountInString(l.Oferta) > NomeMax:
			return Recusa{"cada linha tem a oferta, com até 160 caracteres"}
		case utf8.RuneCountInString(l.Catalogo) > CatalogoMax:
			return Recusa{"a categoria do catálogo cabe em 80 caracteres"}
		case !slices.Contains(Tipos, l.Tipo):
			return Recusa{"o tipo é incidente ou requisicao"}
		case !slices.Contains(Prioridades, l.Prioridade):
			return Recusa{"a prioridade é critica, alta, media ou baixa"}
		}
		chave := fmt.Sprintf("%s\x00%d", strings.ToUpper(l.Secretaria), l.Categoria)
		if vistas[chave] {
			return Recusa{"a mesma secretaria e categoria aparecem duas vezes: " + l.Secretaria}
		}
		vistas[chave] = true
	}
	return nil
}

// Tarefa é uma execução: extrair, simular ou carregar.
type Tarefa struct {
	ID          uuid.UUID       `json:"id"`
	ConexaoID   uuid.UUID       `json:"conexao_id"`
	Tipo        string          `json:"tipo"`
	Estado      string          `json:"estado"`
	Feitos      int             `json:"feitos"`
	Total       int             `json:"total"`
	Relatorio   json.RawMessage `json:"relatorio,omitempty"`
	Erro        string          `json:"erro,omitempty"`
	PedidaPor   string          `json:"pedida_por"`
	CriadaEm    time.Time       `json:"criada_em"`
	IniciadaEm  *time.Time      `json:"iniciada_em,omitempty"`
	BatidaEm    *time.Time      `json:"batida_em,omitempty"`
	TerminadaEm *time.Time      `json:"terminada_em,omitempty"`
}

// NovaTarefa valida o pedido.
func NovaTarefa(conexao uuid.UUID, tipo, quem string) (Tarefa, error) {
	if !slices.Contains([]string{TarefaExtracao, TarefaSimulacao, TarefaCarga}, tipo) {
		return Tarefa{}, Recusa{"a tarefa é extracao, simulacao ou carga"}
	}
	return Tarefa{ID: uuid.New(), ConexaoID: conexao, Tipo: tipo, Estado: EstadoPendente, PedidaPor: quem, CriadaEm: time.Now().UTC()}, nil
}

// Terminada diz se a tarefa já não muda.
func (t Tarefa) Terminada() bool {
	return t.Estado == EstadoConcluida || t.Estado == EstadoFalhou || t.Estado == EstadoCancelada
}

// Secretaria é o 2º nível da árvore de entidades ("RAIZ > SMS > UPA" é
// SMS): é o que separa as categorias e o prazo na origem.
func Secretaria(completo string) string {
	partes := strings.Split(completo, " > ")
	if len(partes) > 1 {
		return strings.TrimSpace(partes[1])
	}
	return strings.TrimSpace(partes[0])
}
