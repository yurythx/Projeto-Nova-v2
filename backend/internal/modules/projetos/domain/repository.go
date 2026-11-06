package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// FiltroProjetos define os critérios de busca de projetos.
type FiltroProjetos struct {
	Estado    string
	Busca     string
	ProdutoID *uuid.UUID
	AlvoTipo  string
	AlvoID    *uuid.UUID
	Unidades  []uuid.UUID
	Restrito  bool
}

// Repository define as operações de persistência de projetos e suas tarefas.
type Repository interface {
	Criar(ctx context.Context, db database.DBTX, p *Projeto) error
	Obter(ctx context.Context, db database.DBTX, id uuid.UUID) (*Projeto, error)
	Listar(ctx context.Context, db database.DBTX, filtro FiltroProjetos) ([]Projeto, error)
	Atualizar(ctx context.Context, db database.DBTX, id uuid.UUID, e EntradaProjeto) (*Projeto, error)
	Transicionar(ctx context.Context, db database.DBTX, id uuid.UUID, novoEstado string) (*Projeto, error)
	PosicaoUnidade(ctx context.Context, db database.DBTX, unidade uuid.UUID) (entidade *uuid.UUID, acima []uuid.UUID, err error)
	Aprovadores(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error)
	SalvarReabertura(ctx context.Context, db database.DBTX, r *Reabertura) error
	UltimaReabertura(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (Reabertura, bool, error)

	EstadoDaTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (string, error)
	CriarTarefa(ctx context.Context, db database.DBTX, t *Tarefa) error
	ObterTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (*Tarefa, error)
	AtualizarTarefa(ctx context.Context, db database.DBTX, id uuid.UUID, titulo, descricao, relato string, responsavelID *uuid.UUID, responsavel, estado string, ordem int, dataLimite *time.Time, predecessoraID *uuid.UUID) (*Tarefa, error)
	ListarAnexos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]Anexo, error)
	ObterAnexo(ctx context.Context, db database.DBTX, id uuid.UUID) (*Anexo, error)
	CriarAnexo(ctx context.Context, db database.DBTX, a *Anexo) (*Anexo, error)
	RegistrarMovimento(ctx context.Context, db database.DBTX, m *Movimento) error
	ListarMovimentos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]Movimento, error)
	PrazosVencidos(ctx context.Context, db database.DBTX, agora time.Time) ([]PrazoTarefa, []PrazoProjeto, error)
	NomeDaPessoa(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error)
	ListarResponsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Responsavel, error)
	NomeDeResponsavel(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error)
	ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Tarefa, error)
	ListarMinhas(ctx context.Context, db database.DBTX, pessoa uuid.UUID, restrito bool, unidades []uuid.UUID) ([]TarefaMinha, error)
	ListarCarga(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID) ([]CargaPessoa, error)
	ExcluirTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) error

	CriarVinculo(ctx context.Context, db database.DBTX, v *Vinculo) error
	ListarVinculos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Vinculo, error)
	ProjetoDoVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error)
	ExcluirVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) error

	CriarApontamento(ctx context.Context, db database.DBTX, a *Apontamento) error
	ListarApontamentos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]Apontamento, error)
	ProjetoDoApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error)
	ExcluirApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) error

	ObterCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (Carta, bool, error)
	SalvarCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID, c EntradaCarta) (Carta, error)
	AtualizarPlano(ctx context.Context, db database.DBTX, id uuid.UUID, p Plano5W2H) (*Projeto, error)
	ContarEtapasHow(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (total, concluidas int, err error)
	SomarCusto(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (float64, error)
}
