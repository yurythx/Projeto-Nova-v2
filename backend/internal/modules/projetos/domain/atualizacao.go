package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const AtualizacaoMax = 2000

var mesAgenda = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Atualizacao é um comentário no fio da tarefa. Não substitui o relato.
type Atualizacao struct {
	ID        uuid.UUID  `json:"id"`
	TarefaID  uuid.UUID  `json:"tarefa_id"`
	Texto     string     `json:"texto"`
	AutorID   *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome string     `json:"autor_nome"`
	CreatedAt time.Time  `json:"created_at"`
}

// PrazoAgenda é uma tarefa ou subtarefa com data-limite, para o calendário.
type PrazoAgenda struct {
	ID              uuid.UUID  `json:"id"`
	Titulo          string     `json:"titulo"`
	Estado          string     `json:"estado"`
	DataLimite      time.Time  `json:"data_limite"`
	ResponsavelID   *uuid.UUID `json:"responsavel_id,omitempty"`
	ResponsavelNome string     `json:"responsavel_nome"`
	Pessoas         []Pessoa   `json:"pessoas,omitempty"`
	PaiID           *uuid.UUID `json:"pai_id,omitempty"`
	ProjetoID       uuid.UUID  `json:"projeto_id"`
	ProjetoNumero   int64      `json:"projeto_numero"`
	ProjetoTitulo   string     `json:"projeto_titulo"`
}

// ValidarAtualizacao recusa texto vazio ou maior que o limite.
func ValidarAtualizacao(texto string) (string, error) {
	limpo := strings.TrimSpace(texto)
	if limpo == "" || utf8.RuneCountInString(limpo) > AtualizacaoMax {
		return "", ErrAtualizacaoInvalida
	}
	return limpo, nil
}

// MesValido aceita AAAA-MM.
func MesValido(mes string) error {
	if !mesAgenda.MatchString(mes) {
		return ErrMesInvalido
	}
	return nil
}
