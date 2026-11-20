package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log/slog"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

func pedido(method, body string, idt *auth.Identity, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rota := chi.NewRouteContext()
	for chave, valor := range params {
		rota.URLParams.Add(chave, valor)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rota)
	if idt != nil {
		ctx = auth.WithIdentity(ctx, *idt)
	}
	return req.WithContext(ctx)
}

func TestParametrosInvalidos(t *testing.T) {
	h := NewHandlers(nil, slog.New(slog.DiscardHandler))
	id := uuid.New().String()
	casos := []struct {
		chamar func(http.ResponseWriter, *http.Request)
		params map[string]string
	}{
		{h.PedirAnexo, map[string]string{"id": "nao"}},
		{h.PedirAnexo, map[string]string{"id": id, "tarefaId": "nao"}},
		{h.RegistrarAnexo, map[string]string{"id": "nao"}},
		{h.RegistrarAnexo, map[string]string{"id": id, "tarefaId": "nao"}},
		{h.ListarMovimentos, map[string]string{"id": "nao"}},
		{h.ListarMovimentos, map[string]string{"id": id, "tarefaId": "nao"}},
		{h.URLAnexo, map[string]string{"id": "nao"}},
		{h.URLAnexo, map[string]string{"id": id, "tarefaId": "nao"}},
		{h.ListarAtualizacoes, map[string]string{"id": "nao"}},
		{h.CriarAtualizacao, map[string]string{"id": "nao"}},
		{h.CriarAtualizacao, map[string]string{"id": id, "tarefaId": "nao"}},
	}
	for _, caso := range casos {
		rec := httptest.NewRecorder()
		caso.chamar(rec, pedido(http.MethodPost, `{}`, nil, caso.params))
		if rec.Code < 400 {
			t.Fatalf("status %d", rec.Code)
		}
	}
}

func TestReabrirEExcluirPelaHTTP(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	for _, id := range []uuid.UUID{pessoa, outro} {
		if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, id, unidade); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	logger := slog.New(slog.DiscardHandler)
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa}
	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, outro); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(svc, logger)
	rec := httptest.NewRecorder()
	h.ExcluirParticipante(rec, pedido(http.MethodDelete, "", &admin, map[string]string{
		"id": p.ID.String(), "pessoaId": outro.String(),
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("excluir %d %s", rec.Code, rec.Body.String())
	}

	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	h.Reabrir(rec, pedido(http.MethodPost, `{"motivo":"voltou o escopo"}`, &admin, map[string]string{
		"id": p.ID.String(),
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("reabrir %d %s", rec.Code, rec.Body.String())
	}
}
