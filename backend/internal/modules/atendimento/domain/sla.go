package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	SLANomeMax           = 120
	SLADescricaoMax      = 500
	CalendarioExpediente = "expediente"
	CalendarioCorrido    = "corrido_24x7"
)

var (
	ErrSLAInvalida      = errors.New("atendimento: SLA inválida")
	ErrSLANaoEncontrada = errors.New("atendimento: SLA não encontrada")
)

// SLA representa a configuração independente de nível de serviço (ITIL v5 / GLPI).
// Define tempo de primeira resposta e tempo de solução por categoria ou geral.
type SLA struct {
	ID                   uuid.UUID  `json:"id"`
	Nome                 string     `json:"nome"`
	Descricao            string     `json:"descricao"`
	TempoRespostaMinutos int        `json:"tempo_resposta_minutos"`
	TempoSolucaoMinutos  int        `json:"tempo_solucao_minutos"`
	PausarEmEspera       bool       `json:"pausar_em_espera"`
	CalendarioTipo       string     `json:"calendario_tipo"`
	UnidadeID            *uuid.UUID `json:"unidade_id,omitempty"`
	Ativa                bool       `json:"ativa"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// NovaSLA constrói e valida uma nova SLA.
func NovaSLA(nome, descricao string, tempoRespostaMin, tempoSolucaoMin int, pausarEmEspera bool, calendarioTipo string, unidadeID *uuid.UUID) (SLA, error) {
	now := time.Now().UTC()
	s := SLA{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Ativa:     true,
	}
	return s.Aplicar(nome, descricao, tempoRespostaMin, tempoSolucaoMin, pausarEmEspera, calendarioTipo, unidadeID, true)
}

// Aplicar valida e altera os campos de uma SLA existente.
func (s SLA) Aplicar(nome, descricao string, tempoRespostaMin, tempoSolucaoMin int, pausarEmEspera bool, calendarioTipo string, unidadeID *uuid.UUID, ativa bool) (SLA, error) {
	nome = strings.TrimSpace(nome)
	descricao = strings.TrimSpace(descricao)
	if nome == "" || utf8.RuneCountInString(nome) > SLANomeMax || utf8.RuneCountInString(descricao) > SLADescricaoMax {
		return SLA{}, ErrSLAInvalida
	}
	if tempoRespostaMin <= 0 || tempoSolucaoMin <= 0 || tempoRespostaMin > MetaMaxMin || tempoSolucaoMin > MetaMaxMin {
		return SLA{}, ErrSLAInvalida
	}
	if tempoRespostaMin > tempoSolucaoMin {
		return SLA{}, ErrSLAInvalida
	}
	if calendarioTipo != CalendarioExpediente && calendarioTipo != CalendarioCorrido {
		calendarioTipo = CalendarioExpediente
	}
	s.Nome = nome
	s.Descricao = descricao
	s.TempoRespostaMinutos = tempoRespostaMin
	s.TempoSolucaoMinutos = tempoSolucaoMin
	s.PausarEmEspera = pausarEmEspera
	s.CalendarioTipo = calendarioTipo
	s.UnidadeID = unidadeID
	s.Ativa = ativa
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

// AplicarSLA aplica as metas da SLA diretamente sobre a solicitação.
func (s Solicitacao) AplicarSLA(sla *SLA) Solicitacao {
	if sla == nil || !sla.Ativa {
		return s
	}
	s.MetaRespostaMin = &sla.TempoRespostaMinutos
	s.MetaSolucaoMin = &sla.TempoSolucaoMinutos
	return s
}

