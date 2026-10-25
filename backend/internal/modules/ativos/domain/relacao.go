package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Tipos de relação entre ativos. "depende" não pode formar ciclo.
var TiposRelacao = []string{"depende", "composto", "conectado"}

// Relacao liga dois ativos. A origem é quem aponta.
type Relacao struct {
	ID          uuid.UUID `json:"id"`
	OrigemID    uuid.UUID `json:"origem_id"`
	DestinoID   uuid.UUID `json:"destino_id"`
	DestinoNome string    `json:"destino_nome"`
	Tipo        string    `json:"tipo"`
	CriadaEm    time.Time `json:"criada_em"`
}

// RelacaoInvalida é um par que o domínio não aceita.
type RelacaoInvalida struct{ Msg string }

func (e RelacaoInvalida) Error() string { return e.Msg }

// ValidarRelacao confere o par antes de olhar o grafo.
func ValidarRelacao(origem, destino uuid.UUID, tipo string) error {
	if origem == uuid.Nil || destino == uuid.Nil || origem == destino {
		return RelacaoInvalida{"escolha dois ativos diferentes"}
	}
	if !slices.Contains(TiposRelacao, tipo) {
		return RelacaoInvalida{"a relação é depende, composto ou conectado"}
	}
	return nil
}
