package domain

import (
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Indicadores são as médias do escopo, calculadas com o relógio de prazo
// já medido. Ponteiro nulo significa amostra insuficiente: a tela não
// inventa número.
type Indicadores struct {
	MTTRHoras     *float64
	MTBFHoras     *float64
	SLAPercentual *float64
	FCRPercentual *float64
	SomaMTTRMin   int
	SLACumprido   int
	Amostra       AmostraIndicadores
}

// AmostraIndicadores diz de quantos registros cada média saiu.
type AmostraIndicadores struct {
	Incidentes           int
	IncidentesResolvidos int
	ComPrazoConcluido    int
	Concluidas           int
	FCR                  int
	Truncado             bool
}

// Resumir calcula MTTR, MTBF, conformidade de SLA e resolução no primeiro
// contato a partir das solicitações já medidas e das passagens.
//
// MTTR é a média do tempo útil da solução dos incidentes concluídos que
// tinham prazo. MTBF é o intervalo médio entre aberturas de incidentes
// (não cancelados), e exige pelo menos dois. O SLA conta só a solução
// parada de solicitações concluídas (cumprida ou descumprida). O primeiro
// contato é a conclusão que não passou por espera nem foi reaberta.
func Resumir(lista []Solicitacao, passagens map[uuid.UUID][]Passagem) Indicadores {
	var ind Indicadores
	var somaMTTR int
	var slaCumprido int
	var incidentes []Solicitacao
	for _, s := range lista {
		if s.Tipo == TipoIncidente && s.Estado != EstadoCancelado {
			incidentes = append(incidentes, s)
		}
		if !concluida(s) {
			continue
		}
		ind.Amostra.Concluidas++
		if primeiroContato(passagens[s.ID]) {
			ind.Amostra.FCR++
		}
		if s.Tipo == TipoIncidente && s.Prazo != nil && s.Prazo.Solucao.Estado == RelogioParado {
			somaMTTR += s.Prazo.Solucao.UsadoMin
			ind.Amostra.IncidentesResolvidos++
		}
		if s.Prazo != nil && (s.Prazo.Solucao.Situacao == SituacaoCumprido || s.Prazo.Solucao.Situacao == SituacaoDescumprido) {
			ind.Amostra.ComPrazoConcluido++
			if s.Prazo.Solucao.Situacao == SituacaoCumprido {
				slaCumprido++
			}
		}
	}
	ind.Amostra.Incidentes = len(incidentes)
	ind.SomaMTTRMin = somaMTTR
	ind.SLACumprido = slaCumprido
	if ind.Amostra.IncidentesResolvidos > 0 {
		ind.MTTRHoras = arredondar(float64(somaMTTR) / float64(ind.Amostra.IncidentesResolvidos) / 60)
	}
	if ind.Amostra.ComPrazoConcluido > 0 {
		ind.SLAPercentual = percentual(slaCumprido, ind.Amostra.ComPrazoConcluido)
	}
	if ind.Amostra.Concluidas > 0 {
		ind.FCRPercentual = percentual(ind.Amostra.FCR, ind.Amostra.Concluidas)
	}
	if len(incidentes) >= 2 {
		slices.SortFunc(incidentes, func(a, b Solicitacao) int {
			if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
				return c
			}
			if a.Numero < b.Numero {
				return -1
			}
			if a.Numero > b.Numero {
				return 1
			}
			return 0
		})
		var soma time.Duration
		for i := 1; i < len(incidentes); i++ {
			soma += incidentes[i].CreatedAt.Sub(incidentes[i-1].CreatedAt)
		}
		ind.MTBFHoras = arredondar(soma.Hours() / float64(len(incidentes)-1))
	}
	return ind
}

func concluida(s Solicitacao) bool {
	switch s.Estado {
	case EstadoResolvido, EstadoAtendido, EstadoEncerrado:
		return true
	default:
		return false
	}
}

// primeiroContato exige o histórico: sem passagem, não há como afirmar.
// Espera do servidor, espera de terceiro ou mais de uma entrada em
// atendimento (reabertura) tiram o chamado do primeiro contato.
func primeiroContato(passagens []Passagem) bool {
	if len(passagens) == 0 {
		return false
	}
	emAtendimento := 0
	for _, p := range passagens {
		switch p.Para {
		case EstadoAguardandoServidor, EstadoAguardandoTerceiro:
			return false
		case EstadoEmAtendimento:
			emAtendimento++
		}
	}
	return emAtendimento <= 1
}

func arredondar(v float64) *float64 {
	r := math.Round(v*10) / 10
	return &r
}

func percentual(parte, total int) *float64 {
	r := math.Round(float64(parte)/float64(total)*1000) / 10
	return &r
}
