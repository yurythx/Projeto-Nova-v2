package domain

import (
	"time"

	"github.com/google/uuid"
)

// Movimento é o registro de uma passagem de coluna ou de uma troca de responsável.
type Movimento struct {
	ID                    uuid.UUID  `json:"id"`
	TarefaID              uuid.UUID  `json:"tarefa_id"`
	EstadoAntes           string     `json:"estado_antes"`
	EstadoDepois          string     `json:"estado_depois"`
	ResponsavelAntesID    *uuid.UUID `json:"responsavel_antes_id,omitempty"`
	ResponsavelAntesNome  string     `json:"responsavel_antes_nome"`
	ResponsavelDepoisID   *uuid.UUID `json:"responsavel_depois_id,omitempty"`
	ResponsavelDepoisNome string     `json:"responsavel_depois_nome"`
	AutorID               *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome             string     `json:"autor_nome"`
	CreatedAt             time.Time  `json:"created_at"`
}
