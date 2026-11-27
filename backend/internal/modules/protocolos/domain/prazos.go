package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Calendario são os feriados municipais, pela data AAAA-MM-DD em Cuiabá.
type Calendario map[string]bool

// Feriado diz se o dia é feriado. O calendário nulo não tem feriado.
func (c Calendario) Feriado(t time.Time) bool {
	return c[t.In(Cuiaba).Format(time.DateOnly)]
}

// util é o dia que não é sábado, domingo nem feriado.
func (c Calendario) util(t time.Time) bool {
	dia := t.In(Cuiaba).Weekday()
	return dia != time.Saturday && dia != time.Sunday && !c.Feriado(t)
}

// unir junta os calendários recebidos no parâmetro opcional.
func unir(cal []Calendario) Calendario {
	if len(cal) == 1 {
		return cal[0]
	}
	out := Calendario{}
	for _, c := range cal {
		for dia, ok := range c {
			out[dia] = ok
		}
	}
	return out
}

// Feriado municipal do protocolo.
type Feriado struct {
	Data string `json:"data"`
	Nome string `json:"nome"`
}

// NovoFeriado confere a data (AAAA-MM-DD) e o nome.
func NovoFeriado(data, nome string) (Feriado, error) {
	data, nome = strings.TrimSpace(data), strings.TrimSpace(nome)
	if _, err := time.ParseInLocation(time.DateOnly, data, Cuiaba); err != nil {
		return Feriado{}, ErrInvalido
	}
	if n := utf8.RuneCountInString(nome); n < 2 || n > 120 {
		return Feriado{}, ErrInvalido
	}
	return Feriado{Data: data, Nome: nome}, nil
}

// CalendarioDe monta o calendário a partir dos feriados cadastrados.
func CalendarioDe(feriados []Feriado) Calendario {
	c := Calendario{}
	for _, f := range feriados {
		c[f.Data] = true
	}
	return c
}

// DetalhesServico é a orientação extra que a carta mostra ao cidadão.
type DetalhesServico struct {
	QuemPode      string `json:"quem_pode"`
	PassoAPasso   string `json:"passo_a_passo"`
	Contato       string `json:"contato"`
	TempoEstimado string `json:"tempo_estimado"`
	DescricaoTaxa string `json:"descricao_taxa"`
}

// normalizar corta os espaços e confere os tamanhos.
func (d DetalhesServico) normalizar() (DetalhesServico, error) {
	d.QuemPode = strings.TrimSpace(d.QuemPode)
	d.PassoAPasso = strings.TrimSpace(d.PassoAPasso)
	d.Contato = strings.TrimSpace(d.Contato)
	d.TempoEstimado = strings.TrimSpace(d.TempoEstimado)
	d.DescricaoTaxa = strings.TrimSpace(d.DescricaoTaxa)
	for _, texto := range []string{d.QuemPode, d.PassoAPasso} {
		if utf8.RuneCountInString(texto) > 4000 {
			return DetalhesServico{}, ErrInvalido
		}
	}
	for _, texto := range []string{d.Contato, d.DescricaoTaxa} {
		if utf8.RuneCountInString(texto) > 500 {
			return DetalhesServico{}, ErrInvalido
		}
	}
	if utf8.RuneCountInString(d.TempoEstimado) > 120 {
		return DetalhesServico{}, ErrInvalido
	}
	return d, nil
}

// PrazoMaximoComplementacao limita os dias úteis dados ao cidadão.
const PrazoMaximoComplementacao = 90

var numeroSEI = regexp.MustCompile(`^[0-9][0-9./-]{3,38}[0-9]$`)

// VincularSEI grava o número do processo no SEI. Vazio desfaz o vínculo.
func (p Pedido) VincularSEI(numero string, agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	numero = strings.TrimSpace(numero)
	if numero != "" && !numeroSEI.MatchString(numero) {
		return Pedido{}, ErrInvalido
	}
	p.ProcessoSEI = numero
	p.UpdatedAt = agora
	return p, nil
}

// PrazoDaComplementacao dá ao cidadão os dias úteis para responder. Zero deixa sem prazo.
func (p Pedido) PrazoDaComplementacao(dias int, agora time.Time, cal ...Calendario) (Pedido, error) {
	if p.Status != StatusComplementacao || dias < 0 || dias > PrazoMaximoComplementacao {
		return Pedido{}, ErrInvalido
	}
	p.ComplementoPrazo = nil
	p.ComplementoAvisado = false
	if dias > 0 {
		dia := diaCivil(SomarDiasUteis(agora, dias, cal...))
		limite := time.Date(dia.Year(), dia.Month(), dia.Day(), 23, 59, 0, 0, Cuiaba)
		p.ComplementoPrazo = &limite
	}
	return p, nil
}

// ExpirarComplementacao cancela o pedido cuja complementação passou do prazo sem resposta.
func (p Pedido) ExpirarComplementacao(agora time.Time) (Pedido, error) {
	if p.Status != StatusComplementacao || p.ComplementoPrazo == nil || !agora.After(*p.ComplementoPrazo) {
		return Pedido{}, ErrInvalido
	}
	p.Status = StatusCancelado
	p.UpdatedAt = agora
	return p, nil
}
