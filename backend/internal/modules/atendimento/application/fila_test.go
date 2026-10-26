package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

func TestRegraNaAberturaSobeAPrioridade(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	c.comPolitica(t)
	var equipe uuid.UUID
	if err := c.pool.QueryRow(ctx, `SELECT id FROM atendimento_equipes WHERE unidade_id = $1`, c.unidade).Scan(&equipe); err != nil {
		t.Fatal(err)
	}
	_, err := c.base.CriarRegra(ctx, admin(), domain.Regra{
		Nome: "Senha crítica", Ordem: 1, Palavra: "senha", Prioridade: domain.PrioridadeCritica,
		EquipeID: &equipe, UnidadeID: &c.unidade, OfertaID: &c.oferta.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_regras WHERE unidade_id = $1`, c.unidade)
	})

	subiu, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "A senha expirou"})
	if err != nil || subiu.Prioridade == nil || *subiu.Prioridade != domain.PrioridadeCritica || subiu.Impacto != 3 {
		t.Fatalf("regra: %+v %v", subiu, err)
	}
	ficou, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Impressora"})
	if err != nil || ficou.Prioridade == nil || *ficou.Prioridade != domain.PrioridadeMedia {
		t.Fatalf("sem regra: %+v %v", ficou, err)
	}
}
