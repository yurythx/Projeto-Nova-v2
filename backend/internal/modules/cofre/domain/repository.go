package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository persiste o segredo já cifrado.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Selado, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) (Selado, error)
	Insert(ctx context.Context, db database.DBTX, item Selado) error
	Update(ctx context.Context, db database.DBTX, item Selado, f Filtro) error
	Delete(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) error
}
