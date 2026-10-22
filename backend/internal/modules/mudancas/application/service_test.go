package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

var errFalha = errors.New("falha")

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func com(u uuid.UUID, perms ...string) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: &u, Permissions: perms, Unidades: []uuid.UUID{u}}}}
}

type repo struct {
	m      domain.Mudanca
	falha  map[string]bool
	ligado bool
}

func (r repo) err(m string) error {
	if r.falha[m] {
		return errFalha
	}
	return nil
}
func (r repo) List(context.Context, database.DBTX, domain.Filtro) ([]domain.Mudanca, error) {
	return []domain.Mudanca{}, r.err("List")
}
func (r repo) Get(context.Context, database.DBTX, uuid.UUID) (domain.Mudanca, error) {
	return r.m, r.err("Get")
}
func (r repo) Travar(context.Context, database.DBTX, uuid.UUID) error { return r.err("Travar") }
func (r repo) Insert(_ context.Context, _ database.DBTX, m domain.Mudanca) (domain.Mudanca, error) {
	return m, r.err("Insert")
}
func (r repo) Update(context.Context, database.DBTX, domain.Mudanca) error { return r.err("Update") }
func (r repo) Aprovacoes(context.Context, database.DBTX, uuid.UUID) ([]domain.Aprovacao, error) {
	return []domain.Aprovacao{}, r.err("Aprovacoes")
}
func (r repo) InsertAprovacao(context.Context, database.DBTX, uuid.UUID, domain.Aprovacao) error {
	return r.err("InsertAprovacao")
}
func (r repo) Vinculos(context.Context, database.DBTX, uuid.UUID) ([]domain.Vinculo, error) {
	return []domain.Vinculo{}, r.err("Vinculos")
}
func (r repo) Vincular(context.Context, database.DBTX, uuid.UUID, domain.Vinculo, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Vincular")
}
func (r repo) Desvincular(context.Context, database.DBTX, uuid.UUID, string, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Desvincular")
}

type eventos struct{ err error }

func (e eventos) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return e.err
}

type portas struct {
	alvo  domain.Alvo
	err   error
	ligar bool
}

func (p portas) Disponivel(string) bool { return p.ligar }
func (p portas) Consultar(context.Context, database.DBTX, string, uuid.UUID) (domain.Alvo, error) {
	return p.alvo, p.err
}

type produtos struct {
	unidade *uuid.UUID
	err     error
}

func (p produtos) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, p.err
}

func TestMapError(t *testing.T) {
	for _, err := range []error{domain.ErrNaoEncontrado, domain.ErrEntradaInvalida, domain.ErrTipoInvalido, domain.ErrForaDoEscopo,
		domain.ErrUnidadeInexistente, domain.ErrTransicaoInvalida, domain.ErrNaoEditavel, domain.ErrSemJanela, domain.ErrSemReversao,
		domain.ErrSemParecer, domain.ErrSemResultado, domain.ErrAutoAprovacao, domain.ErrVinculoInvalido, domain.ErrVinculoInexistente,
		domain.ErrDeOutraUnidade, domain.ErrIntegracaoDesligada, domain.ErrMudancaEncerrada} {
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
	ini := time.Now().Add(time.Hour)
	fim := ini.Add(time.Hour)
	in := domain.Entrada{Titulo: "Trocar fonte", Tipo: domain.TipoNormal, Risco: "baixo", PlanoReversao: "voltar",
		InicioPlanejado: &ini, FimPlanejado: &fim, UnidadeID: &u, ProdutoID: &u}
	gestor, aprovador := com(u, "mudancas:manage"), com(u, "mudancas:aprovar")
	base, _ := domain.Nova(in, gestor.UserID)
	ok := produtos{unidade: &u}
	svc := func(r repo, p domain.Portas, pr domain.Produtos, e eventos) *application.Service {
		return application.NewService(pool, r, e, p, pr)
	}
	falhando := func(ms ...string) repo {
		r := repo{m: base, falha: map[string]bool{}, ligado: true}
		for _, m := range ms {
			r.falha[m] = true
		}
		return r
	}

	// Leitura.
	if lista, err := svc(falhando(), nil, ok, eventos{}).Listar(ctx, auth.Identity{}, application.FiltroEntrada{}); err != nil || len(lista) != 0 {
		t.Fatal("sem cobertura")
	}
	if _, err := svc(falhando(), nil, ok, eventos{}).Listar(ctx, aprovador, application.FiltroEntrada{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc(falhando("List"), nil, ok, eventos{}).Listar(ctx, admin(), application.FiltroEntrada{}); err == nil {
		t.Fatal("lista")
	}
	for _, m := range []string{"Get", "Aprovacoes", "Vinculos"} {
		if _, err := svc(falhando(m), nil, ok, eventos{}).Obter(ctx, gestor, base.ID); err == nil {
			t.Errorf("obter com falha em %s", m)
		}
	}
	if _, err := svc(falhando(), nil, ok, eventos{}).Obter(ctx, com(outra, "mudancas:manage"), base.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("fora do escopo")
	}

	// Criação.
	if _, err := svc(falhando(), nil, ok, eventos{}).Criar(ctx, gestor, domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), nil, ok, eventos{}).Criar(ctx, aprovador, in); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), nil, nil, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, domain.ErrIntegracaoDesligada) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), nil, produtos{err: errFalha}, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, errFalha) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), nil, produtos{unidade: &outra}, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, domain.ErrDeOutraUnidade) {
		t.Fatal(err)
	}
	if _, err := svc(falhando("Insert"), nil, ok, eventos{}).Criar(ctx, gestor, in); err == nil {
		t.Fatal("insert")
	}

	// alterar.
	for _, m := range []string{"Travar", "Get"} {
		if _, err := svc(falhando(m), nil, ok, eventos{}).Transicionar(ctx, gestor, base.ID, domain.Cancelada, ""); err == nil {
			t.Errorf("alterar com falha em %s", m)
		}
	}
	if _, err := svc(falhando(), nil, ok, eventos{}).Transicionar(ctx, com(outra, "mudancas:manage"), base.ID, domain.Cancelada, ""); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("outra unidade")
	}
	if _, err := svc(falhando("Update"), nil, ok, eventos{}).Transicionar(ctx, gestor, base.ID, domain.EmAvaliacao, ""); err == nil {
		t.Fatal("update da passagem")
	}
	avaliacao := falhando("InsertAprovacao")
	avaliacao.m.Estado = domain.EmAvaliacao
	if _, err := svc(avaliacao, nil, ok, eventos{}).Transicionar(ctx, aprovador, base.ID, domain.Aprovada, ""); err == nil {
		t.Fatal("registro da decisão")
	}

	// Edição.
	if _, err := svc(falhando(), nil, ok, eventos{}).Atualizar(ctx, aprovador, base.ID, in); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("aprovador não edita")
	}
	if _, err := svc(falhando(), nil, ok, eventos{}).Atualizar(ctx, gestor, base.ID, domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	fora := in
	fora.UnidadeID = &outra
	if _, err := svc(falhando(), nil, ok, eventos{}).Atualizar(ctx, gestor, base.ID, fora); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("mover para fora")
	}
	if _, err := svc(falhando(), nil, nil, eventos{}).Atualizar(ctx, gestor, base.ID, in); !errors.Is(err, domain.ErrIntegracaoDesligada) {
		t.Fatal("produto na edição")
	}
	if _, err := svc(falhando("Update"), nil, ok, eventos{}).Atualizar(ctx, gestor, base.ID, in); err != nil {
		t.Fatalf("edição sem mudança não grava: %v", err)
	}
	mudada := in
	mudada.Risco = "alto"
	if _, err := svc(falhando("Update"), nil, ok, eventos{}).Atualizar(ctx, gestor, base.ID, mudada); err == nil {
		t.Fatal("update")
	}

	// Vínculos.
	alvo := uuid.New()
	vincular := func(r repo, p domain.Portas, quem auth.Identity, tipo string, ligar bool) error {
		_, err := svc(r, p, ok, eventos{}).Vincular(ctx, quem, base.ID, tipo, alvo, ligar)
		return err
	}
	disponivel := portas{ligar: true, alvo: domain.Alvo{Rotulo: "x", UnidadeID: &u, Tipo: "incidente"}}
	if err := vincular(falhando(), disponivel, aprovador, domain.VinculoAtivo, true); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("aprovador não vincula")
	}
	if err := vincular(falhando(), disponivel, gestor, "cometa", true); !errors.Is(err, domain.ErrVinculoInvalido) {
		t.Fatal(err)
	}
	cancelada := falhando()
	cancelada.m.Estado = domain.Cancelada
	if err := vincular(cancelada, disponivel, gestor, domain.VinculoAtivo, true); !errors.Is(err, domain.ErrMudancaEncerrada) {
		t.Fatal(err)
	}
	for nome, p := range map[string]domain.Portas{
		"sem portas":    nil,
		"desligado":     portas{},
		"inexistente":   portas{ligar: true, err: domain.ErrVinculoInexistente},
		"outra unidade": portas{ligar: true, alvo: domain.Alvo{UnidadeID: &outra}},
	} {
		if err := vincular(falhando(), p, gestor, domain.VinculoAtivo, true); err == nil {
			t.Errorf("vínculo %s", nome)
		}
	}
	if err := vincular(falhando(), portas{ligar: true, alvo: domain.Alvo{UnidadeID: &u, Tipo: "requisicao"}}, gestor, domain.VinculoIncidente, true); !errors.Is(err, domain.ErrVinculoInvalido) {
		t.Fatal("requisição não se liga")
	}
	if err := vincular(falhando("Vincular"), disponivel, gestor, domain.VinculoIncidente, true); err == nil {
		t.Fatal("vincular")
	}
	if err := vincular(falhando("Desvincular"), disponivel, gestor, domain.VinculoAtivo, false); err == nil {
		t.Fatal("desvincular")
	}
	if _, err := svc(falhando(), disponivel, ok, eventos{errFalha}).Vincular(ctx, gestor, base.ID, domain.VinculoAtivo, alvo, true); err == nil {
		t.Fatal("evento do vínculo")
	}
	semMudar := falhando()
	semMudar.ligado = false
	if err := vincular(semMudar, disponivel, gestor, domain.VinculoAtivo, false); err != nil {
		t.Fatal("desvincular o que não havia não é erro")
	}
}

func TestHandlersComFalha(t *testing.T) {
	r := repo{falha: map[string]bool{"List": true}}
	h := transport.NewHandlers(application.NewService(nil, r, eventos{}, nil, nil), slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodGet, "/mudancas", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
}
