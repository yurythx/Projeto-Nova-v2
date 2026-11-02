package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type projetoResp struct {
	ID     string `json:"id"`
	Numero int64  `json:"numero"`
	Titulo string `json:"titulo"`
	Estado string `json:"estado"`
}

func TestProjetosRotas(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	_, semPerm := h.user("user-sem-perm")

	h.expect(http.StatusUnauthorized, http.MethodGet, "/api/v1/projetos", "", "")
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/projetos", semPerm, `{"titulo":"Projeto X"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos", admin, `{"titulo":""}`)

	p := data[projetoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Modernização do Portal","descricao":"Novo portal institucional","prioridade":"alta"}`))

	if p.ID == "" || p.Titulo != "Modernização do Portal" {
		t.Fatalf("resposta inesperada: %+v", p)
	}

	lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos", admin, "").Body.String()
	if !strings.Contains(lista, p.ID) {
		t.Fatalf("projeto não encontrado na listagem: %s", lista)
	}

	obter := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID, admin, "").Body.String()
	if !strings.Contains(obter, "Modernização do Portal") {
		t.Fatalf("detalhes não encontrados: %s", obter)
	}

	h.expect(http.StatusOK, http.MethodPost, "/api/v1/projetos/"+p.ID+"/estado", admin, `{"estado":"em_andamento"}`)

	// Transição inválida de em_andamento para cancelado é permitida, mas direto para concluido precisa passar
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/projetos/"+p.ID+"/estado", admin, `{"estado":"concluido"}`)

	// Tarefas
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos/"+p.ID+"/tarefas", admin,
		`{"titulo":"Levantar requisitos","estado":"a_fazer"}`)

	tarefas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID+"/tarefas", admin, "").Body.String()
	if !strings.Contains(tarefas, "Levantar requisitos") {
		t.Fatalf("tarefa não encontrada: %s", tarefas)
	}

	// Vínculos (valida a existência do item no módulo de origem e preenche rótulo se omitido)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos/"+p.ID+"/vinculos", admin,
		`{"tipo":"mudanca","alvo_id":"`+uuid.NewString()+`","rotulo":"Inexistente"}`)

	ativo := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/ativos", admin,
		`{"nome":"Servidor de Banco","tipo":"hardware","estado":"em_uso"}`))
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos/"+p.ID+"/vinculos", admin,
		`{"tipo":"ativo","alvo_id":"`+ativo.ID+`","rotulo":""}`)

	vinculos := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID+"/vinculos", admin, "").Body.String()
	if !strings.Contains(vinculos, "Ativo · Servidor de Banco") {
		t.Fatalf("vínculo não encontrado: %s", vinculos)
	}

	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/projetos/"+uuid.NewString(), admin, "")
}

func TestProjetosCarta5W2H(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Fora do prazo","data_inicio_planejada":"2026-12-01T12:00:00Z","data_fim_planejada":"2026-01-01T12:00:00Z","carta":{}}`)

	p := data[projetoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos", admin, `{
		"titulo":"Portal 5W2H",
		"descricao":"Escopo do portal",
		"objetivo":"Atender o cidadão",
		"orcamento":1000,
		"carta":{"onde":"SEMED","patrocinador":"Secretário","metodologia":"Entregas quinzenais","centro_custo":"CC-12"},
		"etapas":["Planejamento","Execução"]
	}`))

	carta := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID+"/carta", admin, "").Body.String()
	if !strings.Contains(carta, "SEMED") || !strings.Contains(carta, `"etapas_how":2`) || !strings.Contains(carta, `"existe":true`) {
		t.Fatalf("carta: %s", carta)
	}

	put := `{
		"titulo":"Portal 5W2H",
		"descricao":"Escopo revisado",
		"objetivo":"Atender o cidadão",
		"onde":"SEMED",
		"patrocinador":"Secretário",
		"metodologia":"Entregas quinzenais",
		"centro_custo":"CC-12",
		"orcamento":1000
	}`
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/projetos/"+p.ID+"/carta", admin, put)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/projetos/"+p.ID+"/carta", admin, put)

	tarefas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID+"/tarefas", admin, "").Body.String()
	if strings.Count(tarefas, "Planejamento") != 1 || strings.Count(tarefas, `"origem":"how"`) != 2 {
		t.Fatalf("tarefas depois de salvar a carta duas vezes: %s", tarefas)
	}
	revisada := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID+"/carta", admin, "").Body.String()
	if !strings.Contains(revisada, "Escopo revisado") || !strings.Contains(revisada, `"etapas_how":2`) {
		t.Fatalf("carta revisada: %s", revisada)
	}
}

func TestProjetosPessoaDaUnidade(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	pessoa, _ := h.user("nova-user")
	ctx := context.Background()
	var nome string
	if err := h.d.DB.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, pessoa).Scan(&nome); err != nil {
		t.Fatal(err)
	}
	unidade := dbtest.Unidade(t, h.d.DB)
	outra := dbtest.Unidade(t, h.d.DB)
	if _, err := h.d.DB.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, pessoa, unidade); err != nil {
		t.Fatal(err)
	}

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Nome livre","gerente_nome":"Ana"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Fora da unidade","unidade_id":"`+outra.String()+`","gerente_id":"`+pessoa.String()+`"}`)

	p := data[projetoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Com gerente","unidade_id":"`+unidade.String()+`","gerente_id":"`+pessoa.String()+`"}`))
	corpo := h.expect(http.StatusOK, http.MethodGet, "/api/v1/projetos/"+p.ID, admin, "").Body.String()
	if !strings.Contains(corpo, nome) || !strings.Contains(corpo, pessoa.String()) {
		t.Fatalf("gerente: %s", corpo)
	}

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos/"+p.ID+"/tarefas", admin,
		`{"titulo":"Texto livre","responsavel_nome":"Ana"}`)
	tar := h.expect(http.StatusCreated, http.MethodPost, "/api/v1/projetos/"+p.ID+"/tarefas", admin,
		`{"titulo":"Com pessoa","responsavel_id":"`+pessoa.String()+`"}`).Body.String()
	if !strings.Contains(tar, nome) {
		t.Fatalf("tarefa: %s", tar)
	}

	if _, err := h.d.DB.Exec(ctx, `UPDATE users SET active = false WHERE id = $1`, pessoa); err != nil {
		t.Fatal(err)
	}
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/projetos", admin,
		`{"titulo":"Inativa","gerente_id":"`+pessoa.String()+`"}`)
}
