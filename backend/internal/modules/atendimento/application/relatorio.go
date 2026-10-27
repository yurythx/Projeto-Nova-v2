package application

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// DiasDoRelatorio é a janela padrão e o teto do relatório por secretaria.
const (
	DiasDoRelatorio    = 90
	MaxDiasDoRelatorio = 366
)

// Relatorio é o relatório por secretaria e o OLA das equipes no período.
type Relatorio struct {
	Desde       time.Time                `json:"desde"`
	Dias        int                      `json:"dias"`
	Truncado    bool                     `json:"truncado"`
	Total       domain.LinhaSecretaria   `json:"total"`
	Secretarias []domain.LinhaSecretaria `json:"secretarias"`
	Equipes     []domain.LinhaEquipe     `json:"equipes"`
	// Contratos é o cumprimento dos contratos de apoio (ADR 071).
	Contratos []domain.LinhaContrato `json:"contratos"`
}

// RelatorioPorSecretaria mede o escopo de quem atende nos últimos dias:
// volume, prazo, primeiro contato e experiência por secretaria, e o OLA
// de cada equipe. A amostra tem o mesmo teto da governança; acima dele,
// ficam as mais recentes e a resposta avisa.
func (s *Service) RelatorioPorSecretaria(ctx context.Context, identity auth.Identity, dias int) (Relatorio, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return Relatorio{}, MapError(domain.ErrForaDoEscopo)
	}
	if dias <= 0 {
		dias = DiasDoRelatorio
	}
	dias = min(dias, MaxDiasDoRelatorio)
	agora := s.agora().UTC()
	desde := agora.AddDate(0, 0, -dias)
	restrito := !cob.All
	lista, err := s.repo.ListarDoPeriodo(ctx, s.pool, restrito, cob.Unidades, desde, TetoGovernanca+1)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	out := Relatorio{Desde: desde, Dias: dias, Truncado: len(lista) > TetoGovernanca}
	if out.Truncado {
		lista = lista[:TetoGovernanca]
	}
	if err := s.medir(ctx, lista); err != nil {
		return Relatorio{}, MapError(err)
	}
	passagens, err := s.passagensConcluidas(ctx, lista)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	ids, unidades := make([]uuid.UUID, 0, len(lista)), []uuid.UUID{}
	for _, sol := range lista {
		ids = append(ids, sol.ID)
		if sol.UnidadeID != nil && !slices.Contains(unidades, *sol.UnidadeID) {
			unidades = append(unidades, *sol.UnidadeID)
		}
	}
	avaliacoes, err := s.repo.AvaliacoesDe(ctx, s.pool, ids)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	secretarias, err := s.repo.SecretariasDe(ctx, s.pool, unidades)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	metas, err := s.repo.MetasXLA(ctx, s.pool)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	out.Total = domain.LinhaDe(lista, passagens, avaliacoes, metas)
	out.Total.Nome = "Total"
	out.Secretarias = domain.PorSecretaria(lista, secretarias, passagens, avaliacoes, metas)
	tarefas, err := s.repo.TarefasDoPeriodo(ctx, s.pool, restrito, cob.Unidades, desde)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	atribuicoes, err := s.repo.AtribuicoesDoPeriodo(ctx, s.pool, restrito, cob.Unidades, desde)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	comEquipe := map[uuid.UUID][]domain.Passagem{}
	if len(atribuicoes) > 0 {
		ids := []uuid.UUID{}
		for _, a := range atribuicoes {
			if !slices.Contains(ids, a.SolicitacaoID) {
				ids = append(ids, a.SolicitacaoID)
			}
		}
		if comEquipe, err = s.repo.PassagensDe(ctx, s.pool, ids); err != nil {
			return Relatorio{}, MapError(err)
		}
	}
	out.Equipes = domain.ResumirEquipes(tarefas, atribuicoes, comEquipe, agora)
	comContrato, err := s.repo.ComContratoNoPeriodo(ctx, s.pool, restrito, cob.Unidades, desde)
	if err != nil {
		return Relatorio{}, MapError(err)
	}
	out.Contratos = []domain.LinhaContrato{}
	if len(comContrato) > 0 {
		ids := make([]uuid.UUID, 0, len(comContrato))
		for _, sol := range comContrato {
			ids = append(ids, sol.ID)
		}
		ps, err := s.repo.PassagensDe(ctx, s.pool, ids)
		if err != nil {
			return Relatorio{}, MapError(err)
		}
		out.Contratos = domain.ResumirContratos(comContrato, ps, agora)
	}
	return out, nil
}
