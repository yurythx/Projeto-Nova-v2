package application

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Pessoas lista quem pode assumir o pedido: os usuários ativos com escopo na unidade.
func (s *Service) Pessoas(ctx context.Context, id uuid.UUID) ([]domain.Pessoa, error) {
	p, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if err := s.verPedido(ctx, p); err != nil {
		return nil, err
	}
	lista, err := s.repo.Pessoas(ctx, s.db, p.UnidadeID)
	if lista == nil {
		lista = []domain.Pessoa{}
	}
	return lista, err
}

// Aceitar dá dono ao pedido. Sem responsável indicado, quem aceita assume.
// Indicar outra pessoa pede protocolos:gerir na unidade.
func (s *Service) Aceitar(ctx context.Context, id uuid.UUID, responsavel *uuid.UUID) (domain.Pedido, error) {
	quem, _ := auth.IdentityFromContext(ctx)
	alvo := quem.UserID
	if responsavel != nil && *responsavel != uuid.Nil {
		alvo = *responsavel
	}
	var nome string
	p, err := s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoAceite, domain.AcaoAceite, "pedido em análise", func() string {
		if alvo == quem.UserID {
			return "assumido por " + nome
		}
		return "atribuído a " + nome
	}, func(p domain.Pedido) (domain.Pedido, error) {
		if alvo != quem.UserID {
			if err := s.exigir(ctx, auth.PermProtocolosGerir, p.UnidadeID); err != nil {
				return domain.Pedido{}, err
			}
		}
		var err error
		if nome, err = s.repo.Pessoa(ctx, s.db, alvo); err != nil {
			return domain.Pedido{}, err
		}
		prox, err := p.Aceitar(alvo, s.agora())
		prox.ResponsavelNome = nome
		return prox, err
	})
	if err == nil && alvo != quem.UserID {
		s.avisarResponsavel(ctx, alvo, p, "Você foi indicado como responsável.")
	}
	return p, err
}

// Transferir troca o responsável. Pode quem é o responsável ou quem gere a unidade.
func (s *Service) Transferir(ctx context.Context, id, novo uuid.UUID) (domain.Pedido, error) {
	quem, _ := auth.IdentityFromContext(ctx)
	var nome string
	p, err := s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoTransferencia, domain.AcaoTransferencia, "responsável alterado", func() string { return "transferido para " + nome }, func(p domain.Pedido) (domain.Pedido, error) {
		if p.ResponsavelID == nil || *p.ResponsavelID != quem.UserID {
			if err := s.exigir(ctx, auth.PermProtocolosGerir, p.UnidadeID); err != nil {
				return domain.Pedido{}, err
			}
		}
		var err error
		if nome, err = s.repo.Pessoa(ctx, s.db, novo); err != nil {
			return domain.Pedido{}, err
		}
		prox, err := p.Transferir(novo, s.agora())
		prox.ResponsavelNome = nome
		return prox, err
	})
	if err == nil && novo != quem.UserID {
		s.avisarResponsavel(ctx, novo, p, "O pedido foi transferido para você.")
	}
	return p, err
}

// Encaminhar pede a mudança para outra unidade. O destino aceita ou recusa.
func (s *Service) Encaminhar(ctx context.Context, id, destino uuid.UUID, justificativa string, prazoDias int) (domain.Pedido, error) {
	quem, _ := auth.IdentityFromContext(ctx)
	var nomeDestino string
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoEncaminhamento, domain.AcaoEncaminhamento, "pedido encaminhado", func() string {
		return "encaminhado a " + nomeDestino + ": " + strings.TrimSpace(justificativa)
	}, func(p domain.Pedido) (domain.Pedido, error) {
		var err error
		if _, nomeDestino, err = s.repo.Unidade(ctx, s.db, destino); err != nil {
			return domain.Pedido{}, err
		}
		prox, err := p.Encaminhar(destino, justificativa, prazoDias, quem.UserID, s.agora())
		if err != nil {
			return domain.Pedido{}, err
		}
		prox.Encaminhamento.UnidadeNome = nomeDestino
		return prox, nil
	})
}

// DecidirEncaminhamento é a resposta da unidade de destino. Aceito, o pedido muda de unidade sem responsável.
func (s *Service) DecidirEncaminhamento(ctx context.Context, id uuid.UUID, aceitar bool, motivo string) (domain.Pedido, error) {
	atual, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	if atual.Encaminhamento == nil {
		return domain.Pedido{}, domain.ErrInvalido
	}
	if err := s.exigir(ctx, auth.PermProtocolosTratar, atual.Encaminhamento.UnidadeID); err != nil {
		return domain.Pedido{}, err
	}
	if aceitar {
		prox, err := atual.ReceberEncaminhamento(s.agora())
		if err != nil {
			return domain.Pedido{}, err
		}
		return s.persistir(ctx, atual, prox, EventoEncaminhamentoOK, domain.AcaoEncaminhamentoAceito, "pedido recebido pela nova unidade",
			"recebido por "+atual.Encaminhamento.UnidadeNome)
	}
	motivo, err = domain.TextoInterno(motivo, 2000)
	if err != nil {
		return domain.Pedido{}, err
	}
	prox, err := atual.RecusarEncaminhamento(s.agora())
	if err != nil {
		return domain.Pedido{}, err
	}
	return s.persistir(ctx, atual, prox, EventoEncaminhamentoNo, domain.AcaoEncaminhamentoRecusado, "encaminhamento não aceito",
		atual.Encaminhamento.UnidadeNome+" recusou: "+motivo)
}

// Aprovar confirma a conclusão que aguarda. Pede protocolos:gerir na unidade.
func (s *Service) Aprovar(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.mudarCom(ctx, auth.PermProtocolosGerir, id, EventoAprovacao, domain.AcaoAprovacao, "pedido concluído", texto("conclusão aprovada"), func(p domain.Pedido) (domain.Pedido, error) {
		return p.Aprovar(s.agora())
	})
}

// Devolver recusa a conclusão; o pedido volta ao andamento. Pede protocolos:gerir na unidade.
func (s *Service) Devolver(ctx context.Context, id uuid.UUID, motivo string) (domain.Pedido, error) {
	motivo, err := domain.TextoInterno(motivo, 2000)
	if err != nil {
		return domain.Pedido{}, err
	}
	return s.mudarCom(ctx, auth.PermProtocolosGerir, id, EventoDevolucao, domain.AcaoDevolucao, "conclusão em revisão", texto(motivo), func(p domain.Pedido) (domain.Pedido, error) {
		return p.Devolver(s.agora())
	})
}

func (s *Service) avisarResponsavel(ctx context.Context, usuario uuid.UUID, p domain.Pedido, mensagem string) {
	if s.sino == nil {
		return
	}
	n := notificacoes.Nova{
		Modulo: "protocolos", Titulo: "Pedido " + p.Numero, Mensagem: mensagem,
		Link: "/protocolos/" + p.ID.String(), Chave: "protocolos:responsavel:" + p.ID.String() + ":" + usuario.String(),
	}
	if _, err := s.sino.Enviar(ctx, []uuid.UUID{usuario}, n); err != nil {
		s.logger.Warn("protocolos: aviso ao responsável não entregue", slog.String("erro", err.Error()))
	}
}
