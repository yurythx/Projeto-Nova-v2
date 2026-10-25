package caixa

import (
	"bytes"
	"testing"
)

func TestEnvelopeNaoAbreComOutraUnidadeNemOutraChave(t *testing.T) {
	cifra, err := Nova("MTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTE=")
	if err != nil {
		t.Fatal(err)
	}
	aad := []byte("segredo:1:unidade-a")
	selo, err := cifra.Selar(Conteudo{Senha: "segredo", Notas: "pista"}, aad)
	if err != nil {
		t.Fatal(err)
	}
	aberto, err := cifra.Abrir(selo, aad)
	if err != nil || aberto.Senha != "segredo" || aberto.Notas != "pista" {
		t.Fatalf("abrir: %+v %v", aberto, err)
	}
	if _, err := cifra.Abrir(selo, []byte("segredo:1:unidade-b")); err == nil {
		t.Fatal("abriu com outro contexto")
	}
	outra, err := Nova("MjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjI=")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := outra.Abrir(selo, aad); err == nil {
		t.Fatal("abriu com outra chave mestra")
	}
	if _, err := Nova("curta"); err == nil {
		t.Fatal("aceitou chave curta")
	}
	if bytes.Contains([]byte(selo.Cifra), []byte("segredo")) {
		t.Fatal("o texto cifrado carregou a senha")
	}
}
