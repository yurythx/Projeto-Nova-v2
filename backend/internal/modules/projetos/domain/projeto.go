package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	EstadoPlanejado   = "planejado"
	EstadoEmAndamento = "em_andamento"
	EstadoSuspenso    = "suspenso"
	EstadoConcluido   = "concluido"
	EstadoCancelado   = "cancelado"

	VinculoMudanca     = "mudanca"
	VinculoProblema    = "problema"
	VinculoAtivo       = "ativo"
	VinculoSolicitacao = "solicitacao"

	PrioridadeBaixa   = "baixa"
	PrioridadeMedia   = "media"
	PrioridadeAlta    = "alta"
	PrioridadeCritica = "critica"

	EstadoTarefaAFazer      = "a_fazer"
	EstadoTarefaEmAndamento = "em_andamento"
	EstadoTarefaConcluida   = "concluida"
	EstadoTarefaBloqueada   = "bloqueada"

	TituloMax = 160
)

var (
	ErrProjetoNaoEncontrado     = errors.New("projetos: projeto não encontrado")
	ErrTarefaNaoEncontrada      = errors.New("projetos: tarefa não encontrada")
	ErrTituloObrigatorio        = errors.New("projetos: título obrigatório (até 160 caracteres)")
	ErrTransicaoInvalida        = errors.New("projetos: passagem de estado inválida")
	ErrProjetoEncerrado         = errors.New("projetos: projeto cancelado não aceita alterações")
	ErrPrioridadeInvalida       = errors.New("projetos: prioridade inválida")
	ErrEstadoInvalido           = errors.New("projetos: estado inválido")
	ErrApontamentoInvalido      = errors.New("projetos: horas devem ser maiores que zero e até 24h, com descrição")
	ErrApontamentoNaoEncontrado = errors.New("projetos: apontamento não encontrado")
)

var transicoesProjeto = map[string][]string{
	EstadoPlanejado:   {EstadoEmAndamento, EstadoCancelado},
	EstadoEmAndamento: {EstadoSuspenso, EstadoConcluido, EstadoCancelado},
	EstadoSuspenso:    {EstadoEmAndamento, EstadoCancelado},
	EstadoConcluido:   {EstadoEmAndamento},
	EstadoCancelado:   {},
}

// Projeto representa um projeto ou iniciativa institucional.
type Projeto struct {
	ID                  uuid.UUID  `json:"id"`
	Numero              int64      `json:"numero"`
	Titulo              string     `json:"titulo"`
	Descricao           string     `json:"descricao"`
	Objetivo            string     `json:"objetivo"`
	Estado              string     `json:"estado"`
	Prioridade          string     `json:"prioridade"`
	DataInicioPlanejada *time.Time `json:"data_inicio_planejada,omitempty"`
	DataFimPlanejada    *time.Time `json:"data_fim_planejada,omitempty"`
	DataInicioReal      *time.Time `json:"data_inicio_real,omitempty"`
	DataFimReal         *time.Time `json:"data_fim_real,omitempty"`
	Orcamento           float64    `json:"orcamento"`
	UnidadeID           *uuid.UUID `json:"unidade_id,omitempty"`
	ProdutoID           *uuid.UUID `json:"produto_id,omitempty"`
	GerenteID           *uuid.UUID `json:"gerente_id,omitempty"`
	GerenteNome         string     `json:"gerente_nome"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Proximos            []string   `json:"proximos"`
	Tarefas             []Tarefa   `json:"tarefas,omitempty"`
	Vinculos            []Vinculo  `json:"vinculos,omitempty"`
}

// Vinculo representa a ligação do projeto com uma mudança, problema, ativo ou chamado.
type Vinculo struct {
	ID           uuid.UUID `json:"id"`
	ProjetoID    uuid.UUID `json:"projeto_id"`
	Tipo         string    `json:"tipo"`
	AlvoID       uuid.UUID `json:"alvo_id"`
	Rotulo       string    `json:"rotulo"`
	VinculadoPor string    `json:"vinculado_por"`
	CreatedAt    time.Time `json:"created_at"`
}

// Tarefa representa uma atividade ou marco do projeto.
type Tarefa struct {
	ID              uuid.UUID  `json:"id"`
	ProjetoID       uuid.UUID  `json:"projeto_id"`
	Titulo          string     `json:"titulo"`
	Descricao       string     `json:"descricao"`
	ResponsavelNome string     `json:"responsavel_nome"`
	Estado          string     `json:"estado"`
	Ordem           int        `json:"ordem"`
	DataLimite      *time.Time `json:"data_limite,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// EntradaProjeto reúne os campos de criação e atualização.
type EntradaProjeto struct {
	Titulo              string
	Descricao           string
	Objetivo            string
	Prioridade          string
	DataInicioPlanejada *time.Time
	DataFimPlanejada    *time.Time
	DataInicioReal      *time.Time
	DataFimReal         *time.Time
	Orcamento           float64
	UnidadeID           *uuid.UUID
	ProdutoID           *uuid.UUID
	GerenteID           *uuid.UUID
	GerenteNome         string
}

// Validar valida as invariantes de entrada.
func (e EntradaProjeto) Validar() error {
	t := strings.TrimSpace(e.Titulo)
	if t == "" || utf8.RuneCountInString(t) > TituloMax {
		return ErrTituloObrigatorio
	}
	if e.Prioridade != "" {
		switch e.Prioridade {
		case PrioridadeBaixa, PrioridadeMedia, PrioridadeAlta, PrioridadeCritica:
		default:
			return ErrPrioridadeInvalida
		}
	}
	return nil
}

// ProximosEstados retorna as transições permitidas para o estado atual.
var estadosTarefa = []string{EstadoTarefaAFazer, EstadoTarefaEmAndamento, EstadoTarefaConcluida, EstadoTarefaBloqueada}

func tarefaFabrica(estado string) bool {
	return estado == "" || slices.Contains(estadosTarefa, estado)
}

// EstadoDeTarefaAceito libera os quatro estados de fábrica entre si. A
// coluna nova só entra ou sai por uma passagem deste quadro.
func EstadoDeTarefaAceito(atual, novo string, mapa map[string][]string) bool {
	if novo == "" {
		novo = EstadoTarefaAFazer
	}
	if atual == "" || atual == novo {
		if tarefaFabrica(novo) {
			return true
		}
		for _, destinos := range mapa {
			if slices.Contains(destinos, novo) {
				return true
			}
		}
		return false
	}
	if tarefaFabrica(atual) && tarefaFabrica(novo) {
		return true
	}
	return slices.Contains(mapa[atual], novo)
}

func ProximosEstados(estado string, extras ...string) []string {
	if estado == EstadoCancelado {
		return nil
	}
	out := append([]string{}, transicoesProjeto[estado]...)
	for _, extra := range extras {
		if extra != "" && extra != estado && !slices.Contains(out, extra) {
			out = append(out, extra)
		}
	}
	return out
}

// ValidarTransicao verifica se a mudança de estado é permitida pela máquina de estados.
func ValidarTransicao(atual, novo string, extras ...string) bool {
	if atual == EstadoCancelado {
		return false
	}
	for _, p := range ProximosEstados(atual, extras...) {
		if p == novo {
			return true
		}
	}
	return false
}

// Apontamento representa o registro de esforço (horas trabalhadas) em um projeto ou tarefa.
type Apontamento struct {
	ID              uuid.UUID  `json:"id"`
	ProjetoID       uuid.UUID  `json:"projeto_id"`
	TarefaID        *uuid.UUID `json:"tarefa_id,omitempty"`
	TarefaTitulo    string     `json:"tarefa_titulo,omitempty"`
	UsuarioID       uuid.UUID  `json:"usuario_id"`
	UsuarioNome     string     `json:"usuario_nome"`
	DataApontamento string     `json:"data_apontamento"` // Formato "YYYY-MM-DD"
	Horas           float64    `json:"horas"`
	Descricao       string     `json:"descricao"`
	CustoHora       float64    `json:"custo_hora"`
	CustoTotal      float64    `json:"custo_total"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Validar verifica se o apontamento possui valores coerentes.
func (a Apontamento) Validar() error {
	if a.Horas <= 0 || a.Horas > 24 {
		return ErrApontamentoInvalido
	}
	if strings.TrimSpace(a.Descricao) == "" {
		return ErrApontamentoInvalido
	}
	return nil
}

// ResumoHorasProjeto consolida as horas e custos de um projeto.
type ResumoHorasProjeto struct {
	TotalHoras float64            `json:"total_horas"`
	TotalCusto float64            `json:"total_custo"`
	PorUsuario map[string]float64 `json:"por_usuario"`
}
