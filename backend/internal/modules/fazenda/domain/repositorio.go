package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// ErrNaoEncontrado é o processo ou o código que não está na base.
var ErrNaoEncontrado = errors.New("fazenda: processo não encontrado")

// Repositorio persiste o processo. Quem chama abre a transação, para o
// processo, o evento e a auditoria commitarem juntos.
type Repositorio interface {
	Inserir(ctx context.Context, db database.DBTX, p Processo) error
	Atualizar(ctx context.Context, db database.DBTX, p Processo) error
	Buscar(ctx context.Context, db database.DBTX, id uuid.UUID) (Processo, error)
	Listar(ctx context.Context, db database.DBTX, titular string) ([]Processo, error)
	Proximo(ctx context.Context, db database.DBTX, ano int) (int, error)
	PorCodigo(ctx context.Context, db database.DBTX, codigo string) (Processo, error)
}
