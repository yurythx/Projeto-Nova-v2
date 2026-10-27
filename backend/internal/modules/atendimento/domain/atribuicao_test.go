package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTempoDaEquipe(t *testing.T) {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	h := func(n int) time.Time { return base.Add(time.Duration(n) * time.Hour) }
	fim := h(10)
	a := Atribuicao{Inicio: h(0), Fim: &fim}
	// Esperas: servidor das 2 às 4; fornecedor das 9 até depois do fim do
	// trecho; uma espera antes do trecho começar não conta.
	ps := []Passagem{
		{Para: EstadoEmAtendimento, Em: h(4)},
		{Para: EstadoAguardandoServidor, Em: h(2)},
		{Para: EstadoAguardandoTerceiro, Em: h(-3)},
		{Para: EstadoEmAtendimento, Em: h(-1)},
		{Para: EstadoAguardandoTerceiro, Em: h(9)},
	}
	if got := TempoDaEquipe(a, ps, h(20)); got != 7*time.Hour {
		t.Fatalf("trecho fechado: %v", got)
	}
	aberto := Atribuicao{Inicio: h(0)}
	if got := TempoDaEquipe(aberto, ps[:4], h(6)); got != 4*time.Hour {
		t.Fatalf("trecho aberto conta até agora: %v", got)
	}
	if got := TempoDaEquipe(aberto, nil, h(-1)); got != 0 {
		t.Fatalf("nunca negativo: %v", got)
	}
}

func TestOLADosChamados(t *testing.T) {
	agora := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	rede, n1 := uuid.New(), uuid.New()
	ola := 8
	h := func(n int) time.Time { return agora.Add(time.Duration(-100+n) * time.Hour) }
	f := func(n int) *time.Time { v := h(n); return &v }
	a, b := uuid.New(), uuid.New()
	atribs := []Atribuicao{
		{SolicitacaoID: a, EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Inicio: h(0), Fim: f(6)},
		{SolicitacaoID: b, EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Inicio: h(0), Fim: f(12)},
		{SolicitacaoID: a, EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Inicio: h(90)},
		{SolicitacaoID: b, EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Inicio: h(98)},
		{SolicitacaoID: b, EquipeID: n1, EquipeNome: "N1", Inicio: h(0), Fim: f(2)},
	}
	// O chamado b esperou o servidor por 6 h no primeiro trecho: fica em 6 h.
	passagens := map[uuid.UUID][]Passagem{b: {{Para: EstadoAguardandoServidor, Em: h(3)}, {Para: EstadoEmAtendimento, Em: h(9)}}}
	tarefa := []TarefaMedida{{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaPendente, CreatedAt: h(99)}}
	linhas := ResumirEquipes(tarefa, atribs, passagens, agora)
	if len(linhas) != 2 || linhas[0].Nome != "N1" {
		t.Fatalf("por nome: %+v", linhas)
	}
	r := linhas[1]
	if r.Chamados != 2 || r.ChamadosNoPrazo != 2 || *r.ChamadosNoPrazoPercentual != 100 || *r.ChamadosMedianaHoras != 6 ||
		r.ComAEquipe != 2 || r.ChamadosAtrasados != 1 || r.Pendentes != 1 {
		t.Fatalf("rede: %+v", r)
	}
	if n := linhas[0]; n.Chamados != 1 || n.ChamadosNoPrazoPercentual != nil || *n.ChamadosMedianaHoras != 2 {
		t.Fatalf("sem OLA: %+v", n)
	}
}
