package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// Abrir grava o pedido. No balcão, a sessão precisa de protocolos:abrir na unidade do serviço.
func (s *Service) Abrir(ctx context.Context, in Abertura, balcao bool) (domain.Pedido, error) {
	serv, err := s.repo.BuscarServico(ctx, s.db, in.ServicoID)
	if err != nil {
		return domain.Pedido{}, err
	}
	if !serv.Ativo {
		return domain.Pedido{}, domain.ErrNaoEncontrado
	}
	var usuario *uuid.UUID
	if balcao {
		if err := s.exigir(ctx, auth.PermProtocolosAbrir, serv.UnidadeID); err != nil {
			return domain.Pedido{}, err
		}
		id, _ := auth.IdentityFromContext(ctx)
		usuario = &id.UserID
	}
	form, err := s.repo.BuscarFormulario(ctx, s.db, serv.FormularioID)
	if err != nil {
		return domain.Pedido{}, err
	}
	if err := s.confirmarArquivos(ctx, serv.ID, form, in.Respostas); err != nil {
		return domain.Pedido{}, err
	}
	cpf := domain.DigitosCPF(in.CPF)
	if serv.ImpedirCPFAberto {
		n, err := s.repo.CPFEmAberto(ctx, s.db, serv.ID, cpf)
		if err != nil {
			return domain.Pedido{}, err
		}
		if n > 0 {
			return domain.Pedido{}, domain.ErrCPFAberto
		}
	}
	if serv.SoResponsavelTecnico {
		ok, err := s.repo.ResponsavelAtivo(ctx, s.db, cpf)
		if err != nil {
			return domain.Pedido{}, err
		}
		if !ok {
			return domain.Pedido{}, domain.ErrResponsavel
		}
	}
	sigla, nomeUnidade, err := s.repo.Unidade(ctx, s.db, serv.UnidadeID)
	if err != nil {
		return domain.Pedido{}, err
	}
	agora := s.agora()
	agenda, err := s.horario(ctx, serv, in, agora)
	if err != nil {
		return domain.Pedido{}, err
	}
	var pedido domain.Pedido
	var gravou bool
	for range 10 {
		numero, err := domain.MontarNumero(domain.Iniciais(sigla, nomeUnidade), agora.Year(), s.sorteio())
		if err != nil {
			return domain.Pedido{}, err
		}
		candidato, err := domain.Abrir(serv, form, numero, in.Nome, cpf, in.Email, in.Telefone, in.Prioridade, in.Respostas, usuario, agora)
		if err != nil {
			return domain.Pedido{}, err
		}
		candidato.UnidadeNome = nomeUnidade
		err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
			existe, err := s.repo.NumeroExiste(ctx, tx, numero)
			if err != nil {
				return err
			}
			if existe {
				return domain.ErrNumeroRepetido
			}
			if err := s.repo.InserirPedido(ctx, tx, candidato); err != nil {
				return err
			}
			if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
				ID: uuid.New(), PedidoID: candidato.ID, Acao: "criado", Descricao: "Pedido aberto",
				Status: candidato.Status, UsuarioID: usuario, CreatedAt: agora,
			}); err != nil {
				return err
			}
			if agenda != nil {
				agenda.PedidoID = candidato.ID
				if err := s.repo.InserirAgendamento(ctx, tx, *agenda); err != nil {
					return err
				}
			}
			texto, err := domain.Aviso(numero, "pedido recebido", "acompanhe pelo número")
			if err != nil {
				return err
			}
			return s.gravar(ctx, tx, EventoAberto, "protocolos_pedido", candidato.ID.String(), nil, candidato, map[string]string{
				"numero": numero, "fato": "pedido recebido", "orientacao": "acompanhe pelo número", "texto": texto,
			})
		})
		if errors.Is(err, domain.ErrNumeroRepetido) {
			continue
		}
		if err != nil {
			return domain.Pedido{}, err
		}
		pedido = candidato
		gravou = true
		break
	}
	if !gravou {
		return domain.Pedido{}, domain.ErrNumeroRepetido
	}
	s.avisar(ctx, usuario, pedido.Email, pedido.Telefone, "protocolos:aberto:"+pedido.ID.String(), "Pedido "+pedido.Numero, "Acompanhe pelo número.")
	return pedido, nil
}

func (s *Service) horario(ctx context.Context, serv domain.Servico, in Abertura, agora time.Time) (*domain.Agendamento, error) {
	if !serv.ExigeAgendamento {
		return nil, nil
	}
	faltas, ultima, err := s.repo.LerFaltas(ctx, s.db, domain.DigitosCPF(in.CPF))
	if err != nil {
		return nil, err
	}
	if domain.BloqueioPorFaltas(faltas, ultima, agora) {
		return nil, domain.ErrBloqueio
	}
	hoje := agora.In(domain.Cuiaba).Format("2006-01-02")
	data, hora := in.Data, in.Hora
	if data == "" || hora == "" {
		data, hora, err = domain.ProcurarVaga(serv.Agenda, agora, func(dia string) (int, error) {
			return s.repo.ContarAgendamentos(ctx, s.db, serv.ID, dia)
		})
		if err != nil {
			return nil, err
		}
	} else {
		dia, err := time.ParseInLocation("2006-01-02", data, domain.Cuiaba)
		if err != nil || !serv.Agenda.Cabe(dia) {
			return nil, domain.ErrInvalido
		}
		n, err := s.repo.ContarAgendamentos(ctx, s.db, serv.ID, data)
		if err != nil {
			return nil, err
		}
		if _, ok := serv.Agenda.HorarioLivre(n); !ok {
			return nil, domain.ErrSemVaga
		}
	}
	status, err := domain.StatusDoHorario(data, hoje)
	if err != nil {
		return nil, err
	}
	return &domain.Agendamento{ID: uuid.New(), ServicoID: serv.ID, Data: data, Hora: hora, Status: status}, nil
}

// Consultar devolve o recorte público. CPF e respostas não saem.
func (s *Service) Consultar(ctx context.Context, numero string) (domain.Consulta, error) {
	p, err := s.repo.BuscarPorNumero(ctx, s.db, numero)
	if err != nil {
		return domain.Consulta{}, err
	}
	serv, err := s.repo.BuscarServico(ctx, s.db, p.ServicoID)
	if err != nil {
		return domain.Consulta{}, err
	}
	hist, err := s.repo.ListarHistorico(ctx, s.db, p.ID)
	if err != nil {
		return domain.Consulta{}, err
	}
	var agenda *domain.Agendamento
	ag, err := s.repo.BuscarAgendamento(ctx, s.db, p.ID)
	if err == nil {
		agenda = &ag
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Consulta{}, err
	}
	return domain.Publico(p, serv.CampoAnonimo, hist, agenda), nil
}

// ListarPedidos devolve a mesa da visão da pessoa.
func (s *Service) ListarPedidos(ctx context.Context) ([]domain.Pedido, error) {
	v, err := s.visao(ctx)
	if err != nil {
		return nil, err
	}
	lista, err := s.repo.ListarPedidos(ctx, s.db, v)
	if lista == nil {
		lista = []domain.Pedido{}
	}
	if err == nil {
		s.marcarPrazos(lista)
	}
	return lista, err
}

func (s *Service) marcarPrazos(lista []domain.Pedido) {
	agora := s.agora()
	for i := range lista {
		lista[i].SituacaoPrazo = domain.SituacaoPrazo(lista[i].Status, lista[i].CreatedAt, lista[i].PrazoEm, agora)
	}
}

// BuscarPedido devolve a ficha para quem trata a unidade atual.
func (s *Service) BuscarPedido(ctx context.Context, id uuid.UUID) (Ficha, error) {
	p, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return Ficha{}, err
	}
	if err := s.ver(ctx, p.UnidadeID); err != nil {
		return Ficha{}, err
	}
	p.SituacaoPrazo = domain.SituacaoPrazo(p.Status, p.CreatedAt, p.PrazoEm, s.agora())
	hist, err := s.repo.ListarHistorico(ctx, s.db, p.ID)
	if err != nil {
		return Ficha{}, err
	}
	ficha := Ficha{Pedido: p, Historico: hist}
	if ficha.Historico == nil {
		ficha.Historico = []domain.Historico{}
	}
	ag, err := s.repo.BuscarAgendamento(ctx, s.db, p.ID)
	if err == nil {
		ficha.Agendamento = &ag
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return Ficha{}, err
	}
	return ficha, nil
}

func (s *Service) ver(ctx context.Context, unidade uuid.UUID) error {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return apperrors.Unauthorized("sessão necessária")
	}
	if auth.Can(id, auth.PermProtocolosTratar, auth.InUnidade(unidade)) || auth.Can(id, auth.PermProtocolosGerir, auth.InUnidade(unidade)) {
		return nil
	}
	return domain.ErrNaoEncontrado
}

func (s *Service) mudar(ctx context.Context, id uuid.UUID, evento, acao, fato string, fn func(domain.Pedido) (domain.Pedido, error)) (domain.Pedido, error) {
	atual, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	if err := s.exigir(ctx, auth.PermProtocolosTratar, atual.UnidadeID); err != nil {
		return domain.Pedido{}, err
	}
	prox, err := fn(atual)
	if err != nil {
		return domain.Pedido{}, err
	}
	if _, nome, err := s.repo.Unidade(ctx, s.db, prox.UnidadeID); err == nil {
		prox.UnidadeNome = nome
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Pedido{}, err
	}
	quem, _ := auth.IdentityFromContext(ctx)
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarPedido(ctx, tx, prox); err != nil {
			return err
		}
		if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
			ID: uuid.New(), PedidoID: prox.ID, Acao: acao, Descricao: fato,
			Status: prox.Status, UsuarioID: &quem.UserID, CreatedAt: s.agora(),
		}); err != nil {
			return err
		}
		texto, err := domain.Aviso(prox.Numero, fato, "acompanhe pelo número")
		if err != nil {
			return err
		}
		return s.gravar(ctx, tx, evento, "protocolos_pedido", prox.ID.String(), atual, prox, map[string]string{
			"numero": prox.Numero, "fato": fato, "orientacao": "acompanhe pelo número", "texto": texto,
		})
	})
	if err != nil {
		return domain.Pedido{}, err
	}
	s.avisar(ctx, prox.UsuarioID, prox.Email, prox.Telefone, "protocolos:"+acao+":"+prox.ID.String(), "Pedido "+prox.Numero, fato)
	return prox, nil
}

// Complementar pede informação e trava o avanço.
func (s *Service) Complementar(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoComplemento, "complemento", "há complementação", func(p domain.Pedido) (domain.Pedido, error) {
		return p.Complementar(mensagem, s.agora())
	})
}

// ResponderPublico é a resposta do cidadão à complementação. O CPF tem de ser o do pedido.
func (s *Service) ResponderPublico(ctx context.Context, numero, cpf, mensagem string, arquivos []string) (domain.Consulta, error) {
	p, err := s.pedidoDoCidadao(ctx, numero, cpf)
	if err != nil {
		return domain.Consulta{}, err
	}
	if p.Status != domain.StatusComplementacao {
		return domain.Consulta{}, domain.ErrInvalido
	}
	mensagem = strings.TrimSpace(mensagem)
	if mensagem == "" || utf8.RuneCountInString(mensagem) > 1000 {
		return domain.Consulta{}, domain.ErrInvalido
	}
	chaves, err := s.confirmarComplemento(ctx, p.ID, arquivos)
	if err != nil {
		return domain.Consulta{}, err
	}
	prox, err := p.ResponderComplemento(s.agora())
	if err != nil {
		return domain.Consulta{}, err
	}
	if _, nome, err := s.repo.Unidade(ctx, s.db, prox.UnidadeID); err == nil {
		prox.UnidadeNome = nome
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Consulta{}, err
	}
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarPedido(ctx, tx, prox); err != nil {
			return err
		}
		if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
			ID: uuid.New(), PedidoID: prox.ID, Acao: "resposta", Descricao: mensagem,
			Status: prox.Status, Anexos: chaves, CreatedAt: s.agora(),
		}); err != nil {
			return err
		}
		texto, err := domain.Aviso(prox.Numero, "complementação respondida", "acompanhe pelo número")
		if err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoResposta, "protocolos_pedido", prox.ID.String(), p, prox, map[string]string{
			"numero": prox.Numero, "fato": "complementação respondida", "orientacao": "acompanhe pelo número", "texto": texto,
		})
	})
	if err != nil {
		return domain.Consulta{}, err
	}
	s.avisar(ctx, prox.UsuarioID, prox.Email, prox.Telefone, "protocolos:resposta:"+prox.ID.String(), "Pedido "+prox.Numero, "Complementação respondida.")
	return s.Consultar(ctx, prox.Numero)
}

func (s *Service) pedidoDoCidadao(ctx context.Context, numero, cpf string) (domain.Pedido, error) {
	p, err := s.repo.BuscarPorNumero(ctx, s.db, numero)
	if err != nil {
		return domain.Pedido{}, err
	}
	informado := domain.DigitosCPF(cpf)
	if informado == "" || informado != domain.DigitosCPF(p.CPF) {
		return domain.Pedido{}, domain.ErrNaoEncontrado
	}
	return p, nil
}

// Responder devolve o pedido ao andamento.
func (s *Service) Responder(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoResposta, "resposta", "complementação respondida", func(p domain.Pedido) (domain.Pedido, error) {
		return p.ResponderComplemento(s.agora())
	})
}

// Avancar passa à etapa seguinte do serviço.
func (s *Service) Avancar(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoAvanco, "avanco", "pedido avançou de etapa", func(p domain.Pedido) (domain.Pedido, error) {
		serv, err := s.repo.BuscarServico(ctx, s.db, p.ServicoID)
		if err != nil {
			return domain.Pedido{}, err
		}
		return p.Avancar(serv.Workflow, s.agora())
	})
}

// Concluir grava a decisão.
func (s *Service) Concluir(ctx context.Context, id uuid.UUID, decisao, justificativa string) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoConclusao, "conclusao", "pedido concluído", func(p domain.Pedido) (domain.Pedido, error) {
		return p.Concluir(decisao, justificativa, s.agora())
	})
}

// Cancelar encerra o pedido.
func (s *Service) Cancelar(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoCancelado, "cancelado", "pedido cancelado", func(p domain.Pedido) (domain.Pedido, error) {
		return p.Cancelar(s.agora())
	})
}
