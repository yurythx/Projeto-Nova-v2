// Package application contém os casos de uso do produto digital.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/produtos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	EventoCriado     = "produtos.produto.criado"
	EventoAtualizado = "produtos.produto.atualizado"
	EventoEstagio    = "produtos.produto.estagio_alterado"
)

// Eventos grava o evento no outbox da transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Entrada são os campos editáveis de um produto. O estágio inicial entra
// na criação; depois disso só muda por MudarEstagio.
type Entrada struct {
	Nome, Descricao, Estagio, Utilidade, Garantia string
	UnidadeID                                     *uuid.UUID
}

// Service implementa os casos de uso.
type Service struct {
	pool    *pgxpool.Pool
	repo    domain.Repository
	eventos Eventos
}

// NewService cria o serviço.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos}
}

// MapError traduz erros de domínio para a API e preserva a causa, para
// o chamador ainda reconhecer o erro de domínio.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return causa(err, apperrors.NotFound("produto não encontrado"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe o nome (até 160 caracteres); os textos longos cabem em 4000"))
	case errors.Is(err, domain.ErrEstagioInvalido):
		return causa(err, apperrors.Validation("estágio fora do ciclo de vida do produto"))
	case errors.Is(err, domain.ErrForaDoEscopo):
		return causa(err, apperrors.Forbidden("você não gerencia produtos desta unidade"))
	case errors.Is(err, domain.ErrUnidadeInexistente):
		return causa(err, apperrors.Validation("unidade inexistente"))
	default:
		return err
	}
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

var permissoesLeituraProdutos = []auth.Permission{
	auth.PermProdutosManage,
	auth.PermProdutosRead,
	auth.PermAtendimentoManage,
	auth.PermAtivosManage,
	auth.PermProblemasManage,
	auth.PermMudancasManage,
	auth.PermProjetosRead,
	auth.PermProjetosManage,
}

func leProduto(identity auth.Identity, unidade *uuid.UUID) bool {
	alvo := domain.Dona(unidade)
	for _, perm := range permissoesLeituraProdutos {
		if auth.Can(identity, perm, alvo) {
			return true
		}
	}
	return false
}

func coberturaLeituraProdutos(identity auth.Identity) (bool, []uuid.UUID, bool) {
	var unidades []uuid.UUID
	visto := map[uuid.UUID]struct{}{}
	for _, perm := range permissoesLeituraProdutos {
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

// Listar devolve os produtos que a identidade gerencia ou consulta. Quem só
// tem a permissão num departamento, ou em lugar nenhum, recebe lista vazia:
// o produto pertence à unidade.
func (s *Service) Listar(ctx context.Context, identity auth.Identity) ([]domain.Produto, error) {
	all, unidades, ok := coberturaLeituraProdutos(identity)
	if !ok {
		return []domain.Produto{}, nil
	}
	return s.repo.List(ctx, s.pool, domain.Filtro{Restrito: !all, Unidades: unidades})
}

// Obter devolve o produto. Fora do escopo a resposta é a mesma de quem
// não existe (ADR 013: 404 para ler).
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Produto, error) {
	p, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !leProduto(identity, p.UnidadeID) {
		return domain.Produto{}, MapError(domain.ErrNaoEncontrado)
	}
	return p, nil
}

// Criar persiste o produto, o evento e a auditoria na mesma transação.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, in Entrada) (domain.Produto, error) {
	p, err := domain.Novo(in.Nome, in.Descricao, in.Estagio, in.Utilidade, in.Garantia, in.UnidadeID)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !auth.Can(identity, auth.PermProdutosManage, domain.Dona(p.UnidadeID)) {
		return domain.Produto{}, MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if p, err = s.repo.Insert(ctx, tx, p); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCriado, p, domain.Produto{})
	})
	return p, MapError(err)
}

// Atualizar altera nome, textos e unidade. O estágio permanece.
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, in Entrada) (domain.Produto, error) {
	prev, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !auth.Can(identity, auth.PermProdutosManage, domain.Dona(prev.UnidadeID)) {
		return domain.Produto{}, MapError(domain.ErrNaoEncontrado)
	}
	next, err := prev.Aplicar(in.Nome, in.Descricao, in.Utilidade, in.Garantia, in.UnidadeID)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !auth.Can(identity, auth.PermProdutosManage, domain.Dona(next.UnidadeID)) {
		return domain.Produto{}, MapError(domain.ErrForaDoEscopo)
	}
	if prev.Igual(next) {
		return prev, nil
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoAtualizado, next, prev)
	})
	return next, MapError(err)
}

// MudarEstagio avança ou recua o ciclo. O mesmo estágio não regrava.
func (s *Service) MudarEstagio(ctx context.Context, identity auth.Identity, id uuid.UUID, estagio string) (domain.Produto, error) {
	prev, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !auth.Can(identity, auth.PermProdutosManage, domain.Dona(prev.UnidadeID)) {
		return domain.Produto{}, MapError(domain.ErrNaoEncontrado)
	}
	next, mudou, err := prev.ComEstagio(estagio)
	if err != nil {
		return domain.Produto{}, MapError(err)
	}
	if !mudou {
		return prev, nil
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoEstagio, next, prev)
	})
	return next, MapError(err)
}

func (s *Service) emitir(ctx context.Context, tx pgx.Tx, evento string, atual, antes domain.Produto) error {
	payload := map[string]any{"id": atual.ID.String(), "nome": atual.Nome, "estagio": atual.Estagio}
	if err := s.eventos.Write(ctx, tx, evento, "produto", atual.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, "produto", atual.ID.String(), antesOuNil(antes), atual))
}

func antesOuNil(p domain.Produto) any {
	if p.ID == uuid.Nil {
		return nil
	}
	return p
}
