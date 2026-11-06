package domain

import "github.com/google/uuid"

// RecusarAlteracao trava a ficha cancelada e a concluída.
// Concluído volta só pela reabertura.
func RecusarAlteracao(estado string) error {
	switch estado {
	case EstadoCancelado:
		return ErrProjetoEncerrado
	case EstadoConcluido:
		return ErrProjetoConcluido
	default:
		return nil
	}
}

// TarefasEmAberto conta o que ainda não está concluído.
// Coluna extra também conta: só o estado de fábrica concluida encerra a tarefa.
func TarefasEmAberto(tarefas []Tarefa) int {
	n := 0
	for _, t := range tarefas {
		if t.Estado != EstadoTarefaConcluida {
			n++
		}
	}
	return n
}

// ValidarCadeia recusa a própria tarefa, uma tarefa de outro projeto e o ciclo.
// tarefas são as do projeto, com a predecessora que já está gravada.
func ValidarCadeia(tarefa, predecessora uuid.UUID, tarefas []Tarefa) error {
	if predecessora == uuid.Nil {
		return nil
	}
	if predecessora == tarefa {
		return ErrPredecessoraInvalida
	}
	pai := make(map[uuid.UUID]*uuid.UUID, len(tarefas))
	for i := range tarefas {
		pai[tarefas[i].ID] = tarefas[i].PredecessoraID
	}
	if _, ok := pai[predecessora]; !ok {
		return ErrPredecessoraInvalida
	}
	visto := map[uuid.UUID]bool{}
	cur := predecessora
	for {
		if cur == tarefa || visto[cur] {
			return ErrPredecessoraInvalida
		}
		visto[cur] = true
		prox, ok := pai[cur]
		if !ok || prox == nil || *prox == uuid.Nil {
			return nil
		}
		cur = *prox
	}
}

// PodeSerPai aceita uma tarefa do mesmo projeto que ainda não é subtarefa.
// Subtarefa de subtarefa não existe.
func PodeSerPai(pai Tarefa, projeto uuid.UUID) error {
	if pai.ProjetoID != projeto || pai.PaiID != nil {
		return ErrSubtarefaInvalida
	}
	return nil
}

// ConclusaoExigeSubtarefas trava a conclusão da tarefa enquanto alguma filha
// estiver aberta. Sem filhas, a tarefa conclui como sempre.
func ConclusaoExigeSubtarefas(estado string, filhas []Tarefa) error {
	if estado != EstadoTarefaConcluida {
		return nil
	}
	for _, filha := range filhas {
		if filha.Estado != EstadoTarefaConcluida {
			return ErrSubtarefasAbertas
		}
	}
	return nil
}

// ConferirPredecessora só trava a conclusão. Várias tarefas podem estar
// em andamento ao mesmo tempo; a anterior precisa estar concluída para
// esta ser concluída.
func ConferirPredecessora(estado string, predecessora *Tarefa) error {
	if estado != EstadoTarefaConcluida || predecessora == nil {
		return nil
	}
	if predecessora.Estado != EstadoTarefaConcluida {
		return ErrPredecessoraAberta
	}
	return nil
}

// MovimentoExigeResponsavel libera criar e editar em "a fazer" sem pessoa.
// Qualquer passagem de coluna exige um usuário responsável.
func MovimentoExigeResponsavel(antes, depois string, responsavel *uuid.UUID) error {
	if antes == "" {
		antes = EstadoTarefaAFazer
	}
	if depois == "" {
		depois = EstadoTarefaAFazer
	}
	if antes == depois {
		return nil
	}
	if responsavel == nil || *responsavel == uuid.Nil {
		return ErrResponsavelObrigatorio
	}
	return nil
}
