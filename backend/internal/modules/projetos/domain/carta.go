package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	DescricaoMax         = 4000
	ObjetivoMax          = 4000
	GerenteNomeMax       = 200
	CartaOndeMax         = 2000
	CartaPatrocinadorMax = 400
	CartaMetodologiaMax  = 4000
	CartaCentroMax       = 120
	EtapaMax             = 200
	EtapasMax            = 30
)

// Carta é o plano diretor 5W2H que não cabe nas colunas do projeto.
// What, why, when, who (líder) e how much (orçamento) continuam em projetos.
type Carta struct {
	ProjetoID    uuid.UUID `json:"projeto_id"`
	Onde         string    `json:"onde"`
	Patrocinador string    `json:"patrocinador"`
	Metodologia  string    `json:"metodologia"`
	CentroCusto  string    `json:"centro_custo"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EntradaCarta são os quatro textos que só existem na carta.
type EntradaCarta struct {
	Onde         string
	Patrocinador string
	Metodologia  string
	CentroCusto  string
}

// Plano5W2H é o que a ficha grava de uma vez. Não inclui prioridade,
// unidade, produto nem estado: esses campos ficam como estavam.
type Plano5W2H struct {
	Titulo       string
	Descricao    string
	Objetivo     string
	GerenteID    *uuid.UUID
	GerenteNome  string
	Orcamento    float64
	Inicio       *time.Time
	Fim          *time.Time
	Onde         string
	Patrocinador string
	Metodologia  string
	CentroCusto  string
}

// VisaoCarta é a leitura do painel: campos do projeto, da carta e os
// dois números que a tela desenha (etapas do Como e custo do timesheet).
type VisaoCarta struct {
	ProjetoID           uuid.UUID  `json:"projeto_id"`
	Titulo              string     `json:"titulo"`
	Descricao           string     `json:"descricao"`
	Objetivo            string     `json:"objetivo"`
	Onde                string     `json:"onde"`
	Patrocinador        string     `json:"patrocinador"`
	Metodologia         string     `json:"metodologia"`
	CentroCusto         string     `json:"centro_custo"`
	DataInicioPlanejada *time.Time `json:"data_inicio_planejada,omitempty"`
	DataFimPlanejada    *time.Time `json:"data_fim_planejada,omitempty"`
	UnidadeID           *uuid.UUID `json:"unidade_id,omitempty"`
	GerenteID           *uuid.UUID `json:"gerente_id,omitempty"`
	GerenteNome         string     `json:"gerente_nome"`
	Orcamento           float64    `json:"orcamento"`
	Existe              bool       `json:"existe"`
	EtapasHow           int        `json:"etapas_how"`
	EtapasHowConcluidas int        `json:"etapas_how_concluidas"`
	CustoRealizado      float64    `json:"custo_realizado"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
}

// Normalizar corta espaço e recusa texto longo demais.
func (c EntradaCarta) Normalizar() (EntradaCarta, error) {
	c.Onde = strings.TrimSpace(c.Onde)
	c.Patrocinador = strings.TrimSpace(c.Patrocinador)
	c.Metodologia = strings.TrimSpace(c.Metodologia)
	c.CentroCusto = strings.TrimSpace(c.CentroCusto)
	if utf8.RuneCountInString(c.Onde) > CartaOndeMax ||
		utf8.RuneCountInString(c.Patrocinador) > CartaPatrocinadorMax ||
		utf8.RuneCountInString(c.Metodologia) > CartaMetodologiaMax ||
		utf8.RuneCountInString(c.CentroCusto) > CartaCentroMax {
		return EntradaCarta{}, ErrCartaInvalida
	}
	return c, nil
}

// NormalizarEtapas descarta linhas em branco. Cada etapa vira uma tarefa.
func NormalizarEtapas(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if utf8.RuneCountInString(s) > EtapaMax {
			return nil, ErrEtapaInvalida
		}
		out = append(out, s)
	}
	if len(out) > EtapasMax {
		return nil, ErrEtapaInvalida
	}
	return out, nil
}

func validarTextosPlano(descricao, objetivo, gerente string, orcamento float64, inicio, fim *time.Time) error {
	if utf8.RuneCountInString(descricao) > DescricaoMax || utf8.RuneCountInString(objetivo) > ObjetivoMax || utf8.RuneCountInString(gerente) > GerenteNomeMax {
		return ErrCartaInvalida
	}
	if orcamento < 0 {
		return ErrOrcamentoInvalido
	}
	if inicio != nil && fim != nil && fim.Before(*inicio) {
		return ErrPeriodoInvalido
	}
	return nil
}

// ValidarAbertura confere o projeto guiado antes de abrir a transação.
func ValidarAbertura(e EntradaProjeto, c EntradaCarta, etapas []string) (EntradaCarta, []string, error) {
	if err := e.Validar(); err != nil {
		return EntradaCarta{}, nil, err
	}
	if err := validarTextosPlano(e.Descricao, e.Objetivo, e.GerenteNome, e.Orcamento, e.DataInicioPlanejada, e.DataFimPlanejada); err != nil {
		return EntradaCarta{}, nil, err
	}
	c, err := c.Normalizar()
	if err != nil {
		return EntradaCarta{}, nil, err
	}
	etapas, err = NormalizarEtapas(etapas)
	if err != nil {
		return EntradaCarta{}, nil, err
	}
	return c, etapas, nil
}

// Normalizar prepara o salvamento da ficha. Não devolve etapas: editar a
// carta não cria tarefa.
func (p Plano5W2H) Normalizar() (Plano5W2H, EntradaCarta, error) {
	p.Titulo = strings.TrimSpace(p.Titulo)
	p.Descricao = strings.TrimSpace(p.Descricao)
	p.Objetivo = strings.TrimSpace(p.Objetivo)
	p.GerenteNome = strings.TrimSpace(p.GerenteNome)
	if err := (EntradaProjeto{Titulo: p.Titulo}).Validar(); err != nil {
		return Plano5W2H{}, EntradaCarta{}, err
	}
	if err := validarTextosPlano(p.Descricao, p.Objetivo, p.GerenteNome, p.Orcamento, p.Inicio, p.Fim); err != nil {
		return Plano5W2H{}, EntradaCarta{}, err
	}
	c, err := (EntradaCarta{
		Onde: p.Onde, Patrocinador: p.Patrocinador, Metodologia: p.Metodologia, CentroCusto: p.CentroCusto,
	}).Normalizar()
	if err != nil {
		return Plano5W2H{}, EntradaCarta{}, err
	}
	p.Onde = c.Onde
	p.Patrocinador = c.Patrocinador
	p.Metodologia = c.Metodologia
	p.CentroCusto = c.CentroCusto
	return p, c, nil
}
