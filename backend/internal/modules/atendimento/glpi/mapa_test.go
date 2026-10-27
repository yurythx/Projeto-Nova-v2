package glpi

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

const cabecalho = "glpi_secretaria;glpi_categoria_id;glpi_categoria;chamados;unidade;oferta;tipo;prioridade\n"

func TestLerMapa(t *testing.T) {
	u := uuid.New()
	unidades := map[string]uuid.UUID{"TI da Prefeitura": u}
	m, err := LerMapa(strings.NewReader(bom+cabecalho+
		"PREFEITURA;99;Computadores;10;ti da prefeitura;Computadores;requisicao;alta\n"+
		"sms;*;;3;TI da Saúde;Atendimento geral;incidente;media\n"), unidades)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := m.Destino("Prefeitura", 99); !ok || d.Unidade == nil || *d.Unidade != u || d.Prioridade != "alta" {
		t.Fatalf("linha exata, sem caixa: %+v %v", d, ok)
	}
	if d, ok := m.Destino("SMS", 7); !ok || d.Oferta != "Atendimento geral" || d.Unidade != nil {
		t.Fatalf("o * da secretaria: %+v %v", d, ok)
	}
	if _, ok := m.Destino("SEMECEL", 1); ok {
		t.Fatal("secretaria fora da tabela")
	}
	if len(m.Faltam) != 1 || m.Faltam[0] != "TI da Saúde" {
		t.Fatalf("unidade que falta: %v", m.Faltam)
	}
	for nome, csv := range map[string]string{
		"vazia":              cabecalho,
		"sem coluna":         "glpi_secretaria;oferta\nX;Y\n",
		"categoria ruim":     cabecalho + "P;x;;;U;O;incidente;alta\n",
		"categoria negativa": cabecalho + "P;-2;;;U;O;incidente;alta\n",
		"sem oferta":         cabecalho + "P;1;;;U;;incidente;alta\n",
		"tipo ruim":          cabecalho + "P;1;;;U;O;problema;alta\n",
		"prioridade ruim":    cabecalho + "P;1;;;U;O;incidente;urgente\n",
		"repetida":           cabecalho + "P;1;;;U;O;incidente;alta\np;1;;;U;O;incidente;alta\n",
		"aspas quebradas":    cabecalho + "P;1;\"x;;U;O;incidente;alta\n",
	} {
		if _, err := LerMapa(strings.NewReader(csv), unidades); err == nil {
			t.Errorf("%s: aceitou", nome)
		}
	}
}

func TestSecretariaEMeta(t *testing.T) {
	for completo, quer := range map[string]string{"RONDONOPOLIS > SMS > UPA": "SMS", "RONDONOPOLIS": "RONDONOPOLIS", "": ""} {
		if s := Secretaria(completo); s != quer {
			t.Errorf("%q: %q", completo, s)
		}
	}
	for _, c := range []struct {
		n       int
		unidade string
		min     int
		ok      bool
	}{{30, "minute", 30, true}, {4, "hour", 240, true}, {2, "day", 2880, true}, {0, "hour", 0, false}, {3, "week", 0, false}} {
		if m, ok := MetaMin(c.n, c.unidade); m != c.min || ok != c.ok {
			t.Errorf("%d %s: %d %v", c.n, c.unidade, m, ok)
		}
	}
}

func TestImportacaoComMapa(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	var nome string
	if err := pool.QueryRow(ctx, `SELECT nome FROM unidades WHERE id = $1`, u).Scan(&nome); err != nil {
		t.Fatal(err)
	}
	repo := infrastructure.NewRepository()
	// Uma oferta que já existe com o nome da tabela é reaproveitada.
	geral, err := domain.NovaOferta("Atendimento geral", "", domain.TipoIncidente, nil, &u, true)
	if err != nil {
		t.Fatal(err)
	}
	if geral, err = repo.InsertOferta(ctx, pool, geral); err != nil {
		t.Fatal(err)
	}
	g := &glpiFalso{
		categorias: []Categoria{{ID: 7, Nome: "Computadores"}, {ID: 8, Nome: "Periféricos"}},
		entidades:  []Entidade{{ID: 1, Nome: "RAIZ > PREFEITURA"}, {ID: 4, Nome: "RAIZ > SMS > UPA"}, {ID: 5, Nome: "RAIZ > SEMECEL"}},
		slas:       []SLA{{ID: 1, Tempo: 30, Unidade: "minute"}, {ID: 2, Tempo: 4, Unidade: "hour"}, {ID: 3, Tempo: 1, Unidade: "semana"}},
		chamados: []Chamado{
			{ID: 401, Titulo: "PC não liga", Abertura: "2025-05-05 08:00:00", Fechamento: "2025-05-05 10:00:00", Status: 6, Tipo: 2,
				CategoriaID: 7, EntidadeID: 1, SLASolucao: 2, SLAResposta: 1},
			{ID: 402, Titulo: "Mouse", Abertura: "2025-05-06 08:00:00", Status: 1, Tipo: 2, CategoriaID: 8, EntidadeID: 1, SLASolucao: 3},
			{ID: 403, Titulo: "Sistema lento", Abertura: "2025-05-07 08:00:00", Solucao: "2025-05-07 09:00:00", Status: 5, Tipo: 2, EntidadeID: 4},
			{ID: 404, Titulo: "Fora da tabela", Status: 1, Tipo: 1, CategoriaID: 7, EntidadeID: 5},
			{ID: 405, Titulo: "Também fora", Status: 1, Tipo: 1, CategoriaID: 7, EntidadeID: 5},
		},
		partes: map[int][]Participante{}, acomps: map[int][]Acompanhamento{},
	}
	srv := g.servidor(t)
	cliente := NovoCliente(srv.URL+"/apirest.php", "a1", "u1")
	if err := cliente.Abrir(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_regras WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	})
	_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	// Computadores e Periféricos viram a mesma oferta; a SMS inteira vai
	// para o atendimento geral, como incidente.
	tabela := cabecalho +
		"PREFEITURA;7;;;" + nome + ";Computadores;requisicao;alta\n" +
		"PREFEITURA;8;;;" + nome + ";Computadores;requisicao;alta\n" +
		"SMS;*;;;" + nome + ";Atendimento geral;incidente;media\n"
	mapa := func(extra string) *Mapa {
		m, err := LerMapa(strings.NewReader(tabela+extra), map[string]uuid.UUID{nome: u})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	novo := func(m *Mapa, gravar bool) *Importador {
		return &Importador{Fonte: cliente, Pool: pool, Repo: repo, Gravar: gravar, Mapa: m}
	}

	// Simular com uma unidade que falta só avisa; gravar recusa.
	faltando := mapa("SEMECEL;*;;;TI da SEMECEL;Geral;incidente;baixa\n")
	sim, err := novo(faltando, false).Executar(ctx)
	if err != nil || sim.Importados != 5 || len(sim.UnidadesFaltando) != 1 || sim.ComPrazo != 1 {
		t.Fatalf("simulação: %+v %v", sim, err)
	}
	if _, err := novo(faltando, true).Executar(ctx); err == nil || !strings.Contains(err.Error(), "TI da SEMECEL") {
		t.Fatalf("gravar sem a unidade: %v", err)
	}

	rel, err := novo(mapa(""), true).Executar(ctx)
	if err != nil || rel.Importados != 3 || rel.SemDestino["SEMECEL/7"] != 2 || rel.ComPrazo != 1 || len(rel.Erros) != 0 {
		t.Fatalf("gravar: %+v %v", rel, err)
	}
	sol := func(glpi string) domain.Solicitacao {
		t.Helper()
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT destino FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao' AND id_origem = $1`, glpi).Scan(&id); err != nil {
			t.Fatal(err)
		}
		s, err := repo.GetSolicitacao(ctx, pool, id)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s401, s402, s403 := sol("401"), sol("402"), sol("403")
	if s401.UnidadeID == nil || *s401.UnidadeID != u || s401.Prioridade == nil || *s401.Prioridade != "alta" ||
		s401.MetaRespostaMin == nil || *s401.MetaRespostaMin != 30 || s401.MetaSolucaoMin == nil || *s401.MetaSolucaoMin != 240 {
		t.Fatalf("401 com o SLA do GLPI: %+v", s401)
	}
	if s402.OfertaID != s401.OfertaID || *s402.Prioridade != "alta" || s402.MetaSolucaoMin != nil {
		t.Fatalf("402 junto, sem prazo (SLA ilegível e sem resposta): %+v", s402)
	}
	if s403.OfertaID != geral.ID || s403.Tipo != domain.TipoIncidente || s403.Estado != domain.EstadoResolvido || *s403.Prioridade != "media" {
		t.Fatalf("403 no atendimento geral, como incidente: %+v", s403)
	}
	o, _ := repo.GetOferta(ctx, pool, s401.OfertaID)
	if !o.Ativa || o.Nome != "Computadores" || o.Tipo != domain.TipoRequisicao {
		t.Fatalf("oferta da tabela, ativa: %+v", o)
	}
	regras, _ := repo.ListarRegrasDaUnidade(ctx, pool, &u)
	if len(regras) != 1 || regras[0].OfertaID == nil || *regras[0].OfertaID != o.ID || regras[0].Prioridade != "alta" {
		t.Fatalf("a regra da oferta nova (a que já existia fica sem): %+v", regras)
	}

	// Sem Estrutura na fonte, ou com ela quebrada, o mapa não roda.
	if _, err := (&Importador{Fonte: fonteFalha{Fonte: cliente}, Pool: pool, Repo: repo, Mapa: mapa("")}).Executar(ctx); err == nil {
		t.Fatal("fonte sem entidades")
	}
	for _, caminho := range []string{"/Entity", "/SLA"} {
		g.quebrado = caminho
		if _, err := novo(mapa(""), false).Executar(ctx); err == nil {
			t.Errorf("%s quebrado", caminho)
		}
	}
}

// regraCheia é a unidade que já tem as 30 regras da fila.
type regraCheia struct{ domain.Repository }

func (regraCheia) ContarRegras(context.Context, database.DBTX, *uuid.UUID) (int, error) {
	return TetoRegras, nil
}

func TestRegraRespeitaOTeto(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	im := &Importador{Repo: regraCheia{infrastructure.NewRepository()}}
	if err := im.regra(ctx, tx, Destino{Unidade: &u, Oferta: "Nova", Prioridade: "alta"}, uuid.New()); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM atendimento_regras WHERE unidade_id = $1`, u).Scan(&n)
	if n != 0 {
		t.Fatalf("sem vaga, a oferta fica sem regra: %d", n)
	}
	if err := im.regra(ctx, tx, Destino{Unidade: &u, Oferta: "Nova", Prioridade: "urgente"}, uuid.New()); err != nil {
		t.Fatal("o teto vem antes da validação")
	}
	im.Repo = infrastructure.NewRepository()
	if err := im.regra(ctx, tx, Destino{Unidade: &u, Oferta: "Nova", Prioridade: "urgente"}, uuid.New()); err == nil {
		t.Fatal("prioridade inválida")
	}
}
