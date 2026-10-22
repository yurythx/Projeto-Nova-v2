package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

var errFalha = errors.New("falha")

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func gestor(u uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: &u, Permissions: []string{"ativos:manage"}, Unidades: []uuid.UUID{u}}}}
}

// repo em memória com falhas marcadas.
type repo struct {
	ativo                     domain.Ativo
	get, list, insert, update error
}

func (r repo) List(context.Context, database.DBTX, domain.Filtro) ([]domain.Ativo, error) {
	return []domain.Ativo{}, r.list
}
func (r repo) Get(context.Context, database.DBTX, uuid.UUID) (domain.Ativo, error) {
	return r.ativo, r.get
}
func (r repo) Insert(_ context.Context, _ database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	return a, r.insert
}
func (r repo) Update(_ context.Context, _ database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	return a, r.update
}

type eventos struct{ err error }

func (e eventos) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return e.err
}

type produtos struct {
	unidade *uuid.UUID
	err     error
}

func (p produtos) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, p.err
}

func TestMapError(t *testing.T) {
	for _, err := range []error{domain.ErrNaoEncontrado, domain.ErrEntradaInvalida, domain.ErrTipoInvalido, domain.ErrEstadoInvalido,
		domain.ErrForaDoEscopo, domain.ErrUnidadeInexistente, domain.ErrIdentificadorRepetido, domain.ErrProdutoInexistente,
		domain.ErrProdutoIndisponivel, domain.ErrProdutoDeOutraUnidade} {
		var app *apperrors.Error
		if !errors.As(application.MapError(err), &app) {
			t.Errorf("%v", err)
		}
	}
	if application.MapError(nil) != nil || application.MapError(errFalha) != errFalha {
		t.Fatal("nil e desconhecido")
	}
}

func TestRegras(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u, outra := uuid.New(), uuid.New()
	produto := uuid.New()
	in := domain.Entrada{Nome: "Switch", Tipo: "rede", Estado: "em_uso", UnidadeID: &u, ProdutoID: &produto}
	atual, _ := domain.Novo(in)
	svc := func(r repo, p domain.Produtos, e eventos) *application.Service {
		return application.NewService(pool, r, e, p)
	}

	if lista, err := svc(repo{}, nil, eventos{}).Listar(ctx, auth.Identity{}, nil); err != nil || len(lista) != 0 {
		t.Fatalf("sem cobertura: %v %v", lista, err)
	}
	if _, err := svc(repo{list: errFalha}, nil, eventos{}).Listar(ctx, admin(), nil); err == nil {
		t.Fatal("lista")
	}
	if _, err := svc(repo{get: errFalha}, nil, eventos{}).Obter(ctx, admin(), atual.ID); err == nil {
		t.Fatal("obter")
	}
	if _, err := svc(repo{ativo: atual}, nil, eventos{}).Obter(ctx, gestor(outra), atual.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("fora do escopo")
	}

	if _, err := svc(repo{}, nil, eventos{}).Criar(ctx, admin(), domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := svc(repo{}, nil, eventos{}).Criar(ctx, admin(), in); !errors.Is(err, domain.ErrProdutoIndisponivel) {
		t.Fatalf("sem porta de produtos: %v", err)
	}
	if _, err := svc(repo{}, produtos{err: errFalha}, eventos{}).Criar(ctx, admin(), in); !errors.Is(err, errFalha) {
		t.Fatal("produto ilegível")
	}
	if _, err := svc(repo{}, produtos{unidade: &outra}, eventos{}).Criar(ctx, admin(), in); !errors.Is(err, domain.ErrProdutoDeOutraUnidade) {
		t.Fatal("produto de outra unidade")
	}
	ok := produtos{unidade: &u}
	if _, err := svc(repo{insert: errFalha}, ok, eventos{}).Criar(ctx, admin(), in); err == nil {
		t.Fatal("insert")
	}
	if _, err := svc(repo{}, ok, eventos{errFalha}).Criar(ctx, admin(), in); err == nil {
		t.Fatal("evento")
	}

	if _, err := svc(repo{get: errFalha}, ok, eventos{}).Atualizar(ctx, admin(), atual.ID, in); err == nil {
		t.Fatal("get")
	}
	if _, err := svc(repo{ativo: atual}, ok, eventos{}).Atualizar(ctx, gestor(outra), atual.ID, in); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("fora do escopo")
	}
	if _, err := svc(repo{ativo: atual}, ok, eventos{}).Atualizar(ctx, admin(), atual.ID, domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal("inválida")
	}
	paraOutra := in
	paraOutra.UnidadeID = &outra
	if _, err := svc(repo{ativo: atual}, ok, eventos{}).Atualizar(ctx, gestor(u), atual.ID, paraOutra); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("mover para fora")
	}
	if _, err := svc(repo{ativo: atual}, produtos{unidade: &outra}, eventos{}).Atualizar(ctx, admin(), atual.ID, in); !errors.Is(err, domain.ErrProdutoDeOutraUnidade) {
		t.Fatal("produto na edição")
	}
	mudado := in
	mudado.Estado = "manutencao"
	if _, err := svc(repo{ativo: atual, update: errFalha}, ok, eventos{}).Atualizar(ctx, admin(), atual.ID, mudado); err == nil {
		t.Fatal("update")
	}
	if got, err := svc(repo{ativo: atual}, ok, eventos{}).Atualizar(ctx, admin(), atual.ID, mudado); err != nil || got.Estado != "manutencao" {
		t.Fatalf("atualizar: %+v %v", got, err)
	}
	semProduto := domain.Entrada{Nome: "Licença", Tipo: "software", Estado: "planejado"}
	if _, err := svc(repo{}, nil, eventos{}).Criar(ctx, admin(), semProduto); err != nil {
		t.Fatalf("sem produto, sem porta: %v", err)
	}
}

func TestHandlersComFalha(t *testing.T) {
	h := transport.NewHandlers(application.NewService(nil, repo{list: errFalha}, eventos{}, nil), slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodGet, "/ativos", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
}
