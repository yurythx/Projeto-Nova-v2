package domain

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Contrato de apoio (UC, ADR 071): o chamado que espera o fornecedor
// aponta para o contrato, e o tempo em "aguardando terceiro" se mede
// contra o prazo que o fornecedor assumiu.

var (
	ErrContratosDesligado = errors.New("atendimento: o plug-in Contratos está desligado")
	ErrContratoAusente    = errors.New("atendimento: contrato inexistente, fora de vigência ou fora do alcance")
)

// ContratoDeApoio é o que o atendimento precisa saber do contrato.
type ContratoDeApoio struct {
	ID         uuid.UUID `json:"id"`
	Rotulo     string    `json:"rotulo"`
	Fornecedor string    `json:"fornecedor"`
	PrazoHoras *int      `json:"prazo_horas"`
}

// LinhaContrato é o cumprimento de um contrato de apoio no período.
// Medidos são os chamados que já voltaram do fornecedor; os que estão com
// ele agora contam à parte, e atrasados são os que já passaram do prazo.
type LinhaContrato struct {
	ContratoID        uuid.UUID `json:"contrato_id"`
	Rotulo            string    `json:"rotulo"`
	PrazoHoras        *int      `json:"prazo_horas"`
	Pedidos           int       `json:"pedidos"`
	Medidos           int       `json:"medidos"`
	NoPrazo           int       `json:"no_prazo"`
	NoPrazoPercentual *float64  `json:"no_prazo_percentual"`
	MedianaHoras      *float64  `json:"mediana_horas"`
	ComFornecedor     int       `json:"com_fornecedor"`
	Atrasados         int       `json:"atrasados"`
}

// TempoComTerceiro soma o tempo que o chamado passou em "aguardando
// terceiro", na ordem das passagens. Se ainda está lá, conta até agora e
// diz que segue com o fornecedor.
func TempoComTerceiro(s Solicitacao, passagens []Passagem, agora time.Time) (time.Duration, bool) {
	ps := slices.Clone(passagens)
	slices.SortStableFunc(ps, func(a, b Passagem) int { return a.Em.Compare(b.Em) })
	var total time.Duration
	var desde *time.Time
	for _, p := range ps {
		if desde != nil {
			total += p.Em.Sub(*desde)
			desde = nil
		}
		if p.Para == EstadoAguardandoTerceiro {
			em := p.Em
			desde = &em
		}
	}
	aberto := desde != nil && s.Estado == EstadoAguardandoTerceiro
	if aberto {
		total += agora.Sub(*desde)
	}
	return total, aberto
}

// ResumirContratos mede cada contrato de apoio pelos chamados que apontam
// para ele. O prazo é o copiado no chamado quando foi vinculado.
func ResumirContratos(lista []Solicitacao, passagens map[uuid.UUID][]Passagem, agora time.Time) []LinhaContrato {
	idx := map[uuid.UUID]int{}
	out := []LinhaContrato{}
	horas := map[uuid.UUID][]float64{}
	for _, s := range lista {
		if s.ContratoApoioID == nil {
			continue
		}
		id := *s.ContratoApoioID
		i, ok := idx[id]
		if !ok {
			i = len(out)
			idx[id] = i
			out = append(out, LinhaContrato{ContratoID: id, Rotulo: s.ContratoApoio, PrazoHoras: s.ContratoPrazoHoras})
		}
		l := &out[i]
		l.Pedidos++
		tempo, aberto := TempoComTerceiro(s, passagens[s.ID], agora)
		fora := s.ContratoPrazoHoras != nil && tempo > time.Duration(*s.ContratoPrazoHoras)*time.Hour
		switch {
		case aberto:
			l.ComFornecedor++
			if fora {
				l.Atrasados++
			}
		case tempo > 0:
			l.Medidos++
			horas[id] = append(horas[id], tempo.Hours())
			if s.ContratoPrazoHoras != nil && !fora {
				l.NoPrazo++
			}
		}
	}
	for i := range out {
		l := &out[i]
		if l.PrazoHoras != nil && l.Medidos > 0 {
			l.NoPrazoPercentual = percentual(l.NoPrazo, l.Medidos)
		}
		if h := horas[l.ContratoID]; len(h) > 0 {
			l.MedianaHoras = arredondar(mediana(h))
		}
	}
	slices.SortFunc(out, func(a, b LinhaContrato) int { return cmp.Compare(a.Rotulo, b.Rotulo) })
	return out
}
