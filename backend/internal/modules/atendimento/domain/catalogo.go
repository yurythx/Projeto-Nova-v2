package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Limites da apresentação no catálogo em blocos (ADR 078).
const (
	CategoriaNomeMax      = 80
	CategoriaDescricaoMax = 300
	OrdemMax              = 999
)

// ErrApresentacaoInvalida: nome, descrição, ícone ou ordem fora dos limites.
var ErrApresentacaoInvalida = errors.New("atendimento: categoria ou bloco do catálogo inválido")

// ErrCategoriaRepetida: já há uma categoria com esse nome.
var ErrCategoriaRepetida = errors.New("atendimento: categoria com esse nome já existe")

// ErrCategoriaInexistente: a oferta aponta para uma categoria que não há.
var ErrCategoriaInexistente = errors.New("atendimento: categoria não encontrada")

// icone é o nome de um ícone da biblioteca da tela (letras, dígitos e
// hífen); vazio usa o ícone padrão do tipo.
var icone = regexp.MustCompile(`^[a-z0-9-]{0,40}$`)

// Categoria agrupa ofertas na central do solicitante. É da instituição:
// só quem gere o atendimento em toda ela cria e altera.
type Categoria struct {
	ID        uuid.UUID `json:"id"`
	Nome      string    `json:"nome"`
	Descricao string    `json:"descricao"`
	Icone     string    `json:"icone"`
	Ordem     int       `json:"ordem"`
	Ativa     bool      `json:"ativa"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NovaCategoria valida e cria a categoria.
func NovaCategoria(nome, descricao, ic string, ordem int, ativa bool) (Categoria, error) {
	now := time.Now().UTC()
	c := Categoria{ID: uuid.New(), CreatedAt: now, UpdatedAt: now}
	return c.Aplicar(nome, descricao, ic, ordem, ativa)
}

// Aplicar devolve a cópia editada.
func (c Categoria) Aplicar(nome, descricao, ic string, ordem int, ativa bool) (Categoria, error) {
	nome, descricao = strings.TrimSpace(nome), strings.TrimSpace(descricao)
	if nome == "" || utf8.RuneCountInString(nome) > CategoriaNomeMax || utf8.RuneCountInString(descricao) > CategoriaDescricaoMax {
		return Categoria{}, ErrApresentacaoInvalida
	}
	ic, err := Apresentacao(ic, ordem)
	if err != nil {
		return Categoria{}, err
	}
	c.Nome, c.Descricao, c.Icone, c.Ordem, c.Ativa = nome, descricao, ic, ordem, ativa
	return c, nil
}

// Apresentacao confere o ícone e a ordem de um bloco do catálogo.
func Apresentacao(ic string, ordem int) (string, error) {
	ic = strings.TrimSpace(ic)
	if !icone.MatchString(ic) || ordem < 0 || ordem > OrdemMax {
		return "", ErrApresentacaoInvalida
	}
	return ic, nil
}
