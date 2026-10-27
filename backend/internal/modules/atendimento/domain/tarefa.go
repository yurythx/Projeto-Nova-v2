package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Tarefas por equipe da requisição (ADR 068).
const (
	TarefaPendente   = "pendente"
	TarefaConcluida  = "concluida"
	TarefaDispensada = "dispensada"

	MaxTarefas      = 20
	maxTituloTarefa = 120
	MaxObservacao   = 1000
)

var (
	ErrTarefasSoRequisicao = errors.New("atendimento: só requisição tem tarefas")
	ErrModeloInvalido      = errors.New("atendimento: modelo de tarefas inválido")
	ErrTarefaFeita         = errors.New("atendimento: a tarefa já foi concluída ou dispensada")
	ErrNaoEhDaEquipe       = errors.New("atendimento: a tarefa é de outra equipe")
	ErrDispensaSemMotivo   = errors.New("atendimento: dispensar precisa de motivo, e a observação cabe em 1000 caracteres")
	ErrTarefasPendentes    = errors.New("atendimento: há tarefas pendentes")
)

// ModeloTarefa é uma linha do modelo da oferta: o que fazer e qual equipe faz.
type ModeloTarefa struct {
	Ordem      int       `json:"ordem"`
	Titulo     string    `json:"titulo"`
	EquipeID   uuid.UUID `json:"equipe_id"`
	EquipeNome string    `json:"equipe_nome,omitempty"`
}

// Tarefa é o trabalho de uma equipe num pedido.
type Tarefa struct {
	ID            uuid.UUID  `json:"id"`
	SolicitacaoID uuid.UUID  `json:"solicitacao_id"`
	Ordem         int        `json:"ordem"`
	Titulo        string     `json:"titulo"`
	EquipeID      *uuid.UUID `json:"equipe_id,omitempty"`
	EquipeNome    string     `json:"equipe_nome"`
	Estado        string     `json:"estado"`
	Observacao    string     `json:"observacao,omitempty"`
	FeitaPor      *uuid.UUID `json:"feita_por,omitempty"`
	FeitaPorNome  string     `json:"feita_por_nome,omitempty"`
	FeitaEm       *time.Time `json:"feita_em,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	// Minha diz se quem consulta é da equipe da tarefa e pode concluí-la.
	Minha bool `json:"minha"`
}

// ValidarModelo confere o modelo da oferta e o numera em ordem. Lista
// vazia tira as tarefas da oferta.
func ValidarModelo(tipo string, modelo []ModeloTarefa) ([]ModeloTarefa, error) {
	if len(modelo) == 0 {
		return []ModeloTarefa{}, nil
	}
	if tipo != TipoRequisicao {
		return nil, ErrTarefasSoRequisicao
	}
	if len(modelo) > MaxTarefas {
		return nil, ErrModeloInvalido
	}
	out := make([]ModeloTarefa, 0, len(modelo))
	for i, m := range modelo {
		titulo := strings.TrimSpace(m.Titulo)
		if titulo == "" || utf8.RuneCountInString(titulo) > maxTituloTarefa || m.EquipeID == uuid.Nil {
			return nil, ErrModeloInvalido
		}
		out = append(out, ModeloTarefa{Ordem: i, Titulo: titulo, EquipeID: m.EquipeID, EquipeNome: m.EquipeNome})
	}
	return out, nil
}

// TarefasDoModelo cria as tarefas do pedido a partir do modelo.
func TarefasDoModelo(sol uuid.UUID, modelo []ModeloTarefa, agora time.Time) []Tarefa {
	out := make([]Tarefa, 0, len(modelo))
	for _, m := range modelo {
		equipe := m.EquipeID
		out = append(out, Tarefa{ID: uuid.New(), SolicitacaoID: sol, Ordem: m.Ordem, Titulo: m.Titulo,
			EquipeID: &equipe, EquipeNome: m.EquipeNome, Estado: TarefaPendente, CreatedAt: agora})
	}
	return out
}

// Fazer conclui (com observação opcional) ou dispensa (com motivo) a
// tarefa pendente. Concluir é de quem é da equipe; dispensar, de quem
// atende o chamado.
func (t Tarefa) Fazer(quem uuid.UUID, daEquipe, concluir bool, observacao string, agora time.Time) (Tarefa, error) {
	if t.Estado != TarefaPendente {
		return Tarefa{}, ErrTarefaFeita
	}
	observacao = strings.TrimSpace(observacao)
	if utf8.RuneCountInString(observacao) > MaxObservacao {
		return Tarefa{}, ErrDispensaSemMotivo
	}
	if concluir {
		if !daEquipe {
			return Tarefa{}, ErrNaoEhDaEquipe
		}
		t.Estado = TarefaConcluida
	} else {
		if observacao == "" {
			return Tarefa{}, ErrDispensaSemMotivo
		}
		t.Estado = TarefaDispensada
	}
	t.Observacao, t.FeitaPor, t.FeitaEm = observacao, &quem, &agora
	return t, nil
}

// Pendentes conta as tarefas que ainda impedem dar o pedido por atendido.
func Pendentes(tarefas []Tarefa) int {
	n := 0
	for _, t := range tarefas {
		if t.Estado == TarefaPendente {
			n++
		}
	}
	return n
}
