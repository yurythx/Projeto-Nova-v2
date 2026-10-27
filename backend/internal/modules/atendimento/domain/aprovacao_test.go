package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidarEtapas(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	ok, err := ValidarEtapas(TipoRequisicao, []EtapaAprovacao{{Ordem: 7, Nome: " Chefia ", Aprovadores: []uuid.UUID{a}}, {Nome: "TI", Aprovadores: []uuid.UUID{a, b}}})
	if err != nil || len(ok) != 2 || ok[0].Ordem != 0 || ok[0].Nome != "Chefia" || ok[1].Ordem != 1 {
		t.Fatalf("etapas: %+v %v", ok, err)
	}
	if vazio, err := ValidarEtapas(TipoIncidente, nil); err != nil || len(vazio) != 0 {
		t.Fatalf("sem etapas, qualquer tipo: %v %v", vazio, err)
	}
	if _, err := ValidarEtapas(TipoIncidente, []EtapaAprovacao{{Nome: "x", Aprovadores: []uuid.UUID{a}}}); !errors.Is(err, ErrEtapasSoRequisicao) {
		t.Fatalf("incidente com etapa: %v", err)
	}
	muitas := make([]EtapaAprovacao, MaxEtapas+1)
	for i := range muitas {
		muitas[i] = EtapaAprovacao{Nome: "e", Aprovadores: []uuid.UUID{a}}
	}
	gente := make([]uuid.UUID, MaxAprovadores+1)
	for i := range gente {
		gente[i] = uuid.New()
	}
	for nome, etapas := range map[string][]EtapaAprovacao{
		"etapas demais":      muitas,
		"nome vazio":         {{Nome: " ", Aprovadores: []uuid.UUID{a}}},
		"nome longo":         {{Nome: strings.Repeat("n", maxNomeEtapa+1), Aprovadores: []uuid.UUID{a}}},
		"sem aprovador":      {{Nome: "x"}},
		"aprovadores demais": {{Nome: "x", Aprovadores: gente}},
		"aprovador vazio":    {{Nome: "x", Aprovadores: []uuid.UUID{uuid.Nil}}},
		"aprovador repetido": {{Nome: "x", Aprovadores: []uuid.UUID{a, a}}},
	} {
		if _, err := ValidarEtapas(TipoRequisicao, etapas); !errors.Is(err, ErrEtapaInvalida) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func TestDecidirEmEtapas(t *testing.T) {
	quem, chefe, ti := uuid.New(), uuid.New(), uuid.New()
	etapas := []EtapaAprovacao{{Ordem: 0, Nome: "Chefia", Aprovadores: []uuid.UUID{chefe, quem}}, {Ordem: 1, Nome: "TI", Aprovadores: []uuid.UUID{ti}}}
	agora := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	s := Solicitacao{ID: uuid.New(), Tipo: TipoRequisicao, Estado: EstadoAberto, SolicitanteID: quem}.AguardarAprovacao()
	if s.Estado != EstadoAguardandoAprovacao || *s.EtapaAtual != 0 {
		t.Fatalf("aguardando: %+v", s)
	}

	if _, _, _, err := s.Decidir(etapas, ti, true, "", agora); !errors.Is(err, ErrNaoAprovador) {
		t.Fatalf("aprovador de outra etapa: %v", err)
	}
	if _, _, _, err := s.Decidir(etapas, quem, true, "", agora); !errors.Is(err, ErrAutoaprovacao) {
		t.Fatalf("quem pediu não aprova: %v", err)
	}
	if _, _, _, err := s.Decidir(etapas, chefe, true, strings.Repeat("m", MaxMotivoRecusa+1), agora); !errors.Is(err, ErrMotivoObrigatorio) {
		t.Fatalf("motivo longo: %v", err)
	}
	s1, a1, liberada, err := s.Decidir(etapas, chefe, true, " ok ", agora)
	if err != nil || liberada || *s1.EtapaAtual != 1 || a1.Decisao != DecisaoAprovada || a1.EtapaNome != "Chefia" || a1.Motivo != "ok" {
		t.Fatalf("primeira etapa: %+v %+v %v", s1, a1, err)
	}
	fim, a2, liberada, err := s1.Decidir(etapas, ti, true, "", agora)
	if err != nil || !liberada || fim.Estado != EstadoAberto || fim.EtapaAtual != nil || a2.Etapa != 1 {
		t.Fatalf("última etapa libera: %+v %v", fim, err)
	}
	if _, _, _, err := fim.Decidir(etapas, ti, true, "", agora); !errors.Is(err, ErrSemAprovacaoPendente) {
		t.Fatalf("já liberado: %v", err)
	}
	if _, _, _, err := s1.Decidir(etapas[:1], ti, true, "", agora); !errors.Is(err, ErrSemAprovacaoPendente) {
		t.Fatalf("etapa que deixou de existir: %v", err)
	}

	if _, _, _, err := s1.Decidir(etapas, ti, false, "  ", agora); !errors.Is(err, ErrMotivoObrigatorio) {
		t.Fatalf("recusa sem motivo: %v", err)
	}
	rec, ar, liberada, err := s1.Decidir(etapas, ti, false, "Sem orçamento", agora)
	if err != nil || liberada || rec.Estado != EstadoCancelado || rec.EtapaAtual != nil || rec.EncerradaEm == nil || ar.Decisao != DecisaoRecusa {
		t.Fatalf("recusa: %+v %+v %v", rec, ar, err)
	}
}

func TestAguardandoAprovacaoNaMaquina(t *testing.T) {
	tec, quem := uuid.New(), uuid.New()
	s := Solicitacao{ID: uuid.New(), Tipo: TipoRequisicao, SolicitanteID: quem}.AguardarAprovacao().ComPassagens([]string{"cx1234abcd"})
	if got := s.Proximos(PapelTecnico); len(got) != 1 || got[0] != EstadoCancelado {
		t.Fatalf("a equipe só cancela, sem colunas extras: %v", got)
	}
	if got := s.Proximos(PapelSolicitante); len(got) != 1 || got[0] != EstadoCancelado {
		t.Fatalf("quem pediu desiste: %v", got)
	}
	if _, _, err := s.Atribuir(tec); !errors.Is(err, ErrAguardandoAprovacao) {
		t.Fatalf("ninguém assume antes de aprovar: %v", err)
	}
	c, mudou, err := s.Transicionar(EstadoCancelado, PapelSolicitante)
	if err != nil || !mudou || c.EtapaAtual != nil {
		t.Fatalf("cancelar limpa a etapa: %+v %v", c, err)
	}

	// O prazo fica pausado até a fila: o tempo de espera não conta.
	alta, resp, sol := "alta", 30, 240
	s.Prioridade, s.MetaRespostaMin, s.MetaSolucaoMin = &alta, &resp, &sol
	agora := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	cal, _ := NovoCalendario(politicaPadrao(t), nil)
	p := MedirPrazo(s, []Passagem{{Para: EstadoAguardandoAprovacao, Em: agora.Add(-5 * time.Hour)}}, cal, agora)
	if p.Resposta.Estado != RelogioPausado || p.Solucao.Estado != RelogioPausado || p.Solucao.UsadoMin != 0 {
		t.Fatalf("prazo: %+v", p)
	}
}
