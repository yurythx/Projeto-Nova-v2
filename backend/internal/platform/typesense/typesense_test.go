package typesense

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestQuoteEIn(t *testing.T) {
	if got := Quote("mod`x, y && z"); got != "`mod\\`x, y && z`" {
		t.Fatalf("Quote = %s", got)
	}
	if got := In("module", []string{"wiki", "a`b"}); got != "module:[`wiki`,`a\\`b`]" {
		t.Fatalf("In = %s", got)
	}
}

// fake responde pelo handler e confere a chave de API em toda chamada.
func fake(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-TYPESENSE-API-KEY") != "chave-de-teste-123456" {
			t.Errorf("chave de API ausente em %s %s", r.Method, r.URL.Path)
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL+"/", "chave-de-teste-123456", time.Second)
}

func TestHealth(t *testing.T) {
	ctx := context.Background()
	if err := fake(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"ok":true}`) }).Health(ctx); err != nil {
		t.Fatal(err)
	}
	if err := fake(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"ok":false}`) }).Health(ctx); err == nil {
		t.Fatal("nó não pronto deveria falhar")
	}
	if err := fake(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }).Health(ctx); err == nil {
		t.Fatal("HTTP 503 deveria falhar")
	}
}

func TestEnsureCollection(t *testing.T) {
	ctx := context.Background()
	s := Schema{Name: "c", Fields: []Field{{Name: "title", Type: "string"}}}

	created, err := fake(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"name":"c"}`) }).EnsureCollection(ctx, s)
	if err != nil || created {
		t.Fatalf("coleção existente: created=%v err=%v", created, err)
	}

	var posted string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		posted = string(b)
		_, _ = io.WriteString(w, `{}`)
	})
	if created, err = c.EnsureCollection(ctx, s); err != nil || !created || !strings.Contains(posted, `"name":"c"`) {
		t.Fatalf("criação: created=%v err=%v corpo=%s", created, err, posted)
	}

	if _, err := fake(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }).EnsureCollection(ctx, s); err == nil {
		t.Fatal("erro na consulta da coleção deveria subir")
	}
	c = fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"campo inválido"}`)
	})
	if _, err := c.EnsureCollection(ctx, s); err == nil || !strings.Contains(err.Error(), "campo inválido") {
		t.Fatalf("erro na criação deveria trazer a mensagem do Typesense: %v", err)
	}
}

func TestImport(t *testing.T) {
	ctx := context.Background()
	if err := New("http://127.0.0.1:1", "k", time.Second).Import(ctx, "c", nil); err != nil {
		t.Fatalf("lote vazio não chama o servidor: %v", err)
	}
	var query, body string
	ok := fake(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = io.WriteString(w, "{\"success\":true}\n{\"success\":true}\n")
	})
	if err := ok.Import(ctx, "c", []map[string]any{{"id": "1"}, {"id": "2"}}); err != nil {
		t.Fatal(err)
	}
	if query != "action=upsert" || strings.Count(body, "\n") != 2 {
		t.Fatalf("import deve ser upsert em JSONL: query=%s corpo=%q", query, body)
	}

	parcial := fake(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "{\"success\":true}\n{\"success\":false,\"error\":\"campo ausente\"}\nlixo\n")
	})
	err := parcial.Import(ctx, "c", []map[string]any{{"id": "1"}, {"id": "2"}, {"id": "3"}})
	if err == nil || !strings.Contains(err.Error(), "2 de 3") || !strings.Contains(err.Error(), "campo ausente") {
		t.Fatalf("linhas recusadas devem virar erro: %v", err)
	}
	if err := ok.Import(ctx, "c", []map[string]any{{"x": make(chan int)}}); err == nil {
		t.Fatal("documento não serializável deveria falhar")
	}
	if err := fake(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }).Import(ctx, "c", []map[string]any{{"id": "1"}}); err == nil {
		t.Fatal("HTTP 500 deveria falhar")
	}
}

func TestDeleteByFilterESearch(t *testing.T) {
	ctx := context.Background()
	var got string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path + "?" + r.URL.Query().Encode()
		if r.Method == http.MethodDelete {
			_, _ = io.WriteString(w, `{"num_deleted":3}`)
			return
		}
		_, _ = io.WriteString(w, `{"found":1,"hits":[{"document":{"id":"a"},"text_match":7}]}`)
	})
	n, err := c.DeleteByFilter(ctx, "c", "geracao:<5")
	if err != nil || n != 3 || !strings.Contains(got, "filter_by=geracao") {
		t.Fatalf("delete: n=%d err=%v req=%s", n, err, got)
	}
	res, err := c.Search(ctx, "c", SearchParams{Q: "x", QueryBy: "title", Weights: "2", FilterBy: "module:=`a`", PerPage: 5})
	if err != nil || res.Found != 1 || res.Hits[0].TextMatch != 7 {
		t.Fatalf("search: %+v err=%v", res, err)
	}
	for _, p := range []string{"q=x", "query_by=title", "query_by_weights=2", "filter_by=module", "per_page=5"} {
		if !strings.Contains(got, p) {
			t.Fatalf("parâmetro %s ausente em %s", p, got)
		}
	}
	if _, err := c.Search(ctx, "c", SearchParams{Q: "x", QueryBy: "title"}); err != nil || strings.Contains(got, "per_page") || strings.Contains(got, "filter_by") ||
		strings.Contains(got, "facet_by") || strings.Contains(got, "page=") || strings.Contains(got, "sort_by") {
		t.Fatalf("parâmetros opcionais não deveriam ir vazios: %s err=%v", got, err)
	}
	_, err = c.Search(ctx, "c", SearchParams{Q: "x", QueryBy: "title", Page: 2, FacetBy: "fase,ano", MaxFacetValues: 30, SortBy: "ano:desc",
		IncludeFields: "id,title", ExcludeFields: "texto", HighlightFields: "texto"})
	for _, p := range []string{"page=2", "facet_by=fase%2Cano", "max_facet_values=30", "sort_by=ano%3Adesc", "include_fields=id%2Ctitle",
		"exclude_fields=texto", "highlight_fields=texto"} {
		if err != nil || !strings.Contains(got, p) {
			t.Fatalf("parâmetro %s ausente em %s (%v)", p, got, err)
		}
	}
	if _, err := fake(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }).DeleteByFilter(ctx, "c", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404 deveria ser ErrNotFound: %v", err)
	}
}

func TestFalhasDeTransporte(t *testing.T) {
	ctx := context.Background()
	if err := New("http://exemplo.invalid\x7f", "k", time.Second).Health(ctx); err == nil {
		t.Fatal("URL inválida deveria falhar")
	}
	if err := New("http://127.0.0.1:1", "k", time.Second).Health(ctx); err == nil {
		t.Fatal("servidor fora do ar deveria falhar")
	}
	if err := fake(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{nao-json`) }).Health(ctx); err == nil {
		t.Fatal("resposta inválida deveria falhar")
	}
	corta := fake(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, `{"ok":`)
	})
	if err := corta.Health(ctx); err == nil {
		t.Fatal("corpo truncado deveria falhar")
	}
}

// TestTypesenseReal roda contra um Typesense de verdade (CI e
// desenvolvimento): TEST_TYPESENSE_URL e TEST_TYPESENSE_API_KEY.
func TestTypesenseReal(t *testing.T) {
	base := os.Getenv("TEST_TYPESENSE_URL")
	if base == "" {
		t.Skip("TEST_TYPESENSE_URL não definido")
	}
	ctx := context.Background()
	c := New(base, os.Getenv("TEST_TYPESENSE_API_KEY"), 5*time.Second)
	if err := c.Health(ctx); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("teste_%d", time.Now().UnixNano())
	s := Schema{Name: name, Fields: []Field{{Name: "module", Type: "string", Facet: true}, {Name: "title", Type: "string"}, {Name: "geracao", Type: "int64"}}}
	t.Cleanup(func() { _ = c.do(context.Background(), http.MethodDelete, "/collections/"+name, nil, nil, nil) })
	if created, err := c.EnsureCollection(ctx, s); err != nil || !created {
		t.Fatalf("criar: %v %v", created, err)
	}
	if created, err := c.EnsureCollection(ctx, s); err != nil || created {
		t.Fatalf("segunda chamada não recria: %v %v", created, err)
	}
	docs := []map[string]any{
		{"id": "1", "module": "wiki", "title": "Licitação — pregão eletrônico", "geracao": 1},
		{"id": "2", "module": "mod`x, y", "title": "Ofício && || pregão", "geracao": 1},
	}
	if err := c.Import(ctx, name, docs); err != nil {
		t.Fatal(err)
	}
	res, err := c.Search(ctx, name, SearchParams{Q: "pregao", QueryBy: "title", FilterBy: In("module", []string{"wiki"})})
	if err != nil || res.Found != 1 || res.Hits[0].Document["id"] != "1" {
		t.Fatalf("busca sem acento e filtrada: %+v %v", res, err)
	}
	res, err = c.Search(ctx, name, SearchParams{Q: "*", QueryBy: "title", FilterBy: "module:=" + Quote("mod`x, y")})
	if err != nil || res.Found != 1 {
		t.Fatalf("valor com crase, vírgula e && não pode quebrar o filtro: %+v %v", res, err)
	}
	res, err = c.Search(ctx, name, SearchParams{Q: "pregao", QueryBy: "title", FacetBy: "module", HighlightFields: "title", Page: 1, PerPage: 1})
	if err != nil || res.Found != 2 || len(res.Hits) != 1 || len(res.FacetCounts) != 1 || res.FacetCounts[0].FieldName != "module" ||
		len(res.FacetCounts[0].Counts) != 2 || len(res.Hits[0].Highlights) == 0 || !strings.Contains(res.Hits[0].Highlights[0].Snippet, "<mark>") {
		t.Fatalf("facetas e trechos: %+v %v", res, err)
	}
	if err := c.Import(ctx, name, []map[string]any{{"id": "3"}}); err == nil {
		t.Fatal("documento sem os campos obrigatórios deveria ser recusado")
	}
	if n, err := c.DeleteByFilter(ctx, name, "module:="+Quote("wiki")+" && geracao:<2"); err != nil || n != 1 {
		t.Fatalf("delete: %d %v", n, err)
	}
}
