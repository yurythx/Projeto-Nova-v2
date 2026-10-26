package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	// A base de fusos vai no binário: a imagem slim pode não trazer
	// /usr/share/zoneinfo, e o prazo depende do fuso da unidade.
	_ "time/tzdata"

	"github.com/google/uuid"
)

// Prazo em tempo útil (fase 4). A política da unidade diz quando se
// trabalha (fuso e turnos), como impacto e urgência viram prioridade e
// quanto tempo útil cada prioridade tem para resposta e solução. Os
// feriados e pontos facultativos não contam.

const (
	PrioridadeCritica = "critica"
	PrioridadeAlta    = "alta"
	PrioridadeMedia   = "media"
	PrioridadeBaixa   = "baixa"

	// Níveis de impacto e urgência: 1 baixo, 2 médio, 3 alto.
	NivelMin    = 1
	NivelMax    = 3
	NivelPadrao = 2

	// MetaMaxMin limita a meta a um ano de minutos úteis.
	MetaMaxMin = 525600
	// TurnosPorDia limita os turnos de um dia da semana.
	TurnosPorDia = 4

	FeriadoNacional         = "nacional"
	FeriadoEstadual         = "estadual"
	FeriadoMunicipal        = "municipal"
	FeriadoPontoFacultativo = "ponto_facultativo"

	// Estado do relógio.
	RelogioCorrendo = "correndo"
	RelogioPausado  = "pausado"
	RelogioParado   = "parado"

	// Situação do prazo: as quatro primeiras com o relógio vivo, as duas
	// últimas com ele parado.
	SituacaoNoPrazo     = "no_prazo"
	SituacaoAtencao     = "atencao"
	SituacaoCritico     = "critico"
	SituacaoVencido     = "vencido"
	SituacaoCumprido    = "cumprido"
	SituacaoDescumprido = "descumprido"

	// Limiares de alerta, em porcentagem da meta.
	LimiarAtencao = 75
	LimiarCritico = 90
	LimiarVencido = 100

	// AntecedenciaAlerta avisa quando o relógio correndo vence em até
	// duas horas de relógio de parede. A meta precisa ser maior que essa
	// janela: um prazo que já nasce com menos de duas horas não dispara
	// o aviso na abertura.
	AntecedenciaAlerta = 2 * time.Hour

	// horizonte é até onde a previsão de vencimento procura tempo útil.
	horizonteDias = 400
)

// Prioridades, da mais urgente para a menos.
var Prioridades = []string{PrioridadeCritica, PrioridadeAlta, PrioridadeMedia, PrioridadeBaixa}

var (
	ErrPoliticaInvalida = errors.New("atendimento: política de prazo inválida")
	ErrFeriadoInvalido  = errors.New("atendimento: feriado inválido")
	ErrFeriadoRepetido  = errors.New("atendimento: já há feriado nesta data para a unidade")
	ErrNivelInvalido    = errors.New("atendimento: impacto e urgência vão de 1 a 3")
)

// Turno é um intervalo de expediente num dia da semana (0 = domingo).
type Turno struct {
	Dia    int    `json:"dia"`
	Inicio string `json:"inicio"`
	Fim    string `json:"fim"`
}

// Celula é uma casa da matriz impacto x urgência.
type Celula struct {
	Impacto    int    `json:"impacto"`
	Urgencia   int    `json:"urgencia"`
	Prioridade string `json:"prioridade"`
}

// Meta é o tempo útil, em minutos, de cada prioridade.
type Meta struct {
	Prioridade  string `json:"prioridade"`
	RespostaMin int    `json:"resposta_min"`
	SolucaoMin  int    `json:"solucao_min"`
}

// Politica é a regra de prazo de uma unidade. Herdada indica que veio de
// uma unidade acima (ou da institucional) porque a unidade não tem a sua.
type Politica struct {
	ID        uuid.UUID  `json:"id"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Fuso      string     `json:"fuso"`
	Turnos    []Turno    `json:"turnos"`
	Matriz    []Celula   `json:"matriz"`
	Metas     []Meta     `json:"metas"`
	Herdada   bool       `json:"herdada"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Feriado é um dia sem expediente para a unidade e as abaixo dela.
type Feriado struct {
	ID        uuid.UUID  `json:"id"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Data      string     `json:"data"`
	Nome      string     `json:"nome"`
	Tipo      string     `json:"tipo"`
	CreatedAt time.Time  `json:"created_at"`
}

func invalida(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrPoliticaInvalida}, args...)...)
}

// minutos converte "HH:MM" em minutos do dia (24:00 fecha o dia).
func minutos(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", hhmm)
	if err == nil {
		return t.Hour()*60 + t.Minute(), true
	}
	if hhmm == "24:00" {
		return 24 * 60, true
	}
	return 0, false
}

var diasDaSemana = [7]string{"domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"}

// NovaPolitica valida e normaliza a política: fuso conhecido, turnos sem
// sobreposição, matriz completa e uma meta para cada prioridade usada.
func NovaPolitica(unidade *uuid.UUID, fuso string, turnos []Turno, matriz []Celula, metas []Meta) (Politica, error) {
	fuso = strings.TrimSpace(fuso)
	if fuso == "" {
		return Politica{}, invalida("informe o fuso")
	}
	if _, err := time.LoadLocation(fuso); err != nil || fuso == "Local" {
		return Politica{}, invalida("fuso %q desconhecido", fuso)
	}
	ts, err := normalizarTurnos(turnos)
	if err != nil {
		return Politica{}, err
	}
	ms, usadas, err := normalizarMatriz(matriz)
	if err != nil {
		return Politica{}, err
	}
	mt, err := normalizarMetas(metas, usadas)
	if err != nil {
		return Politica{}, err
	}
	now := time.Now().UTC()
	return Politica{ID: uuid.New(), UnidadeID: unidade, Fuso: fuso, Turnos: ts, Matriz: ms, Metas: mt, CreatedAt: now, UpdatedAt: now}, nil
}

func normalizarTurnos(turnos []Turno) ([]Turno, error) {
	if len(turnos) == 0 {
		return nil, invalida("informe ao menos um turno de expediente")
	}
	out := make([]Turno, 0, len(turnos))
	for _, t := range turnos {
		if t.Dia < 0 || t.Dia > 6 {
			return nil, invalida("dia da semana %d inválido", t.Dia)
		}
		ini, ok1 := minutos(t.Inicio)
		fim, ok2 := minutos(t.Fim)
		if !ok1 || !ok2 || ini >= fim {
			return nil, invalida("turno de %s: use HH:MM e início antes do fim", diasDaSemana[t.Dia])
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b Turno) int {
		if a.Dia != b.Dia {
			return a.Dia - b.Dia
		}
		x, _ := minutos(a.Inicio)
		y, _ := minutos(b.Inicio)
		return x - y
	})
	porDia := map[int]int{}
	for i, t := range out {
		porDia[t.Dia]++
		if porDia[t.Dia] > TurnosPorDia {
			return nil, invalida("%s tem mais de %d turnos", diasDaSemana[t.Dia], TurnosPorDia)
		}
		if i > 0 && out[i-1].Dia == t.Dia {
			fimAnterior, _ := minutos(out[i-1].Fim)
			ini, _ := minutos(t.Inicio)
			if ini < fimAnterior {
				return nil, invalida("turnos de %s se sobrepõem", diasDaSemana[t.Dia])
			}
		}
	}
	return out, nil
}

func normalizarMatriz(matriz []Celula) ([]Celula, []string, error) {
	vistas := map[[2]int]bool{}
	usadas := []string{}
	for _, c := range matriz {
		if c.Impacto < NivelMin || c.Impacto > NivelMax || c.Urgencia < NivelMin || c.Urgencia > NivelMax {
			return nil, nil, invalida("impacto e urgência da matriz vão de 1 a 3")
		}
		if !slices.Contains(Prioridades, c.Prioridade) {
			return nil, nil, invalida("prioridade %q desconhecida", c.Prioridade)
		}
		k := [2]int{c.Impacto, c.Urgencia}
		if vistas[k] {
			return nil, nil, invalida("a matriz repete impacto %d e urgência %d", c.Impacto, c.Urgencia)
		}
		vistas[k] = true
		if !slices.Contains(usadas, c.Prioridade) {
			usadas = append(usadas, c.Prioridade)
		}
	}
	if len(vistas) != (NivelMax-NivelMin+1)*(NivelMax-NivelMin+1) {
		return nil, nil, invalida("a matriz precisa das 9 combinações de impacto e urgência")
	}
	out := slices.Clone(matriz)
	slices.SortFunc(out, func(a, b Celula) int {
		if a.Impacto != b.Impacto {
			return b.Impacto - a.Impacto
		}
		return b.Urgencia - a.Urgencia
	})
	return out, usadas, nil
}

func normalizarMetas(metas []Meta, usadas []string) ([]Meta, error) {
	por := map[string]Meta{}
	for _, m := range metas {
		if !slices.Contains(Prioridades, m.Prioridade) {
			return nil, invalida("prioridade %q desconhecida", m.Prioridade)
		}
		if _, ok := por[m.Prioridade]; ok {
			return nil, invalida("meta repetida para %s", m.Prioridade)
		}
		if m.RespostaMin <= 0 || m.SolucaoMin <= 0 || m.RespostaMin > MetaMaxMin || m.SolucaoMin > MetaMaxMin {
			return nil, invalida("as metas de %s precisam ser positivas e de até um ano", m.Prioridade)
		}
		if m.RespostaMin > m.SolucaoMin {
			return nil, invalida("em %s, a resposta não pode ter prazo maior que a solução", m.Prioridade)
		}
		por[m.Prioridade] = m
	}
	out := []Meta{}
	for _, p := range Prioridades {
		m, ok := por[p]
		if !ok {
			if slices.Contains(usadas, p) {
				return nil, invalida("falta a meta da prioridade %s, usada na matriz", p)
			}
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// Niveis devolve o primeiro par da matriz que produz a prioridade.
func (p Politica) Niveis(prioridade string) (int, int, bool) {
	for _, c := range p.Matriz {
		if c.Prioridade == prioridade {
			return c.Impacto, c.Urgencia, true
		}
	}
	return 0, 0, false
}

// Classificar diz a prioridade e a meta de impacto e urgência.
func (p Politica) Classificar(impacto, urgencia int) (string, Meta) {
	for _, c := range p.Matriz {
		if c.Impacto == impacto && c.Urgencia == urgencia {
			for _, m := range p.Metas {
				if m.Prioridade == c.Prioridade {
					return c.Prioridade, m
				}
			}
		}
	}
	return "", Meta{}
}

// NovoFeriado valida o feriado (data AAAA-MM-DD).
func NovoFeriado(unidade *uuid.UUID, data, nome, tipo string) (Feriado, error) {
	data, nome = strings.TrimSpace(data), strings.TrimSpace(nome)
	if _, err := time.Parse(time.DateOnly, data); err != nil {
		return Feriado{}, ErrFeriadoInvalido
	}
	if nome == "" || utf8.RuneCountInString(nome) > NomeMax {
		return Feriado{}, ErrFeriadoInvalido
	}
	if !slices.Contains([]string{FeriadoNacional, FeriadoEstadual, FeriadoMunicipal, FeriadoPontoFacultativo}, tipo) {
		return Feriado{}, ErrFeriadoInvalido
	}
	return Feriado{ID: uuid.New(), UnidadeID: unidade, Data: data, Nome: nome, Tipo: tipo, CreatedAt: time.Now().UTC()}, nil
}

// Calendario sabe quanto tempo útil há entre dois instantes.
type Calendario struct {
	loc      *time.Location
	turnos   [7][][2]int // minutos do dia, por dia da semana
	feriados map[string]bool
}

// NovoCalendario monta o calendário de uma política já validada.
func NovoCalendario(p Politica, feriados []Feriado) (Calendario, error) {
	loc, err := time.LoadLocation(p.Fuso)
	if err != nil {
		return Calendario{}, invalida("fuso %q desconhecido", p.Fuso)
	}
	c := Calendario{loc: loc, feriados: map[string]bool{}}
	for _, t := range p.Turnos {
		ini, _ := minutos(t.Inicio)
		fim, _ := minutos(t.Fim)
		c.turnos[t.Dia] = append(c.turnos[t.Dia], [2]int{ini, fim})
	}
	for _, f := range feriados {
		c.feriados[f.Data] = true
	}
	return c, nil
}

// expediente devolve os intervalos de trabalho do dia (no fuso) de t.
func (c Calendario) expediente(dia time.Time) [][2]time.Time {
	if c.feriados[dia.Format(time.DateOnly)] {
		return nil
	}
	y, m, d := dia.Date()
	out := [][2]time.Time{}
	for _, t := range c.turnos[dia.Weekday()] {
		// time.Date acerta a hora mesmo em dia de mudança de horário.
		ini := time.Date(y, m, d, 0, t[0], 0, 0, c.loc)
		fim := time.Date(y, m, d, 0, t[1], 0, 0, c.loc)
		out = append(out, [2]time.Time{ini, fim})
	}
	return out
}

func inicioDoDia(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// Util é o tempo de expediente entre a e b.
func (c Calendario) Util(a, b time.Time) time.Duration {
	if !b.After(a) {
		return 0
	}
	var total time.Duration
	for dia := inicioDoDia(a.In(c.loc)); dia.Before(b); dia = dia.AddDate(0, 0, 1) {
		for _, t := range c.expediente(dia) {
			ini, fim := maxTime(a, t[0]), minTime(b, t[1])
			if fim.After(ini) {
				total += fim.Sub(ini)
			}
		}
	}
	return total
}

// Somar devolve o instante em que d de tempo útil terá passado desde a.
// Falso quando não há expediente no horizonte (política sem turnos ou um
// ano inteiro de feriados).
func (c Calendario) Somar(a time.Time, d time.Duration) (time.Time, bool) {
	dia := inicioDoDia(a.In(c.loc))
	for i := 0; i < horizonteDias; i, dia = i+1, dia.AddDate(0, 0, 1) {
		for _, t := range c.expediente(dia) {
			ini := maxTime(a, t[0])
			if !t[1].After(ini) {
				continue
			}
			if livre := t[1].Sub(ini); d <= livre {
				return ini.Add(d), true
			} else {
				d -= livre
			}
		}
	}
	return time.Time{}, false
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Relogio é uma das duas contagens (resposta ou solução).
type Relogio struct {
	MetaMin    int        `json:"meta_min"`
	UsadoMin   int        `json:"usado_min"`
	Percentual int        `json:"percentual"`
	Estado     string     `json:"estado"`
	Situacao   string     `json:"situacao"`
	VenceEm    *time.Time `json:"vence_em,omitempty"`
}

// Prazo é o compromisso da solicitação medido agora.
type Prazo struct {
	Prioridade string  `json:"prioridade"`
	Resposta   Relogio `json:"resposta"`
	Solucao    Relogio `json:"solucao"`
}

// MedirPrazo conta o tempo útil de cada relógio a partir das passagens.
// A resposta corre enquanto a solicitação está aberta e ninguém começou;
// a solução corre aberta ou em atendimento — as esperas a pausam e a
// conclusão a para (reabrir volta a correr). Sem prioridade, não há prazo.
func MedirPrazo(s Solicitacao, passagens []Passagem, cal Calendario, agora time.Time) *Prazo {
	if s.Prioridade == nil || s.MetaRespostaMin == nil || s.MetaSolucaoMin == nil {
		return nil
	}
	var resposta, solucao time.Duration
	respondida := false
	for i, p := range passagens {
		fim := agora
		if i+1 < len(passagens) {
			fim = passagens[i+1].Em
		}
		u := cal.Util(p.Em, fim)
		if p.Para == EstadoEmAtendimento {
			respondida = true
		}
		if p.Para == EstadoAberto && !respondida {
			resposta += u
		}
		if p.Para == EstadoAberto || p.Para == EstadoEmAtendimento {
			solucao += u
		}
	}
	estadoResposta := RelogioParado
	if !respondida && s.Estado == EstadoAberto {
		estadoResposta = RelogioCorrendo
	}
	estadoSolucao := RelogioParado
	switch s.Estado {
	case EstadoAberto, EstadoEmAtendimento:
		estadoSolucao = RelogioCorrendo
	case EstadoAguardandoServidor, EstadoAguardandoTerceiro:
		estadoSolucao = RelogioPausado
	}
	return &Prazo{
		Prioridade: *s.Prioridade,
		Resposta:   relogio(*s.MetaRespostaMin, resposta, estadoResposta, cal, agora),
		Solucao:    relogio(*s.MetaSolucaoMin, solucao, estadoSolucao, cal, agora),
	}
}

func relogio(metaMin int, usado time.Duration, estado string, cal Calendario, agora time.Time) Relogio {
	meta := time.Duration(metaMin) * time.Minute
	r := Relogio{MetaMin: metaMin, UsadoMin: int(usado / time.Minute), Estado: estado}
	r.Percentual = int(usado * 100 / meta)
	switch {
	case estado == RelogioParado && usado <= meta:
		r.Situacao = SituacaoCumprido
	case estado == RelogioParado:
		r.Situacao = SituacaoDescumprido
	case r.Percentual >= LimiarVencido:
		r.Situacao = SituacaoVencido
	case r.Percentual >= LimiarCritico:
		r.Situacao = SituacaoCritico
	case r.Percentual >= LimiarAtencao:
		r.Situacao = SituacaoAtencao
	default:
		r.Situacao = SituacaoNoPrazo
	}
	if estado == RelogioCorrendo && usado < meta {
		if vence, ok := cal.Somar(agora, meta-usado); ok {
			r.VenceEm = &vence
		}
	}
	return r
}

// DentroDaAntecedencia diz se o relógio correndo vence dentro de janela
// e a meta é maior que ela. Relógio parado, vencido ou sem previsão não
// entra: o vencimento já tem o próprio aviso.
func (r Relogio) DentroDaAntecedencia(agora time.Time, janela time.Duration) bool {
	if r.Estado != RelogioCorrendo || r.VenceEm == nil || janela <= 0 {
		return false
	}
	if r.MetaMin <= int(janela/time.Minute) {
		return false
	}
	falta := r.VenceEm.Sub(agora)
	return falta > 0 && falta <= janela
}

// Limiar é o maior limiar de alerta que o relógio correndo já cruzou
// (0 = nenhum).
func (r Relogio) Limiar() int {
	if r.Estado != RelogioCorrendo {
		return 0
	}
	for _, l := range []int{LimiarVencido, LimiarCritico, LimiarAtencao} {
		if r.Percentual >= l {
			return l
		}
	}
	return 0
}

// Classificar aplica a política ao impacto e à urgência e guarda a
// prioridade e as metas do momento. Sem política (ou sem a combinação),
// a solicitação fica sem prazo.
func (s Solicitacao) Classificar(impacto, urgencia int, p *Politica) (Solicitacao, bool, error) {
	if s.Finalizada() {
		return Solicitacao{}, false, ErrSolicitacaoFinalizada
	}
	if impacto < NivelMin || impacto > NivelMax || urgencia < NivelMin || urgencia > NivelMax {
		return Solicitacao{}, false, ErrNivelInvalido
	}
	next := s
	next.Impacto, next.Urgencia = impacto, urgencia
	next = next.AplicarPolitica(p)
	if next.Impacto == s.Impacto && next.Urgencia == s.Urgencia && iguais(next.Prioridade, s.Prioridade) &&
		iguais(next.MetaRespostaMin, s.MetaRespostaMin) && iguais(next.MetaSolucaoMin, s.MetaSolucaoMin) {
		return s, false, nil
	}
	next.UpdatedAt = time.Now().UTC()
	return next, true, nil
}

// AplicarPolitica põe a prioridade e as metas que a política dá ao
// impacto e à urgência atuais (sem política, nenhuma).
func (s Solicitacao) AplicarPolitica(p *Politica) Solicitacao {
	s.Prioridade, s.MetaRespostaMin, s.MetaSolucaoMin = nil, nil, nil
	if p == nil {
		return s
	}
	if prioridade, meta := p.Classificar(s.Impacto, s.Urgencia); prioridade != "" {
		s.Prioridade, s.MetaRespostaMin, s.MetaSolucaoMin = &prioridade, &meta.RespostaMin, &meta.SolucaoMin
	}
	return s
}

func iguais[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
