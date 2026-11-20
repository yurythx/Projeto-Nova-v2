package application

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
)

func TestResumoEPrazos(t *testing.T) {
	agora := time.Now().UTC()
	zero := time.Time{}
	outra := agora.Add(time.Hour)
	id := uuid.New()
	outro := uuid.New()

	if !mesmoPrazo(nil, nil) || !mesmoPrazo(&zero, nil) || mesmoPrazo(&agora, nil) || mesmoPrazo(&agora, &zero) {
		t.Fatal("prazo vazio")
	}
	if !mesmoPrazo(&agora, &agora) || mesmoPrazo(&agora, &outra) {
		t.Fatal("prazo preenchido")
	}

	base := domain.Tarefa{Titulo: "Mapear", Estado: domain.EstadoTarefaAFazer, Relato: "nada"}
	if resumoTarefa(base, domain.Tarefa{Titulo: "Mapear", Estado: domain.EstadoTarefaEmAndamento, ResponsavelID: &id}) != "Mapear ficou com ninguém foi para Em andamento" {
		t.Fatal("estado")
	}
	if resumoTarefa(base, domain.Tarefa{Titulo: "Publicar", Estado: domain.EstadoTarefaAFazer, Relato: "nada"}) != "O título passou a ser Publicar" {
		t.Fatal("título")
	}
	if resumoTarefa(base, domain.Tarefa{Titulo: "Mapear", Estado: domain.EstadoTarefaAFazer, Relato: "novo"}) != "Mapear tem um relato novo" {
		t.Fatal("relato")
	}
	if resumoTarefa(base, domain.Tarefa{Titulo: "Mapear", Estado: domain.EstadoTarefaAFazer, Relato: "nada", Ordem: 2}) != "Mapear foi atualizada" {
		t.Fatal("atualizada")
	}

	if len(diferenca([]uuid.UUID{id, id}, nil)) != 1 {
		t.Fatal("diferença")
	}
	if diferenca([]uuid.UUID{id}, []uuid.UUID{id}) != nil {
		t.Fatal("já estava")
	}

	nilID := uuid.Nil
	soPessoa := destinosDaTarefa(domain.PrazoTarefa{Pessoas: []uuid.UUID{nilID, id, id}, GerenteID: &id})
	if len(soPessoa) != 1 || soPessoa[0] != id {
		t.Fatal("destinos da tarefa")
	}
	comGerente := destinosDaTarefa(domain.PrazoTarefa{Pessoas: []uuid.UUID{id}, GerenteID: &outro})
	if len(comGerente) != 2 {
		t.Fatal("gerente entra")
	}
	if len(destinosDoPrazo(nil, &outro)) != 1 || len(destinosDoPrazo(&nilID, &outro)) != 1 {
		t.Fatal("só o gerente")
	}
	if len(destinosDoPrazo(&id, &id)) != 1 {
		t.Fatal("gerente repetido")
	}
}
