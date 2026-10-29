package glpi

import (
	"context"
	"encoding/json"
	"errors"
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

// Os níveis do GLPI da prefeitura: N1 30 min/4 h, N2 2 h/24 h, N3 2 h/36 h,
// todos no calendário 3.
var slasDaPrefeitura = []SLA{
	{ID: 1, Nome: "Atendimento N1", Tempo: 30, Unidade: "minute", Calendario: 3, SLM: 1, Tipo: 1},
	{ID: 2, Nome: "Solução N1", Tempo: 4, Unidade: "hour", Calendario: 3, SLM: 1},
	{ID: 3, Nome: "Atendimento N2", Tempo: 2, Unidade: "hour", Calendario: 3, SLM: 2, Tipo: 1},
	{ID: 4, Nome: "Solução N2", Tempo: 24, Unidade: "hour", Calendario: 3, SLM: 2},
	{ID: 5, Nome: "Atendimento N3", Tempo: 2, Unidade: "hour", Calendario: 3, SLM: 3, Tipo: 1},
	{ID: 6, Nome: "Solução N3", Tempo: 36, Unidade: "hour", Calendario: 3, SLM: 3},
}

func TestRegrasDoExpediente(t *testing.T) {
	m := matrizUsual()
	if len(m) != 9 || m[8].Prioridade != domain.PrioridadeCritica || m[0].Prioridade != domain.PrioridadeBaixa || m[4].Prioridade != domain.PrioridadeMedia {
		t.Fatalf("matriz: %+v", m)
	}
	metas := metasDosSLAs(slasDaPrefeitura)
	quer := []domain.Meta{
		{Prioridade: "critica", RespostaMin: 15, SolucaoMin: 120},
		{Prioridade: "alta", RespostaMin: 30, SolucaoMin: 240},
		{Prioridade: "media", RespostaMin: 120, SolucaoMin: 1440},
		{Prioridade: "baixa", RespostaMin: 120, SolucaoMin: 2160},
	}
	if len(metas) != 4 {
		t.Fatalf("metas: %+v", metas)
	}
	for i := range quer {
		if metas[i] != quer[i] {
			t.Errorf("meta %d: %+v, quer %+v", i, metas[i], quer[i])
		}
	}
	// Um par só vale para as três; resposta maior que a solução encolhe;
	// sem solução ou sem SLM, nada.
	um := metasDosSLAs([]SLA{{Tempo: 8, Unidade: "hour", SLM: 9}, {Tempo: 10, Unidade: "hour", SLM: 9, Tipo: 1}})
	if len(um) != 4 || um[3].SolucaoMin != 480 || um[1].RespostaMin != 480 || um[0].SolucaoMin != 240 {
		t.Fatalf("um par: %+v", um)
	}
	if metasDosSLAs([]SLA{{Tempo: 1, Unidade: "hour", SLM: 1, Tipo: 1}, {Tempo: 1, Unidade: "hour"}, {Tempo: 1, Unidade: "ano", SLM: 2}}) != nil {
		t.Fatal("sem solução não há meta")
	}

	cals := []Calendario{{ID: 2, Nome: "Vazio"}, {ID: 3, Nome: "Comercial", Turnos: []Turno{{Dia: 1, Inicio: "07:00:00", Fim: "11:00:00"}}}}
	if c, ok := calendarioDosSLAs(slasDaPrefeitura, cals); !ok || c.Nome != "Comercial" {
		t.Fatalf("calendário: %+v %v", c, ok)
	}
	if _, ok := calendarioDosSLAs([]SLA{{Calendario: 2}, {}}, cals); ok {
		t.Fatal("calendário sem turnos não vale")
	}

	for nome, quer := range map[string]string{"Natal": "nacional", "Sexta-feira Santa": "nacional", "Carnaval": "ponto_facultativo",
		"Corpus Christi": "ponto_facultativo", "Aniversário de Rondonópolis": "municipal"} {
		if got := tipoDoFeriado(nome); got != quer {
			t.Errorf("%s: %s", nome, got)
		}
	}
	hoje := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	if d := datasDoFeriado(Feriado{Inicio: "2015-12-10", Fim: "2015-12-10", Perpetuo: 1}, hoje); strings.Join(d, ",") != "2026-12-10,2027-12-10" {
		t.Fatalf("perpétuo: %v", d)
	}
	if d := datasDoFeriado(Feriado{Inicio: "2027-02-08", Fim: "2027-02-09"}, hoje); len(d) != 2 {
		t.Fatalf("carnaval de dois dias: %v", d)
	}
	if d := datasDoFeriado(Feriado{Inicio: "2025-05-01", Fim: "2025-04-01"}, hoje); len(d) != 0 {
		t.Fatalf("passado e fim antes do início: %v", d)
	}
	if d := datasDoFeriado(Feriado{Inicio: "x"}, hoje); d != nil {
		t.Fatalf("data ilegível: %v", d)
	}
	if hhmm("08:00:00") != "08:00" || hhmm("8") != "8" {
		t.Fatal("hhmm")
	}
	if !entidadeRaiz("") || !entidadeRaiz("RAIZ") || !entidadeRaiz("RONDONOPOLIS") || entidadeRaiz("OUTRA") {
		t.Fatal("entidadeRaiz")
	}
	for nome, quer := range map[string]string{
		"RAIZ > Suporte SMS":      "SMS",
		"RAIZ > TI da Saúde":      "SMS",
		"RAIZ > TI SEMECEL":       "SEMECEL",
		"RAIZ > Suporte Educação": "SEMECEL",
		"RAIZ > N1 Geral":         "PREFEITURA",
	} {
		if got := secretariaDoGrupo(nome); got != quer {
			t.Errorf("secretariaDoGrupo(%q): %s, quer %s", nome, got, quer)
		}
	}
}

// glpiDoExpediente responde as rotas do calendário e dos grupos.
func glpiDoExpediente(t *testing.T, quebrado string) *Cliente {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho := strings.TrimPrefix(r.URL.Path, "/apirest.php")
		if caminho == quebrado {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		corpo := map[string]any{
			"/initSession":      map[string]string{"session_token": "s1"},
			"/Calendar":         []map[string]any{{"id": 3, "name": "Horário Comercial"}, {"id": 4, "name": "Plantão"}},
			"/CalendarSegment":  []map[string]any{{"calendars_id": 3, "day": 1, "begin": "07:00:00", "end": "11:00:00"}, {"calendars_id": 4, "day": 0, "begin": "00:00:00", "end": "24:00:00"}},
			"/Holiday":          []map[string]any{{"id": 9, "name": "Natal", "begin_date": "2015-12-25", "end_date": "2015-12-25", "is_perpetual": 1}},
			"/Calendar_Holiday": []map[string]any{{"calendars_id": 3, "holidays_id": 9}, {"calendars_id": 3, "holidays_id": 99}},
			"/Group":            []map[string]any{{"id": 7, "completename": "Grupo Técnico - SMS", "entities_id": 1}},
			"/Group_User":       []map[string]any{{"groups_id": 7, "users_id": 5}, {"groups_id": 8, "users_id": 6}},
			"/Group_Ticket":     []map[string]any{{"groups_id": 7, "type": 2}, {"groups_id": 7, "type": 2}, {"groups_id": 7, "type": 1}},
		}[caminho]
		if corpo == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(corpo)
	}))
	t.Cleanup(srv.Close)
	c := NovoCliente(srv.URL+"/apirest.php", "a1", "u1")
	if err := c.Abrir(context.Background()); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestClienteDoExpediente(t *testing.T) {
	ctx := context.Background()
	c := glpiDoExpediente(t, "")
	cals, err := c.Calendarios(ctx)
	if err != nil || len(cals) != 2 || len(cals[0].Turnos) != 1 || len(cals[0].Feriados) != 1 || cals[0].Feriados[0].Nome != "Natal" ||
		len(cals[1].Turnos) != 1 || len(cals[1].Feriados) != 0 {
		t.Fatalf("calendários: %+v %v", cals, err)
	}
	gs, err := c.Grupos(ctx)
	if err != nil || len(gs) != 1 || gs[0].EntidadeID != 1 || len(gs[0].Membros) != 1 || gs[0].Membros[0] != 5 || gs[0].Chamados != 2 {
		t.Fatalf("grupos: %+v %v", gs, err)
	}
	for _, q := range []string{"/Calendar", "/CalendarSegment", "/Holiday", "/Calendar_Holiday"} {
		if _, err := glpiDoExpediente(t, q).Calendarios(ctx); err == nil {
			t.Errorf("%s quebrado", q)
		}
	}
	for _, q := range []string{"/Group", "/Group_User", "/Group_Ticket"} {
		if _, err := glpiDoExpediente(t, q).Grupos(ctx); err == nil {
			t.Errorf("%s quebrado", q)
		}
	}
}

// expedienteDeTeste é a área de preparo com o GLPI da prefeitura: a
// secretaria SMS, os níveis, o calendário e um grupo com a Ana e um
// técnico sem conta aqui.
func expedienteDeTeste(t *testing.T, login, feriado string) *memoria {
	t.Helper()
	o := novaMemoria()
	o.guardar(ItemPreparo{Tipo: ItemEntidade, ID: 1, Dados: dados(t, Entidade{ID: 1, Nome: "RONDONOPOLIS > SMS"})})
	o.guardar(ItemPreparo{Tipo: ItemEntidade, ID: 2, Dados: dados(t, Entidade{ID: 2, Nome: "RONDONOPOLIS > OUTRA"})})
	for _, s := range slasDaPrefeitura {
		o.guardar(ItemPreparo{Tipo: ItemSLA, ID: s.ID, Dados: dados(t, s)})
	}
	o.guardar(ItemPreparo{Tipo: ItemCalendario, ID: 3, Dados: dados(t, Calendario{ID: 3, Nome: "Horário Comercial",
		Turnos:   []Turno{{Dia: 1, Inicio: "07:00:00", Fim: "11:00:00"}, {Dia: 1, Inicio: "13:00:00", Fim: "17:00:00"}},
		Feriados: []Feriado{{Nome: feriado, Inicio: "2015-12-10", Fim: "2015-12-10", Perpetuo: 1}}})})
	o.guardar(ItemPreparo{Tipo: ItemGrupo, ID: 7, Dados: dados(t, Grupo{ID: 7, Nome: "Grupo Técnico - SMS", EntidadeID: 1, Membros: []int{5, 6}, Chamados: 40})})
	o.guardar(ItemPreparo{Tipo: ItemGrupo, ID: 8, Dados: dados(t, Grupo{ID: 8, Nome: "Grupo de outra", EntidadeID: 2, Membros: []int{5}, Chamados: 3})})
	o.guardar(ItemPreparo{Tipo: ItemGrupo, ID: 9, Dados: dados(t, Grupo{ID: 9, Nome: "Vazio", EntidadeID: 1, Chamados: 1})})
	// Do AD, com membros, mas sem nunca atender: fica de fora.
	o.guardar(ItemPreparo{Tipo: ItemGrupo, ID: 10, Dados: dados(t, Grupo{ID: 10, Nome: "Domain Users", EntidadeID: 1, Membros: []int{5}})})
	o.guardar(ItemPreparo{Tipo: ItemPessoa, ID: 5, Dados: dados(t, pessoaPreparada{Pessoa: Pessoa{ID: 5, Login: login}})})
	o.guardar(ItemPreparo{Tipo: ItemPessoa, ID: 6, Dados: dados(t, pessoaPreparada{Pessoa: Pessoa{ID: 6, Login: "sem-conta-" + uuid.NewString()}})})
	return o
}

func TestCargaDoExpediente(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	u := dbtest.Unidade(t, pool)
	ana := dbtest.User(t, pool)
	var login string
	if err := pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, ana).Scan(&login); err != nil {
		t.Fatal(err)
	}
	sfx := uuid.NewString()[:6]
	feriado := "Aniversário da cidade " + sfx
	limpar := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1`, feriado)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE unidade_id = $1`, u)
	}
	limpar()
	t.Cleanup(limpar)
	mapa := NovoMapa([]LinhaMapa{{Secretaria: "SMS", Categoria: -1, Destino: Destino{Unidade: &u, UnidadeNome: "U", Oferta: "Geral",
		Tipo: "incidente", Prioridade: "media"}}})
	fuso, _ := time.LoadLocation("America/Cuiaba")
	executar := func(gravar bool) Relatorio {
		t.Helper()
		im := &Importador{Fonte: &FontePreparo{Leitor: expedienteDeTeste(t, login, feriado)}, Pool: pool, Repo: infrastructure.NewRepository(),
			Gravar: gravar, Fuso: fuso, Mapa: mapa}
		rel, err := im.Executar(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return rel
	}

	// A simulação conta e não grava.
	sim := executar(false).Expediente
	if sim == nil || sim.Politica != "criada" || sim.Calendario != "Horário Comercial" || len(sim.Metas) != 4 || sim.Feriados != 2 ||
		sim.Equipes != 1 || sim.Membros != 1 || sim.MembrosSemPar != 1 || len(sim.SemUnidade) != 1 || sim.GruposSemChamado != 1 {
		t.Fatalf("simulação: %+v", sim)
	}
	if _, err := infrastructure.NewRepository().GetPolitica(ctx, pool, nil); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("a simulação não grava: %v", err)
	}

	// A carga cria a política, os feriados e a equipe.
	if r := executar(true).Expediente; r.Politica != "criada" || r.Feriados != 2 || r.Equipes != 1 {
		t.Fatalf("carga: %+v", r)
	}
	p, err := infrastructure.NewRepository().GetPolitica(ctx, pool, nil)
	if err != nil || p.Fuso != "America/Cuiaba" || len(p.Turnos) != 2 || p.Turnos[0].Inicio != "07:00" || len(p.Metas) != 4 || p.Metas[1].SolucaoMin != 240 {
		t.Fatalf("política: %+v %v", p, err)
	}
	var feriados, membros int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1 AND tipo = 'municipal'`, feriado).Scan(&feriados); err != nil || feriados != 2 {
		t.Fatalf("feriados: %d %v", feriados, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_equipe_membros m JOIN atendimento_equipes e ON e.id = m.equipe_id
		WHERE e.unidade_id = $1 AND e.nome = 'Grupo Técnico - SMS' AND m.user_id = $2`, u, ana).Scan(&membros); err != nil || membros != 1 {
		t.Fatalf("equipe com a Ana: %d %v", membros, err)
	}

	// De novo: a política configurada fica, nada se repete.
	if r := executar(true).Expediente; r.Politica != "existente" || r.Feriados != 0 || r.Equipes != 1 {
		t.Fatalf("de novo: %+v", r)
	}
	var equipes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_equipes WHERE unidade_id = $1`, u).Scan(&equipes); err != nil || equipes != 1 {
		t.Fatalf("uma equipe só: %d %v", equipes, err)
	}
}

// expedienteFalho falha numa das leituras do expediente.
type expedienteFalho struct {
	*FontePreparo
	passo string
}

func (e expedienteFalho) Calendarios(ctx context.Context) ([]Calendario, error) {
	if e.passo == "calendarios" {
		return nil, errFonte
	}
	return e.FontePreparo.Calendarios(ctx)
}

func (e expedienteFalho) Grupos(ctx context.Context) ([]Grupo, error) {
	if e.passo == "grupos" {
		return nil, errFonte
	}
	return e.FontePreparo.Grupos(ctx)
}

func (e expedienteFalho) Pessoa(ctx context.Context, id int) (Pessoa, error) {
	if e.passo == "pessoa" {
		return Pessoa{}, errFonte
	}
	return e.FontePreparo.Pessoa(ctx, id)
}

// repoDoExpedienteFalho falha nas gravações do expediente.
type repoDoExpedienteFalho struct {
	domain.Repository
	passo string
}

func (r repoDoExpedienteFalho) GetPolitica(ctx context.Context, db database.DBTX, u *uuid.UUID) (domain.Politica, error) {
	if r.passo == "politica" {
		return domain.Politica{}, errFonte
	}
	return r.Repository.GetPolitica(ctx, db, u)
}

func (r repoDoExpedienteFalho) ListFeriados(ctx context.Context, db database.DBTX, u *uuid.UUID) ([]domain.Feriado, error) {
	if r.passo == "feriados" {
		return nil, errFonte
	}
	return r.Repository.ListFeriados(ctx, db, u)
}

func (r repoDoExpedienteFalho) InsertFeriado(ctx context.Context, db database.DBTX, f domain.Feriado) (domain.Feriado, error) {
	if r.passo == "feriado" {
		return domain.Feriado{}, errFonte
	}
	return r.Repository.InsertFeriado(ctx, db, f)
}

func (r repoDoExpedienteFalho) InsertEquipe(ctx context.Context, db database.DBTX, e domain.Equipe) (domain.Equipe, error) {
	if r.passo == "equipe" {
		return domain.Equipe{}, errFonte
	}
	return r.Repository.InsertEquipe(ctx, db, e)
}

func TestCargaDoExpedienteComFalhas(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	u := dbtest.Unidade(t, pool)
	_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE unidade_id = $1`, u)
	})
	mapa := NovoMapa([]LinhaMapa{{Secretaria: "SMS", Categoria: -1, Destino: Destino{Unidade: &u, UnidadeNome: "U", Oferta: "Geral",
		Tipo: "incidente", Prioridade: "media"}}})
	feriado := "Falha " + uuid.NewString()[:6]
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1`, feriado)
	})
	real := infrastructure.NewRepository()
	for _, caso := range []struct {
		fonte, repo string
	}{{"calendarios", ""}, {"grupos", ""}, {"pessoa", ""}, {"", "politica"}, {"", "feriados"}, {"", "feriado"}, {"", "equipe"}} {
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1`, feriado)
		im := &Importador{Fonte: expedienteFalho{&FontePreparo{Leitor: expedienteDeTeste(t, "ninguem-"+uuid.NewString(), feriado)}, caso.fonte},
			Pool: pool, Repo: repoDoExpedienteFalho{real, caso.repo}, Gravar: true, Fuso: time.UTC, Mapa: mapa}
		if _, err := im.Executar(ctx); !errors.Is(err, errFonte) {
			t.Errorf("%+v: %v", caso, err)
		}
	}
	// Turno ilegível: a política não se monta, e o erro diz de qual calendário.
	ruim := expedienteDeTeste(t, "x", feriado)
	ruim.guardar(ItemPreparo{Tipo: ItemCalendario, ID: 3, Dados: dados(t, Calendario{ID: 3, Nome: "Quebrado",
		Turnos: []Turno{{Dia: 9, Inicio: "07:00", Fim: "11:00"}}})})
	im := &Importador{Fonte: &FontePreparo{Leitor: ruim}, Pool: pool, Repo: real, Gravar: true, Fuso: time.UTC, Mapa: mapa}
	if _, err := im.Executar(ctx); err == nil || !strings.Contains(err.Error(), "Quebrado") {
		t.Errorf("turno ilegível: %v", err)
	}
	// Sem calendário nos SLAs: a política não se cria, e as equipes seguem.
	sem := novaMemoria()
	sem.guardar(ItemPreparo{Tipo: ItemEntidade, ID: 1, Dados: dados(t, Entidade{ID: 1, Nome: "RONDONOPOLIS > SMS"})})
	im = &Importador{Fonte: &FontePreparo{Leitor: sem}, Pool: pool, Repo: real, Fuso: time.UTC, Mapa: mapa}
	if rel, err := im.Executar(ctx); err != nil || rel.Expediente.Politica != "sem calendário" {
		t.Errorf("sem calendário: %+v %v", rel.Expediente, err)
	}
}
