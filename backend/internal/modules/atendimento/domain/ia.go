package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Tarefas da IA assistiva (ADR 065). A IA sugere a quem atende; nada no
// chamado muda por causa dela.
const (
	SugestaoClassificacao = "classificacao"
	SugestaoResumo        = "resumo"

	DecisaoAceita   = "aceita"
	DecisaoRecusada = "recusada"

	maxMotivo = 300
	maxResumo = 1500
)

var (
	// ErrSemIA: a função atendimento.assistente não tem conexão.
	ErrSemIA = errors.New("atendimento: IA não configurada")
	// ErrRespostaDaIA: a IA falhou ou respondeu fora do formato pedido.
	ErrRespostaDaIA = errors.New("atendimento: a IA não deu uma resposta aproveitável")
	// ErrSugestaoDecidida: a sugestão só é decidida uma vez.
	ErrSugestaoDecidida = errors.New("atendimento: esta sugestão já foi decidida")
)

// ClassificacaoSugerida é o impacto e a urgência que a IA propõe, com o
// motivo que quem atende lê antes de decidir.
type ClassificacaoSugerida struct {
	Impacto  int    `json:"impacto"`
	Urgencia int    `json:"urgencia"`
	Motivo   string `json:"motivo"`
}

// Sugestao é uma resposta da IA a quem atende e a decisão sobre ela.
type Sugestao struct {
	ID            uuid.UUID              `json:"id"`
	SolicitacaoID uuid.UUID              `json:"solicitacao_id"`
	Tipo          string                 `json:"tipo"`
	Conexao       string                 `json:"conexao"`
	Classificacao *ClassificacaoSugerida `json:"classificacao,omitempty"`
	Resumo        string                 `json:"resumo,omitempty"`
	PedidaPor     *uuid.UUID             `json:"pedida_por,omitempty"`
	Decisao       *string                `json:"decisao,omitempty"`
	DecididaPor   *uuid.UUID             `json:"decidida_por,omitempty"`
	DecididaEm    *time.Time             `json:"decidida_em,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// LerClassificacao lê o objeto JSON da resposta. Modelos pequenos põem
// texto em volta: vale o trecho entre a primeira "{" e a última "}".
// Fora de 1 a 3, ou sem motivo, não é sugestão.
func LerClassificacao(texto string) (ClassificacaoSugerida, error) {
	ini, fim := strings.Index(texto, "{"), strings.LastIndex(texto, "}")
	if ini < 0 || fim < ini {
		return ClassificacaoSugerida{}, ErrRespostaDaIA
	}
	var c ClassificacaoSugerida
	if err := json.Unmarshal([]byte(texto[ini:fim+1]), &c); err != nil {
		return ClassificacaoSugerida{}, ErrRespostaDaIA
	}
	c.Motivo = cortar(strings.TrimSpace(c.Motivo), maxMotivo)
	if c.Impacto < NivelMin || c.Impacto > NivelMax || c.Urgencia < NivelMin || c.Urgencia > NivelMax || c.Motivo == "" {
		return ClassificacaoSugerida{}, ErrRespostaDaIA
	}
	return c, nil
}

// LerResumo aceita o texto da IA, sem linhas em branco nas pontas.
func LerResumo(texto string) (string, error) {
	t := strings.TrimSpace(texto)
	if t == "" {
		return "", ErrRespostaDaIA
	}
	return cortar(t, maxResumo), nil
}

// Decidir registra se quem atende aceitou ou recusou. Uma vez só.
func (s Sugestao) Decidir(aceita bool, quem uuid.UUID, agora time.Time) (Sugestao, error) {
	if s.Decisao != nil {
		return Sugestao{}, ErrSugestaoDecidida
	}
	d := DecisaoRecusada
	if aceita {
		d = DecisaoAceita
	}
	s.Decisao, s.DecididaPor, s.DecididaEm = &d, &quem, &agora
	return s, nil
}

// IndicadorIA conta as sugestões de uma tarefa e o que foi decidido.
type IndicadorIA struct {
	Tipo      string `json:"tipo"`
	Total     int    `json:"total"`
	Aceitas   int    `json:"aceitas"`
	Recusadas int    `json:"recusadas"`
	Pendentes int    `json:"pendentes"`
}

func cortar(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max])) + "…"
}
