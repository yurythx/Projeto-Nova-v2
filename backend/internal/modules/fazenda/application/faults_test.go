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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	fiscal auth.Identity
}

func (e *env) svc(repo domain.Repositorio) *application.Service {
	return application.NewService(e.pool, repo, application.Nucleo{Out: outbox.NewWriter("test")}, govbr.Config{}, silencioso())
}

func silencioso() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *env) real() *application.Service {
	return e.svc(infrastructure.NewPostgresRepository())
}

func (e *env) abrir() domain.Processo {
	e.t.Helper()
	p, err := e.real().Abrir(context.Background(), e.fiscal, domain.Abertura{
		Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true,
		Documentos: documentos(time.Now()),
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return p
}

func documentos(quando time.Time) []domain.Documento {
	tipos := []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"}
	out := make([]domain.Documento, len(tipos))
	for i, tipo := range tipos {
		out[i] = domain.Documento{Tipo: tipo, Presente: true, EmitidaEm: quando}
	}
	return out
}

func TestEveryRepositoryFailureIsPropagated(t *testing.T) {
	e := &env{
		t: t, pool: dbtest.Pool(t),
		fiscal: auth.Identity{
			Subject:     "fiscal-fazenda",
			Permissions: []string{"fazenda:analisar", "fazenda:decidir"},
		},
	}
	ctx := context.Background()
	type op = func() func(s *application.Service) error
	ops := map[string]op{
		"Abrir": func() func(*application.Service) error {
			return func(s *application.Service) error {
				_, err := s.Abrir(ctx, e.fiscal, domain.Abertura{
					Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true,
					Documentos: documentos(time.Now()),
				})
				return err
			}
		},
		"Listar": func() func(*application.Service) error {
			return func(s *application.Service) error { _, err := s.Listar(ctx, e.fiscal); return err }
		},
		"Painel": func() func(*application.Service) error {
			return func(s *application.Service) error { _, err := s.Painel(ctx, e.fiscal); return err }
		},
		"Alertar": func() func(*application.Service) error {
			return func(s *application.Service) error { _, err := s.Alertar(ctx, e.fiscal); return err }
		},
		"Autenticidade": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Autenticidade(ctx, p.Codigo); return err }
		},
		"Buscar": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Buscar(ctx, e.fiscal, p.ID); return err }
		},
		"Exigir": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Exigir(ctx, e.fiscal, p.ID, false); return err }
		},
		"Responder": func() func(*application.Service) error {
			p := e.abrir()
			if _, err := e.real().Exigir(ctx, e.fiscal, p.ID, false); err != nil {
				e.t.Fatal(err)
			}
			return func(s *application.Service) error { _, err := s.Responder(ctx, e.fiscal, p.ID, true); return err }
		},
		"Decidir": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Decidir(ctx, e.fiscal, p.ID); return err }
		},
		"Recorrer": func() func(*application.Service) error {
			p := e.abrir()
			if _, err := e.real().Decidir(ctx, e.fiscal, p.ID); err != nil {
				e.t.Fatal(err)
			}
			return func(s *application.Service) error { _, err := s.Recorrer(ctx, e.fiscal, p.ID, true); return err }
		},
		"GuardarReducao": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error {
				_, err := s.GuardarReducao(ctx, e.fiscal, p.ID, 2000, 5)
				return err
			}
		},
		"Parcelar": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Parcelar(ctx, e.fiscal, p.ID, 120000, 1); return err }
		},
		"Baixar": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.Baixar(ctx, e.fiscal, p.ID, false); return err }
		},
		"MarcarExecucao": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error { _, err := s.MarcarExecucao(ctx, e.fiscal, p.ID, true); return err }
		},
		"Avisar": func() func(*application.Service) error {
			p := e.abrir()
			return func(s *application.Service) error {
				_, err := s.Avisar(ctx, e.fiscal, p.ID, "protocolo aberto", "acompanhe na área restrita")
				return err
			}
		},
	}
	for name, o := range ops {
		probe := &faultRepo{inner: infrastructure.NewPostgresRepository()}
		if err := o()(e.svc(probe)); err != nil {
			t.Fatalf("%s sem falha: %v", name, err)
		}
		if len(probe.trace) == 0 {
			t.Fatalf("%s não chamou o repositório", name)
		}
		for i := range probe.trace {
			for _, poison := range []bool{false, true} {
				r := &faultRepo{inner: infrastructure.NewPostgresRepository()}
				if poison {
					r.poisonAt = i + 1
				} else {
					r.failAt = i + 1
				}
				if err := o()(e.svc(r)); err == nil && !r.noTx {
					t.Errorf("%s: falha (veneno=%v) na chamada %d (%s) foi engolida", name, poison, i+1, probe.trace[i])
				}
			}
		}
	}
	if err := application.MapError(errBoom); !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
}

func TestHandlersComRepositorioFora(t *testing.T) {
	e := &env{
		t: t, pool: dbtest.Pool(t),
		fiscal: auth.Identity{
			Subject:     "fiscal-fazenda",
			Permissions: []string{"fazenda:analisar", "fazenda:decidir"},
		},
	}
	down := &faultRepo{inner: infrastructure.NewPostgresRepository(), failAt: 1}
	h := transport.NewHandlers(e.svc(down), silencioso())
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(auth.WithIdentity(req.Context(), e.fiscal)))
		})
	})
	transport.RegisterPublic(r, h)
	transport.RegisterAuthed(r, h, silencioso())
	pedidos := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/fazenda/processos", nil),
		httptest.NewRequest(http.MethodGet, "/fazenda/processos/painel", nil),
		httptest.NewRequest(http.MethodGet, "/fazenda/autenticidade/codigo-inexistente", nil),
		httptest.NewRequest(http.MethodPost, "/fazenda/processos", strings.NewReader(`{}`)),
	}
	pedidos[3].Header.Set("Content-Type", "application/json")
	for _, req := range pedidos {
		down.calls = 0
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), errBoom.Error()) {
			t.Errorf("%s %s com o banco fora: %d %s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		}
	}
}
