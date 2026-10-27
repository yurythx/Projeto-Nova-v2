package transport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// pegarFalha é o banco que falha ao entregar a tarefa ao conector: o
// único erro do pedido que a API não consegue provocar de fora.
type pegarFalha struct{ domain.Repository }

func (pegarFalha) Pegar(context.Context, database.DBTX, *uuid.UUID, []string, string) (domain.Tarefa, bool, error) {
	return domain.Tarefa{}, false, errors.New("banco fora")
}

func TestPegarComFalhaNoBanco(t *testing.T) {
	h := NewHandlers(application.NewService(nil, pegarFalha{infrastructure.NewRepository()}, nil, nil), slog.New(slog.DiscardHandler))
	id := uuid.New()
	conector := auth.Identity{Subject: id.String(), Source: auth.SourceAgente, Permissions: []string{string(auth.PermIntegracoesConector)}}
	req := httptest.NewRequest(http.MethodPost, "/integracoes/conector/tarefa", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), conector))
	rec := httptest.NewRecorder()
	h.pegar(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}
