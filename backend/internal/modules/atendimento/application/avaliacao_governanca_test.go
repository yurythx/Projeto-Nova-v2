package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func concluir(t *testing.T, c cenario, s domain.Solicitacao) {
	t.Helper()
	if _, err := c.base.Transicionar(context.Background(), c.tecnico, s.ID, domain.EstadoResolvido); err != nil {
		t.Fatal(err)
	}
}

func TestAvaliacaoEGovernanca(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	aberta := c.abrir(t)
	if _, err := c.base.Avaliar(ctx, c.servidor, aberta.ID, 5, ""); !errors.Is(err, domain.ErrAvaliacaoForaDeHora) {
		t.Fatalf("cedo: %v", err)
	}
	em := c.emAtendimento(t)
	concluir(t, c, em)
	if _, err := c.base.Avaliar(ctx, c.tecnico, em.ID, 5, ""); !errors.Is(err, domain.ErrAvaliacaoAlheia) {
		t.Fatalf("alheia: %v", err)
	}
	if _, err := c.base.Avaliar(ctx, c.servidor, em.ID, 0, ""); !errors.Is(err, domain.ErrAvaliacaoInvalida) {
		t.Fatalf("nota: %v", err)
	}
	nota, err := c.base.Avaliar(ctx, c.servidor, em.ID, 5, "  rápido  ")
	if err != nil || nota.Nota != 5 || nota.Comentario != "rápido" {
		t.Fatalf("avaliação: %+v %v", nota, err)
	}
	if _, err := c.base.Avaliar(ctx, c.servidor, em.ID, 4, ""); !errors.Is(err, domain.ErrAvaliacaoExistente) {
		t.Fatalf("segunda: %v", err)
	}
	det, err := c.base.Obter(ctx, c.tecnico, em.ID)
	if err != nil || det.Avaliacao == nil || det.Avaliacao.Nota != 5 {
		t.Fatalf("detalhe: %+v %v", det.Avaliacao, err)
	}
	if _, err := c.com(falhando("GetAvaliacao"), eventosFalham{}).Obter(ctx, c.tecnico, em.ID); err == nil {
		t.Fatal("leitura da avaliação")
	}
	if _, err := c.com(falhando("InsertAvaliacao"), eventosFalham{}).Avaliar(ctx, c.servidor, em.ID, 5, ""); err == nil {
		t.Fatal("insert")
	}
	outra := c.emAtendimento(t)
	concluir(t, c, outra)
	if _, err := c.com(falhando(), eventosFalham{application.EventoSolicitacaoAvaliada}).Avaliar(ctx, c.servidor, outra.ID, 3, ""); err == nil {
		t.Fatal("evento")
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET responsavel_id = NULL WHERE id = $1`, outra.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Avaliar(ctx, c.servidor, outra.ID, 4, ""); err != nil {
		t.Fatalf("sem responsável: %v", err)
	}

	g, err := c.base.Governanca(ctx, c.tecnico)
	if err != nil || g.CSAT == nil || g.CSAT.Total != 2 || g.CSAT.Media != 4.5 {
		t.Fatalf("governança: %+v %v", g, err)
	}
	if _, err := c.base.Governanca(ctx, auth.Identity{}); err != nil {
		t.Fatal(err)
	}
	antes := application.TetoGovernanca
	application.TetoGovernanca = 0
	t.Cleanup(func() { application.TetoGovernanca = antes })
	g, err = c.base.Governanca(ctx, c.tecnico)
	if err != nil || !g.Amostra.Truncado {
		t.Fatalf("truncada: %+v %v", g.Amostra, err)
	}
	application.TetoGovernanca = antes
	if _, err := c.com(falhando("ListarParaGovernanca"), eventosFalham{}).Governanca(ctx, c.tecnico); err == nil {
		t.Fatal("lista")
	}
	if _, err := c.com(falhando("ResumoAvaliacoes"), eventosFalham{}).Governanca(ctx, c.tecnico); err == nil {
		t.Fatal("resumo")
	}
	if _, err := c.com(falhando("PassagensDe"), eventosFalham{}).Governanca(ctx, c.tecnico); err == nil {
		t.Fatal("passagens")
	}
}

func TestConcluirPelaMudanca(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	aberta := c.abrir(t)
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.ConcluirPorMudanca(ctx, tx, aberta.ID, c.tecnico.UserID, 9, "Fonte", ""); err == nil {
		t.Fatal("aberta não conclui")
	}
	_ = tx.Rollback(ctx)

	em := c.emAtendimento(t)
	tx, err = c.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	entrega, err := c.base.ConcluirPorMudanca(ctx, tx, em.ID, c.tecnico.UserID, 9, "Fonte", strings.Repeat("a", domain.TextoMax))
	if err != nil || entrega == nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	entrega(ctx)
	var texto string
	if err := c.pool.QueryRow(ctx, `SELECT texto FROM atendimento_comentarios WHERE solicitacao_id = $1`, em.ID).Scan(&texto); err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(texto) != domain.TextoMax || !strings.Contains(texto, "mudança #9") {
		t.Fatalf("comentário: %d %q", utf8.RuneCountInString(texto), texto[:40])
	}
	tx, err = c.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	entrega, err = c.base.ConcluirPorMudanca(ctx, tx, em.ID, c.tecnico.UserID, 9, "Fonte", "")
	_ = tx.Rollback(ctx)
	if err != nil || entrega != nil {
		t.Fatalf("já concluída: err=%v entrega=%t", err, entrega != nil)
	}

	nova := c.emAtendimento(t)
	for _, m := range []string{"TravarSolicitacao", "GetSolicitacao", "UpdateSolicitacao", "InsertPassagem", "InsertComentario"} {
		tx, err = c.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.com(falhando(m), eventosFalham{}).ConcluirPorMudanca(ctx, tx, nova.ID, c.tecnico.UserID, 1, "T", "")
		_ = tx.Rollback(ctx)
		if err == nil {
			t.Errorf("falha em %s", m)
		}
	}
	tx, err = c.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.com(falhando(), eventosFalham{application.EventoSolicitacaoComentada}).ConcluirPorMudanca(ctx, tx, nova.ID, c.tecnico.UserID, 1, "T", "ok")
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("evento do comentário")
	}
}

func pedido(t *testing.T, method, alvo, id string, body any, quem auth.Identity) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, alvo, &buf)
	rctx := chi.NewRouteContext()
	if id != "" {
		rctx.URLParams.Add("id", id)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(auth.WithIdentity(ctx, quem))
}

func TestHandlersDeAvaliacaoEGovernanca(t *testing.T) {
	c := novoCenario(t)
	h := transport.NewHandlers(c.base, silencio)
	em := c.emAtendimento(t)
	concluir(t, c, em)

	rec := httptest.NewRecorder()
	h.Avaliar(rec, pedido(t, http.MethodPost, "/avaliacao", "nao-e-uuid", nil, c.servidor))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("id: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Avaliar(rec, pedido(t, http.MethodPost, "/avaliacao", em.ID.String(), map[string]any{"nota": 9}, c.servidor))
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("nota: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.Avaliar(rec, pedido(t, http.MethodPost, "/avaliacao", em.ID.String(), map[string]any{"nota": 5, "comentario": "ok"}, c.servidor))
	if rec.Code != http.StatusCreated {
		t.Fatalf("criada: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.Governanca(rec, pedido(t, http.MethodGet, "/governanca", "", nil, c.tecnico))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"media"`) {
		t.Fatalf("governança: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.Governanca(rec, pedido(t, http.MethodGet, "/governanca", "", nil, c.tecnico))
	quebrado := transport.NewHandlers(c.com(falhando("ListarParaGovernanca"), eventosFalham{}), silencio)
	rec = httptest.NewRecorder()
	quebrado.Governanca(rec, pedido(t, http.MethodGet, "/governanca", "", nil, c.tecnico))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("falha: %d", rec.Code)
	}
}
