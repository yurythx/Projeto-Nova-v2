package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Pasta guarda segredos com acesso mais estreito que a unidade.
type Pasta struct {
	ID        uuid.UUID  `json:"id"`
	Nome      string     `json:"nome"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Membros   []Membro   `json:"membros,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Membro vê o metadado da pasta. PodeRevelar libera a senha, com a senha de acesso.
type Membro struct {
	UsuarioID   uuid.UUID `json:"usuario_id"`
	PodeRevelar bool      `json:"pode_revelar"`
}

// Pessoa é quem pode entrar na pasta. Sem senha e sem papel do Keycloak.
type Pessoa struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}

// NovaPasta valida o nome. Os membros são conferidos à parte.
func NovaPasta(nome string, unidade *uuid.UUID, membros []Membro) (Pasta, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > 160 {
		return Pasta{}, EntradaInvalida{"informe o nome da pasta, com até 160 caracteres"}
	}
	limpos, err := MembrosDe(membros)
	if err != nil {
		return Pasta{}, err
	}
	agora := time.Now().UTC()
	return Pasta{ID: uuid.New(), Nome: nome, UnidadeID: unidade, Membros: limpos, CreatedAt: agora, UpdatedAt: agora}, nil
}

// MembrosDe tira repetido e limita a lista.
func MembrosDe(membros []Membro) ([]Membro, error) {
	if len(membros) > 50 {
		return nil, EntradaInvalida{"a pasta cabe em 50 pessoas"}
	}
	vistos := map[uuid.UUID]bool{}
	out := make([]Membro, 0, len(membros))
	for _, m := range membros {
		if m.UsuarioID == uuid.Nil || vistos[m.UsuarioID] {
			return nil, EntradaInvalida{"informe cada pessoa uma vez"}
		}
		vistos[m.UsuarioID] = true
		out = append(out, Membro{UsuarioID: m.UsuarioID, PodeRevelar: m.PodeRevelar})
	}
	return out, nil
}
