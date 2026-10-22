// Package search define o contrato da Busca Global. Cada plugin que tem
// conteúdo pesquisável implementa Provider; o plugin de Busca só consulta
// os providers de módulos ATIVOS (degradação graciosa) — nenhum plugin
// importa outro para isso, o Kernel faz a ponte.
package search

import (
	"context"
	"time"

	"github.com/yurythx/sismarq/internal/platform/auth"
)

// Result é um item da busca global.
type Result struct {
	Module    string     `json:"module"`
	Type      string     `json:"type"`
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Snippet   string     `json:"snippet,omitempty"`
	URL       string     `json:"url"`
	Score     float64    `json:"score"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// Provider pesquisa o conteúdo de um módulo respeitando as permissões e a
// visibilidade de identity (nunca devolve o que o usuário não poderia
// abrir).
type Provider interface {
	Module() string
	Search(ctx context.Context, identity auth.Identity, query string, limit int) ([]Result, error)
}

// Document é um item publicado no índice de busca (Typesense, ADR 031).
type Document struct {
	Type      string
	ID        string
	Title     string
	Body      string
	URL       string
	UpdatedAt *time.Time
}

// Indexer é o Provider de um módulo cujo conteúdo vai para o índice. O
// índice não conhece permissões: só implementa Indexer o módulo cujo
// conteúdo é PÚBLICO (visível a qualquer um, com ou sem login). Módulos com
// ACL, sigilo ou público-alvo continuam respondendo só pelo Search.
type Indexer interface {
	Provider
	// Documents entrega todo o conteúdo indexável, um por vez, a emit.
	Documents(ctx context.Context, emit func(Document) error) error
}
