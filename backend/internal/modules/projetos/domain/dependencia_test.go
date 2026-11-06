package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCadeiaEConclusao(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	tarefas := []Tarefa{
		{ID: a, PredecessoraID: &b},
		{ID: b, PredecessoraID: &c},
		{ID: c},
	}
	if err := ValidarCadeia(c, a, tarefas); !errors.Is(err, ErrPredecessoraInvalida) {
		t.Fatalf("ciclo: %v", err)
	}
	if err := ValidarCadeia(a, a, tarefas); !errors.Is(err, ErrPredecessoraInvalida) {
		t.Fatalf("própria: %v", err)
	}
	if err := ValidarCadeia(a, uuid.New(), tarefas); !errors.Is(err, ErrPredecessoraInvalida) {
		t.Fatalf("fora do projeto: %v", err)
	}
	if err := ValidarCadeia(uuid.New(), c, tarefas); err != nil {
		t.Fatalf("cadeia livre: %v", err)
	}
	if err := ValidarCadeia(a, uuid.Nil, tarefas); err != nil {
		t.Fatalf("sem anterior: %v", err)
	}

	aberta := &Tarefa{Estado: EstadoTarefaAFazer}
	if err := ConferirPredecessora(EstadoTarefaEmAndamento, aberta); err != nil {
		t.Fatalf("andamento em paralelo: %v", err)
	}
	if err := ConferirPredecessora(EstadoTarefaConcluida, aberta); !errors.Is(err, ErrPredecessoraAberta) {
		t.Fatalf("concluir com anterior aberta: %v", err)
	}
	if err := ConferirPredecessora(EstadoTarefaBloqueada, aberta); err != nil {
		t.Fatalf("bloquear pode: %v", err)
	}
	pronta := &Tarefa{Estado: EstadoTarefaConcluida}
	if err := ConferirPredecessora(EstadoTarefaConcluida, pronta); err != nil {
		t.Fatalf("anterior pronta: %v", err)
	}

	if err := RecusarAlteracao(EstadoConcluido); !errors.Is(err, ErrProjetoConcluido) {
		t.Fatalf("concluído: %v", err)
	}
	if err := RecusarAlteracao(EstadoCancelado); !errors.Is(err, ErrProjetoEncerrado) {
		t.Fatalf("cancelado: %v", err)
	}
	if err := RecusarAlteracao(EstadoEmAndamento); err != nil {
		t.Fatalf("em andamento: %v", err)
	}
	if TarefasEmAberto([]Tarefa{{Estado: EstadoTarefaConcluida}, {Estado: EstadoTarefaAFazer}, {Estado: "coluna_extra"}}) != 2 {
		t.Fatal("contagem de tarefas em aberto")
	}
}
