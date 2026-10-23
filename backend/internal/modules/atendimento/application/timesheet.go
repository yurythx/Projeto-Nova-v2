package application

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// ListarApontamentos devolve o resumo de horas e todos os lançamentos da solicitação.
func (s *Service) ListarApontamentos(ctx context.Context, identity auth.Identity, solicitacaoID uuid.UUID) (domain.ResumoApontamentos, error) {
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, solicitacaoID)
	if err != nil {
		return domain.ResumoApontamentos{}, MapError(err)
	}
	p := papeis(identity, sol)
	if len(p) == 0 {
		return domain.ResumoApontamentos{}, MapError(domain.ErrNaoEncontrado)
	}

	lista, err := s.repo.ListApontamentos(ctx, s.pool, solicitacaoID)
	if err != nil {
		return domain.ResumoApontamentos{}, MapError(err)
	}

	var total float64
	for _, a := range lista {
		total += a.Horas
	}

	return domain.ResumoApontamentos{
		TotalHoras:   total,
		Apontamentos: lista,
	}, nil
}

// RegistrarApontamento insere um novo apontamento de horas de um técnico.
func (s *Service) RegistrarApontamento(ctx context.Context, identity auth.Identity, solicitacaoID uuid.UUID, entrada domain.EntradaApontamento) (domain.Apontamento, error) {
	if err := entrada.Validar(); err != nil {
		return domain.Apontamento{}, MapError(err)
	}

	sol, err := s.repo.GetSolicitacao(ctx, s.pool, solicitacaoID)
	if err != nil {
		return domain.Apontamento{}, MapError(err)
	}
	p := papeis(identity, sol)
	if !slices.Contains(p, domain.PapelTecnico) {
		return domain.Apontamento{}, MapError(domain.ErrForaDoEscopo)
	}

	nome := identity.Username
	if nome == "" {
		nome = "Técnico"
	}

	dataApontamento := entrada.DataApontamento
	if dataApontamento == "" {
		dataApontamento = time.Now().Format("2006-01-02")
	}

	apontamento := domain.Apontamento{
		ID:              uuid.New(),
		SolicitacaoID:   solicitacaoID,
		UsuarioID:       identity.UserID,
		UsuarioNome:     nome,
		DataApontamento: dataApontamento,
		Horas:           entrada.Horas,
		Descricao:       entrada.Descricao,
		CreatedAt:       time.Now().UTC(),
	}

	var criado domain.Apontamento
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		criado, err = s.repo.InsertApontamento(ctx, tx, apontamento)
		return err
	})
	if err != nil {
		return domain.Apontamento{}, MapError(err)
	}

	return criado, nil
}

// ExcluirApontamento remove um apontamento de horas da solicitação.
func (s *Service) ExcluirApontamento(ctx context.Context, identity auth.Identity, solicitacaoID, apontamentoID uuid.UUID) error {
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, solicitacaoID)
	if err != nil {
		return MapError(err)
	}
	p := papeis(identity, sol)
	if !slices.Contains(p, domain.PapelTecnico) {
		return MapError(domain.ErrForaDoEscopo)
	}

	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return s.repo.DeleteApontamento(ctx, tx, solicitacaoID, apontamentoID)
	})
}
