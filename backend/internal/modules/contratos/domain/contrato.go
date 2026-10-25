// Package domain é o cadastro de fornecedor, contrato e licença.
// A licença guarda quantidade e vencimento, não a chave secreta.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNaoEncontrado = errors.New("contratos: não encontrado")
	ErrForaDoEscopo  = errors.New("contratos: fora do seu escopo")
)

const nomeMax = 160

// Fornecedor é quem presta o contrato ou cede a licença.
type Fornecedor struct {
	ID        uuid.UUID  `json:"id"`
	Nome      string     `json:"nome"`
	Email     string     `json:"email"`
	Telefone  string     `json:"telefone"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Contrato tem vigência e valor em centavos.
type Contrato struct {
	ID             uuid.UUID  `json:"id"`
	Numero         string     `json:"numero"`
	Titulo         string     `json:"titulo"`
	FornecedorID   uuid.UUID  `json:"fornecedor_id"`
	FornecedorNome string     `json:"fornecedor_nome"`
	Inicio         time.Time  `json:"inicio"`
	Fim            time.Time  `json:"fim"`
	ValorCentavos  int64      `json:"valor_centavos"`
	UnidadeID      *uuid.UUID `json:"unidade_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Licenca é um direito de uso, com quantidade de assentos.
type Licenca struct {
	ID             uuid.UUID  `json:"id"`
	Nome           string     `json:"nome"`
	FornecedorID   uuid.UUID  `json:"fornecedor_id"`
	FornecedorNome string     `json:"fornecedor_nome"`
	Fim            *time.Time `json:"fim,omitempty"`
	Quantidade     int        `json:"quantidade"`
	UnidadeID      *uuid.UUID `json:"unidade_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Vencimento é um contrato ou licença dentro da janela de aviso.
type Vencimento struct {
	ID        uuid.UUID
	Tipo      string
	Titulo    string
	Fim       time.Time
	UnidadeID *uuid.UUID
}

// EntradaInvalida explica o cadastro recusado.
type EntradaInvalida struct{ Msg string }

func (e EntradaInvalida) Error() string { return e.Msg }

func nomeDe(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > nomeMax {
		return "", EntradaInvalida{"informe o nome, com até 160 caracteres"}
	}
	return s, nil
}

func emailDe(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if !strings.Contains(s, "@") || utf8.RuneCountInString(s) > 160 {
		return "", EntradaInvalida{"e-mail do fornecedor inválido"}
	}
	return s, nil
}

// NovoFornecedor valida o cadastro.
func NovoFornecedor(nome, email, telefone string, unidade *uuid.UUID) (Fornecedor, error) {
	nome, err := nomeDe(nome)
	if err != nil {
		return Fornecedor{}, err
	}
	email, err = emailDe(email)
	if err != nil {
		return Fornecedor{}, err
	}
	telefone = strings.TrimSpace(telefone)
	if utf8.RuneCountInString(telefone) > 40 {
		return Fornecedor{}, EntradaInvalida{"telefone longo demais"}
	}
	now := time.Now().UTC()
	return Fornecedor{ID: uuid.New(), Nome: nome, Email: email, Telefone: telefone, UnidadeID: unidade, CreatedAt: now, UpdatedAt: now}, nil
}

// NovoContrato valida a vigência. O valor é em centavos e não pode ser negativo.
func NovoContrato(numero, titulo string, fornecedor uuid.UUID, inicio, fim time.Time, valor int64, unidade *uuid.UUID) (Contrato, error) {
	titulo, err := nomeDe(titulo)
	if err != nil {
		return Contrato{}, err
	}
	numero = strings.TrimSpace(numero)
	if utf8.RuneCountInString(numero) > 40 || fornecedor == uuid.Nil || valor < 0 || fim.Before(inicio) {
		return Contrato{}, EntradaInvalida{"informe o fornecedor, o valor e uma vigência em que o fim não antecede o início"}
	}
	now := time.Now().UTC()
	return Contrato{
		ID: uuid.New(), Numero: numero, Titulo: titulo, FornecedorID: fornecedor,
		Inicio: inicio, Fim: fim, ValorCentavos: valor, UnidadeID: unidade, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// NovaLicenca valida a licença. Sem fim, não há aviso de vencimento.
func NovaLicenca(nome string, fornecedor uuid.UUID, fim *time.Time, quantidade int, unidade *uuid.UUID) (Licenca, error) {
	nome, err := nomeDe(nome)
	if err != nil {
		return Licenca{}, err
	}
	if fornecedor == uuid.Nil || quantidade < 1 {
		return Licenca{}, EntradaInvalida{"informe o fornecedor e ao menos um assento"}
	}
	now := time.Now().UTC()
	return Licenca{ID: uuid.New(), Nome: nome, FornecedorID: fornecedor, Fim: fim, Quantidade: quantidade, UnidadeID: unidade, CreatedAt: now, UpdatedAt: now}, nil
}
