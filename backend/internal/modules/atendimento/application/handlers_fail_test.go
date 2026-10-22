package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func TestHandlersListamComFalha(t *testing.T) {
	h := transport.NewHandlers(
		application.NewService(nil, stubRepo{listErr: errors.New("banco")}, fakeEventos{}, nil),
		slog.New(slog.DiscardHandler),
	)
	for _, call := range []func(http.ResponseWriter, *http.Request){h.ListarOfertas, h.ListarEquipes} {
		req := httptest.NewRequest(http.MethodGet, "/atendimento", nil)
		req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
		rec := httptest.NewRecorder()
		call(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	}
}

// leituraFalha derruba as leituras do portal.
type leituraFalha struct{ stubRepo }

func (leituraFalha) ListOfertasAtivas(context.Context, database.DBTX) ([]domain.Oferta, error) {
	return nil, errors.New("banco")
}
func (leituraFalha) VisitasEmAndamento(context.Context, database.DBTX, domain.Filtro) ([]domain.VisitaEmCampo, error) {
	return nil, errors.New("banco")
}
func (leituraFalha) ListSolicitacoes(context.Context, database.DBTX, domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	return nil, 0, errors.New("banco")
}

func TestHandlersDoPortalComFalha(t *testing.T) {
	h := transport.NewHandlers(
		application.NewService(nil, leituraFalha{stubRepo{candErr: errors.New("banco")}}, fakeEventos{}, nil),
		slog.New(slog.DiscardHandler),
	)
	for _, c := range []struct {
		call func(http.ResponseWriter, *http.Request)
		url  string
	}{
		{h.Catalogo, "/atendimento/catalogo"},
		{h.ListarMinhas, "/atendimento/minhas?situacao=abertas"},
		{h.ListarFila, "/atendimento/fila"},
		{h.Candidatos, "/atendimento/candidatos?unidade_id=" + uuid.NewString()},
		{h.VisitasEmCampo, "/atendimento/visitas/em-campo"},
	} {
		req := httptest.NewRequest(http.MethodGet, c.url, nil)
		req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
		rec := httptest.NewRecorder()
		c.call(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s: status %d: %s", c.url, rec.Code, rec.Body.String())
		}
	}
}
