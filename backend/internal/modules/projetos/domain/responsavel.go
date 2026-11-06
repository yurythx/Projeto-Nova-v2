package domain

import "github.com/google/uuid"

// Responsavel é um usuário que pode gerir projetos da entidade.
type Responsavel struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}
