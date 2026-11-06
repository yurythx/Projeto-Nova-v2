package domain

import (
	"time"

	"github.com/google/uuid"
)

// Anexo é um arquivo ou uma foto ligados à execução da tarefa.
type Anexo struct {
	ID          uuid.UUID  `json:"id"`
	TarefaID    uuid.UUID  `json:"tarefa_id"`
	Nome        string     `json:"nome"`
	ObjectKey   string     `json:"-"`
	ContentType string     `json:"content_type"`
	Tamanho     int64      `json:"tamanho"`
	AutorID     *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome   string     `json:"autor_nome"`
	CreatedAt   time.Time  `json:"created_at"`
}
