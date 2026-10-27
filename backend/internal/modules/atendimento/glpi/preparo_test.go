package glpi

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// memoria é uma área de preparo em memória: Leitor de um lado, Preparo do
// outro. falha faz o passo pedido falhar.
type memoria struct {
	itens map[string]map[int]ItemPreparo
	falha string
}

func novaMemoria() *memoria { return &memoria{itens: map[string]map[int]ItemPreparo{}} }

func (m *memoria) guardar(it ItemPreparo) {
	if m.itens[it.Tipo] == nil {
		m.itens[it.Tipo] = map[int]ItemPreparo{}
	}
	m.itens[it.Tipo][it.ID] = it
}

func (m *memoria) Itens(_ context.Context, tipo string, de, limite int) ([]ItemPreparo, int, error) {
	if m.falha == "itens:"+tipo {
		return nil, 0, errFonte
	}
	ids := []int{}
	for id := range m.itens[tipo] {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := []ItemPreparo{}
	for i := de; i < len(ids) && i < de+limite; i++ {
		out = append(out, m.itens[tipo][ids[i]])
	}
	return out, len(ids), nil
}

func (m *memoria) Item(_ context.Context, tipo string, id int) (ItemPreparo, bool, error) {
	if m.falha == "item" {
		return ItemPreparo{}, false, errFonte
	}
	it, ok := m.itens[tipo][id]
	return it, ok, nil
}

func (m *memoria) Versoes(context.Context) (map[int]string, error) {
	if m.falha == "versoes" {
		return nil, errFonte
	}
	out := map[int]string{}
	for id, it := range m.itens[ItemChamado] {
		out[id] = it.Versao
	}
	return out, nil
}

func (m *memoria) Pessoas(context.Context) (map[int]bool, error) {
	if m.falha == "pessoas" {
		return nil, errFonte
	}
	out := map[int]bool{}
	for id := range m.itens[ItemPessoa] {
		out[id] = true
	}
	return out, nil
}

func (m *memoria) Gravar(_ context.Context, itens []ItemPreparo) error {
	if m.falha == "gravar" {
		return errFonte
	}
	for _, it := range itens {
		m.guardar(it)
	}
	return nil
}

func dados(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// origemDeTeste é um GLPI em memória: a FontePreparo sobre uma memória já
// cheia também é uma origem.
func origemDeTeste(t *testing.T) *memoria {
	t.Helper()
	o := novaMemoria()
	o.guardar(ItemPreparo{Tipo: ItemCategoria, ID: 7, Dados: dados(t, Categoria{ID: 7, Nome: "TI > E-mail"})})
	o.guardar(ItemPreparo{Tipo: ItemEntidade, ID: 1, Dados: dados(t, Entidade{ID: 1, Nome: "RAIZ > SMS"})})
	o.guardar(ItemPreparo{Tipo: ItemSLA, ID: 2, Dados: dados(t, SLA{ID: 2, Nome: "Solução", Tempo: 4, Unidade: "hour"})})
	o.guardar(ItemPreparo{Tipo: ItemSLA, ID: 3, Dados: dados(t, SLA{ID: 3, Nome: "Sem tempo", Unidade: "semana"})})
	cinco := 5
	longo := strings.Repeat("palavra ", 600)
	acomps := []Acompanhamento{}
	for i := 0; i < 400; i++ {
		acomps = append(acomps, Acompanhamento{ID: i, Texto: "&lt;p&gt;" + longo + "&lt;/p&gt;", AutorID: 6})
	}
	o.guardar(ItemPreparo{Tipo: ItemChamado, ID: 101, Dados: dados(t, Pacote{
		Chamado:         Chamado{ID: 101, Titulo: "E-mail", Conteudo: "&lt;b&gt;fora&lt;/b&gt;", Tipo: 1, CategoriaID: 7, EntidadeID: 1, SLASolucao: 2, Modificado: "v1"},
		Participantes:   []Participante{{PessoaID: 5, Papel: 1}, {PessoaID: 6, Papel: 2}},
		Acompanhamentos: acomps,
		Satisfacao:      &Satisfacao{Nota: &cinco},
	})})
	o.guardar(ItemPreparo{Tipo: ItemChamado, ID: 102, Dados: dados(t, Pacote{Chamado: Chamado{ID: 102, Titulo: "Acesso", Tipo: 2, Modificado: "v1"}})})
	o.guardar(ItemPreparo{Tipo: ItemPessoa, ID: 5, Dados: dados(t, pessoaPreparada{Pessoa: Pessoa{ID: 5, Login: "ana", Nome: "Ana"}, Email: "ana@x.test"})})
	o.guardar(ItemPreparo{Tipo: ItemPessoa, ID: 6, Dados: dados(t, pessoaPreparada{Pessoa: Pessoa{ID: 6, Login: "tec"}})})
	return o
}

func TestExtrairParaAreaDePreparo(t *testing.T) {
	ctx := context.Background()
	origem := &FontePreparo{Leitor: origemDeTeste(t)}
	destino := novaMemoria()
	vistos := []int{}
	r, err := Extrair(ctx, origem, destino, 0, func(feitos, total int) { vistos = append(vistos, feitos, total) })
	if err != nil || r.Chamados != 2 || r.Novos != 2 || r.Pessoas != 2 || r.Categorias != 1 || r.Entidades != 1 || r.SLAs != 2 {
		t.Fatalf("resumo: %+v %v", r, err)
	}
	if len(vistos) != 2 || vistos[1] != 2 {
		t.Fatalf("progresso: %v", vistos)
	}
	ch := destino.itens[ItemChamado][101]
	if ch.Classe != "incidente" || *ch.Entidade != 1 || *ch.Categoria != 7 || ch.SLA == nil || *ch.SLA != 2 || ch.Versao != "v1" {
		t.Fatalf("chamado preparado: %+v", ch)
	}
	if len(ch.Dados) > LimiteItem {
		t.Fatalf("o chamado cabe num lote: %d", len(ch.Dados))
	}
	var pac Pacote
	_ = json.Unmarshal(ch.Dados, &pac)
	if pac.Chamado.Conteudo != "fora" || len(pac.Acompanhamentos) == 0 || len(pac.Acompanhamentos) == 400 || strings.Contains(pac.Acompanhamentos[0].Texto, "<p>") {
		t.Fatalf("texto limpo e mensagens cortadas: %q %d", pac.Chamado.Conteudo, len(pac.Acompanhamentos))
	}
	if destino.itens[ItemSLA][3].Minutos != nil || *destino.itens[ItemSLA][2].Minutos != 240 {
		t.Fatalf("minutos do SLA: %+v", destino.itens[ItemSLA])
	}
	if destino.itens[ItemChamado][102].SLA != nil {
		t.Fatal("chamado sem SLA")
	}

	// De novo: o que não mudou não é lido.
	if r, err := Extrair(ctx, origem, destino, 1, nil); err != nil || r.Iguais != 2 || r.Novos != 0 || r.Pessoas != 0 {
		t.Fatalf("incremental: %+v %v", r, err)
	}

	// A FontePreparo lê o que a extração gravou, para o importador.
	f := &FontePreparo{Leitor: destino}
	if cs, total, err := f.Chamados(ctx, 0, 0); err != nil || total != 2 || len(cs) != 1 || cs[0].ID != 101 {
		t.Fatalf("página: %+v %d %v", cs, total, err)
	}
	if ps, _ := f.Participantes(ctx, 101); len(ps) != 2 {
		t.Fatalf("participantes: %+v", ps)
	}
	if as, _ := f.Acompanhamentos(ctx, 101); len(as) == 0 {
		t.Fatal("acompanhamentos")
	}
	if s, _ := f.Satisfacao(ctx, 101); s == nil || *s.Nota != 5 {
		t.Fatalf("satisfação: %+v", s)
	}
	if p, err := f.Pessoa(ctx, 5); err != nil || p.Email != "ana@x.test" || p.Login != "ana" {
		t.Fatalf("pessoa com e-mail: %+v %v", p, err)
	}
	if p, err := f.Pessoa(ctx, 404); err != nil || p.ID != 404 || p.NomeCompleto() != "" {
		t.Fatalf("pessoa que não veio: %+v %v", p, err)
	}
}

// origemFalha falha no passo pedido.
type origemFalha struct {
	*FontePreparo
	passo string
}

func (o origemFalha) Participantes(ctx context.Context, id int) ([]Participante, error) {
	if o.passo == "participantes" {
		return nil, errFonte
	}
	return o.FontePreparo.Participantes(ctx, id)
}
func (o origemFalha) Acompanhamentos(ctx context.Context, id int) ([]Acompanhamento, error) {
	if o.passo == "acompanhamentos" {
		return nil, errFonte
	}
	return o.FontePreparo.Acompanhamentos(ctx, id)
}
func (o origemFalha) Satisfacao(ctx context.Context, id int) (*Satisfacao, error) {
	if o.passo == "satisfacao" {
		return nil, errFonte
	}
	return o.FontePreparo.Satisfacao(ctx, id)
}
func (o origemFalha) Pessoa(ctx context.Context, id int) (Pessoa, error) {
	if o.passo == "pessoa" || (o.passo == "autor" && id == 6) {
		return Pessoa{}, errFonte
	}
	return o.FontePreparo.Pessoa(ctx, id)
}

func TestExtrairFalhas(t *testing.T) {
	ctx := context.Background()
	for _, passo := range []string{"itens:categoria", "itens:entidade", "itens:sla", "itens:chamado"} {
		o := origemDeTeste(t)
		o.falha = passo
		if _, err := Extrair(ctx, &FontePreparo{Leitor: o}, novaMemoria(), 0, nil); !errors.Is(err, errFonte) {
			t.Errorf("origem %s: %v", passo, err)
		}
	}
	for _, passo := range []string{"versoes", "pessoas", "gravar"} {
		d := novaMemoria()
		d.falha = passo
		if _, err := Extrair(ctx, &FontePreparo{Leitor: origemDeTeste(t)}, d, 0, nil); !errors.Is(err, errFonte) {
			t.Errorf("destino %s: %v", passo, err)
		}
	}
	// Uma pessoa a mais por item: o lote enche antes do fim e grava no meio.
	cheio := origemDeTeste(t)
	for i := 0; i < LoteExtracao+5; i++ {
		cheio.guardar(ItemPreparo{Tipo: ItemCategoria, ID: 1000 + i, Dados: dados(t, Categoria{ID: 1000 + i})})
	}
	d := novaMemoria()
	d.falha = "gravar"
	if _, err := Extrair(ctx, &FontePreparo{Leitor: cheio}, d, 0, nil); !errors.Is(err, errFonte) {
		t.Fatalf("lote cheio: %v", err)
	}
	for _, passo := range []string{"participantes", "acompanhamentos", "satisfacao", "pessoa", "autor"} {
		o := origemFalha{FontePreparo: &FontePreparo{Leitor: origemDeTeste(t)}, passo: passo}
		if _, err := Extrair(ctx, o, novaMemoria(), 0, nil); !errors.Is(err, errFonte) {
			t.Errorf("%s: %v", passo, err)
		}
	}
	// Um item que não é JSON do tipo certo não vira dado.
	m := novaMemoria()
	m.guardar(ItemPreparo{Tipo: ItemPessoa, ID: 1, Dados: json.RawMessage(`[]`)})
	for _, tipo := range []string{ItemCategoria, ItemEntidade, ItemSLA, ItemChamado} {
		m.guardar(ItemPreparo{Tipo: tipo, ID: 1, Dados: json.RawMessage(`"x"`)})
	}
	f := &FontePreparo{Leitor: m}
	if _, err := f.Categorias(ctx); !errors.Is(err, ErrGLPI) {
		t.Errorf("categoria ilegível: %v", err)
	}
	if _, err := f.Entidades(ctx); !errors.Is(err, ErrGLPI) {
		t.Errorf("entidade ilegível: %v", err)
	}
	if _, err := f.SLAs(ctx); !errors.Is(err, ErrGLPI) {
		t.Errorf("SLA ilegível: %v", err)
	}
	if _, _, err := f.Chamados(ctx, 0, 9); !errors.Is(err, ErrGLPI) {
		t.Errorf("chamado ilegível: %v", err)
	}
	if _, err := f.Pessoa(ctx, 1); !errors.Is(err, ErrGLPI) {
		t.Errorf("pessoa ilegível: %v", err)
	}
	m.falha = "item"
	if _, err := f.Pessoa(ctx, 1); !errors.Is(err, errFonte) {
		t.Errorf("leitura da pessoa: %v", err)
	}
	m.falha = "itens:chamado"
	if _, _, err := f.Chamados(ctx, 0, 9); !errors.Is(err, errFonte) {
		t.Errorf("leitura dos chamados: %v", err)
	}
	if _, err := Extrair(ctx, origemSemTicket{FontePreparo: &FontePreparo{Leitor: origemDeTeste(t)}}, novaMemoria(), 0, nil); err == nil {
		t.Error("a página de chamados falhou")
	}
}

// origemSemTicket lê a estrutura e falha ao listar os chamados.
type origemSemTicket struct{ *FontePreparo }

func (origemSemTicket) Chamados(context.Context, int, int) ([]Chamado, int, error) {
	return nil, 0, errFonte
}

func TestNovoMapa(t *testing.T) {
	u := ptrUnidade()
	m := NovoMapa([]LinhaMapa{
		{Secretaria: " sms ", Categoria: 7, Destino: Destino{Unidade: u, UnidadeNome: "TI", Oferta: "PCs", Tipo: "incidente", Prioridade: "alta"}},
		{Secretaria: "SMS", Categoria: -1, Destino: Destino{UnidadeNome: "Falta", Oferta: "Geral", Tipo: "requisicao", Prioridade: "media"}},
		{Secretaria: "PREF", Categoria: -1, Destino: Destino{UnidadeNome: "Falta", Oferta: "Geral", Tipo: "requisicao", Prioridade: "media"}},
	})
	if d, ok := m.Destino("SMS", 7); !ok || d.Oferta != "PCs" {
		t.Fatalf("linha exata: %+v", d)
	}
	if d, ok := m.Destino("sms", 99); !ok || d.Oferta != "Geral" {
		t.Fatalf("o * da secretaria: %+v", d)
	}
	if len(m.Faltam) != 1 || m.Faltam[0] != "Falta" {
		t.Fatalf("unidade que falta, uma vez: %v", m.Faltam)
	}
}

func ptrUnidade() *uuid.UUID {
	u := uuid.New()
	return &u
}
