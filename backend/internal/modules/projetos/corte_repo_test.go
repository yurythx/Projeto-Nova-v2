package projetos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

var errCorte = errors.New("corte")

// cortaRepo devolve errCorte na chamada de número na, ou em toda chamada
// quando sempre está ligado. O resto segue para o repositório real.
type cortaRepo struct {
	domain.Repository
	na       int
	chamadas int
	sempre   bool
}

func (c *cortaRepo) erro() error {
	if c.sempre {
		return errCorte
	}
	c.chamadas++
	if c.na > 0 && c.chamadas == c.na {
		return errCorte
	}
	return nil
}

func (c *cortaRepo) cortar(t *testing.T, op func() error) {
	t.Helper()
	for n := 1; n <= 30; n++ {
		c.na = n
		c.chamadas = 0
		err := op()
		if err == nil || !errors.Is(err, errCorte) {
			c.na = 0
			c.chamadas = 0
			return
		}
	}
	c.na = 0
	t.Fatal("o corte não chegou ao fim do caminho")
}

func (c *cortaRepo) Criar(ctx context.Context, db database.DBTX, p *domain.Projeto) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.Criar(ctx, db, p)
}

func (c *cortaRepo) Obter(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Projeto, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.Obter(ctx, db, id)
}

func (c *cortaRepo) Listar(ctx context.Context, db database.DBTX, filtro domain.FiltroProjetos) ([]domain.Projeto, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.Listar(ctx, db, filtro)
}

func (c *cortaRepo) Atualizar(ctx context.Context, db database.DBTX, id uuid.UUID, e domain.EntradaProjeto) (*domain.Projeto, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.Atualizar(ctx, db, id, e)
}

func (c *cortaRepo) Transicionar(ctx context.Context, db database.DBTX, id uuid.UUID, novoEstado string) (*domain.Projeto, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.Transicionar(ctx, db, id, novoEstado)
}

func (c *cortaRepo) PosicaoUnidade(ctx context.Context, db database.DBTX, unidade uuid.UUID) (*uuid.UUID, []uuid.UUID, error) {
	if err := c.erro(); err != nil {
		return nil, nil, err
	}
	return c.Repository.PosicaoUnidade(ctx, db, unidade)
}

func (c *cortaRepo) Aprovadores(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.Aprovadores(ctx, db, unidade)
}

func (c *cortaRepo) SalvarReabertura(ctx context.Context, db database.DBTX, r *domain.Reabertura) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.SalvarReabertura(ctx, db, r)
}

func (c *cortaRepo) UltimaReabertura(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Reabertura, bool, error) {
	if err := c.erro(); err != nil {
		return domain.Reabertura{}, false, err
	}
	return c.Repository.UltimaReabertura(ctx, db, projetoID)
}

func (c *cortaRepo) SalvarCancelamento(ctx context.Context, db database.DBTX, reg *domain.Cancelamento) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.SalvarCancelamento(ctx, db, reg)
}

func (c *cortaRepo) UltimoCancelamento(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Cancelamento, bool, error) {
	if err := c.erro(); err != nil {
		return domain.Cancelamento{}, false, err
	}
	return c.Repository.UltimoCancelamento(ctx, db, projetoID)
}

func (c *cortaRepo) ListarHistorico(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Marco, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarHistorico(ctx, db, projetoID)
}

func (c *cortaRepo) EstadoDaTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (string, error) {
	if err := c.erro(); err != nil {
		return "", err
	}
	return c.Repository.EstadoDaTarefa(ctx, db, id)
}

func (c *cortaRepo) CriarTarefa(ctx context.Context, db database.DBTX, t *domain.Tarefa) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.CriarTarefa(ctx, db, t)
}

func (c *cortaRepo) ObterTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Tarefa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ObterTarefa(ctx, db, id)
}

func (c *cortaRepo) AtualizarTarefa(ctx context.Context, db database.DBTX, id uuid.UUID, titulo, descricao, relato string, responsavelID *uuid.UUID, responsavel, estado string, ordem int, dataLimite *time.Time, predecessoraID *uuid.UUID) (*domain.Tarefa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.AtualizarTarefa(ctx, db, id, titulo, descricao, relato, responsavelID, responsavel, estado, ordem, dataLimite, predecessoraID)
}

func (c *cortaRepo) ListarAnexos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Anexo, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarAnexos(ctx, db, tarefaID)
}

func (c *cortaRepo) ObterAnexo(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Anexo, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ObterAnexo(ctx, db, id)
}

func (c *cortaRepo) CriarAnexo(ctx context.Context, db database.DBTX, a *domain.Anexo) (*domain.Anexo, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.CriarAnexo(ctx, db, a)
}

func (c *cortaRepo) RegistrarMovimento(ctx context.Context, db database.DBTX, m *domain.Movimento) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.RegistrarMovimento(ctx, db, m)
}

func (c *cortaRepo) ListarMovimentos(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Movimento, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarMovimentos(ctx, db, tarefaID)
}

func (c *cortaRepo) CriarAtualizacao(ctx context.Context, db database.DBTX, a *domain.Atualizacao) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.CriarAtualizacao(ctx, db, a)
}

func (c *cortaRepo) ListarAtualizacoes(ctx context.Context, db database.DBTX, tarefaID uuid.UUID) ([]domain.Atualizacao, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarAtualizacoes(ctx, db, tarefaID)
}

func (c *cortaRepo) ListarPrazos(ctx context.Context, db database.DBTX, mes string, restrito bool, unidades []uuid.UUID) ([]domain.PrazoAgenda, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarPrazos(ctx, db, mes, restrito, unidades)
}

func (c *cortaRepo) PrazosVencidos(ctx context.Context, db database.DBTX, agora time.Time) ([]domain.PrazoTarefa, []domain.PrazoProjeto, error) {
	if err := c.erro(); err != nil {
		return nil, nil, err
	}
	return c.Repository.PrazosVencidos(ctx, db, agora)
}

func (c *cortaRepo) NomeDaPessoa(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error) {
	if err := c.erro(); err != nil {
		return "", err
	}
	return c.Repository.NomeDaPessoa(ctx, db, pessoa, unidade)
}

func (c *cortaRepo) ListarResponsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Responsavel, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarResponsaveis(ctx, db, unidade)
}

func (c *cortaRepo) NomeDeResponsavel(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error) {
	if err := c.erro(); err != nil {
		return "", err
	}
	return c.Repository.NomeDeResponsavel(ctx, db, pessoa, unidade)
}

func (c *cortaRepo) ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarTarefas(ctx, db, projetoID)
}

func (c *cortaRepo) IncluirParticipante(ctx context.Context, db database.DBTX, projetoID uuid.UUID, pessoa domain.Pessoa) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.IncluirParticipante(ctx, db, projetoID, pessoa)
}

func (c *cortaRepo) ExcluirParticipante(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.ExcluirParticipante(ctx, db, projetoID, pessoa)
}

func (c *cortaRepo) ListarParticipantes(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Pessoa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarParticipantes(ctx, db, projetoID)
}

func (c *cortaRepo) EhParticipante(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) (bool, error) {
	if err := c.erro(); err != nil {
		return false, err
	}
	return c.Repository.EhParticipante(ctx, db, projetoID, pessoa)
}

func (c *cortaRepo) SubstituirPessoasDaTarefa(ctx context.Context, db database.DBTX, tarefaID uuid.UUID, pessoas []domain.Pessoa) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.SubstituirPessoasDaTarefa(ctx, db, tarefaID, pessoas)
}

func (c *cortaRepo) PessoasPorTarefa(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID][]domain.Pessoa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.PessoasPorTarefa(ctx, db, ids)
}

func (c *cortaRepo) RetirarPessoaDasTarefas(ctx context.Context, db database.DBTX, projetoID, pessoa uuid.UUID) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.RetirarPessoaDasTarefas(ctx, db, projetoID, pessoa)
}

func (c *cortaRepo) ListarMinhas(ctx context.Context, db database.DBTX, pessoa uuid.UUID, restrito bool, unidades []uuid.UUID) ([]domain.TarefaMinha, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarMinhas(ctx, db, pessoa, restrito, unidades)
}

func (c *cortaRepo) ListarCarga(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID) ([]domain.CargaPessoa, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarCarga(ctx, db, restrito, unidades)
}

func (c *cortaRepo) ExcluirTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.ExcluirTarefa(ctx, db, id)
}

func (c *cortaRepo) CriarVinculo(ctx context.Context, db database.DBTX, v *domain.Vinculo) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.CriarVinculo(ctx, db, v)
}

func (c *cortaRepo) ListarVinculos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Vinculo, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarVinculos(ctx, db, projetoID)
}

func (c *cortaRepo) ProjetoDoVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error) {
	if err := c.erro(); err != nil {
		return uuid.Nil, false, err
	}
	return c.Repository.ProjetoDoVinculo(ctx, db, id)
}

func (c *cortaRepo) ExcluirVinculo(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.ExcluirVinculo(ctx, db, id)
}

func (c *cortaRepo) CriarApontamento(ctx context.Context, db database.DBTX, a *domain.Apontamento) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.CriarApontamento(ctx, db, a)
}

func (c *cortaRepo) ListarApontamentos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Apontamento, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.ListarApontamentos(ctx, db, projetoID)
}

func (c *cortaRepo) ProjetoDoApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) (uuid.UUID, bool, error) {
	if err := c.erro(); err != nil {
		return uuid.Nil, false, err
	}
	return c.Repository.ProjetoDoApontamento(ctx, db, id)
}

func (c *cortaRepo) ExcluirApontamento(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if err := c.erro(); err != nil {
		return err
	}
	return c.Repository.ExcluirApontamento(ctx, db, id)
}

func (c *cortaRepo) ObterCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Carta, bool, error) {
	if err := c.erro(); err != nil {
		return domain.Carta{}, false, err
	}
	return c.Repository.ObterCarta(ctx, db, projetoID)
}

func (c *cortaRepo) SalvarCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID, e domain.EntradaCarta) (domain.Carta, error) {
	if err := c.erro(); err != nil {
		return domain.Carta{}, err
	}
	return c.Repository.SalvarCarta(ctx, db, projetoID, e)
}

func (c *cortaRepo) AtualizarPlano(ctx context.Context, db database.DBTX, id uuid.UUID, p domain.Plano5W2H) (*domain.Projeto, error) {
	if err := c.erro(); err != nil {
		return nil, err
	}
	return c.Repository.AtualizarPlano(ctx, db, id, p)
}

func (c *cortaRepo) ContarEtapasHow(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (int, int, error) {
	if err := c.erro(); err != nil {
		return 0, 0, err
	}
	return c.Repository.ContarEtapasHow(ctx, db, projetoID)
}

func (c *cortaRepo) SomarCusto(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (float64, error) {
	if err := c.erro(); err != nil {
		return 0, err
	}
	return c.Repository.SomarCusto(ctx, db, projetoID)
}
