package application

import (
	"errors"
	"testing"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
)

func TestMapErrorCobreCadaRecusa(t *testing.T) {
	casos := []struct {
		err     error
		status  int
		message string
	}{
		{domain.ErrQuadro, 404, "coluna não encontrada"},
		{domain.ErrNaoEncontrada, 404, "coluna não encontrada"},
		{domain.ErrProibido, 403, "você não gere as colunas deste quadro"},
		{domain.ErrPatchVazio, 422, "informe o nome, a cor, a ordem ou se a coluna fica no quadro"},
		{domain.ErrNome, 422, "informe o nome da coluna (até 80 caracteres)"},
		{domain.ErrCor, 422, "a cor usa o formato #rrggbb"},
		{domain.ErrInicial, 409, "a coluna inicial do fluxo não pode ser retirada"},
		{domain.ErrComRegistro, 409, "esta coluna ainda tem registro; esvazie antes de retirar"},
		{domain.ErrRetirada, 409, "devolva a coluna ao quadro antes de mudar a ordem"},
		{domain.ErrSemEntrada, 422, "escolha ao menos uma coluna de onde o card entra"},
		{domain.ErrPassagem, 422, "a passagem usa só colunas ativas deste quadro; a inicial do fim do fluxo e a aprovação não entram"},
	}
	for _, c := range casos {
		t.Run(c.err.Error(), func(t *testing.T) {
			var api *apperrors.Error
			if !errors.As(MapError(c.err), &api) || api.Status != c.status || api.Message != c.message {
				t.Fatalf("api = %+v", MapError(c.err))
			}
		})
	}
	if MapError(nil) != nil {
		t.Fatal("nil")
	}
	outro := errors.New("falha")
	var api *apperrors.Error
	if !errors.As(MapError(outro), &api) || api.Status != 500 {
		t.Fatalf("desconhecido = %+v", MapError(outro))
	}
}
