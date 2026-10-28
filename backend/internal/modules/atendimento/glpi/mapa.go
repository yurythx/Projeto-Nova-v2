package glpi

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

// Mapa é a tabela de-para revisada pela organização (ADR 075): para cada
// secretaria do GLPI e categoria, a unidade e a oferta daqui, o tipo e a
// prioridade. A categoria "*" é o destino padrão da secretaria. Sem mapa,
// vale o ADR 072: uma oferta inativa por categoria, na unidade do comando.
type Mapa struct {
	linhas map[chaveMapa]Destino
	// Faltam são as unidades da tabela que ainda não existem aqui.
	Faltam []string
}

type chaveMapa struct {
	secretaria string
	categoria  int // -1 é o "*"
}

// Destino é para onde vão os chamados de uma linha da tabela.
type Destino struct {
	Unidade     *uuid.UUID // nil enquanto a unidade não existe
	UnidadeNome string
	Oferta      string
	// Catalogo é a categoria do catálogo da oferta criada (ADR 078); vazio
	// a deixa em "Outros".
	Catalogo   string
	Tipo       string
	Prioridade string
}

// colunas que a tabela precisa ter; as outras (volume, observação) são
// para quem revisa.
// bom é a marca que o Excel põe no começo do CSV em UTF-8.
var bom = string(rune(0xFEFF))

var colunasMapa = []string{"glpi_secretaria", "glpi_categoria_id", "unidade", "oferta", "tipo", "prioridade"}

// LerMapa lê a tabela (CSV com ";", como o Excel salva em português).
// unidades é o nome de cada unidade daqui; a comparação ignora caixa.
func LerMapa(r io.Reader, unidades map[string]uuid.UUID) (*Mapa, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'
	cr.FieldsPerRecord = -1
	todas, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("mapa: %w", err)
	}
	if len(todas) < 2 {
		return nil, errors.New("mapa: a tabela está vazia")
	}
	col := map[string]int{}
	for i, nome := range todas[0] {
		col[strings.TrimPrefix(strings.TrimSpace(nome), bom)] = i
	}
	for _, c := range colunasMapa {
		if _, ok := col[c]; !ok {
			return nil, fmt.Errorf("mapa: falta a coluna %q", c)
		}
	}
	porNome := map[string]uuid.UUID{}
	for nome, id := range unidades {
		porNome[strings.ToLower(strings.TrimSpace(nome))] = id
	}
	m := &Mapa{linhas: map[chaveMapa]Destino{}}
	for n, linha := range todas[1:] {
		// Coluna opcional ausente (catalogo, nas tabelas antigas) é vazia,
		// não a primeira coluna.
		campo := func(c string) string {
			if i, ok := col[c]; ok && i < len(linha) {
				return strings.TrimSpace(linha[i])
			}
			return ""
		}
		k := chaveMapa{secretaria: strings.ToUpper(campo("glpi_secretaria")), categoria: -1}
		if c := campo("glpi_categoria_id"); c != "*" {
			if k.categoria, err = strconv.Atoi(c); err != nil || k.categoria < 0 {
				return nil, fmt.Errorf("mapa, linha %d: categoria %q não é um número nem *", n+2, c)
			}
		}
		d := Destino{UnidadeNome: campo("unidade"), Oferta: campo("oferta"), Catalogo: campo("catalogo"), Tipo: campo("tipo"),
			Prioridade: campo("prioridade")}
		switch {
		case k.secretaria == "" || d.UnidadeNome == "" || d.Oferta == "":
			return nil, fmt.Errorf("mapa, linha %d: secretaria, unidade e oferta são obrigatórias", n+2)
		case !domain.TipoValido(d.Tipo):
			return nil, fmt.Errorf("mapa, linha %d: o tipo é incidente ou requisicao, não %q", n+2, d.Tipo)
		case !slices.Contains(domain.Prioridades, d.Prioridade):
			return nil, fmt.Errorf("mapa, linha %d: a prioridade é critica, alta, media ou baixa, não %q", n+2, d.Prioridade)
		}
		if _, repetida := m.linhas[k]; repetida {
			return nil, fmt.Errorf("mapa, linha %d: %s e categoria %s aparecem duas vezes", n+2, k.secretaria, campo("glpi_categoria_id"))
		}
		if id, ok := porNome[strings.ToLower(d.UnidadeNome)]; ok {
			d.Unidade = &id
		} else if !slices.Contains(m.Faltam, d.UnidadeNome) {
			m.Faltam = append(m.Faltam, d.UnidadeNome)
		}
		m.linhas[k] = d
	}
	slices.Sort(m.Faltam)
	return m, nil
}

// LinhaMapa é uma linha da tabela já resolvida: a tela das integrações
// guarda a unidade pelo id. Categoria -1 é o "*".
type LinhaMapa struct {
	Secretaria string
	Categoria  int
	Destino    Destino
}

// NovoMapa monta o mapa das linhas resolvidas.
func NovoMapa(linhas []LinhaMapa) *Mapa {
	m := &Mapa{linhas: map[chaveMapa]Destino{}}
	for _, l := range linhas {
		m.linhas[chaveMapa{strings.ToUpper(strings.TrimSpace(l.Secretaria)), l.Categoria}] = l.Destino
		if l.Destino.Unidade == nil && !slices.Contains(m.Faltam, l.Destino.UnidadeNome) {
			m.Faltam = append(m.Faltam, l.Destino.UnidadeNome)
		}
	}
	slices.Sort(m.Faltam)
	return m
}

// Destino acha a linha da secretaria e da categoria, ou o "*" da secretaria.
func (m *Mapa) Destino(secretaria string, categoria int) (Destino, bool) {
	s := strings.ToUpper(strings.TrimSpace(secretaria))
	if d, ok := m.linhas[chaveMapa{s, categoria}]; ok {
		return d, true
	}
	d, ok := m.linhas[chaveMapa{s, -1}]
	return d, ok
}

// Secretaria é o 2º nível da árvore de entidades do GLPI ("RAIZ > SMS >
// Hospital" é SMS), que é o que separa as categorias e o SLA. A raiz é ela
// mesma.
func Secretaria(completo string) string {
	partes := strings.Split(completo, " > ")
	if len(partes) > 1 {
		return strings.TrimSpace(partes[1])
	}
	return strings.TrimSpace(partes[0])
}

// MetaMin converte o tempo de um SLA do GLPI em minutos. Dia é corrido (24
// horas): o GLPI conta o dia pelo calendário, que aqui é a política.
func MetaMin(n int, unidade string) (int, bool) {
	switch unidade {
	case "minute":
		return n, n > 0
	case "hour":
		return n * 60, n > 0
	case "day":
		return n * 24 * 60, n > 0
	}
	return 0, false
}
