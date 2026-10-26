package domain

import (
	"regexp"
	"slices"
	"strings"
)

// chaveNova é gerada. O nome pode mudar; a chave não.
var chaveNova = regexp.MustCompile(`^c[0-9a-f]{8}$`)

// terminais não originam passagem nova: o fluxo de fábrica já encerrou.
var terminais = map[string][]string{
	"problemas": {"encerrado"},
	"mudancas":  {"concluida", "falhou", "cancelada"},
	"projetos":  {"cancelado"},
	"fila":      {"encerrado", "cancelado"},
}

// reservadas continuam com o papel que já decide. A coluna nova não as usa.
var reservadas = map[string][]string{
	"mudancas": {"aprovada", "rejeitada"},
}

// Nova é o pedido de uma coluna só neste quadro.
type Nova struct {
	Nome  string
	Cor   string
	Ordem int
	De    []string
	Para  []string
}

// Personalizada diz se a chave nasceu nesta tela.
func Personalizada(chave string) bool {
	return chaveNova.MatchString(chave)
}

// Criar acrescenta a coluna e as passagens. A ordem final começa em zero.
// de é de onde o card entra; para é para onde ele sai. Sem entrada, a
// coluna não nasce.
func Criar(cols []Coluna, chave string, n Nova) (Coluna, []Passagem, []Coluna, error) {
	if !Personalizada(chave) {
		return Coluna{}, nil, nil, ErrPassagem
	}
	if len(cols) == 0 {
		return Coluna{}, nil, nil, ErrQuadro
	}
	quadro := cols[0].Quadro
	nome := strings.TrimSpace(n.Nome)
	if nome == "" || len([]rune(nome)) > 80 {
		return Coluna{}, nil, nil, ErrNome
	}
	cor := strings.TrimSpace(n.Cor)
	if !corHex.MatchString(cor) {
		return Coluna{}, nil, nil, ErrCor
	}
	de, err := conferirPontas(cols, quadro, chave, n.De, true)
	if err != nil {
		return Coluna{}, nil, nil, err
	}
	para, err := conferirPontas(cols, quadro, chave, n.Para, false)
	if err != nil {
		return Coluna{}, nil, nil, err
	}
	if len(de) == 0 {
		return Coluna{}, nil, nil, ErrSemEntrada
	}
	col := Coluna{
		Quadro: quadro, Chave: chave, Nome: nome, Cor: cor, Tom: "info",
		Ordem: maiorOrdem(cols) + 1, Entrada: false, Ativa: true,
		De: de, Para: para,
	}
	out, err := mover(append(slices.Clone(cols), col), chave, n.Ordem)
	if err != nil {
		return Coluna{}, nil, nil, err
	}
	pronta, _ := acharColuna(out, chave)
	pronta.De, pronta.Para = de, para
	return pronta, montarPassagens(chave, de, para), out, nil
}

func conferirPontas(cols []Coluna, quadro, chave string, pontas []string, origem bool) ([]string, error) {
	var out []string
	for _, ponta := range pontas {
		ponta = strings.TrimSpace(ponta)
		if ponta == "" || ponta == chave || slices.Contains(out, ponta) {
			return nil, ErrPassagem
		}
		col, ok := acharColuna(cols, ponta)
		if !ok || !col.Ativa || col.Quadro != quadro {
			return nil, ErrPassagem
		}
		if slices.Contains(reservadas[quadro], ponta) {
			return nil, ErrPassagem
		}
		if origem && slices.Contains(terminais[quadro], ponta) {
			return nil, ErrPassagem
		}
		out = append(out, ponta)
	}
	return out, nil
}

func montarPassagens(chave string, de, para []string) []Passagem {
	var out []Passagem
	for _, origem := range de {
		out = append(out, Passagem{De: origem, Para: chave})
	}
	for _, destino := range para {
		out = append(out, Passagem{De: chave, Para: destino})
	}
	return out
}

func acharColuna(cols []Coluna, chave string) (Coluna, bool) {
	for _, c := range cols {
		if c.Chave == chave {
			return c, true
		}
	}
	return Coluna{}, false
}
