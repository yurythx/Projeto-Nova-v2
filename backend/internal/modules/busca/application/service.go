// Package application implementa a Busca Global: consulta em paralelo os
// providers dos módulos ATIVOS (o Kernel só entrega os ativos), cada um
// com seu próprio prazo. Um provider lento ou com erro não derruba a busca
// — o resultado sai parcial e informa quais módulos falharam.
package application

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/metrics"
	"github.com/yurythx/projeto-nova-v2/internal/platform/search"
)

// ProviderSource devolve os providers dos módulos ativos neste instante.
type ProviderSource func() []search.Provider

// Index é o índice da busca (Typesense, ADR 031). Responde pelos módulos
// cujo provider é um search.Indexer; sem índice, todos vão ao Postgres.
type Index interface {
	Search(ctx context.Context, q string, modules []string, limit int) ([]search.Result, error)
}

// Response é o resultado agregado.
type Response struct {
	Query    string          `json:"query"`
	Results  []search.Result `json:"results"`
	Modules  []string        `json:"modules"`
	Degraded []string        `json:"degraded"`
	TookMS   int64           `json:"took_ms"`
}

// Service agrega os providers.
type Service struct {
	source  ProviderSource
	index   Index
	timeout time.Duration
	logger  *slog.Logger
}

// NewService cria o serviço.
func NewService(source ProviderSource, logger *slog.Logger) *Service {
	return &Service{source: source, timeout: 2 * time.Second, logger: logger}
}

// WithIndex liga o índice da busca.
func (s *Service) WithIndex(idx Index) *Service {
	s.index = idx
	return s
}

// fromIndex consulta o índice pelos módulos indexados. ok=false manda
// esses módulos de volta ao Postgres: índice com erro, ou sem nenhum
// resultado (o índice pode ainda não ter recebido a carga — o Postgres
// confirma).
func (s *Service) fromIndex(ctx context.Context, query string, indexed []search.Provider, limit int) ([]search.Result, bool) {
	modules := make([]string, len(indexed))
	for i, p := range indexed {
		modules[i] = p.Module()
	}
	ictx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	res, err := s.index.Search(ictx, query, modules, limit)
	switch {
	case err != nil:
		s.logger.Warn("busca: índice indisponível — usando a busca do Postgres", slog.Any("error", err))
		metrics.SearchIndexFallbackTotal.WithLabelValues("erro").Inc()
		return nil, false
	case len(res) == 0:
		metrics.SearchIndexFallbackTotal.WithLabelValues("vazio").Inc()
		return nil, false
	}
	return res, true
}

// Search executa a busca global.
func (s *Service) Search(ctx context.Context, identity auth.Identity, query string, module string, limit int) (Response, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return Response{}, apperrors.Validation("digite ao menos 2 caracteres")
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	start := time.Now()
	providers := s.source()

	type outcome struct {
		module  string
		results []search.Result
		err     error
	}
	modules := []string{}
	var run, indexed []search.Provider
	for _, p := range providers {
		if module != "" && p.Module() != module {
			continue
		}
		modules = append(modules, p.Module())
		if _, ok := p.(search.Indexer); ok && s.index != nil {
			indexed = append(indexed, p)
			continue
		}
		run = append(run, p)
	}
	resp := Response{Query: query, Results: []search.Result{}, Modules: modules, Degraded: []string{}}
	if len(indexed) > 0 {
		if res, ok := s.fromIndex(ctx, query, indexed, limit); ok {
			resp.Results = append(resp.Results, res...)
		} else {
			run = append(run, indexed...)
		}
	}

	ch := make(chan outcome, len(run))
	var wg sync.WaitGroup
	for _, p := range run {
		wg.Add(1)
		go func(p search.Provider) {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(ctx, s.timeout)
			defer cancel()
			res, err := p.Search(pctx, identity, query, limit)
			ch <- outcome{module: p.Module(), results: res, err: err}
		}(p)
	}
	wg.Wait()
	close(ch)

	for o := range ch {
		if o.err != nil {
			s.logger.Warn("busca: provider falhou — resultado parcial", slog.String("module", o.module), slog.Any("error", o.err))
			resp.Degraded = append(resp.Degraded, o.module)
			continue
		}
		resp.Results = append(resp.Results, o.results...)
	}
	sort.SliceStable(resp.Results, func(i, j int) bool { return resp.Results[i].Score > resp.Results[j].Score })
	if len(resp.Results) > limit*2 {
		resp.Results = resp.Results[:limit*2]
	}
	sort.Strings(resp.Degraded)
	resp.TookMS = time.Since(start).Milliseconds()
	return resp, nil
}
