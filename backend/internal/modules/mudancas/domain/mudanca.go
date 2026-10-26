// Package domain é a mudança: alterar o que está em operação com registro
// de quem pediu, quem aprovou e como desfazer.
package domain

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	TituloMax = 160
	TextoMax  = 4000

	TipoPadrao      = "padrao"
	TipoNormal      = "normal"
	TipoEmergencial = "emergencial"

	Rascunho    = "rascunho"
	EmAvaliacao = "em_avaliacao"
	Aprovada    = "aprovada"
	Rejeitada   = "rejeitada"
	EmExecucao  = "em_execucao"
	Concluida   = "concluida"
	Falhou      = "falhou"
	Cancelada   = "cancelada"

	// Papéis de quem age: quem gere a mudança e quem aprova.
	PapelGestor    = "gestor"
	PapelAprovador = "aprovador"

	DecisaoAprovada   = "aprovada"
	DecisaoRejeitada  = "rejeitada"
	DecisaoAutomatica = "automatica"

	VinculoProblema  = "problema"
	VinculoIncidente = "incidente"
	VinculoAtivo     = "ativo"
)

var (
	Tipos  = []string{TipoPadrao, TipoNormal, TipoEmergencial}
	Riscos = []string{"baixo", "medio", "alto"}
)

var (
	ErrNaoEncontrado       = errors.New("mudancas: mudança não encontrada")
	ErrEntradaInvalida     = errors.New("mudancas: dados inválidos")
	ErrTipoInvalido        = errors.New("mudancas: tipo ou risco desconhecido")
	ErrRiscoDivergente     = errors.New("mudancas: a nota não corresponde ao risco informado")
	ErrCongelada           = errors.New("mudancas: a janela cai num período de congelamento")
	ErrForaDoEscopo        = errors.New("mudancas: fora do seu escopo")
	ErrUnidadeInexistente  = errors.New("mudancas: unidade inexistente")
	ErrTransicaoInvalida   = errors.New("mudancas: passagem de estado inválida")
	ErrNaoEditavel         = errors.New("mudancas: só rascunho ou rejeitada se editam")
	ErrSemJanela           = errors.New("mudancas: informe a janela planejada (início antes do fim)")
	ErrSemReversao         = errors.New("mudancas: informe o plano de reversão")
	ErrSemParecer          = errors.New("mudancas: rejeitar exige parecer")
	ErrSemResultado        = errors.New("mudancas: registrar a falha exige o resultado (o que foi revertido)")
	ErrAutoAprovacao       = errors.New("mudancas: quem pediu a mudança não a aprova")
	ErrVinculoInvalido     = errors.New("mudancas: vínculo inválido")
	ErrVinculoInexistente  = errors.New("mudancas: item do vínculo não encontrado")
	ErrDeOutraUnidade      = errors.New("mudancas: o vínculo precisa ser da mesma unidade da mudança")
	ErrIntegracaoDesligada = errors.New("mudancas: o módulo deste vínculo está desligado")
	ErrMudancaEncerrada    = errors.New("mudancas: mudança encerrada")
	ErrCascataIndisponivel = errors.New("mudancas: a conclusão dos vínculos está indisponível")
)

// Alvo é o que a mudança precisa saber de um item de outro plug-in.
type Alvo struct {
	Rotulo    string
	UnidadeID *uuid.UUID
	// Tipo do incidente (só incidente se liga; requisição não).
	Tipo string
}

// Portas é como a mudança consulta os outros plug-ins (ligadas em
// internal/app). Nil = o vínculo daquele tipo fica indisponível.
type Portas interface {
	Consultar(ctx context.Context, db database.DBTX, tipo string, id uuid.UUID) (Alvo, error)
	Disponivel(tipo string) bool
}

// Resolucao é o que a conclusão da mudança leva aos vínculos.
type Resolucao struct {
	Autor  uuid.UUID
	Texto  string
	Numero int64
	Titulo string
}

// Entrega publica o que foi gravado na transação, depois do commit.
type Entrega func(ctx context.Context)

// Efeitos conclui incidentes e problemas na mesma transação da mudança.
type Efeitos interface {
	Resolver(ctx context.Context, db database.DBTX, tipo string, alvo uuid.UUID, r Resolucao) (Entrega, error)
}

// VinculoRecusado é a recusa de concluir um vínculo, com texto seguro
// para quem está na tela.
type VinculoRecusado struct {
	Mensagem string
}

func (e *VinculoRecusado) Error() string {
	if e == nil {
		return "mudancas: vínculo não concluído"
	}
	return e.Mensagem
}

// Produtos é a porta do plug-in de produtos.
type Produtos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (unidade *uuid.UUID, err error)
}

// Aprovacao é uma decisão sobre a mudança.
type Aprovacao struct {
	ID          uuid.UUID  `json:"id"`
	AprovadorID *uuid.UUID `json:"aprovador_id,omitempty"`
	Aprovador   string     `json:"aprovador"`
	Decisao     string     `json:"decisao"`
	Parecer     string     `json:"parecer"`
	Em          time.Time  `json:"em"`
}

// Vinculo liga a mudança a um problema, incidente ou ativo.
type Vinculo struct {
	Tipo        string    `json:"tipo"`
	AlvoID      uuid.UUID `json:"alvo_id"`
	Rotulo      string    `json:"rotulo"`
	VinculadoEm time.Time `json:"vinculado_em"`
}

// Mudanca é o pedido de alteração, numa unidade (nil = institucional).
type Mudanca struct {
	ID              uuid.UUID  `json:"id"`
	Numero          int64      `json:"numero"`
	Titulo          string     `json:"titulo"`
	Descricao       string     `json:"descricao"`
	Justificativa   string     `json:"justificativa"`
	Tipo            string     `json:"tipo"`
	Risco           string     `json:"risco"`
	RiscoScore      *int       `json:"risco_score,omitempty"`
	Estado          string     `json:"estado"`
	PlanoReversao   string     `json:"plano_reversao"`
	InicioPlanejado *time.Time `json:"inicio_planejado,omitempty"`
	FimPlanejado    *time.Time `json:"fim_planejado,omitempty"`
	Resultado       string     `json:"resultado"`
	ExecutadaInicio *time.Time `json:"executada_inicio,omitempty"`
	ExecutadaFim    *time.Time `json:"executada_fim,omitempty"`
	ProdutoID       *uuid.UUID `json:"produto_id,omitempty"`
	UnidadeID       *uuid.UUID `json:"unidade_id,omitempty"`
	SolicitanteID   uuid.UUID  `json:"solicitante_id"`
	SolicitanteNome string     `json:"solicitante_nome"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	Aprovacoes []Aprovacao `json:"aprovacoes"`
	Vinculos   []Vinculo   `json:"vinculos"`
	Proximos   []string    `json:"proximos"`
	// passagens são caminhos acrescentados neste quadro. Quem gere conduz.
	passagens []string
}

// Entrada são os campos editáveis.
type Entrada struct {
	Titulo, Descricao, Justificativa, Tipo, Risco, PlanoReversao string
	RiscoScore                                                   *int
	InicioPlanejado, FimPlanejado                                *time.Time
	ProdutoID, UnidadeID                                         *uuid.UUID
}

// Congelamento é uma janela em que não se agenda mudança. Unidade nil
// vale para toda a instituição.
type Congelamento struct {
	ID        uuid.UUID  `json:"id"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Inicio    time.Time  `json:"inicio"`
	Fim       time.Time  `json:"fim"`
	Motivo    string     `json:"motivo"`
	CreatedAt time.Time  `json:"created_at"`
}

// Dona é a posição da mudança para as permissões com escopo.
func Dona(unidade *uuid.UUID) auth.Target {
	if unidade == nil {
		return auth.Target{}
	}
	return auth.InUnidade(*unidade)
}

// MesmaUnidade compara donos, tratando o institucional (nil) como valor.
func MesmaUnidade(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func validar(in Entrada) (Entrada, error) {
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao, in.Justificativa, in.PlanoReversao = strings.TrimSpace(in.Descricao), strings.TrimSpace(in.Justificativa), strings.TrimSpace(in.PlanoReversao)
	if in.Titulo == "" || utf8.RuneCountInString(in.Titulo) > TituloMax {
		return Entrada{}, ErrEntradaInvalida
	}
	for _, t := range []string{in.Descricao, in.Justificativa, in.PlanoReversao} {
		if utf8.RuneCountInString(t) > TextoMax {
			return Entrada{}, ErrEntradaInvalida
		}
	}
	if !slices.Contains(Tipos, in.Tipo) || !slices.Contains(Riscos, in.Risco) {
		return Entrada{}, ErrTipoInvalido
	}
	if in.RiscoScore != nil {
		nota := *in.RiscoScore
		if nota < 5 || nota > 15 {
			return Entrada{}, ErrEntradaInvalida
		}
		faixa := "alto"
		if nota <= 7 {
			faixa = "baixo"
		} else if nota <= 11 {
			faixa = "medio"
		}
		if faixa != in.Risco {
			return Entrada{}, ErrRiscoDivergente
		}
	}
	return in, nil
}

// Nova abre a mudança como rascunho.
func Nova(in Entrada, solicitante uuid.UUID) (Mudanca, error) {
	in, err := validar(in)
	if err != nil {
		return Mudanca{}, err
	}
	if solicitante == uuid.Nil {
		return Mudanca{}, ErrEntradaInvalida
	}
	now := time.Now().UTC()
	m := Mudanca{ID: uuid.New(), Estado: Rascunho, SolicitanteID: solicitante, CreatedAt: now, UpdatedAt: now,
		Aprovacoes: []Aprovacao{}, Vinculos: []Vinculo{}}
	m.aplicar(in)
	return m, nil
}

func (m *Mudanca) aplicar(in Entrada) {
	m.Titulo, m.Descricao, m.Justificativa, m.Tipo, m.Risco, m.RiscoScore = in.Titulo, in.Descricao, in.Justificativa, in.Tipo, in.Risco, in.RiscoScore
	m.PlanoReversao, m.InicioPlanejado, m.FimPlanejado = in.PlanoReversao, in.InicioPlanejado, in.FimPlanejado
	m.ProdutoID, m.UnidadeID = in.ProdutoID, in.UnidadeID
}

// Aplicar edita a mudança: só no rascunho ou depois de rejeitada.
func (m Mudanca) Aplicar(in Entrada) (Mudanca, error) {
	if m.Estado != Rascunho && m.Estado != Rejeitada {
		return Mudanca{}, ErrNaoEditavel
	}
	in, err := validar(in)
	if err != nil {
		return Mudanca{}, err
	}
	m.aplicar(in)
	m.UpdatedAt = time.Now().UTC()
	return m, nil
}

// Igual ignora horários e listas: edição vazia não regrava.
func (m Mudanca) Igual(o Mudanca) bool {
	return m.Titulo == o.Titulo && m.Descricao == o.Descricao && m.Justificativa == o.Justificativa &&
		m.Tipo == o.Tipo && m.Risco == o.Risco && mesmaNota(m.RiscoScore, o.RiscoScore) && m.PlanoReversao == o.PlanoReversao &&
		iguais(m.InicioPlanejado, o.InicioPlanejado) && iguais(m.FimPlanejado, o.FimPlanejado) &&
		MesmaUnidade(m.ProdutoID, o.ProdutoID) && MesmaUnidade(m.UnidadeID, o.UnidadeID)
}

func mesmaNota(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func iguais(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// transicoes por papel: o gestor conduz; o aprovador decide.
var transicoes = map[string]map[string][]string{
	PapelGestor: {
		Rascunho:    {EmAvaliacao, Cancelada},
		Rejeitada:   {EmAvaliacao, Cancelada},
		EmAvaliacao: {Cancelada},
		Aprovada:    {EmExecucao, Cancelada},
		EmExecucao:  {Concluida, Falhou},
	},
	PapelAprovador: {
		EmAvaliacao: {Aprovada, Rejeitada},
	},
}

// Encerrada reporta se a mudança terminou.
func (m Mudanca) Encerrada() bool {
	return m.Estado == Concluida || m.Estado == Falhou || m.Estado == Cancelada
}

// PodeIrPara são as passagens dos papéis, sem repetir. Quem pediu não vê
// a aprovação da própria mudança.
func (m Mudanca) PodeIrPara(autor uuid.UUID, papeis ...string) []string {
	out := []string{}
	for _, p := range papeis {
		if p == PapelAprovador && autor == m.SolicitanteID {
			continue
		}
		for _, d := range transicoes[p][m.Estado] {
			if !slices.Contains(out, d) {
				out = append(out, d)
			}
		}
	}
	if slices.Contains(papeis, PapelGestor) {
		for _, d := range m.passagens {
			if d != "" && d != m.Estado && !slices.Contains(out, d) {
				out = append(out, d)
			}
		}
	}
	return out
}

// ComPassagens devolve a cópia com os caminhos acrescentados neste quadro.
func (m Mudanca) ComPassagens(extras []string) Mudanca {
	m.passagens = append([]string{}, extras...)
	return m
}

// Transicao é a passagem pedida, com o que cada uma exige.
type Transicao struct {
	Para  string
	Autor uuid.UUID
	Texto string // parecer (decisão) ou resultado (fim da execução)
}

// Transicionar aplica a passagem. Submeter a mudança padrão já a aprova
// (é pré-aprovada): devolve a decisão automática para registro.
func (m Mudanca) Transicionar(t Transicao, papeis ...string) (Mudanca, *Aprovacao, bool, error) {
	if t.Para == m.Estado {
		return m, nil, false, nil
	}
	if m.Encerrada() {
		return Mudanca{}, nil, false, ErrMudancaEncerrada
	}
	ehAprovacao := t.Para == Aprovada || t.Para == Rejeitada
	if ehAprovacao && slices.Contains(papeis, PapelAprovador) && t.Autor == m.SolicitanteID {
		return Mudanca{}, nil, false, ErrAutoAprovacao
	}
	if !slices.Contains(m.PodeIrPara(t.Autor, papeis...), t.Para) {
		return Mudanca{}, nil, false, ErrTransicaoInvalida
	}
	texto := strings.TrimSpace(t.Texto)
	if utf8.RuneCountInString(texto) > TextoMax {
		return Mudanca{}, nil, false, ErrEntradaInvalida
	}
	now := time.Now().UTC()
	var decisao *Aprovacao
	switch t.Para {
	case EmAvaliacao:
		if m.InicioPlanejado == nil || m.FimPlanejado == nil || !m.FimPlanejado.After(*m.InicioPlanejado) {
			return Mudanca{}, nil, false, ErrSemJanela
		}
		if m.Tipo != TipoPadrao && m.PlanoReversao == "" {
			return Mudanca{}, nil, false, ErrSemReversao
		}
		if m.Tipo == TipoPadrao {
			t.Para = Aprovada
			decisao = &Aprovacao{ID: uuid.New(), Decisao: DecisaoAutomatica, Parecer: "mudança padrão, pré-aprovada", Em: now}
		}
	case Aprovada, Rejeitada:
		if t.Para == Rejeitada && texto == "" {
			return Mudanca{}, nil, false, ErrSemParecer
		}
		autor := t.Autor
		decisao = &Aprovacao{ID: uuid.New(), AprovadorID: &autor, Decisao: t.Para, Parecer: texto, Em: now}
	case EmExecucao:
		m.ExecutadaInicio = &now
	case Concluida, Falhou:
		if t.Para == Falhou && texto == "" {
			return Mudanca{}, nil, false, ErrSemResultado
		}
		m.ExecutadaFim, m.Resultado = &now, texto
	}
	m.Estado, m.UpdatedAt = t.Para, now
	return m, decisao, true, nil
}

// NovoVinculo valida o tipo do vínculo.
func NovoVinculo(tipo string, alvo uuid.UUID) error {
	if !slices.Contains([]string{VinculoProblema, VinculoIncidente, VinculoAtivo}, tipo) || alvo == uuid.Nil {
		return ErrVinculoInvalido
	}
	return nil
}

// Filtro recorta a listagem.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
	Estado   string
	// Alvo lista as mudanças ligadas a um problema, incidente ou ativo.
	AlvoTipo string
	AlvoID   *uuid.UUID
}

// Repository é a porta de persistência.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Mudanca, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID) (Mudanca, error)
	Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error
	Insert(ctx context.Context, db database.DBTX, m Mudanca) (Mudanca, error)
	Update(ctx context.Context, db database.DBTX, m Mudanca) error
	Aprovacoes(ctx context.Context, db database.DBTX, mudanca uuid.UUID) ([]Aprovacao, error)
	InsertAprovacao(ctx context.Context, db database.DBTX, mudanca uuid.UUID, a Aprovacao) error
	Vinculos(ctx context.Context, db database.DBTX, mudanca uuid.UUID) ([]Vinculo, error)
	Vincular(ctx context.Context, db database.DBTX, mudanca uuid.UUID, v Vinculo, autor uuid.UUID) (bool, error)
	Desvincular(ctx context.Context, db database.DBTX, mudanca uuid.UUID, tipo string, alvo uuid.UUID) (bool, error)
	JanelaOcupada(ctx context.Context, db database.DBTX, unidade *uuid.UUID, inicio, fim time.Time) (bool, error)
	ListCongelamentos(ctx context.Context, db database.DBTX, f Filtro) ([]Congelamento, error)
	InsertCongelamento(ctx context.Context, db database.DBTX, c Congelamento) (Congelamento, error)
	DeleteCongelamento(ctx context.Context, db database.DBTX, id uuid.UUID) error
}
