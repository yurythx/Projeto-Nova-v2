// Package domain é o cadastro do cofre. O segredo em claro não mora aqui:
// a lista só carrega nome, usuário e unidade.
package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNaoEncontrado  = errors.New("cofre: não encontrado")
	ErrForaDoEscopo   = errors.New("cofre: fora do seu escopo")
	ErrSemPermissao   = errors.New("cofre: sem permissão para esta ação")
	ErrSenhaRecusada  = errors.New("cofre: senha de acesso recusada")
	ErrBloqueado      = errors.New("cofre: muitas tentativas de senha")
	ErrSemConfirmacao = errors.New("cofre: confirmação indisponível")
	ErrOcupado        = errors.New("cofre: segredo retirado")
)

const (
	// A retirada é curta de propósito: a pessoa usa e devolve.
	RetiradaMinMinutos = 15
	RetiradaMaxMinutos = 8 * 60
	// IntervaloRotacao é o prazo entre rotações de um segredo marcado.
	IntervaloRotacao = 30 * 24 * time.Hour
)

// EntradaInvalida explica o cadastro recusado.
type EntradaInvalida struct{ Msg string }

func (e EntradaInvalida) Error() string { return e.Msg }

const (
	TipoMaquina   = "maquina"
	TipoAplicacao = "aplicacao"
	TipoPainel    = "painel"
	TipoOutro     = "outro"
)

// Segredo é o que a lista pode mostrar. Sem senha e sem texto cifrado.
// Endereço e tipo ficam à vista: servem para achar a máquina, a aplicação
// ou o painel. Administra lista máquinas de um painel, só pelo id.
type Segredo struct {
	ID       uuid.UUID `json:"id"`
	Nome     string    `json:"nome"`
	Usuario  string    `json:"usuario"`
	Tipo     string    `json:"tipo"`
	Endereco string    `json:"endereco"`
	// Rotaciona pede senha nova ao devolver, ao vencer e a cada 30 dias.
	// RotacaoPendente avisa que essa senha ainda não foi aplicada no endereço.
	Rotaciona       bool        `json:"rotaciona"`
	RotacaoPendente bool        `json:"rotacao_pendente"`
	ProximaRotacao  *time.Time  `json:"proxima_rotacao,omitempty"`
	Administra      []uuid.UUID `json:"administra,omitempty"`
	PastaID         *uuid.UUID  `json:"pasta_id,omitempty"`
	// Retirada é quem está com o segredo agora, e até quando. Sem senha.
	Retirada  *Retirada  `json:"retirada,omitempty"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	Versao    int        `json:"versao"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Selado é a linha gravada. Dek e Cifra não saem no JSON.
type Selado struct {
	Segredo
	Dek   string `json:"-"`
	Cifra string `json:"-"`
}

// Retirada é a reserva em aberto. A lista mostra quem e até quando.
type Retirada struct {
	UsuarioID uuid.UUID `json:"usuario_id"`
	Nome      string    `json:"nome"`
	Ate       time.Time `json:"ate"`
}

// RetiradaGravada é a linha aberta, inclusive a que já venceu e ainda não foi encerrada.
type RetiradaGravada struct {
	ID        uuid.UUID
	SegredoID uuid.UUID
	UsuarioID uuid.UUID
	Nome      string
	Ate       time.Time
}

// MinutosDeRetirada aceita de 15 minutos a 8 horas.
func MinutosDeRetirada(minutos int) (int, error) {
	if minutos < RetiradaMinMinutos || minutos > RetiradaMaxMinutos {
		return 0, EntradaInvalida{"a retirada dura de 15 minutos a 8 horas"}
	}
	return minutos, nil
}

// Revelado é a resposta de uma revelação. A senha só existe neste tipo.
type Revelado struct {
	Segredo
	Senha string `json:"senha"`
	Notas string `json:"notas"`
}

// Filtro restringe a listagem às unidades cobertas. Restrito falso vê tudo.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
	// FiltrarPastas esconde o segredo de quem não está na pasta.
	// O worker deixa falso e enxerga a linha para rotacionar.
	FiltrarPastas bool
	Usuario       uuid.UUID
	GereTudo      bool
	GereUnidades  []uuid.UUID
}

// AAD amarra o texto cifrado a este segredo, a esta versão e a esta unidade.
// Trocar a linha de lugar não abre com a chave de outra.
func AAD(id uuid.UUID, versao int, unidade *uuid.UUID) []byte {
	dono := "-"
	if unidade != nil {
		dono = unidade.String()
	}
	return []byte(id.String() + ":" + strconv.Itoa(versao) + ":" + dono)
}

// Validar aceita o cadastro. A senha pode ir vazia quando a alteração não a troca.
func Validar(nome, usuario, senha, notas string, senhaObrigatoria bool) (string, string, string, string, error) {
	nome = strings.TrimSpace(nome)
	usuario = strings.TrimSpace(usuario)
	senha = strings.TrimSpace(senha)
	notas = strings.TrimSpace(notas)
	if nome == "" || utf8.RuneCountInString(nome) > 160 {
		return "", "", "", "", EntradaInvalida{"informe o nome, com até 160 caracteres"}
	}
	if utf8.RuneCountInString(usuario) > 160 {
		return "", "", "", "", EntradaInvalida{"o usuário deve ter até 160 caracteres"}
	}
	if senhaObrigatoria && senha == "" {
		return "", "", "", "", EntradaInvalida{"informe a senha"}
	}
	if utf8.RuneCountInString(senha) > 2000 || utf8.RuneCountInString(notas) > 4000 {
		return "", "", "", "", EntradaInvalida{"a senha deve ter até 2000 caracteres e as notas até 4000"}
	}
	return nome, usuario, senha, notas, nil
}

// TipoDe aceita máquina, aplicação, painel ou outro. Vazio vira outro.
func TipoDe(tipo string) (string, error) {
	tipo = strings.TrimSpace(tipo)
	switch tipo {
	case "", TipoOutro:
		return TipoOutro, nil
	case TipoMaquina, TipoAplicacao, TipoPainel:
		return tipo, nil
	default:
		return "", EntradaInvalida{"o tipo é máquina, aplicação, painel ou outro"}
	}
}

// EnderecoDe guarda IP, host ou URL. Não é a senha e pode ir vazio.
func EnderecoDe(endereco string) (string, error) {
	endereco = strings.TrimSpace(endereco)
	if utf8.RuneCountInString(endereco) > 500 {
		return "", EntradaInvalida{"o endereço deve ter até 500 caracteres"}
	}
	return endereco, nil
}
