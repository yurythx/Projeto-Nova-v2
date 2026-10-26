// Package application contém os casos de uso da mudança.
package application

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	EventoCriada     = "mudancas.mudanca.criada"
	EventoAtualizada = "mudancas.mudanca.atualizada"
	EventoEstado     = "mudancas.mudanca.estado_alterado"
	EventoVinculo    = "mudancas.mudanca.vinculo_alterado"
)

// Eventos grava o evento no outbox da transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Service implementa os casos de uso.
type Service struct {
	pool      *pgxpool.Pool
	repo      domain.Repository
	eventos   Eventos
	portas    domain.Portas
	produtos  domain.Produtos
	efeitos   domain.Efeitos
	passagens Passagens
}

// Passagens lê os caminhos acrescentados no quadro. Nil mantém só o fluxo de fábrica.
type Passagens interface {
	Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error)
}

// NewService cria o serviço. portas e produtos podem ser nil: o vínculo
// correspondente fica indisponível.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, portas domain.Portas, produtos domain.Produtos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, portas: portas, produtos: produtos}
}

// ComEfeitos liga a conclusão dos incidentes e problemas. Sem ela, pedir
// a cascata devolve indisponível.
func (s *Service) ComEfeitos(e domain.Efeitos) *Service {
	s.efeitos = e
	return s
}

// ComPassagens liga os caminhos acrescentados no quadro de mudanças.
func (s *Service) ComPassagens(p Passagens) { s.passagens = p }

func (s *Service) mapa(ctx context.Context, db database.DBTX) (map[string][]string, error) {
	if s.passagens == nil {
		return map[string][]string{}, nil
	}
	return s.passagens.Mapa(ctx, db, "mudancas")
}

// MapError traduz erros de domínio para a API e preserva a causa.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return causa(err, apperrors.NotFound("mudança não encontrada"))
	case errors.Is(err, domain.ErrEntradaInvalida):
		return causa(err, apperrors.Validation("informe o título (até 160 caracteres); os textos cabem em 4000"))
	case errors.Is(err, domain.ErrTipoInvalido):
		return causa(err, apperrors.Validation("tipo (padrão, normal, emergencial) ou risco (baixo, médio, alto) desconhecido"))
	case errors.Is(err, domain.ErrForaDoEscopo):
		return causa(err, apperrors.Forbidden("você não gerencia as mudanças desta unidade"))
	case errors.Is(err, domain.ErrUnidadeInexistente):
		return causa(err, apperrors.Validation("unidade inexistente"))
	case errors.Is(err, domain.ErrTransicaoInvalida):
		return causa(err, apperrors.Conflict("esta mudança de estado não é permitida agora para o seu papel"))
	case errors.Is(err, domain.ErrNaoEditavel):
		return causa(err, apperrors.Conflict("só se edita a mudança em rascunho ou rejeitada"))
	case errors.Is(err, domain.ErrSemJanela):
		return causa(err, apperrors.Validation("informe a janela planejada, com o início antes do fim"))
	case errors.Is(err, domain.ErrSemReversao):
		return causa(err, apperrors.Validation("a mudança normal ou emergencial precisa do plano de reversão"))
	case errors.Is(err, domain.ErrSemParecer):
		return causa(err, apperrors.Validation("rejeitar exige um parecer"))
	case errors.Is(err, domain.ErrSemResultado):
		return causa(err, apperrors.Validation("registrar a falha exige o resultado: o que foi revertido"))
	case errors.Is(err, domain.ErrAutoAprovacao):
		return causa(err, apperrors.Forbidden("quem pediu a mudança não pode aprová-la"))
	case errors.Is(err, domain.ErrVinculoInvalido):
		return causa(err, apperrors.Validation("o vínculo é com problema, incidente ou ativo"))
	case errors.Is(err, domain.ErrVinculoInexistente):
		return causa(err, apperrors.Validation("item do vínculo não encontrado"))
	case errors.Is(err, domain.ErrDeOutraUnidade):
		return causa(err, apperrors.Validation("o vínculo precisa ser da mesma unidade da mudança"))
	case errors.Is(err, domain.ErrIntegracaoDesligada):
		return causa(err, apperrors.Validation("o módulo deste vínculo está desligado"))
	case errors.Is(err, domain.ErrMudancaEncerrada):
		return causa(err, apperrors.Conflict("a mudança já foi encerrada"))
	case errors.Is(err, domain.ErrCascataIndisponivel):
		return causa(err, apperrors.Validation("não dá para concluir os vínculos com os módulos desligados"))
	case errors.Is(err, domain.ErrRiscoDivergente):
		return causa(err, apperrors.Validation("a nota de 5 a 15 precisa corresponder ao risco: até 7 é baixo, até 11 é médio, acima é alto"))
	case errors.Is(err, domain.ErrCongelada):
		return causa(err, apperrors.Conflict("essa janela cai num período de congelamento"))
	default:
		var recusa *domain.VinculoRecusado
		if errors.As(err, &recusa) && recusa.Mensagem != "" {
			return causa(err, apperrors.Conflict(recusa.Mensagem))
		}
		return err
	}
}

func (s *Service) conferirJanela(ctx context.Context, db database.DBTX, unidade *uuid.UUID, inicio, fim *time.Time) error {
	if inicio == nil || fim == nil {
		return nil
	}
	ocupada, err := s.repo.JanelaOcupada(ctx, db, unidade, *inicio, *fim)
	if err != nil || !ocupada {
		return err
	}
	return domain.ErrCongelada
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

// papeis diz como identity age sobre a mudança da unidade.
func papeis(identity auth.Identity, unidade *uuid.UUID) []string {
	out := []string{}
	if auth.Can(identity, auth.PermMudancasManage, domain.Dona(unidade)) {
		out = append(out, domain.PapelGestor)
	}
	if auth.Can(identity, auth.PermMudancasAprovar, domain.Dona(unidade)) {
		out = append(out, domain.PapelAprovador)
	}
	return out
}

// FiltroEntrada são os recortes da lista.
type FiltroEntrada struct {
	Estado   string
	AlvoTipo string
	AlvoID   *uuid.UUID
}

// Listar devolve as mudanças que identity gere ou aprova.
func (s *Service) Listar(ctx context.Context, identity auth.Identity, f FiltroEntrada) ([]domain.Mudanca, error) {
	g, a := auth.CoverageOf(identity, auth.PermMudancasManage), auth.CoverageOf(identity, auth.PermMudancasAprovar)
	unidades := append(append([]uuid.UUID{}, g.Unidades...), a.Unidades...)
	todas := g.All || a.All
	if !todas && len(unidades) == 0 {
		return []domain.Mudanca{}, nil
	}
	lista, err := s.repo.List(ctx, s.pool, domain.Filtro{Restrito: !todas, Unidades: unidades, Estado: f.Estado,
		AlvoTipo: f.AlvoTipo, AlvoID: f.AlvoID})
	if err != nil {
		return nil, MapError(err)
	}
	mapa, err := s.mapa(ctx, s.pool)
	if err != nil {
		return nil, MapError(err)
	}
	for i := range lista {
		ps := papeis(identity, lista[i].UnidadeID)
		if slices.Contains(ps, domain.PapelGestor) {
			lista[i] = lista[i].ComPassagens(mapa[lista[i].Estado])
		}
		if len(ps) > 0 {
			lista[i].Proximos = lista[i].PodeIrPara(identity.UserID, ps...)
		}
	}
	return lista, nil
}

// Obter devolve a mudança com as decisões, os vínculos e o que identity
// pode fazer agora.
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Mudanca, error) {
	m, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	ps := papeis(identity, m.UnidadeID)
	if len(ps) == 0 {
		return domain.Mudanca{}, MapError(domain.ErrNaoEncontrado)
	}
	if m.Aprovacoes, err = s.repo.Aprovacoes(ctx, s.pool, id); err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	if m.Vinculos, err = s.repo.Vinculos(ctx, s.pool, id); err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	if slices.Contains(ps, domain.PapelGestor) {
		mapa, err := s.mapa(ctx, s.pool)
		if err != nil {
			return domain.Mudanca{}, MapError(err)
		}
		m = m.ComPassagens(mapa[m.Estado])
	}
	m.Proximos = m.PodeIrPara(identity.UserID, ps...)
	return m, nil
}

// Criar abre a mudança como rascunho.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, in domain.Entrada) (domain.Mudanca, error) {
	m, err := domain.Nova(in, identity.UserID)
	if err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	if !slices.Contains(papeis(identity, m.UnidadeID), domain.PapelGestor) {
		return domain.Mudanca{}, MapError(domain.ErrForaDoEscopo)
	}
	if err := s.conferirProduto(ctx, s.pool, m.ProdutoID, m.UnidadeID); err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	if err := s.conferirJanela(ctx, s.pool, m.UnidadeID, m.InicioPlanejado, m.FimPlanejado); err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if m, err = s.repo.Insert(ctx, tx, m); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoCriada, m, nil, m, nil)
	})
	return m, MapError(err)
}

type mudanca func(ctx context.Context, tx pgx.Tx, m domain.Mudanca, papeis []string) error

// alterar trava a mudança, confere quem a vê e aplica.
func (s *Service) alterar(ctx context.Context, identity auth.Identity, id uuid.UUID, fn mudanca) (domain.Mudanca, error) {
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		m, err := s.repo.Get(ctx, tx, id)
		if err != nil {
			return err
		}
		ps := papeis(identity, m.UnidadeID)
		if len(ps) == 0 {
			return domain.ErrNaoEncontrado
		}
		return fn(ctx, tx, m, ps)
	})
	if err != nil {
		return domain.Mudanca{}, MapError(err)
	}
	return s.Obter(ctx, identity, id)
}

// Atualizar edita o rascunho (ou a rejeitada, para reapresentar).
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, in domain.Entrada) (domain.Mudanca, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, m domain.Mudanca, ps []string) error {
		if !slices.Contains(ps, domain.PapelGestor) {
			return domain.ErrForaDoEscopo
		}
		next, err := m.Aplicar(in)
		if err != nil {
			return err
		}
		if !slices.Contains(papeis(identity, next.UnidadeID), domain.PapelGestor) {
			return domain.ErrForaDoEscopo
		}
		if err := s.conferirProduto(ctx, tx, next.ProdutoID, next.UnidadeID); err != nil {
			return err
		}
		if err := s.conferirJanela(ctx, tx, next.UnidadeID, next.InicioPlanejado, next.FimPlanejado); err != nil {
			return err
		}
		if m.Igual(next) {
			return nil
		}
		if err := s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoAtualizada, next, m, next, nil)
	})
}

// Transicionar conduz (gestor) ou decide (aprovador) a mudança. A decisão
// fica registrada com quem, quando e o parecer. resolverVinculos, na
// conclusão, resolve incidentes e problemas ligados na mesma transação:
// se um não puder ser concluído, a mudança não muda de estado.
func (s *Service) Transicionar(ctx context.Context, identity auth.Identity, id uuid.UUID, para, texto string, resolverVinculos bool) (domain.Mudanca, error) {
	var depois []domain.Entrega
	m, err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, m domain.Mudanca, ps []string) error {
		if slices.Contains(ps, domain.PapelGestor) {
			mapa, err := s.mapa(ctx, tx)
			if err != nil {
				return err
			}
			m = m.ComPassagens(mapa[m.Estado])
		}
		next, decisao, mudou, err := m.Transicionar(domain.Transicao{Para: para, Autor: identity.UserID, Texto: texto}, ps...)
		if err != nil || !mudou {
			return err
		}
		if para == domain.EmAvaliacao {
			if err := s.conferirJanela(ctx, tx, next.UnidadeID, next.InicioPlanejado, next.FimPlanejado); err != nil {
				return err
			}
		}
		if err := s.repo.Update(ctx, tx, next); err != nil {
			return err
		}
		extra := map[string]any{"de": m.Estado}
		if decisao != nil {
			if err := s.repo.InsertAprovacao(ctx, tx, id, *decisao); err != nil {
				return err
			}
			extra["decisao"] = decisao.Decisao
		}
		if next.Estado == domain.Concluida && resolverVinculos {
			entregas, err := s.resolverVinculos(ctx, tx, id, domain.Resolucao{
				Autor: identity.UserID, Texto: texto, Numero: next.Numero, Titulo: next.Titulo,
			})
			if err != nil {
				return err
			}
			depois = entregas
			extra["resolver_vinculos"] = true
		}
		return s.registrar(ctx, tx, EventoEstado, next, m, next, extra)
	})
	if err != nil {
		return domain.Mudanca{}, err
	}
	for _, entrega := range depois {
		if entrega != nil {
			entrega(ctx)
		}
	}
	return m, nil
}

// resolverVinculos conclui incidentes e problemas em ordem estável, para
// duas mudanças simultâneas não se travarem em ordem diferente.
func (s *Service) resolverVinculos(ctx context.Context, tx pgx.Tx, id uuid.UUID, r domain.Resolucao) ([]domain.Entrega, error) {
	if s.efeitos == nil {
		return nil, domain.ErrCascataIndisponivel
	}
	vinculos, err := s.repo.Vinculos(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(vinculos, func(a, b domain.Vinculo) int {
		if c := strings.Compare(a.Tipo, b.Tipo); c != 0 {
			return c
		}
		return strings.Compare(a.AlvoID.String(), b.AlvoID.String())
	})
	var entregas []domain.Entrega
	for _, v := range vinculos {
		if v.Tipo != domain.VinculoIncidente && v.Tipo != domain.VinculoProblema {
			continue
		}
		entrega, err := s.efeitos.Resolver(ctx, tx, v.Tipo, v.AlvoID, r)
		if err != nil {
			return nil, err
		}
		if entrega != nil {
			entregas = append(entregas, entrega)
		}
	}
	return entregas, nil
}

// Vincular liga (ou desfaz) um problema, incidente ou ativo da mesma
// unidade.
func (s *Service) Vincular(ctx context.Context, identity auth.Identity, id uuid.UUID, tipo string, alvo uuid.UUID, ligar bool) (domain.Mudanca, error) {
	return s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, m domain.Mudanca, ps []string) error {
		if !slices.Contains(ps, domain.PapelGestor) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.NovoVinculo(tipo, alvo); err != nil {
			return err
		}
		if m.Estado == domain.Cancelada {
			return domain.ErrMudancaEncerrada
		}
		mudou := false
		var err error
		if ligar {
			if s.portas == nil || !s.portas.Disponivel(tipo) {
				return domain.ErrIntegracaoDesligada
			}
			item, err := s.portas.Consultar(ctx, tx, tipo, alvo)
			if err != nil {
				return err
			}
			if tipo == domain.VinculoIncidente && item.Tipo != "incidente" {
				return domain.ErrVinculoInvalido
			}
			if !domain.MesmaUnidade(item.UnidadeID, m.UnidadeID) {
				return domain.ErrDeOutraUnidade
			}
			if mudou, err = s.repo.Vincular(ctx, tx, id, domain.Vinculo{Tipo: tipo, AlvoID: alvo, Rotulo: item.Rotulo}, identity.UserID); err != nil {
				return err
			}
		} else if mudou, err = s.repo.Desvincular(ctx, tx, id, tipo, alvo); err != nil {
			return err
		}
		if !mudou {
			return nil
		}
		v := map[string]any{"tipo": tipo, "alvo": alvo.String(), "ligado": ligar}
		return s.registrar(ctx, tx, EventoVinculo, m, nil, v, v)
	})
}

func (s *Service) conferirProduto(ctx context.Context, db database.DBTX, produto, unidade *uuid.UUID) error {
	if produto == nil {
		return nil
	}
	if s.produtos == nil {
		return domain.ErrIntegracaoDesligada
	}
	dona, err := s.produtos.Consultar(ctx, db, *produto)
	if err != nil {
		return err
	}
	if !domain.MesmaUnidade(dona, unidade) {
		return domain.ErrDeOutraUnidade
	}
	return nil
}

// registrar grava o evento (com o extra no payload) e a auditoria.
func (s *Service) registrar(ctx context.Context, tx pgx.Tx, evento string, m domain.Mudanca, antes, depois any, extra map[string]any) error {
	payload := map[string]any{"id": m.ID.String(), "numero": m.Numero, "estado": m.Estado, "tipo": m.Tipo}
	for k, v := range extra {
		payload[k] = v
	}
	if err := s.eventos.Write(ctx, tx, evento, "mudanca", m.ID.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, evento, "mudanca", m.ID.String(), antes, depois))
}

// ListarCongelamentos devolve as janelas do escopo e as institucionais.
func (s *Service) ListarCongelamentos(ctx context.Context, identity auth.Identity) ([]domain.Congelamento, error) {
	cob := auth.CoverageOf(identity, auth.PermMudancasManage)
	if !cob.All && len(cob.Unidades) == 0 {
		return []domain.Congelamento{}, nil
	}
	lista, err := s.repo.ListCongelamentos(ctx, s.pool, domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades})
	return lista, MapError(err)
}

// CriarCongelamento grava a janela em que mudança não é agendada.
func (s *Service) CriarCongelamento(ctx context.Context, identity auth.Identity, unidade *uuid.UUID, inicio, fim time.Time, motivo string) (domain.Congelamento, error) {
	motivo = strings.TrimSpace(motivo)
	if !fim.After(inicio) || utf8.RuneCountInString(motivo) > 4000 {
		return domain.Congelamento{}, MapError(domain.ErrEntradaInvalida)
	}
	if !slices.Contains(papeis(identity, unidade), domain.PapelGestor) {
		return domain.Congelamento{}, MapError(domain.ErrForaDoEscopo)
	}
	c := domain.Congelamento{ID: uuid.New(), UnidadeID: unidade, Inicio: inicio, Fim: fim, Motivo: motivo, CreatedAt: time.Now().UTC()}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if c, err = s.repo.InsertCongelamento(ctx, tx, c); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: "mudancas.congelamento.criado", ResourceType: "mudanca_congelamento", ResourceID: c.ID.String()})
	})
	return c, MapError(err)
}

// ExcluirCongelamento remove a janela.
func (s *Service) ExcluirCongelamento(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if !slices.Contains(papeis(identity, nil), domain.PapelGestor) && auth.CoverageOf(identity, auth.PermMudancasManage).All == false && len(auth.CoverageOf(identity, auth.PermMudancasManage).Unidades) == 0 {
		return MapError(domain.ErrForaDoEscopo)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteCongelamento(ctx, tx, id); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: "mudancas.congelamento.excluido", ResourceType: "mudanca_congelamento", ResourceID: id.String()})
	}))
}
