package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPrimeiraRegraCasaEPara(t *testing.T) {
	oferta := uuid.New()
	outra := uuid.New()
	unidade := uuid.New()
	equipe := uuid.New()
	regras := []Regra{
		{Nome: "outra oferta", Ativa: true, OfertaID: &outra, Prioridade: PrioridadeCritica, Ordem: 1},
		{Nome: "senha", Ativa: true, Palavra: "senha", EquipeID: &equipe, Ordem: 2},
		{Nome: "todas", Ativa: true, Prioridade: PrioridadeBaixa, Ordem: 9},
	}
	pegou := Primeira(regras, oferta, "A Senha expirou", []uuid.UUID{unidade})
	if pegou == nil || pegou.Nome != "senha" {
		t.Fatalf("primeira: %+v", pegou)
	}
	if Primeira(regras, oferta, "impressora", nil).Nome != "todas" {
		t.Fatal("a regra sem condição fica por último")
	}
	inativa := []Regra{{Nome: "parada", Ativa: false, Prioridade: PrioridadeAlta}}
	if Primeira(inativa, oferta, "senha", nil) != nil {
		t.Fatal("regra inativa não casa")
	}
	soUnidade := []Regra{{Nome: "lotação", Ativa: true, UnidadeSolicitanteID: &unidade, Prioridade: PrioridadeAlta}}
	if Primeira(soUnidade, oferta, "x", nil) != nil {
		t.Fatal("sem lotação não casa a unidade")
	}
	if Primeira(soUnidade, oferta, "x", []uuid.UUID{unidade}) == nil {
		t.Fatal("a lotação casa")
	}
}

func TestNovaRegraRecusaAcaoVazia(t *testing.T) {
	if _, err := NovaRegra(Regra{Nome: "Solta"}); err == nil {
		t.Fatal("sem prioridade e sem equipe")
	}
	regra, err := NovaRegra(Regra{Nome: " Senha ", Palavra: "SENHA", Prioridade: PrioridadeCritica, Ordem: 10})
	if err != nil || regra.Palavra != "senha" || regra.ID == uuid.Nil || !regra.Ativa {
		t.Fatalf("%v %+v", err, regra)
	}
	equipe := uuid.New()
	id := uuid.New()
	criada := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	soEquipe, err := NovaRegra(Regra{ID: id, Nome: "Equipe", EquipeID: &equipe, CreatedAt: criada})
	if err != nil || soEquipe.ID != id || !soEquipe.CreatedAt.Equal(criada) {
		t.Fatalf("equipe: %v %+v", err, soEquipe)
	}
}

func TestCadastroDaFilaRecusa(t *testing.T) {
	recusas := []error{
		deveRecusar(t, func() error { _, err := NovaRegra(Regra{Nome: ""}); return err }),
		deveRecusar(t, func() error { _, err := NovaRegra(Regra{Nome: strings.Repeat("a", NomeMax+1)}); return err }),
		deveRecusar(t, func() error {
			_, err := NovaRegra(Regra{Nome: "Ordem", Ordem: 1000, Prioridade: PrioridadeBaixa})
			return err
		}),
		deveRecusar(t, func() error {
			_, err := NovaRegra(Regra{Nome: "Palavra", Palavra: strings.Repeat("á", 81), Prioridade: PrioridadeBaixa})
			return err
		}),
		deveRecusar(t, func() error {
			_, err := NovaRegra(Regra{Nome: "Prioridade", Prioridade: "urgente"})
			return err
		}),
		deveRecusar(t, func() error { _, err := NovaResposta(uuid.Nil, "oi", ""); return err }),
		deveRecusar(t, func() error {
			_, err := NovaResposta(uuid.New(), strings.Repeat("a", TextoMax+1), "")
			return err
		}),
		deveRecusar(t, func() error { _, err := NovaResposta(uuid.New(), " ", " "); return err }),
		deveRecusar(t, func() error { _, err := NovoChamadoMensal(ChamadoMensal{}); return err }),
		deveRecusar(t, func() error {
			_, err := NovoChamadoMensal(ChamadoMensal{OfertaID: uuid.New(), Titulo: "", CriadoPor: uuid.New(), Dia: 1})
			return err
		}),
		deveRecusar(t, func() error {
			_, err := NovoChamadoMensal(ChamadoMensal{OfertaID: uuid.New(), Titulo: "Backup", Descricao: strings.Repeat("a", TextoMax+1), CriadoPor: uuid.New(), Dia: 1})
			return err
		}),
		deveRecusar(t, func() error {
			_, err := NovoChamadoMensal(ChamadoMensal{OfertaID: uuid.New(), Titulo: "Backup", CriadoPor: uuid.New(), Dia: 29})
			return err
		}),
		deveRecusar(t, func() error {
			_, err := NovoChamadoMensal(ChamadoMensal{OfertaID: uuid.New(), Titulo: "Backup", Dia: 1})
			return err
		}),
	}
	for _, err := range recusas {
		if err == nil || err.Error() == "" {
			t.Fatal("recusa sem texto")
		}
	}
	pronta, err := NovaResposta(uuid.New(), " Olá ", "")
	if err != nil || pronta.Publico != "Olá" {
		t.Fatalf("resposta: %v %+v", err, pronta)
	}
	id := uuid.New()
	criada := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	mensal, err := NovoChamadoMensal(ChamadoMensal{
		ID: id, OfertaID: uuid.New(), Titulo: " Backup ", CriadoPor: uuid.New(), Dia: 28, CreatedAt: criada,
	})
	if err != nil || mensal.ID != id || mensal.Titulo != "Backup" || !mensal.CreatedAt.Equal(criada) || !mensal.Ativa {
		t.Fatalf("mensal: %v %+v", err, mensal)
	}
	negativo := ResumirDeflexao(4, -1)
	if negativo == nil || negativo.SemPedido != 0 {
		t.Fatalf("negativo: %+v", negativo)
	}
	estoura := ResumirDeflexao(4, 9)
	if estoura == nil || estoura.SemPedido != 4 || estoura.Taxa != 100 {
		t.Fatalf("teto: %+v", estoura)
	}
}

func deveRecusar(t *testing.T, fn func() error) error {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatal("aceitou o que devia recusar")
	}
	return err
}

func TestDeflexaoSemVistaFicaVazia(t *testing.T) {
	if ResumirDeflexao(0, 0) != nil {
		t.Fatal("sem vista não há taxa")
	}
	d := ResumirDeflexao(4, 1)
	if d == nil || d.Viram != 4 || d.SemPedido != 1 || d.Taxa != 25 {
		t.Fatalf("%+v", d)
	}
}
