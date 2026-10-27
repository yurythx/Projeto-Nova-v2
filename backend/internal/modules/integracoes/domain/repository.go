package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository é a persistência do módulo.
type Repository interface {
	InsertConexao(ctx context.Context, db database.DBTX, c Conexao) (Conexao, error)
	UpdateConexao(ctx context.Context, db database.DBTX, c Conexao) error
	GetConexao(ctx context.Context, db database.DBTX, id uuid.UUID) (Conexao, error)
	ListConexoes(ctx context.Context, db database.DBTX) ([]Conexao, error)
	DeleteConexao(ctx context.Context, db database.DBTX, id uuid.UUID) error
	SetCredenciais(ctx context.Context, db database.DBTX, id uuid.UUID, cifradas string) error
	// SetToken grava o hash do token (nil revoga).
	SetToken(ctx context.Context, db database.DBTX, id uuid.UUID, hash *string) error
	PorToken(ctx context.Context, db database.DBTX, hash string) (Conexao, error)
	MarcarExtracao(ctx context.Context, db database.DBTX, id uuid.UUID, quando time.Time) error

	GravarItens(ctx context.Context, db database.DBTX, conexao uuid.UUID, itens []Item) error
	Versoes(ctx context.Context, db database.DBTX, conexao uuid.UUID) (map[int]string, error)
	IDs(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string) (map[int]bool, error)
	Itens(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string, de, limite int) ([]Item, int, error)
	Item(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string, id int) (Item, bool, error)
	Contagens(ctx context.Context, db database.DBTX, conexao uuid.UUID) (map[string]int, error)
	Grupos(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]Grupo, error)
	Nomes(ctx context.Context, db database.DBTX, conexao uuid.UUID, tipo string) (map[int]string, error)
	SLAs(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]NivelSLA, error)

	ListarTabela(ctx context.Context, db database.DBTX, conexao uuid.UUID) ([]Linha, error)
	TrocarTabela(ctx context.Context, db database.DBTX, conexao uuid.UUID, linhas []Linha) error

	InserirTarefa(ctx context.Context, db database.DBTX, t Tarefa) error
	GetTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (Tarefa, error)
	ListarTarefas(ctx context.Context, db database.DBTX, conexao uuid.UUID, limite int) ([]Tarefa, error)
	// Pegar marca em andamento a tarefa pendente mais antiga dos tipos
	// pedidos; conexao nil vale para todas, conector diz o modo que serve.
	Pegar(ctx context.Context, db database.DBTX, conexao *uuid.UUID, tipos []string, modo string) (Tarefa, bool, error)
	Progresso(ctx context.Context, db database.DBTX, id uuid.UUID, feitos, total int) (estado string, err error)
	Terminar(ctx context.Context, db database.DBTX, id uuid.UUID, estado string, relatorio []byte, erro string) error
	Cancelar(ctx context.Context, db database.DBTX, id uuid.UUID) error
	// Retomar devolve a pendente a tarefa sem sinal de vida desde antes.
	Retomar(ctx context.Context, db database.DBTX, antes time.Time) (int, error)
}
