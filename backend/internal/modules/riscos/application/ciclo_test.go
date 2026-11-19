package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type repoFalha struct{ op string }

func (r repoFalha) falha(op string) error {
	if r.op == op {
		return errors.New("falha")
	}
	return nil
}

func (r repoFalha) ListRiscos(context.Context, database.DBTX, domain.Filtro) ([]domain.Risco, error) {
	return nil, r.falha("list")
}
func (r repoFalha) GetRisco(context.Context, database.DBTX, uuid.UUID) (domain.Risco, error) {
	if err := r.falha("get"); err != nil {
		return domain.Risco{}, err
	}
	return domain.Risco{UnidadeID: uuid.New(), Titulo: "Risco antigo"}, nil
}
func (r repoFalha) InsertRisco(ctx context.Context, db database.DBTX, item domain.Risco) (domain.Risco, error) {
	if r.op == "outbox" {
		_, _ = db.Exec(ctx, `SELECT 1/0`)
		return item, nil
	}
	return item, r.falha("insert")
}
func (r repoFalha) UpdateRisco(ctx context.Context, db database.DBTX, item domain.Risco) (domain.Risco, error) {
	if r.op == "outbox" {
		_, _ = db.Exec(ctx, `SELECT 1/0`)
		return item, nil
	}
	return item, r.falha("update")
}
func (r repoFalha) DeleteRisco(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if r.op == "outbox" {
		_, _ = db.Exec(ctx, `SELECT 1/0`)
		return nil
	}
	return r.falha("delete")
}

func entrada(u uuid.UUID, codigo string, prob, imp int) application.RiscoEntrada {
	return application.RiscoEntrada{
		Codigo: codigo, Titulo: "Risco de teste", Categoria: domain.CategoriaInfra,
		UnidadeID: u, Probabilidade: prob, Impacto: imp, Estrategia: domain.EstrategiaMitigar,
		Status: domain.StatusIdentificado,
	}
}

func TestMapErrorECobre(t *testing.T) {
	if application.MapError(nil) != nil {
		t.Fatal("nil")
	}
	if application.MapError(domain.ErrRiscoNaoEncontrado) == nil || application.MapError(domain.ErrRiscoInvalido) == nil ||
		application.MapError(domain.ErrForaDoEscopo) == nil || application.MapError(errors.New("outra")) == nil {
		t.Fatal("mapa")
	}
}

func TestCicloDoRisco(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	admin := auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: dbtest.User(t, pool)}
	repo := infrastructure.NewRepository()
	svc := application.NewService(pool, repo, outbox.NewWriter("nova.riscos"))
	codigo := "RSK-" + uuid.NewString()[:8]

	if _, err := svc.CriarRisco(ctx, auth.Identity{}, entrada(u, codigo, 1, 1)); err == nil {
		t.Fatal("sem escopo")
	}
	if _, err := svc.CriarRisco(ctx, admin, entrada(u, "", 1, 1)); err == nil {
		t.Fatal("inválido")
	}
	baixo, err := svc.CriarRisco(ctx, admin, entrada(u, codigo, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CriarRisco(ctx, admin, entrada(u, codigo, 2, 2)); err == nil {
		t.Fatal("código repetido")
	}
	for _, par := range [][2]int{{3, 3}, {4, 4}, {5, 5}} {
		if _, err := svc.CriarRisco(ctx, admin, entrada(u, "RSK-"+uuid.NewString()[:8], par[0], par[1])); err != nil {
			t.Fatal(err)
		}
	}
	lista, err := svc.ListarRiscos(ctx, admin, &u, domain.CategoriaInfra, domain.StatusIdentificado)
	if err != nil || len(lista) < 4 {
		t.Fatalf("lista: %d %v", len(lista), err)
	}
	if _, err := svc.ListarRiscos(ctx, auth.Identity{Scopes: []auth.Scope{{UnidadeID: &u, Permissions: []string{"riscos:manage"}, Unidades: []uuid.UUID{u}}}}, nil, "", ""); err != nil {
		t.Fatal(err)
	}
	item, err := svc.ObterRisco(ctx, admin, baixo.ID)
	if err != nil || item.ID != baixo.ID {
		t.Fatal(err)
	}
	outra := uuid.New()
	leitor := auth.Identity{Scopes: []auth.Scope{{UnidadeID: &outra, Permissions: []string{"riscos:read"}, Unidades: []uuid.UUID{outra}}}}
	if _, err := svc.ObterRisco(ctx, leitor, baixo.ID); err == nil {
		t.Fatal("fora do escopo")
	}
	if _, err := svc.ObterRisco(ctx, admin, uuid.New()); err == nil {
		t.Fatal("ausente")
	}
	heat, err := svc.Heatmap(ctx, admin, &u)
	if err != nil || heat.TotalRiscos < 4 || heat.Baixos == 0 || heat.Medios == 0 || heat.Altos == 0 || heat.Criticos == 0 || len(heat.Matriz) != 25 {
		t.Fatalf("heatmap: %+v %v", heat, err)
	}
	baixo.Titulo = "Risco revisado"
	atual := entrada(u, baixo.Codigo, 1, 2)
	atual.Titulo = "Risco revisado"
	atual.Status = domain.StatusEmAnalise
	if _, err := svc.AtualizarRisco(ctx, admin, baixo.ID, atual); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AtualizarRisco(ctx, leitor, baixo.ID, atual); err == nil {
		t.Fatal("atualizar sem escopo")
	}
	if err := svc.ExcluirRisco(ctx, leitor, baixo.ID); err == nil {
		t.Fatal("excluir sem escopo")
	}
	if err := svc.ExcluirRisco(ctx, admin, baixo.ID); err != nil {
		t.Fatal(err)
	}

	sem := application.NewService(pool, repoFalha{op: "list"}, nil)
	if _, err := sem.ListarRiscos(ctx, admin, nil, "", ""); err == nil {
		t.Fatal("lista")
	}
	if _, err := sem.Heatmap(ctx, admin, nil); err == nil {
		t.Fatal("heatmap")
	}
	sem = application.NewService(pool, repoFalha{op: "get"}, nil)
	if _, err := sem.ObterRisco(ctx, admin, uuid.New()); err == nil {
		t.Fatal("obter")
	}
	if _, err := sem.AtualizarRisco(ctx, admin, uuid.New(), entrada(u, "RSK-X", 1, 1)); err == nil {
		t.Fatal("atualizar")
	}
	if err := sem.ExcluirRisco(ctx, admin, uuid.New()); err == nil {
		t.Fatal("excluir")
	}
	sem = application.NewService(pool, repoFalha{op: "insert"}, outbox.NewWriter("nova.riscos"))
	if _, err := sem.CriarRisco(ctx, admin, entrada(u, "RSK-"+uuid.NewString()[:8], 1, 1)); err == nil {
		t.Fatal("insert")
	}
	sem = application.NewService(pool, repoFalha{}, outbox.NewWriter("nova.riscos"))
	if _, err := sem.AtualizarRisco(ctx, admin, uuid.New(), entrada(uuid.Nil, "RSK-Y", 1, 1)); err == nil {
		t.Fatal("unidade do destino")
	}
	sem = application.NewService(pool, repoFalha{op: "update"}, outbox.NewWriter("nova.riscos"))
	if _, err := sem.AtualizarRisco(ctx, admin, uuid.New(), entrada(u, "RSK-Y", 1, 1)); err == nil {
		t.Fatal("update")
	}
	sem = application.NewService(pool, repoFalha{op: "delete"}, outbox.NewWriter("nova.riscos"))
	if err := sem.ExcluirRisco(ctx, admin, uuid.New()); err == nil {
		t.Fatal("delete")
	}
	sem = application.NewService(pool, repoFalha{op: "outbox"}, outbox.NewWriter("nova.riscos"))
	if _, err := sem.CriarRisco(ctx, admin, entrada(u, "RSK-"+uuid.NewString()[:8], 1, 1)); err == nil {
		t.Fatal("outbox criar")
	}
	if _, err := sem.AtualizarRisco(ctx, admin, uuid.New(), entrada(u, "RSK-Y", 1, 1)); err == nil {
		t.Fatal("outbox atualizar")
	}
	if err := sem.ExcluirRisco(ctx, admin, uuid.New()); err == nil {
		t.Fatal("outbox excluir")
	}

	ctxDB := context.Background()
	if _, err := repo.ListRiscos(ctxDB, dbtest.Fail{}, domain.Filtro{}); err == nil {
		t.Fatal("query")
	}
	if _, err := repo.ListRiscos(ctxDB, dbtest.ScanFail{}, domain.Filtro{}); err == nil {
		t.Fatal("scan")
	}
	if _, err := repo.ListRiscos(ctxDB, dbtest.RowsErr{}, domain.Filtro{}); err == nil {
		t.Fatal("rows")
	}
	if _, err := repo.GetRisco(ctxDB, dbtest.ScanFail{}, uuid.New()); err == nil {
		t.Fatal("get")
	}
	if _, err := repo.UpdateRisco(ctxDB, dbtest.ScanFail{}, domain.Risco{ID: uuid.New()}); err == nil {
		t.Fatal("update sql")
	}
	if err := repo.DeleteRisco(ctxDB, dbtest.Fail{}, uuid.New()); err == nil {
		t.Fatal("delete sql")
	}
}

func TestHandlersDosRiscos(t *testing.T) {
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	admin := auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: dbtest.User(t, pool)}
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("nova.riscos"))
	h := transport.NewHandlers(svc, slog.New(slog.DiscardHandler))
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	chamar := func(method, path, corpo string, idt auth.Identity) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		if idt.UserID != uuid.Nil || len(idt.Roles) > 0 {
			req = req.WithContext(auth.WithIdentity(req.Context(), idt))
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	corpo := `{"codigo":"RSK-` + uuid.NewString()[:8] + `","titulo":"Risco http","categoria":"infraestrutura","unidade_id":"` + u.String() + `","probabilidade":2,"impacto":2,"estrategia":"mitigar"}`
	if chamar(http.MethodPost, "/riscos", "{", admin).Code < 400 {
		t.Fatal("json")
	}
	criado := chamar(http.MethodPost, "/riscos", corpo, admin)
	if criado.Code != http.StatusCreated {
		t.Fatalf("criar: %d %s", criado.Code, criado.Body.String())
	}
	id := criado.Body.String()
	ini := strings.Index(id, `"id":"`)
	id = id[ini+6 : ini+42]
	if chamar(http.MethodGet, "/riscos?unidade_id=nao", "", admin).Code < 400 {
		t.Fatal("unidade")
	}
	if chamar(http.MethodGet, "/riscos?unidade_id="+u.String()+"&categoria=infraestrutura&status=identificado", "", admin).Code != http.StatusOK {
		t.Fatal("lista")
	}
	if chamar(http.MethodGet, "/riscos/heatmap?unidade_id="+u.String(), "", admin).Code != http.StatusOK {
		t.Fatal("heatmap")
	}
	if chamar(http.MethodGet, "/riscos/heatmap?unidade_id=nao", "", admin).Code < 400 {
		t.Fatal("heatmap unidade")
	}
	if chamar(http.MethodGet, "/riscos/nao-e-uuid", "", admin).Code < 400 {
		t.Fatal("id")
	}
	if chamar(http.MethodGet, "/riscos/"+id, "", admin).Code != http.StatusOK {
		t.Fatal("obter")
	}
	if chamar(http.MethodPut, "/riscos/nao", "", admin).Code < 400 {
		t.Fatal("put id")
	}
	if chamar(http.MethodPut, "/riscos/"+id, "{", admin).Code < 400 {
		t.Fatal("put json")
	}
	if chamar(http.MethodPut, "/riscos/"+id, corpo, admin).Code != http.StatusOK {
		t.Fatal("put")
	}
	if chamar(http.MethodDelete, "/riscos/nao", "", admin).Code < 400 {
		t.Fatal("delete id")
	}
	if chamar(http.MethodDelete, "/riscos/"+id, "", admin).Code != http.StatusNoContent {
		t.Fatal("delete")
	}
	falho := transport.NewHandlers(application.NewService(pool, repoFalha{op: "list"}, nil), slog.New(slog.DiscardHandler))
	pedir := func(h *transport.Handlers, method, path, corpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		rota := chi.NewRouteContext()
		if i := strings.LastIndex(path, "/"); i > 0 && strings.Contains(path, "/riscos/") && !strings.Contains(path, "heatmap") {
			rota.URLParams.Add("id", path[i+1:])
		}
		req = req.WithContext(auth.WithIdentity(context.WithValue(req.Context(), chi.RouteCtxKey, rota), admin))
		rec := httptest.NewRecorder()
		switch method {
		case http.MethodGet:
			if strings.Contains(path, "heatmap") {
				h.Heatmap(rec, req)
			} else if strings.Count(path, "/") > 1 {
				h.Obter(rec, req)
			} else {
				h.Listar(rec, req)
			}
		case http.MethodPost:
			h.Criar(rec, req)
		case http.MethodPut:
			h.Atualizar(rec, req)
		default:
			h.Excluir(rec, req)
		}
		return rec
	}
	muda := transport.NewHandlers(application.NewService(pool, repoFalha{op: "get"}, nil), slog.New(slog.DiscardHandler))
	if pedir(muda, http.MethodGet, "/riscos/"+uuid.NewString(), "").Code < 400 {
		t.Fatal("obter falha")
	}
	cria := transport.NewHandlers(application.NewService(pool, repoFalha{op: "insert"}, nil), slog.New(slog.DiscardHandler))
	if pedir(cria, http.MethodPost, "/riscos", corpo).Code < 400 {
		t.Fatal("criar falha")
	}
	if pedir(muda, http.MethodPut, "/riscos/"+uuid.NewString(), corpo).Code < 400 {
		t.Fatal("atualizar falha")
	}
	if pedir(muda, http.MethodDelete, "/riscos/"+uuid.NewString(), "").Code < 400 {
		t.Fatal("excluir falha")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/riscos", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), admin))
	falho.Listar(rec, req)
	if rec.Code < 400 {
		t.Fatal("lista falha")
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/riscos/heatmap", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), admin))
	falho.Heatmap(rec, req)
	if rec.Code < 400 {
		t.Fatal("heatmap falha")
	}
}
