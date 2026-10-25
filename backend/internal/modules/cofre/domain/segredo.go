// Package domain é o cadastro do cofre. O segredo em claro não mora aqui:
// a lista só carrega nome, usuário e unidade.
package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNaoEncontrado  = errors.New("cofre: não encontrado")
	ErrForaDoEscopo   = errors.New("cofre: fora do seu escopo")
	ErrSemPermissao   = errors.New("cofre: sem permissão para esta ação")
	ErrSenhaRecusada  = errors.New("cofre: senha de acesso recusada")
	ErrBloqueado      = errors.New("cofre: muitas tentativas de senha")
	ErrSemConfirmacao = errors.New("cofre: confirmação indisponível")
)

// EntradaInvalida explica o cadastro recusado.
type EntradaInvalida struct{ Msg string }

func (e EntradaInvalida) Error() string { return e.Msg }

// Segredo é o que a lista pode mostrar. Sem senha e sem texto cifrado.
type Segredo struct {
	ID        uuid.UUID  `json:"id"`
	Nome      string     `json:"nome"`
	Usuario   string     `json:"usuario"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Versao    int        `json:"versao"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Selado é a linha gravada. Dek e Cifra não saem no JSON.
type Selado struct {
	Segredo
	Dek   string `json:"-"`
	Cifra string `json:"-"`
}

// Revelado é a resposta de uma revelação. A senha só existe neste tipo.
type Revelado struct {
	Segredo
	Senha string `json:"senha"`
	Notas string `json:"notas"`
}

// Filtro restringe a listagem às unidades cobertas. Restrito falso vê tudo.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
}

// AAD amarra o texto cifrado a este segredo, a esta versão e a esta unidade.
// Trocar a linha de lugar não abre com a chave de outra.
func AAD(id uuid.UUID, versao int, unidade *uuid.UUID) []byte {
	dono := "-"
	if unidade != nil {
		dono = unidade.String()
	}
	return []byte(id.String() + ":" + strconv.Itoa(versao) + ":" + dono)
}

// Validar aceita o cadastro. A senha pode ir vazia quando a alteração não a troca.
func Validar(nome, usuario, senha, notas string, senhaObrigatoria bool) (string, string, string, string, error) {
	nome = strings.TrimSpace(nome)
	usuario = strings.TrimSpace(usuario)
	senha = strings.TrimSpace(senha)
	notas = strings.TrimSpace(notas)
	if nome == "" || utf8.RuneCountInString(nome) > 160 {
		return "", "", "", "", EntradaInvalida{"informe o nome, com até 160 caracteres"}
	}
	if utf8.RuneCountInString(usuario) > 160 {
		return "", "", "", "", EntradaInvalida{"o usuário deve ter até 160 caracteres"}
	}
	if senhaObrigatoria && senha == "" {
		return "", "", "", "", EntradaInvalida{"informe a senha"}
	}
	if utf8.RuneCountInString(senha) > 2000 || utf8.RuneCountInString(notas) > 4000 {
		return "", "", "", "", EntradaInvalida{"a senha deve ter até 2000 caracteres e as notas até 4000"}
	}
	return nome, usuario, senha, notas, nil
}
