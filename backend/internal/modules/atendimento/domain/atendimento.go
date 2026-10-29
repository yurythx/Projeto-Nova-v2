// Package domain é o atendimento: a oferta que o servidor escolhe, a
// equipe que atende uma unidade e a solicitação que percorre a fila.
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
	NomeMax    = 160
	TextoMax   = 4000
	MaxMembros = 100
)

var (
	ErrNaoEncontrado         = errors.New("atendimento: registro não encontrado")
	ErrEntradaInvalida       = errors.New("atendimento: dados inválidos")
	ErrCofreDesligado        = errors.New("atendimento: cofre desligado")
	ErrSegredoAusente        = errors.New("atendimento: segredo ausente")
	ErrForaDoEscopo          = errors.New("atendimento: fora do seu escopo de gestão")
	ErrProdutoInexistente    = errors.New("atendimento: produto não encontrado")
	ErrProdutoIndisponivel   = errors.New("atendimento: cadastro de produtos desligado")
	ErrProdutoDeOutraUnidade = errors.New("atendimento: o produto precisa ser da mesma unidade da oferta")
	ErrMembroInvalido        = errors.New("atendimento: membro inexistente ou inativo")
	ErrUnidadeInexistente    = errors.New("atendimento: unidade inexistente")
	ErrTipoInvalido          = errors.New("atendimento: tipo de oferta inválido")
	ErrSolicitacaoInvalida   = errors.New("atendimento: solicitação inválida")
	ErrSolicitanteInvalido   = errors.New("atendimento: solicitante inexistente")
	ErrTransicaoInvalida     = errors.New("atendimento: passagem de estado inválida")
	ErrSemResponsavel        = errors.New("atendimento: a solicitação precisa de um responsável")
	ErrResponsavelInvalido   = errors.New("atendimento: o responsável precisa ser de uma equipe da unidade")
	ErrSolicitacaoFinalizada = errors.New("atendimento: solicitação encerrada ou cancelada")
	// ErrEspelho: o chamado veio do GLPI e a sincronização está ligada; ele
	// muda lá até a virada (ADR 081).
	ErrEspelho             = errors.New("atendimento: chamado atendido no GLPI até a virada")
	ErrComentarioInvalido  = errors.New("atendimento: comentário inválido")
	ErrNotaInternaNegada   = errors.New("atendimento: só a equipe registra nota interna")
	ErrOfertaInexistente   = errors.New("atendimento: oferta não encontrada")
	ErrOfertaInativa       = errors.New("atendimento: oferta inativa")
	ErrAvaliacaoInvalida   = errors.New("atendimento: avaliação inválida")
	ErrAvaliacaoForaDeHora = errors.New("atendimento: só se avalia o atendimento concluído")
	ErrAvaliacaoAlheia       = errors.New("atendimento: só quem abriu avalia o atendimento")
	ErrAvaliacaoExistente    = errors.New("atendimento: esta solicitação já foi avaliada")
	ErrAtivosDesligado       = errors.New("atendimento: cadastro de ativos desligado")
	ErrAtivoAusente          = errors.New("atendimento: ativo não encontrado")
	ErrAtivoDeOutraUnidade   = errors.New("atendimento: o ativo precisa ser da mesma unidade da solicitação")
)

// Produtos é a porta do plugin de produtos. Não há import entre plugins:
// a composição liga esta porta em internal/app.
type Produtos interface {
	Consultar(ctx context.Context, db database.DBTX, id uuid.UUID) (unidadeID *uuid.UUID, err error)
}

// Oferta é um item do catálogo interno de atendimento, distinto da carta
// de serviços do site. O tipo decide a máquina de estado da solicitação
// aberta por ela: o servidor escolhe "não consigo acessar o e-mail", não
// "incidente".
type Oferta struct {
	ID        uuid.UUID  `json:"id"`
	Nome      string     `json:"nome"`
	Descricao string     `json:"descricao"`
	Tipo      string     `json:"tipo"`
	ProdutoID *uuid.UUID `json:"produto_id,omitempty"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Ativa     bool       `json:"ativa"`
	Campos    []Campo    `json:"campos"`
	// Categoria, ícone e ordem do bloco na central (ADR 078). Sem
	// categoria, a oferta aparece em "Outros".
	CategoriaID *uuid.UUID `json:"categoria_id,omitempty"`
	Icone       string     `json:"icone"`
	Ordem       int        `json:"ordem"`
	// SecretariaID, SecretariaNome e SecretariaSigla identificam a
	// entidade/secretaria provedora no catálogo quando há mais de uma
	// central de atendimento (ex.: Prefeitura/SECITI, SMS, SEMED).
	SecretariaID    *uuid.UUID `json:"secretaria_id,omitempty"`
	SecretariaNome  string     `json:"secretaria_nome,omitempty"`
	SecretariaSigla string     `json:"secretaria_sigla,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Equipe atende as solicitações de uma unidade.
type Equipe struct {
	ID        uuid.UUID   `json:"id"`
	Nome      string      `json:"nome"`
	UnidadeID *uuid.UUID  `json:"unidade_id,omitempty"`
	Membros   []uuid.UUID `json:"membros"`
	// OLAHoras é o prazo, em horas corridas, que a equipe assume para as
	// tarefas que recebe (ADR 070). Nulo = sem meta.
	OLAHoras  *int      `json:"ola_horas,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidarOLA confere o prazo da equipe: de 1 hora a um ano.
func ValidarOLA(h *int) error {
	if h != nil && (*h < 1 || *h > 8760) {
		return ErrOLAInvalido
	}
	return nil
}

// Candidato é um usuário ativo lotado na unidade, para compor a equipe.
type Candidato struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}

// Dona é a posição do recurso para atendimento:manage.
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

// TipoValido reporta se o tipo é uma das práticas atendidas nesta fase.
func TipoValido(tipo string) bool {
	return tipo == TipoIncidente || tipo == TipoRequisicao
}

// NovaOferta valida e monta uma oferta.
func NovaOferta(nome, descricao, tipo string, produto, unidade *uuid.UUID, ativa bool) (Oferta, error) {
	nome, descricao, err := campos(nome, descricao)
	if err != nil {
		return Oferta{}, err
	}
	if !TipoValido(tipo) {
		return Oferta{}, ErrTipoInvalido
	}
	now := time.Now().UTC()
	return Oferta{
		ID: uuid.New(), Nome: nome, Descricao: descricao, Tipo: tipo, ProdutoID: produto, UnidadeID: unidade, Ativa: ativa,
		Campos: []Campo{}, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Aplicar devolve a cópia editada. Mudar o tipo não altera as
// solicitações já abertas: cada uma guarda o tipo com que nasceu.
func (o Oferta) Aplicar(nome, descricao, tipo string, produto, unidade *uuid.UUID, ativa bool) (Oferta, error) {
	nome, descricao, err := campos(nome, descricao)
	if err != nil {
		return Oferta{}, err
	}
	if !TipoValido(tipo) {
		return Oferta{}, ErrTipoInvalido
	}
	o.Nome, o.Descricao, o.Tipo, o.ProdutoID, o.UnidadeID, o.Ativa = nome, descricao, tipo, produto, unidade, ativa
	o.UpdatedAt = time.Now().UTC()
	return o, nil
}

// Igual ignora os horários.
func (o Oferta) Igual(other Oferta) bool {
	return o.ID == other.ID && o.Nome == other.Nome && o.Descricao == other.Descricao && o.Tipo == other.Tipo && o.Ativa == other.Ativa &&
		MesmaUnidade(o.ProdutoID, other.ProdutoID) && MesmaUnidade(o.UnidadeID, other.UnidadeID) && camposIguais(o.Campos, other.Campos) &&
		MesmaUnidade(o.CategoriaID, other.CategoriaID) && o.Icone == other.Icone && o.Ordem == other.Ordem
}

func camposIguais(a, b []Campo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Chave != b[i].Chave || a[i].Rotulo != b[i].Rotulo || a[i].Tipo != b[i].Tipo || a[i].Obrigatorio != b[i].Obrigatorio || !slices.Equal(a[i].Opcoes, b[i].Opcoes) {
			return false
		}
	}
	return true
}

// NovaEquipe valida e monta uma equipe. Membros repetidos contam uma vez.
func NovaEquipe(nome string, unidade *uuid.UUID, membros []uuid.UUID) (Equipe, error) {
	nome, _, err := campos(nome, "")
	if err != nil {
		return Equipe{}, err
	}
	membros, err = normalizarMembros(membros)
	if err != nil {
		return Equipe{}, err
	}
	now := time.Now().UTC()
	return Equipe{ID: uuid.New(), Nome: nome, UnidadeID: unidade, Membros: membros, CreatedAt: now, UpdatedAt: now}, nil
}

// Aplicar devolve a cópia editada.
func (e Equipe) Aplicar(nome string, unidade *uuid.UUID, membros []uuid.UUID) (Equipe, error) {
	nome, _, err := campos(nome, "")
	if err != nil {
		return Equipe{}, err
	}
	membros, err = normalizarMembros(membros)
	if err != nil {
		return Equipe{}, err
	}
	e.Nome, e.UnidadeID, e.Membros = nome, unidade, membros
	e.UpdatedAt = time.Now().UTC()
	return e, nil
}

// Igual compara a equipe, inclusive o conjunto de membros, sem depender
// da ordem em que o banco devolve o array.
func (e Equipe) Igual(other Equipe) bool {
	return e.ID == other.ID && e.Nome == other.Nome && MesmaUnidade(e.UnidadeID, other.UnidadeID) && mesmosMembros(e.Membros, other.Membros) &&
		mesmaHora(e.OLAHoras, other.OLAHoras)
}

func mesmaHora(a, b *int) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func mesmosMembros(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	visto := map[uuid.UUID]int{}
	for _, id := range a {
		visto[id]++
	}
	for _, id := range b {
		visto[id]--
		if visto[id] < 0 {
			return false
		}
	}
	return true
}

func campos(nome, descricao string) (string, string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > NomeMax {
		return "", "", ErrEntradaInvalida
	}
	descricao = strings.TrimSpace(descricao)
	if utf8.RuneCountInString(descricao) > TextoMax {
		return "", "", ErrEntradaInvalida
	}
	return nome, descricao, nil
}

func normalizarMembros(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) > MaxMembros {
		return nil, ErrEntradaInvalida
	}
	visto := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, ErrMembroInvalido
		}
		if _, ok := visto[id]; ok {
			continue
		}
		visto[id] = struct{}{}
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return out, nil
}
