// Package application contém os casos de uso do problema.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/sismarq/internal/domain/errors"
	"github.com/yurythx/sismarq/internal/modules/problemas/domain"
	"github.com/yurythx/sismarq/internal/platform/audit"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/database"
)

const (
	EventoCriado     = "problemas.problema.criado"
	EventoAtualizado = "problemas.problema.atualizado"
	EventoEstado     = "problemas.problema.estado_alterado"
	EventoVinculo    = "problemas.problema.vinculo_alterado"
)

// Eventos grava o evento no outbox da transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Portas para os outros plug-ins (qualquer uma pode ser nil: o vínculo
// correspondente fica indisponível).
type Portas struct {
	Incidentes domain.Incidentes
	Ativos     domain.Ativos
	Produtos   domain.Produtos
}

// Service implementa os casos de uso.
type Service struct {
	pool    *pgxpool.Pool
	repo    domain.Repository
	eventos Eventos
	portas  Portas
	agora   func() time.Time
}

// NewService cria o serviço.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, portas Portas) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, portas: portas, agora: time.Now}
}

// MapError traduz erros de domínio para a API e preserva a causa.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return causa(err, apperrors.NotFound("problema não encontrado"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe o título (até 160 caracteres); os textos cabem em 4000"))
	case errors.Is(err, domain.ErrForaDoEscopo):
		return causa(err, apperrors.Forbidden("você não gerencia os problemas desta unidade"))
	case errors.Is(err, domain.ErrUnidadeInexistente):
		return causa(err, apperrors.Validation("unidade inexistente"))
	case errors.Is(err, domain.ErrTransicaoInvalida):
		return causa(err, apperrors.Conflict("esta mudança de estado não é permitida agora"))
	case errors.Is(err, domain.ErrSemContorno):
		return causa(err, apperrors.Validation("registre a solução de contorno para tornar o problema um erro conhecido"))
	case errors.Is(err, domain.ErrSemCausaRaiz):
		return causa(err, apperrors.Validation("registre a causa raiz para resolver o problema"))
	case errors.Is(err, domain.ErrEncerrado):
		return causa(err, apperrors.Conflict("o problema está encerrado"))
	case errors.Is(err, domain.ErrIncidenteInexistente):
		return causa(err, apperrors.Validation("incidente não encontrado"))
	case errors.Is(err, domain.ErrNaoIncidente):
		return causa(err, apperrors.Validation("só incidentes se ligam a um problema; requisição não"))
	case errors.Is(err, domain.ErrAtivoInexistente):
		return causa(err, apperrors.Validation("ativo não encontrado"))
	case errors.Is(err, domain.ErrDeOutraUnidade):
		return causa(err, apperrors.Validation("o vínculo precisa ser da mesma unidade do problema"))
	case errors.Is(err, domain.ErrIntegracaoDesligada):
		return causa(err, apperrors.Validation("o módulo deste vínculo está desligado"))
	case errors.Is(err, domain.ErrProdutoInexistente):
		return causa(err, apperrors.Validation("produto não encontrado"))
	default:
		return err
	}
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

func gere(identity auth.Identity, unidade *uuid.UUID) bool {
	return auth.Can(identity, auth.PermProblemasManage, domain.Dona(unidade))
}

// le: quem gere ou quem consulta (a equipe de atendimento lê os erros
// conhecidos para aplicar o contorno).
func le(identity auth.Identity, unidade *uuid.UUID) bool {
	return gere(identity, unidade) || auth.Can(identity, auth.PermProblemasRead, domain.Dona(unidade))
}

// cobertura une gestão e consulta.
func cobertura(identity auth.Identity) (bool, []uuid.UUID, bool) {
	g, r := auth.CoverageOf(identity, auth.PermProblemasManage), auth.CoverageOf(identity, auth.PermProblemasRead)
	if g.All || r.All {
		return true, nil, true
	}
	unidades := append(append([]uuid.UUID{}, g.Unidades...), r.Unidades...)
	return false, unidades, len(unidades) > 0
}

// FiltroEntrada são os recortes da lista.
type FiltroEntrada struct {
	Estado        string
	SolicitacaoID *uuid.UUID
	Busca         string
}

// Listar devolve os problemas que identity gere ou consulta.
func (s *Service) Listar(ctx context.Context, identity auth.Identity, f FiltroEntrada) ([]domain.Problema, error) {
	todas, unidades, alguma := cobertura(identity)
	if !alguma {
		return []domain.Problema{}, nil
	}
	lista, err := s.repo.List(ctx, s.pool, domain.Filtro{Restrito: !todas, Unidades: unidades, Estado: f.Estado,
		SolicitacaoID: f.SolicitacaoID, Busca: f.Busca})
	return lista, MapError(err)
}

// Obter devolve o problema com os vínculos e as passagens de quem gere.
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Problema, error) {
	p, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Problema{}, MapError(err)
	}
	if !le(identity, p.UnidadeID) {
		return domain.Problema{}, MapError(domain.ErrNaoEncontrado)
	}
	if p.Incidentes, err = s.repo.Incidentes(ctx, s.pool, id); err != nil {
		return domain.Problema{}, MapError(err)
	}
	if p.Ativos, err = s.repo.AtivosDe(ctx, s.pool, id); err != nil {
		return domain.Problema{}, MapError(err)
	}
	for i := range p.Ativos {
		if s.portas.Ativos != nil {
			// Ativo desligado ou apagado: o vínculo aparece sem nome.
			p.Ativos[i].Nome, _, _ = s.portas.Ativos.Consultar(ctx, s.pool, p.Ativos[i].AtivoID)
		}
	}
	p.Proximos = []string{}
	if gere(identity, p.UnidadeID) {
		p.Proximos = p.PodeIrPara()
	}
	return p, nil
}

// Criar registra o problema, o evento e a auditoria.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, in domain.Entrada) (domain.Problema, error) {
	p, err := domain.Novo(in, identity.UserID)
	if err != nil {
		return domain.Problema{}, MapError(err)
	}
	if !gere(identity, p.UnidadeID) {
		return domain.Problema{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, s.pool, p.ProdutoID, p.UnidadeID); err != nil {
		return domain.Problema{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if p, err = s.repo.Insert(ctx, tx, p); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoCriado, p, nil, p, nil)
	})
	return p, MapError(err)
}

type mudanca func(ctx context.Context, tx pgx.Tx, p domain.Problema) error

// alterar trava o problema, confere a gestão e aplica a mudança.
func (s *Service) alterar(ctx context.Context, identity auth.Identity, id uuid.UUID, fn mudanca) (domain.Problema, error) {
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		p, err := s.repo.Get(ctx, tx, id)
		if err != nil {
			return err
		}
		if !gere(identity, p.UnidadeID) {
			if le(identity, p.UnidadeID) {
				return domain.ErrForaDoEscopo
			}
			return domain.ErrNaoEncontrado
		}
		return fn(ctx, tx, p)
	})
	if err != nil {
		return domain.Problema{}, MapError(err)
	}
	return s.Obter(ctx, identity, id)
}

// Atualizar edita o problema. Edição vazia não regrava.
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, in domain.Entrada) (domain.Problema, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, p domain.Problema) error {
		next, err := p.Aplicar(in)
		if err != nil {
			return err
		}
		if !gere(identity, next.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := s.conferirProduto(ctx, tx, next.ProdutoID, next.UnidadeID); err != nil {
			return err
		}
		if p.Igual(next) {
			return nil
		}
		if err := s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoAtualizado, next, p, next, nil)
	})
}

// Transicionar muda o estado com as exigências de cada um.
func (s *Service) Transicionar(ctx context.Context, identity auth.Identity, id uuid.UUID, para string) (domain.Problema, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, p domain.Problema) error {
		next, mudou, err := p.Transicionar(para)
		if err != nil || !mudou {
			return err
		}
		if err := s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoEstado, next, p, next, map[string]any{"de": p.Estado})
	})
}

// VincularIncidente liga um incidente da mesma unidade ao problema.
func (s *Service) VincularIncidente(ctx context.Context, identity auth.Identity, id, solicitacao uuid.UUID, ligar bool) (domain.Problema, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, p domain.Problema) error {
		if p.Estado == domain.Encerrado {
			return domain.ErrEncerrado
		}
		mudou := false
		if ligar {
			if s.portas.Incidentes == nil {
				return domain.ErrIntegracaoDesligada
			}
			inc, err := s.portas.Incidentes.Consultar(ctx, tx, solicitacao)
			if err != nil {
				return err
			}
			if inc.Tipo != "incidente" {
				return domain.ErrNaoIncidente
			}
			if !domain.MesmaUnidade(inc.UnidadeID, p.UnidadeID) {
				return domain.ErrDeOutraUnidade
			}
			if mudou, err = s.repo.VincularIncidente(ctx, tx, id, domain.IncidenteVinculado{
				SolicitacaoID: inc.ID, Numero: inc.Numero, Titulo: inc.Titulo,
			}, identity.UserID); err != nil {
				return err
			}
		} else {
			var err error
			if mudou, err = s.repo.DesvincularIncidente(ctx, tx, id, solicitacao); err != nil {
				return err
			}
		}
		if !mudou {
			return nil
		}
		vinculo := map[string]any{"incidente": solicitacao.String(), "ligado": ligar}
		return s.registrar(ctx, tx, EventoVinculo, p, nil, vinculo, vinculo)
	})
}

// VincularAtivo liga um ativo da mesma unidade ao problema.
func (s *Service) VincularAtivo(ctx context.Context, identity auth.Identity, id, ativo uuid.UUID, ligar bool) (domain.Problema, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, p domain.Problema) error {
		if p.Estado == domain.Encerrado {
			return domain.ErrEncerrado
		}
		mudou := false
		if ligar {
			if s.portas.Ativos == nil {
				return domain.ErrIntegracaoDesligada
			}
			_, dona, err := s.portas.Ativos.Consultar(ctx, tx, ativo)
			if err != nil {
				return err
			}
			if !domain.MesmaUnidade(dona, p.UnidadeID) {
				return domain.ErrDeOutraUnidade
			}
			if mudou, err = s.repo.VincularAtivo(ctx, tx, id, ativo, identity.UserID); err != nil {
				return err
			}
		} else {
			var err error
			if mudou, err = s.repo.DesvincularAtivo(ctx, tx, id, ativo); err != nil {
				return err
			}
		}
		if !mudou {
			return nil
		}
		vinculo := map[string]any{"ativo": ativo.String(), "ligado": ligar}
		return s.registrar(ctx, tx, EventoVinculo, p, nil, vinculo, vinculo)
	})
}

// Sugestoes: ofertas com incidentes repetidos que ainda não estão em
// nenhum problema — candidatas a abrir um.
func (s *Service) Sugestoes(ctx context.Context, identity auth.Identity) ([]domain.Recorrencia, error) {
	cob := auth.CoverageOf(identity, auth.PermProblemasManage)
	if s.portas.Incidentes == nil || (!cob.All && len(cob.Unidades) == 0) {
		return []domain.Recorrencia{}, nil
	}
	desde := s.agora().AddDate(0, 0, -domain.RecorrenciaDias)
	grupos, err := s.portas.Incidentes.Recorrentes(ctx, s.pool, !cob.All, cob.Unidades, desde, domain.RecorrenciaMinima)
	if err != nil {
		return nil, MapError(err)
	}
	todos := []uuid.UUID{}
	for _, g := range grupos {
		todos = append(todos, g.Solicitacoes...)
	}
	ligados, err := s.repo.JaVinculados(ctx, s.pool, todos)
	if err != nil {
		return nil, MapError(err)
	}
	out := []domain.Recorrencia{}
	for _, g := range grupos {
		soltos := []uuid.UUID{}
		for _, id := range g.Solicitacoes {
			if !ligados[id] {
				soltos = append(soltos, id)
			}
		}
		if len(soltos) >= domain.RecorrenciaMinima {
			g.Solicitacoes, g.Total = soltos, len(soltos)
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *Service) conferirProduto(ctx context.Context, db database.DBTX, produto, unidade *uuid.UUID) error {
	if produto == nil {
		return nil
	}
	if s.portas.Produtos == nil {
		return domain.ErrIntegracaoDesligada
	}
	dona, err := s.portas.Produtos.Consultar(ctx, db, *produto)
	if err != nil {
		return err
	}
	if !domain.MesmaUnidade(dona, unidade) {
		return domain.ErrDeOutraUnidade
	}
	return nil
}

// registrar grava o evento (com o extra no payload) e a auditoria.
func (s *Service) registrar(ctx context.Context, tx pgx.Tx, evento string, p domain.Problema, antes, depois any, extra map[string]any) error {
	payload := map[string]any{"id": p.ID.String(), "numero": p.Numero, "estado": p.Estado}
	for k, v := range extra {
		payload[k] = v
	}
	if err := s.eventos.Write(ctx, tx, evento, "problema", p.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, "problema", p.ID.String(), antes, depois))
}
