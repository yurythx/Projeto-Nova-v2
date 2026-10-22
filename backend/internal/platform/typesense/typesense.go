// Package typesense é o cliente HTTP mínimo do Typesense usado pela busca
// (ADR 031). Corrige o que levou à remoção no ADR 015: a coleção é criada
// pelo próprio cliente (EnsureCollection), todo valor que entra num
// filter_by passa por Quote (nunca interpolação crua) e a chave de API é
// obrigatória — sem default.
package typesense

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound: a coleção (ou o documento) não existe.
var ErrNotFound = errors.New("typesense: não encontrado")

// Field é um campo do schema de uma coleção.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Facet    bool   `json:"facet,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Index    *bool  `json:"index,omitempty"`
	Sort     bool   `json:"sort,omitempty"`
}

// Schema descreve uma coleção.
type Schema struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// SearchParams são os parâmetros de uma busca. FilterBy deve ser montado
// com Quote/In.
type SearchParams struct {
	Q        string
	QueryBy  string
	Weights  string
	FilterBy string
	PerPage  int
	// Page começa em 1 (0 = primeira página).
	Page int
	// FacetBy lista os campos (com facet no schema) a contar.
	FacetBy        string
	MaxFacetValues int
	SortBy         string
	// IncludeFields/ExcludeFields limitam o que volta em cada documento.
	IncludeFields string
	ExcludeFields string
	// HighlightFields: os campos de onde sai o trecho destacado.
	HighlightFields string
}

// Hit é um documento encontrado, com a pontuação de texto do Typesense.
type Hit struct {
	Document   map[string]any `json:"document"`
	TextMatch  uint64         `json:"text_match"`
	Highlights []Highlight    `json:"highlights"`
}

// Highlight é o trecho de um campo com os termos encontrados entre <mark>.
type Highlight struct {
	Field   string `json:"field"`
	Snippet string `json:"snippet"`
}

// FacetCount conta os valores de um campo facetado.
type FacetCount struct {
	FieldName string `json:"field_name"`
	Counts    []struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	} `json:"counts"`
}

// SearchResult é a resposta de uma busca.
type SearchResult struct {
	Found       int          `json:"found"`
	Hits        []Hit        `json:"hits"`
	FacetCounts []FacetCount `json:"facet_counts"`
	SearchTime  int          `json:"search_time_ms"`
}

// Client fala com um nó do Typesense.
type Client struct {
	base   string
	apiKey string
	http   *http.Client
}

// New cria o cliente. baseURL é http(s)://host:porta; a chave vai no
// cabeçalho X-TYPESENSE-API-KEY de toda requisição.
func New(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{base: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: &http.Client{Timeout: timeout}}
}

// Quote escapa um valor para o filter_by: entre crases, com crase interna
// escapada — vírgulas, "&&" e espaços deixam de ser sintaxe.
func Quote(v string) string {
	return "`" + strings.ReplaceAll(v, "`", "\\`") + "`"
}

// In monta "campo:[`a`,`b`]" com cada valor escapado.
func In(field string, values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = Quote(v)
	}
	return field + ":[" + strings.Join(quoted, ",") + "]"
}

// Health responde nil se o nó está pronto para atender.
func (c *Client) Health(ctx context.Context) error {
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.do(ctx, http.MethodGet, "/health", nil, nil, &out); err != nil {
		return err
	}
	if !out.OK {
		return errors.New("typesense: nó não está pronto")
	}
	return nil
}

// EnsureCollection cria a coleção se ela não existir. Devolve true quando
// criou (a coleção nasce vazia: quem chama dispara a carga completa).
func (c *Client) EnsureCollection(ctx context.Context, s Schema) (bool, error) {
	err := c.do(ctx, http.MethodGet, "/collections/"+url.PathEscape(s.Name), nil, nil, nil)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return false, err
	}
	body, _ := json.Marshal(s)
	if err := c.do(ctx, http.MethodPost, "/collections", nil, body, nil); err != nil {
		return false, err
	}
	return true, nil
}

// Import grava (upsert) os documentos em lote. O Typesense responde uma
// linha JSON por documento; qualquer linha com falha vira erro.
func (c *Client) Import(ctx context.Context, collection string, docs []map[string]any) error {
	if len(docs) == 0 {
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, d := range docs {
		if err := enc.Encode(d); err != nil {
			return fmt.Errorf("typesense: documento inválido: %w", err)
		}
	}
	var raw []byte
	q := url.Values{"action": {"upsert"}}
	if err := c.do(ctx, http.MethodPost, "/collections/"+url.PathEscape(collection)+"/documents/import", q, buf.Bytes(), &raw); err != nil {
		return err
	}
	failed, first := 0, ""
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var line struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil || !line.Success {
			if failed == 0 {
				first = line.Error
			}
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("typesense: %d de %d documentos recusados (primeiro erro: %s)", failed, len(docs), first)
	}
	return nil
}

// DeleteByFilter apaga os documentos que atendem ao filtro e devolve
// quantos saíram.
func (c *Client) DeleteByFilter(ctx context.Context, collection, filter string) (int, error) {
	var out struct {
		NumDeleted int `json:"num_deleted"`
	}
	q := url.Values{"filter_by": {filter}}
	if err := c.do(ctx, http.MethodDelete, "/collections/"+url.PathEscape(collection)+"/documents", q, nil, &out); err != nil {
		return 0, err
	}
	return out.NumDeleted, nil
}

// Search pesquisa a coleção.
func (c *Client) Search(ctx context.Context, collection string, p SearchParams) (SearchResult, error) {
	q := url.Values{"q": {p.Q}, "query_by": {p.QueryBy}}
	if p.Weights != "" {
		q.Set("query_by_weights", p.Weights)
	}
	if p.FilterBy != "" {
		q.Set("filter_by", p.FilterBy)
	}
	if p.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(p.PerPage))
	}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.FacetBy != "" {
		q.Set("facet_by", p.FacetBy)
	}
	if p.MaxFacetValues > 0 {
		q.Set("max_facet_values", strconv.Itoa(p.MaxFacetValues))
	}
	for k, v := range map[string]string{"sort_by": p.SortBy, "include_fields": p.IncludeFields, "exclude_fields": p.ExcludeFields,
		"highlight_fields": p.HighlightFields} {
		if v != "" {
			q.Set(k, v)
		}
	}
	var out SearchResult
	err := c.do(ctx, http.MethodGet, "/collections/"+url.PathEscape(collection)+"/documents/search", q, nil, &out)
	return out, err
}

// do executa a requisição. out *[]byte recebe o corpo cru; outro ponteiro
// recebe o JSON decodificado; nil descarta.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, body []byte, out any) error {
	target := c.base + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("typesense: %w", err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("typesense: %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if err != nil {
		return fmt.Errorf("typesense: ler resposta: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 300 {
		// A mensagem do Typesense diz o que está errado na requisição; o
		// corpo nunca ecoa a chave.
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		return fmt.Errorf("typesense: %s %s: HTTP %d: %s", method, path, resp.StatusCode, e.Message)
	}
	switch o := out.(type) {
	case nil:
		return nil
	case *[]byte:
		*o = data
		return nil
	default:
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("typesense: resposta inválida: %w", err)
		}
		return nil
	}
}
