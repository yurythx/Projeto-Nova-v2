package domain_test

import (
	"strings"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
)

func TestValidarEntrada(t *testing.T) {
	t.Run("título vazio é recusado", func(t *testing.T) {
		e := domain.EntradaProjeto{Titulo: "   "}
		if err := e.Validar(); err != domain.ErrTituloObrigatorio {
			t.Fatalf("esperava ErrTituloObrigatorio, obteve %v", err)
		}
	})

	t.Run("título longo é recusado", func(t *testing.T) {
		e := domain.EntradaProjeto{Titulo: strings.Repeat("a", domain.TituloMax+1)}
		if err := e.Validar(); err != domain.ErrTituloObrigatorio {
			t.Fatalf("esperava ErrTituloObrigatorio, obteve %v", err)
		}
	})

	t.Run("prioridade desconhecida é recusada", func(t *testing.T) {
		e := domain.EntradaProjeto{Titulo: "Projeto Alpha", Prioridade: "urgente"}
		if err := e.Validar(); err != domain.ErrPrioridadeInvalida {
			t.Fatalf("esperava ErrPrioridadeInvalida, obteve %v", err)
		}
	})

	t.Run("válido", func(t *testing.T) {
		e := domain.EntradaProjeto{Titulo: "Projeto Alpha", Prioridade: domain.PrioridadeAlta}
		if err := e.Validar(); err != nil {
			t.Fatalf("esperava nil, obteve %v", err)
		}
	})
}

func TestTransicoes(t *testing.T) {
	t.Run("planejado pode ir para em_andamento ou cancelado", func(t *testing.T) {
		if !domain.ValidarTransicao(domain.EstadoPlanejado, domain.EstadoEmAndamento) {
			t.Errorf("esperava poder ir de planejado para em_andamento")
		}
		if !domain.ValidarTransicao(domain.EstadoPlanejado, domain.EstadoCancelado) {
			t.Errorf("esperava poder ir de planejado para cancelado")
		}
		if domain.ValidarTransicao(domain.EstadoPlanejado, domain.EstadoConcluido) {
			t.Errorf("não deveria ir direto de planejado para concluído")
		}
	})

	t.Run("cancelado não aceita novas transições", func(t *testing.T) {
		if len(domain.ProximosEstados(domain.EstadoCancelado)) != 0 {
			t.Errorf("cancelado não deve ter próximos estados")
		}
	})
}
