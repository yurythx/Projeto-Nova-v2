package glpi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// PaginaPadrao é quantos chamados se leem por vez.
const PaginaPadrao = 100

// Importador lê a Fonte e grava no atendimento. Sem Gravar, só simula.
type Importador struct {
	Fonte   Fonte
	Pool    *pgxpool.Pool
	Repo    domain.Repository
	Unidade *uuid.UUID
	Gravar  bool
	// Fuso das datas do GLPI, que vêm sem fuso, na hora do servidor dele.
	Fuso   *time.Location
	Pagina int

	categorias map[int]string
	ofertas    map[string]uuid.UUID
	novas      map[string]bool
	pessoas    map[int]*pessoaCasada
}

type pessoaCasada struct {
	id   *uuid.UUID
	nome string
}

// Relatorio é o resultado da execução.
type Relatorio struct {
	Gravou       bool     `json:"gravou"`
	Lidos        int      `json:"lidos"`
	Importados   int      `json:"importados"`
	JaImportados int      `json:"ja_importados"`
	OfertasNovas int      `json:"ofertas_novas"`
	SemPar       []string `json:"sem_par"`
	Erros        []string `json:"erros"`
}

// Executar importa os chamados ainda não importados, página a página.
// Cada chamado entra na sua transação: o erro de um não desfaz os outros.
func (im *Importador) Executar(ctx context.Context) (Relatorio, error) {
	rel := Relatorio{Gravou: im.Gravar, SemPar: []string{}, Erros: []string{}}
	if im.Fuso == nil {
		im.Fuso = time.UTC
	}
	if im.Pagina <= 0 {
		im.Pagina = PaginaPadrao
	}
	cats, err := im.Fonte.Categorias(ctx)
	if err != nil {
		return rel, fmt.Errorf("categorias: %w", err)
	}
	im.categorias = map[int]string{}
	for _, c := range cats {
		im.categorias[c.ID] = c.Nome
	}
	if im.ofertas, err = im.jaImportados(ctx, "oferta"); err != nil {
		return rel, err
	}
	feitos, err := im.jaImportados(ctx, "solicitacao")
	if err != nil {
		return rel, err
	}
	im.novas, im.pessoas = map[string]bool{}, map[int]*pessoaCasada{}
	for de := 0; ; de += im.Pagina {
		pagina, total, err := im.Fonte.Chamados(ctx, de, de+im.Pagina-1)
		if err != nil {
			return rel, fmt.Errorf("chamados %d: %w", de, err)
		}
		for _, ch := range pagina {
			rel.Lidos++
			if _, ok := feitos[strconv.Itoa(ch.ID)]; ok {
				rel.JaImportados++
				continue
			}
			if err := im.importar(ctx, ch, &rel); err != nil {
				rel.Erros = append(rel.Erros, fmt.Sprintf("chamado %d: %v", ch.ID, err))
				continue
			}
			rel.Importados++
		}
		if len(pagina) == 0 || de+len(pagina) >= total {
			break
		}
	}
	rel.OfertasNovas = len(im.novas)
	for _, p := range im.pessoas {
		if p.id == nil && p.nome != "" {
			rel.SemPar = append(rel.SemPar, p.nome)
		}
	}
	if im.Gravar {
		err = database.WithTx(ctx, im.Pool, func(ctx context.Context, tx pgx.Tx) error {
			return audit.NewWriter(tx).Record(ctx, audit.Entry{
				Action: "atendimento.importacao.glpi", ResourceType: "atendimento_importacao", ResourceID: Origem,
				Metadata: map[string]any{"lidos": rel.Lidos, "importados": rel.Importados, "ja_importados": rel.JaImportados,
					"ofertas_novas": rel.OfertasNovas, "sem_par": len(rel.SemPar), "erros": len(rel.Erros)},
			})
		})
	}
	return rel, err
}

func (im *Importador) jaImportados(ctx context.Context, tipo string) (map[string]uuid.UUID, error) {
	rows, err := im.Pool.Query(ctx, `SELECT id_origem, destino FROM atendimento_importacao WHERE origem = $1 AND tipo = $2`, Origem, tipo)
	if err != nil {
		return nil, fmt.Errorf("importados: %w", err)
	}
	defer rows.Close()
	out := map[string]uuid.UUID{}
	for rows.Next() {
		var id string
		var destino uuid.UUID
		if err := rows.Scan(&id, &destino); err != nil {
			return nil, err
		}
		out[id] = destino
	}
	return out, rows.Err()
}

// importar lê o que falta do chamado e, gravando, grava tudo de uma vez.
func (im *Importador) importar(ctx context.Context, ch Chamado, rel *Relatorio) error {
	partes, err := im.Fonte.Participantes(ctx, ch.ID)
	if err != nil {
		return err
	}
	var solicitante, tecnico *pessoaCasada
	for _, p := range partes {
		switch {
		case p.Papel == 1 && solicitante == nil:
			solicitante, err = im.pessoa(ctx, p.PessoaID)
		case p.Papel == 2 && tecnico == nil:
			tecnico, err = im.pessoa(ctx, p.PessoaID)
		}
		if err != nil {
			return err
		}
	}
	acomps, err := im.Fonte.Acompanhamentos(ctx, ch.ID)
	if err != nil {
		return err
	}
	autores := make([]*pessoaCasada, len(acomps))
	for i, a := range acomps {
		if autores[i], err = im.pessoa(ctx, a.AutorID); err != nil {
			return err
		}
	}
	sat, err := im.Fonte.Satisfacao(ctx, ch.ID)
	if err != nil {
		return err
	}
	tipo := Tipo(ch.Tipo)
	chave := fmt.Sprintf("%d:%s", ch.CategoriaID, tipo)
	if _, ok := im.ofertas[chave]; !ok {
		im.novas[chave] = true
	}
	if !im.Gravar {
		return nil
	}
	_, conhecida := im.ofertas[chave]
	err = database.WithTx(ctx, im.Pool, func(ctx context.Context, tx pgx.Tx) error {
		oferta, err := im.oferta(ctx, tx, chave, ch.CategoriaID, tipo)
		if err != nil {
			return err
		}
		sol, err := im.gravarChamado(ctx, tx, ch, oferta, tipo, solicitante, tecnico)
		if err != nil {
			return err
		}
		for i, a := range acomps {
			if err := im.gravarAcompanhamento(ctx, tx, sol, a, autores[i]); err != nil {
				return err
			}
		}
		if sat != nil && domain.PodeAvaliar(sol) {
			fim := sol.CreatedAt
			if sol.ResolvidaEm != nil {
				fim = *sol.ResolvidaEm
			}
			if _, err := im.Repo.InsertAvaliacao(ctx, tx, domain.Avaliacao{
				ID: uuid.New(), SolicitacaoID: sol.ID, AutorID: sol.SolicitanteID, Nota: Nota(*sat.Nota),
				Comentario: Texto(sat.Comentario, domain.TextoMax), CreatedAt: im.data(sat.Em, fim),
			}); err != nil {
				return err
			}
		}
		return marcar(ctx, tx, "solicitacao", strconv.Itoa(ch.ID), sol.ID)
	})
	if err != nil && !conhecida {
		// A oferta criada nesta transação foi desfeita com ela.
		delete(im.ofertas, chave)
	}
	return err
}

// oferta devolve a oferta da categoria e do tipo, criando-a inativa: a
// organização revisa o catálogo antes de abrir para pedidos novos.
func (im *Importador) oferta(ctx context.Context, tx pgx.Tx, chave string, categoria int, tipo string) (uuid.UUID, error) {
	if id, ok := im.ofertas[chave]; ok {
		return id, nil
	}
	nome := im.categorias[categoria]
	if strings.TrimSpace(nome) == "" {
		nome = "Sem categoria"
	}
	o, err := domain.NovaOferta(Cortar(Texto(nome, domain.NomeMax), domain.NomeMax), "Importada do GLPI.", tipo, nil, im.Unidade, false)
	if err != nil {
		return uuid.Nil, err
	}
	if o, err = im.Repo.InsertOferta(ctx, tx, o); err != nil {
		return uuid.Nil, err
	}
	if err := marcar(ctx, tx, "oferta", chave, o.ID); err != nil {
		return uuid.Nil, err
	}
	im.ofertas[chave] = o.ID
	return o.ID, nil
}

func (im *Importador) gravarChamado(ctx context.Context, tx pgx.Tx, ch Chamado, oferta uuid.UUID, tipo string, solicitante, tecnico *pessoaCasada) (domain.Solicitacao, error) {
	abertura := im.data(ch.Abertura, time.Now().UTC())
	estado := Estado(ch.Status, ch.Tipo)
	titulo := Texto(ch.Titulo, domain.NomeMax)
	if titulo == "" {
		titulo = fmt.Sprintf("Chamado %d do GLPI", ch.ID)
	}
	descricao := Texto(ch.Conteudo, domain.TextoMax)
	quem := ContaImportacao
	if solicitante != nil && solicitante.id != nil {
		quem = *solicitante.id
	} else if solicitante != nil && solicitante.nome != "" {
		descricao = Cortar("Pedido por "+solicitante.nome+" no GLPI.\n\n"+descricao, domain.TextoMax)
	}
	sol := domain.Solicitacao{
		ID: uuid.New(), Tipo: tipo, Estado: estado, Titulo: titulo, Descricao: strings.TrimSpace(descricao),
		OfertaID: oferta, UnidadeID: im.Unidade, SolicitanteID: quem,
		Impacto: Nivel(ch.Impacto), Urgencia: Nivel(ch.Urgencia), Respostas: map[string]string{},
		CreatedAt: abertura, UpdatedAt: abertura,
	}
	if tecnico != nil {
		sol.ResponsavelID = tecnico.id
	}
	var fim *time.Time
	switch estado {
	case domain.EstadoResolvido, domain.EstadoAtendido, domain.EstadoEncerrado:
		r := im.data(ch.Solucao, im.data(ch.Fechamento, abertura))
		sol.ResolvidaEm, fim = &r, &r
		if estado == domain.EstadoEncerrado {
			e := im.data(ch.Fechamento, r)
			sol.EncerradaEm, fim = &e, &e
		}
		sol.UpdatedAt = *fim
	}
	salva, err := im.Repo.InsertSolicitacao(ctx, tx, sol)
	if err != nil {
		return domain.Solicitacao{}, err
	}
	salva.ResolvidaEm, salva.EncerradaEm = sol.ResolvidaEm, sol.EncerradaEm
	if err := im.Repo.UpdateSolicitacao(ctx, tx, salva); err != nil {
		return domain.Solicitacao{}, err
	}
	// O histórico mínimo: aberto, em atendimento quando houve, e o fim.
	autor := ContaImportacao
	if salva.ResponsavelID != nil {
		autor = *salva.ResponsavelID
	}
	aberto := domain.EstadoAberto
	passos := []domain.Passagem{{Para: aberto, AutorID: &quem, Em: abertura}}
	if estado != aberto {
		emAtendimento := domain.EstadoEmAtendimento
		passos = append(passos, domain.Passagem{De: &aberto, Para: emAtendimento, AutorID: &autor, Em: abertura})
		if estado != emAtendimento {
			quando := abertura
			if fim != nil {
				quando = *fim
			}
			passos = append(passos, domain.Passagem{De: &emAtendimento, Para: estado, AutorID: &autor, Em: quando})
		}
	}
	for _, p := range passos {
		p.SolicitacaoID = salva.ID
		if _, err := im.Repo.InsertPassagem(ctx, tx, p); err != nil {
			return domain.Solicitacao{}, err
		}
	}
	return salva, nil
}

func (im *Importador) gravarAcompanhamento(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, a Acompanhamento, autor *pessoaCasada) error {
	texto := Texto(a.Texto, domain.TextoMax)
	quem := ContaImportacao
	if autor != nil && autor.id != nil {
		quem = *autor.id
	} else if autor != nil && autor.nome != "" {
		texto = Cortar("Escrito por "+autor.nome+" no GLPI.\n\n"+texto, domain.TextoMax)
	}
	if strings.TrimSpace(texto) == "" {
		return nil
	}
	_, err := im.Repo.InsertComentario(ctx, tx, domain.Comentario{
		ID: uuid.New(), SolicitacaoID: sol.ID, AutorID: &quem, Texto: strings.TrimSpace(texto),
		Interno: a.Privado == 1, CreatedAt: im.data(a.Em, sol.CreatedAt),
	})
	return err
}

// pessoa casa a pessoa do GLPI com um usuário daqui, pelo e-mail e depois
// pelo login. A resposta fica guardada: cada pessoa se pergunta uma vez.
func (im *Importador) pessoa(ctx context.Context, id int) (*pessoaCasada, error) {
	if id <= 0 {
		return nil, nil
	}
	if p, ok := im.pessoas[id]; ok {
		return p, nil
	}
	g, err := im.Fonte.Pessoa(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pessoa %d: %w", id, err)
	}
	p := &pessoaCasada{nome: g.NomeCompleto()}
	var achado uuid.UUID
	err = im.Pool.QueryRow(ctx, `SELECT id FROM users
		WHERE ($1 <> '' AND lower(email) = lower($1)) OR ($2 <> '' AND lower(username) = lower($2))
		ORDER BY ($1 <> '' AND lower(email) = lower($1)) DESC, active DESC LIMIT 1`, strings.TrimSpace(g.Email), strings.TrimSpace(g.Login)).Scan(&achado)
	switch {
	case database.IsNoRows(err):
	case err != nil:
		return nil, fmt.Errorf("pessoa %d: %w", id, err)
	default:
		p.id = &achado
	}
	im.pessoas[id] = p
	return p, nil
}

// data lê a data do GLPI ("2006-01-02 15:04:05", no fuso dele). Vazia ou
// ilegível, vale o padrão.
func (im *Importador) data(s string, padrao time.Time) time.Time {
	t, err := time.ParseInLocation(time.DateTime, strings.TrimSpace(s), im.Fuso)
	if err != nil {
		return padrao
	}
	return t.UTC()
}

func marcar(ctx context.Context, tx pgx.Tx, tipo, id string, destino uuid.UUID) error {
	_, err := tx.Exec(ctx, `INSERT INTO atendimento_importacao (origem, tipo, id_origem, destino) VALUES ($1, $2, $3, $4)`, Origem, tipo, id, destino)
	return err
}
