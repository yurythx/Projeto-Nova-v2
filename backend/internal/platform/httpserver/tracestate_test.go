package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// O tracestate do W3C TraceContext volta na resposta quando o cliente envia.
func TestRequestIDPropagaTracestate(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TracestateHeader, "fornecedor=abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get(TracestateHeader); got != "fornecedor=abc" {
		t.Fatalf("tracestate = %q", got)
	}
}
