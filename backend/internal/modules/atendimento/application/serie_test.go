package application_test

import (
	"context"
	"errors"
	"testing"
)

// A série só era gravada pelo laço que sobe com a aplicação, que corre
// contra o fim dos testes: o retrato do dia é testado aqui, sem corrida.
func TestRegistrarSerieGravaORetratoDoDia(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	c.abrir(t)

	if err := c.base.RegistrarSerie(ctx); err != nil {
		t.Fatal(err)
	}
	serie, err := c.base.Serie(ctx, c.tecnico, 7)
	if err != nil || len(serie) != 1 || serie[0].Incidentes != 1 {
		t.Fatalf("retrato da unidade: %+v %v", serie, err)
	}

	if err := c.com(falhando("ListarParaGovernanca"), eventosFalham{}).RegistrarSerie(ctx); !errors.Is(err, errFalha) {
		t.Fatalf("falha na leitura: %v", err)
	}
}
