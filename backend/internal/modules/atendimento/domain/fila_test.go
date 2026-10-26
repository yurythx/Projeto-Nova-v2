package domain

import (
	"testing"

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
