package domain

import (
	"testing"

	"github.com/google/uuid"
)

func ptr(n int) *int { return &n }

func TestInventarioESugestao(t *testing.T) {
	entidades := map[int]string{1: "RAIZ > PREFEITURA", 2: "RAIZ > PREFEITURA > Projetos", 4: "RAIZ > SMS"}
	categorias := map[int]string{7: "Serviços > Computadores", 8: "Serviços > Redes", 9: "Outros", 70: "Serviços > Computadores"}
	slas := []NivelSLA{{ID: 2, Nome: "Solução N1", Minutos: 240}, {ID: 4, Nome: "Solução N2", Minutos: 1440},
		{ID: 6, Nome: "Solução N3", Minutos: 2160}, {ID: 9, Nome: "Sem uso", Minutos: 60}}
	grupos := []Grupo{
		// PREFEITURA e a subentidade Projetos juntam na mesma secretaria.
		{Entidade: 1, Categoria: 7, Chamados: 100, ComPrazo: 80, SLA: ptr(2), Classe: "requisicao", Primeiro: "2025-02-01", Ultimo: "2026-10-01"},
		{Entidade: 2, Categoria: 7, Chamados: 10, ComPrazo: 1, SLA: ptr(4), Classe: "incidente", Primeiro: "2025-01-01", Ultimo: "2026-11-01"},
		{Entidade: 1, Categoria: 8, Chamados: 50, ComPrazo: 20, SLA: ptr(6), Classe: "incidente"},
		{Entidade: 1, Categoria: 9, Chamados: 5, ComPrazo: 5, SLA: ptr(4)},
		{Entidade: 1, Categoria: 0, Chamados: 30, Primeiro: ""},
		{Entidade: 4, Categoria: 70, Chamados: 40},
		{Entidade: 4, Categoria: 3, Chamados: 2, SLA: ptr(99)},
		{Entidade: 5, Categoria: 8, Chamados: 1},
	}
	inv := MontarInventario(grupos, entidades, categorias, slas, map[string]int{ItemChamado: 238})
	if len(inv.Grupos) != 7 || inv.Secretarias["PREFEITURA"] != 195 || inv.Secretarias["SMS"] != 42 || inv.Itens[ItemChamado] != 238 {
		t.Fatalf("grupos: %+v %+v", inv.Grupos, inv.Secretarias)
	}
	comp := inv.Grupos[0]
	if comp.Secretaria != "(entidade 5)" {
		comp = inv.Grupos[1]
	}
	for _, g := range inv.Grupos {
		if g.Secretaria == "PREFEITURA" && g.Categoria == 7 {
			comp = g
		}
	}
	if comp.Chamados != 110 || comp.ComPrazo != 81 || comp.Nivel != "Solução N1" || comp.Classe != "requisicao" ||
		comp.Primeiro != "2025-01-01" || comp.Ultimo != "2026-11-01" {
		t.Fatalf("PREFEITURA/7 junto: %+v", comp)
	}
	prioridade := map[string]string{}
	for _, n := range inv.Niveis {
		prioridade[n.Nome] = n.Prioridade
	}
	if prioridade["Solução N1"] != "alta" || prioridade["Solução N2"] != "media" || prioridade["Solução N3"] != "baixa" || prioridade["Sem uso"] != "" {
		t.Fatalf("escala dos prazos usados: %+v", inv.Niveis)
	}

	u := uuid.New()
	linhas := Sugerir(inv, map[string]uuid.UUID{" prefeitura ": u})
	porChave := map[string]Linha{}
	for _, l := range linhas {
		porChave[l.Secretaria+"/"+string(rune('0'+l.Categoria+1))] = l
	}
	achar := func(sec string, cat int) Linha {
		for _, l := range linhas {
			if l.Secretaria == sec && l.Categoria == cat {
				return l
			}
		}
		t.Fatalf("sem a linha %s/%d: %+v", sec, cat, linhas)
		return Linha{}
	}
	if l := achar("PREFEITURA", 7); l.Oferta != "Computadores" || l.Prioridade != "alta" || l.UnidadeID == nil || *l.UnidadeID != u || l.Tipo != "requisicao" {
		t.Fatalf("PREFEITURA/7: %+v", l)
	}
	if l := achar("PREFEITURA", 0); l.Oferta != OfertaGeral || l.Prioridade != "media" || l.Tipo != "requisicao" {
		t.Fatalf("sem categoria: %+v", l)
	}
	if l := achar("SMS", 70); l.Prioridade != "alta" || l.Nivel != "Solução N1" || l.UnidadeID != nil {
		t.Fatalf("a mesma categoria em outra secretaria herda o prazo: %+v", l)
	}
	if l := achar("SMS", 3); l.Oferta != OfertaGeral || l.CategoriaNome != "(categoria 3 apagada)" {
		t.Fatalf("categoria apagada: %+v", l)
	}
	if l := achar("PREFEITURA", CategoriaQualquer); l.Oferta != OfertaGeral || l.UnidadeID == nil {
		t.Fatalf("a linha * da secretaria: %+v", l)
	}
	if err := ValidarTabela(linhas); err != nil {
		t.Fatalf("a sugestão já é uma tabela válida: %v", err)
	}
	if len(porChave) == 0 {
		t.Fatal("sem linhas")
	}

	// Um prazo só é alta; dois são alta e média.
	um := MontarInventario([]Grupo{{Entidade: 1, Categoria: 7, Chamados: 1, ComPrazo: 1, SLA: ptr(2)}}, entidades, categorias,
		[]NivelSLA{{ID: 2, Nome: "N1", Minutos: 240}}, nil)
	if um.Niveis[0].Prioridade != "alta" {
		t.Fatalf("um prazo: %+v", um.Niveis)
	}
}
