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

// ConferirPredecessora exige a anterior concluída para sair de "a fazer"
// e de "bloqueada". Essas duas posições continuam livres.
func ConferirPredecessora(estado string, predecessora *Tarefa) error {
	if estado == "" || estado == EstadoTarefaAFazer || estado == EstadoTarefaBloqueada || predecessora == nil {
		return nil
	}
	if predecessora.Estado != EstadoTarefaConcluida {
		return ErrPredecessoraAberta
	}
	return nil
}
