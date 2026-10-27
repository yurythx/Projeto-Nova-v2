// Package application contém os casos de uso do atendimento.
package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

const (
	EventoOfertaCriada     = "atendimento.oferta.criada"
	EventoOfertaAtualizada = "atendimento.oferta.atualizada"
	EventoEquipeCriada     = "atendimento.equipe.criada"
	EventoEquipeAtualizada = "atendimento.equipe.atualizada"
)

// Eventos grava o evento no outbox da transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// OfertaEntrada são os campos de uma oferta. Ativa nula vale true na
// criação e conserva o valor atual na edição; Tipo vazio vale requisição
// na criação e conserva o atual na edição.
type OfertaEntrada struct {
	Nome, Descricao, Tipo string
	ProdutoID, UnidadeID  *uuid.UUID
	Ativa                 *bool
	// Campos nil conserva os atuais na edição e vale nenhum na criação.
	Campos *[]domain.Campo
}

// EquipeEntrada são os campos de uma equipe.
type EquipeEntrada struct {
	Nome      string
	UnidadeID *uuid.UUID
	Membros   []uuid.UUID
	// OLAHoras é o prazo da equipe para as suas tarefas (ADR 070).
	OLAHoras *int
}

// Service implementa os casos de uso.
type Service struct {
	pool     *pgxpool.Pool
	repo     domain.Repository
	eventos  Eventos
	produtos domain.Produtos
	avisos   *notificacoes.Service // avisos da solicitação (WithNotificacoes)
	// tempoReal publica as mudanças para as telas abertas (WithTempoReal).
	tempoReal Publicador
	logger    *slog.Logger
	relogio   func() time.Time // nil = time.Now (WithRelogio)
	arquivos  *Armazenamento   // fotos e assinaturas (WithArmazenamento)
	caixa     mail.Caixa       // entrada de e-mail (WithCaixa); nil = desligada
	segredos  Segredos         // nome do segredo no cofre; nil recusa o vínculo
	passagens Passagens        // caminhos acrescentados na fila; nil = só o fluxo de fábrica
	// assistente é a IA de quem atende (WithAssistente, ADR 065); nil = sem IA.
	assistente Assistente
}

// Passagens lê os caminhos acrescentados no quadro.
type Passagens interface {
	Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error)
}

// NewService cria o serviço. produtos pode ser nil: oferta sem produto
// continua possível; informar produto_id devolve indisponível.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, produtos domain.Produtos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, produtos: produtos, logger: slog.Default()}
}

// Segredos confirma que um id do cofre existe no alcance de quem vincula.
// A resposta é o nome. A senha não atravessa esta porta.
type Segredos interface {
	Nome(ctx context.Context, identity auth.Identity, id uuid.UUID) (string, error)
}

// WithSegredo liga a porta do cofre.
func (s *Service) WithSegredo(segredos Segredos) *Service { s.segredos = segredos; return s }

// ComPassagens liga os caminhos acrescentados na fila.
func (s *Service) ComPassagens(p Passagens) { s.passagens = p }

func (s *Service) mapaFila(ctx context.Context, db database.DBTX) (map[string][]string, error) {
	if s.passagens == nil {
		return map[string][]string{}, nil
	}
	return s.passagens.Mapa(ctx, db, "fila")
}

// WithLogger troca o logger do serviço.
func (s *Service) WithLogger(l *slog.Logger) *Service {
	s.logger = l
	return s
}

// WithNotificacoes liga os avisos da solicitação. Sem ele, nada é avisado.
func (s *Service) WithNotificacoes(n *notificacoes.Service) *Service {
	s.avisos = n
	return s
}

// MapError traduz erros de domínio para a API e preserva a causa.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return causa(err, apperrors.NotFound("registro de atendimento não encontrado"))
	case errors.Is(err, domain.ErrSemIA):
		return causa(err, apperrors.FeatureDisabled("a IA do atendimento não está configurada"))
	case errors.Is(err, domain.ErrRespostaDaIA):
		return causa(err, apperrors.DependencyUnavailable("a IA não respondeu agora; tente de novo ou siga sem ela"))
	case errors.Is(err, domain.ErrSugestaoDecidida):
		return causa(err, apperrors.Conflict("esta sugestão já foi decidida"))
	case errors.Is(err, domain.ErrAguardandoAprovacao):
		return causa(err, apperrors.Conflict("o pedido ainda espera aprovação; ninguém o assume antes"))
	case errors.Is(err, domain.ErrEtapasSoRequisicao):
		return causa(err, apperrors.Validation("só oferta de requisição tem etapas de aprovação"))
	case errors.Is(err, domain.ErrEtapaInvalida):
		return causa(err, apperrors.Validation("até 5 etapas, cada uma com nome (até 80) e de 1 a 20 aprovadores ativos, sem repetir"))
	case errors.Is(err, domain.ErrSemAprovacaoPendente):
		return causa(err, apperrors.Conflict("este pedido não espera aprovação"))
	case errors.Is(err, domain.ErrNaoAprovador):
		return causa(err, apperrors.Forbidden("você não aprova a etapa atual deste pedido"))
	case errors.Is(err, domain.ErrAutoaprovacao):
		return causa(err, apperrors.Forbidden("quem pediu não aprova o próprio pedido"))
	case errors.Is(err, domain.ErrOLAInvalido):
		return causa(err, apperrors.Validation("o prazo da equipe vai de 1 a 8760 horas"))
	case errors.Is(err, domain.ErrFacilidadeInvalida):
		return causa(err, apperrors.Validation("a facilidade vai de 1 a 5"))
	case errors.Is(err, domain.ErrMetasInvalidas):
		return causa(err, apperrors.Validation("metas fora da faixa: nota e facilidade de 1 a 5, reabertura de 0 a 100%, idas e voltas de 0 a 20, espera de 1 a 8760 horas"))
	case errors.Is(err, domain.ErrTarefasSoRequisicao):
		return causa(err, apperrors.Validation("só oferta de requisição tem tarefas"))
	case errors.Is(err, domain.ErrModeloInvalido):
		return causa(err, apperrors.Validation("até 20 tarefas, cada uma com título (até 120) e uma equipe ao seu alcance"))
	case errors.Is(err, domain.ErrTarefaFeita):
		return causa(err, apperrors.Conflict("a tarefa já foi concluída ou dispensada"))
	case errors.Is(err, domain.ErrNaoEhDaEquipe):
		return causa(err, apperrors.Forbidden("só a equipe da tarefa a conclui, e só quem atende o chamado a dispensa"))
	case errors.Is(err, domain.ErrDispensaSemMotivo):
		return causa(err, apperrors.Validation("dispensar precisa de motivo (até 1000 caracteres)"))
	case errors.Is(err, domain.ErrTarefasPendentes):
		return causa(err, apperrors.Conflict("há tarefas pendentes; conclua ou dispense cada uma antes de dar o pedido por atendido"))
	case errors.Is(err, domain.ErrMotivoObrigatorio):
		return causa(err, apperrors.Validation("a recusa precisa de motivo (até 1000 caracteres)"))
	case errors.Is(err, domain.ErrEventosSemOferta):
		return causa(err, apperrors.Conflict("escolha, em Atendimento > Configuração, a oferta em que os alertas abrem incidente"))
	case errors.Is(err, domain.ErrAlertaInvalido):
		return causa(err, apperrors.Validation("alerta fora do formato do Alertmanager (status e fingerprint)"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe o nome (até 160 caracteres); a descrição cabe em 4000 e a equipe em 100 pessoas"))
	case errors.Is(err, domain.ErrForaDoEscopo):
		return causa(err, apperrors.Forbidden("você não gerencia o atendimento desta unidade"))
	case errors.Is(err, domain.ErrProdutoInexistente):
		return causa(err, apperrors.Validation("produto não encontrado"))
	case errors.Is(err, domain.ErrProdutoIndisponivel):
		return causa(err, apperrors.Validation("o cadastro de produtos está desligado"))
	case errors.Is(err, domain.ErrProdutoDeOutraUnidade):
		return causa(err, apperrors.Validation("o produto precisa ser da mesma unidade da oferta"))
	case errors.Is(err, domain.ErrMembroInvalido):
		return causa(err, apperrors.Validation("há um membro inexistente ou inativo"))
	case errors.Is(err, domain.ErrUnidadeInexistente):
		return causa(err, apperrors.Validation("unidade inexistente"))
	case errors.Is(err, domain.ErrTipoInvalido):
		return causa(err, apperrors.Validation("o tipo da oferta é incidente ou requisicao"))
	case errors.Is(err, domain.ErrOfertaInexistente):
		return causa(err, apperrors.Validation("oferta não encontrada"))
	case errors.Is(err, domain.ErrOfertaInativa):
		return causa(err, apperrors.Validation("esta oferta não está mais disponível"))
	case errors.Is(err, domain.ErrSolicitacaoInvalida):
		return causa(err, apperrors.Validation("informe o título (até 160 caracteres); a descrição cabe em 4000"))
	case errors.Is(err, domain.ErrSolicitanteInvalido):
		return causa(err, apperrors.Validation("seu usuário ainda não está cadastrado na plataforma"))
	case errors.Is(err, domain.ErrTransicaoInvalida):
		return causa(err, apperrors.Conflict("esta mudança de estado não é permitida agora"))
	case errors.Is(err, domain.ErrSemResponsavel):
		return causa(err, apperrors.Conflict("assuma a solicitação ou escolha um responsável antes de atender"))
	case errors.Is(err, domain.ErrResponsavelInvalido):
		return causa(err, apperrors.Validation("o responsável precisa ser membro ativo de uma equipe da unidade"))
	case errors.Is(err, domain.ErrSolicitacaoFinalizada):
		return causa(err, apperrors.Conflict("a solicitação já foi encerrada ou cancelada"))
	case errors.Is(err, domain.ErrCofreDesligado):
		return causa(err, apperrors.Validation("o cofre está desligado"))
	case errors.Is(err, domain.ErrSegredoAusente):
		return causa(err, apperrors.Validation("segredo não encontrado no seu cofre"))
	case errors.Is(err, domain.ErrComentarioInvalido):
		return causa(err, apperrors.Validation("escreva o comentário (até 4000 caracteres)"))
	case errors.Is(err, domain.ErrNotaInternaNegada):
		return causa(err, apperrors.Forbidden("só a equipe registra nota interna"))
	case errors.Is(err, domain.ErrAvaliacaoInvalida):
		return causa(err, apperrors.Validation("a nota vai de 1 a 5 e o comentário cabe em 4000 caracteres"))
	case errors.Is(err, domain.ErrAvaliacaoForaDeHora):
		return causa(err, apperrors.Conflict("só se avalia a solicitação resolvida, atendida ou encerrada"))
	case errors.Is(err, domain.ErrAvaliacaoAlheia):
		return causa(err, apperrors.Forbidden("só quem abriu a solicitação avalia o atendimento"))
	case errors.Is(err, domain.ErrAvaliacaoExistente):
		return causa(err, apperrors.Conflict("esta solicitação já foi avaliada"))
	case errors.Is(err, domain.ErrPoliticaInvalida):
		// O motivo é do domínio, escrito para quem cadastra.
		return causa(err, apperrors.Validation(strings.TrimPrefix(err.Error(), domain.ErrPoliticaInvalida.Error()+": ")))
	case errors.Is(err, domain.ErrFeriadoInvalido):
		return causa(err, apperrors.Validation("informe a data (AAAA-MM-DD), o nome (até 160 caracteres) e o tipo do feriado"))
	case errors.Is(err, domain.ErrFeriadoRepetido):
		return causa(err, apperrors.Conflict("já há feriado nesta data para a unidade"))
	case errors.Is(err, domain.ErrNivelInvalido):
		return causa(err, apperrors.Validation("impacto e urgência vão de 1 (baixo) a 3 (alto)"))
	case errors.Is(err, domain.ErrVisitaInvalida):
		return causa(err, apperrors.Validation("visita inválida: confira o identificador e o local (até 300 caracteres)"))
	case errors.Is(err, domain.ErrLocalizacao):
		return causa(err, apperrors.Validation("localização inválida (latitude, longitude, precisão ou raio do destino)"))
	case errors.Is(err, domain.ErrCapturaForaPrazo):
		return causa(err, apperrors.Validation("o horário registrado no aparelho está fora do aceito (até 7 dias de atraso)"))
	case errors.Is(err, domain.ErrVisitaAberta):
		return causa(err, apperrors.Conflict("você já tem uma visita em andamento nesta solicitação"))
	case errors.Is(err, domain.ErrVisitaEncerrada):
		return causa(err, apperrors.Conflict("a visita já foi encerrada ou cancelada"))
	case errors.Is(err, domain.ErrVisitaDeOutro):
		return causa(err, apperrors.Forbidden("só o técnico da visita pode alterá-la"))
	case errors.Is(err, domain.ErrEvidenciaInvalida):
		return causa(err, apperrors.Validation("a evidência é foto ou assinatura"))
	default:
		var campo domain.CampoInvalido
		if errors.As(err, &campo) {
			return causa(err, apperrors.Validation(campo.Msg))
		}
		var recusa domain.Recusa
		if errors.As(err, &recusa) {
			return causa(err, apperrors.Validation(recusa.Msg))
		}
		return err
	}
}

func aplicarCampos(o domain.Oferta, campos *[]domain.Campo) (domain.Oferta, error) {
	var lista []domain.Campo
	if campos != nil {
		lista = *campos
	}
	validos, err := domain.ValidarCampos(lista)
	if err != nil {
		return domain.Oferta{}, err
	}
	o.Campos = validos
	return o, nil
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

func ver(identity auth.Identity, unidade *uuid.UUID) bool {
	return auth.Can(identity, auth.PermAtendimentoManage, domain.Dona(unidade))
}

func filtro(identity auth.Identity) (domain.Filtro, bool) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoManage)
	if !cob.All && len(cob.Unidades) == 0 {
		return domain.Filtro{}, false
	}
	return domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades}, true
}

func ativaDe(v *bool, padrao bool) bool {
	if v == nil {
		return padrao
	}
	return *v
}

func tipoDe(v, padrao string) string {
	if v == "" {
		return padrao
	}
	return v
}

// ListarOfertas devolve as ofertas do escopo.
func (s *Service) ListarOfertas(ctx context.Context, identity auth.Identity) ([]domain.Oferta, error) {
	f, ok := filtro(identity)
	if !ok {
		return []domain.Oferta{}, nil
	}
	return s.repo.ListOfertas(ctx, s.pool, f)
}

// ObterOferta devolve a oferta, ou não encontrado fora do escopo.
func (s *Service) ObterOferta(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Oferta, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, id)
	if err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return domain.Oferta{}, MapError(domain.ErrNaoEncontrado)
	}
	return o, nil
}

// CriarOferta persiste a oferta, o evento e a auditoria na mesma transação.
func (s *Service) CriarOferta(ctx context.Context, identity auth.Identity, in OfertaEntrada) (domain.Oferta, error) {
	o, err := domain.NovaOferta(in.Nome, in.Descricao, tipoDe(in.Tipo, domain.TipoRequisicao), in.ProdutoID, in.UnidadeID, ativaDe(in.Ativa, true))
	if err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if o, err = aplicarCampos(o, in.Campos); err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return domain.Oferta{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, o.ProdutoID, o.UnidadeID); err != nil {
		return domain.Oferta{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if o, err = s.repo.InsertOferta(ctx, tx, o); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoOfertaCriada, "oferta", o.ID, o, nil)
	})
	return o, MapError(err)
}

// AtualizarOferta altera a oferta. Edição vazia não regrava.
func (s *Service) AtualizarOferta(ctx context.Context, identity auth.Identity, id uuid.UUID, in OfertaEntrada) (domain.Oferta, error) {
	prev, err := s.repo.GetOferta(ctx, s.pool, id)
	if err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if !ver(identity, prev.UnidadeID) {
		return domain.Oferta{}, MapError(domain.ErrNaoEncontrado)
	}
	next, err := prev.Aplicar(in.Nome, in.Descricao, tipoDe(in.Tipo, prev.Tipo), in.ProdutoID, in.UnidadeID, ativaDe(in.Ativa, prev.Ativa))
	if err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if in.Campos == nil {
		next.Campos = prev.Campos
	} else if next, err = aplicarCampos(next, in.Campos); err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if !ver(identity, next.UnidadeID) {
		return domain.Oferta{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, next.ProdutoID, next.UnidadeID); err != nil {
		return domain.Oferta{}, MapError(err)
	}
	if prev.Igual(next) {
		return prev, nil
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.UpdateOferta(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoOfertaAtualizada, "oferta", next.ID, next, prev)
	})
	return next, MapError(err)
}

// ListarEquipes devolve as equipes do escopo.
func (s *Service) ListarEquipes(ctx context.Context, identity auth.Identity) ([]domain.Equipe, error) {
	f, ok := filtro(identity)
	if !ok {
		return []domain.Equipe{}, nil
	}
	return s.repo.ListEquipes(ctx, s.pool, f)
}

// ObterEquipe devolve a equipe, ou não encontrado fora do escopo.
func (s *Service) ObterEquipe(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Equipe, error) {
	e, err := s.repo.GetEquipe(ctx, s.pool, id)
	if err != nil {
		return domain.Equipe{}, MapError(err)
	}
	if !ver(identity, e.UnidadeID) {
		return domain.Equipe{}, MapError(domain.ErrNaoEncontrado)
	}
	return e, nil
}

// CriarEquipe persiste a equipe, os membros, o evento e a auditoria na mesma transação.
func (s *Service) CriarEquipe(ctx context.Context, identity auth.Identity, in EquipeEntrada) (domain.Equipe, error) {
	e, err := domain.NovaEquipe(in.Nome, in.UnidadeID, in.Membros)
	if err == nil {
		err = domain.ValidarOLA(in.OLAHoras)
	}
	if err != nil {
		return domain.Equipe{}, MapError(err)
	}
	e.OLAHoras = in.OLAHoras
	if !ver(identity, e.UnidadeID) {
		return domain.Equipe{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirMembros(ctx, e.Membros); err != nil {
		return domain.Equipe{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if e, err = s.repo.InsertEquipe(ctx, tx, e); err != nil {
			return err
		}
		if err = s.repo.SubstituirMembros(ctx, tx, e.ID, e.Membros); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoEquipeCriada, "equipe", e.ID, e, nil)
	})
	return e, MapError(err)
}

// AtualizarEquipe altera a equipe e substitui os membros.
func (s *Service) AtualizarEquipe(ctx context.Context, identity auth.Identity, id uuid.UUID, in EquipeEntrada) (domain.Equipe, error) {
	prev, err := s.repo.GetEquipe(ctx, s.pool, id)
	if err != nil {
		return domain.Equipe{}, MapError(err)
	}
	if !ver(identity, prev.UnidadeID) {
		return domain.Equipe{}, MapError(domain.ErrNaoEncontrado)
	}
	next, err := prev.Aplicar(in.Nome, in.UnidadeID, in.Membros)
	if err == nil {
		err = domain.ValidarOLA(in.OLAHoras)
	}
	if err != nil {
		return domain.Equipe{}, MapError(err)
	}
	next.OLAHoras = in.OLAHoras
	if !ver(identity, next.UnidadeID) {
		return domain.Equipe{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirMembros(ctx, next.Membros); err != nil {
		return domain.Equipe{}, MapError(err)
	}
	if prev.Igual(next) {
		return prev, nil
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.UpdateEquipe(ctx, tx, next); err != nil {
			return err
		}
		if err = s.repo.SubstituirMembros(ctx, tx, next.ID, next.Membros); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoEquipeAtualizada, "equipe", next.ID, next, prev)
	})
	return next, MapError(err)
}

// Candidatos lista quem pode entrar na equipe da unidade.
func (s *Service) Candidatos(ctx context.Context, identity auth.Identity, unidade uuid.UUID) ([]domain.Candidato, error) {
	if !ver(identity, &unidade) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.Candidatos(ctx, s.pool, unidade)
	if err != nil {
		return nil, MapError(err)
	}
	if lista == nil {
		return []domain.Candidato{}, nil
	}
	return lista, nil
}

func (s *Service) conferirProduto(ctx context.Context, produto, unidade *uuid.UUID) error {
	if produto == nil {
		return nil
	}
	if s.produtos == nil {
		return domain.ErrProdutoIndisponivel
	}
	dona, err := s.produtos.Consultar(ctx, s.pool, *produto)
	if err != nil {
		return err
	}
	if !domain.MesmaUnidade(dona, unidade) {
		return domain.ErrProdutoDeOutraUnidade
	}
	return nil
}

func (s *Service) conferirMembros(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	n, err := s.repo.ContarAtivos(ctx, s.pool, ids)
	if err != nil {
		return err
	}
	if n != len(ids) {
		return domain.ErrMembroInvalido
	}
	return nil
}

func (s *Service) emitir(ctx context.Context, tx pgx.Tx, evento, recurso string, id uuid.UUID, atual, antes any) error {
	return s.registrar(ctx, tx, evento, recurso, id, map[string]any{"id": id.String()}, atual, antes)
}

// registrar grava o evento no outbox e a auditoria, na transação de negócio.
func (s *Service) registrar(ctx context.Context, tx pgx.Tx, evento, recurso string, id uuid.UUID, payload, atual, antes any) error {
	if err := s.eventos.Write(ctx, tx, evento, recurso, id.String(), uuid.Nil, payload); err != nil {
		return err
	}
	var before any
	if antes != nil {
		before = antes
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, recurso, id.String(), before, atual))
}
