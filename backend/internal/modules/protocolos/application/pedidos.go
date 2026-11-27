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

// textoDeAviso e responderComplemento são as funções do domínio. O teste
// troca quando o número válido e o estado de complementação não falham sozinhos.
var textoDeAviso = domain.Aviso

var responderComplemento = func(p domain.Pedido, agora time.Time) (domain.Pedido, error) {
	return p.ResponderComplemento(agora)
}

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
	cal, err := s.calendario(ctx)
	if err != nil {
		return domain.Pedido{}, err
	}
	agenda, err := s.horario(ctx, serv, in, agora, cal)
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
		candidato, err := domain.Abrir(serv, form, numero, in.Nome, cpf, in.Email, in.Telefone, in.Prioridade, in.Respostas, usuario, agora, cal)
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
			texto, err := textoDeAviso(numero, "pedido recebido", "acompanhe pelo número")
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

func (s *Service) horario(ctx context.Context, serv domain.Servico, in Abertura, agora time.Time, cal domain.Calendario) (*domain.Agendamento, error) {
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
	data, hora := in.Data, domain.NormalizarHora(in.Hora)
	if in.Data == "" || in.Hora == "" {
		data, hora, err = domain.ProcurarVaga(serv.Agenda, agora, func(dia string) ([]string, error) {
			return s.repo.ListarHoras(ctx, s.db, serv.ID, dia)
		}, cal)
		if err != nil {
			return nil, err
		}
	} else {
		dia, err := time.ParseInLocation("2006-01-02", data, domain.Cuiaba)
		if err != nil || !serv.Agenda.Cabe(dia) || cal.Feriado(dia) || !serv.Agenda.HoraNaGrade(hora) {
			return nil, domain.ErrInvalido
		}
		tomadas, err := s.repo.ListarHoras(ctx, s.db, serv.ID, data)
		if err != nil {
			return nil, err
		}
		if horaTomada(tomadas, hora) {
			return nil, domain.ErrVaga
		}
	}
	status, err := domain.StatusDoHorario(data, hoje)
	if err != nil {
		return nil, err
	}
	return &domain.Agendamento{ID: uuid.New(), ServicoID: serv.ID, Data: data, Hora: hora, Status: status}, nil
}

func horaTomada(tomadas []string, hora string) bool {
	for _, item := range tomadas {
		if domain.NormalizarHora(item) == hora {
			return true
		}
	}
	return false
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
		cal, err := s.calendario(ctx)
		if err != nil {
			return nil, err
		}
		s.marcarPrazos(lista, cal)
		for i := range lista {
			lista[i].Respostas = nil
			lista[i].Campos = nil
			lista[i].Email = ""
			lista[i].Telefone = ""
		}
	}
	return lista, err
}

func (s *Service) marcarPrazos(lista []domain.Pedido, cal domain.Calendario) {
	agora := s.agora()
	for i := range lista {
		lista[i].SituacaoPrazo = domain.SituacaoPrazo(lista[i].Status, lista[i].CreatedAt, lista[i].PrazoEm, agora, cal)
	}
}

// BuscarPedido devolve a ficha para quem trata a unidade atual.
func (s *Service) BuscarPedido(ctx context.Context, id uuid.UUID) (Ficha, error) {
	p, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return Ficha{}, err
	}
	if err := s.verPedido(ctx, p); err != nil {
		return Ficha{}, err
	}
	cal, err := s.calendario(ctx)
	if err != nil {
		return Ficha{}, err
	}
	p.SituacaoPrazo = domain.SituacaoPrazo(p.Status, p.CreatedAt, p.PrazoEm, s.agora(), cal)
	hist, err := s.repo.ListarHistorico(ctx, s.db, p.ID)
	if err != nil {
		return Ficha{}, err
	}
	serv, err := s.repo.BuscarServico(ctx, s.db, p.ServicoID)
	if err != nil {
		return Ficha{}, err
	}
	ficha := Ficha{Pedido: p, Historico: hist, Etapas: serv.Workflow.Ordenadas()}
	if ficha.Historico == nil {
		ficha.Historico = []domain.Historico{}
	}
	if ficha.Etapas == nil {
		ficha.Etapas = []domain.Etapa{}
	}
	ag, err := s.repo.BuscarAgendamento(ctx, s.db, p.ID)
	if err == nil {
		ficha.Agendamento = &ag
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return Ficha{}, err
	}
	return ficha, nil
}

// verPedido aceita quem trata a unidade atual ou a de destino do encaminhamento pendente.
func (s *Service) verPedido(ctx context.Context, p domain.Pedido) error {
	err := s.ver(ctx, p.UnidadeID)
	if err != nil && p.Encaminhamento != nil && s.ver(ctx, p.Encaminhamento.UnidadeID) == nil {
		return nil
	}
	return err
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

// acoesComDono só andam com responsável quando o serviço exige aceite.
var acoesComDono = map[string]bool{"complemento": true, "avanco": true, "conclusao": true, domain.AcaoAguardaAprovacao: true}

func (s *Service) mudar(ctx context.Context, id uuid.UUID, evento, acao, fato, descricao string, fn func(domain.Pedido) (domain.Pedido, error)) (domain.Pedido, error) {
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, evento, acao, fato, texto(descricao), nil, fn)
}

// texto adia uma descrição já conhecida.
func texto(t string) func() string { return func() string { return t } }

// mudarCom confere a permissão na unidade, recusa o pedido com encaminhamento
// pendente e o pedido sem dono quando o serviço exige aceite. A descrição é
// lida depois de fn, que pode descobrir o nome de quem assume.
// Os anexos são lidos depois de fn, que os confirma no armazenamento.
// Permissão vazia: quem chama já conferiu (o aprovador indicado no serviço).
func (s *Service) mudarCom(ctx context.Context, perm auth.Permission, id uuid.UUID, evento, acao, fato string, descricao func() string, anexos *[]string, fn func(domain.Pedido) (domain.Pedido, error)) (domain.Pedido, error) {
	atual, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	if perm != "" {
		if err := s.exigir(ctx, perm, atual.UnidadeID); err != nil {
			return domain.Pedido{}, err
		}
	}
	if atual.Encaminhamento != nil {
		return domain.Pedido{}, domain.ErrEncaminhamento
	}
	if acoesComDono[acao] {
		if err := s.exigirDono(ctx, atual); err != nil {
			return domain.Pedido{}, err
		}
	}
	prox, err := fn(atual)
	if err != nil {
		return domain.Pedido{}, err
	}
	var chaves []string
	if anexos != nil {
		chaves = *anexos
	}
	return s.persistir(ctx, atual, prox, evento, acao, fato, descricao(), chaves)
}

// exigirDono recusa o pedido sem responsável quando o serviço exige aceite.
func (s *Service) exigirDono(ctx context.Context, p domain.Pedido) error {
	if p.ResponsavelID != nil {
		return nil
	}
	serv, err := s.repo.BuscarServico(ctx, s.db, p.ServicoID)
	if err != nil {
		return err
	}
	if serv.ExigeAceite {
		return domain.ErrSemResponsavel
	}
	return nil
}

// persistir grava o pedido, a linha da trilha, o evento e a auditoria, e avisa.
// Sem sessão (a rotina de prazos), a linha da trilha fica sem usuário.
func (s *Service) persistir(ctx context.Context, atual, prox domain.Pedido, evento, acao, fato, descricao string, anexos []string) (domain.Pedido, error) {
	if _, nome, err := s.repo.Unidade(ctx, s.db, prox.UnidadeID); err == nil {
		prox.UnidadeNome = nome
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Pedido{}, err
	}
	usuario := quemAge(ctx)
	if strings.TrimSpace(descricao) == "" {
		descricao = fato
	}
	err := s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarPedido(ctx, tx, prox); err != nil {
			return err
		}
		if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
			ID: uuid.New(), PedidoID: prox.ID, Acao: acao, Descricao: descricao,
			Status: prox.Status, Anexos: anexos, UsuarioID: usuario, CreatedAt: s.agora(),
		}); err != nil {
			return err
		}
		texto, err := textoDeAviso(prox.Numero, fato, "acompanhe pelo número")
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

// Complementacao é o pedido de informação ao cidadão, com prazo e anexos da mesa.
type Complementacao struct {
	Mensagem  string
	PrazoDias int
	Anexos    []string
}

// Complementar pede informação e trava o avanço.
func (s *Service) Complementar(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error) {
	return s.ComplementarCom(ctx, id, Complementacao{Mensagem: mensagem})
}

// ComplementarCom pede informação com prazo em dias úteis. Vencido sem resposta, o pedido é cancelado.
func (s *Service) ComplementarCom(ctx context.Context, id uuid.UUID, in Complementacao) (domain.Pedido, error) {
	var chaves []string
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoComplemento, "complemento", "há complementação", texto(strings.TrimSpace(in.Mensagem)), &chaves, func(p domain.Pedido) (domain.Pedido, error) {
		var err error
		if chaves, err = s.confirmarMesa(ctx, p.ID, in.Anexos); err != nil {
			return domain.Pedido{}, err
		}
		cal, err := s.calendario(ctx)
		if err != nil {
			return domain.Pedido{}, err
		}
		if p, err = p.Complementar(in.Mensagem, s.agora()); err != nil {
			return domain.Pedido{}, err
		}
		return p.PrazoDaComplementacao(in.PrazoDias, s.agora(), cal)
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
	prox, err := responderComplemento(p, s.agora())
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
		texto, err := textoDeAviso(prox.Numero, "complementação respondida", "acompanhe pelo número")
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

// Responder devolve o pedido ao andamento e guarda o texto recebido na mesa.
func (s *Service) Responder(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error) {
	mensagem, err := domain.TextoInterno(mensagem, 2000)
	if err != nil {
		return domain.Pedido{}, err
	}
	return s.mudar(ctx, id, EventoResposta, "resposta", "complementação respondida", mensagem, func(p domain.Pedido) (domain.Pedido, error) {
		return p.ResponderComplemento(s.agora())
	})
}

// Anotar grava um texto que a consulta pública não mostra.
func (s *Service) Anotar(ctx context.Context, id uuid.UUID, mensagem string) (domain.Pedido, error) {
	return s.AnotarCom(ctx, id, mensagem, nil)
}

// AnotarCom grava a nota interna com anexos da mesa.
func (s *Service) AnotarCom(ctx context.Context, id uuid.UUID, mensagem string, anexos []string) (domain.Pedido, error) {
	mensagem, err := domain.TextoInterno(mensagem, 2000)
	if err != nil {
		return domain.Pedido{}, err
	}
	atual, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	if err := s.exigir(ctx, auth.PermProtocolosTratar, atual.UnidadeID); err != nil {
		return domain.Pedido{}, err
	}
	if domain.EstadoFinal(atual.Status) {
		return domain.Pedido{}, domain.ErrEstadoFinal
	}
	if err := s.exigirDono(ctx, atual); err != nil {
		return domain.Pedido{}, err
	}
	chaves, err := s.confirmarMesa(ctx, atual.ID, anexos)
	if err != nil {
		return domain.Pedido{}, err
	}
	quem, _ := auth.IdentityFromContext(ctx)
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InserirHistorico(ctx, tx, domain.Historico{
			ID: uuid.New(), PedidoID: atual.ID, Acao: domain.AcaoNota, Descricao: mensagem,
			Status: atual.Status, Anexos: chaves, UsuarioID: &quem.UserID, CreatedAt: s.agora(),
		}); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoNota, "protocolos_pedido", atual.ID.String(), nil, map[string]string{"acao": domain.AcaoNota}, map[string]string{
			"numero": atual.Numero, "fato": "nota interna", "orientacao": "a nota fica na mesa",
		})
	})
	return atual, err
}

// Edicao é o que a mesa troca no pedido. Respostas nulas, prioridade e prazo vazios mantêm o atual.
type Edicao struct {
	Justificativa string
	Respostas     map[string]any
	Prioridade    string
	Prazo         string
	// ProcessoSEI nulo mantém; vazio desfaz o vínculo.
	ProcessoSEI *string
}

// Editar troca as respostas, a prioridade e o prazo, e guarda a justificativa só na mesa.
func (s *Service) Editar(ctx context.Context, id uuid.UUID, in Edicao) (domain.Pedido, error) {
	justificativa, err := domain.TextoInterno(in.Justificativa, 4000)
	if err != nil {
		return domain.Pedido{}, err
	}
	return s.mudar(ctx, id, EventoEdicao, domain.AcaoEdicao, "pedido atualizado", justificativa, func(p domain.Pedido) (domain.Pedido, error) {
		if in.Respostas != nil {
			var err error
			if p, err = p.Editar(preservarArquivos(p, in.Respostas), s.agora()); err != nil {
				return domain.Pedido{}, err
			}
		}
		if in.ProcessoSEI != nil {
			var err error
			if p, err = p.VincularSEI(*in.ProcessoSEI, s.agora()); err != nil {
				return domain.Pedido{}, err
			}
		}
		return p.Ajustar(in.Prioridade, in.Prazo, s.agora())
	})
}

func preservarArquivos(p domain.Pedido, respostas map[string]any) map[string]any {
	if respostas == nil {
		respostas = map[string]any{}
	}
	for _, campo := range p.Campos {
		if campo.Tipo != "file" {
			continue
		}
		if atual, ok := p.Respostas[campo.ID]; ok {
			respostas[campo.ID] = atual
		}
	}
	return respostas
}

// Avancar passa à etapa seguinte do serviço.
func (s *Service) Avancar(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.AvancarCom(ctx, id, nil)
}

// AvancarCom passa à etapa seguinte com os anexos da mesa.
func (s *Service) AvancarCom(ctx context.Context, id uuid.UUID, anexos []string) (domain.Pedido, error) {
	var chaves []string
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, EventoAvanco, "avanco", "pedido avançou de etapa", texto(""), &chaves, func(p domain.Pedido) (domain.Pedido, error) {
		var err error
		if chaves, err = s.confirmarMesa(ctx, p.ID, anexos); err != nil {
			return domain.Pedido{}, err
		}
		serv, err := s.repo.BuscarServico(ctx, s.db, p.ServicoID)
		if err != nil {
			return domain.Pedido{}, err
		}
		cal, err := s.calendario(ctx)
		if err != nil {
			return domain.Pedido{}, err
		}
		return p.Avancar(serv.Workflow, s.agora(), cal)
	})
}

// Conclusao é a decisão, com o processo SEI e os anexos da mesa.
type Conclusao struct {
	Decisao       string
	Justificativa string
	// ProcessoSEI nulo mantém o vínculo atual.
	ProcessoSEI *string
	Anexos      []string
}

// Concluir grava a decisão. Se o serviço exige aprovação, a conclusão espera quem gere a unidade.
func (s *Service) Concluir(ctx context.Context, id uuid.UUID, decisao, justificativa string) (domain.Pedido, error) {
	return s.ConcluirCom(ctx, id, Conclusao{Decisao: decisao, Justificativa: justificativa})
}

// ConcluirCom grava a decisão, o processo SEI e os anexos.
func (s *Service) ConcluirCom(ctx context.Context, id uuid.UUID, in Conclusao) (domain.Pedido, error) {
	decisao, justificativa := in.Decisao, in.Justificativa
	atual, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	serv, err := s.repo.BuscarServico(ctx, s.db, atual.ServicoID)
	if err != nil {
		return domain.Pedido{}, err
	}
	evento, acao, fato := EventoConclusao, "conclusao", "pedido concluído"
	concluir := func(p domain.Pedido) (domain.Pedido, error) { return p.Concluir(decisao, justificativa, s.agora()) }
	if serv.ExigeAprovacao {
		evento, acao, fato = EventoAguardaAprovacao, domain.AcaoAguardaAprovacao, "conclusão enviada para aprovação"
		concluir = func(p domain.Pedido) (domain.Pedido, error) {
			return p.PedirAprovacao(decisao, justificativa, s.agora())
		}
	}
	var chaves []string
	return s.mudarCom(ctx, auth.PermProtocolosTratar, id, evento, acao, fato, texto(justificativa), &chaves, func(p domain.Pedido) (domain.Pedido, error) {
		var err error
		if chaves, err = s.confirmarMesa(ctx, p.ID, in.Anexos); err != nil {
			return domain.Pedido{}, err
		}
		if in.ProcessoSEI != nil {
			if p, err = p.VincularSEI(*in.ProcessoSEI, s.agora()); err != nil {
				return domain.Pedido{}, err
			}
		}
		return concluir(p)
	})
}

// Cancelar encerra o pedido.
func (s *Service) Cancelar(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	return s.mudar(ctx, id, EventoCancelado, "cancelado", "pedido cancelado", "", func(p domain.Pedido) (domain.Pedido, error) {
		return p.Cancelar(s.agora())
	})
}
