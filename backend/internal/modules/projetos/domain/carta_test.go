package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
)

func TestCartaEEtapas(t *testing.T) {
	t.Run("etapa em branco some e o título longo é recusado", func(t *testing.T) {
		etapas, err := domain.NormalizarEtapas([]string{"  Planejamento ", "", "  "})
		if err != nil || len(etapas) != 1 || etapas[0] != "Planejamento" {
			t.Fatalf("etapas = %v, erro %v", etapas, err)
		}
		if _, err := domain.NormalizarEtapas([]string{strings.Repeat("a", domain.EtapaMax+1)}); err != domain.ErrEtapaInvalida {
			t.Fatalf("esperava etapa inválida, obteve %v", err)
		}
		muitas := make([]string, domain.EtapasMax+1)
		for i := range muitas {
			muitas[i] = "etapa"
		}
		if _, err := domain.NormalizarEtapas(muitas); err != domain.ErrEtapaInvalida {
			t.Fatalf("esperava limite de etapas, obteve %v", err)
		}
	})

	t.Run("término antes do início e orçamento negativo", func(t *testing.T) {
		inicio := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
		fim := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		_, _, err := domain.ValidarAbertura(domain.EntradaProjeto{
			Titulo: "Portal", DataInicioPlanejada: &inicio, DataFimPlanejada: &fim,
		}, domain.EntradaCarta{}, nil)
		if err != domain.ErrPeriodoInvalido {
			t.Fatalf("esperava período inválido, obteve %v", err)
		}
		_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Portal", Orcamento: -1}, domain.EntradaCarta{}, nil)
		if err != domain.ErrOrcamentoInvalido {
			t.Fatalf("esperava orçamento inválido, obteve %v", err)
		}
	})

	t.Run("salvar a carta não pede etapas", func(t *testing.T) {
		plano, carta, err := (domain.Plano5W2H{
			Titulo: " Portal ", Onde: " SEMED ", CentroCusto: "CC-1",
		}).Normalizar()
		if err != nil {
			t.Fatal(err)
		}
		if plano.Titulo != "Portal" || carta.Onde != "SEMED" || carta.CentroCusto != "CC-1" {
			t.Fatalf("plano %+v carta %+v", plano, carta)
		}
	})
}
