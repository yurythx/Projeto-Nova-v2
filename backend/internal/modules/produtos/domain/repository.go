package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/platform/database"
)

// Filtro restringe a listagem ao escopo de produtos:manage (ADR 013).
// Restrito falso vê a plataforma toda, inclusive o institucional.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
}

// Repository é a porta de persistência. O executor chega de fora: leitura
// no pool, escrita na transação que também grava outbox e auditoria.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Produto, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID) (Produto, error)
	Insert(ctx context.Context, db database.DBTX, p Produto) (Produto, error)
	Update(ctx context.Context, db database.DBTX, p Produto) (Produto, error)
}
