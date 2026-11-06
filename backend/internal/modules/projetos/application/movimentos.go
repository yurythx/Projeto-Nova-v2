package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func mesmoUUID(a, b *uuid.UUID) bool {
	if a == nil || *a == uuid.Nil {
		return b == nil || *b == uuid.Nil
	}
	return b != nil && *a == *b
}

func mudouTrilha(antes, depois domain.Tarefa) bool {
	ea, ed := antes.Estado, depois.Estado
	if ea == "" {
		ea = domain.EstadoTarefaAFazer
	}
	if ed == "" {
		ed = domain.EstadoTarefaAFazer
	}
	if ea != ed {
		return true
	}
	return !mesmoUUID(antes.ResponsavelID, depois.ResponsavelID) || antes.ResponsavelNome != depois.ResponsavelNome
}

func (s *Service) anotarMovimento(ctx context.Context, tx pgx.Tx, idt auth.Identity, antes, depois domain.Tarefa) error {
	if !mudouTrilha(antes, depois) {
		return nil
	}
	autorNome := strings.TrimSpace(idt.Name)
	if autorNome == "" {
		autorNome = idt.Username
	}
	var autorID *uuid.UUID
	if idt.UserID != uuid.Nil {
		id := idt.UserID
		autorID = &id
	}
	m := &domain.Movimento{
		ID: uuid.New(), TarefaID: depois.ID,
		EstadoAntes: antes.Estado, EstadoDepois: depois.Estado,
		ResponsavelAntesID: antes.ResponsavelID, ResponsavelAntesNome: antes.ResponsavelNome,
		ResponsavelDepoisID: depois.ResponsavelID, ResponsavelDepoisNome: depois.ResponsavelNome,
		AutorID: autorID, AutorNome: autorNome, CreatedAt: time.Now().UTC(),
	}
	if antes.Estado == "" {
		m.EstadoAntes = domain.EstadoTarefaAFazer
	}
	if err := s.repo.RegistrarMovimento(ctx, tx, m); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.tarefa.moved", "projeto_tarefa", depois.ID.String(), antes, depois))
}

func (s *Service) ListarMovimentos(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID) ([]domain.Movimento, error) {
	_, projeto, err := s.tarefaNoProjeto(ctx, s.pool, projetoID, tarefaID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	return s.repo.ListarMovimentos(ctx, s.pool, tarefaID)
}
