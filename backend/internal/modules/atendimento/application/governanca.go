package application

import (
	"context"
	"math"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// tetoGovernanca limita a amostra dos indicadores. Acima disso a resposta
// marca a amostra como truncada em vez de fingir que viu a fila inteira.
// TetoGovernanca limita a amostra. O teste abaixa o teto para marcar
// o truncamento sem cadastrar milhares de solicitações.
var TetoGovernanca = 5000

// Governanca é o que a central mostra sem recalcular no navegador.
type Governanca struct {
	CSAT          *CSAT    `json:"csat,omitempty"`
	MTTRHoras     *float64 `json:"mttr_horas"`
	MTBFHoras     *float64 `json:"mtbf_horas"`
	SLAPercentual *float64 `json:"sla_percentual"`
	FCRPercentual *float64 `json:"fcr_percentual"`
	Amostra       Amostra  `json:"amostra"`
}

// CSAT é a média das notas de 1 a 5 e as avaliações mais recentes.
type CSAT struct {
	Media    float64            `json:"media"`
	Total    int                `json:"total"`
	Recentes []domain.Avaliacao `json:"recentes"`
}

// Amostra explica de onde saiu cada média.
type Amostra struct {
	Incidentes           int  `json:"incidentes"`
	IncidentesResolvidos int  `json:"incidentes_resolvidos"`
	ComPrazoConcluido    int  `json:"com_prazo_concluido"`
	Concluidas           int  `json:"concluidas"`
	FCR                  int  `json:"fcr"`
	Truncado             bool `json:"truncado"`
}

// Governanca mede o escopo de quem atende. Sem cobertura, a resposta vem
// vazia: não há amostra, e nenhum indicador é preenchido com constante.
func (s *Service) Governanca(ctx context.Context, identity auth.Identity) (Governanca, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return Governanca{}, nil
	}
	restrito := !cob.All
	lista, err := s.repo.ListarParaGovernanca(ctx, s.pool, restrito, cob.Unidades, TetoGovernanca+1)
	if err != nil {
		return Governanca{}, MapError(err)
	}
	truncado := len(lista) > TetoGovernanca
	if truncado {
		lista = lista[:TetoGovernanca]
	}
	if err := s.medir(ctx, lista); err != nil {
		return Governanca{}, MapError(err)
	}
	passagens, err := s.passagensConcluidas(ctx, lista)
	if err != nil {
		return Governanca{}, MapError(err)
	}
	ind := domain.Resumir(lista, passagens)
	ind.Amostra.Truncado = truncado
	resumo, err := s.repo.ResumoAvaliacoes(ctx, s.pool, restrito, cob.Unidades, 5)
	if err != nil {
		return Governanca{}, MapError(err)
	}
	out := Governanca{
		MTTRHoras: ind.MTTRHoras, MTBFHoras: ind.MTBFHoras,
		SLAPercentual: ind.SLAPercentual, FCRPercentual: ind.FCRPercentual,
		Amostra: Amostra{
			Incidentes: ind.Amostra.Incidentes, IncidentesResolvidos: ind.Amostra.IncidentesResolvidos,
			ComPrazoConcluido: ind.Amostra.ComPrazoConcluido, Concluidas: ind.Amostra.Concluidas,
			FCR: ind.Amostra.FCR, Truncado: ind.Amostra.Truncado,
		},
	}
	if resumo.Total > 0 {
		media := math.Round(float64(resumo.Soma)/float64(resumo.Total)*10) / 10
		out.CSAT = &CSAT{Media: media, Total: resumo.Total, Recentes: resumo.Recentes}
	}
	return out, nil
}

func (s *Service) passagensConcluidas(ctx context.Context, lista []domain.Solicitacao) (map[uuid.UUID][]domain.Passagem, error) {
	ids := make([]uuid.UUID, 0)
	for _, sol := range lista {
		switch sol.Estado {
		case domain.EstadoResolvido, domain.EstadoAtendido, domain.EstadoEncerrado:
			ids = append(ids, sol.ID)
		}
	}
	if len(ids) == 0 {
		return map[uuid.UUID][]domain.Passagem{}, nil
	}
	return s.repo.PassagensDe(ctx, s.pool, ids)
}
