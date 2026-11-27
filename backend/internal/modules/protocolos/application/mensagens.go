package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// EnviarMensagemPublica é a mensagem do cidadão pela consulta. O CPF tem de ser o do pedido.
// O responsável recebe o aviso no sino; sem responsável, a novidade fica na mesa.
func (s *Service) EnviarMensagemPublica(ctx context.Context, numero, cpf, mensagem string) (domain.Consulta, error) {
	p, err := s.pedidoDoCidadao(ctx, numero, cpf)
	if err != nil {
		return domain.Consulta{}, err
	}
	hist, err := s.repo.ListarHistorico(ctx, s.db, p.ID)
	if err != nil {
		return domain.Consulta{}, err
	}
	prox, texto, err := p.MensagemDoCidadao(mensagem, domain.SeguidasDoCidadao(hist), s.agora())
	if err != nil {
		return domain.Consulta{}, err
	}
	linha := uuid.New()
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarPedido(ctx, tx, prox); err != nil {
			return err
		}
		if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
			ID: linha, PedidoID: prox.ID, Acao: domain.AcaoMensagemCidadao, Descricao: texto,
			Status: prox.Status, CreatedAt: s.agora(),
		}); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoMensagemCidadao, "protocolos_pedido", prox.ID.String(), p, prox, map[string]string{
			"numero": prox.Numero, "fato": "mensagem do cidadão", "orientacao": "responda pela mesa",
		})
	})
	if err != nil {
		return domain.Consulta{}, err
	}
	if prox.ResponsavelID != nil {
		s.avisarResponsavel(ctx, *prox.ResponsavelID, prox, "Nova mensagem do cidadão.", "protocolos:mensagem:"+linha.String())
	}
	return s.Consultar(ctx, prox.Numero)
}

// EnviarMensagem responde ao cidadão. A consulta mostra o texto e o cidadão recebe o aviso.
func (s *Service) EnviarMensagem(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error) {
	mensagem, err := domain.TextoInterno(mensagem, 2000)
	if err != nil {
		return domain.Pedido{}, err
	}
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoMensagemMesa, domain.AcaoMensagemMesa, "nova mensagem da prefeitura", texto(mensagem), nil, func(p domain.Pedido) (domain.Pedido, error) {
		prox, _, err := p.MensagemDaMesa(mensagem, s.agora())
		return prox, err
	})
}

// MarcarMensagensLidas apaga a novidade sem responder. Não entra na trilha.
func (s *Service) MarcarMensagensLidas(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	p, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	if err := s.exigir(ctx, auth.PermProtocolosTratar, p.UnidadeID); err != nil {
		return domain.Pedido{}, err
	}
	if !p.MensagemNova {
		return p, nil
	}
	p.MensagemNova = false
	if err := s.repo.AtualizarPedido(ctx, s.db, p); err != nil {
		return domain.Pedido{}, err
	}
	return p, nil
}
