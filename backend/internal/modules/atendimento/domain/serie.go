package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// FusoPadrao é o fuso do expediente quando a unidade ainda não tem política.
const FusoPadrao = "America/Sao_Paulo"

// DiaNoFuso devolve o dia civil de agora. O instante volta à meia-noite
// UTC dessa data, para gravar um DATE sem depender do fuso da sessão.
// Fuso vazio, "Local" ou desconhecido cai em FusoPadrao.
func DiaNoFuso(agora time.Time, fuso string) time.Time {
	fuso = strings.TrimSpace(fuso)
	loc, err := time.LoadLocation(fuso)
	if err != nil || fuso == "" || fuso == "Local" {
		loc, err = time.LoadLocation(FusoPadrao)
		if err != nil {
			loc = time.UTC
		}
	}
	ano, mes, dia := agora.In(loc).Date()
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

// Retrato é o retrato de um dia. Ponteiro nulo é amostra vazia, não zero.
// As somas servem para reunir unidades sem tirar média de médias.
type Retrato struct {
	Dia           time.Time  `json:"dia"`
	UnidadeID     *uuid.UUID `json:"-"`
	MTTRHoras     *float64   `json:"mttr_horas"`
	MTBFHoras     *float64   `json:"mtbf_horas"`
	SLAPercentual *float64   `json:"sla_percentual"`
	FCRPercentual *float64   `json:"fcr_percentual"`
	CSATMedia     *float64   `json:"csat_media"`
	SomaMTTRMin   int        `json:"-"`
	MTTRN         int        `json:"-"`
	SLACumprido   int        `json:"-"`
	SLAN          int        `json:"-"`
	FCRN          int        `json:"-"`
	Concluidas    int        `json:"-"`
	Incidentes    int        `json:"-"`
	CSATSoma      int        `json:"-"`
	CSATN         int        `json:"-"`
	Truncado      bool       `json:"truncado"`
}

// AgregarSerie reúne os pontos do mesmo dia. Um ponto só conserva o MTBF.
// Vários pontos não têm um intervalo único entre falhas, então o MTBF fica
// vazio. As outras médias saem das somas.
func AgregarSerie(pontos []Retrato) []Retrato {
	if len(pontos) == 0 {
		return []Retrato{}
	}
	ordem := []string{}
	porDia := map[string][]Retrato{}
	for _, p := range pontos {
		chave := p.Dia.UTC().Format("2006-01-02")
		if _, ok := porDia[chave]; !ok {
			ordem = append(ordem, chave)
		}
		porDia[chave] = append(porDia[chave], p)
	}
	out := make([]Retrato, 0, len(ordem))
	for _, chave := range ordem {
		grupo := porDia[chave]
		if len(grupo) == 1 {
			out = append(out, grupo[0])
			continue
		}
		var soma Retrato
		soma.Dia = grupo[0].Dia
		for _, p := range grupo {
			soma.SomaMTTRMin += p.SomaMTTRMin
			soma.MTTRN += p.MTTRN
			soma.SLACumprido += p.SLACumprido
			soma.SLAN += p.SLAN
			soma.FCRN += p.FCRN
			soma.Concluidas += p.Concluidas
			soma.Incidentes += p.Incidentes
			soma.CSATSoma += p.CSATSoma
			soma.CSATN += p.CSATN
			soma.Truncado = soma.Truncado || p.Truncado
		}
		if soma.MTTRN > 0 {
			soma.MTTRHoras = arredondar(float64(soma.SomaMTTRMin) / float64(soma.MTTRN) / 60)
		}
		if soma.SLAN > 0 {
			soma.SLAPercentual = percentual(soma.SLACumprido, soma.SLAN)
		}
		if soma.Concluidas > 0 {
			soma.FCRPercentual = percentual(soma.FCRN, soma.Concluidas)
		}
		if soma.CSATN > 0 {
			soma.CSATMedia = arredondar(float64(soma.CSATSoma) / float64(soma.CSATN))
		}
		out = append(out, soma)
	}
	return out
}
