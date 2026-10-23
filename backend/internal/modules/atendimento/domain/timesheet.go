package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrApontamentoInvalido = errors.New("atendimento: apontamento de horas inválido")
)

// Apontamento representa o registro de esforço (horas trabalhadas) em uma solicitação de atendimento.
type Apontamento struct {
	ID              uuid.UUID `json:"id"`
	SolicitacaoID   uuid.UUID `json:"solicitacao_id"`
	UsuarioID       uuid.UUID `json:"usuario_id"`
	UsuarioNome     string    `json:"usuario_nome"`
	DataApontamento string    `json:"data_apontamento"` // Formato "YYYY-MM-DD"
	Horas           float64   `json:"horas"`
	Descricao       string    `json:"descricao"`
	CreatedAt       time.Time `json:"created_at"`
}

// EntradaApontamento reúne os dados recebidos para registrar horas na solicitação.
type EntradaApontamento struct {
	DataApontamento string  `json:"data_apontamento"`
	Horas           float64 `json:"horas"`
	Descricao       string  `json:"descricao"`
}

// Validar verifica se os dados do apontamento são válidos.
func (e EntradaApontamento) Validar() error {
	if e.Horas <= 0 || e.Horas > 24 {
		return ErrApontamentoInvalido
	}
	if strings.TrimSpace(e.Descricao) == "" {
		return ErrApontamentoInvalido
	}
	return nil
}

// ResumoApontamentos consolida os apontamentos e o total de horas gastas na solicitação.
type ResumoApontamentos struct {
	TotalHoras   float64       `json:"total_horas"`
	Apontamentos []Apontamento `json:"apontamentos"`
}
