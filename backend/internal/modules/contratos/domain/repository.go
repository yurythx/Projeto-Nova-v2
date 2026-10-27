package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Filtro recorta o cadastro pelo escopo.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
}

// Repository persiste fornecedores, contratos e licenças.
type Repository interface {
	ListFornecedores(ctx context.Context, db database.DBTX, f Filtro) ([]Fornecedor, error)
	InsertFornecedor(ctx context.Context, db database.DBTX, item Fornecedor) (Fornecedor, error)
	DeleteFornecedor(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) error

	ListContratos(ctx context.Context, db database.DBTX, f Filtro) ([]Contrato, error)
	InsertContrato(ctx context.Context, db database.DBTX, item Contrato) (Contrato, error)
	DeleteContrato(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) error
	// UpdatePrazo grava o prazo do fornecedor (nil tira) no contrato do escopo.
	UpdatePrazo(ctx context.Context, db database.DBTX, id uuid.UUID, prazo *int, f Filtro) error

	ListLicencas(ctx context.Context, db database.DBTX, f Filtro) ([]Licenca, error)
	InsertLicenca(ctx context.Context, db database.DBTX, item Licenca) (Licenca, error)
	DeleteLicenca(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) error

	Vencendo(ctx context.Context, db database.DBTX) ([]Vencimento, error)
	Destinatarios(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error)
}
