package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/glpi"
)

func ambiente(v map[string]string) func(string) string { return func(k string) string { return v[k] } }

var envOK = map[string]string{"NOVA_API": "https://api.exemplo", "NOVA_TOKEN": "nova_con_x", "GLPI_URL": "http://glpi/apirest.php",
	"GLPI_LOGIN": "yuri", "GLPI_PASSWORD": "s"}

func TestLer(t *testing.T) {
	op, err := ler([]string{"--vigiar", "--intervalo", "5s"}, ambiente(envOK))
	if err != nil || !op.vigiar || op.intervalo != 5*time.Second || op.api != "https://api.exemplo" || op.login != "yuri" {
		t.Fatalf("opções: %+v %v", op, err)
	}
	com := func(k, v string) map[string]string {
		m := map[string]string{}
		for a, b := range envOK {
			m[a] = b
		}
		m[k] = v
		return m
	}
	for nome, c := range map[string]struct {
		args []string
		env  map[string]string
	}{
		"opção":      {[]string{"--nada"}, envOK},
		"api":        {nil, com("NOVA_API", "http://api.exemplo")},
		"api ruim":   {nil, com("NOVA_API", "api")},
		"token":      {nil, com("NOVA_TOKEN", "eyJ")},
		"credencial": {nil, com("GLPI_PASSWORD", "")},
		"sem glpi":   {nil, com("GLPI_URL", "")},
		"intervalo":  {[]string{"--intervalo", "1ms"}, envOK},
	} {
		if _, err := ler(c.args, ambiente(c.env)); err == nil {
			t.Errorf("%s: aceitou", nome)
		}
	}
	if u, err := enderecoDaAPI("http://localhost:8010/"); err != nil || u != "http://localhost:8010" {
		t.Fatalf("localhost em http: %q %v", u, err)
	}
}

// apiFalsa é a API do Projeto Nova do lado do conector.
type apiFalsa struct {
	mu        sync.Mutex
	tarefa    bool
	estado    string
	lotes     [][]glpi.ItemPreparo
	fim       map[string]any
	versoes   map[string]string
	falhaItem bool
}

func (a *apiFalsa) servidor(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer nova_con_x" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"token inválido"}}`))
			return
		}
		responder := func(v any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": v}) }
		caminho := strings.TrimPrefix(r.URL.Path, "/api/v1/integracoes/conector")
		switch {
		case caminho == "/tarefa":
			if !a.tarefa {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			a.tarefa = false
			responder(map[string]string{"id": "t1"})
		case caminho == "/versoes":
			responder(map[string]any{"versoes": a.versoes, "pessoas": []int{6}})
		case caminho == "/itens":
			if a.falhaItem {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"error":{"message":"lote recusado"}}`))
				return
			}
			var corpo struct {
				Itens []glpi.ItemPreparo `json:"itens"`
			}
			_ = json.NewDecoder(r.Body).Decode(&corpo)
			a.lotes = append(a.lotes, corpo.Itens)
			w.WriteHeader(http.StatusNoContent)
		case caminho == "/tarefas/t1/progresso":
			responder(map[string]string{"estado": a.estado})
		case caminho == "/tarefas/t1/fim":
			_ = json.NewDecoder(r.Body).Decode(&a.fim)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// memoria é o GLPI do teste: uma área de preparo já cheia serve de origem.
type memoria map[string][]glpi.ItemPreparo

func (m memoria) Itens(_ context.Context, tipo string, de, limite int) ([]glpi.ItemPreparo, int, error) {
	l := m[tipo]
	sort.Slice(l, func(i, j int) bool { return l[i].ID < l[j].ID })
	fim := min(len(l), de+limite)
	if de > fim {
		de = fim
	}
	return l[de:fim], len(l), nil
}

func (m memoria) Item(_ context.Context, tipo string, id int) (glpi.ItemPreparo, bool, error) {
	for _, it := range m[tipo] {
		if it.ID == id {
			return it, true, nil
		}
	}
	return glpi.ItemPreparo{}, false, nil
}

func glpiEmMemoria(t *testing.T) abridor {
	t.Helper()
	m := memoria{}
	pac := func(id int) json.RawMessage {
		b, _ := json.Marshal(glpi.Pacote{Chamado: glpi.Chamado{ID: id, Titulo: "c", Tipo: 1, Modificado: "v"},
			Participantes: []glpi.Participante{{PessoaID: 5, Papel: 1}, {PessoaID: 6, Papel: 2}}})
		return b
	}
	m[glpi.ItemChamado] = []glpi.ItemPreparo{{Tipo: glpi.ItemChamado, ID: 1, Dados: pac(1)}, {Tipo: glpi.ItemChamado, ID: 2, Dados: pac(2)}}
	m[glpi.ItemPessoa] = []glpi.ItemPreparo{{Tipo: glpi.ItemPessoa, ID: 5, Dados: json.RawMessage(`{"id":5,"name":"ana"}`)}}
	return func(context.Context, opcoes) (glpi.Extraivel, func(), error) {
		return &glpi.FontePreparo{Leitor: m}, func() {}, nil
	}
}

func opcoesPara(srv *httptest.Server) opcoes {
	return opcoes{api: srv.URL, token: "nova_con_x", intervalo: 10 * time.Millisecond}
}

func TestConectorExtrai(t *testing.T) {
	a := &apiFalsa{tarefa: true, estado: "em_andamento", versoes: map[string]string{"2": "v"}}
	srv := a.servidor(t)
	var saida bytes.Buffer
	if err := rodar(context.Background(), opcoesPara(srv), glpiEmMemoria(t), &saida); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saida.String(), "Concluída: 2 chamados, 1 novos ou alterados, 1 pessoas") {
		t.Fatalf("saída: %s", saida.String())
	}
	total := 0
	for _, l := range a.lotes {
		total += len(l)
	}
	// O chamado 2 é igual (mesma versão) e a pessoa 6 já estava lá.
	if total != 2 || a.fim["erro"] != nil {
		t.Fatalf("lotes: %+v fim: %v", a.lotes, a.fim)
	}

	// Sem pedido, avisa e sai.
	saida.Reset()
	if err := rodar(context.Background(), opcoesPara(srv), glpiEmMemoria(t), &saida); err != nil || !strings.Contains(saida.String(), "Nenhuma extração pedida") {
		t.Fatalf("sem pedido: %v %s", err, saida.String())
	}
}

func TestConectorFalhasECancelamento(t *testing.T) {
	ctx := context.Background()
	// A sessão no GLPI recusada vira o erro da tarefa.
	a := &apiFalsa{tarefa: true, estado: "em_andamento"}
	srv := a.servidor(t)
	recusa := func(context.Context, opcoes) (glpi.Extraivel, func(), error) {
		return nil, nil, errors.New("sessão no GLPI: 401")
	}
	var saida bytes.Buffer
	if err := rodar(ctx, opcoesPara(srv), recusa, &saida); err != nil || a.fim["erro"] != "sessão no GLPI: 401" {
		t.Fatalf("sessão recusada: %v %v", err, a.fim)
	}
	// O lote recusado pela API também.
	a.tarefa, a.falhaItem = true, true
	if err := rodar(ctx, opcoesPara(srv), glpiEmMemoria(t), &saida); err != nil || !strings.Contains(a.fim["erro"].(string), "lote recusado") {
		t.Fatalf("lote recusado: %v %v", err, a.fim)
	}
	// Cancelada na tela: o progresso diz para parar.
	a.tarefa, a.falhaItem, a.estado, a.fim = true, false, "cancelada", nil
	if err := rodar(ctx, opcoesPara(srv), glpiEmMemoria(t), &saida); err != nil || a.fim == nil {
		t.Fatalf("cancelada: %v %v", err, a.fim)
	}
	// Token errado: o erro da API chega a quem rodou.
	op := opcoesPara(srv)
	op.token = "nova_con_errado"
	if err := rodar(ctx, op, glpiEmMemoria(t), &saida); err == nil || !strings.Contains(err.Error(), "token inválido") {
		t.Fatalf("token errado: %v", err)
	}
	// Vigiando, para quando o processo pede.
	vigia, parar := context.WithTimeout(ctx, 50*time.Millisecond)
	defer parar()
	op = opcoesPara(srv)
	op.vigiar = true
	if err := rodar(vigia, op, glpiEmMemoria(t), &saida); err != nil {
		t.Fatalf("vigiar: %v", err)
	}
	// O processo parando no meio da extração não fecha a tarefa.
	a.tarefa, a.estado, a.fim = true, "em_andamento", nil
	lento := func(ctx context.Context, _ opcoes) (glpi.Extraivel, func(), error) {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	curto, fim := context.WithTimeout(ctx, 30*time.Millisecond)
	defer fim()
	if err := rodar(curto, opcoesPara(srv), lento, &saida); err != nil || a.fim != nil {
		t.Fatalf("parado no meio: %v %v", err, a.fim)
	}
}

func TestLotesPorTamanho(t *testing.T) {
	a := &apiFalsa{}
	srv := a.servidor(t)
	d := &destinoAPI{api: &cliente{base: srv.URL + "/api/v1/integracoes/conector", token: "nova_con_x", http: srv.Client()}}
	grande := json.RawMessage(`"` + strings.Repeat("a", 400<<10) + `"`)
	itens := []glpi.ItemPreparo{}
	for i := 0; i < 3; i++ {
		itens = append(itens, glpi.ItemPreparo{Tipo: glpi.ItemPessoa, ID: i, Dados: grande})
	}
	for i := 0; i < 250; i++ {
		itens = append(itens, glpi.ItemPreparo{Tipo: glpi.ItemPessoa, ID: 100 + i, Dados: json.RawMessage(`{}`)})
	}
	if err := d.Gravar(context.Background(), itens); err != nil {
		t.Fatal(err)
	}
	for _, l := range a.lotes {
		if len(l) > 200 {
			t.Fatalf("lote com %d itens", len(l))
		}
	}
	if len(a.lotes) < 3 {
		t.Fatalf("os grandes vão separados: %d lotes", len(a.lotes))
	}
	if err := d.Gravar(context.Background(), nil); err != nil {
		t.Fatal("nada a enviar")
	}
}
