package glpi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

// A área de preparo (ADR 076): o GLPI é lido de uma vez para registros
// brutos, e o importador lê desses registros, não da API. A extração pode
// rodar longe do servidor (no conector, na rede do GLPI) e ser repetida:
// só traz o chamado que mudou.

// Tipos de item da área de preparo.
const (
	ItemChamado   = "chamado"
	ItemPessoa    = "pessoa"
	ItemCategoria = "categoria"
	ItemEntidade  = "entidade"
	ItemSLA       = "sla"
)

// ItemPreparo é um registro da área de preparo. Os campos fora de Dados
// são o que o inventário agrupa sem abrir o JSON.
type ItemPreparo struct {
	Tipo      string `json:"tipo"`
	ID        int    `json:"id"`
	Versao    string `json:"versao,omitempty"`
	Nome      string `json:"nome,omitempty"`
	Entidade  *int   `json:"entidade,omitempty"`
	Categoria *int   `json:"categoria,omitempty"`
	SLA       *int   `json:"sla,omitempty"`
	Minutos   *int   `json:"minutos,omitempty"`
	// Classe é como a origem classifica o chamado: incidente ou requisicao.
	Classe string          `json:"classe,omitempty"`
	Data   string          `json:"data,omitempty"`
	Dados  json.RawMessage `json:"dados"`
}

// Pacote é o chamado com o que se lê dele: a unidade da área de preparo.
type Pacote struct {
	Chamado         Chamado          `json:"chamado"`
	Participantes   []Participante   `json:"participantes"`
	Acompanhamentos []Acompanhamento `json:"acompanhamentos"`
	Satisfacao      *Satisfacao      `json:"satisfacao,omitempty"`
}

// pessoaPreparada guarda o e-mail, que a Pessoa da API não serializa.
type pessoaPreparada struct {
	Pessoa
	Email string `json:"email"`
}

// Preparo é onde a extração grava: o banco (no servidor) ou a API (no
// conector).
type Preparo interface {
	// Versoes diz, de cada chamado já extraído, a versão (date_mod).
	Versoes(ctx context.Context) (map[int]string, error)
	// Pessoas são os ids de pessoa já extraídos.
	Pessoas(ctx context.Context) (map[int]bool, error)
	Gravar(ctx context.Context, itens []ItemPreparo) error
}

// Extraivel é o que a extração lê: a API do GLPI, com a árvore e os SLAs.
type Extraivel interface {
	Fonte
	Estrutura
}

// ResumoExtracao é o que a extração fez.
type ResumoExtracao struct {
	Chamados   int `json:"chamados"`
	Novos      int `json:"novos_ou_alterados"`
	Iguais     int `json:"iguais"`
	Pessoas    int `json:"pessoas"`
	Categorias int `json:"categorias"`
	Entidades  int `json:"entidades"`
	SLAs       int `json:"slas"`
}

// LoteExtracao é quantos itens vão por gravação.
const LoteExtracao = 50

// Extrair lê a origem e grava na área de preparo. O chamado igual ao já
// extraído (mesma versão) não é lido de novo. progresso recebe os chamados
// vistos e o total.
func Extrair(ctx context.Context, o Extraivel, destino Preparo, pagina int, progresso func(feitos, total int)) (ResumoExtracao, error) {
	var r ResumoExtracao
	if pagina <= 0 {
		pagina = PaginaPadrao
	}
	lote := []ItemPreparo{}
	gravar := func(forcar bool) error {
		if len(lote) == 0 || (!forcar && len(lote) < LoteExtracao) {
			return nil
		}
		err := destino.Gravar(ctx, lote)
		lote = lote[:0]
		return err
	}
	incluir := func(it ItemPreparo, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		it.Dados = b
		lote = append(lote, it)
		return gravar(false)
	}

	cats, err := o.Categorias(ctx)
	if err != nil {
		return r, fmt.Errorf("categorias: %w", err)
	}
	for _, c := range cats {
		if err := incluir(ItemPreparo{Tipo: ItemCategoria, ID: c.ID, Nome: c.Nome}, c); err != nil {
			return r, err
		}
	}
	ents, err := o.Entidades(ctx)
	if err != nil {
		return r, fmt.Errorf("entidades: %w", err)
	}
	for _, e := range ents {
		if err := incluir(ItemPreparo{Tipo: ItemEntidade, ID: e.ID, Nome: e.Nome}, e); err != nil {
			return r, err
		}
	}
	slas, err := o.SLAs(ctx)
	if err != nil {
		return r, fmt.Errorf("SLAs: %w", err)
	}
	for _, s := range slas {
		it := ItemPreparo{Tipo: ItemSLA, ID: s.ID, Nome: s.Nome}
		if m, ok := MetaMin(s.Tempo, s.Unidade); ok {
			it.Minutos = &m
		}
		if err := incluir(it, s); err != nil {
			return r, err
		}
	}
	r.Categorias, r.Entidades, r.SLAs = len(cats), len(ents), len(slas)

	versoes, err := destino.Versoes(ctx)
	if err != nil {
		return r, err
	}
	pessoas, err := destino.Pessoas(ctx)
	if err != nil {
		return r, err
	}
	pessoa := func(id int) error {
		if id <= 0 || pessoas[id] {
			return nil
		}
		pessoas[id] = true
		p, err := o.Pessoa(ctx, id)
		if err != nil {
			return fmt.Errorf("pessoa %d: %w", id, err)
		}
		r.Pessoas++
		return incluir(ItemPreparo{Tipo: ItemPessoa, ID: id, Nome: p.NomeCompleto()}, pessoaPreparada{Pessoa: p, Email: p.Email})
	}
	for de := 0; ; de += pagina {
		chamados, total, err := o.Chamados(ctx, de, de+pagina-1)
		if err != nil {
			return r, fmt.Errorf("chamados %d: %w", de, err)
		}
		for _, ch := range chamados {
			r.Chamados++
			if v, ok := versoes[ch.ID]; ok && v == ch.Modificado && v != "" {
				r.Iguais++
				continue
			}
			pac := Pacote{Chamado: ch}
			if pac.Participantes, err = o.Participantes(ctx, ch.ID); err != nil {
				return r, fmt.Errorf("chamado %d: %w", ch.ID, err)
			}
			if pac.Acompanhamentos, err = o.Acompanhamentos(ctx, ch.ID); err != nil {
				return r, fmt.Errorf("chamado %d: %w", ch.ID, err)
			}
			if pac.Satisfacao, err = o.Satisfacao(ctx, ch.ID); err != nil {
				return r, fmt.Errorf("chamado %d: %w", ch.ID, err)
			}
			encolher(&pac)
			for _, p := range pac.Participantes {
				if err := pessoa(p.PessoaID); err != nil {
					return r, err
				}
			}
			for _, a := range pac.Acompanhamentos {
				if err := pessoa(a.AutorID); err != nil {
					return r, err
				}
			}
			ent, cat := ch.EntidadeID, ch.CategoriaID
			it := ItemPreparo{Tipo: ItemChamado, ID: ch.ID, Versao: ch.Modificado, Entidade: &ent, Categoria: &cat, Data: ch.Abertura, Classe: Tipo(ch.Tipo)}
			if ch.SLASolucao > 0 {
				sla := ch.SLASolucao
				it.SLA = &sla
			}
			if err := incluir(it, pac); err != nil {
				return r, err
			}
			r.Novos++
		}
		if progresso != nil {
			progresso(r.Chamados, total)
		}
		if len(chamados) == 0 || de+len(chamados) >= total {
			break
		}
	}
	return r, gravar(true)
}

// LimiteItem é o tamanho máximo de um chamado na área de preparo: ele
// precisa caber num lote do conector (1 MiB por requisição).
const LimiteItem = 800 << 10

// encolher guarda o texto já limpo, no limite do atendimento (a carga
// cortaria do mesmo jeito) e, se ainda passar do limite, tira as últimas
// mensagens.
func encolher(p *Pacote) {
	p.Chamado.Conteudo = Texto(p.Chamado.Conteudo, domain.TextoMax)
	for i := range p.Acompanhamentos {
		p.Acompanhamentos[i].Texto = Texto(p.Acompanhamentos[i].Texto, domain.TextoMax)
	}
	for len(p.Acompanhamentos) > 0 {
		if b, _ := json.Marshal(p); len(b) <= LimiteItem {
			return
		}
		p.Acompanhamentos = p.Acompanhamentos[:len(p.Acompanhamentos)-1]
	}
}

// Leitor lê a área de preparo, na ordem do id da origem.
type Leitor interface {
	Itens(ctx context.Context, tipo string, de, limite int) ([]ItemPreparo, int, error)
	Item(ctx context.Context, tipo string, id int) (ItemPreparo, bool, error)
}

// FontePreparo é a Fonte do importador sobre a área de preparo: o mesmo
// importador carrega da API ou do que a extração trouxe.
type FontePreparo struct {
	Leitor  Leitor
	pacotes map[int]Pacote
}

var _ Extraivel = (*FontePreparo)(nil)

func decodificar[T any](itens []ItemPreparo) ([]T, error) {
	out := make([]T, 0, len(itens))
	for _, it := range itens {
		var v T
		if err := json.Unmarshal(it.Dados, &v); err != nil {
			return nil, fmt.Errorf("%w: %s %d: %v", ErrGLPI, it.Tipo, it.ID, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// limiteTudo lê de uma vez as listas pequenas (categorias, entidades, SLAs).
const limiteTudo = 100000

func (f *FontePreparo) tudo(ctx context.Context, tipo string) ([]ItemPreparo, error) {
	itens, _, err := f.Leitor.Itens(ctx, tipo, 0, limiteTudo)
	return itens, err
}

func (f *FontePreparo) Categorias(ctx context.Context) ([]Categoria, error) {
	itens, err := f.tudo(ctx, ItemCategoria)
	if err != nil {
		return nil, err
	}
	return decodificar[Categoria](itens)
}

func (f *FontePreparo) Entidades(ctx context.Context) ([]Entidade, error) {
	itens, err := f.tudo(ctx, ItemEntidade)
	if err != nil {
		return nil, err
	}
	return decodificar[Entidade](itens)
}

func (f *FontePreparo) SLAs(ctx context.Context) ([]SLA, error) {
	itens, err := f.tudo(ctx, ItemSLA)
	if err != nil {
		return nil, err
	}
	return decodificar[SLA](itens)
}

// Chamados lê a página e guarda os pacotes: participantes, acompanhamentos
// e satisfação vêm deles, sem outra leitura.
func (f *FontePreparo) Chamados(ctx context.Context, de, ate int) ([]Chamado, int, error) {
	itens, total, err := f.Leitor.Itens(ctx, ItemChamado, de, ate-de+1)
	if err != nil {
		return nil, 0, err
	}
	pacotes, err := decodificar[Pacote](itens)
	if err != nil {
		return nil, 0, err
	}
	f.pacotes = map[int]Pacote{}
	out := make([]Chamado, 0, len(pacotes))
	for _, p := range pacotes {
		f.pacotes[p.Chamado.ID] = p
		out = append(out, p.Chamado)
	}
	return out, total, nil
}

func (f *FontePreparo) Participantes(_ context.Context, chamado int) ([]Participante, error) {
	return f.pacotes[chamado].Participantes, nil
}

func (f *FontePreparo) Acompanhamentos(_ context.Context, chamado int) ([]Acompanhamento, error) {
	return f.pacotes[chamado].Acompanhamentos, nil
}

func (f *FontePreparo) Satisfacao(_ context.Context, chamado int) (*Satisfacao, error) {
	return f.pacotes[chamado].Satisfacao, nil
}

// Pessoa sem registro volta vazia: fica na conta de importação, sem nome.
func (f *FontePreparo) Pessoa(ctx context.Context, id int) (Pessoa, error) {
	it, ok, err := f.Leitor.Item(ctx, ItemPessoa, id)
	if err != nil || !ok {
		return Pessoa{ID: id}, err
	}
	var p pessoaPreparada
	if err := json.Unmarshal(it.Dados, &p); err != nil {
		return Pessoa{}, fmt.Errorf("%w: pessoa %d: %v", ErrGLPI, id, err)
	}
	p.Pessoa.Email = p.Email
	return p.Pessoa, nil
}
