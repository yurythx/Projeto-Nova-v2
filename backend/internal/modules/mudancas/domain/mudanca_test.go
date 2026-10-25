package domain

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func janela() (*time.Time, *time.Time) {
	a := time.Now().Add(time.Hour)
	b := a.Add(time.Hour)
	return &a, &b
}

func entrada(tipo string) Entrada {
	ini, fim := janela()
	u := uuid.New()
	return Entrada{Titulo: " Trocar fonte ", Tipo: tipo, Risco: "medio", PlanoReversao: " voltar ", InicioPlanejado: ini, FimPlanejado: fim, UnidadeID: &u}
}

func TestNovaEEdicao(t *testing.T) {
	quem := uuid.New()
	m, err := Nova(entrada(TipoNormal), quem)
	if err != nil || m.Titulo != "Trocar fonte" || m.Estado != Rascunho || m.PlanoReversao != "voltar" || len(m.Vinculos) != 0 {
		t.Fatalf("nova: %+v %v", m, err)
	}
	if _, err := Nova(entrada(TipoNormal), uuid.Nil); err != ErrEntradaInvalida {
		t.Fatal("sem solicitante")
	}
	for nome, mudar := range map[string]func(*Entrada){
		"sem título":     func(e *Entrada) { e.Titulo = " " },
		"título longo":   func(e *Entrada) { e.Titulo = strings.Repeat("a", TituloMax+1) },
		"reversão longa": func(e *Entrada) { e.PlanoReversao = strings.Repeat("a", TextoMax+1) },
	} {
		in := entrada(TipoNormal)
		mudar(&in)
		if _, err := Nova(in, quem); err != ErrEntradaInvalida {
			t.Errorf("%s: %v", nome, err)
		}
	}
	tipo := entrada("magica")
	if _, err := Nova(tipo, quem); err != ErrTipoInvalido {
		t.Fatal(err)
	}
	risco := entrada(TipoNormal)
	risco.Risco = "extremo"
	if _, err := m.Aplicar(risco); err != ErrTipoInvalido {
		t.Fatal(err)
	}
	in := entrada(TipoNormal)
	in.UnidadeID = m.UnidadeID
	in.InicioPlanejado, in.FimPlanejado = m.InicioPlanejado, m.FimPlanejado
	mesma, err := m.Aplicar(in)
	if err != nil || !mesma.Igual(m) {
		t.Fatalf("mesma edição: %v", err)
	}
	outra := in
	outra.FimPlanejado = nil
	if diferente, _ := m.Aplicar(outra); diferente.Igual(m) {
		t.Fatal("janela diferente")
	}
	aprovada := m
	aprovada.Estado = Aprovada
	if _, err := aprovada.Aplicar(in); err != ErrNaoEditavel {
		t.Fatal(err)
	}
	u := uuid.New()
	if Dona(nil).UnidadeID != nil || Dona(&u).UnidadeID == nil || !MesmaUnidade(nil, nil) || MesmaUnidade(&u, nil) || !MesmaUnidade(&u, &u) {
		t.Fatal("dona e unidade")
	}
	if NovoVinculo("cometa", u) != ErrVinculoInvalido || NovoVinculo(VinculoAtivo, uuid.Nil) != ErrVinculoInvalido || NovoVinculo(VinculoAtivo, u) != nil {
		t.Fatal("vínculo")
	}
}

func passar(t *testing.T, m Mudanca, para string, autor uuid.UUID, texto string, papeis ...string) (Mudanca, *Aprovacao) {
	t.Helper()
	next, decisao, mudou, err := m.Transicionar(Transicao{Para: para, Autor: autor, Texto: texto}, papeis...)
	if err != nil || !mudou {
		t.Fatalf("%s -> %s: %v", m.Estado, para, err)
	}
	return next, decisao
}

func TestFluxoNormal(t *testing.T) {
	quem, aprovador := uuid.New(), uuid.New()
	m, _ := Nova(entrada(TipoNormal), quem)
	semJanela := m
	semJanela.FimPlanejado = semJanela.InicioPlanejado
	if _, _, _, err := semJanela.Transicionar(Transicao{Para: EmAvaliacao, Autor: quem}, PapelGestor); err != ErrSemJanela {
		t.Fatal(err)
	}
	semReversao := m
	semReversao.PlanoReversao = ""
	if _, _, _, err := semReversao.Transicionar(Transicao{Para: EmAvaliacao, Autor: quem}, PapelGestor); err != ErrSemReversao {
		t.Fatal(err)
	}
	m, decisao := passar(t, m, EmAvaliacao, quem, "", PapelGestor)
	if decisao != nil {
		t.Fatal("normal não é pré-aprovada")
	}
	if _, _, mudou, _ := m.Transicionar(Transicao{Para: EmAvaliacao, Autor: quem}, PapelGestor); mudou {
		t.Fatal("mesmo estado")
	}
	if got := m.PodeIrPara(quem, PapelGestor, PapelAprovador); !slices.Equal(got, []string{Cancelada}) {
		t.Fatalf("quem pediu não vê a aprovação: %v", got)
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Aprovada, Autor: quem}, PapelGestor, PapelAprovador); err != ErrAutoAprovacao {
		t.Fatal(err)
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Aprovada, Autor: aprovador}, PapelGestor); err != ErrTransicaoInvalida {
		t.Fatal("gestor não aprova")
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Rejeitada, Autor: aprovador}, PapelAprovador); err != ErrSemParecer {
		t.Fatal(err)
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Rejeitada, Autor: aprovador, Texto: strings.Repeat("a", TextoMax+1)}, PapelAprovador); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	rej, d := passar(t, m, Rejeitada, aprovador, "não", PapelAprovador)
	if d.Decisao != DecisaoRejeitada || *d.AprovadorID != aprovador {
		t.Fatalf("decisão: %+v", d)
	}
	m, _ = passar(t, rej, EmAvaliacao, quem, "", PapelGestor)
	m, d = passar(t, m, Aprovada, aprovador, " ok ", PapelAprovador)
	if d.Parecer != "ok" {
		t.Fatalf("parecer: %+v", d)
	}
	m, _ = passar(t, m, EmExecucao, quem, "", PapelGestor)
	if m.ExecutadaInicio == nil {
		t.Fatal("início da execução")
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Falhou, Autor: quem}, PapelGestor); err != ErrSemResultado {
		t.Fatal(err)
	}
	m, _ = passar(t, m, Concluida, quem, "feito", PapelGestor)
	if !m.Encerrada() || m.ExecutadaFim == nil || m.Resultado != "feito" {
		t.Fatalf("concluída: %+v", m)
	}
	if _, _, _, err := m.Transicionar(Transicao{Para: Cancelada, Autor: quem}, PapelGestor); err != ErrMudancaEncerrada {
		t.Fatal(err)
	}
}

func TestPadraoEFalha(t *testing.T) {
	quem := uuid.New()
	in := entrada(TipoPadrao)
	in.PlanoReversao = ""
	m, _ := Nova(in, quem)
	m, d := passar(t, m, EmAvaliacao, quem, "", PapelGestor)
	if m.Estado != Aprovada || d.Decisao != DecisaoAutomatica || d.AprovadorID != nil {
		t.Fatalf("padrão pré-aprovada: %+v %+v", m, d)
	}
	m, _ = passar(t, m, EmExecucao, quem, "", PapelGestor)
	m, _ = passar(t, m, Falhou, quem, "revertido", PapelGestor)
	if m.Estado != Falhou {
		t.Fatal("falhou")
	}
}

func TestVinculoRecusado(t *testing.T) {
	var vazio *VinculoRecusado
	if vazio.Error() == "" {
		t.Fatal("nil")
	}
	if (&VinculoRecusado{Mensagem: "parada"}).Error() != "parada" {
		t.Fatal("mensagem")
	}
}
