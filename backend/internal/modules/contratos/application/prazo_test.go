package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type semEventos struct{ falha string }

func (e semEventos) Write(_ context.Context, _ pgx.Tx, tipo, _, _ string, _ uuid.UUID, _ any) error {
	if tipo == e.falha {
		return errors.New("falhou")
	}
	return nil
}

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func gestor(u uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: &u, Permissions: []string{"contratos:manage"}, Unidades: []uuid.UUID{u}}}}
}

func TestPrazoEVigentes(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u, outra := dbtest.Unidade(t, pool), dbtest.Unidade(t, pool)
	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	svc := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).WithRelogio(func() time.Time { return hoje.Add(10 * time.Hour) })
	f, err := svc.CriarFornecedor(ctx, admin(), "Provedor "+uuid.NewString()[:6], "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	criar := func(titulo string, unidade *uuid.UUID, inicio, fim time.Time) domain.Contrato {
		t.Helper()
		c, err := svc.CriarContrato(ctx, admin(), "", titulo, f.ID, inicio, fim, 0, unidade)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	daUnidade := criar("Link da unidade", &u, hoje.AddDate(0, -1, 0), hoje.AddDate(1, 0, 0))
	institucional := criar("Link institucional", nil, hoje, hoje)
	deOutra := criar("Link de outra", &outra, hoje.AddDate(0, -1, 0), hoje.AddDate(1, 0, 0))
	vencido := criar("Link vencido", &u, hoje.AddDate(-1, 0, 0), hoje.AddDate(0, 0, -1))
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_contratos WHERE fornecedor_id = $1`, f.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_fornecedores WHERE id = $1`, f.ID)
	})

	vinte, zero := 20, 0
	if err := svc.DefinirPrazo(ctx, gestor(u), daUnidade.ID, &vinte); err != nil {
		t.Fatal(err)
	}
	for nome, err := range map[string]error{
		"prazo zero":                svc.DefinirPrazo(ctx, admin(), daUnidade.ID, &zero),
		"contrato de outra unidade": svc.DefinirPrazo(ctx, gestor(u), deOutra.ID, &vinte),
		"sem permissão":             svc.DefinirPrazo(ctx, auth.Identity{UserID: uuid.New()}, daUnidade.ID, &vinte),
		"falha no outbox": application.NewService(pool, infrastructure.NewRepository(), semEventos{"contratos.contrato.prazo"}).
			DefinirPrazo(ctx, admin(), daUnidade.ID, nil),
	} {
		if err == nil {
			t.Errorf("%s: aceitou", nome)
		}
	}

	vigentes, err := svc.Vigentes(ctx, []uuid.UUID{u}, false)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[uuid.UUID]domain.Contrato{}
	for _, c := range vigentes {
		ids[c.ID] = c
	}
	if _, ok := ids[daUnidade.ID]; !ok || *ids[daUnidade.ID].PrazoHoras != 20 {
		t.Fatalf("o da unidade, com prazo: %+v", ids[daUnidade.ID])
	}
	if _, ok := ids[institucional.ID]; !ok {
		t.Fatal("o institucional serve a todas, e vale no último dia")
	}
	if _, ok := ids[deOutra.ID]; ok {
		t.Fatal("o de outra unidade não entra")
	}
	if _, ok := ids[vencido.ID]; ok {
		t.Fatal("o vencido não entra")
	}
	todos, _ := svc.Vigentes(ctx, nil, true)
	achou := false
	for _, c := range todos {
		achou = achou || c.ID == deOutra.ID
	}
	if !achou {
		t.Fatal("sem recorte, entram todas as unidades")
	}
	if err := svc.DefinirPrazo(ctx, admin(), daUnidade.ID, nil); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	transport.NewHandlers(svc, slog.New(slog.NewTextHandler(io.Discard, nil))).RegisterRoutes(router)
	put := func(id, corpo string) int {
		r := httptest.NewRequest(http.MethodPut, "/contratos/contratos/"+id+"/prazo", strings.NewReader(corpo))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(auth.WithIdentity(r.Context(), admin()))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, r)
		return rec.Code
	}
	for nome, k := range map[string]struct {
		id, corpo string
		espera    int
	}{
		"define":         {daUnidade.ID.String(), `{"prazo_horas":8}`, http.StatusNoContent},
		"tira":           {daUnidade.ID.String(), `{"prazo_horas":null}`, http.StatusNoContent},
		"fora da faixa":  {daUnidade.ID.String(), `{"prazo_horas":0}`, http.StatusUnprocessableEntity},
		"id inválido":    {"x", `{}`, http.StatusBadRequest},
		"corpo inválido": {daUnidade.ID.String(), `{`, http.StatusBadRequest},
		"inexistente":    {uuid.NewString(), `{"prazo_horas":8}`, http.StatusNotFound},
	} {
		if got := put(k.id, k.corpo); got != k.espera {
			t.Errorf("%s: %d", nome, got)
		}
	}
}
