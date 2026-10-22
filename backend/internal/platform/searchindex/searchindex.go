// Package searchindex mantém o índice da Busca Global no Typesense
// (ADR 031) e consulta esse índice.
//
// Quem alimenta o índice são os plugins cujo Provider implementa
// search.Indexer (só conteúdo público). O worker roda o Engine: carga
// completa de cada módulo no boot e a cada intervalo, e — ao chegar um
// evento do módulo pelo RabbitMQ — uma nova carga dele, agrupada por um
// curto atraso. Cada carga grava os documentos com uma geração nova e
// apaga os da geração anterior: o que saiu do módulo sai do índice.
//
// A API só consulta (Search). Se o Typesense falhar, a Busca Global volta
// à busca do Postgres de cada módulo.
package searchindex

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/domain/events"
	"github.com/yurythx/projeto-nova-v2/internal/platform/messaging"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
	"github.com/yurythx/projeto-nova-v2/internal/platform/typesense"
)

// batchSize é o tamanho de cada lote do import.
const batchSize = 500

// Backend é o que o Engine usa do Typesense (o *typesense.Client).
type Backend interface {
	EnsureCollection(ctx context.Context, s typesense.Schema) (bool, error)
	Import(ctx context.Context, collection string, docs []map[string]any) error
	DeleteByFilter(ctx context.Context, collection, filter string) (int, error)
	Search(ctx context.Context, collection string, p typesense.SearchParams) (typesense.SearchResult, error)
}

// Engine indexa e consulta uma coleção.
type Engine struct {
	ts         Backend
	collection string
	logger     *slog.Logger
	sources    func() []search.Indexer
	interval   time.Duration
	debounce   time.Duration
	dirty      chan string
	now        func() time.Time
}

// New cria o Engine. sources devolve os Indexers dos módulos ATIVOS.
func New(ts Backend, collection string, interval time.Duration, logger *slog.Logger, sources func() []search.Indexer) *Engine {
	return &Engine{
		ts: ts, collection: collection, logger: logger, sources: sources,
		interval: interval, debounce: 2 * time.Second, dirty: make(chan string, 64), now: time.Now,
	}
}

// Schema é o schema da coleção. Mudar um campo pede um nome de coleção
// novo (TYPESENSE_COLLECTION): a coleção nova nasce vazia e recebe a carga
// completa.
func (e *Engine) Schema() typesense.Schema {
	noIndex := false
	return typesense.Schema{Name: e.collection, Fields: []typesense.Field{
		{Name: "module", Type: "string", Facet: true},
		{Name: "type", Type: "string", Facet: true},
		{Name: "ref", Type: "string"},
		{Name: "title", Type: "string"},
		{Name: "body", Type: "string", Optional: true},
		{Name: "url", Type: "string", Optional: true, Index: &noIndex},
		{Name: "updated_at", Type: "int64", Optional: true},
		{Name: "geracao", Type: "int64"},
	}}
}

// Indexers filtra, entre os providers, os que alimentam o índice.
func Indexers(providers []search.Provider) []search.Indexer {
	var out []search.Indexer
	for _, p := range providers {
		if idx, ok := p.(search.Indexer); ok {
			out = append(out, idx)
		}
	}
	return out
}

// QueueName é a fila dos eventos que disparam a recarga dos módulos
// indexados.
const QueueName = "nexus.search.index"

// Queue declara QueueName ligada aos eventos dos módulos indexados.
func Queue(modules []string) messaging.QueueSpec {
	keys := make([]string, len(modules))
	for i, m := range modules {
		keys[i] = m + ".#"
	}
	return messaging.QueueSpec{Name: QueueName, DLQName: QueueName + ".dlq", RoutingKeys: keys}
}

// Handle é o handler da fila: marca o módulo do evento para recarga.
func (e *Engine) Handle(_ context.Context, ev events.Event) error {
	module, _, _ := strings.Cut(ev.Type, ".")
	e.Trigger(module)
	return nil
}

// Trigger pede a recarga de um módulo. Não bloqueia: com a fila interna
// cheia, a carga periódica cobre.
func (e *Engine) Trigger(module string) {
	select {
	case e.dirty <- module:
	default:
		e.logger.Warn("searchindex: recarga descartada (fila cheia) — fica para a carga periódica", slog.String("module", module))
	}
}

// Run mantém o índice até ctx acabar. Falha (para o supervisor reiniciar)
// só se a coleção não puder ser garantida no início.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.SyncAll(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	pending := map[string]bool{}
	var due <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := e.SyncAll(ctx); err != nil {
				e.logger.Error("searchindex: carga periódica falhou", slog.Any("error", err))
			}
		case m := <-e.dirty:
			pending[m] = true
			if due == nil {
				due = time.After(e.debounce)
			}
		case <-due:
			for _, idx := range e.sources() {
				if pending[idx.Module()] {
					e.syncLogged(ctx, idx)
				}
			}
			pending, due = map[string]bool{}, nil
		}
	}
}

// SyncAll garante a coleção e recarrega todos os módulos ativos. Erro só
// quando a coleção não pode ser garantida; a falha de um módulo é
// registrada e não impede os demais.
func (e *Engine) SyncAll(ctx context.Context) error {
	if _, err := e.ts.EnsureCollection(ctx, e.Schema()); err != nil {
		return fmt.Errorf("searchindex: garantir a coleção %s: %w", e.collection, err)
	}
	for _, idx := range e.sources() {
		e.syncLogged(ctx, idx)
	}
	return nil
}

func (e *Engine) syncLogged(ctx context.Context, idx search.Indexer) {
	start := time.Now()
	n, err := e.Sync(ctx, idx)
	if err != nil {
		e.logger.Error("searchindex: carga do módulo falhou", slog.String("module", idx.Module()), slog.Any("error", err))
		return
	}
	e.logger.Info("searchindex: módulo indexado", slog.String("module", idx.Module()), slog.Int("documentos", n), slog.Duration("duracao", time.Since(start)))
}

// Sync grava todo o conteúdo de um módulo numa geração nova e apaga o que
// sobrou da anterior. Devolve quantos documentos foram gravados.
func (e *Engine) Sync(ctx context.Context, idx search.Indexer) (int, error) {
	module := idx.Module()
	gen := e.now().UnixNano()
	batch := make([]map[string]any, 0, batchSize)
	flush := func() error {
		err := e.ts.Import(ctx, e.collection, batch)
		batch = batch[:0]
		return err
	}
	n := 0
	err := idx.Documents(ctx, func(d search.Document) error {
		batch = append(batch, document(module, d, gen))
		n++
		if len(batch) == batchSize {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		// Sem apagar nada: o índice fica com a carga anterior (mais o que
		// já entrou desta) até a próxima.
		return n, fmt.Errorf("searchindex: carga de %s: %w", module, err)
	}
	filter := "module:=" + typesense.Quote(module) + " && geracao:<" + strconv.FormatInt(gen, 10)
	if _, err := e.ts.DeleteByFilter(ctx, e.collection, filter); err != nil {
		return n, fmt.Errorf("searchindex: limpeza de %s: %w", module, err)
	}
	return n, nil
}

func document(module string, d search.Document, gen int64) map[string]any {
	doc := map[string]any{
		"id": module + ":" + d.Type + ":" + d.ID, "module": module, "type": d.Type, "ref": d.ID,
		"title": d.Title, "body": d.Body, "url": d.URL, "geracao": gen,
	}
	if d.UpdatedAt != nil {
		doc["updated_at"] = d.UpdatedAt.Unix()
	}
	return doc
}

// Search consulta o índice nos módulos informados. A pontuação é a do
// Typesense normalizada para 0..1 (1 = melhor resultado da consulta).
func (e *Engine) Search(ctx context.Context, q string, modules []string, limit int) ([]search.Result, error) {
	res, err := e.ts.Search(ctx, e.collection, typesense.SearchParams{
		Q: q, QueryBy: "title,body", Weights: "3,1", FilterBy: typesense.In("module", modules), PerPage: limit,
	})
	if err != nil {
		return nil, err
	}
	var best uint64
	for _, h := range res.Hits {
		best = max(best, h.TextMatch)
	}
	out := make([]search.Result, 0, len(res.Hits))
	for _, h := range res.Hits {
		r := search.Result{
			Module: str(h.Document, "module"), Type: str(h.Document, "type"), ID: str(h.Document, "ref"),
			Title: str(h.Document, "title"), Snippet: modkit.Snippet(str(h.Document, "body"), 180), URL: str(h.Document, "url"),
		}
		if best > 0 {
			r.Score = float64(h.TextMatch) / float64(best)
		}
		if ts, ok := h.Document["updated_at"].(float64); ok {
			t := time.Unix(int64(ts), 0).UTC()
			r.UpdatedAt = &t
		}
		out = append(out, r)
	}
	return out, nil
}

func str(doc map[string]any, key string) string {
	s, _ := doc[key].(string)
	return s
}
