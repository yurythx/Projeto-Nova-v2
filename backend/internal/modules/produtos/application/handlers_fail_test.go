package application_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yurythx/sismarq/internal/modules/produtos/application"
	"github.com/yurythx/sismarq/internal/modules/produtos/transport"
	"github.com/yurythx/sismarq/internal/platform/auth"
)

func TestHandlerListaComFalha(t *testing.T) {
	h := transport.NewHandlers(
		application.NewService(nil, stubRepo{listErr: errors.New("banco")}, fakeEventos{}),
		slog.New(slog.DiscardHandler),
	)
	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}
