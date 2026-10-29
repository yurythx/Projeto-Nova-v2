package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

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

func TestVinculosEApontamentos(t *testing.T) {
	id := uuid.New()
	for _, tipo := range []string{domain.VinculoMudanca, domain.VinculoProblema, domain.VinculoAtivo, domain.VinculoSolicitacao} {
		if err := domain.NovoVinculo(tipo, id); err != nil {
			t.Fatalf("tipo %s deveria ser válido: %v", tipo, err)
		}
	}
	if err := domain.NovoVinculo("invalido", id); err != domain.ErrVinculoInvalido {
		t.Fatalf("esperava ErrVinculoInvalido, obteve %v", err)
	}
	if err := domain.NovoVinculo(domain.VinculoMudanca, uuid.Nil); err != domain.ErrVinculoInvalido {
		t.Fatalf("esperava ErrVinculoInvalido para uuid.Nil, obteve %v", err)
	}

	uA, uB := uuid.New(), uuid.New()
	if !domain.MesmaUnidadeCompativel(nil, &uA) || !domain.MesmaUnidadeCompativel(&uA, nil) || !domain.MesmaUnidadeCompativel(&uA, &uA) {
		t.Fatal("unidades compatíveis recusadas")
	}
	if domain.MesmaUnidadeCompativel(&uA, &uB) {
		t.Fatal("unidades distintas aceitas")
	}

	if err := (domain.Apontamento{Horas: 0, Descricao: "ok"}).Validar(); err != domain.ErrApontamentoInvalido {
		t.Fatalf("horas zero: %v", err)
	}
	if err := (domain.Apontamento{Horas: 25, Descricao: "ok"}).Validar(); err != domain.ErrApontamentoInvalido {
		t.Fatalf("horas > 24: %v", err)
	}
	if err := (domain.Apontamento{Horas: 4, Descricao: "  "}).Validar(); err != domain.ErrApontamentoInvalido {
		t.Fatalf("descrição vazia: %v", err)
	}
	if err := (domain.Apontamento{Horas: 4, Descricao: "Análise"}).Validar(); err != nil {
		t.Fatalf("apontamento válido: %v", err)
	}
}
