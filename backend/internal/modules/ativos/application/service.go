// Package application contém os casos de uso do ativo.
package application

import (
	"context"
	"errors"

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
	pool     *pgxpool.Pool
	repo     domain.Repository
	eventos  Eventos
	produtos domain.Produtos
}

// NewService cria o serviço. produtos nil: ativo sem produto continua
// possível; informar produto devolve indisponível.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, produtos domain.Produtos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, produtos: produtos}
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
	case errors.Is(err, domain.ErrProdutoInexistente):
		return causa(err, apperrors.Validation("produto não encontrado"))
	case errors.Is(err, domain.ErrProdutoIndisponivel):
		return causa(err, apperrors.Validation("o cadastro de produtos está desligado"))
	case errors.Is(err, domain.ErrProdutoDeOutraUnidade):
		return causa(err, apperrors.Validation("o produto precisa ser da mesma unidade do ativo"))
	default:
		return err
	}
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

func pode(identity auth.Identity, unidade *uuid.UUID) bool {
	return auth.Can(identity, auth.PermAtivosManage, domain.Dona(unidade))
}

// Listar devolve os ativos do escopo; com produto, só os que o sustentam.
func (s *Service) Listar(ctx context.Context, identity auth.Identity, produto *uuid.UUID) ([]domain.Ativo, error) {
	cob := auth.CoverageOf(identity, auth.PermAtivosManage)
	if !cob.All && len(cob.Unidades) == 0 {
		return []domain.Ativo{}, nil
	}
	lista, err := s.repo.List(ctx, s.pool, domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades, ProdutoID: produto})
	return lista, MapError(err)
}

// Obter devolve o ativo, ou não encontrado fora do escopo.
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Ativo, error) {
	a, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Ativo{}, MapError(err)
	}
	if !pode(identity, a.UnidadeID) {
		return domain.Ativo{}, MapError(domain.ErrNaoEncontrado)
	}
	return a, nil
}

// Criar persiste o ativo, o evento e a auditoria na mesma transação.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, in domain.Entrada) (domain.Ativo, error) {
	a, err := domain.Novo(in)
	if err != nil {
		return domain.Ativo{}, MapError(err)
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
	next, err := prev.Aplicar(in)
	if err != nil {
		return domain.Ativo{}, MapError(err)
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

func (s *Service) emitir(ctx context.Context, tx pgx.Tx, evento string, atual domain.Ativo, antes *domain.Ativo) error {
	payload := map[string]any{"id": atual.ID.String(), "nome": atual.Nome, "estado": atual.Estado}
	if err := s.eventos.Write(ctx, tx, evento, "ativo", atual.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	var before any
	if antes != nil {
		before = *antes
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, "ativo", atual.ID.String(), before, atual))
}
