package application

import (
	"crypto/rand"
	"math/big"
)

// alfabetoRotacao evita caracteres que se confundem ao copiar.
const alfabetoRotacao = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

const tamanhoRotacao = 20

func gerarSenha() (string, error) {
	buf := make([]byte, tamanhoRotacao)
	maximo := big.NewInt(int64(len(alfabetoRotacao)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, maximo)
		if err != nil {
			return "", err
		}
		buf[i] = alfabetoRotacao[n.Int64()]
	}
	return string(buf), nil
}
