package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	EventoSlasCriada     = "atendimento.sla.criada"
	EventoSlasAtualizada = "atendimento.sla.atualizada"
	EventoSlasExcluida   = "atendimento.sla.excluida"
	EventoCategoriaSLA   = "atendimento.categoria.sla_vinculada"
)

// SLAEntrada são os campos para criar ou atualizar uma SLA.
type SLAEntrada struct {
	Nome                 string     `json:"nome"`
	Descricao            string     `json:"descricao"`
	TempoRespostaMinutos int        `json:"tempo_resposta_minutos"`
	TempoSolucaoMinutos  int        `json:"tempo_solucao_minutos"`
	PausarEmEspera       bool       `json:"pausar_em_espera"`
	CalendarioTipo       string     `json:"calendario_tipo"`
	UnidadeID            *uuid.UUID `json:"unidade_id"`
	Ativa                *bool      `json:"ativa"`
}

// ListarSLAs lista as SLAs visíveis para a identidade / unidade informada.
func (s *Service) ListarSLAs(ctx context.Context, identity auth.Identity, unidade *uuid.UUID) ([]domain.SLA, error) {
	if !ver(identity, unidade) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.ListSLAs(ctx, s.pool, unidade)
	return lista, MapError(err)
}

// ObterSLA busca uma SLA por ID.
func (s *Service) ObterSLA(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.SLA, error) {
	sla, err := s.repo.GetSLA(ctx, s.pool, id)
	if err != nil {
		return domain.SLA{}, MapError(err)
	}
	if !ver(identity, sla.UnidadeID) {
		return domain.SLA{}, MapError(domain.ErrForaDoEscopo)
	}
	return sla, nil
}

// CriarSLA cadastra uma nova SLA.
func (s *Service) CriarSLA(ctx context.Context, identity auth.Identity, in SLAEntrada) (domain.SLA, error) {
	if !ver(identity, in.UnidadeID) {
		return domain.SLA{}, MapError(domain.ErrForaDoEscopo)
	}
	sla, err := domain.NovaSLA(in.Nome, in.Descricao, in.TempoRespostaMinutos, in.TempoSolucaoMinutos, in.PausarEmEspera, in.CalendarioTipo, in.UnidadeID)
	if err != nil {
		return domain.SLA{}, MapError(err)
	}
	if in.Ativa != nil {
		sla.Ativa = *in.Ativa
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if sla, err = s.repo.InsertSLA(ctx, tx, sla); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoSlasCriada, "sla", sla.ID, sla, nil)
	})
	return sla, MapError(err)
}

// AtualizarSLA altera dados de uma SLA existente.
func (s *Service) AtualizarSLA(ctx context.Context, identity auth.Identity, id uuid.UUID, in SLAEntrada) (domain.SLA, error) {
	prev, err := s.repo.GetSLA(ctx, s.pool, id)
	if err != nil {
		return domain.SLA{}, MapError(err)
	}
	if !ver(identity, prev.UnidadeID) || !ver(identity, in.UnidadeID) {
		return domain.SLA{}, MapError(domain.ErrForaDoEscopo)
	}
	ativa := prev.Ativa
	if in.Ativa != nil {
		ativa = *in.Ativa
	}
	next, err := prev.Aplicar(in.Nome, in.Descricao, in.TempoRespostaMinutos, in.TempoSolucaoMinutos, in.PausarEmEspera, in.CalendarioTipo, in.UnidadeID, ativa)
	if err != nil {
		return domain.SLA{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if next, err = s.repo.UpdateSLA(ctx, tx, next); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoSlasAtualizada, "sla", next.ID, next, prev)
	})
	return next, MapError(err)
}

// ExcluirSLA desativa uma SLA.
func (s *Service) ExcluirSLA(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	sla, err := s.repo.GetSLA(ctx, s.pool, id)
	if err != nil {
		return MapError(err)
	}
	if !ver(identity, sla.UnidadeID) {
		return MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteSLA(ctx, tx, id); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoSlasExcluida, "sla", id, map[string]any{"id": id.String()}, sla)
	})
	return MapError(err)
}

// AssociarCategoriaSLA vincula uma SLA a uma categoria do catálogo (estilo GLPI).
func (s *Service) AssociarCategoriaSLA(ctx context.Context, identity auth.Identity, categoriaID uuid.UUID, slaID *uuid.UUID) error {
	if err := gereCatalogo(identity); err != nil {
		return err
	}
	cat, err := s.repo.GetCategoria(ctx, s.pool, categoriaID)
	if err != nil {
		return MapError(err)
	}
	if slaID != nil {
		if _, err := s.repo.GetSLA(ctx, s.pool, *slaID); err != nil {
			return MapError(domain.ErrSLANaoEncontrada)
		}
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.VincularCategoriaSLA(ctx, tx, categoriaID, slaID); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoCategoriaSLA, "categoria", categoriaID, map[string]any{
			"categoria_id": categoriaID.String(),
			"sla_id":       slaID,
		}, cat)
	})
	return MapError(err)
}

