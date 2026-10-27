package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidarModelo(t *testing.T) {
	eq := uuid.New()
	if out, err := ValidarModelo(TipoIncidente, nil); err != nil || len(out) != 0 {
		t.Fatalf("vazio vale para qualquer tipo: %v %v", out, err)
	}
	um := []ModeloTarefa{{Titulo: " Criar conta ", EquipeID: eq}}
	if _, err := ValidarModelo(TipoIncidente, um); !errors.Is(err, ErrTarefasSoRequisicao) {
		t.Fatalf("incidente: %v", err)
	}
	demais := make([]ModeloTarefa, MaxTarefas+1)
	for nome, m := range map[string][]ModeloTarefa{
		"demais":       demais,
		"sem título":   {{Titulo: " ", EquipeID: eq}},
		"título longo": {{Titulo: strings.Repeat("a", maxTituloTarefa+1), EquipeID: eq}},
		"sem equipe":   {{Titulo: "x"}},
	} {
		if _, err := ValidarModelo(TipoRequisicao, m); !errors.Is(err, ErrModeloInvalido) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	out, err := ValidarModelo(TipoRequisicao, append(um, ModeloTarefa{Ordem: 9, Titulo: "Notebook", EquipeID: eq}))
	if err != nil || len(out) != 2 || out[0].Titulo != "Criar conta" || out[1].Ordem != 1 {
		t.Fatalf("numera e apara: %+v %v", out, err)
	}
	agora := time.Now()
	tarefas := TarefasDoModelo(uuid.New(), out, agora)
	if len(tarefas) != 2 || tarefas[1].Estado != TarefaPendente || *tarefas[1].EquipeID != eq || Pendentes(tarefas) != 2 {
		t.Fatalf("tarefas do modelo: %+v", tarefas)
	}
}

func TestFazerTarefa(t *testing.T) {
	quem, agora := uuid.New(), time.Now()
	p := Tarefa{Estado: TarefaPendente}
	if _, err := p.Fazer(quem, false, true, "", agora); !errors.Is(err, ErrNaoEhDaEquipe) {
		t.Fatalf("de fora da equipe: %v", err)
	}
	if _, err := p.Fazer(quem, true, false, " ", agora); !errors.Is(err, ErrDispensaSemMotivo) {
		t.Fatalf("dispensa sem motivo: %v", err)
	}
	if _, err := p.Fazer(quem, true, true, strings.Repeat("a", MaxObservacao+1), agora); !errors.Is(err, ErrDispensaSemMotivo) {
		t.Fatalf("observação longa: %v", err)
	}
	feita, err := p.Fazer(quem, true, true, " ok ", agora)
	if err != nil || feita.Estado != TarefaConcluida || feita.Observacao != "ok" || *feita.FeitaPor != quem {
		t.Fatalf("concluir: %+v %v", feita, err)
	}
	if _, err := feita.Fazer(quem, true, true, "", agora); !errors.Is(err, ErrTarefaFeita) {
		t.Fatalf("já feita: %v", err)
	}
	dispensada, err := p.Fazer(quem, false, false, "Não precisa", agora)
	if err != nil || dispensada.Estado != TarefaDispensada {
		t.Fatalf("dispensar, mesmo de fora da equipe: %+v %v", dispensada, err)
	}
	if Pendentes([]Tarefa{p, feita, dispensada}) != 1 {
		t.Fatal("pendentes")
	}
}
