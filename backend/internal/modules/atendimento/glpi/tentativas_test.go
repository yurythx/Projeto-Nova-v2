package glpi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Nos testes, a espera entre as tentativas é curta.
func init() { Espera = time.Millisecond }

// A leitura que falha pela rede ou com 5xx se repete; a recusa e a escrita,
// não.
func TestTentativas(t *testing.T) {
	ctx := context.Background()
	var chamadas, falhas atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas.Add(1)
		switch {
		case r.URL.Path == "/recusa":
			w.WriteHeader(http.StatusForbidden)
		case falhas.Add(-1) >= 0:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_, _ = w.Write([]byte(`[{"id":1}]`))
		}
	}))
	t.Cleanup(srv.Close)
	c := NovoCliente(srv.URL, "a", "u")
	var out []Entidade

	falhas.Store(2)
	if _, err := c.get(ctx, "/x", &out); err != nil || chamadas.Load() != 3 || len(out) != 1 {
		t.Fatalf("duas falhas e o acerto: %v %d", err, chamadas.Load())
	}
	chamadas.Store(0)
	falhas.Store(9)
	if _, err := c.get(ctx, "/x", &out); !errors.Is(err, ErrGLPI) || chamadas.Load() != int32(Tentativas) {
		t.Fatalf("desiste na última: %v %d", err, chamadas.Load())
	}
	chamadas.Store(0)
	if _, err := c.get(ctx, "/recusa", &out); !errors.Is(err, ErrGLPI) || chamadas.Load() != 1 {
		t.Fatalf("recusa não se repete: %v %d", err, chamadas.Load())
	}
	chamadas.Store(0)
	falhas.Store(9)
	if _, err := c.chamar(ctx, http.MethodPost, "/x", map[string]int{"a": 1}, nil); err == nil || chamadas.Load() != 1 {
		t.Fatalf("escrita não se repete: %v %d", err, chamadas.Load())
	}

	// Sem rede, repete e desiste; cancelado na espera, para na hora.
	fora := NovoCliente("http://127.0.0.1:1", "a", "u")
	if _, err := fora.get(ctx, "/x", &out); !errors.Is(err, errPassageiro) {
		t.Fatalf("sem rede: %v", err)
	}
	antes := Espera
	Espera = time.Hour
	t.Cleanup(func() { Espera = antes })
	falhas.Store(9)
	cancelado, cancelar := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelar()
	if _, err := c.get(cancelado, "/x", &out); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelado na espera: %v", err)
	}
}
