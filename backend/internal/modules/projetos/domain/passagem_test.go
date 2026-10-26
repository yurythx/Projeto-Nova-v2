package domain

import "testing"

func TestPassagemExtraDoProjetoEDaTarefa(t *testing.T) {
	const coluna = "c01234567"
	if ProximosEstados(EstadoCancelado, coluna) != nil {
		t.Fatal("cancelado não origina passagem")
	}
	if ValidarTransicao(EstadoCancelado, coluna, coluna) || ValidarTransicao(EstadoCancelado, EstadoEmAndamento, coluna) {
		t.Fatal("cancelado não sai")
	}
	if !ValidarTransicao(EstadoPlanejado, coluna, coluna) {
		t.Fatal("a passagem extra entra no projeto")
	}
	if ValidarTransicao(EstadoPlanejado, coluna) {
		t.Fatal("sem extra o projeto não pula para a coluna nova")
	}
	if !ValidarTransicao(EstadoConcluido, EstadoEmAndamento) {
		t.Fatal("concluído ainda volta para em andamento")
	}

	if !EstadoDeTarefaAceito("", "", nil) || !EstadoDeTarefaAceito(EstadoTarefaAFazer, EstadoTarefaConcluida, nil) {
		t.Fatal("os estados de fábrica da tarefa continuam livres")
	}
	if EstadoDeTarefaAceito("", coluna, nil) || EstadoDeTarefaAceito(EstadoTarefaAFazer, coluna, map[string][]string{"concluida": {coluna}}) {
		t.Fatal("a coluna nova da tarefa exige a passagem a partir do estado atual")
	}
	if !EstadoDeTarefaAceito("", coluna, map[string][]string{"a_fazer": {coluna}}) {
		t.Fatal("criar aceita a coluna nova quando alguma passagem aponta para ela")
	}
	if !EstadoDeTarefaAceito(EstadoTarefaAFazer, coluna, map[string][]string{EstadoTarefaAFazer: {coluna}}) {
		t.Fatal("atualizar aceita a passagem a partir do estado atual")
	}
}
