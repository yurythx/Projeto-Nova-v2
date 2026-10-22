package localauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func doRefresh(h *Handlers, id *auth.Identity) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	if id != nil {
		r = r.WithContext(auth.WithIdentity(r.Context(), *id))
	}
	rec := httptest.NewRecorder()
	h.Refresh(rec, r)
	return rec
}

func localIdentity(a *Account, authTime time.Time) *auth.Identity {
	return &auth.Identity{Subject: a.ID.String(), Username: a.Username, Source: auth.SourceLocal, AuthTime: authTime}
}

// A sessão local é renovada enquanto há uso: token novo, roles relidas da
// conta e last_seen_at atualizado.
func TestRefreshRenewsLocalSession(t *testing.T) {
	a := account(t)
	a.Roles = []string{"nexus-user", "nexus-admin"}
	store := newFakeStore(a)
	h := NewHandlers(store, testSigner(t), nil, nil, nil, testLogger())

	rec := doRefresh(h, localIdentity(a, time.Now().Add(-2*time.Hour)))
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("renovação: %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data loginResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Data.AccessToken == "" {
		t.Fatalf("resposta: %v %+v", err, resp)
	}
	if exp, _ := time.Parse(time.RFC3339, resp.Data.ExpiresAt); time.Until(exp) < 50*time.Minute {
		t.Errorf("token renovado deveria valer ~1h, vence em %v", time.Until(exp))
	}
	if len(resp.Data.User.Roles) != 2 || !store.touched[a.ID] {
		t.Errorf("roles relidas e last_seen: %+v touched=%v", resp.Data.User, store.touched[a.ID])
	}
}

func TestRefreshRefusals(t *testing.T) {
	a := account(t)
	inativa := account(t)
	inativa.Username, inativa.Active = "inativa", false
	until := time.Now().Add(time.Minute)
	bloqueada := account(t)
	bloqueada.Username, bloqueada.LockedUntil = "bloqueada", &until
	h := NewHandlers(newFakeStore(a, inativa, bloqueada), testSigner(t), nil, nil, nil, testLogger())
	agora := time.Now()

	casos := map[string]struct {
		id   *auth.Identity
		want int
	}{
		"sem identidade":    {nil, http.StatusUnauthorized},
		"token do Keycloak": {&auth.Identity{Subject: "kc", Username: "ana", Source: auth.SourceKeycloak}, http.StatusForbidden},
		"conta inexistente": {&auth.Identity{Subject: uuid.NewString(), Username: "ninguem", Source: auth.SourceLocal, AuthTime: agora}, http.StatusForbidden},
		"outro titular":     {&auth.Identity{Subject: uuid.NewString(), Username: "ana", Source: auth.SourceLocal, AuthTime: agora}, http.StatusForbidden},
		"conta desativada":  {localIdentity(inativa, agora), http.StatusForbidden},
		"conta bloqueada":   {localIdentity(bloqueada, agora), http.StatusForbidden},
		// Teto absoluto (12h no testSigner): a senha volta a ser pedida.
		"sessão no teto": {localIdentity(a, agora.Add(-13*time.Hour)), http.StatusForbidden},
	}
	for nome, c := range casos {
		if rec := doRefresh(h, c.id); rec.Code != c.want {
			t.Errorf("%s: %d, want %d (%s)", nome, rec.Code, c.want, rec.Body.String())
		}
	}

	if rec := doRefresh(NewHandlers(newFakeStore(a), nil, nil, nil, nil, testLogger()), localIdentity(a, agora)); rec.Code != http.StatusNotFound {
		t.Errorf("login local desligado: %d", rec.Code)
	}
	if rec := doRefresh(&Handlers{store: &storeFaults{fakeStore: newFakeStore(a), get: true}, signer: testSigner(t), logger: testLogger()}, localIdentity(a, agora)); rec.Code != http.StatusInternalServerError {
		t.Errorf("banco fora: %d", rec.Code)
	}
	if rec := doRefresh(&Handlers{store: newFakeStore(a), signer: failingIssuer{}, logger: testLogger()}, localIdentity(a, agora)); rec.Code != http.StatusInternalServerError {
		t.Errorf("falha ao assinar: %d", rec.Code)
	}
	// Não conseguir anotar o último acesso não derruba a renovação.
	if rec := doRefresh(&Handlers{store: &storeFaults{fakeStore: newFakeStore(a), touch: true}, signer: testSigner(t), logger: testLogger()}, localIdentity(a, agora)); rec.Code != http.StatusOK {
		t.Errorf("last_seen fora: %d", rec.Code)
	}
}
