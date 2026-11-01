package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	TituloMin = 3
	TituloMax = 160
	TextoMax  = 4000

	// Categorias ISO 27001 / GRC
	CategoriaSeguranca     = "seguranca_informacao"
	CategoriaInfra         = "infraestrutura"
	CategoriaFornecedores  = "fornecedores"
	CategoriaContinuidade  = "continuidade"
	CategoriaLGPD          = "conformidade_lgpd"
	CategoriaOperacional   = "operacional"

	// Estratégias de Tratamento
	EstrategiaMitigar    = "mitigar"
	EstrategiaAceitar    = "aceitar"
	EstrategiaTransferir = "transferir"
	EstrategiaEvitar     = "evitar"

	// Ciclo de Vida do Risco
	StatusIdentificado = "identificado"
	StatusEmAnalise    = "em_analise"
	StatusEmMitigacao  = "em_mitigacao"
	StatusMonitorado   = "monitorado"
	StatusResolvido    = "resolvido"
)

var (
	ErrRiscoInvalido      = errors.New("riscos: dados do risco inválidos")
	ErrRiscoNaoEncontrado = errors.New("riscos: risco não encontrado")
	ErrForaDoEscopo       = errors.New("riscos: fora do escopo de gestão da unidade")
)

// Risco representa um item da Matriz de Riscos de TI conforme a norma ISO/IEC 27001.
type Risco struct {
	ID              uuid.UUID  `json:"id"`
	Codigo          string     `json:"codigo"`
	Titulo          string     `json:"titulo"`
	Descricao       string     `json:"descricao"`
	Categoria       string     `json:"categoria"`
	AtivoID         *uuid.UUID `json:"ativo_id,omitempty"`
	UnidadeID       uuid.UUID  `json:"unidade_id"`
	Probabilidade   int        `json:"probabilidade"`
	Impacto         int        `json:"impacto"`
	Severidade      int        `json:"severidade"`
	Estrategia      string     `json:"estrategia"`
	PlanoMitigacao  string     `json:"plano_mitigacao"`
	MudancaID       *uuid.UUID `json:"mudanca_id,omitempty"`
	ProjetoID       *uuid.UUID `json:"projeto_id,omitempty"`
	ResponsavelID   *uuid.UUID `json:"responsavel_id,omitempty"`
	ResponsavelNome string     `json:"responsavel_nome"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// NivelSeveridade devolve a classificação textual do risco.
func (r Risco) NivelSeveridade() string {
	s := r.Probabilidade * r.Impacto
	switch {
	case s >= 20:
		return "critico"
	case s >= 13:
		return "alto"
	case s >= 7:
		return "medio"
	default:
		return "baixo"
	}
}

// NovoRisco valida e instancia um risco conforme a ISO 27001.
func NovoRisco(codigo, titulo, descricao, categoria string, unidadeID uuid.UUID, prob, imp int, estrategia, plano string, respID *uuid.UUID, respNome string) (Risco, error) {
	now := time.Now().UTC()
	r := Risco{
		ID:        uuid.New(),
		Status:    StatusIdentificado,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.Aplicar(codigo, titulo, descricao, categoria, unidadeID, prob, imp, estrategia, plano, respID, respNome, StatusIdentificado)
}

// Aplicar valida os limites e aplica alterações no risco.
func (r Risco) Aplicar(codigo, titulo, descricao, categoria string, unidadeID uuid.UUID, prob, imp int, estrategia, plano string, respID *uuid.UUID, respNome string, status string) (Risco, error) {
	codigo = strings.TrimSpace(codigo)
	titulo = strings.TrimSpace(titulo)
	descricao = strings.TrimSpace(descricao)
	plano = strings.TrimSpace(plano)
	respNome = strings.TrimSpace(respNome)

	if codigo == "" || utf8.RuneCountInString(titulo) < TituloMin || utf8.RuneCountInString(titulo) > TituloMax {
		return Risco{}, ErrRiscoInvalido
	}
	if utf8.RuneCountInString(descricao) > TextoMax || utf8.RuneCountInString(plano) > TextoMax {
		return Risco{}, ErrRiscoInvalido
	}
	if prob < 1 || prob > 5 || imp < 1 || imp > 5 {
		return Risco{}, ErrRiscoInvalido
	}
	if unidadeID == uuid.Nil {
		return Risco{}, ErrRiscoInvalido
	}

	r.Codigo = codigo
	r.Titulo = titulo
	r.Descricao = descricao
	r.Categoria = categoria
	r.UnidadeID = unidadeID
	r.Probabilidade = prob
	r.Impacto = imp
	r.Severidade = prob * imp
	r.Estrategia = estrategia
	r.PlanoMitigacao = plano
	r.ResponsavelID = respID
	r.ResponsavelNome = respNome
	if status != "" {
		r.Status = status
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

