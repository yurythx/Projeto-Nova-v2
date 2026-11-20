package application

import (
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
)

func TestRotulosDosEstados(t *testing.T) {
	projetos := map[string]string{
		domain.EstadoPlanejado:   "Planejado",
		domain.EstadoEmAndamento: "Em andamento",
		domain.EstadoSuspenso:    "Suspenso",
		domain.EstadoConcluido:   "Concluído",
		domain.EstadoCancelado:   "Cancelado",
		"coluna_extra":           "coluna_extra",
	}
	for estado, rotulo := range projetos {
		if got := rotuloEstadoProjeto(estado); got != rotulo {
			t.Errorf("%s: %s", estado, got)
		}
	}
	tarefas := map[string]string{
		domain.EstadoTarefaAFazer:      "A fazer",
		domain.EstadoTarefaEmAndamento: "Em andamento",
		domain.EstadoTarefaConcluida:   "Concluída",
		domain.EstadoTarefaBloqueada:   "Bloqueada",
		"coluna_extra":                 "coluna_extra",
	}
	for estado, rotulo := range tarefas {
		if got := rotuloEstadoTarefa(estado); got != rotulo {
			t.Errorf("%s: %s", estado, got)
		}
	}
	if !mudouTrilha(domain.Tarefa{}, domain.Tarefa{Estado: domain.EstadoTarefaEmAndamento}) {
		t.Fatal("sair de a fazer muda a trilha")
	}
	if mudouTrilha(domain.Tarefa{}, domain.Tarefa{}) {
		t.Fatal("as duas em a fazer não mudam")
	}
}
