package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func f64(v float64) *float64 { return &v }

func TestMetasXLAValidar(t *testing.T) {
	if err := (MetasXLA{}).Validar(); err != nil {
		t.Fatalf("sem metas: %v", err)
	}
	horas := func(v int) *int { return &v }
	for nome, m := range map[string]MetasXLA{
		"nota baixa":          {CSATMin: f64(0.5)},
		"facilidade alta":     {FacilidadeMin: f64(5.5)},
		"reabertura demais":   {ReaberturaMax: f64(101)},
		"idas negativas":      {IdasMax: f64(-1)},
		"espera zero":         {EsperaMaxHoras: horas(0)},
		"espera de dois anos": {EsperaMaxHoras: horas(9000)},
	} {
		if err := m.Validar(); !errors.Is(err, ErrMetasInvalidas) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if err := (MetasXLA{CSATMin: f64(4), FacilidadeMin: f64(4), ReaberturaMax: f64(5), IdasMax: f64(1), EsperaMaxHoras: horas(48)}).Validar(); err != nil {
		t.Fatal(err)
	}
}

func TestComFacilidade(t *testing.T) {
	cinco, zero := 5, 0
	if a, err := (Avaliacao{}).ComFacilidade(&cinco); err != nil || *a.Facilidade != 5 {
		t.Fatalf("válida: %+v %v", a, err)
	}
	if a, err := (Avaliacao{}).ComFacilidade(nil); err != nil || a.Facilidade != nil {
		t.Fatalf("sem resposta: %+v %v", a, err)
	}
	if _, err := (Avaliacao{}).ComFacilidade(&zero); !errors.Is(err, ErrFacilidadeInvalida) {
		t.Fatalf("fora da faixa: %v", err)
	}
}

func TestMedirExperiencia(t *testing.T) {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	em := func(h int) *time.Time { v := base.Add(time.Duration(h) * time.Hour); return &v }
	de := func(e string) *string { return &e }
	direta, reaberta, devolvida, semHistorico, aberta := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	lista := []Solicitacao{
		{ID: direta, Estado: EstadoResolvido, CreatedAt: base, ResolvidaEm: em(2)},
		{ID: reaberta, Estado: EstadoEncerrado, CreatedAt: base, ResolvidaEm: em(10)},
		{ID: devolvida, Estado: EstadoAtendido, CreatedAt: base, ResolvidaEm: em(30)},
		{ID: semHistorico, Estado: EstadoResolvido, CreatedAt: base, ResolvidaEm: em(4)},
		{ID: aberta, Estado: EstadoEmAtendimento, CreatedAt: base},
	}
	passagens := map[uuid.UUID][]Passagem{
		direta:   {{Para: EstadoAberto}, {De: de(EstadoAberto), Para: EstadoEmAtendimento}, {De: de(EstadoEmAtendimento), Para: EstadoResolvido}},
		reaberta: {{Para: EstadoEmAtendimento, De: de(EstadoAberto)}, {Para: EstadoResolvido}, {De: de(EstadoResolvido), Para: EstadoEmAtendimento}, {Para: EstadoResolvido}},
		devolvida: {{Para: EstadoEmAtendimento, De: de(EstadoAberto)}, {Para: EstadoAguardandoServidor}, {Para: EstadoEmAtendimento, De: de(EstadoAguardandoServidor)},
			{Para: EstadoAguardandoServidor}, {Para: EstadoEmAtendimento, De: de(EstadoAguardandoServidor)}, {Para: EstadoAtendido}},
	}
	resumo := ResumoAvaliacoes{Total: 4, Soma: 17, TotalFacilidade: 2, SomaFacilidade: 7}
	horas := 6
	x := MedirExperiencia(lista, passagens, resumo, MetasXLA{CSATMin: f64(4), FacilidadeMin: f64(4), ReaberturaMax: f64(20), IdasMax: f64(1), EsperaMaxHoras: &horas})

	confere := func(nome string, m MedidaXLA, valor float64, amostra int, atingida bool) {
		t.Helper()
		if m.Valor == nil || *m.Valor != valor || m.Amostra != amostra || m.Atingida == nil || *m.Atingida != atingida {
			t.Errorf("%s: %+v (valor %v)", nome, m, m.Valor)
		}
	}
	confere("nota", x.CSAT, 4.3, 4, true)
	confere("facilidade", x.Facilidade, 3.5, 2, false)
	// Das três com histórico, uma reaberta (33,3%) e duas idas ao todo.
	confere("reabertura", x.Reabertura, 33.3, 3, false)
	confere("idas e voltas", x.IdasEVoltas, 0.7, 3, true)
	// Esperas 2, 4, 10 e 30 horas: mediana 7.
	confere("espera", x.EsperaHoras, 7, 4, false)
	if !x.CSAT.MaiorMelhor || x.EsperaHoras.MaiorMelhor {
		t.Fatal("sentido das metas")
	}

	vazio := MedirExperiencia(lista[4:], nil, ResumoAvaliacoes{}, MetasXLA{CSATMin: f64(4)})
	if vazio.CSAT.Valor != nil || vazio.CSAT.Atingida != nil || vazio.Reabertura.Valor != nil || vazio.EsperaHoras.Valor != nil {
		t.Fatalf("sem amostra não se afirma nada: %+v", vazio)
	}
	impar := MedirExperiencia(lista[:3], passagens, ResumoAvaliacoes{}, MetasXLA{})
	if *impar.EsperaHoras.Valor != 10 || impar.EsperaHoras.Atingida != nil {
		t.Fatalf("mediana ímpar, sem meta: %+v", impar.EsperaHoras)
	}
}
