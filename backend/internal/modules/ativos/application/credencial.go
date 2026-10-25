package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// CredencialEmitida é a resposta da criação. Token só existe nessa resposta.
type CredencialEmitida struct {
	domain.Credencial
	Token string `json:"token"`
}

// EmitirCredencial grava o hash e devolve o segredo uma vez.
func (s *Service) EmitirCredencial(ctx context.Context, identity auth.Identity, nome string, unidade uuid.UUID) (CredencialEmitida, error) {
	if !pode(identity, &unidade) {
		return CredencialEmitida{}, MapError(domain.ErrForaDoEscopo)
	}
	c, token, err := domain.EmitirCredencial(nome, unidade)
	if err != nil {
		return CredencialEmitida{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InsertCredencial(ctx, tx, c); err != nil {
			return err
		}
		return s.auditarCredencial(ctx, tx, "ativos.credencial.criada", c.ID, map[string]any{"nome": c.Nome, "unidade_id": c.UnidadeID.String()})
	})
	if err != nil {
		return CredencialEmitida{}, MapError(err)
	}
	return CredencialEmitida{Credencial: c, Token: token}, nil
}

// ListarCredenciais devolve as credenciais vigentes do escopo, sem segredo.
func (s *Service) ListarCredenciais(ctx context.Context, identity auth.Identity) ([]domain.Credencial, error) {
	f, err := s.filtro(identity)
	if err != nil {
		return []domain.Credencial{}, nil
	}
	lista, err := s.repo.ListCredenciais(ctx, s.pool, f)
	return lista, MapError(err)
}

// RevogarCredencial invalida o bearer. O hash deixa de autenticar.
func (s *Service) RevogarCredencial(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	f, err := s.filtro(identity)
	if err != nil {
		return MapError(domain.ErrNaoEncontrado)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.RevogarCredencial(ctx, tx, id, f); err != nil {
			return err
		}
		return s.auditarCredencial(ctx, tx, "ativos.credencial.revogada", id, nil)
	})
	return MapError(err)
}

// Reconhecer autentica o bearer da máquina. ok false: não é credencial.
func (s *Service) Reconhecer(ctx context.Context, token string) (auth.Identity, bool, error) {
	if len(token) < len(domain.PrefixoCredencial) || token[:len(domain.PrefixoCredencial)] != domain.PrefixoCredencial {
		return auth.Identity{}, false, nil
	}
	c, err := s.repo.PorHash(ctx, s.pool, domain.HashCredencial(token))
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return auth.Identity{}, true, apperrors.Unauthorized("invalid or expired access token")
	}
	if err != nil {
		return auth.Identity{}, true, MapError(err)
	}
	unidade := c.UnidadeID
	return auth.Identity{
		Subject: c.ID.String(), Username: c.Nome, Name: c.Nome, Source: auth.SourceAgente,
		Permissions: []string{string(auth.PermAtivosReportar)},
		Scopes: []auth.Scope{{
			Origem: "agente", UnidadeID: &unidade, Permissions: []string{string(auth.PermAtivosReportar)},
		}},
	}, true, nil
}

func (s *Service) auditarCredencial(ctx context.Context, tx pgx.Tx, evento string, id uuid.UUID, payload any) error {
	if err := s.eventos.Write(ctx, tx, evento, "ativo_credencial", id.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: evento, ResourceType: "ativo_credencial", ResourceID: id.String()})
}

func (s *Service) filtro(identity auth.Identity) (domain.Filtro, error) {
	cob := auth.CoverageOf(identity, auth.PermAtivosManage)
	if !cob.All && len(cob.Unidades) == 0 {
		return domain.Filtro{}, domain.ErrForaDoEscopo
	}
	return domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades}, nil
}
