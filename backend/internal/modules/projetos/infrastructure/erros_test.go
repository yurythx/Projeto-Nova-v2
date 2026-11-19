package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// banco programável: as primeiras chamadas passam e a partir de na falham.
// na == 0 nunca falha. A consulta que passa devolve zero linhas.
type banco struct {
	na int
	i  int
}

func (b *banco) estoura() error {
	b.i++
	if b.na > 0 && b.i >= b.na {
		return dbtest.ErrInjected
	}
	return nil
}

func (b *banco) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if err := b.estoura(); err != nil {
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (b *banco) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if err := b.estoura(); err != nil {
		return nil, err
	}
	return dbtest.NoRows(), nil
}

func (b *banco) QueryRow(context.Context, string, ...any) pgx.Row {
	if err := b.estoura(); err != nil {
		return linha{err}
	}
	return linha{}
}

type linha struct{ err error }

func (l linha) Scan(...any) error { return l.err }

type soLinha struct{ row pgx.Row }

func (soLinha) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (soLinha) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return dbtest.NoRows(), nil
}
func (s soLinha) QueryRow(context.Context, string, ...any) pgx.Row { return s.row }

func TestErrosDoBanco(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	id := uuid.New()
	agora := time.Now().UTC()
	unidade := id
	if !errors.Is(mapDB(&pgconn.PgError{Code: "23503"}), domain.ErrTarefaNaoEncontrada) {
		t.Fatal("chave estrangeira")
	}

	projeto := &domain.Projeto{ID: id, Titulo: "t"}
	tarefa := &domain.Tarefa{ID: id, ProjetoID: id, Titulo: "t"}
	anexo := &domain.Anexo{ID: id, TarefaID: id}
	movimento := &domain.Movimento{ID: id, TarefaID: id}
	atualizacao := &domain.Atualizacao{ID: id, TarefaID: id}
	vinculo := &domain.Vinculo{ID: id, ProjetoID: id, AlvoID: id}
	apontamento := &domain.Apontamento{ID: id, ProjetoID: id}
	reabertura := &domain.Reabertura{ID: id, ProjetoID: id}
	cancelamento := &domain.Cancelamento{ID: id, ProjetoID: id}
	pessoa := domain.Pessoa{ID: id, Nome: "Ana"}
	filtro := domain.FiltroProjetos{
		Estado: "planejado", Busca: "ana", ProdutoID: &id, AlvoID: &id, AlvoTipo: "ativo",
		Restrito: true, Unidades: []uuid.UUID{id},
	}
	filtroSemTipo := filtro
	filtroSemTipo.AlvoTipo = ""

	for _, db := range []database.DBTX{dbtest.Fail{}, dbtest.ScanFail{}, dbtest.RowsErr{}} {
		_ = r.Criar(ctx, db, projeto)
		_, _ = r.Obter(ctx, db, id)
		_, _ = r.Listar(ctx, db, filtro)
		_, _ = r.Listar(ctx, db, filtroSemTipo)
		_, _ = r.Atualizar(ctx, db, id, domain.EntradaProjeto{Titulo: "t"})
		_, _ = r.Transicionar(ctx, db, id, "em_andamento")
		_, _, _ = r.PosicaoUnidade(ctx, db, unidade)
		_, _ = r.Aprovadores(ctx, db, &unidade)
		_, _ = r.Aprovadores(ctx, db, nil)
		_ = r.SalvarReabertura(ctx, db, reabertura)
		_, _, _ = r.UltimaReabertura(ctx, db, id)
		_ = r.SalvarCancelamento(ctx, db, cancelamento)
		_, _, _ = r.UltimoCancelamento(ctx, db, id)
		_, _ = r.ListarHistorico(ctx, db, id)
		_, _ = r.EstadoDaTarefa(ctx, db, id)
		_ = r.CriarTarefa(ctx, db, tarefa)
		_, _ = r.ObterTarefa(ctx, db, id)
		_, _ = r.AtualizarTarefa(ctx, db, id, "t", "", "", nil, "", "a_fazer", 1, &agora, nil)
		_, _ = r.ListarAnexos(ctx, db, id)
		_, _ = r.ObterAnexo(ctx, db, id)
		_, _ = r.CriarAnexo(ctx, db, anexo)
		_ = r.RegistrarMovimento(ctx, db, movimento)
		_, _ = r.ListarMovimentos(ctx, db, id)
		_ = r.CriarAtualizacao(ctx, db, atualizacao)
		_, _ = r.ListarAtualizacoes(ctx, db, id)
		_, _ = r.ListarPrazos(ctx, db, "2026-11", true, []uuid.UUID{id})
		_, _ = r.ListarPrazos(ctx, db, "2026-11", false, nil)
		_, _, _ = r.PrazosVencidos(ctx, db, agora)
		_, _ = r.NomeDaPessoa(ctx, db, id, &unidade)
		_, _ = r.NomeDaPessoa(ctx, db, id, nil)
		_, _ = r.ListarResponsaveis(ctx, db, &unidade)
		_, _ = r.NomeDeResponsavel(ctx, db, id, &unidade)
		_, _ = r.ListarTarefas(ctx, db, id)
		_ = r.IncluirParticipante(ctx, db, id, pessoa)
		_ = r.ExcluirParticipante(ctx, db, id, id)
		_, _ = r.ListarParticipantes(ctx, db, id)
		_, _ = r.EhParticipante(ctx, db, id, id)
		_ = r.SubstituirPessoasDaTarefa(ctx, db, id, []domain.Pessoa{pessoa})
		_, _ = r.PessoasPorTarefa(ctx, db, []uuid.UUID{id})
		_, _ = r.PessoasPorTarefa(ctx, db, nil)
		_ = r.RetirarPessoaDasTarefas(ctx, db, id, id)
		_, _ = r.ListarMinhas(ctx, db, id, true, []uuid.UUID{id})
		_, _ = r.ListarMinhas(ctx, db, id, false, nil)
		_, _ = r.ListarCarga(ctx, db, true, []uuid.UUID{id})
		_, _ = r.ListarCarga(ctx, db, false, nil)
		_ = r.ExcluirTarefa(ctx, db, id)
		_ = r.CriarVinculo(ctx, db, vinculo)
		_, _ = r.ListarVinculos(ctx, db, id)
		_, _, _ = r.ProjetoDoVinculo(ctx, db, id)
		_ = r.ExcluirVinculo(ctx, db, id)
		_ = r.CriarApontamento(ctx, db, apontamento)
		_, _ = r.ListarApontamentos(ctx, db, id)
		_, _, _ = r.ProjetoDoApontamento(ctx, db, id)
		_ = r.ExcluirApontamento(ctx, db, id)
		_, _, _ = r.ObterCarta(ctx, db, id)
		_, _ = r.SalvarCarta(ctx, db, id, domain.EntradaCarta{})
		_, _ = r.AtualizarPlano(ctx, db, id, domain.Plano5W2H{Titulo: "t"})
		_, _, _ = r.ContarEtapasHow(ctx, db, id)
		_, _ = r.SomarCusto(ctx, db, id)
	}

	vazia := soLinha{row: linha{pgx.ErrNoRows}}
	_, _, _ = r.PosicaoUnidade(ctx, vazia, unidade)
	_, _, _ = r.UltimaReabertura(ctx, vazia, id)
	_, _, _ = r.UltimoCancelamento(ctx, vazia, id)
	_, _ = r.NomeDaPessoa(ctx, vazia, id, nil)
	_, _ = r.NomeDeResponsavel(ctx, vazia, id, nil)
	_, _, _ = r.ProjetoDoVinculo(ctx, vazia, id)
	_, _, _ = r.ProjetoDoApontamento(ctx, vazia, id)
	_, _, _ = r.ObterCarta(ctx, vazia, id)
	_, _ = r.EstadoDaTarefa(ctx, vazia, id)
	_, _ = r.EhParticipante(ctx, vazia, id, id)

	_, _, _ = r.PrazosVencidos(ctx, &banco{na: 2}, agora)
	_ = r.SubstituirPessoasDaTarefa(ctx, &banco{na: 2}, id, []domain.Pessoa{pessoa})
	_ = r.SubstituirPessoasDaTarefa(ctx, &banco{na: 3}, id, []domain.Pessoa{pessoa})
	_ = r.SubstituirPessoasDaTarefa(ctx, &banco{na: 2}, id, nil)
	_ = r.RetirarPessoaDasTarefas(ctx, &banco{na: 2}, id, id)
}
