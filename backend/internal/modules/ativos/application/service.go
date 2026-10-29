// Package application contém os casos de uso do ativo.
package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	EventoCriado     = "ativos.ativo.criado"
	EventoAtualizado = "ativos.ativo.atualizado"
)

// Eventos grava o evento no outbox da transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Service implementa os casos de uso.
type Service struct {
	pool      *pgxpool.Pool
	repo      domain.Repository
	eventos   Eventos
	produtos  domain.Produtos
	passagens Passagens
}

// Passagens lê os caminhos acrescentados no quadro. Nil mantém só os estados de fábrica.
type Passagens interface {
	Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error)
}

// NewService cria o serviço. produtos nil: ativo sem produto continua
// possível; informar produto devolve indisponível.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, produtos domain.Produtos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, produtos: produtos}
}

// ComPassagens liga os caminhos acrescentados no quadro de ativos.
func (s *Service) ComPassagens(p Passagens) { s.passagens = p }

func (s *Service) mapa(ctx context.Context, db database.DBTX) (map[string][]string, error) {
	if s.passagens == nil {
		return map[string][]string{}, nil
	}
	return s.passagens.Mapa(ctx, db, "ativos")
}

// MapError traduz erros de domínio para a API e preserva a causa.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return causa(err, apperrors.NotFound("ativo não encontrado"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe o nome (até 160 caracteres); identificador até 80 e descrição até 4000"))
	case errors.Is(err, domain.ErrTipoInvalido):
		return causa(err, apperrors.Validation("tipo de ativo desconhecido"))
	case errors.Is(err, domain.ErrEstadoInvalido):
		return causa(err, apperrors.Validation("estado de ativo desconhecido"))
	case errors.Is(err, domain.ErrForaDoEscopo):
		return causa(err, apperrors.Forbidden("você não gerencia ativos desta unidade"))
	case errors.Is(err, domain.ErrUnidadeInexistente):
		return causa(err, apperrors.Validation("unidade inexistente"))
	case errors.Is(err, domain.ErrIdentificadorRepetido):
		return causa(err, apperrors.Conflict("já há ativo com este identificador"))
	case errors.Is(err, domain.ErrCadastroManual):
		return causa(err, apperrors.Conflict("esse patrimônio foi cadastrado à mão e o agente não o altera"))
	case errors.Is(err, domain.ErrProdutoInexistente):
		return causa(err, apperrors.Validation("produto não encontrado"))
	case errors.Is(err, domain.ErrProdutoIndisponivel):
		return causa(err, apperrors.Validation("o cadastro de produtos está desligado"))
	case errors.Is(err, domain.ErrProdutoDeOutraUnidade):
		return causa(err, apperrors.Validation("o produto precisa ser da mesma unidade do ativo"))
	default:
		var relacao domain.RelacaoInvalida
		if errors.As(err, &relacao) {
			return causa(err, apperrors.Validation(relacao.Msg))
		}
		return err
	}
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

var permissoesLeituraAtivos = []auth.Permission{
	auth.PermAtivosManage,
	auth.PermAtivosRead,
	auth.PermAtendimentoAtender,
	auth.PermProblemasManage,
	auth.PermMudancasManage,
	auth.PermProjetosRead,
	auth.PermProjetosManage,
}

func pode(identity auth.Identity, unidade *uuid.UUID) bool {
	return auth.Can(identity, auth.PermAtivosManage, domain.Dona(unidade))
}

func le(identity auth.Identity, unidade *uuid.UUID) bool {
	alvo := domain.Dona(unidade)
	for _, perm := range permissoesLeituraAtivos {
		if auth.Can(identity, perm, alvo) {
			return true
		}
	}
	return false
}

func coberturaLeitura(identity auth.Identity) (bool, []uuid.UUID, bool) {
	var unidades []uuid.UUID
	visto := map[uuid.UUID]struct{}{}
	for _, perm := range permissoesLeituraAtivos {
		cob := auth.CoverageOf(identity, perm)
		if cob.All {
			return true, nil, true
		}
		for _, u := range cob.Unidades {
			if _, ok := visto[u]; !ok {
				visto[u] = struct{}{}
				unidades = append(unidades, u)
			}
		}
	}
	return false, unidades, len(unidades) > 0
}

// Listar devolve os ativos do escopo; com produto, só os que o sustentam.
func (s *Service) Listar(ctx context.Context, identity auth.Identity, produto *uuid.UUID) ([]domain.Ativo, error) {
	all, unidades, ok := coberturaLeitura(identity)
	if !ok {
		return []domain.Ativo{}, nil
	}
	lista, err := s.repo.List(ctx, s.pool, domain.Filtro{Restrito: !all, Unidades: unidades, ProdutoID: produto})
	return lista, MapError(err)
}

// Obter devolve o ativo, ou não encontrado fora do escopo.
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Ativo, error) {
	a, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if !le(identity, a.UnidadeID) {
		return domain.Ativo{}, MapError(domain.ErrNaoEncontrado)
	}
	return a, nil
}

// Criar persiste o ativo, o evento e a auditoria na mesma transação.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, in domain.Entrada) (domain.Ativo, error) {
	mapa, err := s.mapa(ctx, s.pool)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	a, err := domain.Novo(in, domain.EstadosLivres(mapa)...)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if !domain.PassagemDeAtivo("", a.Estado, mapa) {
		return domain.Ativo{}, MapError(domain.ErrEstadoInvalido)
	}
	if !pode(identity, a.UnidadeID) {
		return domain.Ativo{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, a.ProdutoID, a.UnidadeID); err != nil {
		return domain.Ativo{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if a, err = s.repo.Insert(ctx, tx, a); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCriado, a, nil)
	})
	return a, MapError(err)
}

// Atualizar altera o ativo. Edição vazia não regrava.
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, in domain.Entrada) (domain.Ativo, error) {
	prev, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if !pode(identity, prev.UnidadeID) {
		return domain.Ativo{}, MapError(domain.ErrNaoEncontrado)
	}
	mapa, err := s.mapa(ctx, s.pool)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	next, err := prev.Aplicar(in, domain.EstadosLivres(mapa)...)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if !domain.PassagemDeAtivo(prev.Estado, next.Estado, mapa) {
		return domain.Ativo{}, MapError(domain.ErrEstadoInvalido)
	}
	if !pode(identity, next.UnidadeID) {
		return domain.Ativo{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, next.ProdutoID, next.UnidadeID); err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if prev.Igual(next) {
		return prev, nil
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoAtualizado, next, &prev)
	})
	return next, MapError(err)
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

// Inventariar grava o relato da própria máquina. O mesmo relato não
// reabre auditoria. Cadastro manual não é tocado.
func (s *Service) Inventariar(ctx context.Context, identity auth.Identity, relato domain.Relato) (domain.Ativo, error) {
	unidade, err := unidadeDoRelato(identity, relato.UnidadeID)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	relato.UnidadeID = unidade
	if _, err := domain.ValidarInventario(relato.Inventario); err != nil {
		return domain.Ativo{}, MapError(err)
	}
	atual, err := s.repo.PorIdentificador(ctx, s.pool, strings.TrimSpace(relato.Identificador))
	if errors.Is(err, domain.ErrNaoEncontrado) {
		atualPtr := (*domain.Ativo)(nil)
		proximo, _, err := domain.AplicarRelato(atualPtr, relato)
		if err != nil {
			return domain.Ativo{}, MapError(err)
		}
		err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
			var err error
			if proximo, err = s.repo.Insert(ctx, tx, proximo); err != nil {
				return err
			}
			return s.emitir(ctx, tx, "ativos.inventario.criado", proximo, nil)
		})
		return proximo, MapError(err)
	}
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	proximo, mudou, err := domain.AplicarRelato(&atual, relato)
	if err != nil || !mudou {
		return atual, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if proximo, err = s.repo.Update(ctx, tx, proximo); err != nil {
			return err
		}
		return s.emitir(ctx, tx, "ativos.inventario.atualizado", proximo, &atual)
	})
	return proximo, MapError(err)
}

func unidadeDoRelato(identity auth.Identity, informada uuid.UUID) (uuid.UUID, error) {
	if informada != uuid.Nil {
		if !auth.Can(identity, auth.PermAtivosReportar, auth.InUnidade(informada)) {
			return uuid.Nil, domain.ErrForaDoEscopo
		}
		return informada, nil
	}
	cob := auth.CoverageOf(identity, auth.PermAtivosReportar)
	if cob.All || len(cob.Unidades) != 1 {
		return uuid.Nil, domain.ErrEntradaInvalida
	}
	return cob.Unidades[0], nil
}

func (s *Service) emitir(ctx context.Context, tx pgx.Tx, evento string, atual domain.Ativo, antes *domain.Ativo) error {
	payload := map[string]any{"id": atual.ID.String(), "nome": atual.Nome, "estado": atual.Estado}
	if err := s.eventos.Write(ctx, tx, evento, "ativo", atual.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	var before any
	if antes != nil {
		before = antes
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, "ativo", atual.ID.String(), before, atual))
}

// ListarRelacoes devolve o que o ativo aponta.
func (s *Service) ListarRelacoes(ctx context.Context, identity auth.Identity, origem uuid.UUID) ([]domain.Relacao, error) {
	if _, err := s.Obter(ctx, identity, origem); err != nil {
		return nil, err
	}
	lista, err := s.repo.ListRelacoes(ctx, s.pool, origem)
	return lista, MapError(err)
}

// Relacionar liga dois ativos do escopo. Dependência em ciclo é recusada.
func (s *Service) Relacionar(ctx context.Context, identity auth.Identity, origem, destino uuid.UUID, tipo string) (domain.Relacao, error) {
	if err := domain.ValidarRelacao(origem, destino, tipo); err != nil {
		return domain.Relacao{}, MapError(err)
	}
	aOrig, err := s.Obter(ctx, identity, origem)
	if err != nil {
		return domain.Relacao{}, err
	}
	if !pode(identity, aOrig.UnidadeID) {
		return domain.Relacao{}, MapError(domain.ErrNaoEncontrado)
	}
	aDest, err := s.Obter(ctx, identity, destino)
	if err != nil {
		return domain.Relacao{}, err
	}
	if !pode(identity, aDest.UnidadeID) {
		return domain.Relacao{}, MapError(domain.ErrNaoEncontrado)
	}
	if tipo == "depende" {
		ciclo, err := s.repo.FechaCiclo(ctx, s.pool, origem, destino)
		if err != nil {
			return domain.Relacao{}, MapError(err)
		}
		if ciclo {
			return domain.Relacao{}, MapError(domain.RelacaoInvalida{Msg: "essa dependência fecharia um ciclo"})
		}
	}
	item := domain.Relacao{ID: uuid.New(), OrigemID: origem, DestinoID: destino, Tipo: tipo}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InsertRelacao(ctx, tx, item); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: "ativos.relacao.criada", ResourceType: "ativo_relacao", ResourceID: item.ID.String()})
	})
	return item, MapError(err)
}

// Desrelacionar remove o apontamento.
func (s *Service) Desrelacionar(ctx context.Context, identity auth.Identity, origem, id uuid.UUID) error {
	aOrig, err := s.Obter(ctx, identity, origem)
	if err != nil {
		return err
	}
	if !pode(identity, aOrig.UnidadeID) {
		return MapError(domain.ErrNaoEncontrado)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteRelacao(ctx, tx, id, origem); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: "ativos.relacao.excluida", ResourceType: "ativo_relacao", ResourceID: id.String()})
	}))
}
