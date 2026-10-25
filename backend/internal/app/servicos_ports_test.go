package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	atendimento "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	ativos "github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	mudancas "github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	problemas "github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	produtos "github.com/yurythx/projeto-nova-v2/internal/modules/produtos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

var errBanco = errors.New("banco")

type stubAtivos struct {
	ativos.Repository
	item ativos.Ativo
	err  error
}

func (s stubAtivos) Get(context.Context, database.DBTX, uuid.UUID) (ativos.Ativo, error) {
	return s.item, s.err
}

type stubAtendimento struct {
	atendimento.Repository
	sol    atendimento.Solicitacao
	err    error
	grupos []atendimento.Recorrencia
}

func (s stubAtendimento) GetSolicitacao(context.Context, database.DBTX, uuid.UUID) (atendimento.Solicitacao, error) {
	return s.sol, s.err
}
func (s stubAtendimento) IncidentesRecorrentes(context.Context, database.DBTX, bool, []uuid.UUID, time.Time, int) ([]atendimento.Recorrencia, error) {
	return s.grupos, s.err
}

type stubProblemas struct {
	problemas.Repository
	p   problemas.Problema
	err error
}

func (s stubProblemas) Get(context.Context, database.DBTX, uuid.UUID) (problemas.Problema, error) {
	return s.p, s.err
}

func ligadoSe(ok bool) func(string) bool { return func(string) bool { return ok } }

func TestPortasDosServicos(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	u := uuid.New()

	// Produto visto por ativos, problemas e mudanças.
	porta := portaProdutoAtivos(stubProdutos{item: produtos.Produto{UnidadeID: &u}}, ligadoSe(true))
	if got, err := porta.Consultar(ctx, nil, id); err != nil || *got != u {
		t.Fatalf("produto: %v %v", got, err)
	}
	if _, err := portaProdutoAtivos(stubProdutos{}, ligadoSe(false)).Consultar(ctx, nil, id); !errors.Is(err, ativos.ErrProdutoIndisponivel) {
		t.Fatal(err)
	}
	if _, err := portaProdutoAtivos(stubProdutos{err: produtos.ErrNaoEncontrado}, ligadoSe(true)).Consultar(ctx, nil, id); !errors.Is(err, ativos.ErrProdutoInexistente) {
		t.Fatal(err)
	}
	if _, err := portaProdutoAtivos(stubProdutos{err: errBanco}, ligadoSe(true)).Consultar(ctx, nil, id); !errors.Is(err, errBanco) {
		t.Fatal(err)
	}

	// Ativo e incidente vistos pelos problemas.
	pp := portasProblemas(stubProdutos{}, stubAtendimento{sol: atendimento.Solicitacao{ID: id, Numero: 3, Tipo: "incidente"},
		grupos: []atendimento.Recorrencia{{Total: 3}}}, stubAtivos{item: ativos.Ativo{Nome: "Switch"}}, ligadoSe(true))
	if nome, _, err := pp.Ativos.Consultar(ctx, nil, id); err != nil || nome != "Switch" {
		t.Fatal(err)
	}
	if inc, err := pp.Incidentes.Consultar(ctx, nil, id); err != nil || inc.Numero != 3 {
		t.Fatal(err)
	}
	if gs, err := pp.Incidentes.Recorrentes(ctx, nil, false, nil, time.Now(), 3); err != nil || len(gs) != 1 {
		t.Fatal(err)
	}
	desligado := portasProblemas(stubProdutos{}, stubAtendimento{}, stubAtivos{}, ligadoSe(false))
	if _, _, err := desligado.Ativos.Consultar(ctx, nil, id); !errors.Is(err, problemas.ErrIntegracaoDesligada) {
		t.Fatal(err)
	}
	if _, err := desligado.Incidentes.Consultar(ctx, nil, id); !errors.Is(err, problemas.ErrIntegracaoDesligada) {
		t.Fatal(err)
	}
	if gs, err := desligado.Incidentes.Recorrentes(ctx, nil, false, nil, time.Now(), 3); err != nil || len(gs) != 0 {
		t.Fatal("atendimento desligado: sem sugestão")
	}
	for nome, err := range map[string]error{"não encontrado": nil, "banco": errBanco} {
		ativoErr, solErr := ativos.ErrNaoEncontrado, atendimento.ErrNaoEncontrado
		if err != nil {
			ativoErr, solErr = err, err
		}
		quebrado := portasProblemas(stubProdutos{}, stubAtendimento{err: solErr}, stubAtivos{err: ativoErr}, ligadoSe(true))
		if _, _, e := quebrado.Ativos.Consultar(ctx, nil, id); e == nil {
			t.Errorf("ativo %s", nome)
		}
		if _, e := quebrado.Incidentes.Consultar(ctx, nil, id); e == nil {
			t.Errorf("incidente %s", nome)
		}
	}
	if _, err := portasProblemas(stubProdutos{}, stubAtendimento{err: errBanco}, stubAtivos{}, ligadoSe(true)).Incidentes.Recorrentes(ctx, nil, false, nil, time.Now(), 3); !errors.Is(err, errBanco) {
		t.Fatal("recorrentes com o banco fora")
	}

	// Alvos da mudança.
	alvos, prodMud := portasMudancas(stubProdutos{}, stubAtendimento{sol: atendimento.Solicitacao{Numero: 4, Titulo: "Caiu", Tipo: "incidente"}},
		stubAtivos{item: ativos.Ativo{Nome: "Switch"}}, stubProblemas{p: problemas.Problema{Numero: 2, Titulo: "Fonte"}}, ligadoSe(true))
	for tipo, rotulo := range map[string]string{
		mudancas.VinculoProblema: "Problema #2 · Fonte", mudancas.VinculoIncidente: "Solicitação #4 · Caiu", mudancas.VinculoAtivo: "Ativo · Switch",
	} {
		if a, err := alvos.Consultar(ctx, nil, tipo, id); err != nil || a.Rotulo != rotulo {
			t.Errorf("%s: %+v %v", tipo, a, err)
		}
		if !alvos.Disponivel(tipo) {
			t.Errorf("%s disponível", tipo)
		}
	}
	if alvos.Disponivel("cometa") {
		t.Fatal("tipo desconhecido")
	}
	if _, err := prodMud.Consultar(ctx, nil, id); err != nil {
		t.Fatal(err)
	}
	quebrados, _ := portasMudancas(stubProdutos{}, stubAtendimento{err: atendimento.ErrNaoEncontrado},
		stubAtivos{err: errBanco}, stubProblemas{err: problemas.ErrNaoEncontrado}, ligadoSe(true))
	if _, err := quebrados.Consultar(ctx, nil, mudancas.VinculoProblema, id); !errors.Is(err, mudancas.ErrVinculoInexistente) {
		t.Fatal(err)
	}
	if _, err := quebrados.Consultar(ctx, nil, mudancas.VinculoIncidente, id); !errors.Is(err, mudancas.ErrVinculoInexistente) {
		t.Fatal(err)
	}
	if _, err := quebrados.Consultar(ctx, nil, mudancas.VinculoAtivo, id); !errors.Is(err, errBanco) {
		t.Fatal(err)
	}
}

type concluiChamado struct {
	err     error
	entrega func(context.Context)
}

func (c concluiChamado) ConcluirPorMudanca(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, int64, string, string) (func(context.Context), error) {
	return c.entrega, c.err
}

type concluiCausa struct{ err error }

func (c concluiCausa) ConcluirPorMudanca(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, string) error {
	return c.err
}

func TestEfeitosDaCascata(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	r := mudancas.Resolucao{Numero: 3, Titulo: "Fonte"}
	if _, err := (efeitosCascata{}).Resolver(ctx, nil, mudancas.VinculoIncidente, id, r); !errors.Is(err, mudancas.ErrCascataIndisponivel) {
		t.Fatal(err)
	}
	pool := dbtest.Pool(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if _, err := (efeitosCascata{atend: concluiChamado{}}).Resolver(ctx, tx, mudancas.VinculoIncidente, id, r); !errors.Is(err, mudancas.ErrCascataIndisponivel) {
		t.Fatal(err)
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(false)}).Resolver(ctx, tx, mudancas.VinculoIncidente, id, r); !errors.Is(err, mudancas.ErrCascataIndisponivel) {
		t.Fatal(err)
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(false), prob: concluiCausa{}}).Resolver(ctx, tx, mudancas.VinculoProblema, id, r); !errors.Is(err, mudancas.ErrCascataIndisponivel) {
		t.Fatal(err)
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(true), atend: concluiChamado{}}).Resolver(ctx, tx, "ativo", id, r); !errors.Is(err, mudancas.ErrVinculoInvalido) {
		t.Fatal(err)
	}
	entregou := false
	ent, err := (efeitosCascata{ligado: ligadoSe(true), atend: concluiChamado{entrega: func(context.Context) { entregou = true }}}).Resolver(ctx, tx, mudancas.VinculoIncidente, id, r)
	if err != nil || ent == nil {
		t.Fatal(err)
	}
	ent(ctx)
	if !entregou {
		t.Fatal("entrega")
	}
	for _, causa := range []error{atendimento.ErrSemResponsavel, atendimento.ErrSolicitacaoFinalizada, atendimento.ErrTransicaoInvalida} {
		_, err := (efeitosCascata{ligado: ligadoSe(true), atend: concluiChamado{err: &atendimento.NaoConcluivel{Numero: 3, Estado: "xyz", Causa: causa}}}).Resolver(ctx, tx, mudancas.VinculoIncidente, id, r)
		var recusa *mudancas.VinculoRecusado
		if !errors.As(err, &recusa) || recusa.Mensagem == "" {
			t.Fatalf("%v -> %v", causa, err)
		}
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(true), atend: concluiChamado{err: errBanco}}).Resolver(ctx, tx, mudancas.VinculoIncidente, id, r); !errors.Is(err, errBanco) {
		t.Fatal(err)
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(true), prob: concluiCausa{}}).Resolver(ctx, tx, mudancas.VinculoProblema, id, r); err != nil {
		t.Fatal(err)
	}
	for _, causa := range []error{problemas.ErrSemCausaRaiz, problemas.ErrTransicaoInvalida} {
		_, err := (efeitosCascata{ligado: ligadoSe(true), prob: concluiCausa{err: &problemas.NaoConcluivel{Numero: 4, Estado: "xyz", Causa: causa}}}).Resolver(ctx, tx, mudancas.VinculoProblema, id, r)
		var recusa *mudancas.VinculoRecusado
		if !errors.As(err, &recusa) || recusa.Mensagem == "" {
			t.Fatalf("%v -> %v", causa, err)
		}
	}
	if _, err := (efeitosCascata{ligado: ligadoSe(true), prob: concluiCausa{err: errBanco}}).Resolver(ctx, tx, mudancas.VinculoProblema, id, r); !errors.Is(err, errBanco) {
		t.Fatal(err)
	}
	if rotuloEstado("desconhecido", rotuloSolicitacao) != "desconhecido" {
		t.Fatal("rótulo")
	}
}
