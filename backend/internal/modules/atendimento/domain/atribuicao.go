package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Atribuicao é um trecho em que o chamado esteve com uma pessoa e a sua
// equipe (ADR 073). Fim nulo: ainda está com ela.
type Atribuicao struct {
	SolicitacaoID uuid.UUID
	EquipeID      uuid.UUID
	EquipeNome    string
	UnidadeID     *uuid.UUID
	OLAHoras      *int
	Inicio        time.Time
	Fim           *time.Time
}

// TempoDaEquipe é o tempo do trecho em que a vez era da equipe: tira o
// que o chamado passou esperando o servidor ou o fornecedor. Trecho
// aberto conta até agora.
func TempoDaEquipe(a Atribuicao, passagens []Passagem, agora time.Time) time.Duration {
	fim := agora
	if a.Fim != nil {
		fim = *a.Fim
	}
	total := fim.Sub(a.Inicio)
	ps := slices.Clone(passagens)
	slices.SortStableFunc(ps, func(x, y Passagem) int { return x.Em.Compare(y.Em) })
	var desde *time.Time
	pausa := func(ate time.Time) {
		ini, f := maxTempo(*desde, a.Inicio), minTempo(ate, fim)
		if f.After(ini) {
			total -= f.Sub(ini)
		}
		desde = nil
	}
	for _, p := range ps {
		if desde != nil {
			pausa(p.Em)
		}
		if p.Para == EstadoAguardandoServidor || p.Para == EstadoAguardandoTerceiro {
			em := p.Em
			desde = &em
		}
	}
	if desde != nil {
		pausa(fim)
	}
	return max(total, 0)
}

func maxTempo(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTempo(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
