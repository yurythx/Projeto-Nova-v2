package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type recursoResp struct {
	ID        string  `json:"id"`
	Nome      string  `json:"nome"`
	Estagio   string  `json:"estagio"`
	UnidadeID *string `json:"unidade_id"`
	ProdutoID *string `json:"produto_id"`
	Ativa     bool    `json:"ativa"`
}

func TestProdutosCicloEEscopo(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	o := h.orgEscopo(t)
	gestor := h.gestorEm(t, o.a, "produtos:manage")
	_, semPerm := h.user("nexus-user")

	h.expect(http.StatusUnauthorized, http.MethodGet, "/api/v1/produtos", "", "")
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/produtos", semPerm, `{"nome":"Protocolo","estagio":"operacao"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/produtos", admin, `{`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/produtos", admin, `{"nome":"Sem estágio"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/produtos", admin, `{"nome":"X","estagio":"legado"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/produtos", gestor, `{"nome":"De B","estagio":"operacao","unidade_id":"`+o.b+`"}`)
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/produtos", gestor, `{"nome":"Institucional","estagio":"operacao"}`)

	deA := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", gestor,
		`{"nome":"Protocolo `+uuid.NewString()[:6]+`","estagio":"descoberta","descricao":"registro","unidade_id":"`+o.a+`"}`))
	deSub := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", gestor,
		`{"nome":"Rede da subunidade","estagio":"operacao","unidade_id":"`+o.sub+`"}`))
	deB := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", admin,
		`{"nome":"De B","estagio":"operacao","unidade_id":"`+o.b+`"}`))
	inst := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", admin,
		`{"nome":"E-mail institucional","estagio":"operacao"}`))

	if outboxCount(t, h, "produtos.produto.criado", deA.ID) != 1 {
		t.Fatal("evento de criação")
	}
	lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/produtos", gestor, "").Body.String()
	if !strings.Contains(lista, deA.ID) || !strings.Contains(lista, deSub.ID) || strings.Contains(lista, deB.ID) || strings.Contains(lista, inst.ID) {
		t.Fatalf("lista do gestor de A: %s", lista)
	}
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/produtos/"+deB.ID, gestor, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/produtos/"+uuid.NewString(), admin, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/produtos/nao-e-uuid", admin, "")
	h.expect(http.StatusOK, http.MethodGet, "/api/v1/produtos/"+deA.ID, gestor, "")

	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/produtos/"+deA.ID, gestor,
		`{"nome":"Para B","unidade_id":"`+o.b+`"}`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/produtos/"+deB.ID, gestor, `{"nome":"Tomado","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/produtos/"+deA.ID, gestor,
		`{"nome":"Protocolo web","utilidade":"registrar","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/produtos/"+deA.ID, gestor,
		`{"nome":"Protocolo web","utilidade":"registrar","unidade_id":"`+o.a+`"}`)
	if outboxCount(t, h, "produtos.produto.atualizado", deA.ID) != 1 {
		t.Fatal("edição vazia não emite outro evento")
	}

	h.expect(http.StatusOK, http.MethodPost, "/api/v1/produtos/"+deA.ID+"/estagio", gestor, `{"estagio":"operacao"}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/produtos/"+deA.ID+"/estagio", gestor, `{"estagio":"operacao"}`)
	if outboxCount(t, h, "produtos.produto.estagio_alterado", deA.ID) != 1 {
		t.Fatal("repetir o estágio não emite outro evento")
	}
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/produtos/x/estagio", admin, `{"estagio":"operacao"}`)
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/produtos/"+uuid.NewString()+"/estagio", admin, `{"estagio":"operacao"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/produtos/"+deA.ID+"/estagio", gestor, `{"estagio":"legado"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/produtos/x", admin, `{"nome":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/produtos/"+deA.ID, gestor, `{`)
}

func TestAtendimentoOfertasEquipes(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	o := h.orgEscopo(t)
	uid, gestor := h.user("nexus-user")
	perms := `["atendimento:manage","produtos:manage"]`
	perfil := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/iam/perfis", admin,
		`{"nome":"Atendimento `+uuid.NewString()[:6]+`","permissoes":`+perms+`}`)).ID
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/users/"+uid.String()+"/lotacoes", admin,
		`{"perfil_id":"`+perfil+`","unidade_id":"`+o.a+`"}`)

	produto := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", gestor,
		`{"nome":"E-mail `+uuid.NewString()[:6]+`","estagio":"operacao","unidade_id":"`+o.a+`"}`))
	deB := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/produtos", admin,
		`{"nome":"Rede de B","estagio":"operacao","unidade_id":"`+o.b+`"}`))

	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"De B","unidade_id":"`+o.b+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"E-mail","unidade_id":"`+o.a+`","produto_id":"`+deB.ID+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"E-mail","unidade_id":"`+o.a+`","produto_id":"`+uuid.NewString()+`"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/ofertas", gestor, `{`)

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Não entro no e-mail","descricao":"acesso","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`))
	if !oferta.Ativa || oferta.ProdutoID == nil || *oferta.ProdutoID != produto.ID {
		t.Fatalf("oferta: %+v", oferta)
	}
	if outboxCount(t, h, "atendimento.oferta.criada", oferta.ID) != 1 {
		t.Fatal("evento da oferta")
	}
	h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor,
		`{"nome":"Não entro no e-mail","descricao":"acesso","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`","ativa":true}`)
	if outboxCount(t, h, "atendimento.oferta.atualizada", oferta.ID) != 0 {
		t.Fatal("edição vazia não emite evento")
	}
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor,
		`{"nome":"Acesso ao e-mail","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`","ativa":false}`)
	lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/ofertas", gestor, "").Body.String()
	if !strings.Contains(lista, oferta.ID) {
		t.Fatalf("lista de ofertas: %s", lista)
	}
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/ofertas/"+uuid.NewString(), gestor, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/ofertas/x", gestor, "")

	h.setModule(admin, "produtos", false)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Com produto desligado","unidade_id":"`+o.a+`","produto_id":"`+produto.ID+`"}`)
	semProduto := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Pedido geral","unidade_id":"`+o.a+`"}`))
	if semProduto.ID == "" {
		t.Fatal("oferta sem produto")
	}
	h.setModule(admin, "produtos", true)

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/equipes", gestor,
		`{"nome":"N1","unidade_id":"`+o.a+`","membros":["`+uuid.NewString()+`"]}`)
	equipe := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", gestor,
		`{"nome":"Nível 1","unidade_id":"`+o.a+`","membros":["`+uid.String()+`"]}`))
	if outboxCount(t, h, "atendimento.equipe.criada", equipe.ID) != 1 {
		t.Fatal("evento da equipe")
	}
	h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/equipes/"+equipe.ID, gestor, "")
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/equipes/"+equipe.ID, gestor,
		`{"nome":"Nível 1","unidade_id":"`+o.a+`","membros":["`+uid.String()+`"]}`)
	if outboxCount(t, h, "atendimento.equipe.atualizada", equipe.ID) != 0 {
		t.Fatal("edição vazia da equipe")
	}
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/equipes/"+equipe.ID, gestor,
		`{"nome":"Suporte","unidade_id":"`+o.a+`","membros":[]}`)
	equipes := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/equipes", gestor, "").Body.String()
	if !strings.Contains(equipes, equipe.ID) {
		t.Fatalf("lista de equipes: %s", equipes)
	}

	cands := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/candidatos?unidade_id="+o.a, gestor, "").Body.String()
	if !strings.Contains(cands, uid.String()) {
		t.Fatalf("candidatos da unidade: %s", cands)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/candidatos?unidade_id="+o.b, gestor, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/candidatos", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/candidatos?unidade_id=x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/ofertas/x", gestor, `{"nome":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor, `{`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/atendimento/ofertas/"+uuid.NewString(), gestor, `{"nome":"Ausente","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/equipes", gestor, `{`)
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/equipes/x", gestor, "")
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/equipes/x", gestor, `{"nome":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/equipes/"+equipe.ID, gestor, `{`)
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/equipes/"+uuid.NewString(), gestor, "")
}
