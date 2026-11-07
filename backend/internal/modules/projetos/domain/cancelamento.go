package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// Cancelamento é a autorização do administrador, com o motivo.
type Cancelamento struct {
	ID            uuid.UUID `json:"id"`
	ProjetoID     uuid.UUID `json:"projeto_id"`
	Motivo        string    `json:"motivo"`
	AprovadorID   uuid.UUID `json:"aprovador_id"`
	AprovadorNome string    `json:"aprovador_nome"`
	CreatedAt     time.Time `json:"created_at"`
}

// PodeCancelar é só o administrador da plataforma.
// Quem gerencia o projeto não cancela.
func PodeCancelar(idt auth.Identity) bool {
	return idt.IsAdmin()
}
