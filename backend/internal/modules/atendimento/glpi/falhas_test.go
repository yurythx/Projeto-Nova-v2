package glpi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// gravacaoFalha falha no método de gravação pedido.
type gravacaoFalha struct {
	domain.Repository
	metodo string
}

func (r gravacaoFalha) InsertOferta(ctx context.Context, db database.DBTX, o domain.Oferta) (domain.Oferta, error) {
	if r.metodo == "InsertOferta" {
		return domain.Oferta{}, errFonte
	}
	return r.Repository.InsertOferta(ctx, db, o)
}
func (r gravacaoFalha) UpdateSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) error {
	if r.metodo == "UpdateSolicitacao" {
		return errFonte
	}
	return r.Repository.UpdateSolicitacao(ctx, db, s)
}
func (r gravacaoFalha) InsertPassagem(ctx context.Context, db database.DBTX, p domain.Passagem) (domain.Passagem, error) {
	if r.metodo == "InsertPassagem" {
		return domain.Passagem{}, errFonte
	}
	return r.Repository.InsertPassagem(ctx, db, p)
}
func (r gravacaoFalha) InsertComentario(ctx context.Context, db database.DBTX, c domain.Comentario) (domain.Comentario, error) {
	if r.metodo == "InsertComentario" {
		return domain.Comentario{}, errFonte
	}
	return r.Repository.InsertComentario(ctx, db, c)
}
func (r gravacaoFalha) InsertAvaliacao(ctx context.Context, db database.DBTX, a domain.Avaliacao) (domain.Avaliacao, error) {
	if r.metodo == "InsertAvaliacao" {
		return domain.Avaliacao{}, errFonte
	}
	return r.Repository.InsertAvaliacao(ctx, db, a)
}

func TestImportacaoFalhasNaGravacao(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	casado := dbtest.User(t, pool)
	var login string
	if err := pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, casado).Scan(&login); err != nil {
		t.Fatal(err)
	}
	quatro := 4
	g := &glpiFalso{
		chamados: []Chamado{{ID: 401, Titulo: "Senha", Status: 5, Tipo: 1, CategoriaID: 3}},
		partes:   map[int][]Participante{401: {{PessoaID: 0, Papel: 1}, {PessoaID: 5, Papel: 1}}},
		acomps:   map[int][]Acompanhamento{401: {{ID: 1, Texto: "Feito", AutorID: 5}}},
		satisf:   map[int][]Satisfacao{401: {{Nota: &quatro}}},
		pessoas:  map[int]Pessoa{5: {ID: 5, Login: strings.ToUpper(login)}},
	}
	cliente := NovoCliente(g.servidor(t).URL+"/apirest.php", "a1", "u1")
	if err := cliente.Abrir(ctx); err != nil {
		t.Fatal(err)
	}
	limpar := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	}
	limpar()
	t.Cleanup(limpar)
	for _, m := range []string{"InsertOferta", "UpdateSolicitacao", "InsertPassagem", "InsertComentario", "InsertAvaliacao"} {
		im := &Importador{Fonte: cliente, Pool: pool, Repo: gravacaoFalha{Repository: infrastructure.NewRepository(), metodo: m}, Unidade: &u, Gravar: true}
		rel, err := im.Executar(ctx)
		if err != nil || len(rel.Erros) != 1 || rel.Importados != 0 {
			t.Errorf("%s: %+v %v", m, rel, err)
		}
	}
	// Sem falha: o solicitante casa pelo login (sem diferença de caixa) e
	// o id zero do GLPI é ignorado.
	rel, err := (&Importador{Fonte: cliente, Pool: pool, Repo: infrastructure.NewRepository(), Unidade: &u, Gravar: true}).Executar(ctx)
	if err != nil || rel.Importados != 1 || len(rel.SemPar) != 0 {
		t.Fatalf("importa: %+v %v", rel, err)
	}
	var solicitante uuid.UUID
	_ = pool.QueryRow(ctx, `SELECT solicitante_id FROM atendimento_solicitacoes WHERE unidade_id = $1`, u).Scan(&solicitante)
	if solicitante != casado {
		t.Fatalf("casou pelo login: %v", solicitante)
	}

	// O autor do acompanhamento que a fonte não entrega.
	g.chamados = []Chamado{{ID: 402, Titulo: "Rede", Status: 1, Tipo: 1}}
	g.acomps = map[int][]Acompanhamento{402: {{ID: 2, Texto: "x", AutorID: 6}}}
	rel, err = (&Importador{Fonte: fonteFalha{Fonte: cliente, passo: "pessoa"}, Pool: pool, Repo: infrastructure.NewRepository()}).Executar(ctx)
	if err != nil || len(rel.Erros) != 1 {
		t.Fatalf("autor: %+v %v", rel, err)
	}

	// Banco fora: as leituras de antes e a busca da pessoa.
	url := os.Getenv("TEST_DATABASE_URL")
	fechado, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	fechado.Close()
	if _, err := (&Importador{Fonte: cliente, Pool: fechado, Repo: infrastructure.NewRepository()}).Executar(ctx); err == nil {
		t.Fatal("banco fechado")
	}
	im := &Importador{Fonte: cliente, Pool: fechado, pessoas: map[int]*pessoaCasada{}}
	if _, err := im.pessoa(ctx, 5); err == nil {
		t.Fatal("pessoa com o banco fechado")
	}
}

func TestClienteFalhas(t *testing.T) {
	ctx := context.Background()
	resposta := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, ok := resposta[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(corpo))
	}))
	t.Cleanup(srv.Close)
	c := NovoCliente(srv.URL, "", "u")

	resposta["/initSession"] = `{}`
	if err := c.Abrir(ctx); !errors.Is(err, ErrGLPI) {
		t.Fatalf("sessão vazia: %v", err)
	}
	resposta["/initSession"] = `{"session_token":"s"}`
	resposta["/getMyProfiles"] = `{"myprofiles":[]}`
	if err := c.Abrir(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Chamados(ctx, 0, 9); !errors.Is(err, ErrGLPI) {
		t.Fatalf("chamados: %v", err)
	}
	if _, err := c.Satisfacao(ctx, 1); !errors.Is(err, ErrGLPI) {
		t.Fatalf("satisfação: %v", err)
	}
	resposta["/Ticket/1/TicketSatisfaction"] = `[{"satisfaction":null}]`
	if s, err := c.Satisfacao(ctx, 1); err != nil || s != nil {
		t.Fatalf("sem nota respondida: %+v %v", s, err)
	}
	if _, err := c.Pessoa(ctx, 1); !errors.Is(err, ErrGLPI) {
		t.Fatalf("pessoa: %v", err)
	}
	resposta["/User/1"] = `{"id":1,"name":"ana"}`
	if _, err := c.Pessoa(ctx, 1); !errors.Is(err, ErrGLPI) {
		t.Fatalf("e-mails: %v", err)
	}
	resposta["/User/1/UserEmail"] = `{`
	if _, err := c.Pessoa(ctx, 1); !errors.Is(err, ErrGLPI) {
		t.Fatalf("JSON inválido: %v", err)
	}
	if p := (Pessoa{Login: "ana"}); p.NomeCompleto() != "ana" {
		t.Fatal("sem nome, o login")
	}
	resposta["/Ticket"] = `[{"id":1}]`
	if lista, total, err := c.Chamados(ctx, 0, 9); err != nil || len(lista) != 1 || total != 1 {
		t.Fatalf("sem Content-Range: %d %v", total, err)
	}

	quebrado := NovoCliente("http://[::1]:namedport", "", "u")
	if err := quebrado.Abrir(ctx); err == nil {
		t.Fatal("URL inválida")
	}
	fora := NovoCliente(srv.URL, "", "u")
	srv.Close()
	if err := fora.Abrir(ctx); err == nil {
		t.Fatal("servidor fora")
	}
}
