package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// expediente de segunda a sexta, 8h–12h e 14h–18h.
func turnosUteis() []Turno {
	out := []Turno{}
	for dia := 1; dia <= 5; dia++ {
		out = append(out, Turno{Dia: dia, Inicio: "14:00", Fim: "18:00"}, Turno{Dia: dia, Inicio: "08:00", Fim: "12:00"})
	}
	return out
}

func matrizPadrao() []Celula {
	out := []Celula{}
	for i := 1; i <= 3; i++ {
		for u := 1; u <= 3; u++ {
			p := PrioridadeMedia
			switch {
			case i == 3 && u == 3:
				p = PrioridadeCritica
			case i+u >= 5:
				p = PrioridadeAlta
			case i+u <= 2:
				p = PrioridadeBaixa
			}
			out = append(out, Celula{Impacto: i, Urgencia: u, Prioridade: p})
		}
	}
	return out
}

func metasPadrao() []Meta {
	return []Meta{
		{Prioridade: PrioridadeBaixa, RespostaMin: 480, SolucaoMin: 2400},
		{Prioridade: PrioridadeCritica, RespostaMin: 30, SolucaoMin: 240},
		{Prioridade: PrioridadeAlta, RespostaMin: 60, SolucaoMin: 480},
		{Prioridade: PrioridadeMedia, RespostaMin: 120, SolucaoMin: 960},
	}
}

func politicaPadrao(t *testing.T) Politica {
	t.Helper()
	p, err := NovaPolitica(nil, " America/Sao_Paulo ", turnosUteis(), matrizPadrao(), metasPadrao())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var sp, _ = time.LoadLocation("America/Sao_Paulo")

func em(data, hora string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", data+" "+hora, sp)
	if err != nil {
		panic(err)
	}
	return t
}

func TestNovaPoliticaNormaliza(t *testing.T) {
	p := politicaPadrao(t)
	if p.Fuso != "America/Sao_Paulo" || p.ID == uuid.Nil {
		t.Fatalf("fuso: %+v", p)
	}
	if p.Turnos[0] != (Turno{Dia: 1, Inicio: "08:00", Fim: "12:00"}) || p.Turnos[1].Inicio != "14:00" {
		t.Fatalf("turnos em ordem de dia e hora: %+v", p.Turnos[:2])
	}
	if p.Matriz[0] != (Celula{Impacto: 3, Urgencia: 3, Prioridade: PrioridadeCritica}) {
		t.Fatalf("matriz do mais grave ao mais leve: %+v", p.Matriz[0])
	}
	if p.Metas[0].Prioridade != PrioridadeCritica || p.Metas[3].Prioridade != PrioridadeBaixa {
		t.Fatalf("metas por prioridade: %+v", p.Metas)
	}
	if prio, meta := p.Classificar(3, 3); prio != PrioridadeCritica || meta.RespostaMin != 30 {
		t.Fatalf("classificação: %s %+v", prio, meta)
	}
	if prio, _ := p.Classificar(9, 9); prio != "" {
		t.Fatal("combinação fora da matriz")
	}
	semMeta := p
	semMeta.Metas = nil
	if prio, _ := semMeta.Classificar(3, 3); prio != "" {
		t.Fatal("prioridade sem meta não classifica")
	}
	// Meta de prioridade que a matriz não usa é aceita.
	soMedia := []Celula{}
	for _, c := range matrizPadrao() {
		c.Prioridade = PrioridadeMedia
		soMedia = append(soMedia, c)
	}
	if q, err := NovaPolitica(nil, "UTC", turnosUteis(), soMedia, metasPadrao()); err != nil || len(q.Metas) != 4 {
		t.Fatalf("meta sobrando: %v", err)
	}
	if q, err := NovaPolitica(nil, "UTC", turnosUteis(), soMedia, metasPadrao()[3:]); err != nil || len(q.Metas) != 1 {
		t.Fatalf("só a meta da prioridade usada: %v", err)
	}
	if _, err := NovaPolitica(nil, "UTC", []Turno{{Dia: 0, Inicio: "00:00", Fim: "24:00"}}, matrizPadrao(), metasPadrao()); err != nil {
		t.Fatalf("dia inteiro: %v", err)
	}
}

func TestNovaPoliticaRecusa(t *testing.T) {
	cheia := append(turnosUteis(), Turno{Dia: 1, Inicio: "12:00", Fim: "13:00"}, Turno{Dia: 1, Inicio: "13:00", Fim: "14:00"}, Turno{Dia: 1, Inicio: "18:00", Fim: "19:00"})
	sobreposta := append(turnosUteis(), Turno{Dia: 2, Inicio: "11:00", Fim: "13:00"})
	repetida := append(matrizPadrao()[:8], matrizPadrao()[0])
	foraDaFaixa := append(matrizPadrao()[:8], Celula{Impacto: 4, Urgencia: 1, Prioridade: PrioridadeBaixa})
	semAlta := []Meta{metasPadrao()[0], metasPadrao()[1], metasPadrao()[3]}
	casos := []struct {
		nome   string
		fuso   string
		turnos []Turno
		matriz []Celula
		metas  []Meta
		motivo string
	}{
		{"sem fuso", " ", turnosUteis(), matrizPadrao(), metasPadrao(), "informe o fuso"},
		{"fuso desconhecido", "Marte/Olimpo", turnosUteis(), matrizPadrao(), metasPadrao(), "desconhecido"},
		{"fuso local", "Local", turnosUteis(), matrizPadrao(), metasPadrao(), "desconhecido"},
		{"sem turnos", "UTC", nil, matrizPadrao(), metasPadrao(), "ao menos um turno"},
		{"dia inválido", "UTC", []Turno{{Dia: 7, Inicio: "08:00", Fim: "12:00"}}, matrizPadrao(), metasPadrao(), "dia da semana 7"},
		{"hora inválida", "UTC", []Turno{{Dia: 1, Inicio: "8h", Fim: "12:00"}}, matrizPadrao(), metasPadrao(), "HH:MM"},
		{"fim antes do início", "UTC", []Turno{{Dia: 1, Inicio: "12:00", Fim: "08:00"}}, matrizPadrao(), metasPadrao(), "HH:MM"},
		{"turnos demais", "UTC", cheia, matrizPadrao(), metasPadrao(), "mais de 4 turnos"},
		{"sobreposição", "UTC", sobreposta, matrizPadrao(), metasPadrao(), "se sobrepõem"},
		{"matriz incompleta", "UTC", turnosUteis(), matrizPadrao()[:8], metasPadrao(), "9 combinações"},
		{"matriz repetida", "UTC", turnosUteis(), repetida, metasPadrao(), "repete"},
		{"nível fora da faixa", "UTC", turnosUteis(), foraDaFaixa, metasPadrao(), "vão de 1 a 3"},
		{"prioridade desconhecida", "UTC", turnosUteis(), append(matrizPadrao()[:8], Celula{Impacto: 1, Urgencia: 1, Prioridade: "urgente"}), metasPadrao(), "desconhecida"},
		{"meta desconhecida", "UTC", turnosUteis(), matrizPadrao(), append(metasPadrao(), Meta{Prioridade: "urgente", RespostaMin: 1, SolucaoMin: 1}), "desconhecida"},
		{"meta repetida", "UTC", turnosUteis(), matrizPadrao(), append(metasPadrao(), metasPadrao()[0]), "repetida"},
		{"meta zerada", "UTC", turnosUteis(), matrizPadrao(), []Meta{{Prioridade: PrioridadeBaixa}}, "positivas"},
		{"meta longa", "UTC", turnosUteis(), matrizPadrao(), []Meta{{Prioridade: PrioridadeBaixa, RespostaMin: 1, SolucaoMin: MetaMaxMin + 1}}, "positivas"},
		{"resposta maior que solução", "UTC", turnosUteis(), matrizPadrao(), []Meta{{Prioridade: PrioridadeBaixa, RespostaMin: 10, SolucaoMin: 5}}, "maior que a solução"},
		{"falta meta", "UTC", turnosUteis(), matrizPadrao(), semAlta, "falta a meta da prioridade alta"},
	}
	for _, c := range casos {
		_, err := NovaPolitica(nil, c.fuso, c.turnos, c.matriz, c.metas)
		if !errors.Is(err, ErrPoliticaInvalida) || !strings.Contains(err.Error(), c.motivo) {
			t.Errorf("%s: %v", c.nome, err)
		}
	}
}

func TestNovoFeriado(t *testing.T) {
	f, err := NovoFeriado(nil, " 2026-11-02 ", " Finados ", FeriadoNacional)
	if err != nil || f.Data != "2026-11-02" || f.Nome != "Finados" {
		t.Fatalf("feriado: %+v %v", f, err)
	}
	for _, c := range [][3]string{
		{"02/11/2026", "Finados", FeriadoNacional},
		{"2026-11-02", " ", FeriadoNacional},
		{"2026-11-02", strings.Repeat("a", NomeMax+1), FeriadoNacional},
		{"2026-11-02", "Finados", "religioso"},
	} {
		if _, err := NovoFeriado(nil, c[0], c[1], c[2]); err != ErrFeriadoInvalido {
			t.Errorf("%v: %v", c, err)
		}
	}
}

func TestCalendario(t *testing.T) {
	p := politicaPadrao(t)
	finados, _ := NovoFeriado(nil, "2026-11-02", "Finados", FeriadoNacional)
	cal, err := NovoCalendario(p, []Feriado{finados})
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nome string
		a, b time.Time
		want time.Duration
	}{
		{"dentro de um turno", em("2026-10-19", "09:00"), em("2026-10-19", "10:30"), 90 * time.Minute},
		{"atravessa o almoço", em("2026-10-19", "11:00"), em("2026-10-19", "15:00"), 2 * time.Hour},
		{"antes do expediente", em("2026-10-19", "06:00"), em("2026-10-19", "07:59"), 0},
		{"fim de semana não conta", em("2026-10-23", "17:00"), em("2026-10-26", "09:00"), 2 * time.Hour},
		{"feriado não conta", em("2026-10-30", "17:00"), em("2026-11-03", "09:00"), 2 * time.Hour},
		{"intervalo invertido", em("2026-10-19", "10:00"), em("2026-10-19", "09:00"), 0},
		{"semana inteira", em("2026-10-19", "00:00"), em("2026-10-26", "00:00"), 40 * time.Hour},
	}
	for _, c := range casos {
		if got := cal.Util(c.a, c.b); got != c.want {
			t.Errorf("%s: %v, queria %v", c.nome, got, c.want)
		}
	}
	// O instante em UTC é lido no fuso da unidade.
	if got := cal.Util(em("2026-10-19", "09:00").UTC(), em("2026-10-19", "10:00").UTC()); got != time.Hour {
		t.Errorf("UTC: %v", got)
	}

	vence, ok := cal.Somar(em("2026-10-23", "17:00"), 2*time.Hour)
	if !ok || !vence.Equal(em("2026-10-26", "09:00")) {
		t.Fatalf("soma atravessa o fim de semana: %v", vence)
	}
	vence, ok = cal.Somar(em("2026-10-30", "17:30"), 30*time.Minute)
	if !ok || !vence.Equal(em("2026-10-30", "18:00")) {
		t.Fatalf("soma que cabe no turno: %v", vence)
	}
	if vence, ok = cal.Somar(em("2026-10-30", "18:00"), time.Hour); !ok || !vence.Equal(em("2026-11-03", "09:00")) {
		t.Fatalf("soma pula o feriado: %v", vence)
	}
	if _, ok := cal.Somar(em("2026-10-19", "08:00"), 24*365*time.Hour); ok {
		t.Fatal("além do horizonte, sem previsão")
	}
	if _, err := NovoCalendario(Politica{Fuso: "Marte/Olimpo"}, nil); !errors.Is(err, ErrPoliticaInvalida) {
		t.Fatalf("fuso inválido no calendário: %v", err)
	}
}

func TestMedirPrazo(t *testing.T) {
	p := politicaPadrao(t)
	cal, _ := NovoCalendario(p, nil)
	o := Oferta{ID: uuid.New(), Nome: "E-mail", Tipo: TipoIncidente, Ativa: true}
	sol, _ := NovaSolicitacao(o, "Pedido", "", uuid.New())
	if MedirPrazo(sol, nil, cal, time.Now()) != nil {
		t.Fatal("sem prioridade, sem prazo")
	}
	sol = sol.AplicarPolitica(&p) // média: resposta 120, solução 960
	if sol.Prioridade == nil || *sol.Prioridade != PrioridadeMedia || *sol.MetaRespostaMin != 120 {
		t.Fatalf("política na abertura: %+v", sol)
	}
	abertura := em("2026-10-19", "08:00")
	passagem := func(para, hora string) Passagem { return Passagem{Para: para, Em: em("2026-10-19", hora)} }
	hist := []Passagem{{Para: EstadoAberto, Em: abertura}}

	// Aberta há 1h30: a resposta corre (75%), a solução também.
	pr := MedirPrazo(sol, hist, cal, em("2026-10-19", "09:30"))
	if pr.Resposta.UsadoMin != 90 || pr.Resposta.Percentual != 75 || pr.Resposta.Situacao != SituacaoAtencao || pr.Resposta.Estado != RelogioCorrendo {
		t.Fatalf("resposta correndo: %+v", pr.Resposta)
	}
	if pr.Resposta.VenceEm == nil || !pr.Resposta.VenceEm.Equal(em("2026-10-19", "10:00")) || pr.Resposta.Limiar() != LimiarAtencao {
		t.Fatalf("previsão da resposta: %+v", pr.Resposta)
	}
	if pr.Prioridade != PrioridadeMedia || pr.Solucao.Situacao != SituacaoNoPrazo || pr.Solucao.Limiar() != 0 {
		t.Fatalf("solução no prazo: %+v", pr.Solucao)
	}

	// Atendida às 9h45, aguardando o servidor das 10h às 15h (pausa), resolvida às 16h.
	sol.Estado = EstadoResolvido
	hist = append(hist, passagem(EstadoEmAtendimento, "09:45"), passagem(EstadoAguardandoServidor, "10:00"),
		passagem(EstadoEmAtendimento, "15:00"), passagem(EstadoResolvido, "16:00"))
	pr = MedirPrazo(sol, hist, cal, em("2026-10-20", "12:00"))
	if pr.Resposta.UsadoMin != 105 || pr.Resposta.Estado != RelogioParado || pr.Resposta.Situacao != SituacaoCumprido || pr.Resposta.VenceEm != nil {
		t.Fatalf("resposta cumprida: %+v", pr.Resposta)
	}
	if pr.Solucao.UsadoMin != 180 || pr.Solucao.Estado != RelogioParado || pr.Solucao.Situacao != SituacaoCumprido || pr.Solucao.Limiar() != 0 {
		t.Fatalf("solução sem a pausa: %+v", pr.Solucao)
	}

	// Em espera, o relógio pausa e não prevê vencimento.
	sol.Estado = EstadoAguardandoTerceiro
	pr = MedirPrazo(sol, hist[:3], cal, em("2026-10-19", "17:00"))
	if pr.Solucao.Estado != RelogioPausado || pr.Solucao.VenceEm != nil || pr.Solucao.UsadoMin != 120 {
		t.Fatalf("pausa: %+v", pr.Solucao)
	}

	// Vencida e descumprida.
	sol.Estado = EstadoAberto
	tarde := MedirPrazo(sol, hist[:1], cal, em("2026-10-19", "15:00"))
	if tarde.Resposta.Situacao != SituacaoVencido || tarde.Resposta.Limiar() != LimiarVencido || tarde.Resposta.VenceEm != nil {
		t.Fatalf("resposta vencida: %+v", tarde.Resposta)
	}
	quase := MedirPrazo(sol, hist[:1], cal, em("2026-10-19", "09:50"))
	if quase.Resposta.Situacao != SituacaoCritico || quase.Resposta.Limiar() != LimiarCritico {
		t.Fatalf("resposta crítica: %+v", quase.Resposta)
	}
	sol.Estado = EstadoCancelado
	cancelada := MedirPrazo(sol, []Passagem{{Para: EstadoAberto, Em: abertura}, passagem(EstadoCancelado, "15:00")}, cal, em("2026-10-19", "16:00"))
	if cancelada.Resposta.Situacao != SituacaoDescumprido || cancelada.Resposta.Estado != RelogioParado {
		t.Fatalf("cancelada depois do prazo: %+v", cancelada.Resposta)
	}

	// Sem expediente no horizonte, sem previsão.
	vazio := Calendario{loc: sp}
	sol.Estado = EstadoAberto
	if pr := MedirPrazo(sol, hist[:1], vazio, em("2026-10-19", "09:00")); pr.Resposta.VenceEm != nil {
		t.Fatal("calendário sem turnos")
	}
}

func TestClassificarSolicitacao(t *testing.T) {
	p := politicaPadrao(t)
	o := Oferta{ID: uuid.New(), Nome: "E-mail", Tipo: TipoRequisicao, Ativa: true}
	sol, _ := NovaSolicitacao(o, "Pedido", "", uuid.New())
	if sol.Impacto != NivelPadrao || sol.Urgencia != NivelPadrao {
		t.Fatalf("abertura com níveis médios: %+v", sol)
	}
	next, mudou, err := sol.Classificar(3, 3, &p)
	if err != nil || !mudou || *next.Prioridade != PrioridadeCritica || *next.MetaSolucaoMin != 240 {
		t.Fatalf("crítica: %+v %v", next, err)
	}
	if _, mudou, _ := next.Classificar(3, 3, &p); mudou {
		t.Fatal("mesma classificação não é mudança")
	}
	sem, mudou, err := next.Classificar(3, 3, nil)
	if err != nil || !mudou || sem.Prioridade != nil {
		t.Fatalf("sem política, sem prazo: %+v %v", sem, err)
	}
	if _, _, err := sol.Classificar(0, 2, &p); err != ErrNivelInvalido {
		t.Fatal(err)
	}
	if _, _, err := sol.Classificar(2, 4, &p); err != ErrNivelInvalido {
		t.Fatal(err)
	}
	sol.Estado = EstadoEncerrado
	if _, _, err := sol.Classificar(1, 1, &p); err != ErrSolicitacaoFinalizada {
		t.Fatal(err)
	}
	if !iguais[int](nil, nil) || iguais(&p.Metas[0].RespostaMin, nil) {
		t.Fatal("comparação de ponteiros")
	}
}
