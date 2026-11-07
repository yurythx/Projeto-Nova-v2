package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Marco é uma linha do histórico do projeto, lida da trilha de auditoria.
type Marco struct {
	ID        uuid.UUID       `json:"id"`
	Acao      string          `json:"acao"`
	AutorNome string          `json:"autor_nome,omitempty"`
	Quando    time.Time       `json:"quando"`
	Texto     string          `json:"texto"`
	Antes     json.RawMessage `json:"-"`
	Depois    json.RawMessage `json:"-"`
}
