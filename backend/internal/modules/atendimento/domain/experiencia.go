package domain

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Experiência de quem pede (XLA, ADR 069). Os indicadores da equipe
// (MTTR, SLA, FCR) dizem se o processo cumpriu o combinado; estes dizem
// como foi para a pessoa: nota, facilidade, ter de reabrir, ser chamada a
// responder de novo e quanto tempo de relógio esperou.

var (
	ErrFacilidadeInvalida = errors.New("atendimento: a facilidade vai de 1 a 5")
	ErrMetasInvalidas     = errors.New("atendimento: meta de experiência fora da faixa")
)

// MetasXLA são as metas de experiência da organização. Nula = sem meta.
type MetasXLA struct {
	CSATMin        *float64   `json:"csat_min"`
	FacilidadeMin  *float64   `json:"facilidade_min"`
	ReaberturaMax  *float64   `json:"reabertura_max"`
	IdasMax        *float64   `json:"idas_max"`
	EsperaMaxHoras *int       `json:"espera_max_horas"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// Validar confere as faixas, as mesmas do banco.
func (m MetasXLA) Validar() error {
	fora := func(v *float64, min, max float64) bool { return v != nil && (*v < min || *v > max) }
	if fora(m.CSATMin, 1, 5) || fora(m.FacilidadeMin, 1, 5) || fora(m.ReaberturaMax, 0, 100) || fora(m.IdasMax, 0, 20) ||
		(m.EsperaMaxHoras != nil && (*m.EsperaMaxHoras < 1 || *m.EsperaMaxHoras > 8760)) {
		return ErrMetasInvalidas
	}
	return nil
}

// MedidaXLA é um indicador com a sua meta. Atingida fica nula sem
// amostra ou sem meta: não se afirma o que não se mediu.
type MedidaXLA struct {
	Valor       *float64 `json:"valor"`
	Meta        *float64 `json:"meta"`
	Atingida    *bool    `json:"atingida"`
	Amostra     int      `json:"amostra"`
	MaiorMelhor bool     `json:"maior_melhor"`
}

// Experiencia reúne as medidas de experiência do escopo.
type Experiencia struct {
	CSAT        MedidaXLA `json:"csat"`
	Facilidade  MedidaXLA `json:"facilidade"`
	Reabertura  MedidaXLA `json:"reabertura"`
	IdasEVoltas MedidaXLA `json:"idas_e_voltas"`
	EsperaHoras MedidaXLA `json:"espera_horas"`
}

// ComFacilidade acrescenta a resposta opcional à avaliação.
func (a Avaliacao) ComFacilidade(f *int) (Avaliacao, error) {
	if f != nil && (*f < 1 || *f > 5) {
		return Avaliacao{}, ErrFacilidadeInvalida
	}
	a.Facilidade = f
	return a, nil
}

// MedirExperiencia calcula as medidas a partir das solicitações concluídas
// com histórico, do resumo das avaliações e das metas. A espera é a
// mediana, em horas corridas, da abertura à conclusão: o relógio de quem
// pediu, não o expediente da equipe.
func MedirExperiencia(lista []Solicitacao, passagens map[uuid.UUID][]Passagem, r ResumoAvaliacoes, metas MetasXLA) Experiencia {
	comHistorico, reabertas, idas := 0, 0, 0
	esperas := []float64{}
	for _, s := range lista {
		if !concluida(s) {
			continue
		}
		if s.ResolvidaEm != nil {
			esperas = append(esperas, s.ResolvidaEm.Sub(s.CreatedAt).Hours())
		}
		ps := passagens[s.ID]
		if len(ps) == 0 {
			continue
		}
		comHistorico++
		reaberta := false
		for _, p := range ps {
			if p.Para == EstadoAguardandoServidor {
				idas++
			}
			if p.Para == EstadoEmAtendimento && p.De != nil && (*p.De == EstadoResolvido || *p.De == EstadoAtendido) {
				reaberta = true
			}
		}
		if reaberta {
			reabertas++
		}
	}
	var out Experiencia
	if r.Total > 0 {
		out.CSAT.Valor = arredondar(float64(r.Soma) / float64(r.Total))
	}
	out.CSAT.Amostra = r.Total
	if r.TotalFacilidade > 0 {
		out.Facilidade.Valor = arredondar(float64(r.SomaFacilidade) / float64(r.TotalFacilidade))
	}
	out.Facilidade.Amostra = r.TotalFacilidade
	if comHistorico > 0 {
		out.Reabertura.Valor = percentual(reabertas, comHistorico)
		out.IdasEVoltas.Valor = arredondar(float64(idas) / float64(comHistorico))
	}
	out.Reabertura.Amostra, out.IdasEVoltas.Amostra = comHistorico, comHistorico
	if len(esperas) > 0 {
		out.EsperaHoras.Valor = arredondar(mediana(esperas))
	}
	out.EsperaHoras.Amostra = len(esperas)

	var espera *float64
	if metas.EsperaMaxHoras != nil {
		h := float64(*metas.EsperaMaxHoras)
		espera = &h
	}
	out.CSAT = comMeta(out.CSAT, metas.CSATMin, true)
	out.Facilidade = comMeta(out.Facilidade, metas.FacilidadeMin, true)
	out.Reabertura = comMeta(out.Reabertura, metas.ReaberturaMax, false)
	out.IdasEVoltas = comMeta(out.IdasEVoltas, metas.IdasMax, false)
	out.EsperaHoras = comMeta(out.EsperaHoras, espera, false)
	return out
}

func comMeta(m MedidaXLA, meta *float64, maiorMelhor bool) MedidaXLA {
	m.Meta, m.MaiorMelhor = meta, maiorMelhor
	if m.Valor != nil && meta != nil {
		ok := *m.Valor <= *meta
		if maiorMelhor {
			ok = *m.Valor >= *meta
		}
		m.Atingida = &ok
	}
	return m
}

func mediana(v []float64) float64 {
	s := slices.Clone(v)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
