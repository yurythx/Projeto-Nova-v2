package transport

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

func TestRotas(t *testing.T) {
	repo := &memoria{cols: []domain.Coluna{
		{Quadro: "problemas", Chave: "registrado", Nome: "Registrados", Tom: "info", Ordem: 0, Entrada: true, Ativa: true},
		{Quadro: "problemas", Chave: "em_analise", Nome: "Em análise", Tom: "info", Ordem: 1, Ativa: true},
	}}
	pool := testPool(t)
	h := NewHandlers(application.NewService(pool, repo, outbox.NewWriter("nova.quadros")), slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_type = 'quadro_coluna' AND aggregate_id LIKE 'problemas:c%'`)
	})
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	sem := pedir(r, http.MethodGet, "/quadros/problemas", "", auth.Identity{})
	if sem.Code != http.StatusForbidden {
		t.Fatalf("leitura: %d %s", sem.Code, sem.Body.String())
	}
	ok := pedir(r, http.MethodGet, "/quadros/problemas", "", gestor())
	if ok.Code != http.StatusOK {
		t.Fatalf("lista: %d %s", ok.Code, ok.Body.String())
	}
	ruim := pedir(r, http.MethodPost, "/quadros/problemas", "{", gestor())
	if ruim.Code < 400 {
		t.Fatalf("json: %d", ruim.Code)
	}
	ausente := pedir(r, http.MethodPost, "/quadros/outro", `{"nome":"Triagem","cor":"#112233","ordem":1,"de":["registrado"]}`, gestor())
	if ausente.Code != http.StatusNotFound {
		t.Fatalf("quadro: %d %s", ausente.Code, ausente.Body.String())
	}
	criado := pedir(r, http.MethodPost, "/quadros/problemas", `{"nome":"Triagem","cor":"#112233","ordem":1,"de":["registrado"],"para":["em_analise"]}`, gestor())
	if criado.Code != http.StatusCreated {
		t.Fatalf("criar: %d %s", criado.Code, criado.Body.String())
	}
	var env struct {
		Data domain.Coluna `json:"data"`
	}
	if err := json.Unmarshal(criado.Body.Bytes(), &env); err != nil || env.Data.Chave == "" {
		t.Fatal(err, criado.Body.String())
	}
	quebrado := pedir(r, http.MethodPut, "/quadros/problemas/"+env.Data.Chave, "{", gestor())
	if quebrado.Code < 400 {
		t.Fatalf("json: %d", quebrado.Code)
	}
	vazio := pedir(r, http.MethodPut, "/quadros/problemas/"+env.Data.Chave, "{}", gestor())
	if vazio.Code < 400 {
		t.Fatalf("patch: %d %s", vazio.Code, vazio.Body.String())
	}
	corpo := `{"nome":"Triagem nova"}`
	atual := pedir(r, http.MethodPut, "/quadros/problemas/"+env.Data.Chave, corpo, gestor())
	if atual.Code != http.StatusOK || !strings.Contains(atual.Body.String(), "Triagem nova") {
		t.Fatalf("atualizar: %d %s", atual.Code, atual.Body.String())
	}
}

func pedir(r http.Handler, metodo, url, corpo string, id auth.Identity) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, url, strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), id))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func gestor() auth.Identity {
	return auth.Identity{Subject: "ana", Permissions: []string{"problemas:manage"}}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type memoria struct {
	cols []domain.Coluna
}

func (m *memoria) Listar(context.Context, database.DBTX, string) ([]domain.Coluna, error) {
	return append([]domain.Coluna{}, m.cols...), nil
}
func (m *memoria) Salvar(_ context.Context, _ database.DBTX, cols []domain.Coluna) error {
	m.cols = append([]domain.Coluna{}, cols...)
	return nil
}
func (m *memoria) Inserir(_ context.Context, _ database.DBTX, c domain.Coluna) error {
	m.cols = append(m.cols, c)
	return nil
}
func (m *memoria) SoltarOrdens(context.Context, database.DBTX, string) error { return nil }
func (m *memoria) Contar(context.Context, database.DBTX, string, string) (int, error) {
	return 0, nil
}
func (m *memoria) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	return nil, nil
}
func (m *memoria) SubstituirPassagens(context.Context, database.DBTX, string, string, []domain.Passagem) error {
	return nil
}
