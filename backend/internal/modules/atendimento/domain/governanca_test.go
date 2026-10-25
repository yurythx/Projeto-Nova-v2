package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func solConcluida(estado string) Solicitacao {
	return Solicitacao{
		ID: uuid.New(), Estado: estado, SolicitanteID: uuid.New(), Tipo: TipoIncidente,
	}
}

func TestNaoConcluivel(t *testing.T) {
	var vazio *NaoConcluivel
	if vazio.Error() == "" || vazio.Unwrap() != nil {
		t.Fatal(vazio)
	}
	sem := &NaoConcluivel{}
	if sem.Error() == "" || sem.Unwrap() != nil {
		t.Fatal(sem)
	}
	com := &NaoConcluivel{Causa: ErrTransicaoInvalida}
	if com.Unwrap() != ErrTransicaoInvalida || com.Error() != ErrTransicaoInvalida.Error() {
		t.Fatal(com)
	}
}

func TestNovaAvaliacao(t *testing.T) {
	sol := solConcluida(EstadoResolvido)
	if _, err := NovaAvaliacao(sol, uuid.New(), 5, ""); err != ErrAvaliacaoAlheia {
		t.Fatalf("outro autor: %v", err)
	}
	aberta := sol
	aberta.Estado = EstadoAberto
	if _, err := NovaAvaliacao(aberta, aberta.SolicitanteID, 5, ""); err != ErrAvaliacaoForaDeHora {
		t.Fatalf("aberta: %v", err)
	}
	if _, err := NovaAvaliacao(sol, sol.SolicitanteID, 0, ""); err != ErrAvaliacaoInvalida {
		t.Fatalf("nota: %v", err)
	}
	if _, err := NovaAvaliacao(sol, sol.SolicitanteID, 5, strings.Repeat("a", TextoMax+1)); err != ErrAvaliacaoInvalida {
		t.Fatalf("comentário: %v", err)
	}
	got, err := NovaAvaliacao(sol, sol.SolicitanteID, 4, "  rápido  ")
	if err != nil || got.Nota != 4 || got.Comentario != "rápido" || got.SolicitacaoID != sol.ID {
		t.Fatalf("avaliação: %+v %v", got, err)
	}
	encerrada := solConcluida(EstadoEncerrado)
	if _, err := NovaAvaliacao(encerrada, encerrada.SolicitanteID, 1, ""); err != nil {
		t.Fatal(err)
	}
}

func TestResumirSemInventarNumero(t *testing.T) {
	ind := Resumir(nil, nil)
	if ind.MTTRHoras != nil || ind.MTBFHoras != nil || ind.SLAPercentual != nil || ind.FCRPercentual != nil {
		t.Fatalf("amostra vazia não inventa média: %+v", ind)
	}
}

func TestResumir(t *testing.T) {
	a := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	b := a.Add(4 * time.Hour)
	meta := 60
	prio := PrioridadeMedia
	um := uuid.New()
	dois := uuid.New()
	resolvido := Solicitacao{
		ID: um, Numero: 1, Tipo: TipoIncidente, Estado: EstadoResolvido, CreatedAt: a,
		Prioridade: &prio, Prazo: &Prazo{Solucao: Relogio{UsadoMin: 90, Estado: RelogioParado, Situacao: SituacaoCumprido, MetaMin: meta}},
	}
	outro := Solicitacao{
		ID: dois, Numero: 2, Tipo: TipoIncidente, Estado: EstadoEncerrado, CreatedAt: b,
		Prioridade: &prio, Prazo: &Prazo{Solucao: Relogio{UsadoMin: 30, Estado: RelogioParado, Situacao: SituacaoDescumprido, MetaMin: meta}},
	}
	aberto := Solicitacao{ID: uuid.New(), Numero: 3, Tipo: TipoIncidente, Estado: EstadoAberto, CreatedAt: a}
	passagens := map[uuid.UUID][]Passagem{
		um:   {{Para: EstadoAberto}, {Para: EstadoEmAtendimento}, {Para: EstadoResolvido}},
		dois: {{Para: EstadoAberto}, {Para: EstadoEmAtendimento}, {Para: EstadoAguardandoServidor}, {Para: EstadoEncerrado}},
	}
	ind := Resumir([]Solicitacao{outro, aberto, resolvido}, passagens)
	if ind.Amostra.Incidentes != 3 || ind.Amostra.Concluidas != 2 || ind.Amostra.FCR != 1 {
		t.Fatalf("amostra: %+v", ind.Amostra)
	}
	if ind.MTTRHoras == nil || *ind.MTTRHoras != 1 {
		t.Fatalf("MTTR = média de 90 e 30 minutos: %v", ind.MTTRHoras)
	}
	if ind.MTBFHoras == nil || *ind.MTBFHoras != 2 {
		t.Fatalf("MTBF = média dos intervalos 0 h e 4 h entre as três aberturas: %v", *ind.MTBFHoras)
	}
	if ind.SLAPercentual == nil || *ind.SLAPercentual != 50 {
		t.Fatalf("SLA: %v", ind.SLAPercentual)
	}
	if ind.FCRPercentual == nil || *ind.FCRPercentual != 50 {
		t.Fatalf("FCR: %v", ind.FCRPercentual)
	}
}
