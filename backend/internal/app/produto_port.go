package app

import (
	"context"
	"errors"

	"github.com/google/uuid"

	atendimento "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	produtos "github.com/yurythx/projeto-nova-v2/internal/modules/produtos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// produtoPort liga o atendimento ao cadastro de produtos sem os plugins
// se importarem. Com o módulo desligado, a oferta não ganha vínculo novo.
type produtoPort struct {
	repo  produtos.Repository
	ativo func() bool
}

func (p produtoPort) Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (*uuid.UUID, error) {
	if p.ativo == nil || !p.ativo() {
		return nil, atendimento.ErrProdutoIndisponivel
	}
	item, err := p.repo.Get(ctx, db, id)
	if errors.Is(err, produtos.ErrNaoEncontrado) {
		return nil, atendimento.ErrProdutoInexistente
	}
	if err != nil {
		return nil, err
	}
	return item.UnidadeID, nil
}
