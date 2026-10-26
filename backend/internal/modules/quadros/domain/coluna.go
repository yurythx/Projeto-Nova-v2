// Package domain guarda as regras do catálogo de colunas. A chave é o
// estado do registro. Nome, cor e ordem são rótulo. Uma coluna nova nasce
// só no quadro pedido, com chave gerada e passagens só entre as colunas
// desse quadro. O fluxo de fábrica de cada módulo continua valendo.
package domain

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

var (
	ErrQuadro        = errors.New("quadro desconhecido")
	ErrNaoEncontrada = errors.New("coluna não encontrada")
	ErrProibido      = errors.New("sem permissão")
	ErrPatchVazio    = errors.New("nada a alterar")
	ErrNome          = errors.New("nome inválido")
	ErrCor           = errors.New("cor inválida")
	ErrInicial       = errors.New("coluna inicial")
	ErrComRegistro   = errors.New("coluna com registro")
	ErrRetirada      = errors.New("coluna retirada")
	ErrPassagem      = errors.New("passagem inválida")
	ErrSemEntrada    = errors.New("coluna sem entrada")
)

// Coluna é uma faixa do quadro. Chave não muda. Entrada é a coluna em
// que o fluxo começa e não pode ser retirada.
type Coluna struct {
	Quadro  string   `json:"quadro"`
	Chave   string   `json:"chave"`
	Nome    string   `json:"nome"`
	Cor     string   `json:"cor"`
	Tom     string   `json:"tom"`
	Ordem   int      `json:"ordem"`
	Entrada bool     `json:"entrada"`
	Ativa   bool     `json:"ativa"`
	De      []string `json:"de,omitempty"`
	Para    []string `json:"para,omitempty"`
}

// Passagem é um caminho a mais, só dentro de um quadro. A de fábrica não
// fica nesta lista.
type Passagem struct {
	De   string
	Para string
}

// Patch é uma alteração parcial. Ativa false retira; true devolve no fim.
type Patch struct {
	Nome  *string
	Cor   *string
	Ordem *int
	Ativa *bool
}

// Repository persiste o catálogo e conta os registros da coluna. A
// contagem lê as tabelas dos outros módulos; este pacote não as importa.
type Repository interface {
	Listar(ctx context.Context, db database.DBTX, quadro string) ([]Coluna, error)
	Salvar(ctx context.Context, db database.DBTX, cols []Coluna) error
	Inserir(ctx context.Context, db database.DBTX, c Coluna) error
	SoltarOrdens(ctx context.Context, db database.DBTX, quadro string) error
	Contar(ctx context.Context, db database.DBTX, quadro, chave string) (int, error)
	Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error)
	SubstituirPassagens(ctx context.Context, db database.DBTX, quadro, chave string, itens []Passagem) error
}

// acesso de leitura e de escrita, com as permissões que o módulo já tem.
type acesso struct {
	ler, escrever []string
}

var quadros = map[string]acesso{
	"problemas": {ler: []string{"problemas:read", "problemas:manage"}, escrever: []string{"problemas:manage"}},
	"mudancas":  {ler: []string{"mudancas:manage", "mudancas:aprovar"}, escrever: []string{"mudancas:manage"}},
	"projetos":  {ler: []string{"projetos:read", "projetos:manage"}, escrever: []string{"projetos:manage"}},
	"tarefas":   {ler: []string{"projetos:read", "projetos:manage"}, escrever: []string{"projetos:manage"}},
	"ativos":    {ler: []string{"ativos:manage", "ativos:reportar"}, escrever: []string{"ativos:manage"}},
	"fila":      {ler: []string{"atendimento:atender", "atendimento:manage"}, escrever: []string{"atendimento:manage"}},
}

var corHex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Conhecido diz se o quadro faz parte do catálogo de fábrica.
func Conhecido(quadro string) bool {
	_, ok := quadros[quadro]
	return ok
}

// PodeLer é quem já abre aquele quadro.
func PodeLer(id auth.Identity, quadro string) bool {
	regra, ok := quadros[quadro]
	return ok && tem(id, regra.ler)
}

// PodeEscrever é quem já gere aquele módulo.
func PodeEscrever(id auth.Identity, quadro string) bool {
	regra, ok := quadros[quadro]
	return ok && tem(id, regra.escrever)
}

func tem(id auth.Identity, perms []string) bool {
	return slices.ContainsFunc(perms, func(p string) bool {
		return auth.HasPermission(id, auth.Permission(p))
	})
}

// Aplicar devolve as colunas depois do patch. A chave não muda. A ordem
// das ativas fica única e contínua a partir de zero.
func Aplicar(cols []Coluna, chave string, registros int, p Patch) ([]Coluna, error) {
	if p.Nome == nil && p.Cor == nil && p.Ordem == nil && p.Ativa == nil {
		return nil, ErrPatchVazio
	}
	out := slices.Clone(cols)
	i := slices.IndexFunc(out, func(c Coluna) bool { return c.Chave == chave })
	if i < 0 {
		return nil, ErrNaoEncontrada
	}
	if p.Nome != nil {
		nome := strings.TrimSpace(*p.Nome)
		if nome == "" || utf8.RuneCountInString(nome) > 80 {
			return nil, ErrNome
		}
		out[i].Nome = nome
	}
	if p.Cor != nil {
		cor := strings.TrimSpace(*p.Cor)
		if cor != "" && !corHex.MatchString(cor) {
			return nil, ErrCor
		}
		out[i].Cor = cor
	}
	if p.Ativa != nil && *p.Ativa != out[i].Ativa {
		if *p.Ativa {
			out[i].Ativa = true
			out[i].Ordem = maiorOrdem(out) + 1
			out = compactar(out)
		} else {
			if out[i].Entrada {
				return nil, ErrInicial
			}
			if registros > 0 {
				return nil, ErrComRegistro
			}
			out[i].Ativa = false
			out = compactar(out)
		}
	}
	if p.Ordem != nil && p.Ativa == nil {
		if !ativa(out, chave) {
			return nil, ErrRetirada
		}
		return mover(out, chave, *p.Ordem)
	}
	return out, nil
}

func ativa(cols []Coluna, chave string) bool {
	i := slices.IndexFunc(cols, func(c Coluna) bool { return c.Chave == chave })
	return i >= 0 && cols[i].Ativa
}

func maiorOrdem(cols []Coluna) int {
	maior := -1
	for _, c := range cols {
		if c.Ativa && c.Ordem > maior {
			maior = c.Ordem
		}
	}
	return maior
}

func compactar(cols []Coluna) []Coluna {
	out := slices.Clone(cols)
	type ref struct {
		idx, ordem int
		chave      string
	}
	var ativas []ref
	for i, c := range out {
		if c.Ativa {
			ativas = append(ativas, ref{i, c.Ordem, c.Chave})
		}
	}
	slices.SortStableFunc(ativas, func(a, b ref) int {
		if a.ordem != b.ordem {
			return a.ordem - b.ordem
		}
		return strings.Compare(a.chave, b.chave)
	})
	for n, r := range ativas {
		out[r.idx].Ordem = n
	}
	return out
}

// mover coloca a coluna na posição destino entre as ativas. destino é o
// índice final, a partir de zero.
func mover(cols []Coluna, chave string, destino int) ([]Coluna, error) {
	out := slices.Clone(cols)
	var ativas []Coluna
	for _, c := range out {
		if c.Ativa {
			ativas = append(ativas, c)
		}
	}
	slices.SortStableFunc(ativas, func(a, b Coluna) int {
		if a.Ordem != b.Ordem {
			return a.Ordem - b.Ordem
		}
		return strings.Compare(a.Chave, b.Chave)
	})
	pos := slices.IndexFunc(ativas, func(c Coluna) bool { return c.Chave == chave })
	if pos < 0 {
		return nil, ErrRetirada
	}
	if destino < 0 {
		destino = 0
	}
	if destino >= len(ativas) {
		destino = len(ativas) - 1
	}
	if destino == pos {
		return out, nil
	}
	item := ativas[pos]
	rest := append(append([]Coluna{}, ativas[:pos]...), ativas[pos+1:]...)
	final := append(append([]Coluna{}, rest[:destino]...), item)
	final = append(final, rest[destino:]...)
	ordem := map[string]int{}
	for i, c := range final {
		ordem[c.Chave] = i
	}
	for i := range out {
		if n, ok := ordem[out[i].Chave]; ok {
			out[i].Ordem = n
		}
	}
	return out, nil
}

// Sequencia devolve as chaves ativas, da ordem zero em diante.
func Sequencia(cols []Coluna) []string {
	ativas := []Coluna{}
	for _, c := range cols {
		if c.Ativa {
			ativas = append(ativas, c)
		}
	}
	slices.SortStableFunc(ativas, func(a, b Coluna) int { return a.Ordem - b.Ordem })
	out := make([]string, len(ativas))
	for i, c := range ativas {
		out[i] = c.Chave
	}
	return out
}

// Iguais compara o que a auditoria precisa ver.
func Iguais(a, b []Coluna) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Chave != b[i].Chave || a[i].Nome != b[i].Nome || a[i].Cor != b[i].Cor || a[i].Ordem != b[i].Ordem || a[i].Ativa != b[i].Ativa {
			return false
		}
	}
	return true
}
