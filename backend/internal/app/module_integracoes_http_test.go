package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/glpi"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes"
	intapp "github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/application"
	intdomain "github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
)

// glpiDeTeste responde como a API do GLPI: duas secretarias, um SLA e três
// chamados (um sem categoria). Basic Auth com yuri/senha.
func glpiDeTeste(t *testing.T) *httptest.Server {
	t.Helper()
	chamados := []map[string]any{
		{"id": 11, "name": "PC não liga", "content": "&lt;p&gt;Tela preta&lt;/p&gt;", "date": "2025-03-10 09:00:00", "closedate": "2025-03-10 12:00:00",
			"status": 6, "type": 2, "urgency": 3, "impact": 3, "itilcategories_id": 7, "entities_id": 1, "slas_id_ttr": 2, "slas_id_tto": 1, "date_mod": "2025-03-10 12:00:00"},
		{"id": 12, "name": "Internet lenta", "date": "2025-04-01 08:00:00", "status": 2, "type": 1, "urgency": 3, "impact": 3,
			"itilcategories_id": 7, "entities_id": 4, "date_mod": "2025-04-01 08:00:00"},
		{"id": 13, "name": "Sem categoria", "date": "2025-04-02 08:00:00", "status": 1, "type": 2, "entities_id": 1, "date_mod": "2025-04-02 08:00:00"},
	}
	responder := func(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho := strings.TrimPrefix(r.URL.Path, "/apirest.php")
		if caminho == "/initSession" {
			if u, p, ok := r.BasicAuth(); !ok || u != "yuri" || p != "senha" {
				w.WriteHeader(http.StatusUnauthorized)
				responder(w, []string{"ERROR_GLPI_LOGIN", "inválido"})
				return
			}
			responder(w, map[string]string{"session_token": "s"})
			return
		}
		var id int
		_, _ = fmt.Sscanf(caminho, "/User/%d", &id)
		switch {
		case caminho == "/getMyProfiles":
			responder(w, map[string]any{"myprofiles": []map[string]any{{"id": 4, "name": "Super-Admin"}}})
		case caminho == "/changeActiveProfile", caminho == "/killSession":
		case caminho == "/ITILCategory":
			responder(w, []map[string]any{{"id": 7, "completename": "Serviços > Computadores"}})
		case caminho == "/Entity":
			responder(w, []map[string]any{{"id": 0, "completename": "RAIZ"}, {"id": 1, "completename": "RAIZ > PREFEITURA"}, {"id": 4, "completename": "RAIZ > SMS"}})
		case caminho == "/SLA":
			responder(w, []map[string]any{{"id": 1, "name": "Atendimento N1", "number_time": 30, "definition_time": "minute"},
				{"id": 2, "name": "Solução N1", "number_time": 4, "definition_time": "hour"}})
		case caminho == "/Ticket":
			w.Header().Set("Content-Range", fmt.Sprintf("0-%d/%d", len(chamados)-1, len(chamados)))
			responder(w, chamados)
		case fmt.Sprintf("/Ticket/%d/Ticket_User", 11) == caminho:
			responder(w, []map[string]any{{"users_id": 5, "type": 1}, {"users_id": 6, "type": 2}})
		case strings.HasPrefix(caminho, "/User/") && strings.HasSuffix(caminho, "/UserEmail"):
			responder(w, []map[string]any{{"email": fmt.Sprintf("pessoa%d@glpi.test", id), "is_default": 1}})
		case strings.HasPrefix(caminho, "/User/"):
			responder(w, map[string]any{"id": id, "name": fmt.Sprintf("p%d", id), "firstname": "Pessoa", "realname": fmt.Sprint(id)})
		default:
			responder(w, []any{})
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func modulo(t *testing.T, h *apiHarness) *integracoes.Module {
	t.Helper()
	for _, p := range h.d.Kernel.Plugins() {
		if m, ok := p.(*integracoes.Module); ok {
			return m
		}
	}
	t.Fatal("módulo de integrações não registrado")
	return nil
}

func TestIntegracoesHTTP(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	dbtest.Exclusivo(t, h.d.DB, "glpi")
	admin := h.admin()
	_, comum := h.user("nova-user")
	h.setModule(admin, "atendimento", true)
	h.setModule(admin, integracoes.Key, true)
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/integracoes/conexoes", comum, "")
	mod := modulo(t, h)
	if ws := mod.Workers(); len(ws) != 1 || ws[0].Process != kernel.ProcessWorker {
		t.Fatalf("worker: %+v", ws)
	}

	srv := glpiDeTeste(t)
	base := "/api/v1/integracoes/conexoes"
	h.expect(http.StatusBadRequest, http.MethodPost, base, admin, `{`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, base, admin, `{"tipo":"otrs","nome":"x","modo":"direto"}`)
	conexao := data[intdomain.Conexao](t, h.expect(http.StatusCreated, http.MethodPost, base, admin,
		`{"tipo":"glpi","nome":"GLPI de teste","modo":"direto","url":"`+srv.URL+`/apirest.php"}`))
	t.Cleanup(func() {
		_, _ = h.d.DB.Exec(context.Background(), `DELETE FROM integracoes_conexoes WHERE id = $1`, conexao.ID)
	})
	um := base + "/" + conexao.ID.String()
	h.expect(http.StatusOK, http.MethodGet, base, admin, "")
	h.expect(http.StatusBadRequest, http.MethodGet, base+"/x", admin, "")
	h.expect(http.StatusOK, http.MethodGet, um, admin, "")
	h.expect(http.StatusBadRequest, http.MethodPut, um, admin, `{`)
	h.expect(http.StatusOK, http.MethodPut, um, admin, `{"nome":"GLPI da prefeitura","url":"`+srv.URL+`/apirest.php","perfil":"Super-Admin"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, um+"/credenciais", admin, `{`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, um+"/credenciais", admin, `{"login":"yuri"}`)
	h.expect(http.StatusNoContent, http.MethodPut, um+"/credenciais", admin, `{"login":"yuri","senha":"senha"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, um+"/token", admin, "")

	// Extração direta: o worker lê o GLPI falso.
	h.expect(http.StatusBadRequest, http.MethodPost, um+"/tarefas", admin, `{`)
	h.expect(http.StatusCreated, http.MethodPost, um+"/tarefas", admin, `{"tipo":"extracao"}`)
	if err := mod.Servico().Passo(ctx); err != nil {
		t.Fatal(err)
	}
	tarefas := data[[]intdomain.Tarefa](t, h.expect(http.StatusOK, http.MethodGet, um+"/tarefas", admin, ""))
	if len(tarefas) != 1 || tarefas[0].Estado != intdomain.EstadoConcluida {
		t.Fatalf("extração: %+v", tarefas)
	}
	inv := data[intdomain.Inventario](t, h.expect(http.StatusOK, http.MethodGet, um+"/inventario", admin, ""))
	if inv.Itens[intdomain.ItemChamado] != 3 || inv.Itens[intdomain.ItemPessoa] != 2 || inv.Secretarias["PREFEITURA"] != 2 || inv.Secretarias["SMS"] != 1 {
		t.Fatalf("inventário: %+v", inv)
	}

	// Sugestão com a unidade da PREFEITURA; a SMS fica sem e trava a carga.
	u := dbtest.Unidade(t, h.d.DB)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_regras WHERE unidade_id = $1`, u)
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	})
	_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	h.expect(http.StatusBadRequest, http.MethodPost, um+"/tabela/sugestao", admin, `{`)
	tabela := data[[]intdomain.Linha](t, h.expect(http.StatusOK, http.MethodPost, um+"/tabela/sugestao", admin, `{"unidades":{"PREFEITURA":"`+u.String()+`"}}`))
	h.expect(http.StatusOK, http.MethodGet, um+"/tabela", admin, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, um+"/tarefas", admin, `{"tipo":"carga"}`)
	for i := range tabela {
		tabela[i].UnidadeID = &u
	}
	corpo, _ := json.Marshal(map[string]any{"linhas": tabela})
	h.expect(http.StatusBadRequest, http.MethodPut, um+"/tabela", admin, `{`)
	h.expect(http.StatusOK, http.MethodPut, um+"/tabela", admin, string(corpo))

	h.expect(http.StatusCreated, http.MethodPost, um+"/tarefas", admin, `{"tipo":"carga"}`)
	if err := mod.Servico().Passo(ctx); err != nil {
		t.Fatal(err)
	}
	tarefas = data[[]intdomain.Tarefa](t, h.expect(http.StatusOK, http.MethodGet, um+"/tarefas", admin, ""))
	if tarefas[0].Estado != intdomain.EstadoConcluida || !strings.Contains(string(tarefas[0].Relatorio), `"importados":3`) {
		t.Fatalf("carga: %+v", tarefas[0])
	}
	var prioridade string
	var meta *int
	if err := h.d.DB.QueryRow(ctx, `SELECT s.prioridade, s.meta_solucao_min FROM atendimento_solicitacoes s
		JOIN atendimento_importacao i ON i.destino = s.id AND i.origem = 'glpi' AND i.tipo = 'solicitacao' AND i.id_origem = '11'`).Scan(&prioridade, &meta); err != nil {
		t.Fatal(err)
	}
	if prioridade != "alta" || meta == nil || *meta != 240 {
		t.Fatalf("chamado 11 com a prioridade da tabela e o prazo do GLPI: %s %v", prioridade, meta)
	}

	// Sincronizar até a virada (ADR 081): ligada, o importado é só leitura
	// no atendimento; desligada, volta a ser daqui.
	var importado string
	if err := h.d.DB.QueryRow(ctx, `SELECT i.destino::text FROM atendimento_importacao i JOIN atendimento_solicitacoes s ON s.id = i.destino
		WHERE i.origem = 'glpi' AND i.tipo = 'solicitacao' AND s.estado NOT IN ('encerrado', 'cancelado') LIMIT 1`).Scan(&importado); err != nil {
		t.Fatal(err)
	}
	h.expect(http.StatusBadRequest, http.MethodPut, um+"/sincronizacao", admin, `{`)
	h.expect(http.StatusBadRequest, http.MethodPut, base+"/x/sincronizacao", admin, `{}`)
	if c := data[intdomain.Conexao](t, h.expect(http.StatusOK, http.MethodPut, um+"/sincronizacao", admin, `{"ligada":true}`)); !c.Sincronizar {
		t.Fatalf("ligada: %+v", c)
	}
	comentar := "/api/v1/atendimento/solicitacoes/" + importado + "/comentarios"
	if r := h.expect(http.StatusConflict, http.MethodPost, comentar, admin, `{"texto":"daqui"}`).Body.String(); !strings.Contains(r, "GLPI") {
		t.Fatalf("espelho: %s", r)
	}
	h.expect(http.StatusOK, http.MethodPut, um+"/sincronizacao", admin, `{"ligada":false}`)
	h.expect(http.StatusCreated, http.MethodPost, comentar, admin, `{"texto":"daqui"}`)
	_, _ = h.d.DB.Exec(ctx, `DELETE FROM atendimento_comentarios WHERE solicitacao_id = $1 AND texto = 'daqui'`, importado)

	// Desfazer a carga (ADR 079): a simulação conta e não remove; sem o
	// nome da conexão, recusa; com ele, remove os dois chamados que
	// ninguém mexeu aqui (o comentado acima fica).
	importados := func() int {
		var n int
		if err := h.d.DB.QueryRow(ctx, `SELECT count(*) FROM atendimento_solicitacoes WHERE unidade_id = $1`, u).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, pedido := range []string{`{"tipo":"desfazer_simulacao"}`, `{"tipo":"desfazer","confirmacao":"GLPI da prefeitura"}`} {
		h.expect(http.StatusUnprocessableEntity, http.MethodPost, um+"/tarefas", admin, `{"tipo":"desfazer","confirmacao":"outra"}`)
		h.expect(http.StatusCreated, http.MethodPost, um+"/tarefas", admin, pedido)
		if err := mod.Servico().Passo(ctx); err != nil {
			t.Fatal(err)
		}
		tarefas = data[[]intdomain.Tarefa](t, h.expect(http.StatusOK, http.MethodGet, um+"/tarefas", admin, ""))
		gravou := strings.Contains(pedido, `"desfazer"`)
		rel := string(tarefas[0].Relatorio)
		if tarefas[0].Estado != intdomain.EstadoConcluida || !strings.Contains(rel, `"chamados":2`) || !strings.Contains(rel, `"chamados_mantidos":1`) ||
			strings.Contains(rel, `"gravou":true`) != gravou || (importados() == 1) != gravou {
			t.Fatalf("%s: %+v %d", pedido, tarefas[0], importados())
		}
	}

	// Cancelar a tarefa pendente.
	pendente := data[intdomain.Tarefa](t, h.expect(http.StatusCreated, http.MethodPost, um+"/tarefas", admin, `{"tipo":"simulacao"}`))
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/integracoes/tarefas/x/cancelar", admin, "")
	h.expect(http.StatusNoContent, http.MethodPost, "/api/v1/integracoes/tarefas/"+pendente.ID.String()+"/cancelar", admin, "")
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/integracoes/tarefas/"+pendente.ID.String()+"/cancelar", admin, "")

	// O conector, com o token de máquina.
	k := data[intdomain.Conexao](t, h.expect(http.StatusCreated, http.MethodPost, base, admin, `{"tipo":"glpi","nome":"Pelo conector","modo":"conector"}`))
	kUm := base + "/" + k.ID.String()
	token := data[map[string]string](t, h.expect(http.StatusCreated, http.MethodPost, kUm+"/token", admin, ""))["token"]
	conector := "/api/v1/integracoes/conector"
	h.expect(http.StatusForbidden, http.MethodGet, conector+"/versoes", admin, "")
	h.expect(http.StatusForbidden, http.MethodGet, base, token, "")
	h.expect(http.StatusNoContent, http.MethodPost, conector+"/tarefa", token, "")
	h.expect(http.StatusCreated, http.MethodPost, kUm+"/tarefas", admin, `{"tipo":"extracao"}`)
	tf := data[intdomain.Tarefa](t, h.expect(http.StatusOK, http.MethodPost, conector+"/tarefa", token, ""))
	h.expect(http.StatusBadRequest, http.MethodPost, conector+"/itens", token, `{`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, conector+"/itens", token, `{"itens":[]}`)
	h.expect(http.StatusNoContent, http.MethodPost, conector+"/itens", token, `{"itens":[{"tipo":"pessoa","id":5,"dados":{}},{"tipo":"chamado","id":1,"versao":"v","dados":{}}]}`)
	v := data[struct {
		Versoes map[string]string `json:"versoes"`
		Pessoas []int             `json:"pessoas"`
	}](t, h.expect(http.StatusOK, http.MethodGet, conector+"/versoes", token, ""))
	if v.Versoes["1"] != "v" || len(v.Pessoas) != 1 {
		t.Fatalf("versões: %+v", v)
	}
	tfURL := conector + "/tarefas/" + tf.ID.String()
	h.expect(http.StatusBadRequest, http.MethodPost, conector+"/tarefas/x/progresso", token, `{}`)
	h.expect(http.StatusBadRequest, http.MethodPost, tfURL+"/progresso", token, `{`)
	if e := data[map[string]string](t, h.expect(http.StatusOK, http.MethodPost, tfURL+"/progresso", token, `{"feitos":1,"total":2}`)); e["estado"] != "em_andamento" {
		t.Fatalf("progresso: %v", e)
	}
	h.expect(http.StatusBadRequest, http.MethodPost, conector+"/tarefas/x/fim", token, `{}`)
	h.expect(http.StatusBadRequest, http.MethodPost, tfURL+"/fim", token, `{`)
	h.expect(http.StatusNoContent, http.MethodPost, tfURL+"/fim", token, `{"relatorio":{"chamados":1}}`)
	h.expect(http.StatusNoContent, http.MethodPost, tfURL+"/fim", token, `{}`)
	h.expect(http.StatusNotFound, http.MethodPost, conector+"/tarefas/"+uuid.NewString()+"/fim", token, `{}`)
	h.expect(http.StatusNoContent, http.MethodDelete, kUm+"/token", admin, "")
	h.expect(http.StatusUnauthorized, http.MethodGet, conector+"/versoes", token, "")
	// Um token de agente que não existe para antes de chegar ao conector.
	h.expect(http.StatusUnauthorized, http.MethodGet, conector+"/versoes", "nova_agt_nenhum", "")

	h.expect(http.StatusNoContent, http.MethodDelete, kUm, admin, "")
	for _, rota := range []string{"/api/v1/integracoes/conexoes/" + uuid.NewString()} {
		h.expect(http.StatusNotFound, http.MethodDelete, rota, admin, "")
	}
	for _, metodo := range []string{http.MethodGet + " /inventario", http.MethodGet + " /tabela", http.MethodGet + " /tarefas"} {
		partes := strings.SplitN(metodo, " ", 2)
		h.expect(http.StatusNotFound, partes[0], "/api/v1/integracoes/conexoes/"+uuid.NewString()+partes[1], admin, "")
	}
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/token", admin, "")
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/credenciais", admin, `{"user_token":"u"}`)
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/tabela/sugestao", admin, `{}`)
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/tarefas", admin, `{"tipo":"carga"}`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/tabela", admin, `{"linhas":[]}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/integracoes/conexoes/"+uuid.NewString()+"/tarefas", admin, `{}`)
	nada := "/api/v1/integracoes/conexoes/" + uuid.NewString()
	h.expect(http.StatusNotFound, http.MethodGet, nada, admin, "")
	h.expect(http.StatusNotFound, http.MethodPut, nada, admin, `{"nome":"x"}`)
	for _, r := range [][2]string{{http.MethodPut, ""}, {http.MethodDelete, ""}, {http.MethodPut, "/credenciais"}, {http.MethodPost, "/token"},
		{http.MethodDelete, "/token"}, {http.MethodGet, "/inventario"}, {http.MethodGet, "/tabela"}, {http.MethodPut, "/tabela"},
		{http.MethodPost, "/tabela/sugestao"}, {http.MethodGet, "/tarefas"}, {http.MethodPost, "/tarefas"}} {
		h.expect(http.StatusBadRequest, r[0], "/api/v1/integracoes/conexoes/x"+r[1], admin, `{}`)
	}
}

// As portas, direto: a origem que não é GLPI, a sessão recusada, o
// atendimento desligado e o fuso ilegível.
func TestPortasDasIntegracoes(t *testing.T) {
	ctx := context.Background()
	srv := glpiDeTeste(t)
	c := intdomain.Conexao{Tipo: intdomain.TipoGLPI, URL: srv.URL + "/apirest.php", Fuso: "America/Sao_Paulo"}
	ext := extratorGLPI{abrir: abrirGLPI}
	if _, err := ext.Extrair(ctx, intdomain.Conexao{Tipo: "otrs"}, intdomain.Credenciais{}, nil, nil); err == nil {
		t.Fatal("origem sem extrator")
	}
	if _, err := ext.Extrair(ctx, c, intdomain.Credenciais{Login: "yuri", Senha: "errada"}, nil, nil); err == nil || !strings.Contains(err.Error(), "sessão") {
		t.Fatalf("sessão recusada: %v", err)
	}
	desligado := carregadorGLPI{ativo: func() bool { return false }}
	if _, err := desligado.Carregar(ctx, c, nil, false, nil, nil); err == nil {
		t.Fatal("atendimento desligado")
	}
	if _, err := desligado.Desfazer(ctx, false, nil); err == nil {
		t.Fatal("desfazer com o atendimento desligado")
	}
	if ativo, err := (espelhoGLPI{ativo: func() bool { return false }}).Ativo(ctx); ativo || err != nil {
		t.Fatalf("integrações desligadas, sem espelho: %v %v", ativo, err)
	}
	ligado := carregadorGLPI{ativo: func() bool { return true }}
	c.Fuso = "Lugar/Nenhum"
	if _, err := ligado.Carregar(ctx, c, nil, false, nil, nil); err == nil {
		t.Fatal("fuso ilegível")
	}
	// A linha sem unidade aparece como falta, com o nome da secretaria.
	c.Fuso = "UTC"
	if _, err := ligado.Carregar(ctx, c, []intdomain.Linha{{Secretaria: "SMS", Categoria: -1, Oferta: "O", Tipo: "incidente", Prioridade: "alta"}},
		true, leitorVazio{}, nil); err == nil || !strings.Contains(err.Error(), "(sem unidade) SMS") {
		t.Fatalf("gravar sem unidade: %v", err)
	}
	var _ glpi.Leitor = leitorGLPI{}
}

// leitorVazio é uma área de preparo sem nada.
type leitorVazio struct{}

func (leitorVazio) Itens(context.Context, string, int, int) ([]intdomain.Item, int, error) {
	return nil, 0, nil
}
func (leitorVazio) Item(context.Context, string, int) (intdomain.Item, bool, error) {
	return intdomain.Item{}, false, nil
}

var _ intapp.Leitor = leitorVazio{}
