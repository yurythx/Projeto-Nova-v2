package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var t0 = time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

func declarada(t *testing.T) (Ocorrencia, Atualizacao) {
	t.Helper()
	chamado, quem := uuid.New(), uuid.New()
	o, a, err := Declarar("  Protocolo eletrônico fora do ar ", []string{" Protocolo ", "Protocolo", "", "SEI"}, ImpactoFora,
		" Estamos investigando. ", &chamado, quem, t0)
	if err != nil {
		t.Fatal(err)
	}
	return o, a
}

func TestDeclarar(t *testing.T) {
	o, a := declarada(t)
	if o.Titulo != "Protocolo eletrônico fora do ar" || o.Estado != EstadoInvestigando || o.Impacto != ImpactoFora ||
		len(o.Servicos) != 2 || o.Servicos[0] != "Protocolo" || o.CoordenadorID == nil || !o.IniciadaEm.Equal(t0) || o.Resolvida() {
		t.Fatalf("ocorrência: %+v", o)
	}
	if a.OcorrenciaID != o.ID || a.Texto != "Estamos investigando." || a.AutorID == nil || *a.AutorID != *o.CoordenadorID {
		t.Fatalf("primeira notícia: %+v", a)
	}
	// Conta de serviço: sem coordenador nem autor.
	sem, semAutor, err := Declarar("X", []string{"Y"}, ImpactoParcial, "z", nil, uuid.Nil, t0)
	if err != nil || sem.CoordenadorID != nil || semAutor.AutorID != nil {
		t.Fatalf("sem pessoa: %+v %+v %v", sem, semAutor, err)
	}
	muitos := make([]string, maxServicos+1)
	for i := range muitos {
		muitos[i] = strings.Repeat("s", i+1)
	}
	for nome, k := range map[string]struct {
		titulo, impacto, texto string
		servicos               []string
	}{
		"título vazio":         {"  ", ImpactoFora, "x", []string{"A"}},
		"título longo":         {strings.Repeat("t", maxTitulo+1), ImpactoFora, "x", []string{"A"}},
		"sem serviço":          {"T", ImpactoFora, "x", []string{" ", ""}},
		"serviço longo":        {"T", ImpactoFora, "x", []string{strings.Repeat("s", maxServico+1)}},
		"serviços demais":      {"T", ImpactoFora, "x", muitos},
		"impacto inventado":    {"T", "catastrofe", "x", []string{"A"}},
		"sem texto ao público": {"T", ImpactoFora, "  ", []string{"A"}},
	} {
		if _, _, err := Declarar(k.titulo, k.servicos, k.impacto, k.texto, nil, uuid.New(), t0); !errors.Is(err, ErrEntradaInvalida) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func TestCicloDaOcorrencia(t *testing.T) {
	o, _ := declarada(t)
	quem := uuid.New()
	t1 := t0.Add(time.Hour)
	o, a, err := o.Atualizar(EstadoIdentificado, ImpactoParcial, "O certificado venceu.", quem, t1)
	if err != nil || o.Estado != EstadoIdentificado || o.Impacto != ImpactoParcial || !o.UpdatedAt.Equal(t1) || a.Impacto != ImpactoParcial {
		t.Fatalf("identificado: %+v %v", o, err)
	}
	if _, err := o.Revisar("cedo demais", t1); !errors.Is(err, ErrAindaAberta) {
		t.Fatalf("revisar aberta: %v", err)
	}
	if o, err = o.TrocarCoordenador(quem, t1); err != nil || *o.CoordenadorID != quem {
		t.Fatalf("troca de coordenador: %+v %v", o, err)
	}
	if _, err := o.TrocarCoordenador(uuid.Nil, t1); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("coordenador vazio: %v", err)
	}
	for nome, k := range map[string][3]string{
		"estado inventado":  {"pausado", ImpactoFora, "x"},
		"impacto inventado": {EstadoMonitorando, "leve", "x"},
		"texto vazio":       {EstadoMonitorando, ImpactoFora, " "},
		"texto longo":       {EstadoMonitorando, ImpactoFora, strings.Repeat("a", maxTexto+1)},
	} {
		if _, _, err := o.Atualizar(k[0], k[1], k[2], quem, t1); !errors.Is(err, ErrEntradaInvalida) {
			t.Errorf("%s: %v", nome, err)
		}
	}

	t2 := t1.Add(time.Hour)
	o, _, err = o.Atualizar(EstadoResolvido, ImpactoParcial, "Voltou ao normal.", quem, t2)
	if err != nil || !o.Resolvida() || o.ResolvidaEm == nil || !o.ResolvidaEm.Equal(t2) {
		t.Fatalf("resolvida: %+v %v", o, err)
	}
	if _, _, err := o.Atualizar(EstadoMonitorando, ImpactoFora, "reabrir", quem, t2); !errors.Is(err, ErrJaResolvida) {
		t.Fatalf("atualizar resolvida: %v", err)
	}
	if _, err := o.TrocarCoordenador(quem, t2); !errors.Is(err, ErrJaResolvida) {
		t.Fatalf("coordenar resolvida: %v", err)
	}
	if _, err := o.Revisar(" ", t2); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("revisão vazia: %v", err)
	}
	if r, err := o.Revisar(" Causa: certificado sem renovação automática. ", t2); err != nil || r.Revisao != "Causa: certificado sem renovação automática." || !r.RevisadaEm.Equal(t2) {
		t.Fatalf("revisão: %+v %v", r, err)
	}
}

func TestPaginaPublicaSoLevaOQueEPublico(t *testing.T) {
	aberta, a := declarada(t)
	aberta.CoordenadorNome = "Maria Coordenadora"
	a.AutorNome = "Maria Coordenadora"
	segunda := Atualizacao{Estado: EstadoIdentificado, Texto: "Causa encontrada.", CreatedAt: t0.Add(time.Hour), AutorNome: "João"}
	aberta.Atualizacoes = []Atualizacao{a, segunda}

	leve, _, _ := Declarar("Lentidão no e-mail", []string{"E-mail"}, ImpactoDegradado, "Lento.", nil, uuid.New(), t0)
	resolvida, _, _ := Declarar("Rede da Saúde", []string{"Rede"}, ImpactoFora, "Fora.", nil, uuid.New(), t0)
	resolvida, _, _ = resolvida.Atualizar(EstadoResolvido, ImpactoFora, "Voltou.", uuid.New(), t0.Add(time.Hour))
	resolvida.Revisao = "Interno: o switch caiu."

	s := MontarSituacao([]Ocorrencia{leve, aberta, resolvida})
	if s.Geral != ImpactoFora || len(s.Abertas) != 2 || len(s.Recentes) != 1 {
		t.Fatalf("situação: %+v", s)
	}
	pub := s.Abertas[1]
	if len(pub.Atualizacoes) != 2 || pub.Atualizacoes[0].Texto != "Causa encontrada." {
		t.Fatalf("a mais nova vem primeiro: %+v", pub.Atualizacoes)
	}
	corpo, _ := json.Marshal(s)
	for _, interno := range []string{"Maria", "João", "chamado", "coordenador", "autor", "revisao", "switch caiu"} {
		if strings.Contains(string(corpo), interno) {
			t.Errorf("a página pública leva %q: %s", interno, corpo)
		}
	}
	if MontarSituacao(nil).Geral != SituacaoOperacional || MontarSituacao([]Ocorrencia{resolvida}).Geral != SituacaoOperacional {
		t.Fatal("sem ocorrência aberta, operacional")
	}
	if MontarSituacao([]Ocorrencia{leve}).Geral != ImpactoDegradado {
		t.Fatal("o pior impacto em aberto")
	}
}
