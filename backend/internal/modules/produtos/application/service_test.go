package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/sismarq/internal/domain/errors"
	"github.com/yurythx/sismarq/internal/modules/produtos/application"
	"github.com/yurythx/sismarq/internal/modules/produtos/domain"
	"github.com/yurythx/sismarq/internal/modules/produtos/infrastructure"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/database/dbtest"
	"github.com/yurythx/sismarq/internal/platform/outbox"
)

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func gestor(unidade uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &unidade, Permissions: []string{"produtos:manage"}, Unidades: []uuid.UUID{unidade},
	}}}
}

type stubRepo struct {
	produto   domain.Produto
	getErr    error
	insertErr error
	updateErr error
	listErr   error
}

func (s stubRepo) List(context.Context, database.DBTX, domain.Filtro) ([]domain.Produto, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.produto.ID == uuid.Nil {
		return []domain.Produto{}, nil
	}
	return []domain.Produto{s.produto}, nil
}
func (s stubRepo) Get(context.Context, database.DBTX, uuid.UUID) (domain.Produto, error) {
	if s.getErr != nil {
		return domain.Produto{}, s.getErr
	}
	return s.produto, nil
}
func (s stubRepo) Insert(_ context.Context, _ database.DBTX, p domain.Produto) (domain.Produto, error) {
	if s.insertErr != nil {
		return domain.Produto{}, s.insertErr
	}
	return p, nil
}
func (s stubRepo) Update(_ context.Context, _ database.DBTX, p domain.Produto) (domain.Produto, error) {
	if s.updateErr != nil {
		return domain.Produto{}, s.updateErr
	}
	return p, nil
}

type fakeEventos struct {
	err    error
	poison bool
}

func (f fakeEventos) Write(ctx context.Context, tx pgx.Tx, _, _, _ string, _ uuid.UUID, _ any) error {
	if f.poison {
		_, _ = tx.Exec(ctx, `SELECT 1/0`)
		return nil
	}
	return f.err
}

func entrada() application.Entrada {
	return application.Entrada{Nome: "Protocolo", Estagio: "operacao"}
}

func TestMapError(t *testing.T) {
	casos := []struct {
		err  error
		code string
	}{
		{nil, ""},
		{domain.ErrNaoEncontrado, "NOT_FOUND"},
		{domain.ErrEntradaInvalida, "VALIDATION_ERROR"},
		{domain.ErrEstagioInvalido, "VALIDATION_ERROR"},
		{domain.ErrForaDoEscopo, "FORBIDDEN"},
		{domain.ErrUnidadeInexistente, "VALIDATION_ERROR"},
	}
	for _, c := range casos {
		got := application.MapError(c.err)
		if c.err == nil {
			if got != nil {
				t.Fatal("nil")
			}
			continue
		}
		var app *apperrors.Error
		if !errors.As(got, &app) || string(app.Code) != c.code {
			t.Fatalf("%v → %v, want %s", c.err, got, c.code)
		}
	}
	boom := errors.New("banco")
	if application.MapError(boom) != boom {
		t.Fatal("erro desconhecido precisa seguir intacto")
	}
}

func TestRegrasSemBanco(t *testing.T) {
	ctx := context.Background()
	svc := application.NewService(nil, stubRepo{}, fakeEventos{})
	u := uuid.New()
	if _, err := svc.Criar(ctx, admin(), application.Entrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("nome: %v", err)
	}
	if _, err := svc.Criar(ctx, auth.Identity{}, entrada()); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("sem permissão: %v", err)
	}
	if _, err := svc.Criar(ctx, gestor(u), application.Entrada{Nome: "Protocolo", Estagio: "operacao", UnidadeID: &[]uuid.UUID{uuid.New()}[0]}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("outra unidade: %v", err)
	}
	dept := uuid.New()
	soDepartamento := auth.Identity{Scopes: []auth.Scope{{DepartamentoID: &dept, Permissions: []string{"produtos:manage"}}}}
	lista, err := svc.Listar(ctx, soDepartamento)
	if err != nil || len(lista) != 0 {
		t.Fatalf("departamento não gerencia produto da unidade: %v %v", err, lista)
	}
	if _, err := svc.Listar(ctx, auth.Identity{}); err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	prod := domain.Produto{ID: id, Nome: "Protocolo", Estagio: "operacao", UnidadeID: &u}
	com := application.NewService(nil, stubRepo{produto: prod}, fakeEventos{})
	if _, err := com.Obter(ctx, gestor(uuid.New()), id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("ler fora do escopo: %v", err)
	}
	if _, err := com.Obter(ctx, auth.Identity{}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("ler sem achar: espera 404, veio %v", err)
	}
	visto, err := com.Obter(ctx, gestor(u), id)
	if err != nil || visto.Nome != "Protocolo" {
		t.Fatalf("obter: %+v %v", visto, err)
	}
	if _, err := application.NewService(nil, stubRepo{getErr: domain.ErrNaoEncontrado}, fakeEventos{}).Obter(ctx, admin(), id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := application.NewService(nil, stubRepo{listErr: errors.New("x")}, fakeEventos{}).Listar(ctx, admin()); err == nil {
		t.Fatal("falha da lista")
	}

	if _, err := com.Atualizar(ctx, gestor(uuid.New()), id, entrada()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("editar o que não vê: %v", err)
	}
	if _, err := com.Atualizar(ctx, gestor(u), id, application.Entrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("editar nome: %v", err)
	}
	outra := uuid.New()
	if _, err := com.Atualizar(ctx, gestor(u), id, application.Entrada{Nome: "Protocolo", UnidadeID: &outra}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("mover para fora: %v", err)
	}
	igual, err := com.Atualizar(ctx, gestor(u), id, application.Entrada{Nome: "Protocolo", Estagio: "suporte", UnidadeID: &u})
	if err != nil || igual.Estagio != "operacao" {
		t.Fatalf("edição vazia não muda o estágio nem regrava: %+v %v", igual, err)
	}
	if _, err := com.MudarEstagio(ctx, gestor(uuid.New()), id, "suporte"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := com.MudarEstagio(ctx, gestor(u), id, "legado"); !errors.Is(err, domain.ErrEstagioInvalido) {
		t.Fatal(err)
	}
	mesmo, err := com.MudarEstagio(ctx, gestor(u), id, "operacao")
	if err != nil || mesmo.Estagio != "operacao" {
		t.Fatalf("mesmo estágio: %+v %v", mesmo, err)
	}
	if _, err := application.NewService(nil, stubRepo{getErr: errors.New("banco")}, fakeEventos{}).MudarEstagio(ctx, admin(), id, "suporte"); err == nil {
		t.Fatal("get")
	}
	if _, err := application.NewService(nil, stubRepo{getErr: errors.New("banco")}, fakeEventos{}).Atualizar(ctx, admin(), id, entrada()); err == nil {
		t.Fatal("get no atualizar")
	}
}

func TestTransacao(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	repo := infrastructure.NewRepository()
	svc := application.NewService(pool, repo, outbox.NewWriter("nexus.produtos"))
	quem := gestor(u)

	criado, err := svc.Criar(ctx, quem, application.Entrada{Nome: "Protocolo " + uuid.NewString()[:6], Estagio: "descoberta", UnidadeID: &u, Descricao: "registro"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM produtos WHERE id = $1`, criado.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_id = $1`, criado.ID.String())
	})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type = $1 AND aggregate_id = $2`, application.EventoCriado, criado.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("evento de criação: %d %v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action = $1 AND resource_id = $2`, application.EventoCriado, criado.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("auditoria: %d %v", n, err)
	}

	edit, err := svc.Atualizar(ctx, quem, criado.ID, application.Entrada{Nome: "Protocolo web", UnidadeID: &u, Utilidade: "registrar"})
	if err != nil || edit.Nome != "Protocolo web" || edit.Estagio != "descoberta" {
		t.Fatalf("atualizar: %+v %v", edit, err)
	}
	if _, err := svc.MudarEstagio(ctx, quem, criado.ID, "operacao"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type = $1 AND aggregate_id = $2`, application.EventoEstagio, criado.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("evento de estágio: %d %v", n, err)
	}
	if _, err := svc.MudarEstagio(ctx, quem, criado.ID, "operacao"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type = $1 AND aggregate_id = $2`, application.EventoEstagio, criado.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatal("repetir o estágio não emite outro evento")
	}

	falha := application.NewService(pool, stubRepo{insertErr: errors.New("insert")}, fakeEventos{})
	if _, err := falha.Criar(ctx, admin(), entrada()); err == nil {
		t.Fatal("insert")
	}
	falha = application.NewService(pool, stubRepo{}, fakeEventos{err: errors.New("outbox")})
	if _, err := falha.Criar(ctx, admin(), entrada()); err == nil {
		t.Fatal("outbox")
	}
	falha = application.NewService(pool, stubRepo{}, fakeEventos{poison: true})
	if _, err := falha.Criar(ctx, admin(), entrada()); err == nil {
		t.Fatal("auditoria")
	}
	falha = application.NewService(pool, stubRepo{produto: criado, updateErr: errors.New("update")}, fakeEventos{})
	if _, err := falha.Atualizar(ctx, admin(), criado.ID, application.Entrada{Nome: "Outro"}); err == nil {
		t.Fatal("update")
	}
	if _, err := falha.MudarEstagio(ctx, admin(), criado.ID, "suporte"); err == nil {
		t.Fatal("update do estágio")
	}
	falha = application.NewService(pool, stubRepo{produto: criado}, fakeEventos{err: errors.New("outbox")})
	if _, err := falha.MudarEstagio(ctx, admin(), criado.ID, "entrega"); err == nil {
		t.Fatal("outbox do estágio")
	}
	if _, err := falha.Atualizar(ctx, admin(), criado.ID, application.Entrada{Nome: "Outro nome"}); err == nil {
		t.Fatal("outbox da edição")
	}
}
