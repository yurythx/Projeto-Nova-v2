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

	apperrors "github.com/yurythx/sismarq/internal/domain/errors"
	"github.com/yurythx/sismarq/internal/modules/problemas/application"
	"github.com/yurythx/sismarq/internal/modules/problemas/domain"
	"github.com/yurythx/sismarq/internal/modules/problemas/transport"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/database/dbtest"
)

var errFalha = errors.New("falha")

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func com(u uuid.UUID, perms ...string) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{UnidadeID: &u, Permissions: perms, Unidades: []uuid.UUID{u}}}}
}

// repo em memória; falha[m] derruba o método m.
type repo struct {
	p      domain.Problema
	falha  map[string]bool
	ligado bool
}

func (r repo) err(m string) error {
	if r.falha[m] {
		return errFalha
	}
	return nil
}
func (r repo) List(context.Context, database.DBTX, domain.Filtro) ([]domain.Problema, error) {
	return []domain.Problema{}, r.err("List")
}
func (r repo) Get(context.Context, database.DBTX, uuid.UUID) (domain.Problema, error) {
	return r.p, r.err("Get")
}
func (r repo) Travar(context.Context, database.DBTX, uuid.UUID) error { return r.err("Travar") }
func (r repo) Insert(_ context.Context, _ database.DBTX, p domain.Problema) (domain.Problema, error) {
	return p, r.err("Insert")
}
func (r repo) Update(context.Context, database.DBTX, domain.Problema) error { return r.err("Update") }
func (r repo) Incidentes(context.Context, database.DBTX, uuid.UUID) ([]domain.IncidenteVinculado, error) {
	return []domain.IncidenteVinculado{}, r.err("Incidentes")
}
func (r repo) VincularIncidente(context.Context, database.DBTX, uuid.UUID, domain.IncidenteVinculado, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Vincular")
}
func (r repo) DesvincularIncidente(context.Context, database.DBTX, uuid.UUID, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Desvincular")
}
func (r repo) AtivosDe(context.Context, database.DBTX, uuid.UUID) ([]domain.AtivoVinculado, error) {
	return []domain.AtivoVinculado{{AtivoID: uuid.New()}}, r.err("AtivosDe")
}
func (r repo) VincularAtivo(context.Context, database.DBTX, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Vincular")
}
func (r repo) DesvincularAtivo(context.Context, database.DBTX, uuid.UUID, uuid.UUID) (bool, error) {
	return r.ligado, r.err("Desvincular")
}
func (r repo) JaVinculados(context.Context, database.DBTX, []uuid.UUID) (map[uuid.UUID]bool, error) {
	return map[uuid.UUID]bool{}, r.err("JaVinculados")
}

type eventos struct{ err error }

func (e eventos) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return e.err
}

type incidentes struct {
	inc    domain.Incidente
	err    error
	grupos []domain.Recorrencia
	rerr   error
}

func (i incidentes) Consultar(context.Context, database.DBTX, uuid.UUID) (domain.Incidente, error) {
	return i.inc, i.err
}
func (i incidentes) Recorrentes(context.Context, database.DBTX, bool, []uuid.UUID, time.Time, int) ([]domain.Recorrencia, error) {
	return i.grupos, i.rerr
}

type ativos struct {
	unidade *uuid.UUID
	err     error
}

func (a ativos) Consultar(context.Context, database.DBTX, uuid.UUID) (string, *uuid.UUID, error) {
	return "Switch", a.unidade, a.err
}

type produtos struct {
	unidade *uuid.UUID
	err     error
}

func (p produtos) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, p.err
}

func TestMapError(t *testing.T) {
	for _, err := range []error{domain.ErrNaoEncontrado, domain.ErrEntradaInvalida, domain.ErrForaDoEscopo, domain.ErrUnidadeInexistente,
		domain.ErrTransicaoInvalida, domain.ErrSemContorno, domain.ErrSemCausaRaiz, domain.ErrEncerrado, domain.ErrIncidenteInexistente,
		domain.ErrNaoIncidente, domain.ErrAtivoInexistente, domain.ErrDeOutraUnidade, domain.ErrIntegracaoDesligada, domain.ErrProdutoInexistente} {
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
	base, _ := domain.Novo(domain.Entrada{Titulo: "Quedas", UnidadeID: &u}, uuid.New())
	gestor, leitor := com(u, "problemas:manage"), com(u, "problemas:read")
	tudo := application.Portas{
		Incidentes: incidentes{inc: domain.Incidente{ID: uuid.New(), Tipo: "incidente", UnidadeID: &u}},
		Ativos:     ativos{unidade: &u}, Produtos: produtos{unidade: &u},
	}
	svc := func(r repo, p application.Portas, e eventos) *application.Service {
		return application.NewService(pool, r, e, p)
	}
	falhando := func(ms ...string) repo {
		r := repo{p: base, falha: map[string]bool{}, ligado: true}
		for _, m := range ms {
			r.falha[m] = true
		}
		return r
	}

	// Leitura.
	if lista, err := svc(falhando(), tudo, eventos{}).Listar(ctx, auth.Identity{}, application.FiltroEntrada{}); err != nil || len(lista) != 0 {
		t.Fatal("sem cobertura")
	}
	if _, err := svc(falhando(), tudo, eventos{}).Listar(ctx, leitor, application.FiltroEntrada{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc(falhando("List"), tudo, eventos{}).Listar(ctx, admin(), application.FiltroEntrada{}); err == nil {
		t.Fatal("lista")
	}
	for _, m := range []string{"Get", "Incidentes", "AtivosDe"} {
		if _, err := svc(falhando(m), tudo, eventos{}).Obter(ctx, admin(), base.ID); err == nil {
			t.Errorf("obter com falha em %s", m)
		}
	}
	if _, err := svc(falhando(), tudo, eventos{}).Obter(ctx, com(outra, "problemas:read"), base.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("fora do escopo")
	}
	if p, err := svc(falhando(), application.Portas{}, eventos{}).Obter(ctx, leitor, base.ID); err != nil || len(p.Proximos) != 0 || p.Ativos[0].Nome != "" {
		t.Fatalf("leitor sem portas: %+v %v", p, err)
	}

	// Criação.
	in := domain.Entrada{Titulo: "Novo", UnidadeID: &u, ProdutoID: &u}
	if _, err := svc(falhando(), tudo, eventos{}).Criar(ctx, gestor, domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), tudo, eventos{}).Criar(ctx, leitor, in); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), application.Portas{}, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, domain.ErrIntegracaoDesligada) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), application.Portas{Produtos: produtos{err: errFalha}}, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, errFalha) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), application.Portas{Produtos: produtos{unidade: &outra}}, eventos{}).Criar(ctx, gestor, in); !errors.Is(err, domain.ErrDeOutraUnidade) {
		t.Fatal(err)
	}
	if _, err := svc(falhando("Insert"), tudo, eventos{}).Criar(ctx, gestor, in); err == nil {
		t.Fatal("insert")
	}

	// alterar: trava, leitura e escopo.
	for _, m := range []string{"Travar", "Get"} {
		if _, err := svc(falhando(m), tudo, eventos{}).Transicionar(ctx, gestor, base.ID, domain.EmAnalise); err == nil {
			t.Errorf("alterar com falha em %s", m)
		}
	}
	if _, err := svc(falhando(), tudo, eventos{}).Transicionar(ctx, leitor, base.ID, domain.EmAnalise); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("leitor não altera")
	}
	if _, err := svc(falhando(), tudo, eventos{}).Transicionar(ctx, com(outra, "problemas:manage"), base.ID, domain.EmAnalise); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("outra unidade")
	}
	if _, err := svc(falhando("Update"), tudo, eventos{}).Transicionar(ctx, gestor, base.ID, domain.EmAnalise); err == nil {
		t.Fatal("update da passagem")
	}

	// Edição.
	editar := domain.Entrada{Titulo: "Editado", UnidadeID: &u}
	if _, err := svc(falhando(), tudo, eventos{}).Atualizar(ctx, gestor, base.ID, domain.Entrada{}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := svc(falhando(), tudo, eventos{}).Atualizar(ctx, gestor, base.ID, domain.Entrada{Titulo: "x", UnidadeID: &outra}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal("mover para fora")
	}
	if _, err := svc(falhando(), application.Portas{}, eventos{}).Atualizar(ctx, gestor, base.ID, domain.Entrada{Titulo: "x", UnidadeID: &u, ProdutoID: &u}); !errors.Is(err, domain.ErrIntegracaoDesligada) {
		t.Fatal("produto na edição")
	}
	if _, err := svc(falhando("Update"), tudo, eventos{}).Atualizar(ctx, gestor, base.ID, domain.Entrada{Titulo: "Quedas", UnidadeID: &u}); err != nil {
		t.Fatalf("edição sem mudança não grava: %v", err)
	}
	if _, err := svc(falhando("Update"), tudo, eventos{}).Atualizar(ctx, gestor, base.ID, editar); err == nil {
		t.Fatal("update")
	}
	if _, err := svc(falhando(), tudo, eventos{errFalha}).Atualizar(ctx, gestor, base.ID, editar); err == nil {
		t.Fatal("evento")
	}

	// Vínculos.
	alvo := uuid.New()
	encerrado := falhando()
	encerrado.p.Estado, encerrado.p.CausaRaiz = domain.Encerrado, "x"
	if _, err := svc(encerrado, tudo, eventos{}).VincularIncidente(ctx, gestor, base.ID, alvo, true); !errors.Is(err, domain.ErrEncerrado) {
		t.Fatal(err)
	}
	if _, err := svc(encerrado, tudo, eventos{}).VincularAtivo(ctx, gestor, base.ID, alvo, true); !errors.Is(err, domain.ErrEncerrado) {
		t.Fatal(err)
	}
	casos := map[string]application.Portas{
		"sem porta":     {},
		"inexistente":   {Incidentes: incidentes{err: domain.ErrIncidenteInexistente}},
		"requisição":    {Incidentes: incidentes{inc: domain.Incidente{Tipo: "requisicao", UnidadeID: &u}}},
		"outra unidade": {Incidentes: incidentes{inc: domain.Incidente{Tipo: "incidente", UnidadeID: &outra}}},
	}
	for nome, p := range casos {
		if _, err := svc(falhando(), p, eventos{}).VincularIncidente(ctx, gestor, base.ID, alvo, true); err == nil {
			t.Errorf("incidente %s", nome)
		}
	}
	for nome, p := range map[string]application.Portas{
		"sem porta": {}, "inexistente": {Ativos: ativos{err: domain.ErrAtivoInexistente}}, "outra unidade": {Ativos: ativos{unidade: &outra}},
	} {
		if _, err := svc(falhando(), p, eventos{}).VincularAtivo(ctx, gestor, base.ID, alvo, true); err == nil {
			t.Errorf("ativo %s", nome)
		}
	}
	for _, m := range []string{"Vincular", "Desvincular"} {
		ligar := m == "Vincular"
		if _, err := svc(falhando(m), tudo, eventos{}).VincularIncidente(ctx, gestor, base.ID, alvo, ligar); err == nil {
			t.Errorf("incidente com falha em %s", m)
		}
		if _, err := svc(falhando(m), tudo, eventos{}).VincularAtivo(ctx, gestor, base.ID, alvo, ligar); err == nil {
			t.Errorf("ativo com falha em %s", m)
		}
	}
	if _, err := svc(falhando(), tudo, eventos{errFalha}).VincularAtivo(ctx, gestor, base.ID, alvo, true); err == nil {
		t.Fatal("evento do vínculo")
	}

	// Sugestões.
	if lista, err := svc(falhando(), application.Portas{}, eventos{}).Sugestoes(ctx, gestor); err != nil || len(lista) != 0 {
		t.Fatal("sem porta, sem sugestão")
	}
	if lista, _ := svc(falhando(), tudo, eventos{}).Sugestoes(ctx, leitor); len(lista) != 0 {
		t.Fatal("leitor não recebe sugestão")
	}
	if _, err := svc(falhando(), application.Portas{Incidentes: incidentes{rerr: errFalha}}, eventos{}).Sugestoes(ctx, gestor); err == nil {
		t.Fatal("recorrentes")
	}
	grupo := domain.Recorrencia{Total: 3, Solicitacoes: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}}
	if _, err := svc(falhando("JaVinculados"), application.Portas{Incidentes: incidentes{grupos: []domain.Recorrencia{grupo}}}, eventos{}).Sugestoes(ctx, gestor); err == nil {
		t.Fatal("já vinculados")
	}
	if lista, err := svc(falhando(), application.Portas{Incidentes: incidentes{grupos: []domain.Recorrencia{grupo}}}, eventos{}).Sugestoes(ctx, gestor); err != nil || len(lista) != 1 {
		t.Fatalf("sugestão: %v %v", lista, err)
	}
}

func TestHandlersComFalha(t *testing.T) {
	r := repo{falha: map[string]bool{"List": true}}
	h := transport.NewHandlers(application.NewService(nil, r, eventos{}, application.Portas{Incidentes: incidentes{rerr: errFalha}}), slog.New(slog.DiscardHandler))
	for _, call := range []func(http.ResponseWriter, *http.Request){h.Listar, h.Sugestoes} {
		req := httptest.NewRequest(http.MethodGet, "/problemas", nil)
		req = req.WithContext(auth.WithIdentity(req.Context(), admin()))
		rec := httptest.NewRecorder()
		call(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status %d", rec.Code)
		}
	}
}
