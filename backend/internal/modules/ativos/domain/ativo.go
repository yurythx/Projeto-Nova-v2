// Package domain é o ativo (item de configuração): o que sustenta um
// produto digital e passa por planejamento, uso, manutenção e desativação.
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
	NomeMax          = 160
	TextoMax         = 4000
	IdentificadorMax = 80
)

// Tipos e estados do ativo.
var (
	Tipos   = []string{"hardware", "software", "servico", "rede", "outro"}
	Estados = []string{"planejado", "em_uso", "manutencao", "desativado"}
)

var (
	ErrNaoEncontrado         = errors.New("ativos: ativo não encontrado")
	ErrEntradaInvalida       = errors.New("ativos: dados inválidos")
	ErrTipoInvalido          = errors.New("ativos: tipo desconhecido")
	ErrEstadoInvalido        = errors.New("ativos: estado desconhecido")
	ErrForaDoEscopo          = errors.New("ativos: fora do seu escopo de gestão")
	ErrUnidadeInexistente    = errors.New("ativos: unidade inexistente")
	ErrIdentificadorRepetido = errors.New("ativos: identificador já usado por outro ativo")
	ErrProdutoInexistente    = errors.New("ativos: produto não encontrado")
	ErrProdutoIndisponivel   = errors.New("ativos: cadastro de produtos desligado")
	ErrProdutoDeOutraUnidade = errors.New("ativos: o produto precisa ser da mesma unidade do ativo")
)

// Produtos é a porta do plugin de produtos, ligada em internal/app.
type Produtos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (unidadeID *uuid.UUID, err error)
}

// Ativo é um item de configuração posicionado numa unidade.
type Ativo struct {
	ID            uuid.UUID  `json:"id"`
	Nome          string     `json:"nome"`
	Tipo          string     `json:"tipo"`
	Estado        string     `json:"estado"`
	Identificador string     `json:"identificador"`
	Descricao     string     `json:"descricao"`
	ProdutoID     *uuid.UUID `json:"produto_id,omitempty"`
	UnidadeID     *uuid.UUID `json:"unidade_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Entrada são os campos de um ativo.
type Entrada struct {
	Nome, Tipo, Estado, Identificador, Descricao string
	ProdutoID, UnidadeID                         *uuid.UUID
}

// Dona é a posição do ativo para ativos:manage.
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
	in.Nome, in.Identificador, in.Descricao = strings.TrimSpace(in.Nome), strings.TrimSpace(in.Identificador), strings.TrimSpace(in.Descricao)
	if in.Nome == "" || utf8.RuneCountInString(in.Nome) > NomeMax ||
		utf8.RuneCountInString(in.Identificador) > IdentificadorMax || utf8.RuneCountInString(in.Descricao) > TextoMax {
		return Entrada{}, ErrEntradaInvalida
	}
	if !slices.Contains(Tipos, in.Tipo) {
		return Entrada{}, ErrTipoInvalido
	}
	if !slices.Contains(Estados, in.Estado) {
		return Entrada{}, ErrEstadoInvalido
	}
	return in, nil
}

// Novo valida e monta um ativo.
func Novo(in Entrada) (Ativo, error) {
	in, err := validar(in)
	if err != nil {
		return Ativo{}, err
	}
	now := time.Now().UTC()
	return Ativo{
		ID: uuid.New(), Nome: in.Nome, Tipo: in.Tipo, Estado: in.Estado, Identificador: in.Identificador,
		Descricao: in.Descricao, ProdutoID: in.ProdutoID, UnidadeID: in.UnidadeID, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Aplicar devolve a cópia editada.
func (a Ativo) Aplicar(in Entrada) (Ativo, error) {
	in, err := validar(in)
	if err != nil {
		return Ativo{}, err
	}
	a.Nome, a.Tipo, a.Estado, a.Identificador, a.Descricao = in.Nome, in.Tipo, in.Estado, in.Identificador, in.Descricao
	a.ProdutoID, a.UnidadeID = in.ProdutoID, in.UnidadeID
	a.UpdatedAt = time.Now().UTC()
	return a, nil
}

// Igual ignora os horários: uma edição vazia não regrava.
func (a Ativo) Igual(o Ativo) bool {
	return a.ID == o.ID && a.Nome == o.Nome && a.Tipo == o.Tipo && a.Estado == o.Estado &&
		a.Identificador == o.Identificador && a.Descricao == o.Descricao &&
		MesmaUnidade(a.ProdutoID, o.ProdutoID) && MesmaUnidade(a.UnidadeID, o.UnidadeID)
}

// Filtro restringe a listagem ao escopo e, opcionalmente, a um produto.
type Filtro struct {
	Restrito  bool
	Unidades  []uuid.UUID
	ProdutoID *uuid.UUID
}

// Repository é a porta de persistência.
type Repository interface {
	List(ctx context.Context, db database.DBTX, f Filtro) ([]Ativo, error)
	Get(ctx context.Context, db database.DBTX, id uuid.UUID) (Ativo, error)
	Insert(ctx context.Context, db database.DBTX, a Ativo) (Ativo, error)
	Update(ctx context.Context, db database.DBTX, a Ativo) (Ativo, error)
}
