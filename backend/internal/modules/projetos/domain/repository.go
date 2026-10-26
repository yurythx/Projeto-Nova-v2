package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// FiltroProjetos define os critérios de busca de projetos.
type FiltroProjetos struct {
	Estado   string
	Busca    string
	Unidades []uuid.UUID
	Restrito bool
}

// Repository define as operações de persistência de projetos e suas tarefas.
type Repository interface {
	Criar(ctx context.Context, db database.DBTX, p *Projeto) error
	Obter(ctx context.Context, db database.DBTX, id uuid.UUID) (*Projeto, error)
	Listar(ctx context.Context, db database.DBTX, filtro FiltroProjetos) ([]Projeto, error)
	Atualizar(ctx context.Context, db database.DBTX, id uuid.UUID, e EntradaProjeto) (*Projeto, error)
	Transicionar(ctx context.Context, db database.DBTX, id uuid.UUID, novoEstado string) (*Projeto, error)

	EstadoDaTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (string, error)
	CriarTarefa(ctx context.Context, db database.DBTX, t *Tarefa) error
	AtualizarTarefa(ctx context.Context, db database.DBTX, id uuid.UUID, titulo, descricao, responsavel, estado string, ordem int) (*Tarefa, error)
	ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Tarefa, error)
	ExcluirTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) error

	CriarVinculo(ctx context.Context, db database.DBTX, v *Vinculo) error
	ListarVinculos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Vinculo, error)
	ExcluirVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) error

	CriarApontamento(ctx context.Context, db database.DBTX, a *Apontamento) error
	ListarApontamentos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Apontamento, error)
	ExcluirApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) error
}
