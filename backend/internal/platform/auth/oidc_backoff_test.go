package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
)

// O discovery com issuer fora do ar: tentativas mínimas, contexto já
// cancelado e cancelamento durante a espera entre tentativas.
func TestBuildStateDiscoveryFalhaEAborta(t *testing.T) {
	cfg := config.KeycloakConfig{IssuerURL: "http://127.0.0.1:1/realms/x"}

	if _, err := buildState(context.Background(), cfg, 0); err == nil || !strings.Contains(err.Error(), "discovery failed") {
		t.Fatalf("maxAttempts <= 0 vira uma tentativa: %v", err)
	}

	cancelado, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := buildState(cancelado, cfg, 3); err == nil || !strings.Contains(err.Error(), "discovery aborted:") {
		t.Fatalf("contexto cancelado: %v", err)
	}

	curto, cancel2 := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel2()
	if _, err := buildState(curto, cfg, 3); err == nil || !strings.Contains(err.Error(), "during backoff") {
		t.Fatalf("cancelado na espera: %v", err)
	}
}
