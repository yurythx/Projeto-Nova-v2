package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Filtro define os critérios de busca de riscos.
type Filtro struct {
	Unidades  []uuid.UUID
	Categoria string
	Status    string
}

// Repository persiste e consulta riscos de TI.
type Repository interface {
	ListRiscos(ctx context.Context, db database.DBTX, f Filtro) ([]Risco, error)
	GetRisco(ctx context.Context, db database.DBTX, id uuid.UUID) (Risco, error)
	InsertRisco(ctx context.Context, db database.DBTX, r Risco) (Risco, error)
	UpdateRisco(ctx context.Context, db database.DBTX, r Risco) (Risco, error)
	DeleteRisco(ctx context.Context, db database.DBTX, id uuid.UUID) error
}

