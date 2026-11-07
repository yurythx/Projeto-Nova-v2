// Package nome junta nome e sobrenome sem repetir o trecho que o diretório
// já gravou nos dois lados (o AD e o GLPI costumam fazer isso).
package nome

import "strings"

// Limpar junta as duas partes e tira o bloco de palavras que se repete.
func Limpar(primeiro, ultimo string) string {
	return SemRepeticao(Juntar(primeiro, ultimo))
}

// Juntar cola as duas partes. Se uma já contém a outra, fica a maior.
// Se o fim da primeira é o começo da segunda, o trecho comum entra uma vez.
func Juntar(primeiro, ultimo string) string {
	a := strings.Fields(primeiro)
	b := strings.Fields(ultimo)
	switch {
	case len(a) == 0 && len(b) == 0:
		return ""
	case len(a) == 0:
		return strings.Join(b, " ")
	case len(b) == 0:
		return strings.Join(a, " ")
	}
	if contem(a, b) {
		return strings.Join(a, " ")
	}
	if contem(b, a) {
		return strings.Join(b, " ")
	}
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for k := max; k >= 1; k-- {
		if iguais(a[len(a)-k:], b[:k]) {
			return strings.Join(append(append([]string{}, a...), b[k:]...), " ")
		}
	}
	return strings.Join(append(append([]string{}, a...), b...), " ")
}

// SemRepeticao tira a cópia mais longa de um bloco de palavras seguido
// dele mesmo. "Ana Souza" permanece. A comparação ignora maiúsculas.
func SemRepeticao(s string) string {
	w := strings.Fields(s)
	if len(w) < 2 {
		return strings.Join(w, " ")
	}
	for k := len(w) / 2; k >= 1; k-- {
		for i := 0; i+2*k <= len(w); i++ {
			if iguais(w[i:i+k], w[i+k:i+2*k]) {
				out := append([]string{}, w[:i+k]...)
				out = append(out, w[i+2*k:]...)
				return strings.Join(out, " ")
			}
		}
	}
	return strings.Join(w, " ")
}

func contem(texto, trecho []string) bool {
	if len(trecho) == 0 || len(trecho) > len(texto) {
		return false
	}
	for i := 0; i+len(trecho) <= len(texto); i++ {
		if iguais(texto[i:i+len(trecho)], trecho) {
			return true
		}
	}
	return false
}

func iguais(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}
