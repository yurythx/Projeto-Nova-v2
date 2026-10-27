package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	atendimento "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	ativos "github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	mudancas "github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	problemasapp "github.com/yurythx/projeto-nova-v2/internal/modules/problemas/application"
	problemas "github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
	produtos "github.com/yurythx/projeto-nova-v2/internal/modules/produtos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Portas entre os plug-ins da gestão de serviços (fases 6 e 7). Plug-ins
// não se importam: cada adaptador lê o repositório do outro e traduz os
// erros para os do plug-in que pergunta. Com o módulo de origem
// desligado, o vínculo fica indisponível.

// produtoUnidade devolve a unidade dona de um produto, com os erros de
// quem pergunta.
type produtoUnidade struct {
	repo          produtos.Repository
	ativo         func() bool
	desligado     error
	naoEncontrado error
}

func (p produtoUnidade) Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (*uuid.UUID, error) {
	if !p.ativo() {
		return nil, p.desligado
	}
	item, err := p.repo.Get(ctx, db, id)
	if errors.Is(err, produtos.ErrNaoEncontrado) {
		return nil, p.naoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return item.UnidadeID, nil
}

// ativoProblema é o ativo visto pelos problemas.
type ativoProblema struct {
	repo  ativos.Repository
	ativo func() bool
}

func (a ativoProblema) Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (string, *uuid.UUID, error) {
	if !a.ativo() {
		return "", nil, problemas.ErrIntegracaoDesligada
	}
	item, err := a.repo.Get(ctx, db, id)
	if errors.Is(err, ativos.ErrNaoEncontrado) {
		return "", nil, problemas.ErrAtivoInexistente
	}
	if err != nil {
		return "", nil, err
	}
	return item.Nome, item.UnidadeID, nil
}

// incidenteProblema é o incidente visto pelos problemas.
type incidenteProblema struct {
	repo  atendimento.Repository
	ativo func() bool
}

func (i incidenteProblema) Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (problemas.Incidente, error) {
	if !i.ativo() {
		return problemas.Incidente{}, problemas.ErrIntegracaoDesligada
	}
	s, err := i.repo.GetSolicitacao(ctx, db, id)
	if errors.Is(err, atendimento.ErrNaoEncontrado) {
		return problemas.Incidente{}, problemas.ErrIncidenteInexistente
	}
	if err != nil {
		return problemas.Incidente{}, err
	}
	return problemas.Incidente{ID: s.ID, Numero: s.Numero, Titulo: s.Titulo, Tipo: s.Tipo, UnidadeID: s.UnidadeID}, nil
}

// Recorrentes: com o atendimento desligado, não há sugestão.
func (i incidenteProblema) Recorrentes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, minimo int) ([]problemas.Recorrencia, error) {
	if !i.ativo() {
		return []problemas.Recorrencia{}, nil
	}
	grupos, err := i.repo.IncidentesRecorrentes(ctx, db, restrito, unidades, desde, minimo)
	if err != nil {
		return nil, err
	}
	out := make([]problemas.Recorrencia, 0, len(grupos))
	for _, g := range grupos {
		out = append(out, problemas.Recorrencia{OfertaID: g.OfertaID, OfertaNome: g.OfertaNome, UnidadeID: g.UnidadeID,
			Total: g.Total, UltimoEm: g.UltimoEm, Solicitacoes: g.Solicitacoes})
	}
	return out, nil
}

// alvosMudanca é como a mudança enxerga problemas, incidentes e ativos.
type alvosMudanca struct {
	problemas   problemas.Repository
	solicitacao atendimento.Repository
	ativos      ativos.Repository
	ligado      func(key string) bool
}

var moduloDoVinculo = map[string]string{
	mudancas.VinculoProblema: "problemas", mudancas.VinculoIncidente: "atendimento", mudancas.VinculoAtivo: "ativos",
}

func (a alvosMudanca) Disponivel(tipo string) bool {
	key, ok := moduloDoVinculo[tipo]
	return ok && a.ligado(key)
}

func (a alvosMudanca) Consultar(ctx context.Context, db database.DBTX, tipo string, id uuid.UUID) (mudancas.Alvo, error) {
	switch tipo {
	case mudancas.VinculoProblema:
		p, err := a.problemas.Get(ctx, db, id)
		if err != nil {
			return mudancas.Alvo{}, inexistente(err, problemas.ErrNaoEncontrado)
		}
		return mudancas.Alvo{Rotulo: fmt.Sprintf("Problema #%d · %s", p.Numero, p.Titulo), UnidadeID: p.UnidadeID}, nil
	case mudancas.VinculoIncidente:
		s, err := a.solicitacao.GetSolicitacao(ctx, db, id)
		if err != nil {
			return mudancas.Alvo{}, inexistente(err, atendimento.ErrNaoEncontrado)
		}
		return mudancas.Alvo{Rotulo: fmt.Sprintf("Solicitação #%d · %s", s.Numero, s.Titulo), UnidadeID: s.UnidadeID, Tipo: s.Tipo}, nil
	default:
		item, err := a.ativos.Get(ctx, db, id)
		if err != nil {
			return mudancas.Alvo{}, inexistente(err, ativos.ErrNaoEncontrado)
		}
		return mudancas.Alvo{Rotulo: "Ativo · " + item.Nome, UnidadeID: item.UnidadeID}, nil
	}
}

// inexistente traduz o "não encontrado" do outro plug-in.
func inexistente(err, naoEncontrado error) error {
	if errors.Is(err, naoEncontrado) {
		return mudancas.ErrVinculoInexistente
	}
	return err
}

// Montagem das portas, a partir dos repositórios e do estado dos módulos.

func portaProdutoAtivos(repo produtos.Repository, ligado func(string) bool) ativos.Produtos {
	return produtoUnidade{repo: repo, ativo: func() bool { return ligado("produtos") },
		desligado: ativos.ErrProdutoIndisponivel, naoEncontrado: ativos.ErrProdutoInexistente}
}

func portasProblemas(prod produtos.Repository, atend atendimento.Repository, ativ ativos.Repository, ligado func(string) bool) problemasapp.Portas {
	return problemasapp.Portas{
		Incidentes: incidenteProblema{repo: atend, ativo: func() bool { return ligado("atendimento") }},
		Ativos:     ativoProblema{repo: ativ, ativo: func() bool { return ligado("ativos") }},
		Produtos: produtoUnidade{repo: prod, ativo: func() bool { return ligado("produtos") },
			desligado: problemas.ErrIntegracaoDesligada, naoEncontrado: problemas.ErrProdutoInexistente},
	}
}

func portasMudancas(prod produtos.Repository, atend atendimento.Repository, ativ ativos.Repository, prob problemas.Repository, ligado func(string) bool) (mudancas.Portas, mudancas.Produtos) {
	return alvosMudanca{problemas: prob, solicitacao: atend, ativos: ativ, ligado: ligado},
		produtoUnidade{repo: prod, ativo: func() bool { return ligado("produtos") },
			desligado: mudancas.ErrIntegracaoDesligada, naoEncontrado: mudancas.ErrVinculoInexistente}
}

// concluiSolicitacao e concluiProblema são os casos de uso que a mudança
// chama na própria transação. Os módulos concretos os implementam.
type concluiSolicitacao interface {
	ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, numero int64, titulo, texto string) (func(context.Context), error)
}

type concluiProblema interface {
	ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, texto string) error
}

type efeitosCascata struct {
	atend  concluiSolicitacao
	prob   concluiProblema
	ligado func(string) bool
}

func (e efeitosCascata) Resolver(ctx context.Context, db database.DBTX, tipo string, alvo uuid.UUID, r mudancas.Resolucao) (mudancas.Entrega, error) {
	tx, ok := db.(pgx.Tx)
	if !ok {
		return nil, mudancas.ErrCascataIndisponivel
	}
	switch tipo {
	case mudancas.VinculoIncidente:
		if e.ligado == nil || !e.ligado("atendimento") {
			return nil, mudancas.ErrCascataIndisponivel
		}
		entrega, err := e.atend.ConcluirPorMudanca(ctx, tx, alvo, r.Autor, r.Numero, r.Titulo, r.Texto)
		if err != nil {
			return nil, recusaSolicitacao(err)
		}
		return entrega, nil
	case mudancas.VinculoProblema:
		if e.ligado == nil || !e.ligado("problemas") {
			return nil, mudancas.ErrCascataIndisponivel
		}
		if err := e.prob.ConcluirPorMudanca(ctx, tx, alvo, r.Autor, r.Texto); err != nil {
			return nil, recusaProblema(err)
		}
		return nil, nil
	default:
		return nil, mudancas.ErrVinculoInvalido
	}
}

func recusaSolicitacao(err error) error {
	var n *atendimento.NaoConcluivel
	if !errors.As(err, &n) {
		return err
	}
	if errors.Is(n.Causa, atendimento.ErrSemResponsavel) {
		return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("a solicitação #%d precisa de um responsável antes de ser concluída pela mudança", n.Numero)}
	}
	if errors.Is(n.Causa, atendimento.ErrTarefasPendentes) {
		return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("a solicitação #%d tem tarefas de equipe pendentes e não é concluída pela mudança", n.Numero)}
	}
	if errors.Is(n.Causa, atendimento.ErrSolicitacaoFinalizada) {
		return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("a solicitação #%d está cancelada e não é concluída pela mudança", n.Numero)}
	}
	return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("a solicitação #%d está %s; só se conclui a partir do atendimento", n.Numero, rotuloEstado(n.Estado, rotuloSolicitacao))}
}

func recusaProblema(err error) error {
	var n *problemas.NaoConcluivel
	if !errors.As(err, &n) {
		return err
	}
	if errors.Is(n.Causa, problemas.ErrSemCausaRaiz) {
		return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("o problema #%d precisa da causa raiz antes de ser resolvido pela mudança", n.Numero)}
	}
	return &mudancas.VinculoRecusado{Mensagem: fmt.Sprintf("o problema #%d está %s e não passa direto para resolvido", n.Numero, rotuloEstado(n.Estado, rotuloProblema))}
}

func rotuloEstado(estado string, tabela map[string]string) string {
	if rotulo, ok := tabela[estado]; ok {
		return rotulo
	}
	return estado
}

var rotuloSolicitacao = map[string]string{
	"aberto": "aberta", "em_atendimento": "em atendimento",
	"aguardando_servidor": "aguardando o servidor", "aguardando_terceiro": "aguardando terceiro",
	"cancelado": "cancelada",
}

var rotuloProblema = map[string]string{
	"registrado": "registrado", "em_analise": "em análise", "erro_conhecido": "como erro conhecido",
}
