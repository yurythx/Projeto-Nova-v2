package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
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
		`{"titulo":"Levantar requisitos","responsavel_nome":"Yuri","estado":"a_fazer"}`)

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
