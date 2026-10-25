package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas/domain"
)

// ConcluirPorMudanca resolve o problema na transação da mudança.
// Já resolvido ou encerrado não é erro. Sem a passagem direta para
// resolvido, ou sem causa raiz, a mudança inteira recua.
func (s *Service) ConcluirPorMudanca(ctx context.Context, tx pgx.Tx, id, autor uuid.UUID, texto string) error {
	if err := s.repo.Travar(ctx, tx, id); err != nil {
		return err
	}
	p, err := s.repo.Get(ctx, tx, id)
	if err != nil {
		return err
	}
	if p.Estado == domain.Resolvido || p.Estado == domain.Encerrado {
		return nil
	}
	next, _, err := p.Transicionar(domain.Resolvido)
	if err != nil {
		return &domain.NaoConcluivel{Numero: p.Numero, Estado: p.Estado, Causa: err}
	}
	if err := s.repo.Update(ctx, tx, next); err != nil {
		return err
	}
	extra := map[string]any{"de": p.Estado, "origem": "mudanca", "autor_id": autor.String()}
	if texto != "" {
		extra["texto"] = texto
	}
	return s.registrar(ctx, tx, EventoEstado, next, p, next, extra)
}
