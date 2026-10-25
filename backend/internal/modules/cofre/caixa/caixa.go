// Package caixa cifra cada segredo com a própria chave, e embrulha essa
// chave com a chave mestra do processo. A chave mestra não é a que cifra
// a configuração do Keycloak.
package caixa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const keySize = 32

var (
	ErrChave  = errors.New("cofre: chave mestra inválida")
	ErrAberto = errors.New("cofre: não foi possível abrir o segredo")
)

// Conteudo é o que fica dentro do envelope. Não vai para a lista.
type Conteudo struct {
	Senha string `json:"senha"`
	Notas string `json:"notas"`
}

// Selo é o que se grava: a chave do segredo embrulhada e o texto cifrado.
type Selo struct {
	Dek   string
	Cifra string
}

// Caixa usa a chave mestra só para embrulhar a chave de cada segredo.
type Caixa struct {
	master cipher.AEAD
}

// Nova exige a chave em base64, com 32 bytes depois de decodificar.
func Nova(chaveBase64 string) (*Caixa, error) {
	raw, err := base64.StdEncoding.DecodeString(chaveBase64)
	if err != nil || len(raw) != keySize {
		return nil, ErrChave
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, ErrChave
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrChave
	}
	return &Caixa{master: aead}, nil
}

// Selar gera uma chave nova para este segredo e devolve as duas partes cifradas.
func (c *Caixa) Selar(conteudo Conteudo, aad []byte) (Selo, error) {
	dek := make([]byte, keySize)
	if _, err := rand.Read(dek); err != nil {
		return Selo{}, fmt.Errorf("cofre: gerar chave: %w", err)
	}
	plain, err := json.Marshal(conteudo)
	if err != nil {
		return Selo{}, err
	}
	bloco, err := aes.NewCipher(dek)
	if err != nil {
		return Selo{}, err
	}
	interno, err := cipher.NewGCM(bloco)
	if err != nil {
		return Selo{}, err
	}
	cifra, err := embrulhar(interno, plain, aad)
	if err != nil {
		return Selo{}, err
	}
	dekCifra, err := embrulhar(c.master, dek, aad)
	if err != nil {
		return Selo{}, err
	}
	return Selo{Dek: dekCifra, Cifra: cifra}, nil
}

// Abrir recupera a senha. AAD diferente ou chave errada falha fechado.
func (c *Caixa) Abrir(selo Selo, aad []byte) (Conteudo, error) {
	dek, err := abrir(c.master, selo.Dek, aad)
	if err != nil || len(dek) != keySize {
		return Conteudo{}, ErrAberto
	}
	bloco, err := aes.NewCipher(dek)
	if err != nil {
		return Conteudo{}, ErrAberto
	}
	interno, err := cipher.NewGCM(bloco)
	if err != nil {
		return Conteudo{}, ErrAberto
	}
	plain, err := abrir(interno, selo.Cifra, aad)
	if err != nil {
		return Conteudo{}, ErrAberto
	}
	var conteudo Conteudo
	if err := json.Unmarshal(plain, &conteudo); err != nil {
		return Conteudo{}, ErrAberto
	}
	return conteudo, nil
}

func embrulhar(aead cipher.AEAD, plain, aad []byte) (string, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := aead.Seal(nonce, nonce, plain, aad)
	return base64.StdEncoding.EncodeToString(out), nil
}

func abrir(aead cipher.AEAD, cifra string, aad []byte) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(cifra)
	if err != nil || len(raw) < aead.NonceSize() {
		return nil, ErrAberto
	}
	nonce, corpo := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, corpo, aad)
	if err != nil {
		return nil, ErrAberto
	}
	return plain, nil
}
