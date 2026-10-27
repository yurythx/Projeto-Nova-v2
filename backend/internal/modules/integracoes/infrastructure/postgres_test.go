package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func ptr(n int) *int { return &n }

func novaConexao(t *testing.T, db database.DBTX, modo string) domain.Conexao {
	t.Helper()
	c, err := domain.NovaConexao("glpi", "GLPI "+uuid.NewString()[:6], modo, "http://glpi/apirest.php", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if c, err = NewRepository().InsertConexao(context.Background(), db, c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConexoes(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	r := NewRepository()
	c := novaConexao(t, pool, domain.ModoDireto)
	t.Cleanup(func() { _ = r.DeleteConexao(context.Background(), pool, c.ID) })

	c.Nome, c.Perfil = "Renomeada", "Super-Admin"
	if err := r.UpdateConexao(ctx, pool, c); err != nil {
		t.Fatal(err)
	}
	if err := r.SetCredenciais(ctx, pool, c.ID, "cifrado"); err != nil {
		t.Fatal(err)
	}
	hash := "h-" + uuid.NewString()
	if err := r.SetToken(ctx, pool, c.ID, &hash); err != nil {
		t.Fatal(err)
	}
	quando := time.Now().UTC().Truncate(time.Second)
	if err := r.MarcarExtracao(ctx, pool, c.ID, quando); err != nil {
		t.Fatal(err)
	}
	g, err := r.GetConexao(ctx, pool, c.ID)
	if err != nil || g.Nome != "Renomeada" || !g.TemCredenciais || g.Cifradas != "cifrado" || !g.TemToken || g.TokenEm == nil ||
		g.ExtraidoEm == nil || !g.ExtraidoEm.Equal(quando) {
		t.Fatalf("conexão: %+v %v", g, err)
	}
	if p, err := r.PorToken(ctx, pool, hash); err != nil || p.ID != c.ID {
		t.Fatalf("pelo token: %+v %v", p, err)
	}
	if err := r.SetToken(ctx, pool, c.ID, nil); err != nil {
		t.Fatal(err)
	}
	if g, _ := r.GetConexao(ctx, pool, c.ID); g.TemToken || g.TokenEm != nil {
		t.Fatalf("token revogado: %+v", g)
	}
	if _, err := r.PorToken(ctx, pool, hash); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("token velho: %v", err)
	}
	lista, err := r.ListConexoes(ctx, pool)
	if err != nil || len(lista) == 0 {
		t.Fatalf("lista: %v", err)
	}
	nada := uuid.New()
	for nome, err := range map[string]error{
		"get":         func() error { _, err := r.GetConexao(ctx, pool, nada); return err }(),
		"update":      r.UpdateConexao(ctx, pool, domain.Conexao{ID: nada, Nome: "x"}),
		"delete":      r.DeleteConexao(ctx, pool, nada),
		"credenciais": r.SetCredenciais(ctx, pool, nada, "x"),
		"token":       r.SetToken(ctx, pool, nada, nil),
		"extração":    r.MarcarExtracao(ctx, pool, nada, quando),
	} {
		if !errors.Is(err, domain.ErrNaoEncontrado) {
			t.Errorf("%s inexistente: %v", nome, err)
		}
	}
	// O banco recusa o que o domínio deixaria passar errado.
	ruim := c
	ruim.ID, ruim.Tipo = uuid.New(), "otrs"
	if _, err := r.InsertConexao(ctx, pool, ruim); !errors.As(err, new(domain.Recusa)) {
		t.Fatalf("check: %v", err)
	}
}

func TestAreaDePreparoETabela(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	r := NewRepository()
	c := novaConexao(t, pool, domain.ModoConector)
	t.Cleanup(func() { _ = r.DeleteConexao(context.Background(), pool, c.ID) })
	dados := json.RawMessage(`{"x":1}`)
	itens := []domain.Item{
		{Tipo: domain.ItemEntidade, ID: 0, Nome: "RAIZ", Dados: dados},
		{Tipo: domain.ItemEntidade, ID: 1, Nome: "RAIZ > SMS", Dados: dados},
		{Tipo: domain.ItemCategoria, ID: 7, Nome: "Computadores", Dados: dados},
		{Tipo: domain.ItemSLA, ID: 2, Nome: "Solução N1", Minutos: ptr(240), Dados: dados},
		{Tipo: domain.ItemSLA, ID: 3, Nome: "Sem tempo", Dados: dados},
		{Tipo: domain.ItemPessoa, ID: 9, Nome: "Ana", Dados: dados},
		{Tipo: domain.ItemChamado, ID: 101, Versao: "v1", Entidade: ptr(1), Categoria: ptr(7), SLA: ptr(2), Classe: "incidente", Data: "2025-03-10 09:00:00", Dados: dados},
		{Tipo: domain.ItemChamado, ID: 102, Versao: "v1", Entidade: ptr(1), Categoria: ptr(7), Classe: "incidente", Data: "2025-04-10 09:00:00", Dados: dados},
		{Tipo: domain.ItemChamado, ID: 103, Versao: "v1", Entidade: ptr(1), Dados: dados},
	}
	if err := r.GravarItens(ctx, pool, c.ID, itens); err != nil {
		t.Fatal(err)
	}
	// De novo, com outra versão: atualiza, não duplica.
	itens[6].Versao = "v2"
	if err := r.GravarItens(ctx, pool, c.ID, itens[6:7]); err != nil {
		t.Fatal(err)
	}
	if v, err := r.Versoes(ctx, pool, c.ID); err != nil || len(v) != 3 || v[101] != "v2" {
		t.Fatalf("versões: %v %v", v, err)
	}
	if ids, err := r.IDs(ctx, pool, c.ID, domain.ItemPessoa); err != nil || !ids[9] || len(ids) != 1 {
		t.Fatalf("pessoas: %v %v", ids, err)
	}
	pag, total, err := r.Itens(ctx, pool, c.ID, domain.ItemChamado, 1, 1)
	if err != nil || total != 3 || len(pag) != 1 || pag[0].ID != 102 || string(pag[0].Dados) != `{"x": 1}` {
		t.Fatalf("página: %+v %d %v", pag, total, err)
	}
	if it, ok, err := r.Item(ctx, pool, c.ID, domain.ItemPessoa, 9); err != nil || !ok || it.Nome != "Ana" {
		t.Fatalf("item: %+v %v %v", it, ok, err)
	}
	if _, ok, err := r.Item(ctx, pool, c.ID, domain.ItemPessoa, 404); err != nil || ok {
		t.Fatalf("item que não há: %v %v", ok, err)
	}
	if n, err := r.Contagens(ctx, pool, c.ID); err != nil || n[domain.ItemChamado] != 3 || n[domain.ItemSLA] != 2 {
		t.Fatalf("contagens: %v %v", n, err)
	}
	grupos, err := r.Grupos(ctx, pool, c.ID)
	if err != nil || len(grupos) != 2 {
		t.Fatalf("grupos: %+v %v", grupos, err)
	}
	for _, g := range grupos {
		if g.Categoria == 7 && (g.Chamados != 2 || g.ComPrazo != 1 || g.SLA == nil || *g.SLA != 2 || g.Classe != "incidente" ||
			g.Primeiro != "2025-03-10" || g.Ultimo != "2025-04-10") {
			t.Fatalf("grupo 7: %+v", g)
		}
		if g.Categoria == 0 && (g.SLA != nil || g.Classe != "" || g.Primeiro != "") {
			t.Fatalf("sem categoria: %+v", g)
		}
	}
	if nomes, err := r.Nomes(ctx, pool, c.ID, domain.ItemEntidade); err != nil || nomes[1] != "RAIZ > SMS" || nomes[0] != "RAIZ" {
		t.Fatalf("nomes: %v %v", nomes, err)
	}
	if slas, err := r.SLAs(ctx, pool, c.ID); err != nil || len(slas) != 2 || slas[0].Minutos != 240 || slas[1].Minutos != 0 {
		t.Fatalf("SLAs: %+v %v", slas, err)
	}

	u := dbtest.Unidade(t, pool)
	tabela := []domain.Linha{
		{Secretaria: "SMS", Categoria: domain.CategoriaQualquer, Oferta: "Geral", Tipo: "requisicao", Prioridade: "media"},
		{Secretaria: "SMS", Categoria: 7, CategoriaNome: "Computadores", Chamados: 2, ComPrazo: 1, Nivel: "Solução N1", UnidadeID: &u,
			Oferta: "Computadores", Tipo: "incidente", Prioridade: "alta"},
	}
	if err := r.TrocarTabela(ctx, pool, c.ID, tabela); err != nil {
		t.Fatal(err)
	}
	lidas, err := r.ListarTabela(ctx, pool, c.ID)
	if err != nil || len(lidas) != 2 || lidas[0].Categoria != 7 || lidas[0].UnidadeNome == "" || lidas[1].UnidadeID != nil {
		t.Fatalf("tabela (o * por último): %+v %v", lidas, err)
	}
	nada := uuid.New()
	tabela[1].UnidadeID = &nada
	if err := r.TrocarTabela(ctx, pool, c.ID, tabela); !errors.As(err, new(domain.Recusa)) {
		t.Fatalf("unidade que não existe: %v", err)
	}
}

func TestTarefas(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	r := NewRepository()
	direto := novaConexao(t, pool, domain.ModoDireto)
	conector := novaConexao(t, pool, domain.ModoConector)
	t.Cleanup(func() {
		_ = r.DeleteConexao(context.Background(), pool, direto.ID)
		_ = r.DeleteConexao(context.Background(), pool, conector.ID)
	})
	nova := func(c uuid.UUID, tipo string) domain.Tarefa {
		tf, err := domain.NovaTarefa(c, tipo, "ana")
		if err != nil {
			t.Fatal(err)
		}
		if err := r.InserirTarefa(ctx, pool, tf); err != nil {
			t.Fatal(err)
		}
		return tf
	}
	ext := nova(conector.ID, domain.TarefaExtracao)
	if err := r.InserirTarefa(ctx, pool, ext); !errors.Is(err, domain.ErrTarefaAtiva) {
		t.Fatalf("uma por conexão: %v", err)
	}
	// O worker não pega a extração do conector; o conector, sim.
	if _, ok, err := r.Pegar(ctx, pool, nil, []string{domain.TarefaExtracao}, domain.ModoDireto); err != nil || ok {
		t.Fatalf("worker: %v %v", ok, err)
	}
	pega, ok, err := r.Pegar(ctx, pool, &conector.ID, []string{domain.TarefaExtracao}, domain.ModoConector)
	if err != nil || !ok || pega.ID != ext.ID || pega.Estado != domain.EstadoEmAndamento || pega.IniciadaEm == nil {
		t.Fatalf("conector: %+v %v %v", pega, ok, err)
	}
	if estado, err := r.Progresso(ctx, pool, ext.ID, 5, 10); err != nil || estado != domain.EstadoEmAndamento {
		t.Fatalf("progresso: %q %v", estado, err)
	}
	if err := r.Terminar(ctx, pool, ext.ID, domain.EstadoConcluida, []byte(`{"chamados":10}`), ""); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetTarefa(ctx, pool, ext.ID); err != nil || got.Estado != domain.EstadoConcluida || got.Feitos != 5 ||
		string(got.Relatorio) != `{"chamados": 10}` || got.TerminadaEm == nil {
		t.Fatalf("terminada: %+v %v", got, err)
	}
	// Terminada não muda mais.
	if err := r.Terminar(ctx, pool, ext.ID, domain.EstadoFalhou, nil, "x"); err != nil {
		t.Fatal(err)
	}
	if err := r.Cancelar(ctx, pool, ext.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("cancelar a terminada: %v", err)
	}

	carga := nova(direto.ID, domain.TarefaCarga)
	if _, ok, err := r.Pegar(ctx, pool, nil, []string{domain.TarefaSimulacao, domain.TarefaCarga}, ""); err != nil || !ok {
		t.Fatalf("worker pega a carga: %v %v", ok, err)
	}
	// Sem sinal de vida desde antes de agora + 1 min: volta a pendente.
	if n, err := r.Retomar(ctx, pool, time.Now().Add(time.Minute)); err != nil || n < 1 {
		t.Fatalf("retomar: %d %v", n, err)
	}
	if got, _ := r.GetTarefa(ctx, pool, carga.ID); got.Estado != domain.EstadoPendente {
		t.Fatalf("retomada: %+v", got)
	}
	if err := r.Cancelar(ctx, pool, carga.ID); err != nil {
		t.Fatal(err)
	}
	if estado, err := r.Progresso(ctx, pool, carga.ID, 1, 1); err != nil || estado != domain.EstadoCancelada {
		t.Fatalf("o progresso diz que foi cancelada: %q %v", estado, err)
	}
	if lista, err := r.ListarTarefas(ctx, pool, direto.ID, 10); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	if _, err := r.GetTarefa(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("tarefa que não há: %v", err)
	}
	if _, err := r.Progresso(ctx, pool, uuid.New(), 1, 1); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("progresso de tarefa que não há: %v", err)
	}
}

// contaOK responde a contagem e falha na consulta (o caminho do Itens).
type contaOK struct{ dbtest.Fail }

type linhaOK struct{}

func (linhaOK) Scan(...any) error { return nil }

func (contaOK) QueryRow(context.Context, string, ...any) pgx.Row { return linhaOK{} }

type contaOKScan struct{ dbtest.ScanFail }

func (contaOKScan) QueryRow(context.Context, string, ...any) pgx.Row { return linhaOK{} }

type contaOKRows struct{ dbtest.RowsErr }

func (contaOKRows) QueryRow(context.Context, string, ...any) pgx.Row { return linhaOK{} }

// execOK aceita o primeiro Exec (o DELETE da tabela) e falha no INSERT.
type execOK struct {
	dbtest.Fail
	n *int
}

func (e execOK) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	*e.n++
	if *e.n == 1 {
		return pgconn.CommandTag{}, nil
	}
	return pgconn.CommandTag{}, dbtest.ErrInjected
}

func TestFalhasDoBanco(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	id := uuid.New()
	um := []domain.Item{{Tipo: domain.ItemPessoa, ID: 1, Dados: json.RawMessage(`{}`)}}
	linha := []domain.Linha{{Secretaria: "S", Oferta: "O", Tipo: "incidente", Prioridade: "alta"}}
	for _, db := range []database.DBTX{dbtest.Fail{}, dbtest.ScanFail{}, dbtest.RowsErr{}} {
		chamadas := map[string]error{
			"insert":      func() error { _, err := r.InsertConexao(ctx, db, domain.Conexao{}); return err }(),
			"update":      r.UpdateConexao(ctx, db, domain.Conexao{}),
			"get":         func() error { _, err := r.GetConexao(ctx, db, id); return err }(),
			"list":        func() error { _, err := r.ListConexoes(ctx, db); return err }(),
			"delete":      r.DeleteConexao(ctx, db, id),
			"credenciais": r.SetCredenciais(ctx, db, id, ""),
			"token":       r.SetToken(ctx, db, id, nil),
			"porToken":    func() error { _, err := r.PorToken(ctx, db, ""); return err }(),
			"extracao":    r.MarcarExtracao(ctx, db, id, time.Now()),
			"gravar":      r.GravarItens(ctx, db, id, um),
			"versoes":     func() error { _, err := r.Versoes(ctx, db, id); return err }(),
			"ids":         func() error { _, err := r.IDs(ctx, db, id, ""); return err }(),
			"itens":       func() error { _, _, err := r.Itens(ctx, db, id, "", 0, 1); return err }(),
			"item":        func() error { _, _, err := r.Item(ctx, db, id, "", 1); return err }(),
			"contagens":   func() error { _, err := r.Contagens(ctx, db, id); return err }(),
			"grupos":      func() error { _, err := r.Grupos(ctx, db, id); return err }(),
			"nomes":       func() error { _, err := r.Nomes(ctx, db, id, ""); return err }(),
			"slas":        func() error { _, err := r.SLAs(ctx, db, id); return err }(),
			"listarTab":   func() error { _, err := r.ListarTabela(ctx, db, id); return err }(),
			"trocarTab":   r.TrocarTabela(ctx, db, id, linha),
			"inserirTar":  r.InserirTarefa(ctx, db, domain.Tarefa{}),
			"getTarefa":   func() error { _, err := r.GetTarefa(ctx, db, id); return err }(),
			"listarTar":   func() error { _, err := r.ListarTarefas(ctx, db, id, 1); return err }(),
			"pegar":       func() error { _, _, err := r.Pegar(ctx, db, nil, nil, ""); return err }(),
			"progresso":   func() error { _, err := r.Progresso(ctx, db, id, 0, 0); return err }(),
			"terminar":    r.Terminar(ctx, db, id, "", nil, ""),
			"cancelar":    r.Cancelar(ctx, db, id),
			"retomar":     func() error { _, err := r.Retomar(ctx, db, time.Now()); return err }(),
		}
		// ScanFail e RowsErr só falham na leitura: o Exec passa.
		soExec := map[string]bool{"inserirTar": true, "trocarTab": true, "terminar": true, "retomar": true, "gravar": true}
		_, falhaTudo := db.(dbtest.Fail)
		for nome, err := range chamadas {
			if err == nil && (falhaTudo || !soExec[nome]) {
				t.Errorf("%T %s: sem erro", db, nome)
			}
		}
	}
	// Itens: a contagem passa e a consulta falha, ou a leitura das linhas.
	for _, db := range []database.DBTX{contaOK{}, contaOKScan{}, contaOKRows{}} {
		if _, _, err := r.Itens(ctx, db, id, "", 0, 1); err == nil {
			t.Errorf("%T: itens sem erro", db)
		}
	}
	n := 0
	if err := r.TrocarTabela(ctx, execOK{n: &n}, id, linha); err == nil {
		t.Fatal("insert da tabela")
	}
	if !errors.Is(mapDB(dbtest.ErrInjected), dbtest.ErrInjected) {
		t.Fatal("o erro desconhecido segue embrulhado")
	}
}
