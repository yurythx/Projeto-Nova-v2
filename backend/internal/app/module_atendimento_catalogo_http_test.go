package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Catálogo em blocos (ADR 078): a administração cria a categoria, a gestão
// da unidade põe a oferta nela, e quem pede vê as duas na central.
func TestAtendimentoCatalogoHTTP(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	o := h.orgEscopo(t)
	_, gestor := h.pessoaEm(t, o.a, "atendimento:manage")
	_, servidor := h.pessoaEm(t, o.a, "atendimento:abrir")
	nome := "Acesso " + uuid.NewString()[:6]

	cat := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/categorias", admin,
		`{"nome":"`+nome+`","descricao":"Contas e senhas","icone":"key-round","ordem":1}`))
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/categorias", gestor, `{"nome":"Outra"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/categorias", admin, `{"nome":""}`)
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/categorias/"+cat.ID, admin, `{"nome":"`+nome+`","icone":"lock"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, "/api/v1/atendimento/categorias/"+cat.ID, admin, `{}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/categorias/nao-e-uuid", admin, `{"nome":"X"}`)
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/atendimento/categorias/"+cat.ID, gestor, `{"nome":"X"}`)
	if body := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/categorias", gestor, "").Body.String(); !strings.Contains(body, nome) {
		t.Fatalf("gestão lista: %s", body)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/categorias", servidor, "")

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"Troca de senha","unidade_id":"`+o.a+`","categoria_id":"`+cat.ID+`","icone":"key","ordem":2}`))
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/ofertas", gestor,
		`{"nome":"X","unidade_id":"`+o.a+`","categoria_id":"nao-e-uuid"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor,
		`{"nome":"X","unidade_id":"`+o.a+`","categoria_id":"nao-e-uuid"}`)
	for _, c := range []struct{ path, quer string }{
		{"/api/v1/atendimento/catalogo/categorias", nome},
		{"/api/v1/atendimento/catalogo", `"categoria_id":"` + cat.ID + `"`},
	} {
		if body := h.expect(http.StatusOK, http.MethodGet, c.path, servidor, "").Body.String(); !strings.Contains(body, c.quer) {
			t.Fatalf("%s: %s", c.path, body)
		}
	}

	// Tirar da categoria e apagar.
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/ofertas/"+oferta.ID, gestor,
		`{"nome":"Troca de senha","unidade_id":"`+o.a+`","categoria_id":""}`)
	h.expect(http.StatusForbidden, http.MethodDelete, "/api/v1/atendimento/categorias/"+cat.ID, gestor, "")
	h.expect(http.StatusBadRequest, http.MethodDelete, "/api/v1/atendimento/categorias/nao-e-uuid", admin, "")
	h.expect(http.StatusNoContent, http.MethodDelete, "/api/v1/atendimento/categorias/"+cat.ID, admin, "")
	h.expect(http.StatusNotFound, http.MethodDelete, "/api/v1/atendimento/categorias/"+cat.ID, admin, "")
}
