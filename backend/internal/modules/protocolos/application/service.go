// Package application abre o pedido e grava, na mesma transação, o evento e a auditoria.
package application

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

const (
	EventoAberto      = "protocolos.pedido.aberto"
	EventoComplemento = "protocolos.pedido.complemento"
	EventoResposta    = "protocolos.pedido.resposta"
	EventoAvanco      = "protocolos.pedido.avancado"
	EventoConclusao   = "protocolos.pedido.concluido"
	EventoCancelado   = "protocolos.pedido.cancelado"
	EventoFalta       = "protocolos.falta.registrada"
	EventoNota        = "protocolos.pedido.nota"
	// A conversa com o cidadão.
	EventoMensagemCidadao = "protocolos.pedido.mensagem_cidadao"
	EventoMensagemMesa    = "protocolos.pedido.mensagem_mesa"
	EventoEdicao          = "protocolos.pedido.editado"
	EventoFormulario      = "protocolos.formulario.salvo"
	EventoServico         = "protocolos.servico.salvo"
	// Aceite, encaminhamento e aprovação.
	EventoAceite           = "protocolos.pedido.aceito"
	EventoTransferencia    = "protocolos.pedido.transferido"
	EventoEncaminhamento   = "protocolos.pedido.encaminhado"
	EventoEncaminhamentoOK = "protocolos.pedido.encaminhamento_aceito"
	EventoEncaminhamentoNo = "protocolos.pedido.encaminhamento_recusado"
	EventoAguardaAprovacao = "protocolos.pedido.aguarda_aprovacao"
	EventoAprovacao        = "protocolos.pedido.aprovado"
	EventoDevolucao        = "protocolos.pedido.devolvido"
)

// Sino é a caixa da plataforma. E-mail e WhatsApp saem dela quando o canal está ligado.
type Sino interface {
	Enviar(ctx context.Context, usuarios []uuid.UUID, n notificacoes.Nova) (int, error)
}

// Ficha é o pedido com a trilha e o horário, para a mesa.
type Ficha struct {
	Pedido      domain.Pedido       `json:"pedido"`
	Historico   []domain.Historico  `json:"historico"`
	Agendamento *domain.Agendamento `json:"agendamento,omitempty"`
	// Etapas do workflow do serviço, na ordem: a ficha desenha o fluxo.
	Etapas []domain.Etapa `json:"etapas"`
}

// Abertura é o que o cidadão ou o balcão enviam.
type Abertura struct {
	ServicoID  uuid.UUID
	Nome       string
	CPF        string
	Email      string
	Telefone   string
	Prioridade string
	Data       string
	Hora       string
	Respostas  map[string]any
}

// Service é o caso de uso do plug-in.
type Service struct {
	db       database.DBTX
	run      func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error
	repo     domain.Repository
	out      *outbox.Writer
	sino     Sino
	arquivos Arquivos
	logger   *slog.Logger
	agora    func() time.Time
	sorteio  func() int
}

// NewService liga o pool, o outbox e o relógio de Cuiabá.
func NewService(pool *pgxpool.Pool, repo domain.Repository, ob *outbox.Writer, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db: pool, repo: repo, out: ob, logger: logger,
		run: func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
			return database.WithTx(ctx, pool, fn)
		},
		agora:   func() time.Time { return time.Now().In(domain.Cuiaba) },
		sorteio: sortear,
	}
}

// ComSino liga a caixa. Sem ela, o pedido continua e o aviso fica no evento.
func (s *Service) ComSino(n *notificacoes.Service) *Service {
	if n != nil {
		s.sino = n
	}
	return s
}

// MapError traduz o domínio para HTTP.
func MapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNaoEncontrado):
		return apperrors.NotFound("não encontrado")
	case errors.Is(err, domain.ErrInvalido):
		return apperrors.Validation("confira os dados do formulário, do serviço ou do pedido")
	case errors.Is(err, domain.ErrEstadoFinal):
		return apperrors.Conflict("pedido encerrado")
	case errors.Is(err, domain.ErrComplemento):
		return apperrors.Conflict("pedido em complementação")
	case errors.Is(err, domain.ErrNumeroRepetido):
		return apperrors.Conflict("número já existe")
	case errors.Is(err, domain.ErrCPFAberto):
		return apperrors.Conflict("já há pedido deste CPF neste serviço")
	case errors.Is(err, domain.ErrResponsavel):
		return apperrors.Forbidden("serviço exclusivo de responsável técnico")
	case errors.Is(err, domain.ErrCPFCadastrado):
		return apperrors.Conflict("CPF já cadastrado como responsável técnico")
	case errors.Is(err, domain.ErrBloqueio):
		return apperrors.Conflict("agendamento bloqueado por faltas")
	case errors.Is(err, domain.ErrSemVaga):
		return apperrors.Conflict("sem horário nos próximos 30 dias")
	case errors.Is(err, domain.ErrVaga):
		return apperrors.Conflict("horário indisponível")
	case errors.Is(err, domain.ErrFalta):
		return apperrors.Conflict("falta já registrada neste horário")
	case errors.Is(err, domain.ErrSemResponsavel):
		return apperrors.Conflict("aceite o pedido antes de tratá-lo")
	case errors.Is(err, domain.ErrEncaminhamento):
		return apperrors.Conflict("pedido com encaminhamento pendente")
	case errors.Is(err, domain.ErrAprovacao):
		return apperrors.Conflict("conclusão aguardando aprovação")
	case errors.Is(err, domain.ErrMuitasMensagens):
		return apperrors.Conflict("aguarde a resposta da prefeitura antes de mandar outra mensagem")
	default:
		return err
	}
}

var inteiroAleatorio = func() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func sortear() int {
	n, err := inteiroAleatorio()
	if err != nil {
		return 0
	}
	return n
}

func (s *Service) exigir(ctx context.Context, perm auth.Permission, unidade uuid.UUID) error {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return apperrors.Unauthorized("sessão necessária")
	}
	if !auth.Can(id, perm, auth.InUnidade(unidade)) {
		return apperrors.Forbidden("fora da unidade")
	}
	return nil
}

func (s *Service) visao(ctx context.Context) (domain.Visao, error) {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return domain.Visao{}, apperrors.Unauthorized("sessão necessária")
	}
	c := auth.CoverageOf(id, auth.PermProtocolosTratar)
	if !c.All && len(c.Unidades) == 0 && len(c.Entidades) == 0 {
		c = auth.CoverageOf(id, auth.PermProtocolosGerir)
	}
	v := domain.Visao{Tudo: c.All, Unidades: c.Unidades, Entidades: c.Entidades}
	if v.Unidades == nil {
		v.Unidades = []uuid.UUID{}
	}
	if v.Entidades == nil {
		v.Entidades = []uuid.UUID{}
	}
	return v, nil
}

func (s *Service) gravar(ctx context.Context, tx pgx.Tx, tipo, agregado, id string, antes, depois any, payload map[string]string) error {
	if err := s.out.Write(ctx, tx, tipo, agregado, id, uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, tipo, agregado, id, antes, depois))
}

func (s *Service) avisar(ctx context.Context, usuario *uuid.UUID, email, telefone, chave, titulo, mensagem string) {
	n := notificacoes.Nova{
		Modulo: "protocolos", Titulo: titulo, Mensagem: mensagem, Link: "/carta", Chave: chave,
	}
	if s.sino != nil && usuario != nil && *usuario != uuid.Nil {
		interno := n
		interno.Link = "/protocolos"
		if _, err := s.sino.Enviar(ctx, []uuid.UUID{*usuario}, interno); err != nil {
			s.logger.Warn("protocolos: aviso não entregue", slog.String("erro", err.Error()))
		}
	}
	if contato, ok := s.sino.(interface {
		AvisarContato(context.Context, string, string, notificacoes.Nova)
	}); ok {
		contato.AvisarContato(ctx, email, telefone, n)
	}
}

// SalvarFormulario cria ou substitui o desenho.
func (s *Service) SalvarFormulario(ctx context.Context, f domain.Formulario) (domain.Formulario, error) {
	if err := s.exigir(ctx, auth.PermProtocolosGerir, f.UnidadeID); err != nil {
		return domain.Formulario{}, err
	}
	if _, _, err := s.repo.Unidade(ctx, s.db, f.UnidadeID); err != nil {
		return domain.Formulario{}, err
	}
	agora := s.agora()
	if f.ID == uuid.Nil {
		novo, err := domain.NovoFormulario(f.UnidadeID, f.Titulo, f.Descricao, f.Instrucao, f.Campos, agora)
		if err != nil {
			return domain.Formulario{}, err
		}
		err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
			if err := s.repo.InserirFormulario(ctx, tx, novo); err != nil {
				return err
			}
			return s.gravar(ctx, tx, EventoFormulario, "protocolos_formulario", novo.ID.String(), nil, novo, map[string]string{"id": novo.ID.String()})
		})
		return novo, err
	}
	atual, err := s.repo.BuscarFormulario(ctx, s.db, f.ID)
	if err != nil {
		return domain.Formulario{}, err
	}
	if err := s.exigir(ctx, auth.PermProtocolosGerir, atual.UnidadeID); err != nil {
		return domain.Formulario{}, err
	}
	novo, err := domain.NovoFormulario(atual.UnidadeID, f.Titulo, f.Descricao, f.Instrucao, f.Campos, agora)
	if err != nil {
		return domain.Formulario{}, err
	}
	novo.ID, novo.CreatedAt, novo.Ativo = atual.ID, atual.CreatedAt, f.Ativo
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarFormulario(ctx, tx, novo); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoFormulario, "protocolos_formulario", novo.ID.String(), atual, novo, map[string]string{"id": novo.ID.String()})
	})
	return novo, err
}

// ListarFormularios devolve os desenhos da visão da pessoa.
func (s *Service) ListarFormularios(ctx context.Context) ([]domain.Formulario, error) {
	v, err := s.visao(ctx)
	if err != nil {
		return nil, err
	}
	lista, err := s.repo.ListarFormularios(ctx, s.db, v)
	if lista == nil {
		lista = []domain.Formulario{}
	}
	return lista, err
}

// SalvarServico cria ou substitui o serviço da carta.
func (s *Service) SalvarServico(ctx context.Context, in domain.Servico) (domain.Servico, error) {
	if err := s.exigir(ctx, auth.PermProtocolosGerir, in.UnidadeID); err != nil {
		return domain.Servico{}, err
	}
	if _, _, err := s.repo.Unidade(ctx, s.db, in.UnidadeID); err != nil {
		return domain.Servico{}, err
	}
	form, err := s.repo.BuscarFormulario(ctx, s.db, in.FormularioID)
	if err != nil {
		return domain.Servico{}, err
	}
	if form.UnidadeID != in.UnidadeID {
		return domain.Servico{}, domain.ErrInvalido
	}
	agora := s.agora()
	if in.ID == uuid.Nil {
		novo, err := domain.NovoServico(in, agora)
		if err != nil {
			return domain.Servico{}, err
		}
		err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
			if err := s.repo.InserirServico(ctx, tx, novo); err != nil {
				return err
			}
			return s.gravar(ctx, tx, EventoServico, "protocolos_servico", novo.ID.String(), nil, novo, map[string]string{"id": novo.ID.String()})
		})
		return novo, err
	}
	atual, err := s.repo.BuscarServico(ctx, s.db, in.ID)
	if err != nil {
		return domain.Servico{}, err
	}
	if err := s.exigir(ctx, auth.PermProtocolosGerir, atual.UnidadeID); err != nil {
		return domain.Servico{}, err
	}
	novo, err := domain.NovoServico(in, agora)
	if err != nil {
		return domain.Servico{}, err
	}
	novo.ID, novo.CreatedAt, novo.Ativo = atual.ID, atual.CreatedAt, in.Ativo
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarServico(ctx, tx, novo); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoServico, "protocolos_servico", novo.ID.String(), atual, novo, map[string]string{"id": novo.ID.String()})
	})
	return novo, err
}

// ListarServicosGestao devolve a carta da visão da pessoa, inclusive a inativa.
func (s *Service) ListarServicosGestao(ctx context.Context) ([]domain.Servico, error) {
	v, err := s.visao(ctx)
	if err != nil {
		return nil, err
	}
	lista, err := s.repo.ListarServicosDaGestao(ctx, s.db, v)
	if lista == nil {
		lista = []domain.Servico{}
	}
	return lista, err
}

// Carta lista os serviços ativos. Unidade vazia lista todos.
func (s *Service) Carta(ctx context.Context, unidade *uuid.UUID) ([]domain.Servico, error) {
	lista, err := s.repo.ListarServicos(ctx, s.db, true, unidade)
	if lista == nil {
		lista = []domain.Servico{}
	}
	// O aprovador é da mesa: a carta pública não o mostra.
	for i := range lista {
		lista[i].AprovadorID = nil
	}
	return lista, err
}

// ServicoPublico devolve um serviço ativo, com o formulário.
func (s *Service) ServicoPublico(ctx context.Context, id uuid.UUID) (domain.Servico, domain.Formulario, error) {
	serv, err := s.repo.BuscarServico(ctx, s.db, id)
	if err != nil {
		return domain.Servico{}, domain.Formulario{}, err
	}
	if !serv.Ativo {
		return domain.Servico{}, domain.Formulario{}, domain.ErrNaoEncontrado
	}
	form, err := s.repo.BuscarFormulario(ctx, s.db, serv.FormularioID)
	if err != nil {
		return domain.Servico{}, domain.Formulario{}, err
	}
	serv.AprovadorID = nil
	return serv, form, nil
}

// ListarResponsaveis devolve o cadastro municipal. O CPF sai mascarado.
// Quem não tem protocolos:gerir em nenhuma unidade não vê a lista.
func (s *Service) ListarResponsaveis(ctx context.Context) ([]domain.ResponsavelTecnico, error) {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return nil, apperrors.Unauthorized("sessão necessária")
	}
	c := auth.CoverageOf(id, auth.PermProtocolosGerir)
	if !c.All && len(c.Unidades) == 0 && len(c.Entidades) == 0 {
		return nil, apperrors.Forbidden("fora da unidade")
	}
	lista, err := s.repo.ListarResponsaveis(ctx, s.db)
	if err != nil {
		return nil, err
	}
	if lista == nil {
		lista = []domain.ResponsavelTecnico{}
	}
	for i := range lista {
		lista[i].CPF = domain.CPFMascarado(lista[i].CPF)
	}
	return lista, nil
}

// CadastrarResponsavel inclui o técnico que alguns serviços exigem.
func (s *Service) CadastrarResponsavel(ctx context.Context, nome, cpf string, unidade uuid.UUID) (domain.ResponsavelTecnico, error) {
	if err := s.exigir(ctx, auth.PermProtocolosGerir, unidade); err != nil {
		return domain.ResponsavelTecnico{}, err
	}
	r, err := domain.NovoResponsavel(nome, cpf)
	if err != nil {
		return domain.ResponsavelTecnico{}, err
	}
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InserirResponsavel(ctx, tx, r); err != nil {
			return err
		}
		return s.gravar(ctx, tx, "protocolos.responsavel.cadastrado", "protocolos_responsavel", r.ID.String(), nil, r, map[string]string{"id": r.ID.String()})
	})
	return r, err
}

// RegistrarFalta marca o horário do pedido e soma uma falta do CPF. A segunda chamada não soma.
func (s *Service) RegistrarFalta(ctx context.Context, pedido uuid.UUID) error {
	p, err := s.repo.BuscarPedido(ctx, s.db, pedido)
	if err != nil {
		return err
	}
	if err := s.exigir(ctx, auth.PermProtocolosTratar, p.UnidadeID); err != nil {
		return err
	}
	ag, err := s.repo.BuscarAgendamento(ctx, s.db, p.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return domain.ErrInvalido
		}
		return err
	}
	if ag.Status == domain.HorarioFalta {
		return domain.ErrFalta
	}
	if ag.Status != domain.HorarioAgendado && ag.Status != domain.HorarioConfirmado {
		return domain.ErrInvalido
	}
	ag.Status = domain.HorarioFalta
	quando := s.agora()
	return s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.AtualizarAgendamento(ctx, tx, ag); err != nil {
			return err
		}
		if err := s.repo.SomarFalta(ctx, tx, p.CPF, quando); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoFalta, "protocolos_falta", ag.ID.String(), nil, map[string]string{"quando": quando.Format(time.RFC3339)}, map[string]string{
			"fato": "falta registrada", "orientacao": "dois registros bloqueiam novos horários por 30 dias",
		})
	})
}
