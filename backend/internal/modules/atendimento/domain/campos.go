package domain

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const maxCampos = 20

var chaveCampo = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// Campo é uma pergunta da oferta, respondida na abertura.
type Campo struct {
	Chave       string   `json:"chave"`
	Rotulo      string   `json:"rotulo"`
	Tipo        string   `json:"tipo"`
	Obrigatorio bool     `json:"obrigatorio"`
	Opcoes      []string `json:"opcoes,omitempty"`
}

// CampoInvalido explica a pergunta ou a resposta que não entrou.
type CampoInvalido struct{ Msg string }

func (e CampoInvalido) Error() string { return e.Msg }

// ValidarCampos normaliza as perguntas da oferta.
func ValidarCampos(campos []Campo) ([]Campo, error) {
	if len(campos) > maxCampos {
		return nil, CampoInvalido{"uma oferta tem no máximo 20 campos"}
	}
	vistos := map[string]bool{}
	out := make([]Campo, 0, len(campos))
	for _, c := range campos {
		c.Chave = strings.TrimSpace(c.Chave)
		c.Rotulo = strings.TrimSpace(c.Rotulo)
		c.Tipo = strings.TrimSpace(c.Tipo)
		if c.Chave == "" {
			c.Chave = chaveDe(c.Rotulo)
		}
		if !chaveCampo.MatchString(c.Chave) || c.Rotulo == "" || utf8.RuneCountInString(c.Rotulo) > 80 {
			return nil, CampoInvalido{"cada campo precisa de um rótulo de até 80 caracteres"}
		}
		if vistos[c.Chave] {
			return nil, CampoInvalido{"há dois campos com o mesmo nome"}
		}
		vistos[c.Chave] = true
		if c.Tipo == "" {
			c.Tipo = "texto"
		}
		if !slices.Contains([]string{"texto", "numero", "lista"}, c.Tipo) {
			return nil, CampoInvalido{"o tipo do campo é texto, número ou lista"}
		}
		opcoes := make([]string, 0, len(c.Opcoes))
		for _, op := range c.Opcoes {
			op = strings.TrimSpace(op)
			if op != "" && utf8.RuneCountInString(op) <= 80 {
				opcoes = append(opcoes, op)
			}
		}
		if c.Tipo == "lista" && len(opcoes) == 0 {
			return nil, CampoInvalido{"a lista " + c.Rotulo + " precisa de opções"}
		}
		if c.Tipo != "lista" {
			opcoes = nil
		}
		c.Opcoes = opcoes
		out = append(out, c)
	}
	return out, nil
}

// ValidarRespostas confere o que foi respondido na abertura. Campo a mais
// é recusado. Obrigatório vazio também.
func ValidarRespostas(campos []Campo, respostas map[string]string) (map[string]string, error) {
	if respostas == nil {
		respostas = map[string]string{}
	}
	conhecidos := map[string]Campo{}
	for _, c := range campos {
		conhecidos[c.Chave] = c
	}
	out := map[string]string{}
	for chave, valor := range respostas {
		c, ok := conhecidos[chave]
		if !ok {
			return nil, CampoInvalido{"há resposta para um campo que a oferta não pede"}
		}
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		if utf8.RuneCountInString(valor) > 400 {
			return nil, CampoInvalido{"a resposta de " + c.Rotulo + " passa de 400 caracteres"}
		}
		switch c.Tipo {
		case "numero":
			for _, r := range valor {
				if r < '0' || r > '9' {
					return nil, CampoInvalido{c.Rotulo + " recebe só números"}
				}
			}
		case "lista":
			if !slices.Contains(c.Opcoes, valor) {
				return nil, CampoInvalido{c.Rotulo + " não tem essa opção"}
			}
		}
		out[chave] = valor
	}
	for _, c := range campos {
		if c.Obrigatorio && out[c.Chave] == "" {
			return nil, CampoInvalido{"informe " + c.Rotulo}
		}
	}
	return out, nil
}

func chaveDe(rotulo string) string {
	var b strings.Builder
	ultimo := false
	for _, r := range strings.ToLower(rotulo) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			ultimo = false
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 && !ultimo {
				b.WriteByte('_')
				ultimo = true
			}
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		s = "c_" + s
	}
	if utf8.RuneCountInString(s) > 40 {
		s = string([]rune(s)[:40])
	}
	return strings.Trim(s, "_")
}
