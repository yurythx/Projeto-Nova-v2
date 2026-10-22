package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/notificacoes"
)

const (
	EventoPoliticaSalva           = "atendimento.politica.salva"
	EventoFeriadoCriado           = "atendimento.feriado.criado"
	EventoFeriadoExcluido         = "atendimento.feriado.excluido"
	EventoSolicitacaoClassificada = "atendimento.solicitacao.classificada"
)

// IntervaloAlertas é de quanto em quanto tempo o worker confere os prazos
// (variável só para os testes encurtarem).
var IntervaloAlertas = time.Minute

// PoliticaEntrada é a política de prazo de uma unidade.
type PoliticaEntrada struct {
	UnidadeID *uuid.UUID
	Fuso      string
	Turnos    []domain.Turno
	Matriz    []domain.Celula
	Metas     []domain.Meta
}

// FeriadoEntrada é um feriado ou ponto facultativo.
type FeriadoEntrada struct {
	UnidadeID        *uuid.UUID
	Data, Nome, Tipo string
}

// agora é o relógio do serviço (substituível em teste).
func (s *Service) agora() time.Time {
	if s.relogio != nil {
		return s.relogio()
	}
	return time.Now()
}

// WithRelogio fixa o "agora" do serviço — só para teste.
func (s *Service) WithRelogio(fn func() time.Time) *Service {
	s.relogio = fn
	return s
}

// ObterPolitica devolve a política que vale para a unidade: a dela ou,
// marcada como herdada, a de cima. Nil quando nenhuma foi cadastrada.
func (s *Service) ObterPolitica(ctx context.Context, identity auth.Identity, unidade *uuid.UUID) (*domain.Politica, error) {
	if !ver(identity, unidade) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	p, err := s.repo.ResolverPolitica(ctx, s.pool, unidade)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil, nil
	}
	if err != nil {
		return nil, MapError(err)
	}
	return &p, nil
}

// SalvarPolitica grava a política da unidade. Vale para as solicitações
// classificadas daqui em diante; as já classificadas mantêm o compromisso.
func (s *Service) SalvarPolitica(ctx context.Context, identity auth.Identity, in PoliticaEntrada) (domain.Politica, error) {
	if !ver(identity, in.UnidadeID) {
		return domain.Politica{}, MapError(domain.ErrForaDoEscopo)
	}
	p, err := domain.NovaPolitica(in.UnidadeID, in.Fuso, in.Turnos, in.Matriz, in.Metas)
	if err != nil {
		return domain.Politica{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var antes any
		if prev, err := s.repo.GetPolitica(ctx, tx, in.UnidadeID); err == nil {
			antes, p.ID = prev, prev.ID
		} else if !errors.Is(err, domain.ErrNaoEncontrado) {
			return err
		}
		if p, err = s.repo.SalvarPolitica(ctx, tx, p); err != nil {
			return err
		}
		return s.registrar(ctx, tx, EventoPoliticaSalva, "politica", p.ID, map[string]any{"id": p.ID.String()}, p, antes)
	})
	return p, MapError(err)
}

// ListarFeriados devolve os feriados cadastrados na unidade.
func (s *Service) ListarFeriados(ctx context.Context, identity auth.Identity, unidade *uuid.UUID) ([]domain.Feriado, error) {
	if !ver(identity, unidade) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.ListFeriados(ctx, s.pool, unidade)
	return lista, MapError(err)
}

// CriarFeriado cadastra o feriado da unidade.
func (s *Service) CriarFeriado(ctx context.Context, identity auth.Identity, in FeriadoEntrada) (domain.Feriado, error) {
	f, err := domain.NovoFeriado(in.UnidadeID, in.Data, in.Nome, in.Tipo)
	if err != nil {
		return domain.Feriado{}, MapError(err)
	}
	if !ver(identity, f.UnidadeID) {
		return domain.Feriado{}, MapError(domain.ErrForaDoEscopo)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if f, err = s.repo.InsertFeriado(ctx, tx, f); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoFeriadoCriado, "feriado", f.ID, f, nil)
	})
	return f, MapError(err)
}

// ExcluirFeriado remove o feriado.
func (s *Service) ExcluirFeriado(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		f, err := s.repo.GetFeriado(ctx, tx, id)
		if err != nil {
			return err
		}
		if !ver(identity, f.UnidadeID) {
			return domain.ErrNaoEncontrado
		}
		if err := s.repo.DeleteFeriado(ctx, tx, id); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoFeriadoExcluido, "feriado", f.ID, nil, f)
	})
	return MapError(err)
}

// politicaDe é a política que vale para a unidade, ou nil.
func (s *Service) politicaDe(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (*domain.Politica, error) {
	p, err := s.repo.ResolverPolitica(ctx, db, unidade)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Classificar muda impacto e urgência e, com eles, a prioridade e as
// metas da solicitação pela política de agora.
func (s *Service) Classificar(ctx context.Context, identity auth.Identity, id uuid.UUID, impacto, urgencia int) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrForaDoEscopo
		}
		p, err := s.politicaDe(ctx, tx, sol.UnidadeID)
		if err != nil {
			return nil, err
		}
		next, mudou, err := sol.Classificar(impacto, urgencia, p)
		if err != nil || !mudou {
			return nil, err
		}
		if err := s.repo.UpdateSolicitacao(ctx, tx, next); err != nil {
			return nil, err
		}
		return nil, s.registrar(ctx, tx, EventoSolicitacaoClassificada, "solicitacao", next.ID, payloadDe(next), next, sol)
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}

// medir preenche o prazo das solicitações, com um calendário por
// unidade. Sem política, a solicitação fica sem prazo.
func (s *Service) medir(ctx context.Context, lista []domain.Solicitacao) error {
	ids := []uuid.UUID{}
	for _, sol := range lista {
		if sol.Prioridade != nil {
			ids = append(ids, sol.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	passagens, err := s.repo.PassagensDe(ctx, s.pool, ids)
	if err != nil {
		return err
	}
	agora := s.agora()
	calendarios := map[string]*domain.Calendario{}
	for i, sol := range lista {
		if sol.Prioridade == nil {
			continue
		}
		cal, err := s.calendarioDe(ctx, calendarios, sol.UnidadeID, sol.CreatedAt, agora)
		if err != nil {
			return err
		}
		if cal != nil {
			lista[i].Prazo = domain.MedirPrazo(sol, passagens[sol.ID], *cal, agora)
		}
	}
	return nil
}

// calendarioDe monta (uma vez por unidade) o calendário com os feriados
// do período que o relógio e a previsão de vencimento podem alcançar.
func (s *Service) calendarioDe(ctx context.Context, cache map[string]*domain.Calendario, unidade *uuid.UUID, desde, agora time.Time) (*domain.Calendario, error) {
	chave := "institucional"
	if unidade != nil {
		chave = unidade.String()
	}
	if cal, ok := cache[chave]; ok {
		return cal, nil
	}
	p, err := s.politicaDe(ctx, s.pool, unidade)
	if err != nil || p == nil {
		cache[chave] = nil
		return nil, err
	}
	// Uma janela larga cobre todas as solicitações da página e o horizonte
	// da previsão; poucos feriados por ano tornam isso barato.
	de := desde.AddDate(-1, 0, 0).Format(time.DateOnly)
	ate := agora.AddDate(1, 2, 0).Format(time.DateOnly)
	feriados, err := s.repo.FeriadosVigentes(ctx, s.pool, unidade, de, ate)
	if err != nil {
		return nil, err
	}
	cal, err := domain.NovoCalendario(*p, feriados)
	if err != nil {
		return nil, err
	}
	cache[chave] = &cal
	return &cal, nil
}

// RunAlertas confere os prazos a cada IntervaloAlertas até ctx acabar.
func (s *Service) RunAlertas(ctx context.Context) error {
	t := time.NewTicker(IntervaloAlertas)
	defer t.Stop()
	for {
		if err := s.VerificarPrazos(ctx); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "atendimento: verificação de prazos falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

var nomeRelogio = map[string]string{"resposta": "resposta", "solucao": "solução"}

// VerificarPrazos avisa o responsável (ou, sem ele, a equipe da unidade)
// quando um relógio correndo cruza 75%, 90% e 100% da meta. A chave do
// aviso inclui a solicitação, o relógio e o limiar: conferir de novo, em
// qualquer réplica, não repete o aviso.
func (s *Service) VerificarPrazos(ctx context.Context) error {
	if s.avisos == nil {
		return nil
	}
	lista, err := s.repo.ComPrazoCorrendo(ctx, s.pool)
	if err != nil {
		return err
	}
	if err := s.medir(ctx, lista); err != nil {
		return err
	}
	equipes := map[string][]uuid.UUID{}
	var enviadas []notificacoes.Enviada
	for _, sol := range lista {
		if sol.Prazo == nil {
			continue
		}
		for _, r := range []struct {
			chave   string
			relogio domain.Relogio
		}{{"resposta", sol.Prazo.Resposta}, {"solucao", sol.Prazo.Solucao}} {
			limiar := r.relogio.Limiar()
			if limiar == 0 {
				continue
			}
			destinos, err := s.destinosDoPrazo(ctx, equipes, sol)
			if err != nil {
				return err
			}
			titulo := fmt.Sprintf("Prazo de %s da solicitação #%d em %d%%", nomeRelogio[r.chave], sol.Numero, limiar)
			if limiar >= domain.LimiarVencido {
				titulo = fmt.Sprintf("Prazo de %s da solicitação #%d vencido", nomeRelogio[r.chave], sol.Numero)
			}
			e, err := notificacoes.Registrar(ctx, s.pool, destinos, notificacoes.Nova{
				Modulo: "atendimento", Titulo: titulo, Mensagem: sol.OfertaNome + ": " + sol.Titulo,
				Link:  "/atendimento/solicitacoes/" + sol.ID.String(),
				Chave: fmt.Sprintf("atendimento:%s:prazo:%s:%d", sol.ID, r.chave, limiar),
			})
			if err != nil {
				return err
			}
			enviadas = append(enviadas, e...)
		}
	}
	s.entregar(ctx, enviadas)
	return nil
}

func (s *Service) destinosDoPrazo(ctx context.Context, equipes map[string][]uuid.UUID, sol domain.Solicitacao) ([]uuid.UUID, error) {
	if sol.ResponsavelID != nil {
		return []uuid.UUID{*sol.ResponsavelID}, nil
	}
	chave := "institucional"
	if sol.UnidadeID != nil {
		chave = sol.UnidadeID.String()
	}
	if ids, ok := equipes[chave]; ok {
		return ids, nil
	}
	membros, err := s.repo.Responsaveis(ctx, s.pool, sol.UnidadeID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(membros))
	for _, m := range membros {
		ids = append(ids, m.ID)
	}
	equipes[chave] = ids
	return ids, nil
}
