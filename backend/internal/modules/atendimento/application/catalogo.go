package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Eventos das categorias do catálogo (ADR 078).
const (
	EventoCategoriaCriada     = "atendimento.categoria.criada"
	EventoCategoriaAtualizada = "atendimento.categoria.atualizada"
	EventoCategoriaApagada    = "atendimento.categoria.apagada"
)

// CategoriaEntrada são os campos de uma categoria. Ativa nil vale ativa na
// criação e conserva na edição.
type CategoriaEntrada struct {
	Nome, Descricao, Icone string
	Ordem                  int
	Ativa                  *bool
}

// gereCatalogo: a categoria é da instituição, então só quem gere o
// atendimento nela toda cria, altera e apaga.
func gereCatalogo(identity auth.Identity) error {
	if !ver(identity, nil) {
		return MapError(domain.ErrForaDoEscopo)
	}
	return nil
}

// CategoriasDoCatalogo devolve as categorias ativas, para a central.
func (s *Service) CategoriasDoCatalogo(ctx context.Context) ([]domain.Categoria, error) {
	lista, err := s.repo.ListCategorias(ctx, s.pool, true)
	return lista, MapError(err)
}

// ListarCategorias devolve todas, ativas ou não, para quem gere ofertas.
func (s *Service) ListarCategorias(ctx context.Context, identity auth.Identity) ([]domain.Categoria, error) {
	if _, ok := filtro(identity); !ok {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.ListCategorias(ctx, s.pool, false)
	return lista, MapError(err)
}

// CriarCategoria grava a categoria, o evento e a auditoria.
func (s *Service) CriarCategoria(ctx context.Context, identity auth.Identity, in CategoriaEntrada) (domain.Categoria, error) {
	if err := gereCatalogo(identity); err != nil {
		return domain.Categoria{}, err
	}
	c, err := domain.NovaCategoria(in.Nome, in.Descricao, in.Icone, in.Ordem, ativaDe(in.Ativa, true))
	if err != nil {
		return domain.Categoria{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if c, err = s.repo.InsertCategoria(ctx, tx, c); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCategoriaCriada, "categoria", c.ID, c, nil)
	})
	return c, MapError(err)
}

// AtualizarCategoria altera a categoria.
func (s *Service) AtualizarCategoria(ctx context.Context, identity auth.Identity, id uuid.UUID, in CategoriaEntrada) (domain.Categoria, error) {
	if err := gereCatalogo(identity); err != nil {
		return domain.Categoria{}, err
	}
	prev, err := s.repo.GetCategoria(ctx, s.pool, id)
	if err != nil {
		return domain.Categoria{}, MapError(err)
	}
	next, err := prev.Aplicar(in.Nome, in.Descricao, in.Icone, in.Ordem, ativaDe(in.Ativa, prev.Ativa))
	if err != nil {
		return domain.Categoria{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if next, err = s.repo.UpdateCategoria(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCategoriaAtualizada, "categoria", next.ID, next, prev)
	})
	return next, MapError(err)
}

// ApagarCategoria apaga a categoria; as ofertas dela vão para "Outros".
func (s *Service) ApagarCategoria(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if err := gereCatalogo(identity); err != nil {
		return err
	}
	prev, err := s.repo.GetCategoria(ctx, s.pool, id)
	if err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteCategoria(ctx, tx, id); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCategoriaApagada, "categoria", id, nil, prev)
	}))
}

// apresentar aplica categoria, ícone e ordem do bloco da oferta. Categoria
// nil conserva a atual; ponteiro para uuid.Nil tira da categoria.
func (s *Service) apresentar(ctx context.Context, o domain.Oferta, in OfertaEntrada) (domain.Oferta, error) {
	if in.CategoriaID != nil {
		o.CategoriaID = nil
		if *in.CategoriaID != uuid.Nil {
			if _, err := s.repo.GetCategoria(ctx, s.pool, *in.CategoriaID); err != nil {
				if errors.Is(err, domain.ErrNaoEncontrado) {
					return o, domain.ErrCategoriaInexistente
				}
				return o, err
			}
			id := *in.CategoriaID
			o.CategoriaID = &id
		}
	}
	if in.Icone != nil {
		o.Icone = *in.Icone
	}
	if in.Ordem != nil {
		o.Ordem = *in.Ordem
	}
	ic, err := domain.Apresentacao(o.Icone, o.Ordem)
	o.Icone = ic
	return o, err
}
