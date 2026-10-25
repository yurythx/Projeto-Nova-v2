// Package application contém os casos de uso de fornecedor, contrato e licença.
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

const (
	EventoFornecedor = "contratos.fornecedor.criado"
	EventoContrato   = "contratos.contrato.criado"
	EventoLicenca    = "contratos.licenca.criada"
)

// Eventos grava o outbox na transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Service implementa os casos de uso.
type Service struct {
	pool    *pgxpool.Pool
	repo    domain.Repository
	eventos Eventos
	avisos  *notificacoes.Service
	logger  *slog.Logger
	agora   func() time.Time
}

// NewService cria o serviço.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, agora: time.Now}
}

// WithNotificacoes liga o sino do vencimento.
func (s *Service) WithNotificacoes(n *notificacoes.Service) *Service { s.avisos = n; return s }

// WithRelogio fixa o dia do aviso nos testes.
func (s *Service) WithRelogio(agora func() time.Time) *Service { s.agora = agora; return s }

// MapError traduz o domínio para a API.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return apperrors.NotFound("cadastro não encontrado")
	case errors.Is(err, domain.ErrForaDoEscopo):
		return apperrors.Forbidden("você não gerencia contratos desta unidade")
	}
	var invalida domain.EntradaInvalida
	if errors.As(err, &invalida) {
		return apperrors.Validation(invalida.Msg)
	}
	return err
}

func pode(identity auth.Identity, unidade *uuid.UUID) bool {
	alvo := auth.Target{}
	if unidade != nil {
		alvo = auth.InUnidade(*unidade)
	}
	return auth.Can(identity, auth.PermContratosManage, alvo)
}

func (s *Service) filtro(identity auth.Identity) (domain.Filtro, error) {
	cob := auth.CoverageOf(identity, auth.PermContratosManage)
	if !cob.All && len(cob.Unidades) == 0 {
		return domain.Filtro{}, domain.ErrForaDoEscopo
	}
	return domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades}, nil
}

func (s *Service) emitir(ctx context.Context, tx pgx.Tx, evento, tipo string, id uuid.UUID, payload any) error {
	if err := s.eventos.Write(ctx, tx, evento, tipo, id.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: evento, ResourceType: tipo, ResourceID: id.String()})
}

// ListarFornecedores devolve o cadastro do escopo.
func (s *Service) ListarFornecedores(ctx context.Context, identity auth.Identity) ([]domain.Fornecedor, error) {
	f, err := s.filtro(identity)
	if errors.Is(err, domain.ErrForaDoEscopo) {
		return []domain.Fornecedor{}, nil
	}
	if err != nil {
		return nil, MapError(err)
	}
	lista, err := s.repo.ListFornecedores(ctx, s.pool, f)
	return lista, MapError(err)
}

// CriarFornecedor grava o fornecedor, o evento e a auditoria.
func (s *Service) CriarFornecedor(ctx context.Context, identity auth.Identity, nome, email, telefone string, unidade *uuid.UUID) (domain.Fornecedor, error) {
	item, err := domain.NovoFornecedor(nome, email, telefone, unidade)
	if err != nil {
		return domain.Fornecedor{}, MapError(err)
	}
	if !pode(identity, item.UnidadeID) {
		return domain.Fornecedor{}, MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if item, err = s.repo.InsertFornecedor(ctx, tx, item); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoFornecedor, "contratos_fornecedor", item.ID, map[string]any{"id": item.ID.String()})
	})
	return item, MapError(err)
}

// ExcluirFornecedor remove o fornecedor sem contrato nem licença.
func (s *Service) ExcluirFornecedor(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	f, err := s.filtro(identity)
	if err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteFornecedor(ctx, tx, id, f); err != nil {
			return err
		}
		return s.emitir(ctx, tx, "contratos.fornecedor.excluido", "contratos_fornecedor", id, map[string]any{"id": id.String()})
	}))
}

// ListarContratos devolve os contratos do escopo.
func (s *Service) ListarContratos(ctx context.Context, identity auth.Identity) ([]domain.Contrato, error) {
	f, err := s.filtro(identity)
	if errors.Is(err, domain.ErrForaDoEscopo) {
		return []domain.Contrato{}, nil
	}
	lista, err := s.repo.ListContratos(ctx, s.pool, f)
	return lista, MapError(err)
}

// CriarContrato grava o contrato.
func (s *Service) CriarContrato(ctx context.Context, identity auth.Identity, numero, titulo string, fornecedor uuid.UUID, inicio, fim time.Time, valor int64, unidade *uuid.UUID) (domain.Contrato, error) {
	item, err := domain.NovoContrato(numero, titulo, fornecedor, inicio, fim, valor, unidade)
	if err != nil {
		return domain.Contrato{}, MapError(err)
	}
	if !pode(identity, item.UnidadeID) {
		return domain.Contrato{}, MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if item, err = s.repo.InsertContrato(ctx, tx, item); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoContrato, "contratos_contrato", item.ID, map[string]any{"id": item.ID.String()})
	})
	return item, MapError(err)
}

// ExcluirContrato remove o contrato.
func (s *Service) ExcluirContrato(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return s.excluir(ctx, identity, id, s.repo.DeleteContrato, "contratos.contrato.excluido", "contratos_contrato")
}

// ListarLicencas devolve as licenças do escopo.
func (s *Service) ListarLicencas(ctx context.Context, identity auth.Identity) ([]domain.Licenca, error) {
	f, err := s.filtro(identity)
	if errors.Is(err, domain.ErrForaDoEscopo) {
		return []domain.Licenca{}, nil
	}
	lista, err := s.repo.ListLicencas(ctx, s.pool, f)
	return lista, MapError(err)
}

// CriarLicenca grava a licença. A chave secreta não é pedida.
func (s *Service) CriarLicenca(ctx context.Context, identity auth.Identity, nome string, fornecedor uuid.UUID, fim *time.Time, quantidade int, unidade *uuid.UUID) (domain.Licenca, error) {
	item, err := domain.NovaLicenca(nome, fornecedor, fim, quantidade, unidade)
	if err != nil {
		return domain.Licenca{}, MapError(err)
	}
	if !pode(identity, item.UnidadeID) {
		return domain.Licenca{}, MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if item, err = s.repo.InsertLicenca(ctx, tx, item); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoLicenca, "contratos_licenca", item.ID, map[string]any{"id": item.ID.String()})
	})
	return item, MapError(err)
}

// ExcluirLicenca remove a licença.
func (s *Service) ExcluirLicenca(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return s.excluir(ctx, identity, id, s.repo.DeleteLicenca, "contratos.licenca.excluida", "contratos_licenca")
}

func (s *Service) excluir(ctx context.Context, identity auth.Identity, id uuid.UUID, apagar func(context.Context, database.DBTX, uuid.UUID, domain.Filtro) error, evento, tipo string) error {
	f, err := s.filtro(identity)
	if err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := apagar(ctx, tx, id, f); err != nil {
			return err
		}
		return s.emitir(ctx, tx, evento, tipo, id, map[string]any{"id": id.String()})
	}))
}

// VerificarVencimentos avisa, uma vez por marco, quem gerencia contratos.
// Os marcos são 30 dias, 7 dias, o dia do vencimento e o atraso.
func (s *Service) VerificarVencimentos(ctx context.Context) error {
	if s.avisos == nil {
		return nil
	}
	lista, err := s.repo.Vencendo(ctx, s.pool)
	if err != nil {
		return err
	}
	hoje := s.agora().UTC().Truncate(24 * time.Hour)
	for _, v := range lista {
		dias := int(v.Fim.UTC().Truncate(24*time.Hour).Sub(hoje).Hours() / 24)
		marco := ""
		switch {
		case dias < 0:
			marco = "vencido"
		case dias == 0:
			marco = "0"
		case dias <= 7:
			marco = "7"
		case dias <= 30:
			marco = "30"
		default:
			continue
		}
		destinos, err := s.repo.Destinatarios(ctx, s.pool, v.UnidadeID)
		if err != nil {
			return err
		}
		titulo := fmt.Sprintf("%s vence em %d dias", v.Titulo, dias)
		if marco == "0" {
			titulo = v.Titulo + " vence hoje"
		}
		if marco == "vencido" {
			titulo = v.Titulo + " está vencido"
		}
		if _, err := s.avisos.Enviar(ctx, destinos, notificacoes.Nova{
			Modulo: "contratos", Titulo: titulo, Mensagem: "Contrato ou licença no cadastro.",
			Link: "/contratos", Chave: "contratos:" + v.ID.String() + ":" + marco,
		}); err != nil && s.logger != nil {
			s.logger.WarnContext(ctx, "contratos: aviso de vencimento falhou", slog.Any("error", err))
		}
	}
	return nil
}

// RunVencimentos repete a verificação uma vez por dia.
func (s *Service) RunVencimentos(ctx context.Context) error {
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		if err := s.VerificarVencimentos(ctx); err != nil && ctx.Err() == nil && s.logger != nil {
			s.logger.WarnContext(ctx, "contratos: verificação de vencimento falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
