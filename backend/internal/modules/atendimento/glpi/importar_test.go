package glpi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// glpiFalso responde como a API REST do GLPI, com os dados do teste.
type glpiFalso struct {
	categorias []Categoria
	chamados   []Chamado
	partes     map[int][]Participante
	acomps     map[int][]Acompanhamento
	satisf     map[int][]Satisfacao
	pessoas    map[int]Pessoa
	emails     map[int]string
	entidades  []Entidade
	slas       []SLA
	quebrado   string // caminho que responde 500
	sessoes    int
}

func (g *glpiFalso) servidor(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho := strings.TrimPrefix(r.URL.Path, "/apirest.php")
		if g.quebrado != "" && caminho == g.quebrado {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if caminho != "/initSession" && r.Header.Get("Session-Token") != "s1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		responder := func(v any) { _ = json.NewEncoder(w).Encode(v) }
		var id int
		switch {
		case caminho == "/initSession":
			if r.Header.Get("Authorization") != "user_token u1" || r.Header.Get("App-Token") != "a1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			g.sessoes++
			responder(map[string]string{"session_token": "s1"})
		case caminho == "/killSession":
			g.sessoes--
		case caminho == "/ITILCategory":
			responder(g.categorias)
		case caminho == "/Entity":
			responder(g.entidades)
		case caminho == "/SLA":
			responder(g.slas)
		case caminho == "/Ticket" && r.URL.Query().Get("sort") != "id":
			// Como o GLPI 10: sort pelo número da opção de busca é 400.
			w.WriteHeader(http.StatusBadRequest)
		case caminho == "/Ticket":
			var de, ate int
			_, _ = fmt.Sscanf(r.URL.Query().Get("range"), "%d-%d", &de, &ate)
			if de >= len(g.chamados) {
				w.WriteHeader(http.StatusBadRequest)
				responder([]string{"ERROR_RANGE_EXCEED_TOTAL", "fora"})
				return
			}
			ate = min(ate, len(g.chamados)-1)
			w.Header().Set("Content-Range", fmt.Sprintf("%d-%d/%d", de, ate, len(g.chamados)))
			w.WriteHeader(http.StatusPartialContent)
			responder(g.chamados[de : ate+1])
		case sscan(caminho, "/Ticket/%d/Ticket_User", &id):
			responder(g.partes[id])
		case sscan(caminho, "/Ticket/%d/ITILFollowup", &id):
			responder(g.acomps[id])
		case sscan(caminho, "/Ticket/%d/TicketSatisfaction", &id):
			if _, ok := g.satisf[id]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			responder(g.satisf[id])
		case sscan(caminho, "/User/%d/UserEmail", &id):
			responder([]map[string]any{{"email": "outro@x.test", "is_default": 0}, {"email": g.emails[id], "is_default": 1}})
		case sscan(caminho, "/User/%d", &id):
			responder(g.pessoas[id])
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func sscan(caminho, formato string, id *int) bool {
	n, err := fmt.Sscanf(caminho, formato, id)
	return err == nil && n == 1 && fmt.Sprintf(formato, *id) == caminho
}

func TestImportacaoDoGLPI(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	u := dbtest.Unidade(t, pool)
	tecnico := dbtest.User(t, pool)
	var tecEmail string
	if err := pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, tecnico).Scan(&tecEmail); err != nil {
		t.Fatal(err)
	}
	cinco := 5
	g := &glpiFalso{
		categorias: []Categoria{{ID: 7, Nome: "TI > E-mail"}},
		chamados: []Chamado{
			{ID: 101, Titulo: "E-mail fora", Conteudo: "&lt;p&gt;Não envia&lt;/p&gt;", Abertura: "2025-03-10 09:00:00",
				Solucao: "2025-03-10 11:30:00", Fechamento: "2025-03-11 08:00:00", Status: 6, Tipo: 1, Urgencia: 4, Impacto: 3, CategoriaID: 7},
			{ID: 102, Titulo: "", Conteudo: "Pedido de acesso", Abertura: "2025-04-01 14:00:00", Status: 2, Tipo: 2, Urgencia: 1, Impacto: 1,
				Modificado: "2025-04-02 10:00:00"},
			{ID: 103, Titulo: "Impressora", Abertura: "ilegível", Status: 1, Tipo: 1, CategoriaID: 7},
		},
		partes: map[int][]Participante{
			101: {{PessoaID: 1, Papel: 1}, {PessoaID: 2, Papel: 2}, {PessoaID: 1, Papel: 3}},
			102: {{PessoaID: 1, Papel: 1}},
		},
		acomps: map[int][]Acompanhamento{
			101: {{ID: 1, Texto: "Reiniciei o serviço", Em: "2025-03-10 11:00:00", AutorID: 2},
				{ID: 2, Texto: "Ver com o fornecedor", Em: "2025-03-10 11:10:00", AutorID: 2, Privado: 1},
				{ID: 3, Texto: "Obrigada!", Em: "2025-03-10 12:00:00", AutorID: 1},
				{ID: 4, Texto: "&lt;p&gt; &lt;/p&gt;", AutorID: 2}},
		},
		satisf:  map[int][]Satisfacao{101: {{Nota: &cinco, Comentario: "Rápido", Em: "2025-03-11 09:00:00"}}},
		pessoas: map[int]Pessoa{1: {ID: 1, Login: "ana.glpi", Nome: "Ana", Sobrenome: "Souza"}, 2: {ID: 2, Login: "tec.glpi"}},
		emails:  map[int]string{2: tecEmail},
	}
	srv := g.servidor(t)
	cliente := NovoCliente(srv.URL+"/apirest.php/", "a1", "u1")
	if err := cliente.Abrir(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	})
	_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	sp, _ := time.LoadLocation("America/Sao_Paulo")
	novo := func(gravar bool) *Importador {
		return &Importador{Fonte: cliente, Pool: pool, Repo: infrastructure.NewRepository(), Unidade: &u, Gravar: gravar, Fuso: sp, Pagina: 2}
	}

	sim, err := novo(false).Executar(ctx)
	if err != nil || sim.Gravou || sim.Lidos != 3 || sim.Importados != 3 || sim.OfertasNovas != 2 || len(sim.SemPar) != 1 || sim.SemPar[0] != "Ana Souza" {
		t.Fatalf("simulação: %+v %v", sim, err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_solicitacoes WHERE unidade_id = $1`, u).Scan(&n)
	if n != 0 {
		t.Fatal("a simulação não grava")
	}

	rel, err := novo(true).Executar(ctx)
	if err != nil || rel.Importados != 3 || rel.OfertasNovas != 2 || len(rel.Erros) != 0 {
		t.Fatalf("gravar: %+v %v", rel, err)
	}
	repo := infrastructure.NewRepository()
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT destino FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao' AND id_origem = '101'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	sol, err := repo.GetSolicitacao(ctx, pool, id)
	if err != nil {
		t.Fatal(err)
	}
	abertura := time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)
	if sol.Estado != domain.EstadoEncerrado || !sol.CreatedAt.Equal(abertura) || sol.ResolvidaEm == nil || !sol.ResolvidaEm.Equal(abertura.Add(150*time.Minute)) ||
		sol.EncerradaEm == nil || sol.SolicitanteID != ContaImportacao || sol.ResponsavelID == nil || *sol.ResponsavelID != tecnico ||
		sol.Urgencia != 3 || sol.Impacto != 2 || sol.Titulo != "E-mail fora" || !strings.HasPrefix(sol.Descricao, "Pedido por Ana Souza no GLPI.") {
		t.Fatalf("chamado 101: %+v", sol)
	}
	// Sem a data de mudança do GLPI, a última atualização é o fim.
	if !sol.UpdatedAt.Equal(*sol.EncerradaEm) {
		t.Fatalf("última atualização do 101: %v", sol.UpdatedAt)
	}
	ps, _ := repo.ListPassagens(ctx, pool, id)
	if len(ps) != 3 || ps[2].Para != domain.EstadoEncerrado {
		t.Fatalf("passagens: %+v", ps)
	}
	cs, _ := repo.ListComentarios(ctx, pool, id, true)
	if len(cs) != 3 || !cs[1].Interno || !strings.HasPrefix(cs[2].Texto, "Escrito por Ana Souza no GLPI.") {
		t.Fatalf("comentários: %+v", cs)
	}
	if a, err := repo.GetAvaliacao(ctx, pool, id); err != nil || a.Nota != 5 || a.Comentario != "Rápido" {
		t.Fatalf("avaliação: %+v %v", a, err)
	}
	o, _ := repo.GetOferta(ctx, pool, sol.OfertaID)
	if o.Nome != "TI > E-mail" || o.Ativa || o.Tipo != domain.TipoIncidente {
		t.Fatalf("oferta inativa da categoria: %+v", o)
	}
	_ = pool.QueryRow(ctx, `SELECT destino FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao' AND id_origem = '102'`).Scan(&id)
	if s2, _ := repo.GetSolicitacao(ctx, pool, id); s2.Titulo != "Chamado 102 do GLPI" || s2.Estado != domain.EstadoEmAtendimento || s2.Tipo != domain.TipoRequisicao {
		t.Fatalf("chamado 102: %+v", s2)
	} else if !s2.UpdatedAt.Equal(time.Date(2025, 4, 2, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("última atualização do 102 é a do GLPI: %v", s2.UpdatedAt)
	}

	// De novo: nada se repete.
	outra, err := novo(true).Executar(ctx)
	if err != nil || outra.Importados != 0 || outra.JaImportados != 3 || outra.OfertasNovas != 0 {
		t.Fatalf("segunda vez: %+v %v", outra, err)
	}
	if err := cliente.Fechar(ctx); err != nil || g.sessoes != 0 {
		t.Fatalf("fechar a sessão: %v %d", err, g.sessoes)
	}
}

// fonteFalha falha no passo pedido, e no chamado pedido.
type fonteFalha struct {
	Fonte
	passo   string
	chamado int
}

var errFonte = errors.New("fonte fora")

func (f fonteFalha) Categorias(ctx context.Context) ([]Categoria, error) {
	if f.passo == "categorias" {
		return nil, errFonte
	}
	return f.Fonte.Categorias(ctx)
}
func (f fonteFalha) Chamados(ctx context.Context, de, ate int) ([]Chamado, int, error) {
	if f.passo == "chamados" {
		return nil, 0, errFonte
	}
	return f.Fonte.Chamados(ctx, de, ate)
}
func (f fonteFalha) Participantes(ctx context.Context, id int) ([]Participante, error) {
	if f.passo == "participantes" && id == f.chamado {
		return nil, errFonte
	}
	return f.Fonte.Participantes(ctx, id)
}
func (f fonteFalha) Acompanhamentos(ctx context.Context, id int) ([]Acompanhamento, error) {
	if f.passo == "acompanhamentos" && id == f.chamado {
		return nil, errFonte
	}
	return f.Fonte.Acompanhamentos(ctx, id)
}
func (f fonteFalha) Satisfacao(ctx context.Context, id int) (*Satisfacao, error) {
	if f.passo == "satisfacao" && id == f.chamado {
		return nil, errFonte
	}
	return f.Fonte.Satisfacao(ctx, id)
}
func (f fonteFalha) Pessoa(ctx context.Context, id int) (Pessoa, error) {
	if f.passo == "pessoa" {
		return Pessoa{}, errFonte
	}
	return f.Fonte.Pessoa(ctx, id)
}

// insertFalha recusa gravar a solicitação com o título pedido.
type insertFalha struct {
	domain.Repository
	titulo string
}

func (r insertFalha) InsertSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) (domain.Solicitacao, error) {
	if s.Titulo == r.titulo {
		return domain.Solicitacao{}, errFonte
	}
	return r.Repository.InsertSolicitacao(ctx, db, s)
}

func TestImportacaoFalhas(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	u := dbtest.Unidade(t, pool)
	g := &glpiFalso{
		categorias: []Categoria{},
		chamados:   []Chamado{{ID: 201, Titulo: "A", Status: 1, Tipo: 1}, {ID: 202, Titulo: "B", Status: 1, Tipo: 1}},
		partes:     map[int][]Participante{201: {{PessoaID: 9, Papel: 1}}},
		acomps:     map[int][]Acompanhamento{},
		pessoas:    map[int]Pessoa{9: {ID: 9, Login: "x"}},
	}
	srv := g.servidor(t)
	cliente := NovoCliente(srv.URL+"/apirest.php", "a1", "u1")
	if err := cliente.Abrir(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	})
	_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	novo := func(f Fonte) *Importador {
		return &Importador{Fonte: f, Pool: pool, Repo: infrastructure.NewRepository(), Unidade: &u}
	}
	for _, passo := range []string{"categorias", "chamados"} {
		if _, err := novo(fonteFalha{Fonte: cliente, passo: passo}).Executar(ctx); !errors.Is(err, errFonte) {
			t.Errorf("%s: %v", passo, err)
		}
	}
	// O erro de um chamado não para os outros.
	for _, passo := range []string{"participantes", "acompanhamentos", "satisfacao", "pessoa"} {
		rel, err := novo(fonteFalha{Fonte: cliente, passo: passo, chamado: 201}).Executar(ctx)
		if err != nil || len(rel.Erros) != 1 || !strings.Contains(rel.Erros[0], "chamado 201") || rel.Importados != 1 {
			t.Errorf("%s: %+v %v", passo, rel, err)
		}
	}

	// Uma falha no banco desfaz o chamado inteiro, a oferta nova junto, e o
	// próximo chamado cria a oferta de novo.
	g.chamados = []Chamado{{ID: 301, Titulo: strings.Repeat("x", 10), Status: 1, Tipo: 2, CategoriaID: 5}, {ID: 302, Titulo: "ok", Status: 1, Tipo: 2, CategoriaID: 5}}
	g.partes = map[int][]Participante{}
	quebrado := &Importador{Fonte: cliente, Pool: pool, Repo: insertFalha{Repository: infrastructure.NewRepository(), titulo: strings.Repeat("x", 10)}, Unidade: &u, Gravar: true}
	rel, err := quebrado.Executar(ctx)
	if err != nil || rel.Importados != 1 || len(rel.Erros) != 1 {
		t.Fatalf("falha no banco: %+v %v", rel, err)
	}
	var ofertas int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_ofertas WHERE unidade_id = $1 AND tipo = 'requisicao'`, u).Scan(&ofertas)
	if ofertas != 1 {
		t.Fatalf("uma oferta só, criada pelo chamado que entrou: %d", ofertas)
	}

	// A sessão recusada e o caminho quebrado.
	if err := NovoCliente(srv.URL+"/apirest.php", "errado", "u1").Abrir(ctx); !errors.Is(err, ErrGLPI) {
		t.Fatalf("token errado: %v", err)
	}
	g.quebrado = "/Ticket/202/Ticket_User"
	if _, err := cliente.Participantes(ctx, 202); !errors.Is(err, ErrGLPI) {
		t.Fatalf("500: %v", err)
	}
}
