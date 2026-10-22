package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	atendimento "github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	produtos "github.com/yurythx/sismarq/internal/modules/produtos/domain"
	"github.com/yurythx/sismarq/internal/platform/database"
)

type stubProdutos struct {
	item produtos.Produto
	err  error
}

func (s stubProdutos) List(context.Context, database.DBTX, produtos.Filtro) ([]produtos.Produto, error) {
	return nil, s.err
}
func (s stubProdutos) Get(context.Context, database.DBTX, uuid.UUID) (produtos.Produto, error) {
	return s.item, s.err
}
func (s stubProdutos) Insert(context.Context, database.DBTX, produtos.Produto) (produtos.Produto, error) {
	return produtos.Produto{}, s.err
}
func (s stubProdutos) Update(context.Context, database.DBTX, produtos.Produto) (produtos.Produto, error) {
	return produtos.Produto{}, s.err
}

func TestProdutoPort(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	if _, err := (produtoPort{}).Consultar(ctx, nil, id); !errors.Is(err, atendimento.ErrProdutoIndisponivel) {
		t.Fatalf("sem função de ativação: %v", err)
	}
	off := produtoPort{ativo: func() bool { return false }, repo: stubProdutos{}}
	if _, err := off.Consultar(ctx, nil, id); !errors.Is(err, atendimento.ErrProdutoIndisponivel) {
		t.Fatalf("módulo desligado: %v", err)
	}
	sumido := produtoPort{ativo: func() bool { return true }, repo: stubProdutos{err: produtos.ErrNaoEncontrado}}
	if _, err := sumido.Consultar(ctx, nil, id); !errors.Is(err, atendimento.ErrProdutoInexistente) {
		t.Fatalf("inexistente: %v", err)
	}
	quebrado := produtoPort{ativo: func() bool { return true }, repo: stubProdutos{err: errors.New("banco")}}
	if _, err := quebrado.Consultar(ctx, nil, id); err == nil || errors.Is(err, atendimento.ErrProdutoInexistente) {
		t.Fatalf("erro do repositório: %v", err)
	}
	u := uuid.New()
	ok := produtoPort{ativo: func() bool { return true }, repo: stubProdutos{item: produtos.Produto{UnidadeID: &u}}}
	got, err := ok.Consultar(ctx, nil, id)
	if err != nil || got == nil || *got != u {
		t.Fatalf("unidade do produto: %v %v", got, err)
	}
}
