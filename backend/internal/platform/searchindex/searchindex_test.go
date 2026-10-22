package searchindex

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/domain/events"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
	"github.com/yurythx/projeto-nova-v2/internal/platform/typesense"
)

// backend é um Typesense em memória: guarda os lotes, os filtros de
// limpeza e responde a busca com hits fixos.
type backend struct {
	mu        sync.Mutex
	ensureErr error
	importErr error
	deleteErr error
	searchErr error
	imported  [][]map[string]any
	deleted   []string
	hits      []typesense.Hit
	search    typesense.SearchParams
	ensures   int
}

func (b *backend) EnsureCollection(_ context.Context, s typesense.Schema) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensures++
	return true, b.ensureErr
}

func (b *backend) Import(_ context.Context, _ string, docs []map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.imported = append(b.imported, append([]map[string]any(nil), docs...))
	return b.importErr
}

func (b *backend) DeleteByFilter(_ context.Context, _ string, filter string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.deleted = append(b.deleted, filter)
	return 1, b.deleteErr
}

func (b *backend) Search(_ context.Context, _ string, p typesense.SearchParams) (typesense.SearchResult, error) {
	b.search = p
	return typesense.SearchResult{Found: len(b.hits), Hits: b.hits}, b.searchErr
}

func (b *backend) counts() (imports, deletes int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.imported), len(b.deleted)
}

type indexer struct {
	module string
	docs   int
	err    error
}

func (i indexer) Module() string { return i.module }
func (i indexer) Search(context.Context, auth.Identity, string, int) ([]search.Result, error) {
	return nil, nil
}
func (i indexer) Documents(_ context.Context, emit func(search.Document) error) error {
	now := time.Unix(1700000000, 0)
	for n := range i.docs {
		d := search.Document{Type: "t", ID: strings.Repeat("x", n+1), Title: "T", Body: "corpo", URL: "/u"}
		if n == 0 {
			d.UpdatedAt = &now
		}
		if err := emit(d); err != nil {
			return err
		}
	}
	return i.err
}

// plain é um Provider que não indexa.
type plain struct{}

func (plain) Module() string { return "wiki" }
func (plain) Search(context.Context, auth.Identity, string, int) ([]search.Result, error) {
	return nil, nil
}

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func engine(b *backend, sources ...search.Indexer) *Engine {
	e := New(b, "col", time.Hour, logger(), func() []search.Indexer { return sources })
	e.now = func() time.Time { return time.Unix(0, 42) }
	return e
}

func TestSchemaIndexersEQueue(t *testing.T) {
	s := engine(&backend{}).Schema()
	if s.Name != "col" || len(s.Fields) != 8 {
		t.Fatalf("schema: %+v", s)
	}
	idx := Indexers([]search.Provider{plain{}, indexer{module: "wiki"}})
	if len(idx) != 1 || idx[0].Module() != "wiki" {
		t.Fatalf("só o Indexer entra: %v", idx)
	}
	q := Queue([]string{"wiki", "catalog"})
	if q.Name != QueueName || q.DLQName != QueueName+".dlq" || strings.Join(q.RoutingKeys, ",") != "wiki.#,catalog.#" {
		t.Fatalf("fila: %+v", q)
	}
}

func TestSyncGravaEmLotesEApagaGeracaoAnterior(t *testing.T) {
	b := &backend{}
	n, err := engine(b).Sync(context.Background(), indexer{module: "wiki", docs: batchSize + 1})
	if err != nil || n != batchSize+1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(b.imported) != 2 || len(b.imported[0]) != batchSize || len(b.imported[1]) != 1 {
		t.Fatalf("esperado um lote cheio e um resto, veio %d lotes", len(b.imported))
	}
	first := b.imported[0][0]
	if first["id"] != "wiki:t:x" || first["geracao"] != int64(42) || first["updated_at"] != int64(1700000000) {
		t.Fatalf("documento: %+v", first)
	}
	if _, ok := b.imported[0][1]["updated_at"]; ok {
		t.Fatal("sem UpdatedAt o campo não vai")
	}
	if len(b.deleted) != 1 || b.deleted[0] != "module:=`wiki` && geracao:<42" {
		t.Fatalf("limpeza: %v", b.deleted)
	}
}

func TestSyncFalhas(t *testing.T) {
	ctx := context.Background()
	// Falha do módulo: nada é apagado.
	b := &backend{}
	if _, err := engine(b).Sync(ctx, indexer{module: "wiki", docs: 1, err: errors.New("banco")}); err == nil || len(b.deleted) != 0 {
		t.Fatalf("falha do módulo não pode limpar o índice: err=%v deleted=%v", err, b.deleted)
	}
	// Falha no lote cheio (dentro do emit) e no resto (depois).
	b = &backend{importErr: errors.New("import")}
	if _, err := engine(b).Sync(ctx, indexer{module: "wiki", docs: batchSize}); err == nil || len(b.deleted) != 0 {
		t.Fatalf("falha no import: err=%v", err)
	}
	if _, err := engine(b).Sync(ctx, indexer{module: "wiki", docs: 1}); err == nil {
		t.Fatal("falha no último lote deveria subir")
	}
	b = &backend{deleteErr: errors.New("delete")}
	if _, err := engine(b).Sync(ctx, indexer{module: "wiki", docs: 1}); err == nil || !strings.Contains(err.Error(), "limpeza") {
		t.Fatalf("falha na limpeza: %v", err)
	}
}

func TestSyncAll(t *testing.T) {
	ctx := context.Background()
	b := &backend{}
	e := engine(b, indexer{module: "wiki", docs: 1}, indexer{module: "ruim", err: errors.New("x")})
	if err := e.SyncAll(ctx); err != nil {
		t.Fatal(err)
	}
	// "ruim" falha sem entregar nada: só o Wiki importa e limpa.
	if imports, deletes := b.counts(); imports != 1 || deletes != 1 {
		t.Fatalf("a falha de um módulo não impede o outro: imports=%d deletes=%d", imports, deletes)
	}
	if err := engine(&backend{ensureErr: errors.New("fora")}).SyncAll(ctx); err == nil {
		t.Fatal("sem coleção a carga falha")
	}
}

func TestRun(t *testing.T) {
	if err := engine(&backend{ensureErr: errors.New("fora")}).Run(context.Background()); err == nil {
		t.Fatal("Run falha (para o supervisor reiniciar) se a coleção não sobe")
	}

	b := &backend{}
	e := engine(b, indexer{module: "wiki", docs: 1}, indexer{module: "catalog", docs: 1})
	e.interval, e.debounce = 30*time.Millisecond, 10*time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.Run(ctx) }()

	// Carga inicial + periódica.
	waitFor(t, func() bool { _, d := b.counts(); return d >= 4 })
	// Evento do Wiki: recarga só dele, agrupando os repetidos.
	b.mu.Lock()
	b.deleted, b.ensureErr = nil, errors.New("periódica falha e só registra")
	prev := b.ensures
	b.mu.Unlock()
	waitFor(t, func() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.ensures >= prev+2 })
	for range 3 {
		if err := e.Handle(ctx, events.Event{Type: "wiki.workflow.created"}); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		for _, f := range b.deleted {
			if strings.Contains(f, "`wiki`") {
				return true
			}
		}
		return false
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTriggerComFilaCheia(t *testing.T) {
	e := engine(&backend{})
	for range cap(e.dirty) + 1 {
		e.Trigger("wiki")
	}
	if len(e.dirty) != cap(e.dirty) {
		t.Fatal("excedente descartado sem bloquear")
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	b := &backend{hits: []typesense.Hit{
		{TextMatch: 50, Document: map[string]any{"module": "wiki", "type": "pagina", "ref": "01.03", "title": "Página", "body": strings.Repeat("a ", 200), "url": "/wiki/01.03", "updated_at": float64(1700000000)}},
		{TextMatch: 100, Document: map[string]any{"module": "wiki", "type": "modelo", "ref": "m1", "title": "Modelo"}},
	}}
	res, err := engine(b).Search(ctx, "página", []string{"wiki"}, 5)
	if err != nil || len(res) != 2 {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if b.search.FilterBy != "module:[`wiki`]" || b.search.PerPage != 5 || b.search.QueryBy != "title,body" {
		t.Fatalf("parâmetros: %+v", b.search)
	}
	if res[0].Score != 0.5 || res[1].Score != 1 || res[0].ID != "01.03" || res[0].URL != "/wiki/01.03" {
		t.Fatalf("pontuação normalizada e campos: %+v", res)
	}
	if res[0].UpdatedAt == nil || res[0].UpdatedAt.Unix() != 1700000000 || res[1].UpdatedAt != nil {
		t.Fatalf("updated_at: %+v", res)
	}
	if len([]rune(res[0].Snippet)) != 181 {
		t.Fatalf("trecho cortado em 180: %d", len([]rune(res[0].Snippet)))
	}

	zero := &backend{hits: []typesense.Hit{{Document: map[string]any{}}}}
	if res, _ := engine(zero).Search(ctx, "x", []string{"wiki"}, 5); res[0].Score != 0 {
		t.Fatal("sem text_match a pontuação é 0")
	}
	if _, err := engine(&backend{searchErr: errors.New("fora")}).Search(ctx, "x", []string{"wiki"}, 5); err == nil {
		t.Fatal("erro do Typesense sobe")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condição não atingida a tempo")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
