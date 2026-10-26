package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (s *Service) autorizaRevelar(ctx context.Context, tx pgx.Tx, identity auth.Identity, item domain.Selado) error {
	if item.PastaID != nil && !pode(identity, auth.PermCofreManage, item.UnidadeID) {
		membro, err := s.repo.Membro(ctx, tx, *item.PastaID, identity.UserID)
		if err != nil {
			return domain.ErrNaoEncontrado
		}
		if !membro.PodeRevelar || !pode(identity, auth.PermCofreReveal, item.UnidadeID) {
			return domain.ErrSemPermissao
		}
		return nil
	}
	if !pode(identity, auth.PermCofreReveal, item.UnidadeID) && !pode(identity, auth.PermCofreManage, item.UnidadeID) {
		return domain.ErrSemPermissao
	}
	return nil
}

func (s *Service) conferirPasta(ctx context.Context, db database.DBTX, id uuid.UUID, unidade *uuid.UUID) error {
	pasta, err := s.repo.ObterPasta(ctx, db, id)
	if err != nil {
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return domain.EntradaInvalida{Msg: "a pasta não existe"}
		}
		return err
	}
	if !mesmaUnidade(pasta.UnidadeID, unidade) {
		return domain.EntradaInvalida{Msg: "a pasta precisa ser da mesma unidade do segredo"}
	}
	return nil
}

// ListarPastas devolve as pastas do alcance. Quem não administra não vê os membros.
func (s *Service) ListarPastas(ctx context.Context, identity auth.Identity) ([]domain.Pasta, error) {
	f, err := s.filtro(identity)
	if err != nil {
		return []domain.Pasta{}, nil
	}
	var lista []domain.Pasta
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		lista, err = s.repo.ListarPastas(ctx, tx, f)
		return err
	})
	if err != nil {
		return nil, MapError(err)
	}
	for i := range lista {
		if !pode(identity, auth.PermCofreManage, lista[i].UnidadeID) {
			lista[i].Membros = nil
		}
	}
	return lista, nil
}

// CriarPasta grava a pasta e quem entra nela.
func (s *Service) CriarPasta(ctx context.Context, identity auth.Identity, nome string, unidade *uuid.UUID, membros []domain.Membro) (domain.Pasta, error) {
	if !pode(identity, auth.PermCofreManage, unidade) {
		return domain.Pasta{}, MapError(domain.ErrForaDoEscopo)
	}
	pasta, err := domain.NovaPasta(nome, unidade, membros)
	if err != nil {
		return domain.Pasta{}, MapError(err)
	}
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.conferirMembros(ctx, tx, pasta.Membros); err != nil {
			return err
		}
		if err := s.repo.InserirPasta(ctx, tx, pasta); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, "cofre.pasta.criada", pasta.ID, pista(pasta.ID, pasta.UnidadeID))
	})
	return pasta, MapError(err)
}

// DefinirMembros troca quem entra na pasta.
func (s *Service) DefinirMembros(ctx context.Context, identity auth.Identity, id uuid.UUID, membros []domain.Membro) (domain.Pasta, error) {
	limpos, err := domain.MembrosDe(membros)
	if err != nil {
		return domain.Pasta{}, MapError(err)
	}
	var pasta domain.Pasta
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterPasta(ctx, tx, id)
		if err != nil {
			return err
		}
		if !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrNaoEncontrado
		}
		if err := s.conferirMembros(ctx, tx, limpos); err != nil {
			return err
		}
		if err := s.repo.SubstituirMembros(ctx, tx, id, limpos); err != nil {
			return err
		}
		atual.Membros = limpos
		pasta = atual
		return s.trilha.Registrar(ctx, tx, "cofre.pasta.membros", id, pista(id, atual.UnidadeID))
	})
	return pasta, MapError(err)
}

// ExcluirPasta só apaga a pasta vazia. Senha dentro dela impede a exclusão.
func (s *Service) ExcluirPasta(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return MapError(s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterPasta(ctx, tx, id)
		if err != nil {
			return err
		}
		if !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrNaoEncontrado
		}
		n, err := s.repo.ContarNaPasta(ctx, tx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return domain.EntradaInvalida{Msg: "a pasta ainda tem senha; esvazie antes de apagar"}
		}
		if err := s.repo.ExcluirPasta(ctx, tx, id); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, "cofre.pasta.excluida", id, pista(id, atual.UnidadeID))
	}))
}

// Pessoas lista quem pode entrar na pasta, no máximo cem.
func (s *Service) Pessoas(ctx context.Context, identity auth.Identity, unidade *uuid.UUID) ([]domain.Pessoa, error) {
	if !pode(identity, auth.PermCofreManage, unidade) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	var lista []domain.Pessoa
	err := s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		lista, err = s.repo.Pessoas(ctx, tx, unidade)
		return err
	})
	return lista, MapError(err)
}

func (s *Service) conferirMembros(ctx context.Context, db database.DBTX, membros []domain.Membro) error {
	if len(membros) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(membros))
	for _, m := range membros {
		ids = append(ids, m.UsuarioID)
	}
	ok, err := s.repo.UsuariosExistem(ctx, db, ids)
	if err != nil {
		return err
	}
	if !ok {
		return domain.EntradaInvalida{Msg: "há uma pessoa que não está cadastrada"}
	}
	return nil
}
