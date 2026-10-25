package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// PrefixoCredencial marca o bearer da máquina. Não é um JWT de pessoa.
const PrefixoCredencial = "nova_agt_"

// Credencial autoriza uma máquina a informar o próprio patrimônio.
// O segredo não fica guardado: só o hash.
type Credencial struct {
	ID         uuid.UUID  `json:"id"`
	Nome       string     `json:"nome"`
	UnidadeID  uuid.UUID  `json:"unidade_id"`
	TokenHash  []byte     `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	RevogadaEm *time.Time `json:"revogada_em,omitempty"`
}

// EmitirCredencial cria a credencial e devolve o segredo uma única vez.
func EmitirCredencial(nome string, unidade uuid.UUID) (Credencial, string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > NomeMax || unidade == uuid.Nil {
		return Credencial{}, "", ErrEntradaInvalida
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return Credencial{}, "", err
	}
	token := PrefixoCredencial + base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now().UTC()
	return Credencial{
		ID: uuid.New(), Nome: nome, UnidadeID: unidade, TokenHash: HashCredencial(token), CreatedAt: now,
	}, token, nil
}

// HashCredencial é o que se compara no banco.
func HashCredencial(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
