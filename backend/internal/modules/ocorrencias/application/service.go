// Package application contém os casos de uso das ocorrências (ADR 066).
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Eventos que as ocorrências publicam pelo outbox.
const (
	EventoDeclarada   = "ocorrencias.declarada"
	EventoAtualizada  = "ocorrencias.atualizada"
	EventoResolvida   = "ocorrencias.resolvida"
	EventoCoordenacao = "ocorrencias.coordenacao"
	EventoRevisada    = "ocorrencias.revisada"
	recurso           = "ocorrencia"
	LimitePainel      = 50
)

// Eventos é a porta do outbox (outbox.Writer em produção).
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Service atende o painel da coordenação e a página pública.
type Service struct {
	pool    *pgxpool.Pool
	repo    domain.Repository
	eventos Eventos
	relogio func() time.Time
}

// NewService cria o serviço.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, relogio: time.Now}
}

// WithRelogio troca o relógio (testes).
func (s *Service) WithRelogio(f func() time.Time) *Service { s.relogio = f; return s }

func (s *Service) agora() time.Time { return s.relogio().UTC().Truncate(time.Microsecond) }

// DeclaracaoEntrada é o que se sabe na hora de declarar.
type DeclaracaoEntrada struct {
	Titulo    string
	Servicos  []string
	Impacto   string
	Texto     string
	ChamadoID *uuid.UUID
}

// AtualizacaoEntrada é a próxima notícia ao público.
type AtualizacaoEntrada struct {
	Estado  string
	Impacto string
	Texto   string
}

// MapError traduz os erros de domínio para HTTP.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrada):
		return causa(err, apperrors.NotFound("ocorrência não encontrada"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe título (até 160), de 1 a 10 serviços (até 80 cada), impacto, estado e o texto ao público (até 4000); o chamado precisa existir"))
	case errors.Is(err, domain.ErrJaResolvida):
		return causa(err, apperrors.Conflict("a ocorrência já foi resolvida"))
	case errors.Is(err, domain.ErrAindaAberta):
		return causa(err, apperrors.Conflict("a revisão vem depois de resolver a ocorrência"))
	case errors.Is(err, domain.ErrCoordenadorInvalido):
		return causa(err, apperrors.Validation("a pessoa escolhida para coordenar não existe"))
	}
	// Erros HTTP prontos (Bind, UUIDParam) seguem como vieram; o resto vira
	// 500 na escrita da resposta.
	return err
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

// Declarar abre a ocorrência com a primeira notícia. Quem declara coordena.
func (s *Service) Declarar(ctx context.Context, identity auth.Identity, in DeclaracaoEntrada) (domain.Ocorrencia, error) {
	o, a, err := domain.Declarar(in.Titulo, in.Servicos, in.Impacto, in.Texto, in.ChamadoID, identity.UserID, s.agora())
	if err != nil {
		return domain.Ocorrencia{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Insert(ctx, tx, o); err != nil {
			return err
		}
		if err := s.repo.InsertAtualizacao(ctx, tx, a); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoDeclarada, o, nil)
	})
	if err != nil {
		return domain.Ocorrencia{}, MapError(err)
	}
	return s.Obter(ctx, o.ID)
}

// Atualizar registra a nova notícia; "resolvido" encerra.
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, in AtualizacaoEntrada) (domain.Ocorrencia, error) {
	return s.alterar(ctx, id, func(ctx context.Context, tx pgx.Tx, antes domain.Ocorrencia) (domain.Ocorrencia, string, error) {
		depois, a, err := antes.Atualizar(in.Estado, in.Impacto, in.Texto, identity.UserID, s.agora())
		if err != nil {
			return domain.Ocorrencia{}, "", err
		}
		if err := s.repo.InsertAtualizacao(ctx, tx, a); err != nil {
			return domain.Ocorrencia{}, "", err
		}
		if depois.Resolvida() {
			return depois, EventoResolvida, nil
		}
		return depois, EventoAtualizada, nil
	})
}

// TrocarCoordenador passa a coordenação a outra pessoa.
func (s *Service) TrocarCoordenador(ctx context.Context, id, quem uuid.UUID) (domain.Ocorrencia, error) {
	return s.alterar(ctx, id, func(_ context.Context, _ pgx.Tx, antes domain.Ocorrencia) (domain.Ocorrencia, string, error) {
		depois, err := antes.TrocarCoordenador(quem, s.agora())
		return depois, EventoCoordenacao, err
	})
}

// Revisar grava a revisão pós-incidente (interna).
func (s *Service) Revisar(ctx context.Context, id uuid.UUID, texto string) (domain.Ocorrencia, error) {
	return s.alterar(ctx, id, func(_ context.Context, _ pgx.Tx, antes domain.Ocorrencia) (domain.Ocorrencia, string, error) {
		depois, err := antes.Revisar(texto, s.agora())
		return depois, EventoRevisada, err
	})
}

type mudanca func(ctx context.Context, tx pgx.Tx, antes domain.Ocorrencia) (domain.Ocorrencia, string, error)

// alterar trava a ocorrência, aplica a mudança e grava, com evento e
// auditoria, na mesma transação.
func (s *Service) alterar(ctx context.Context, id uuid.UUID, fn mudanca) (domain.Ocorrencia, error) {
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		antes, err := s.repo.Travar(ctx, tx, id)
		if err != nil {
			return err
		}
		depois, evento, err := fn(ctx, tx, antes)
		if err != nil {
			return err
		}
		if err := s.repo.Update(ctx, tx, depois); err != nil {
			return err
		}
		return s.registrar(ctx, tx, evento, depois, &antes)
	})
	if err != nil {
		return domain.Ocorrencia{}, MapError(err)
	}
	return s.Obter(ctx, id)
}

func (s *Service) registrar(ctx context.Context, tx pgx.Tx, evento string, o domain.Ocorrencia, antes *domain.Ocorrencia) error {
	// O evento leva só o que é público: quem consome pode avisar alguém.
	payload := map[string]any{"id": o.ID.String(), "titulo": o.Titulo, "impacto": o.Impacto, "estado": o.Estado, "servicos": o.Servicos}
	if err := s.eventos.Write(ctx, tx, evento, recurso, o.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	var before any
	if antes != nil {
		before = antes
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, recurso, o.ID.String(), before, o))
}

// Obter devolve a ocorrência com a linha do tempo.
func (s *Service) Obter(ctx context.Context, id uuid.UUID) (domain.Ocorrencia, error) {
	o, err := s.repo.Get(ctx, s.pool, id)
	return o, MapError(err)
}

// Listar devolve o painel: abertas primeiro.
func (s *Service) Listar(ctx context.Context) ([]domain.Ocorrencia, error) {
	lista, err := s.repo.List(ctx, s.pool, LimitePainel)
	return lista, MapError(err)
}

// Situacao é a página de status: abertas e resolvidas dos últimos 7 dias,
// só com os campos públicos.
func (s *Service) Situacao(ctx context.Context) (domain.Situacao, error) {
	lista, err := s.repo.ParaPagina(ctx, s.pool, s.agora().Add(-domain.JanelaRecentes))
	if err != nil {
		return domain.Situacao{}, MapError(err)
	}
	return domain.MontarSituacao(lista), nil
}
