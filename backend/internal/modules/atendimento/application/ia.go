package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Assistente é a porta da IA do atendimento (ADR 065). internal/app liga
// o roteador de iaconfig, que escolhe a conexão, mascara dados pessoais
// para fornecedor externo e devolve domain.ErrSemIA sem conexão.
type Assistente interface {
	Disponivel(ctx context.Context) (bool, error)
	// Perguntar devolve o texto da IA e o nome da conexão que respondeu.
	Perguntar(ctx context.Context, sistema, pergunta string, maxTokens int) (texto, conexao string, err error)
}

// WithAssistente liga a IA. Sem ela, os pedidos devolvem domain.ErrSemIA.
func (s *Service) WithAssistente(a Assistente) *Service { s.assistente = a; return s }

// JanelaIndicadoresIA é o período dos indicadores da IA.
const JanelaIndicadoresIA = 30 * 24 * time.Hour

const sistemaClassificacao = `Você ajuda a equipe de uma central de serviços de TI de uma prefeitura a classificar chamados.
Responda só com um objeto JSON, sem texto em volta: {"impacto": N, "urgencia": N, "motivo": "uma frase"}.
impacto: 1 = uma pessoa; 2 = um setor ou um sistema usado por um setor; 3 = vários setores, atendimento ao cidadão ou sistema crítico.
urgencia: 1 = pode esperar; 2 = atrapalha o trabalho; 3 = impede o trabalho agora.
Em dúvida entre dois níveis, escolha o menor e diga a dúvida no motivo. Escreva o motivo em português.`

const sistemaResumo = `Você resume chamados de uma central de serviços de TI para quem vai assumir o atendimento.
Em até 5 linhas curtas, em português: o problema, o que já foi feito e o que falta.
Não invente fatos, nomes ou datas que não estejam no texto. Se faltar informação, diga o que falta.`

// EstadoIA diz se a IA do atendimento pode ser usada agora.
type EstadoIA struct {
	Disponivel bool `json:"disponivel"`
}

// EstadoDaIA responde só a quem atende alguma unidade: para os demais,
// a IA não existe.
func (s *Service) EstadoDaIA(ctx context.Context, identity auth.Identity) (EstadoIA, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if s.assistente == nil || (!cob.All && len(cob.Unidades) == 0) {
		return EstadoIA{}, nil
	}
	ok, err := s.assistente.Disponivel(ctx)
	if err != nil {
		return EstadoIA{}, MapError(err)
	}
	return EstadoIA{Disponivel: ok}, nil
}

// SugerirClassificacao pede impacto e urgência. A classificação do
// chamado não muda: quem atende aplica pela rota de sempre.
func (s *Service) SugerirClassificacao(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Sugestao, error) {
	sol, err := s.paraIA(ctx, identity, id)
	if err != nil {
		return domain.Sugestao{}, err
	}
	pergunta := fmt.Sprintf("Oferta: %s\nTítulo: %s\nDescrição: %s", sol.OfertaNome, sol.Titulo, sol.Descricao)
	texto, conexao, err := s.assistente.Perguntar(ctx, sistemaClassificacao, pergunta, 200)
	if err != nil {
		return domain.Sugestao{}, MapError(erroDaIA(err))
	}
	c, err := domain.LerClassificacao(texto)
	if err != nil {
		return domain.Sugestao{}, MapError(err)
	}
	return s.gravarSugestao(ctx, identity, domain.Sugestao{
		SolicitacaoID: id, Tipo: domain.SugestaoClassificacao, Conexao: conexao, Classificacao: &c,
	})
}

// Resumir pede o resumo do chamado para quem vai assumir. A conversa
// inteira vai, inclusive as notas internas: só a equipe vê o resumo.
func (s *Service) Resumir(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Sugestao, error) {
	sol, err := s.paraIA(ctx, identity, id)
	if err != nil {
		return domain.Sugestao{}, err
	}
	comentarios, err := s.repo.ListComentarios(ctx, s.pool, id, true)
	if err != nil {
		return domain.Sugestao{}, MapError(err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Oferta: %s\nEstado: %s\nTítulo: %s\nDescrição: %s\n", sol.OfertaNome, sol.Estado, sol.Titulo, sol.Descricao)
	for _, c := range comentarios {
		// Sem nomes: o resumo diz o que aconteceu, não quem disse.
		quem := "Resposta"
		switch {
		case c.Interno:
			quem = "Nota interna da equipe"
		case c.AutorID != nil && *c.AutorID == sol.SolicitanteID:
			quem = "Quem abriu"
		}
		fmt.Fprintf(&b, "\n%s (%s): %s", quem, c.CreatedAt.Format("02/01 15:04"), c.Texto)
	}
	texto, conexao, err := s.assistente.Perguntar(ctx, sistemaResumo, b.String(), 400)
	if err != nil {
		return domain.Sugestao{}, MapError(erroDaIA(err))
	}
	resumo, err := domain.LerResumo(texto)
	if err != nil {
		return domain.Sugestao{}, MapError(err)
	}
	return s.gravarSugestao(ctx, identity, domain.Sugestao{
		SolicitacaoID: id, Tipo: domain.SugestaoResumo, Conexao: conexao, Resumo: resumo,
	})
}

// paraIA confere que a IA está ligada e que quem pede atende o chamado.
func (s *Service) paraIA(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Solicitacao, error) {
	if s.assistente == nil {
		return domain.Solicitacao{}, MapError(domain.ErrSemIA)
	}
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, id)
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	ps := papeis(identity, sol)
	if len(ps) == 0 {
		return domain.Solicitacao{}, MapError(domain.ErrNaoEncontrado)
	}
	if !slices.Contains(ps, domain.PapelTecnico) {
		return domain.Solicitacao{}, MapError(domain.ErrForaDoEscopo)
	}
	return sol, MapError(s.recusarEspelho(ctx, s.pool, id))
}

func (s *Service) gravarSugestao(ctx context.Context, identity auth.Identity, sug domain.Sugestao) (domain.Sugestao, error) {
	sug.ID, sug.CreatedAt = uuid.New(), s.agora().UTC()
	if identity.UserID != uuid.Nil {
		quem := identity.UserID
		sug.PedidaPor = &quem
	}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InsertSugestao(ctx, tx, sug); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "atendimento.ia.sugerida", "solicitacao", sug.SolicitacaoID.String(), nil, sug))
	})
	if err != nil {
		return domain.Sugestao{}, MapError(err)
	}
	return sug, nil
}

// DecidirSugestao registra se quem atende aceitou ou recusou. Aceitar não
// aplica nada: a classificação segue pela rota de sempre.
func (s *Service) DecidirSugestao(ctx context.Context, identity auth.Identity, id uuid.UUID, aceita bool) (domain.Sugestao, error) {
	var out domain.Sugestao
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		sug, err := s.repo.TravarSugestao(ctx, tx, id)
		if err != nil {
			return err
		}
		sol, err := s.repo.GetSolicitacao(ctx, tx, sug.SolicitacaoID)
		if err != nil {
			return err
		}
		if !slices.Contains(papeis(identity, sol), domain.PapelTecnico) {
			return domain.ErrNaoEncontrado
		}
		if out, err = sug.Decidir(aceita, identity.UserID, s.agora().UTC()); err != nil {
			return err
		}
		if err := s.repo.DecidirSugestao(ctx, tx, out); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "atendimento.ia.decidida", "solicitacao", sol.ID.String(), sug, out))
	})
	if err != nil {
		return domain.Sugestao{}, MapError(err)
	}
	return out, nil
}

// IndicadoresIA conta as sugestões dos últimos 30 dias no escopo de quem
// atende: a taxa de aceite é o que diz se o modelo serve.
func (s *Service) IndicadoresIA(ctx context.Context, identity auth.Identity) ([]domain.IndicadorIA, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	out, err := s.repo.IndicadoresIA(ctx, s.pool, s.agora().Add(-JanelaIndicadoresIA), !cob.All, cob.Unidades)
	return out, MapError(err)
}

// erroDaIA traduz o erro da porta para o domínio. O adaptador já devolve
// domain.ErrSemIA quando falta conexão; o resto é falha da IA.
func erroDaIA(err error) error {
	if errors.Is(err, domain.ErrSemIA) {
		return domain.ErrSemIA
	}
	return fmt.Errorf("%w: %w", domain.ErrRespostaDaIA, err)
}
