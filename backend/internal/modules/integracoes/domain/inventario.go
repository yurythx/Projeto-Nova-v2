package domain

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Grupo é o que a área de preparo tem de uma entidade e uma categoria da
// origem: o banco agrupa, o domínio junta por secretaria.
type Grupo struct {
	Entidade  int
	Categoria int
	Chamados  int
	ComPrazo  int
	SLA       *int // o prazo de solução mais comum
	Classe    string
	Primeiro  string
	Ultimo    string
}

// NivelSLA é um prazo da origem, com a prioridade que ele sugere.
type NivelSLA struct {
	ID         int    `json:"id"`
	Nome       string `json:"nome"`
	Minutos    int    `json:"minutos"`
	Chamados   int    `json:"chamados"`
	Prioridade string `json:"prioridade,omitempty"`
}

// GrupoInventario é uma linha do inventário: secretaria e categoria.
type GrupoInventario struct {
	Secretaria    string `json:"secretaria"`
	Categoria     int    `json:"categoria"`
	CategoriaNome string `json:"categoria_nome"`
	Chamados      int    `json:"chamados"`
	ComPrazo      int    `json:"com_prazo"`
	Nivel         string `json:"nivel"`
	Classe        string `json:"classe"`
	Primeiro      string `json:"primeiro"`
	Ultimo        string `json:"ultimo"`
}

// Inventario é o retrato da área de preparo.
type Inventario struct {
	Itens       map[string]int    `json:"itens"`
	Secretarias map[string]int    `json:"secretarias"`
	Grupos      []GrupoInventario `json:"grupos"`
	Niveis      []NivelSLA        `json:"niveis"`
	ExtraidoEm  *time.Time        `json:"extraido_em,omitempty"`
}

// MontarInventario junta os grupos por secretaria e dá a cada prazo da
// origem a prioridade que ele sugere: o mais curto é alta, o mais longo é
// baixa.
func MontarInventario(grupos []Grupo, entidades, categorias map[int]string, slas []NivelSLA, itens map[string]int) Inventario {
	inv := Inventario{Itens: itens, Secretarias: map[string]int{}, Grupos: []GrupoInventario{}, Niveis: []NivelSLA{}}
	porSLA := map[int]*NivelSLA{}
	for i := range slas {
		porSLA[slas[i].ID] = &slas[i]
	}
	type acumulado struct {
		GrupoInventario
		melhorPrazo, melhorClasse int
	}
	juntos := map[string]*acumulado{}
	ordem := []string{}
	for _, g := range grupos {
		sec := "(entidade " + fmt.Sprint(g.Entidade) + ")"
		if nome, ok := entidades[g.Entidade]; ok {
			sec = Secretaria(nome)
		}
		chave := fmt.Sprintf("%s\x00%d", sec, g.Categoria)
		a, ok := juntos[chave]
		if !ok {
			a = &acumulado{GrupoInventario: GrupoInventario{Secretaria: sec, Categoria: g.Categoria, CategoriaNome: nomeCategoria(g.Categoria, categorias),
				Primeiro: g.Primeiro, Ultimo: g.Ultimo}}
			juntos[chave] = a
			ordem = append(ordem, chave)
		}
		a.Chamados += g.Chamados
		a.ComPrazo += g.ComPrazo
		inv.Secretarias[sec] += g.Chamados
		if g.Primeiro != "" && (a.Primeiro == "" || g.Primeiro < a.Primeiro) {
			a.Primeiro = g.Primeiro
		}
		if g.Ultimo > a.Ultimo {
			a.Ultimo = g.Ultimo
		}
		if n, ok := porSLA[deref(g.SLA)]; ok && g.SLA != nil {
			n.Chamados += g.ComPrazo
			if g.ComPrazo > a.melhorPrazo {
				a.Nivel, a.melhorPrazo = n.Nome, g.ComPrazo
			}
		}
		if g.Classe != "" && g.Chamados > a.melhorClasse {
			a.Classe, a.melhorClasse = g.Classe, g.Chamados
		}
	}
	for _, chave := range ordem {
		inv.Grupos = append(inv.Grupos, juntos[chave].GrupoInventario)
	}
	slices.SortStableFunc(inv.Grupos, func(a, b GrupoInventario) int {
		return cmp.Or(cmp.Compare(a.Secretaria, b.Secretaria), cmp.Compare(b.Chamados, a.Chamados))
	})
	// Só os prazos que algum chamado usou entram na escala.
	usados := []*NivelSLA{}
	for i := range slas {
		if slas[i].Chamados > 0 && slas[i].Minutos > 0 {
			usados = append(usados, &slas[i])
		}
	}
	slices.SortFunc(usados, func(a, b *NivelSLA) int { return cmp.Compare(a.Minutos, b.Minutos) })
	escala := []string{"alta", "media", "baixa"}
	for i, n := range usados {
		n.Prioridade = escala[min(len(escala)-1, i*len(escala)/len(usados))]
	}
	inv.Niveis = append(inv.Niveis, slas...)
	slices.SortFunc(inv.Niveis, func(a, b NivelSLA) int { return cmp.Compare(a.Minutos, b.Minutos) })
	return inv
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func nomeCategoria(id int, categorias map[int]string) string {
	if id == 0 {
		return "(sem categoria)"
	}
	if nome, ok := categorias[id]; ok {
		return nome
	}
	return fmt.Sprintf("(categoria %d apagada)", id)
}

// curto é o último nível do nome ("Serviços > Computadores" é
// Computadores). Nome entre parênteses é o que a origem não tem.
func curto(nome string) string {
	if strings.HasPrefix(nome, "(") {
		return ""
	}
	partes := strings.Split(nome, ">")
	return strings.TrimSpace(partes[len(partes)-1])
}

// Sugerir monta a tabela a partir do inventário: a oferta é o nome da
// categoria, a prioridade vem do prazo que os chamados tiveram e, sem
// prazo, do prazo que a mesma categoria teve em outra secretaria. Cada
// secretaria ganha a linha "*", para a categoria que surgir depois.
// unidades leva a secretaria à unidade daqui (sem caixa).
func Sugerir(inv Inventario, unidades map[string]uuid.UUID) []Linha {
	prioridade := map[string]string{}
	for _, n := range inv.Niveis {
		if n.Prioridade != "" {
			prioridade[n.Nome] = n.Prioridade
		}
	}
	nivelDoNome, peso := map[string]string{}, map[string]int{}
	for _, g := range inv.Grupos {
		if c := curto(g.CategoriaNome); c != "" && g.Nivel != "" && g.ComPrazo > peso[c] {
			nivelDoNome[c], peso[c] = g.Nivel, g.ComPrazo
		}
	}
	unidade := func(sec string) *uuid.UUID {
		for nome, id := range unidades {
			if strings.EqualFold(strings.TrimSpace(nome), sec) {
				return &id
			}
		}
		return nil
	}
	linhas, vistas := []Linha{}, map[string]bool{}
	for _, g := range inv.Grupos {
		c := curto(g.CategoriaNome)
		l := Linha{Secretaria: g.Secretaria, Categoria: g.Categoria, CategoriaNome: g.CategoriaNome, Chamados: g.Chamados,
			ComPrazo: g.ComPrazo, Nivel: g.Nivel, UnidadeID: unidade(g.Secretaria), Oferta: c, Tipo: g.Classe}
		if l.Oferta == "" {
			l.Oferta = OfertaGeral
		}
		if l.Nivel == "" {
			l.Nivel = nivelDoNome[c]
		}
		l.Prioridade = cmp.Or(prioridade[l.Nivel], "media")
		if !slices.Contains(Tipos, l.Tipo) {
			l.Tipo = "requisicao"
		}
		linhas = append(linhas, l)
		if !vistas[g.Secretaria] {
			vistas[g.Secretaria] = true
			linhas = append(linhas, Linha{Secretaria: g.Secretaria, Categoria: CategoriaQualquer, CategoriaNome: "(qualquer outra)",
				UnidadeID: unidade(g.Secretaria), Oferta: OfertaGeral, Tipo: "requisicao", Prioridade: "media"})
		}
	}
	return linhas
}
