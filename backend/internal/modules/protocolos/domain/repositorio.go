package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository é a porta de persistência. O método recebe o executor:
// o pool na leitura, a transação na escrita.
type Repository interface {
	Unidade(ctx context.Context, db database.DBTX, id uuid.UUID) (sigla, nome string, err error)
	InserirFormulario(ctx context.Context, db database.DBTX, f Formulario) error
	AtualizarFormulario(ctx context.Context, db database.DBTX, f Formulario) error
	BuscarFormulario(ctx context.Context, db database.DBTX, id uuid.UUID) (Formulario, error)
	ListarFormularios(ctx context.Context, db database.DBTX, v Visao) ([]Formulario, error)
	InserirServico(ctx context.Context, db database.DBTX, s Servico) error
	AtualizarServico(ctx context.Context, db database.DBTX, s Servico) error
	BuscarServico(ctx context.Context, db database.DBTX, id uuid.UUID) (Servico, error)
	ListarServicos(ctx context.Context, db database.DBTX, soAtivos bool, unidade *uuid.UUID) ([]Servico, error)
	ListarServicosDaGestao(ctx context.Context, db database.DBTX, v Visao) ([]Servico, error)
	InserirPedido(ctx context.Context, db database.DBTX, p Pedido) error
	AtualizarPedido(ctx context.Context, db database.DBTX, p Pedido) error
	BuscarPedido(ctx context.Context, db database.DBTX, id uuid.UUID) (Pedido, error)
	BuscarPorNumero(ctx context.Context, db database.DBTX, numero string) (Pedido, error)
	ListarPedidos(ctx context.Context, db database.DBTX, v Visao) ([]Pedido, error)
	NumeroExiste(ctx context.Context, db database.DBTX, numero string) (bool, error)
	CPFEmAberto(ctx context.Context, db database.DBTX, servico uuid.UUID, cpf string) (int, error)
	InserirHistorico(ctx context.Context, db database.DBTX, h Historico) error
	ListarHistorico(ctx context.Context, db database.DBTX, pedido uuid.UUID) ([]Historico, error)
	InserirResponsavel(ctx context.Context, db database.DBTX, r ResponsavelTecnico) error
	ListarResponsaveis(ctx context.Context, db database.DBTX) ([]ResponsavelTecnico, error)
	ResponsavelAtivo(ctx context.Context, db database.DBTX, cpf string) (bool, error)
	InserirAgendamento(ctx context.Context, db database.DBTX, a Agendamento) error
	AtualizarAgendamento(ctx context.Context, db database.DBTX, a Agendamento) error
	BuscarAgendamento(ctx context.Context, db database.DBTX, pedido uuid.UUID) (Agendamento, error)
	ContarAgendamentos(ctx context.Context, db database.DBTX, servico uuid.UUID, dia string) (int, error)
	ListarHoras(ctx context.Context, db database.DBTX, servico uuid.UUID, dia string) ([]string, error)
	LerFaltas(ctx context.Context, db database.DBTX, cpf string) (int, time.Time, error)
	SomarFalta(ctx context.Context, db database.DBTX, cpf string, quando time.Time) error
	// Pessoa devolve o nome do usuário ativo.
	Pessoa(ctx context.Context, db database.DBTX, id uuid.UUID) (string, error)
	// Pessoas lista os usuários ativos com escopo na unidade.
	Pessoas(ctx context.Context, db database.DBTX, unidade uuid.UUID) ([]Pessoa, error)
	ListarFeriados(ctx context.Context, db database.DBTX) ([]Feriado, error)
	SalvarFeriado(ctx context.Context, db database.DBTX, f Feriado) error
	RemoverFeriado(ctx context.Context, db database.DBTX, data string) error
	// ComplementacoesVencidas são os pedidos em complementação com o prazo já passado.
	ComplementacoesVencidas(ctx context.Context, db database.DBTX, agora time.Time) ([]Pedido, error)
	// PrazosVencendo são os pedidos abertos com prazo, ou prazo de complementação, até "ate".
	PrazosVencendo(ctx context.Context, db database.DBTX, agora, ate time.Time) ([]Pedido, error)
	MarcarAvisoComplemento(ctx context.Context, db database.DBTX, id uuid.UUID) error
}
