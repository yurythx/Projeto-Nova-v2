package transport

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

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
)

type falso struct{ err error }

func (f falso) Simular(domain.Simulacao) (domain.Simulado, error) {
	return domain.Simulado{Devido: 1}, f.err
}
func (f falso) IniciarGovBR() (govbr.Desafio, error) {
	return govbr.Desafio{URL: "https://sso.example/authorize"}, f.err
}
func (f falso) TrocarGovBR(string, string) (application.SessaoGovBR, error) {
	return application.SessaoGovBR{Sub: "ana"}, f.err
}
func (f falso) Autenticidade(context.Context, string) (domain.Autenticidade, error) {
	return domain.Autenticidade{Valido: true}, f.err
}
func (f falso) Abrir(context.Context, auth.Identity, domain.Abertura) (domain.Processo, error) {
	return domain.Processo{Numero: "000001/2026"}, f.err
}
func (f falso) Listar(context.Context, auth.Identity) ([]domain.Processo, error) {
	return []domain.Processo{}, f.err
}
func (f falso) Painel(context.Context, auth.Identity) ([]domain.PrazoPainel, error) {
	return []domain.PrazoPainel{{Nome: "relatório semestral", Situacao: domain.PrazoAtrasado}}, f.err
}
func (f falso) Alertar(context.Context, auth.Identity) (int, error) {
	return 1, f.err
}
func (f falso) Buscar(context.Context, auth.Identity, uuid.UUID) (domain.Processo, error) {
	return domain.Processo{Numero: "000001/2026"}, f.err
}
func (f falso) RegistrarConsulta(context.Context, auth.Identity, uuid.UUID) (domain.Processo, error) {
	return domain.Processo{Numero: "000001/2026"}, f.err
}
func (f falso) Exigir(context.Context, auth.Identity, uuid.UUID, bool) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) Responder(context.Context, auth.Identity, uuid.UUID, bool) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) Decidir(context.Context, auth.Identity, uuid.UUID) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) Recorrer(context.Context, auth.Identity, uuid.UUID, bool) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) GuardarReducao(context.Context, auth.Identity, uuid.UUID, int64, int) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) Parcelar(context.Context, auth.Identity, uuid.UUID, int64, int) (domain.Plano, error) {
	return domain.Plano{Estado: domain.SituacaoAguardandoPagamento}, f.err
}
func (f falso) Baixar(context.Context, auth.Identity, uuid.UUID, bool) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) MarcarExecucao(context.Context, auth.Identity, uuid.UUID, bool) (domain.Processo, error) {
	return domain.Processo{}, f.err
}
func (f falso) Avisar(context.Context, auth.Identity, uuid.UUID, string, string) (string, error) {
	return "000001/2026", f.err
}
func (f falso) ReceberDUA([]domain.LinhaDUA, int) (domain.ReciboDUA, error) {
	return domain.ReciboDUA{Recibo: "entrega recebida", Linhas: 1}, f.err
}

func pedido(method, body string, quem *auth.Identity, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	ctx := req.Context()
	if quem != nil {
		ctx = auth.WithIdentity(ctx, *quem)
	}
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	}
	return req.WithContext(ctx)
}

func chama(t *testing.T, h http.HandlerFunc, req *http.Request, status int) {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != status {
		t.Fatalf("status %d, esperado %d: %s", rec.Code, status, rec.Body.String())
	}
}

func TestHandlers(t *testing.T) {
	h := NewHandlers(falso{}, nil)
	id := uuid.New().String()
	quem := auth.Identity{Subject: "ana", Permissions: []string{"*"}}
	chama(t, h.Simular, pedido(http.MethodPost, "{", nil, nil), http.StatusBadRequest)
	chama(t, h.Simular, pedido(http.MethodPost, "{}", nil, nil), http.StatusOK)
	chama(t, h.IniciarGovBR, pedido(http.MethodGet, "", nil, nil), http.StatusOK)
	chama(t, h.TrocarGovBR, pedido(http.MethodPost, "{}", nil, nil), http.StatusUnprocessableEntity)
	chama(t, h.TrocarGovBR, pedido(http.MethodPost, `{"code":"c","verifier":"v"}`, nil, nil), http.StatusOK)
	chama(t, h.Autenticidade, pedido(http.MethodGet, "", nil, map[string]string{"codigo": "abc"}), http.StatusOK)
	chama(t, h.Abrir, pedido(http.MethodPost, "{}", nil, nil), http.StatusUnauthorized)
	chama(t, h.Abrir, pedido(http.MethodPost, "{", &quem, nil), http.StatusBadRequest)
	chama(t, h.Abrir, pedido(http.MethodPost, "{}", &quem, nil), http.StatusCreated)
	chama(t, h.Listar, pedido(http.MethodGet, "", nil, nil), http.StatusUnauthorized)
	chama(t, h.Listar, pedido(http.MethodGet, "", &quem, nil), http.StatusOK)
	chama(t, h.Painel, pedido(http.MethodGet, "", nil, nil), http.StatusUnauthorized)
	chama(t, h.Painel, pedido(http.MethodGet, "", &quem, nil), http.StatusOK)
	chama(t, h.Alertar, pedido(http.MethodPost, "", nil, nil), http.StatusUnauthorized)
	chama(t, h.Alertar, pedido(http.MethodPost, "", &quem, nil), http.StatusOK)
	chama(t, h.Buscar, pedido(http.MethodGet, "", nil, map[string]string{"id": id}), http.StatusUnauthorized)
	chama(t, h.Buscar, pedido(http.MethodGet, "", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Buscar, pedido(http.MethodGet, "", &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Exigir, pedido(http.MethodPost, "{}", nil, map[string]string{"id": id}), http.StatusUnauthorized)
	chama(t, h.Exigir, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Exigir, pedido(http.MethodPost, "{", &quem, map[string]string{"id": id}), http.StatusBadRequest)
	chama(t, h.Exigir, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Responder, pedido(http.MethodPost, "{}", nil, map[string]string{"id": id}), http.StatusUnauthorized)
	chama(t, h.Responder, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Responder, pedido(http.MethodPost, "{", &quem, map[string]string{"id": id}), http.StatusBadRequest)
	chama(t, h.Responder, pedido(http.MethodPost, `{"respondeu":true}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Decidir, pedido(http.MethodPost, "", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Decidir, pedido(http.MethodPost, "", &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Recorrer, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Reducao, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Parcelar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Baixar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Execucao, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Avisar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": "x"}), http.StatusBadRequest)
	chama(t, h.Recorrer, pedido(http.MethodPost, "{", &quem, map[string]string{"id": id}), http.StatusBadRequest)
	chama(t, h.Recorrer, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Reducao, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusUnprocessableEntity)
	chama(t, h.Reducao, pedido(http.MethodPost, `{"milessimos":2000,"condicoes":5}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Parcelar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusUnprocessableEntity)
	chama(t, h.Parcelar, pedido(http.MethodPost, `{"total":120000,"parcelas":12}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Baixar, pedido(http.MethodPost, "{", &quem, map[string]string{"id": id}), http.StatusBadRequest)
	chama(t, h.Baixar, pedido(http.MethodPost, `{"primeira":true}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Execucao, pedido(http.MethodPost, "{", &quem, map[string]string{"id": id}), http.StatusBadRequest)
	chama(t, h.Execucao, pedido(http.MethodPost, `{"em_execucao":true}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.Avisar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusUnprocessableEntity)
	chama(t, h.Avisar, pedido(http.MethodPost, `{"fato":"registro informado","orientacao":"acompanhe"}`, &quem, map[string]string{"id": id}), http.StatusOK)
	chama(t, h.DUA, pedido(http.MethodPost, "{}", nil, nil), http.StatusUnauthorized)
	chama(t, h.DUA, pedido(http.MethodPost, "{", &quem, nil), http.StatusBadRequest)
	chama(t, h.DUA, pedido(http.MethodPost, `{"omitidas":1}`, &quem, nil), http.StatusCreated)

	ruim := NewHandlers(falso{err: errors.New("falha")}, h.logger)
	chama(t, ruim.Simular, pedido(http.MethodPost, "{}", nil, nil), http.StatusInternalServerError)
	chama(t, ruim.IniciarGovBR, pedido(http.MethodGet, "", nil, nil), http.StatusInternalServerError)
	chama(t, ruim.TrocarGovBR, pedido(http.MethodPost, `{"code":"c","verifier":"v"}`, nil, nil), http.StatusInternalServerError)
	chama(t, ruim.Autenticidade, pedido(http.MethodGet, "", nil, map[string]string{"codigo": "abc"}), http.StatusInternalServerError)
	chama(t, ruim.Abrir, pedido(http.MethodPost, "{}", &quem, nil), http.StatusInternalServerError)
	chama(t, ruim.Listar, pedido(http.MethodGet, "", &quem, nil), http.StatusInternalServerError)
	chama(t, ruim.Painel, pedido(http.MethodGet, "", &quem, nil), http.StatusInternalServerError)
	chama(t, ruim.Alertar, pedido(http.MethodPost, "", &quem, nil), http.StatusInternalServerError)
	chama(t, ruim.Buscar, pedido(http.MethodGet, "", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Exigir, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Responder, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Decidir, pedido(http.MethodPost, "", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Recorrer, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Reducao, pedido(http.MethodPost, `{"milessimos":2000,"condicoes":5}`, &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Parcelar, pedido(http.MethodPost, `{"total":120000,"parcelas":12}`, &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Baixar, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Execucao, pedido(http.MethodPost, "{}", &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.Avisar, pedido(http.MethodPost, `{"fato":"a","orientacao":"b"}`, &quem, map[string]string{"id": id}), http.StatusInternalServerError)
	chama(t, ruim.DUA, pedido(http.MethodPost, `{"omitidas":1}`, &quem, nil), http.StatusInternalServerError)
}

type libera struct{}

func (libera) Allow(context.Context, string) (bool, error) { return true, nil }

func TestRotas(t *testing.T) {
	h := NewHandlers(falso{}, nil)
	r := chi.NewRouter()
	RegisterPublic(r, h)
	RegisterAuthed(r, h, slog.Default())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fazenda/govbr", nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/fazenda/processos/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatal(rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/fazenda/processos/", strings.NewReader("{}"))
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{Subject: "ana"}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("abrir o próprio processo: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/fazenda/processos/dua", strings.NewReader(`{"omitidas":1}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{Subject: "ana"}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("dua: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/fazenda/processos/painel", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{Subject: "fiscal", Permissions: []string{"fazenda:analisar"}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("painel: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/fazenda/processos/alertas", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{Subject: "fiscal", Permissions: []string{"fazenda:analisar"}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("alertas: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/fazenda/processos/"+uuid.New().String()+"/decisao", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{Permissions: []string{"fazenda:protocolar"}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sem decidir: %d", rec.Code)
	}
}
