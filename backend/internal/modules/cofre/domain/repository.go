package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository persiste o segredo já cifrado.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Selado, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) (Selado, error)
	Insert(ctx context.Context, db database.DBTX, item Selado) error
	Update(ctx context.Context, db database.DBTX, item Selado, f Filtro) error
	Delete(ctx context.Context, db database.DBTX, id uuid.UUID, f Filtro) error
	// ListarAdministracao devolve, por painel, as máquinas que os dois lados
	// do vínculo estão no filtro. Fora do alcance não aparece.
	ListarAdministracao(ctx context.Context, db database.DBTX, f Filtro) (map[uuid.UUID][]uuid.UUID, error)
	// SubstituirAdministracao troca as máquinas visíveis. As que estão fora
	// do filtro permanecem.
	SubstituirAdministracao(ctx context.Context, db database.DBTX, painel uuid.UUID, maquinas []uuid.UUID, f Filtro) error
	// Travar segura a linha do segredo até o fim da transação.
	Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error
	// RetiradaAberta devolve a reserva que ainda não foi encerrada, mesmo vencida.
	RetiradaAberta(ctx context.Context, db database.DBTX, segredo uuid.UUID) (RetiradaGravada, error)
	InserirRetirada(ctx context.Context, db database.DBTX, item RetiradaGravada) error
	RenovarRetirada(ctx context.Context, db database.DBTX, id uuid.UUID, ate time.Time, nome string) error
	EncerrarRetirada(ctx context.Context, db database.DBTX, id uuid.UUID, quando time.Time) error
	// ListarRetiradas devolve só as reservas ainda no prazo e no alcance.
	ListarRetiradas(ctx context.Context, db database.DBTX, f Filtro, agora time.Time) (map[uuid.UUID]Retirada, error)
	// ListarParaRotacionar devolve segredos marcados, no prazo e sem aviso pendente.
	ListarParaRotacionar(ctx context.Context, db database.DBTX, agora time.Time) ([]Selado, error)

	ListarPastas(ctx context.Context, db database.DBTX, f Filtro) ([]Pasta, error)
	ObterPasta(ctx context.Context, db database.DBTX, id uuid.UUID) (Pasta, error)
	InserirPasta(ctx context.Context, db database.DBTX, p Pasta) error
	ExcluirPasta(ctx context.Context, db database.DBTX, id uuid.UUID) error
	// ContarNaPasta conta segredos ainda dentro da pasta, sem filtrar quem vê.
	ContarNaPasta(ctx context.Context, db database.DBTX, pasta uuid.UUID) (int, error)
	SubstituirMembros(ctx context.Context, db database.DBTX, pasta uuid.UUID, membros []Membro) error
	Membro(ctx context.Context, db database.DBTX, pasta, usuario uuid.UUID) (Membro, error)
	Pessoas(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Pessoa, error)
	UsuariosExistem(ctx context.Context, db database.DBTX, ids []uuid.UUID) (bool, error)
}
