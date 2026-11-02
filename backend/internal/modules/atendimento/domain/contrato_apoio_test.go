package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTempoComTerceiro(t *testing.T) {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	h := func(n int) time.Time { return base.Add(time.Duration(n) * time.Hour) }
	de := func(e string) *string { return &e }
	// Fora de ordem de propósito: a soma ordena pelas datas.
	ps := []Passagem{
		{Para: EstadoEmAtendimento, De: de(EstadoAguardandoTerceiro), Em: h(5)},
		{Para: EstadoAberto, Em: h(0)},
		{Para: EstadoAguardandoTerceiro, De: de(EstadoEmAtendimento), Em: h(2)},
		{Para: EstadoAguardandoTerceiro, De: de(EstadoEmAtendimento), Em: h(10)},
		{Para: EstadoResolvido, De: de(EstadoAguardandoTerceiro), Em: h(14)},
	}
	if tempo, aberto := TempoComTerceiro(Solicitacao{Estado: EstadoResolvido}, ps, h(20)); tempo != 7*time.Hour || aberto {
		t.Fatalf("duas idas ao fornecedor: %v %v", tempo, aberto)
	}
	aindaLa := ps[:4]
	if tempo, aberto := TempoComTerceiro(Solicitacao{Estado: EstadoAguardandoTerceiro}, aindaLa, h(12)); tempo != 5*time.Hour || !aberto {
		t.Fatalf("ainda com o fornecedor conta até agora: %v %v", tempo, aberto)
	}
	if tempo, _ := TempoComTerceiro(Solicitacao{Estado: EstadoResolvido}, ps[:2], h(12)); tempo != 0 {
		t.Fatalf("sem ida ao fornecedor: %v", tempo)
	}
}

func TestResumirContratos(t *testing.T) {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	h := func(n int) time.Time { return base.Add(time.Duration(n) * time.Hour) }
	link, energia := uuid.New(), uuid.New()
	prazo := 4
	ida := func(desde, ate int) []Passagem {
		ps := []Passagem{{Para: EstadoAguardandoTerceiro, Em: h(desde)}}
		if ate >= 0 {
			ps = append(ps, Passagem{Para: EstadoEmAtendimento, Em: h(ate)})
		}
		return ps
	}
	a, b, c, d, e, f := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	com := func(id uuid.UUID, contrato uuid.UUID, rotulo string, p *int, estado string) Solicitacao {
		return Solicitacao{ID: id, Estado: estado, ContratoApoioID: &contrato, ContratoApoio: rotulo, ContratoPrazoHoras: p}
	}
	lista := []Solicitacao{
		com(a, link, "Link", &prazo, EstadoResolvido),
		com(b, link, "Link", &prazo, EstadoResolvido),
		com(c, link, "Link", &prazo, EstadoAguardandoTerceiro),
		com(d, link, "Link", &prazo, EstadoAguardandoTerceiro),
		com(e, link, "Link", &prazo, EstadoEmAtendimento),
		com(f, energia, "Energia", nil, EstadoResolvido),
		{ID: uuid.New(), Estado: EstadoResolvido},
	}
	passagens := map[uuid.UUID][]Passagem{
		a: ida(0, 3),  // no prazo
		b: ida(0, 6),  // fora
		c: ida(0, -1), // com o fornecedor há 10 h: atrasado
		d: ida(9, -1), // com o fornecedor há 1 h
		f: ida(0, 2),
	}
	linhas := ResumirContratos(lista, passagens, h(10), nil)
	if len(linhas) != 2 || linhas[0].Rotulo != "Energia" {
		t.Fatalf("por rótulo: %+v", linhas)
	}
	l := linhas[1]
	if l.Pedidos != 5 || l.Medidos != 2 || l.NoPrazo != 1 || *l.NoPrazoPercentual != 50 || *l.MedianaHoras != 4.5 || l.ComFornecedor != 2 || l.Atrasados != 1 {
		t.Fatalf("link: %+v", l)
	}
	if s := linhas[0]; s.NoPrazoPercentual != nil || *s.MedianaHoras != 2 || s.Medidos != 1 {
		t.Fatalf("sem prazo não há percentual: %+v", s)
	}
}

// Sexta 16h até segunda 10h são 4 h de expediente (sexta 16–18 e segunda
// 8–10) e dezenas de horas corridas. O prazo de 4 h só cabe no relógio útil.
func TestResumirContratosNoExpediente(t *testing.T) {
	cal, err := NovoCalendario(politicaPadrao(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	id, contrato := uuid.New(), uuid.New()
	prazo := 4
	sol := Solicitacao{
		ID: id, Estado: EstadoAguardandoTerceiro,
		ContratoApoioID: &contrato, ContratoApoio: "Link", ContratoPrazoHoras: &prazo,
	}
	passagens := map[uuid.UUID][]Passagem{
		id: {{Para: EstadoAguardandoTerceiro, Em: em("2026-10-02", "16:00")}},
	}
	agora := em("2026-10-05", "10:00")
	uteis := ResumirContratos([]Solicitacao{sol}, passagens, agora, map[uuid.UUID]func(time.Time, time.Time) time.Duration{
		id: cal.Util,
	})
	if len(uteis) != 1 || uteis[0].ComFornecedor != 1 || uteis[0].Atrasados != 0 {
		t.Fatalf("4 h úteis cabem no prazo: %+v", uteis)
	}
	corridas := ResumirContratos([]Solicitacao{sol}, passagens, agora, nil)
	if corridas[0].Atrasados != 1 {
		t.Fatalf("hora corrida estoura as 4 h: %+v", corridas[0])
	}
}
