package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCalendarioNosPrazos(t *testing.T) {
	natal := Calendario{"2026-12-25": true}
	vespera := time.Date(2026, 12, 24, 10, 0, 0, 0, Cuiaba) // quinta
	if d := SomarDiasUteis(vespera, 1, natal); d.Format(time.DateOnly) != "2026-12-28" {
		t.Fatal("o Natal não conta", d)
	}
	if d := SomarDiasUteis(vespera, 1); d.Format(time.DateOnly) != "2026-12-25" {
		t.Fatal("sem calendário", d)
	}
	anoNovo := Calendario{"2026-12-28": true}
	if d := SomarDiasUteis(vespera, 1, natal, anoNovo); d.Format(time.DateOnly) != "2026-12-29" {
		t.Fatal("dois calendários", d)
	}
	if !natal.Feriado(time.Date(2026, 12, 25, 23, 0, 0, 0, Cuiaba)) || natal.Feriado(vespera) {
		t.Fatal("feriado")
	}
	// Com o feriado, o prazo de dois dias úteis entra em atenção mais cedo.
	prazo := time.Date(2026, 12, 28, 23, 0, 0, 0, Cuiaba)
	if s := SituacaoPrazo(StatusAndamento, vespera, &prazo, vespera, natal); s != PrazoNoPrazo {
		t.Fatal(s)
	}

	a := Agenda{Dias: []int{4, 5}, Inicio: "08:00", Fim: "09:00", DuracaoMin: 30, VagasDia: 2}
	quarta := time.Date(2026, 12, 23, 10, 0, 0, 0, Cuiaba)
	dia, hora, err := ProcurarVaga(a, quarta, func(string) ([]string, error) { return nil, nil }, Calendario{"2026-12-24": true, "2026-12-25": true})
	if err != nil || dia != "2026-12-31" || hora != "08:00" {
		t.Fatal(dia, hora, err)
	}

	if CalendarioDe([]Feriado{{Data: "2026-12-25", Nome: "Natal"}})["2026-12-25"] != true {
		t.Fatal("calendário")
	}
}

func TestFeriado(t *testing.T) {
	f, err := NovoFeriado(" 2026-12-25 ", " Natal ")
	if err != nil || f.Data != "2026-12-25" || f.Nome != "Natal" {
		t.Fatal(f, err)
	}
	for _, c := range [][2]string{{"25/12/2026", "Natal"}, {"2026-12-25", "N"}, {"2026-12-25", strings.Repeat("a", 121)}} {
		if _, err := NovoFeriado(c[0], c[1]); !errors.Is(err, ErrInvalido) {
			t.Fatal(c, err)
		}
	}
}

func TestDetalhesEAprovadorDoServico(t *testing.T) {
	agora := time.Now()
	base := Servico{UnidadeID: uuid.New(), FormularioID: uuid.New(), Nome: "Alvará", AprovadorID: &uuid.Nil,
		Detalhes: DetalhesServico{QuemPode: "  Empresas  ", PassoAPasso: "1. Abra", Contato: "66 3411", TempoEstimado: "10 dias", DescricaoTaxa: "UPF"}}
	s, err := NovoServico(base, agora)
	if err != nil || s.AprovadorID != nil || s.Detalhes.QuemPode != "Empresas" {
		t.Fatal(s.Detalhes, err)
	}
	for _, d := range []DetalhesServico{
		{QuemPode: strings.Repeat("a", 4001)},
		{PassoAPasso: strings.Repeat("a", 4001)},
		{Contato: strings.Repeat("a", 501)},
		{DescricaoTaxa: strings.Repeat("a", 501)},
		{TempoEstimado: strings.Repeat("a", 121)},
	} {
		ruim := base
		ruim.Detalhes = d
		if _, err := NovoServico(ruim, agora); !errors.Is(err, ErrInvalido) {
			t.Fatal(d, err)
		}
	}
	ana := uuid.New()
	base.AprovadorID = &ana
	comAprovador, err := NovoServico(base, agora)
	if err != nil || *comAprovador.AprovadorID != ana {
		t.Fatal(err)
	}
	form := Formulario{ID: comAprovador.FormularioID, Ativo: true}
	if p, err := Abrir(comAprovador, form, "PREF202600001", "Maria Souza", "12345678900", "", "", "", nil, nil, agora); err != nil || *p.AprovadorID != ana {
		t.Fatal(err)
	}
}

func TestProcessoSEI(t *testing.T) {
	agora := time.Now()
	p := Pedido{Status: StatusAndamento}
	vinculado, err := p.VincularSEI(" 12345.000001/2026-11 ", agora)
	if err != nil || vinculado.ProcessoSEI != "12345.000001/2026-11" {
		t.Fatal(err)
	}
	if limpo, err := vinculado.VincularSEI("", agora); err != nil || limpo.ProcessoSEI != "" {
		t.Fatal(err)
	}
	for _, ruim := range []string{"abc", "12", "12345.000001/2026-1x", "-12345"} {
		if _, err := p.VincularSEI(ruim, agora); !errors.Is(err, ErrInvalido) {
			t.Fatal(ruim, err)
		}
	}
	if _, err := (Pedido{Status: StatusConcluido}).VincularSEI("12345", agora); !errors.Is(err, ErrEstadoFinal) {
		t.Fatal(err)
	}
	c := Publico(vinculado, "", nil, nil)
	if c.ProcessoSEI != "12345.000001/2026-11" {
		t.Fatal(c.ProcessoSEI)
	}
}

func TestPrazoDaComplementacao(t *testing.T) {
	sexta := time.Date(2026, 11, 20, 10, 0, 0, 0, Cuiaba)
	p, err := (Pedido{Status: StatusAndamento}).Complementar("envie o RG", sexta)
	if err != nil {
		t.Fatal(err)
	}
	comPrazo, err := p.PrazoDaComplementacao(2, sexta, Calendario{"2026-11-23": true})
	if err != nil || comPrazo.ComplementoPrazo.Format("2006-01-02 15:04") != "2026-11-25 23:59" {
		t.Fatal(comPrazo.ComplementoPrazo, err)
	}
	if semPrazo, err := comPrazo.PrazoDaComplementacao(0, sexta); err != nil || semPrazo.ComplementoPrazo != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		p    Pedido
		dias int
	}{{Pedido{Status: StatusAndamento}, 2}, {p, -1}, {p, PrazoMaximoComplementacao + 1}} {
		if _, err := c.p.PrazoDaComplementacao(c.dias, sexta); !errors.Is(err, ErrInvalido) {
			t.Fatal(c.dias, err)
		}
	}
	consulta := Publico(comPrazo, "", nil, nil)
	if consulta.ComplementoPrazo == nil {
		t.Fatal("o cidadão vê o prazo")
	}

	depois := comPrazo.ComplementoPrazo.Add(time.Minute)
	cancelado, err := comPrazo.ExpirarComplementacao(depois)
	if err != nil || cancelado.Status != StatusCancelado {
		t.Fatal(err)
	}
	if _, err := comPrazo.ExpirarComplementacao(sexta); !errors.Is(err, ErrInvalido) {
		t.Fatal("ainda no prazo", err)
	}
	if _, err := p.ExpirarComplementacao(depois); !errors.Is(err, ErrInvalido) {
		t.Fatal("sem prazo", err)
	}

	comPrazo.ComplementoAvisado = true
	respondido, err := comPrazo.ResponderComplemento(sexta)
	if err != nil || respondido.ComplementoPrazo != nil || respondido.ComplementoAvisado {
		t.Fatal(err)
	}
	if _, err := respondido.ExpirarComplementacao(depois); !errors.Is(err, ErrInvalido) {
		t.Fatal("respondido", err)
	}
}
