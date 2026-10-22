// Package domain é o produto digital do ITIL 5: o que a organização
// mantém ao longo do ciclo de vida (descoberta, desenho, aquisição,
// construção, transição, operação, entrega e suporte).
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

const (
	NomeMax  = 160
	TextoMax = 4000
)

var (
	ErrNaoEncontrado      = errors.New("produtos: produto não encontrado")
	ErrEntradaInvalida    = errors.New("produtos: nome obrigatório (até 160 caracteres) e textos até 4000")
	ErrEstagioInvalido    = errors.New("produtos: estágio fora do ciclo de vida")
	ErrForaDoEscopo       = errors.New("produtos: produto fora do seu escopo de gestão")
	ErrUnidadeInexistente = errors.New("produtos: unidade inexistente")
)

// Estagios é o ciclo de vida do produto (ITIL 5), nesta ordem.
var Estagios = []string{
	"descoberta", "desenho", "aquisicao", "construcao", "transicao", "operacao", "entrega", "suporte",
}

// Produto é um produto digital posicionado numa unidade (ADR 013).
// Unidade nula é institucional: só a gestão global o alcança.
type Produto struct {
	ID        uuid.UUID  `json:"id"`
	Nome      string     `json:"nome"`
	Descricao string     `json:"descricao"`
	Estagio   string     `json:"estagio"`
	Utilidade string     `json:"utilidade"`
	Garantia  string     `json:"garantia"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Dona é a posição do produto para produtos:manage.
func Dona(unidade *uuid.UUID) auth.Target {
	if unidade == nil {
		return auth.Target{}
	}
	return auth.InUnidade(*unidade)
}

// EstagioValido reporta se o estágio pertence ao ciclo.
func EstagioValido(estagio string) bool {
	for _, e := range Estagios {
		if e == estagio {
			return true
		}
	}
	return false
}

// Novo valida e monta um produto.
func Novo(nome, descricao, estagio, utilidade, garantia string, unidade *uuid.UUID) (Produto, error) {
	nome, err := nomeObrigatorio(nome)
	if err != nil {
		return Produto{}, err
	}
	descricao, utilidade, garantia, err = textos(descricao, utilidade, garantia)
	if err != nil {
		return Produto{}, err
	}
	if !EstagioValido(estagio) {
		return Produto{}, ErrEstagioInvalido
	}
	now := time.Now().UTC()
	return Produto{
		ID: uuid.New(), Nome: nome, Descricao: descricao, Estagio: estagio,
		Utilidade: utilidade, Garantia: garantia, UnidadeID: unidade,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Aplicar devolve a cópia com os campos editáveis trocados. O estágio não
// muda aqui: ele tem caso de uso próprio, para o evento ser específico.
func (p Produto) Aplicar(nome, descricao, utilidade, garantia string, unidade *uuid.UUID) (Produto, error) {
	nome, err := nomeObrigatorio(nome)
	if err != nil {
		return Produto{}, err
	}
	descricao, utilidade, garantia, err = textos(descricao, utilidade, garantia)
	if err != nil {
		return Produto{}, err
	}
	p.Nome, p.Descricao, p.Utilidade, p.Garantia, p.UnidadeID = nome, descricao, utilidade, garantia, unidade
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

// ComEstagio troca o estágio. changed é falso quando o produto já está
// nele: repetir a transição não gera auditoria nem evento.
func (p Produto) ComEstagio(estagio string) (Produto, bool, error) {
	if !EstagioValido(estagio) {
		return Produto{}, false, ErrEstagioInvalido
	}
	if p.Estagio == estagio {
		return p, false, nil
	}
	p.Estagio = estagio
	p.UpdatedAt = time.Now().UTC()
	return p, true, nil
}

// Igual ignora os horários: serve para não regravar uma edição vazia.
func (p Produto) Igual(o Produto) bool {
	return p.ID == o.ID && p.Nome == o.Nome && p.Descricao == o.Descricao && p.Estagio == o.Estagio &&
		p.Utilidade == o.Utilidade && p.Garantia == o.Garantia && mesmaUnidade(p.UnidadeID, o.UnidadeID)
}

func nomeObrigatorio(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > NomeMax {
		return "", ErrEntradaInvalida
	}
	return s, nil
}

func textos(descricao, utilidade, garantia string) (string, string, string, error) {
	var err error
	if descricao, err = texto(descricao); err != nil {
		return "", "", "", err
	}
	if utilidade, err = texto(utilidade); err != nil {
		return "", "", "", err
	}
	if garantia, err = texto(garantia); err != nil {
		return "", "", "", err
	}
	return descricao, utilidade, garantia, nil
}

func texto(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > TextoMax {
		return "", ErrEntradaInvalida
	}
	return s, nil
}

func mesmaUnidade(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
