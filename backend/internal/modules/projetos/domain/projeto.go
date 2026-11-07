package domain

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
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
	RelatoMax = 4000
)

var (
	ErrProjetoNaoEncontrado      = errors.New("projetos: projeto não encontrado")
	ErrTarefaNaoEncontrada       = errors.New("projetos: tarefa não encontrada")
	ErrTituloObrigatorio         = errors.New("projetos: título obrigatório (até 160 caracteres)")
	ErrTransicaoInvalida         = errors.New("projetos: passagem de estado inválida")
	ErrProjetoEncerrado          = errors.New("projetos: projeto cancelado não aceita alterações")
	ErrProjetoConcluido          = errors.New("projetos: projeto concluído só muda depois da reabertura")
	ErrTarefasAbertas            = errors.New("projetos: há tarefas em aberto")
	ErrPredecessoraInvalida      = errors.New("projetos: predecessora inválida")
	ErrPredecessoraAberta        = errors.New("projetos: a tarefa anterior ainda não foi concluída")
	ErrSubtarefaInvalida         = errors.New("projetos: subtarefa inválida")
	ErrSubtarefasAbertas         = errors.New("projetos: há subtarefas em aberto")
	ErrResponsavelObrigatorio    = errors.New("projetos: a tarefa precisa de um responsável para ser movida")
	ErrExclusaoNaoConfirmada     = errors.New("projetos: confirme o título da tarefa para apagar")
	ErrRelatoInvalido            = errors.New("projetos: relato longo demais")
	ErrAtualizacaoInvalida       = errors.New("projetos: atualização inválida")
	ErrAtualizacaoNaoAutorizada  = errors.New("projetos: atualização não autorizada")
	ErrMesInvalido               = errors.New("projetos: mês inválido")
	ErrAnexoNaoEncontrado        = errors.New("projetos: anexo não encontrado")
	ErrPrioridadeInvalida        = errors.New("projetos: prioridade inválida")
	ErrEstadoInvalido            = errors.New("projetos: estado inválido")
	ErrApontamentoInvalido       = errors.New("projetos: horas devem ser maiores que zero e até 24h, com descrição")
	ErrApontamentoNaoEncontrado  = errors.New("projetos: apontamento não encontrado")
	ErrForaDoEscopo              = errors.New("projetos: fora do seu escopo de gestão")
	ErrVinculoInvalido           = errors.New("projetos: tipo de vínculo inválido")
	ErrVinculoInexistente        = errors.New("projetos: item do vínculo não encontrado")
	ErrDeOutraUnidade            = errors.New("projetos: o vínculo ou produto precisa ser da mesma unidade do projeto")
	ErrIntegracaoDesligada       = errors.New("projetos: o módulo deste vínculo está desligado")
	ErrPeriodoInvalido           = errors.New("projetos: término antes do início")
	ErrOrcamentoInvalido         = errors.New("projetos: orçamento negativo")
	ErrEtapaInvalida             = errors.New("projetos: etapa inválida")
	ErrCartaInvalida             = errors.New("projetos: carta inválida")
	ErrPessoaInvalida            = errors.New("projetos: pessoa inválida")
	ErrReaberturaExigeAprovacao  = errors.New("projetos: concluído só reabre com aprovação")
	ErrReaberturaNaoAutorizada   = errors.New("projetos: reabertura não autorizada")
	ErrCancelamentoNaoAutorizado = errors.New("projetos: cancelamento não autorizado")
	ErrMotivoObrigatorio         = errors.New("projetos: motivo obrigatório")
)

const (
	// OrigemHow marca a tarefa nascida de uma etapa do passo Como.
	// Vazio é tarefa lançada à mão. Salvar a carta não mexe nisso.
	OrigemHow = "how"
)

// Alvo é o que o projeto precisa saber de um item de outro plug-in.
type Alvo struct {
	Rotulo    string
	UnidadeID *uuid.UUID
}

// Portas é como o projeto consulta os outros plug-ins (ligadas em
// internal/app). Nil libera o rótulo informado sem consultar a origem.
type Portas interface {
	Consultar(ctx context.Context, db database.DBTX, tipo string, id uuid.UUID) (Alvo, error)
	Disponivel(tipo string) bool
}

// HorasVinculadas soma as horas lançadas nos chamados ligados ao projeto.
// Nil deixa o painel só com o timesheet daqui. O chamado não tem custo:
// o número entra como hora, sem virar dinheiro.
type HorasVinculadas interface {
	Somar(ctx context.Context, db database.DBTX, solicitacoes []uuid.UUID) (float64, error)
}

// Produtos é a porta do plug-in de produtos.
type Produtos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (unidade *uuid.UUID, err error)
}

// NovoVinculo valida o tipo e o identificador do vínculo.
func NovoVinculo(tipo string, alvo uuid.UUID) error {
	if !slices.Contains([]string{VinculoMudanca, VinculoProblema, VinculoAtivo, VinculoSolicitacao}, tipo) || alvo == uuid.Nil {
		return ErrVinculoInvalido
	}
	return nil
}

// MesmaUnidadeVerifica se o recurso vinculado é compatível com a unidade do projeto
// (projetos ou alvos institucionais — nil — aceitam qualquer unidade; quando ambos
// têm unidade definida, precisam coincidir).
func MesmaUnidadeCompativel(projetoUnidade, alvoUnidade *uuid.UUID) bool {
	if projetoUnidade == nil || alvoUnidade == nil {
		return true
	}
	return *projetoUnidade == *alvoUnidade
}

var transicoesProjeto = map[string][]string{
	EstadoPlanejado:   {EstadoEmAndamento, EstadoCancelado},
	EstadoEmAndamento: {EstadoSuspenso, EstadoConcluido, EstadoCancelado},
	EstadoSuspenso:    {EstadoEmAndamento, EstadoCancelado},
	EstadoConcluido:   {},
	EstadoCancelado:   {},
}

// Projeto representa um projeto ou iniciativa institucional.
type Projeto struct {
	ID                  uuid.UUID     `json:"id"`
	Numero              int64         `json:"numero"`
	Titulo              string        `json:"titulo"`
	Descricao           string        `json:"descricao"`
	Objetivo            string        `json:"objetivo"`
	Estado              string        `json:"estado"`
	Prioridade          string        `json:"prioridade"`
	DataInicioPlanejada *time.Time    `json:"data_inicio_planejada,omitempty"`
	DataFimPlanejada    *time.Time    `json:"data_fim_planejada,omitempty"`
	DataInicioReal      *time.Time    `json:"data_inicio_real,omitempty"`
	DataFimReal         *time.Time    `json:"data_fim_real,omitempty"`
	Orcamento           float64       `json:"orcamento"`
	UnidadeID           *uuid.UUID    `json:"unidade_id,omitempty"`
	ProdutoID           *uuid.UUID    `json:"produto_id,omitempty"`
	GerenteID           *uuid.UUID    `json:"gerente_id,omitempty"`
	GerenteNome         string        `json:"gerente_nome"`
	TarefasTotal        int           `json:"tarefas_total"`
	TarefasConcluidas   int           `json:"tarefas_concluidas"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	Proximos            []string      `json:"proximos"`
	PodeReabrir         bool          `json:"pode_reabrir,omitempty"`
	Reabertura          *Reabertura   `json:"reabertura,omitempty"`
	Cancelamento        *Cancelamento `json:"cancelamento,omitempty"`
	Tarefas             []Tarefa      `json:"tarefas,omitempty"`
	Vinculos            []Vinculo     `json:"vinculos,omitempty"`
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
	ResponsavelID   *uuid.UUID `json:"responsavel_id,omitempty"`
	ResponsavelNome string     `json:"responsavel_nome"`
	Estado          string     `json:"estado"`
	Ordem           int        `json:"ordem"`
	DataLimite      *time.Time `json:"data_limite,omitempty"`
	Origem          string     `json:"origem,omitempty"`
	PredecessoraID  *uuid.UUID `json:"predecessora_id,omitempty"`
	PaiID           *uuid.UUID `json:"pai_id,omitempty"`
	Relato          string     `json:"relato"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TarefaMinha é uma tarefa aberta de quem está logado, com o projeto ao lado.
type TarefaMinha struct {
	Tarefa
	ProjetoNumero int64  `json:"projeto_numero"`
	ProjetoTitulo string `json:"projeto_titulo"`
	ProjetoEstado string `json:"projeto_estado"`
}

// CargaPessoa conta as tarefas abertas e as atrasadas de um responsável.
type CargaPessoa struct {
	ID        uuid.UUID `json:"id"`
	Nome      string    `json:"nome"`
	Abertas   int       `json:"abertas"`
	Atrasadas int       `json:"atrasadas"`
}

// PrazoTarefa é uma tarefa cujo limite já passou e o projeto ainda corre.
type PrazoTarefa struct {
	TarefaID      uuid.UUID
	Titulo        string
	DataLimite    time.Time
	ResponsavelID *uuid.UUID
	ProjetoID     uuid.UUID
	ProjetoNumero int64
	ProjetoTitulo string
	GerenteID     *uuid.UUID
}

// PrazoProjeto é um projeto cujo fim planejado já passou.
type PrazoProjeto struct {
	ID        uuid.UUID
	Numero    int64
	Titulo    string
	Fim       time.Time
	GerenteID *uuid.UUID
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
	if estado == EstadoCancelado || estado == EstadoConcluido {
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
	if atual == EstadoCancelado || atual == EstadoConcluido {
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
