// Package domain é o problema: a causa por trás de incidentes que se
// repetem. Com a solução de contorno ele vira erro conhecido; com a causa
// raiz, resolvido.
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

	Registrado    = "registrado"
	EmAnalise     = "em_analise"
	ErroConhecido = "erro_conhecido"
	Resolvido     = "resolvido"
	Encerrado     = "encerrado"

	// Recorrência: a partir de quantos incidentes da mesma oferta, em
	// quantos dias, a equipe recebe a sugestão de abrir um problema.
	RecorrenciaMinima = 3
	RecorrenciaDias   = 30
)

var (
	ErrNaoEncontrado        = errors.New("problemas: problema não encontrado")
	ErrEntradaInvalida      = errors.New("problemas: dados inválidos")
	ErrForaDoEscopo         = errors.New("problemas: fora do seu escopo de gestão")
	ErrUnidadeInexistente   = errors.New("problemas: unidade inexistente")
	ErrTransicaoInvalida    = errors.New("problemas: passagem de estado inválida")
	ErrSemContorno          = errors.New("problemas: erro conhecido precisa da solução de contorno")
	ErrSemCausaRaiz         = errors.New("problemas: resolver exige a causa raiz")
	ErrEncerrado            = errors.New("problemas: problema encerrado")
	ErrIncidenteInexistente = errors.New("problemas: incidente não encontrado")
	ErrNaoIncidente         = errors.New("problemas: só incidentes se ligam a um problema")
	ErrAtivoInexistente     = errors.New("problemas: ativo não encontrado")
	ErrDeOutraUnidade       = errors.New("problemas: o vínculo precisa ser da mesma unidade do problema")
	ErrIntegracaoDesligada  = errors.New("problemas: o módulo do vínculo está desligado")
	ErrProdutoInexistente   = errors.New("problemas: produto não encontrado")
)

// NaoConcluivel explica por que a mudança não pode resolver este problema.
type NaoConcluivel struct {
	Numero int64
	Estado string
	Causa  error
}

func (e *NaoConcluivel) Error() string {
	if e == nil || e.Causa == nil {
		return "problemas: problema não concluível"
	}
	return e.Causa.Error()
}

func (e *NaoConcluivel) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Causa
}

// transicoes: para onde cada estado pode ir. Encerrado não sai.
var transicoes = map[string][]string{
	Registrado:    {EmAnalise},
	EmAnalise:     {ErroConhecido, Resolvido},
	ErroConhecido: {Resolvido, EmAnalise},
	Resolvido:     {Encerrado, EmAnalise},
}

// Portas para os outros plug-ins, ligadas em internal/app.

// Incidente é o que o problema precisa saber da solicitação.
type Incidente struct {
	ID        uuid.UUID
	Numero    int64
	Titulo    string
	Tipo      string
	UnidadeID *uuid.UUID
}

// Recorrencia são incidentes da mesma oferta num período.
type Recorrencia struct {
	OfertaID     uuid.UUID   `json:"oferta_id"`
	OfertaNome   string      `json:"oferta_nome"`
	UnidadeID    *uuid.UUID  `json:"unidade_id,omitempty"`
	Total        int         `json:"total"`
	UltimoEm     time.Time   `json:"ultimo_em"`
	Solicitacoes []uuid.UUID `json:"solicitacoes"`
}

// Incidentes é a porta do atendimento.
type Incidentes interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (Incidente, error)
	// Recorrentes agrupa por oferta os incidentes desde a data, nas
	// unidades dadas (restrito falso: todas), com pelo menos minimo.
	Recorrentes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, minimo int) ([]Recorrencia, error)
}

// Ativos é a porta do plug-in de ativos.
type Ativos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (nome string, unidade *uuid.UUID, err error)
}

// Produtos é a porta do plug-in de produtos.
type Produtos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (unidade *uuid.UUID, err error)
}

// IncidenteVinculado é o incidente ligado ao problema.
type IncidenteVinculado struct {
	SolicitacaoID uuid.UUID `json:"solicitacao_id"`
	Numero        int64     `json:"numero"`
	Titulo        string    `json:"titulo"`
	VinculadoEm   time.Time `json:"vinculado_em"`
}

// AtivoVinculado é o ativo afetado pelo problema.
type AtivoVinculado struct {
	AtivoID     uuid.UUID `json:"ativo_id"`
	Nome        string    `json:"nome"`
	VinculadoEm time.Time `json:"vinculado_em"`
}

// Problema é a causa investigada, numa unidade (nil = institucional).
type Problema struct {
	ID          uuid.UUID  `json:"id"`
	Numero      int64      `json:"numero"`
	Titulo      string     `json:"titulo"`
	Descricao   string     `json:"descricao"`
	Estado      string     `json:"estado"`
	CausaRaiz   string     `json:"causa_raiz"`
	Contorno    string     `json:"contorno"`
	ProdutoID   *uuid.UUID `json:"produto_id,omitempty"`
	UnidadeID   *uuid.UUID `json:"unidade_id,omitempty"`
	CriadoPor   *uuid.UUID `json:"criado_por,omitempty"`
	ResolvidoEm *time.Time `json:"resolvido_em,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Incidentes []IncidenteVinculado `json:"incidentes"`
	Ativos     []AtivoVinculado     `json:"ativos"`
	Proximos   []string             `json:"proximos"`
}

// Entrada são os campos editáveis.
type Entrada struct {
	Titulo, Descricao, CausaRaiz, Contorno string
	ProdutoID, UnidadeID                   *uuid.UUID
}

// Dona é a posição do problema para as permissões com escopo.
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
	in.Descricao, in.CausaRaiz, in.Contorno = strings.TrimSpace(in.Descricao), strings.TrimSpace(in.CausaRaiz), strings.TrimSpace(in.Contorno)
	if in.Titulo == "" || utf8.RuneCountInString(in.Titulo) > TituloMax {
		return Entrada{}, ErrEntradaInvalida
	}
	for _, t := range []string{in.Descricao, in.CausaRaiz, in.Contorno} {
		if utf8.RuneCountInString(t) > TextoMax {
			return Entrada{}, ErrEntradaInvalida
		}
	}
	return in, nil
}

// Novo registra o problema.
func Novo(in Entrada, autor uuid.UUID) (Problema, error) {
	in, err := validar(in)
	if err != nil {
		return Problema{}, err
	}
	now := time.Now().UTC()
	p := Problema{
		ID: uuid.New(), Titulo: in.Titulo, Descricao: in.Descricao, Estado: Registrado, CausaRaiz: in.CausaRaiz,
		Contorno: in.Contorno, ProdutoID: in.ProdutoID, UnidadeID: in.UnidadeID, CreatedAt: now, UpdatedAt: now,
		Incidentes: []IncidenteVinculado{}, Ativos: []AtivoVinculado{},
	}
	if autor != uuid.Nil {
		p.CriadoPor = &autor
	}
	return p, nil
}

// Aplicar edita os campos. O que o estado exige continua exigido: não se
// apaga o contorno de um erro conhecido nem a causa de um resolvido.
func (p Problema) Aplicar(in Entrada) (Problema, error) {
	if p.Estado == Encerrado {
		return Problema{}, ErrEncerrado
	}
	in, err := validar(in)
	if err != nil {
		return Problema{}, err
	}
	next := p
	next.Titulo, next.Descricao, next.CausaRaiz, next.Contorno = in.Titulo, in.Descricao, in.CausaRaiz, in.Contorno
	next.ProdutoID, next.UnidadeID = in.ProdutoID, in.UnidadeID
	if err := next.exigencias(next.Estado); err != nil {
		return Problema{}, err
	}
	next.UpdatedAt = time.Now().UTC()
	return next, nil
}

func (p Problema) exigencias(estado string) error {
	switch {
	case estado == ErroConhecido && p.Contorno == "":
		return ErrSemContorno
	case (estado == Resolvido || estado == Encerrado) && p.CausaRaiz == "":
		return ErrSemCausaRaiz
	}
	return nil
}

// Transicionar muda o estado. O mesmo estado não é mudança.
func (p Problema) Transicionar(para string) (Problema, bool, error) {
	if para == p.Estado {
		return p, false, nil
	}
	if p.Estado == Encerrado {
		return Problema{}, false, ErrEncerrado
	}
	if !slices.Contains(transicoes[p.Estado], para) {
		return Problema{}, false, ErrTransicaoInvalida
	}
	if err := p.exigencias(para); err != nil {
		return Problema{}, false, err
	}
	now := time.Now().UTC()
	switch para {
	case Resolvido:
		p.ResolvidoEm = &now
	case EmAnalise:
		p.ResolvidoEm = nil
	}
	p.Estado, p.UpdatedAt = para, now
	return p, true, nil
}

// PodeIrPara são as passagens possíveis agora.
func (p Problema) PodeIrPara() []string {
	return append([]string{}, transicoes[p.Estado]...)
}

// Igual ignora horários e vínculos: edição vazia não regrava.
func (p Problema) Igual(o Problema) bool {
	return p.Titulo == o.Titulo && p.Descricao == o.Descricao && p.CausaRaiz == o.CausaRaiz && p.Contorno == o.Contorno &&
		MesmaUnidade(p.ProdutoID, o.ProdutoID) && MesmaUnidade(p.UnidadeID, o.UnidadeID)
}

// Filtro recorta a listagem.
type Filtro struct {
	Restrito      bool
	Unidades      []uuid.UUID
	Estado        string
	SolicitacaoID *uuid.UUID
	Busca         string
}

// Repository é a porta de persistência.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Problema, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID) (Problema, error)
	Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error
	Insert(ctx context.Context, db database.DBTX, p Problema) (Problema, error)
	Update(ctx context.Context, db database.DBTX, p Problema) error
	Incidentes(ctx context.Context, db database.DBTX, problema uuid.UUID) ([]IncidenteVinculado, error)
	VincularIncidente(ctx context.Context, db database.DBTX, problema uuid.UUID, i IncidenteVinculado, autor uuid.UUID) (bool, error)
	DesvincularIncidente(ctx context.Context, db database.DBTX, problema, solicitacao uuid.UUID) (bool, error)
	AtivosDe(ctx context.Context, db database.DBTX, problema uuid.UUID) ([]AtivoVinculado, error)
	VincularAtivo(ctx context.Context, db database.DBTX, problema, ativo, autor uuid.UUID) (bool, error)
	DesvincularAtivo(ctx context.Context, db database.DBTX, problema, ativo uuid.UUID) (bool, error)
	// JaVinculados diz quais das solicitações já estão em algum problema.
	JaVinculados(ctx context.Context, db database.DBTX, solicitacoes []uuid.UUID) (map[uuid.UUID]bool, error)
}
