package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository é a porta de persistência. Cada método recebe o executor:
// o pool numa leitura, a transação numa escrita, para que a ocorrência, o
// evento no outbox e a auditoria commitem juntos.
type Repository interface {
	Insert(ctx context.Context, db database.DBTX, o Ocorrencia) error
	Update(ctx context.Context, db database.DBTX, o Ocorrencia) error
	InsertAtualizacao(ctx context.Context, db database.DBTX, a Atualizacao) error
	// Travar lê a ocorrência e bloqueia a linha até o fim da transação.
	Travar(ctx context.Context, db database.DBTX, id uuid.UUID) (Ocorrencia, error)
	// Get devolve a ocorrência com a linha do tempo e o nome de quem
	// coordena e de quem escreveu cada atualização.
	Get(ctx context.Context, db database.DBTX, id uuid.UUID) (Ocorrencia, error)
	// List devolve as abertas e depois as mais recentes, até limite.
	List(ctx context.Context, db database.DBTX, limite int) ([]Ocorrencia, error)
	// ParaPagina devolve as abertas e as resolvidas desde a data, com a
	// linha do tempo.
	ParaPagina(ctx context.Context, db database.DBTX, desde time.Time) ([]Ocorrencia, error)
}
