package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
)

func TestApontamentoDeHoras(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.emAtendimento(t)
	svc := c.com(infrastructure.NewRepository(), eventosFalham{})

	// Validação de entrada inválida
	_, err := svc.RegistrarApontamento(ctx, c.tecnico, sol.ID, domain.EntradaApontamento{
		Horas:     0,
		Descricao: "Trabalho",
	})
	if err == nil {
		t.Fatal("esperava erro para 0 horas")
	}

	// Solicitante não pode apontar horas técnicas
	_, err = svc.RegistrarApontamento(ctx, c.servidor, sol.ID, domain.EntradaApontamento{
		Horas:     2.5,
		Descricao: "Diagnóstico inicial",
	})
	if !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("servidor não pode registrar apontamento: %v", err)
	}

	// Técnico registra apontamento com sucesso
	ap, err := svc.RegistrarApontamento(ctx, c.tecnico, sol.ID, domain.EntradaApontamento{
		DataApontamento: "2026-10-22",
		Horas:           3.5,
		Descricao:       "Configuração e testes de conectividade",
	})
	if err != nil {
		t.Fatalf("falha ao registrar apontamento: %v", err)
	}
	if ap.Horas != 3.5 || ap.Descricao != "Configuração e testes de conectividade" {
		t.Fatalf("apontamento com dados incorretos: %+v", ap)
	}

	// Listagem por técnico
	resumo, err := svc.ListarApontamentos(ctx, c.tecnico, sol.ID)
	if err != nil {
		t.Fatalf("falha ao listar apontamentos: %v", err)
	}
	if resumo.TotalHoras != 3.5 || len(resumo.Apontamentos) != 1 {
		t.Fatalf("esperava 3.5 horas e 1 apontamento, obteve: %+v", resumo)
	}

	// Listagem por solicitante (leitura permitida)
	resumoSol, err := svc.ListarApontamentos(ctx, c.servidor, sol.ID)
	if err != nil {
		t.Fatalf("solicitante deve poder visualizar as horas apontadas: %v", err)
	}
	if resumoSol.TotalHoras != 3.5 {
		t.Fatalf("total de horas divergente para solicitante: %v", resumoSol.TotalHoras)
	}

	// Exclusão por técnico
	if err := svc.ExcluirApontamento(ctx, c.tecnico, sol.ID, ap.ID); err != nil {
		t.Fatalf("falha ao excluir apontamento: %v", err)
	}

	resumoAposExclusao, err := svc.ListarApontamentos(ctx, c.tecnico, sol.ID)
	if err != nil {
		t.Fatalf("falha ao listar apos exclusão: %v", err)
	}
	if resumoAposExclusao.TotalHoras != 0 || len(resumoAposExclusao.Apontamentos) != 0 {
		t.Fatalf("apontamento deveria ter sido excluído: %+v", resumoAposExclusao)
	}
}
