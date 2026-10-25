package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDiaNoFuso(t *testing.T) {
	cedo := time.Date(2026, 10, 24, 2, 30, 0, 0, time.UTC)
	if DiaNoFuso(cedo, "America/Sao_Paulo").Format("2006-01-02") != "2026-10-23" {
		t.Fatal("antes da meia-noite de Brasília ainda é o dia anterior")
	}
	tarde := time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC)
	if DiaNoFuso(tarde, "America/Sao_Paulo").Format("2006-01-02") != "2026-10-24" {
		t.Fatal("meia-noite em Brasília abre o dia")
	}
	if DiaNoFuso(cedo, "Marte/Olimpo").Format("2006-01-02") != "2026-10-23" {
		t.Fatal("fuso desconhecido usa Brasília")
	}
	if DiaNoFuso(cedo, "Local").Format("2006-01-02") != "2026-10-23" {
		t.Fatal("Local não é um fuso da política")
	}
}

func TestAgregarSerie(t *testing.T) {
	if len(AgregarSerie(nil)) != 0 {
		t.Fatal("vazio")
	}
	dia := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	mtbf := 4.0
	sozinho := AgregarSerie([]Retrato{{Dia: dia, MTBFHoras: &mtbf, MTTRN: 0}})
	if sozinho[0].MTBFHoras == nil || *sozinho[0].MTBFHoras != 4 {
		t.Fatal("um ponto conserva o intervalo")
	}
	u1, u2 := uuid.New(), uuid.New()
	juntos := AgregarSerie([]Retrato{
		{Dia: dia, UnidadeID: &u1, SomaMTTRMin: 60, MTTRN: 1, SLACumprido: 1, SLAN: 1, FCRN: 1, Concluidas: 1, CSATSoma: 5, CSATN: 1},
		{Dia: dia, UnidadeID: &u2, SomaMTTRMin: 180, MTTRN: 1, SLACumprido: 0, SLAN: 1, FCRN: 0, Concluidas: 1, CSATSoma: 3, CSATN: 1, Truncado: true},
	})
	if len(juntos) != 1 || juntos[0].MTBFHoras != nil {
		t.Fatal("duas unidades não têm um intervalo só")
	}
	if juntos[0].MTTRHoras == nil || *juntos[0].MTTRHoras != 2 {
		t.Fatalf("mttr %#v", juntos[0].MTTRHoras)
	}
	if juntos[0].SLAPercentual == nil || *juntos[0].SLAPercentual != 50 {
		t.Fatalf("sla %#v", juntos[0].SLAPercentual)
	}
	if juntos[0].FCRPercentual == nil || *juntos[0].FCRPercentual != 50 {
		t.Fatalf("fcr %#v", juntos[0].FCRPercentual)
	}
	if juntos[0].CSATMedia == nil || *juntos[0].CSATMedia != 4 {
		t.Fatalf("csat %#v", juntos[0].CSATMedia)
	}
	if !juntos[0].Truncado {
		t.Fatal("truncado")
	}
	sem := AgregarSerie([]Retrato{{Dia: dia, UnidadeID: &u1}, {Dia: dia, UnidadeID: &u2}})
	if sem[0].MTTRHoras != nil || sem[0].SLAPercentual != nil || sem[0].CSATMedia != nil {
		t.Fatal("sem amostra continua vazio")
	}
}
