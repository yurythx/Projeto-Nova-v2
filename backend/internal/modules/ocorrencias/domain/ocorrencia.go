// Package domain modela a ocorrência (incidente grave comunicado ao
// público, ADR 066): o que está fora, desde quando e o que se sabe.
package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Impacto, do menor para o maior.
const (
	ImpactoDegradado = "degradado" // funciona, devagar ou com falhas
	ImpactoParcial   = "parcial"   // parte do serviço fora
	ImpactoFora      = "fora"      // fora do ar
)

// Estados da ocorrência, na ordem em que costumam acontecer. Só
// "resolvido" encerra.
const (
	EstadoInvestigando = "investigando"
	EstadoIdentificado = "identificado"
	EstadoMonitorando  = "monitorando"
	EstadoResolvido    = "resolvido"
)

// SituacaoOperacional é a situação geral sem ocorrência aberta.
const SituacaoOperacional = "operacional"

// JanelaRecentes é por quanto tempo a resolvida continua na página.
const JanelaRecentes = 7 * 24 * time.Hour

const (
	maxTitulo   = 160
	maxTexto    = 4000
	maxServicos = 10
	maxServico  = 80
)

var (
	ErrNaoEncontrada = errors.New("ocorrencias: ocorrência não encontrada")
	// ErrEntradaInvalida: título, serviço, impacto, estado ou texto fora da regra.
	ErrEntradaInvalida = errors.New("ocorrencias: dados inválidos")
	// ErrJaResolvida: a resolvida não recebe atualização nem troca de coordenador.
	ErrJaResolvida = errors.New("ocorrencias: a ocorrência já foi resolvida")
	// ErrAindaAberta: a revisão vem depois de resolver.
	ErrAindaAberta = errors.New("ocorrencias: a revisão vem depois de resolver")
	// ErrCoordenadorInvalido: a pessoa não existe.
	ErrCoordenadorInvalido = errors.New("ocorrencias: coordenador inexistente")
)

var (
	impactos = []string{ImpactoDegradado, ImpactoParcial, ImpactoFora}
	estados  = []string{EstadoInvestigando, EstadoIdentificado, EstadoMonitorando, EstadoResolvido}
)

// Ocorrencia é o incidente grave como a coordenação o acompanha. Título e
// serviços são escritos para o público: não são copiados do chamado.
type Ocorrencia struct {
	ID              uuid.UUID     `json:"id"`
	Titulo          string        `json:"titulo"`
	Impacto         string        `json:"impacto"`
	Estado          string        `json:"estado"`
	Servicos        []string      `json:"servicos"`
	ChamadoID       *uuid.UUID    `json:"chamado_id,omitempty"`
	CoordenadorID   *uuid.UUID    `json:"coordenador_id,omitempty"`
	CoordenadorNome string        `json:"coordenador_nome,omitempty"`
	IniciadaEm      time.Time     `json:"iniciada_em"`
	ResolvidaEm     *time.Time    `json:"resolvida_em,omitempty"`
	Revisao         string        `json:"revisao,omitempty"`
	RevisadaEm      *time.Time    `json:"revisada_em,omitempty"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Atualizacoes    []Atualizacao `json:"atualizacoes"`
}

// Atualizacao é uma entrada da linha do tempo: o estado e o texto público.
type Atualizacao struct {
	ID           uuid.UUID  `json:"id"`
	OcorrenciaID uuid.UUID  `json:"ocorrencia_id"`
	Estado       string     `json:"estado"`
	Impacto      string     `json:"impacto"`
	Texto        string     `json:"texto"`
	AutorID      *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome    string     `json:"autor_nome,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Resolvida diz se a ocorrência terminou.
func (o Ocorrencia) Resolvida() bool { return o.Estado == EstadoResolvido }

// Declarar abre a ocorrência em "investigando", com a primeira mensagem ao
// público. Quem declara coordena.
func Declarar(titulo string, servicos []string, impacto, texto string, chamado *uuid.UUID, coordenador uuid.UUID, agora time.Time) (Ocorrencia, Atualizacao, error) {
	titulo = strings.TrimSpace(titulo)
	if titulo == "" || utf8.RuneCountInString(titulo) > maxTitulo {
		return Ocorrencia{}, Atualizacao{}, ErrEntradaInvalida
	}
	lista, err := limparServicos(servicos)
	if err != nil {
		return Ocorrencia{}, Atualizacao{}, err
	}
	o := Ocorrencia{
		ID: uuid.New(), Titulo: titulo, Estado: EstadoInvestigando, Servicos: lista, ChamadoID: chamado,
		IniciadaEm: agora, UpdatedAt: agora,
	}
	if coordenador != uuid.Nil {
		o.CoordenadorID = &coordenador
	}
	o, a, err := o.Atualizar(EstadoInvestigando, impacto, texto, coordenador, agora)
	if err != nil {
		return Ocorrencia{}, Atualizacao{}, err
	}
	return o, a, nil
}

// Atualizar registra o que mudou, com o texto que o público lê. Impacto
// pode subir ou descer; resolver registra o fim.
func (o Ocorrencia) Atualizar(estado, impacto, texto string, autor uuid.UUID, agora time.Time) (Ocorrencia, Atualizacao, error) {
	if o.Resolvida() {
		return Ocorrencia{}, Atualizacao{}, ErrJaResolvida
	}
	texto = strings.TrimSpace(texto)
	if !slices.Contains(estados, estado) || !slices.Contains(impactos, impacto) || texto == "" || utf8.RuneCountInString(texto) > maxTexto {
		return Ocorrencia{}, Atualizacao{}, ErrEntradaInvalida
	}
	o.Estado, o.Impacto, o.UpdatedAt = estado, impacto, agora
	if estado == EstadoResolvido {
		fim := agora
		o.ResolvidaEm = &fim
	}
	a := Atualizacao{ID: uuid.New(), OcorrenciaID: o.ID, Estado: estado, Impacto: impacto, Texto: texto, CreatedAt: agora}
	if autor != uuid.Nil {
		a.AutorID = &autor
	}
	return o, a, nil
}

// TrocarCoordenador passa a coordenação enquanto a ocorrência está aberta.
func (o Ocorrencia) TrocarCoordenador(quem uuid.UUID, agora time.Time) (Ocorrencia, error) {
	if o.Resolvida() {
		return Ocorrencia{}, ErrJaResolvida
	}
	if quem == uuid.Nil {
		return Ocorrencia{}, ErrEntradaInvalida
	}
	o.CoordenadorID, o.UpdatedAt = &quem, agora
	return o, nil
}

// Revisar grava a revisão pós-incidente. Só depois de resolver; pode ser
// reescrita enquanto a coordenação aprende mais.
func (o Ocorrencia) Revisar(texto string, agora time.Time) (Ocorrencia, error) {
	if !o.Resolvida() {
		return Ocorrencia{}, ErrAindaAberta
	}
	texto = strings.TrimSpace(texto)
	if texto == "" || utf8.RuneCountInString(texto) > maxTexto {
		return Ocorrencia{}, ErrEntradaInvalida
	}
	o.Revisao, o.RevisadaEm, o.UpdatedAt = texto, &agora, agora
	return o, nil
}

func limparServicos(servicos []string) ([]string, error) {
	out := []string{}
	for _, s := range servicos {
		s = strings.TrimSpace(s)
		if s == "" || slices.Contains(out, s) {
			continue
		}
		if utf8.RuneCountInString(s) > maxServico {
			return nil, ErrEntradaInvalida
		}
		out = append(out, s)
	}
	if len(out) == 0 || len(out) > maxServicos {
		return nil, ErrEntradaInvalida
	}
	return out, nil
}

// Publica é o que a página de status mostra: sem quem atuou, sem chamado,
// sem revisão.
type Publica struct {
	ID           uuid.UUID            `json:"id"`
	Titulo       string               `json:"titulo"`
	Impacto      string               `json:"impacto"`
	Estado       string               `json:"estado"`
	Servicos     []string             `json:"servicos"`
	IniciadaEm   time.Time            `json:"iniciada_em"`
	ResolvidaEm  *time.Time           `json:"resolvida_em,omitempty"`
	Atualizacoes []AtualizacaoPublica `json:"atualizacoes"`
}

// AtualizacaoPublica é a entrada da linha do tempo sem o autor.
type AtualizacaoPublica struct {
	Estado string    `json:"estado"`
	Texto  string    `json:"texto"`
	Em     time.Time `json:"em"`
}

// Situacao é a página de status inteira.
type Situacao struct {
	Geral    string    `json:"geral"`
	Abertas  []Publica `json:"abertas"`
	Recentes []Publica `json:"recentes"`
}

// Publicar tira da ocorrência o que não é público. A linha do tempo vem
// da mais nova para a mais antiga.
func (o Ocorrencia) Publicar() Publica {
	p := Publica{ID: o.ID, Titulo: o.Titulo, Impacto: o.Impacto, Estado: o.Estado, Servicos: o.Servicos,
		IniciadaEm: o.IniciadaEm, ResolvidaEm: o.ResolvidaEm, Atualizacoes: []AtualizacaoPublica{}}
	for i := len(o.Atualizacoes) - 1; i >= 0; i-- {
		a := o.Atualizacoes[i]
		p.Atualizacoes = append(p.Atualizacoes, AtualizacaoPublica{Estado: a.Estado, Texto: a.Texto, Em: a.CreatedAt})
	}
	return p
}

// MontarSituacao separa abertas e recentes e dá a situação geral: o maior
// impacto em aberto, ou operacional.
func MontarSituacao(ocorrencias []Ocorrencia) Situacao {
	s := Situacao{Geral: SituacaoOperacional, Abertas: []Publica{}, Recentes: []Publica{}}
	pior := -1
	for _, o := range ocorrencias {
		if o.Resolvida() {
			s.Recentes = append(s.Recentes, o.Publicar())
			continue
		}
		s.Abertas = append(s.Abertas, o.Publicar())
		if i := slices.Index(impactos, o.Impacto); i > pior {
			pior = i
			s.Geral = o.Impacto
		}
	}
	return s
}
