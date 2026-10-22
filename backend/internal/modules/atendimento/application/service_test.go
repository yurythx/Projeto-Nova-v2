package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

func admin() auth.Identity { return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New()} }

func gestor(unidade uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &unidade, Permissions: []string{"atendimento:manage"}, Unidades: []uuid.UUID{unidade},
	}}}
}

// stubRepo cobre oferta e equipe; a solicitação é testada contra o banco.
type stubRepo struct {
	domain.SolicitacaoRepository
	oferta     domain.Oferta
	equipe     domain.Equipe
	getErr     error
	insertErr  error
	updateErr  error
	membrosErr error
	listErr    error
	ativos     int
	ativosErr  error
	candidatos []domain.Candidato
	candErr    error
	candNil    bool
}

func (s stubRepo) ListOfertas(context.Context, database.DBTX, domain.Filtro) ([]domain.Oferta, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.Oferta{}, nil
}
func (s stubRepo) GetOferta(context.Context, database.DBTX, uuid.UUID) (domain.Oferta, error) {
	if s.getErr != nil {
		return domain.Oferta{}, s.getErr
	}
	return s.oferta, nil
}
func (s stubRepo) InsertOferta(_ context.Context, _ database.DBTX, o domain.Oferta) (domain.Oferta, error) {
	if s.insertErr != nil {
		return domain.Oferta{}, s.insertErr
	}
	return o, nil
}
func (s stubRepo) UpdateOferta(_ context.Context, _ database.DBTX, o domain.Oferta) (domain.Oferta, error) {
	if s.updateErr != nil {
		return domain.Oferta{}, s.updateErr
	}
	return o, nil
}
func (s stubRepo) ListEquipes(context.Context, database.DBTX, domain.Filtro) ([]domain.Equipe, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.Equipe{}, nil
}
func (s stubRepo) GetEquipe(context.Context, database.DBTX, uuid.UUID) (domain.Equipe, error) {
	if s.getErr != nil {
		return domain.Equipe{}, s.getErr
	}
	return s.equipe, nil
}
func (s stubRepo) InsertEquipe(_ context.Context, _ database.DBTX, e domain.Equipe) (domain.Equipe, error) {
	if s.insertErr != nil {
		return domain.Equipe{}, s.insertErr
	}
	return e, nil
}
func (s stubRepo) UpdateEquipe(_ context.Context, _ database.DBTX, e domain.Equipe) (domain.Equipe, error) {
	if s.updateErr != nil {
		return domain.Equipe{}, s.updateErr
	}
	return e, nil
}
func (s stubRepo) SubstituirMembros(context.Context, database.DBTX, uuid.UUID, []uuid.UUID) error {
	return s.membrosErr
}
func (s stubRepo) ContarAtivos(context.Context, database.DBTX, []uuid.UUID) (int, error) {
	return s.ativos, s.ativosErr
}
func (s stubRepo) Candidatos(context.Context, database.DBTX, uuid.UUID) ([]domain.Candidato, error) {
	if s.candNil {
		return nil, s.candErr
	}
	if s.candidatos == nil {
		return []domain.Candidato{}, s.candErr
	}
	return s.candidatos, s.candErr
}

type fakeProdutos struct {
	unidade *uuid.UUID
	err     error
}

func (f fakeProdutos) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return f.unidade, f.err
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

func svc(repo stubRepo, produtos domain.Produtos) *application.Service {
	return application.NewService(nil, repo, fakeEventos{}, produtos)
}

func TestMapError(t *testing.T) {
	casos := []error{
		nil, domain.ErrNaoEncontrado, domain.ErrEntradaInvalida, domain.ErrForaDoEscopo,
		domain.ErrProdutoInexistente, domain.ErrProdutoIndisponivel, domain.ErrProdutoDeOutraUnidade,
		domain.ErrMembroInvalido, domain.ErrUnidadeInexistente, domain.ErrTipoInvalido,
		domain.ErrOfertaInexistente, domain.ErrOfertaInativa, domain.ErrSolicitacaoInvalida,
		domain.ErrSolicitanteInvalido, domain.ErrTransicaoInvalida, domain.ErrSemResponsavel,
		domain.ErrResponsavelInvalido, domain.ErrSolicitacaoFinalizada, domain.ErrComentarioInvalido,
		domain.ErrNotaInternaNegada, domain.ErrPoliticaInvalida, domain.ErrFeriadoInvalido, domain.ErrFeriadoRepetido,
		domain.ErrNivelInvalido, domain.ErrVisitaInvalida, domain.ErrLocalizacao, domain.ErrCapturaForaPrazo,
		domain.ErrVisitaAberta, domain.ErrVisitaEncerrada, domain.ErrVisitaDeOutro, domain.ErrEvidenciaInvalida,
	}
	for _, err := range casos {
		got := application.MapError(err)
		if err == nil {
			if got != nil {
				t.Fatal("nil")
			}
			continue
		}
		var app *apperrors.Error
		if !errors.As(got, &app) {
			t.Fatalf("%v não virou erro de API: %v", err, got)
		}
	}
	boom := errors.New("banco")
	if application.MapError(boom) != boom {
		t.Fatal("erro desconhecido")
	}
}

func TestRegrasSemBanco(t *testing.T) {
	ctx := context.Background()
	u := uuid.New()
	s := svc(stubRepo{}, nil)
	dept := uuid.New()
	soDept := auth.Identity{Scopes: []auth.Scope{{DepartamentoID: &dept, Permissions: []string{"atendimento:manage"}}}}
	if lista, err := s.ListarOfertas(ctx, soDept); err != nil || len(lista) != 0 {
		t.Fatalf("ofertas do departamento: %v %v", err, lista)
	}
	if lista, err := s.ListarEquipes(ctx, soDept); err != nil || len(lista) != 0 {
		t.Fatalf("equipes do departamento: %v %v", err, lista)
	}
	if _, err := svc(stubRepo{listErr: errors.New("x")}, nil).ListarOfertas(ctx, admin()); err == nil {
		t.Fatal("lista")
	}
	if _, err := svc(stubRepo{listErr: errors.New("x")}, nil).ListarEquipes(ctx, admin()); err == nil {
		t.Fatal("lista de equipes")
	}

	if _, err := s.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := s.CriarOferta(ctx, auth.Identity{}, application.OfertaEntrada{Nome: "E-mail"}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	produto := uuid.New()
	if _, err := s.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "E-mail", ProdutoID: &produto}); !errors.Is(err, domain.ErrProdutoIndisponivel) {
		t.Fatal(err)
	}
	comProduto := svc(stubRepo{}, fakeProdutos{err: domain.ErrProdutoInexistente})
	if _, err := comProduto.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "E-mail", ProdutoID: &produto}); !errors.Is(err, domain.ErrProdutoInexistente) {
		t.Fatal(err)
	}
	outra := uuid.New()
	comProduto = svc(stubRepo{}, fakeProdutos{unidade: &outra})
	if _, err := comProduto.CriarOferta(ctx, gestor(u), application.OfertaEntrada{Nome: "E-mail", ProdutoID: &produto, UnidadeID: &u}); !errors.Is(err, domain.ErrProdutoDeOutraUnidade) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{}, fakeProdutos{err: errors.New("banco")}).CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "E-mail", ProdutoID: &produto}); err == nil {
		t.Fatal("erro do produto")
	}

	oferta := domain.Oferta{ID: uuid.New(), Nome: "E-mail", Tipo: domain.TipoRequisicao, UnidadeID: &u, Ativa: true, ProdutoID: &produto}
	guardada := svc(stubRepo{oferta: oferta}, fakeProdutos{unidade: &u})
	if _, err := guardada.ObterOferta(ctx, gestor(uuid.New()), oferta.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{getErr: domain.ErrNaoEncontrado}, nil).ObterOferta(ctx, admin(), oferta.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if visto, err := guardada.ObterOferta(ctx, gestor(u), oferta.ID); err != nil || visto.Nome != "E-mail" {
		t.Fatal(err)
	}
	if _, err := guardada.AtualizarOferta(ctx, gestor(uuid.New()), oferta.ID, application.OfertaEntrada{Nome: "E-mail"}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := guardada.AtualizarOferta(ctx, gestor(u), oferta.ID, application.OfertaEntrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := guardada.AtualizarOferta(ctx, gestor(u), oferta.ID, application.OfertaEntrada{Nome: "E-mail", UnidadeID: &outra}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if got, err := guardada.AtualizarOferta(ctx, gestor(u), oferta.ID, application.OfertaEntrada{Nome: "E-mail", UnidadeID: &u, ProdutoID: &produto}); err != nil || !got.Igual(oferta) {
		t.Fatalf("edição vazia com o mesmo produto: %+v %v", got, err)
	}
	outroDono := svc(stubRepo{oferta: oferta}, fakeProdutos{unidade: &outra})
	if _, err := outroDono.AtualizarOferta(ctx, gestor(u), oferta.ID, application.OfertaEntrada{Nome: "E-mail", UnidadeID: &u, ProdutoID: &produto}); !errors.Is(err, domain.ErrProdutoDeOutraUnidade) {
		t.Fatalf("edição com produto de outra unidade: %v", err)
	}
	semProduto := svc(stubRepo{oferta: domain.Oferta{ID: oferta.ID, Nome: "E-mail", Tipo: domain.TipoRequisicao, UnidadeID: &u, Ativa: true}}, nil)
	if got, err := semProduto.AtualizarOferta(ctx, gestor(u), oferta.ID, application.OfertaEntrada{Nome: "E-mail", UnidadeID: &u}); err != nil || got.Nome != "E-mail" {
		t.Fatalf("edição vazia: %+v %v", got, err)
	}

	pessoa := uuid.New()
	if _, err := s.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := s.CriarEquipe(ctx, auth.Identity{}, application.EquipeEntrada{Nome: "N1"}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{ativos: 0}, nil).CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "N1", Membros: []uuid.UUID{pessoa}}); !errors.Is(err, domain.ErrMembroInvalido) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{ativosErr: errors.New("x")}, nil).CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "N1", Membros: []uuid.UUID{pessoa}}); err == nil {
		t.Fatal("contagem")
	}
	equipe := domain.Equipe{ID: uuid.New(), Nome: "N1", UnidadeID: &u, Membros: []uuid.UUID{}}
	eq := svc(stubRepo{equipe: equipe, ativos: 1}, nil)
	if _, err := eq.ObterEquipe(ctx, gestor(uuid.New()), equipe.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{getErr: errors.New("x")}, nil).ObterEquipe(ctx, admin(), equipe.ID); err == nil {
		t.Fatal(err)
	}
	if _, err := eq.AtualizarEquipe(ctx, gestor(uuid.New()), equipe.ID, application.EquipeEntrada{Nome: "N1"}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := eq.AtualizarEquipe(ctx, gestor(u), equipe.ID, application.EquipeEntrada{Nome: " "}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatal(err)
	}
	if _, err := eq.AtualizarEquipe(ctx, gestor(u), equipe.ID, application.EquipeEntrada{Nome: "N1", UnidadeID: &outra}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{equipe: equipe}, nil).AtualizarEquipe(ctx, gestor(u), equipe.ID, application.EquipeEntrada{Nome: "N1", UnidadeID: &u, Membros: []uuid.UUID{pessoa}}); !errors.Is(err, domain.ErrMembroInvalido) {
		t.Fatal(err)
	}
	if got, err := eq.AtualizarEquipe(ctx, gestor(u), equipe.ID, application.EquipeEntrada{Nome: "N1", UnidadeID: &u}); err != nil || got.Nome != "N1" {
		t.Fatalf("equipe igual: %+v %v", got, err)
	}
	if _, err := svc(stubRepo{getErr: errors.New("x")}, nil).AtualizarOferta(ctx, admin(), oferta.ID, application.OfertaEntrada{Nome: "E-mail"}); err == nil {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{getErr: errors.New("x")}, nil).AtualizarEquipe(ctx, admin(), equipe.ID, application.EquipeEntrada{Nome: "N1"}); err == nil {
		t.Fatal(err)
	}

	if _, err := s.Candidatos(ctx, auth.Identity{}, u); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := svc(stubRepo{candErr: errors.New("x")}, nil).Candidatos(ctx, admin(), u); err == nil {
		t.Fatal(err)
	}
	lista, err := svc(stubRepo{candNil: true}, nil).Candidatos(ctx, admin(), u)
	if err != nil || lista == nil || len(lista) != 0 {
		t.Fatalf("candidatos nulos viram lista vazia: %v %v", err, lista)
	}
	quem := uuid.New()
	achou, err := svc(stubRepo{candidatos: []domain.Candidato{{ID: quem, Nome: "Ana"}}}, nil).Candidatos(ctx, gestor(u), u)
	if err != nil || len(achou) != 1 || achou[0].Nome != "Ana" {
		t.Fatalf("candidatos: %+v %v", achou, err)
	}
}

func TestTransacao(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	repo := infrastructure.NewRepository()
	produtosRepo := produtoRepo{id: uuid.New(), unidade: &u}
	svc := application.NewService(pool, repo, outbox.NewWriter("nova.atendimento"), produtosRepo)
	quem := gestor(u)

	oferta, err := svc.CriarOferta(ctx, quem, application.OfertaEntrada{Nome: "E-mail " + uuid.NewString()[:6], UnidadeID: &u, ProdutoID: &produtosRepo.id})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, oferta.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_id = $1`, oferta.ID.String())
	})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type = $1 AND aggregate_id = $2`, application.EventoOfertaCriada, oferta.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("evento da oferta: %d %v", n, err)
	}
	if _, err := svc.AtualizarOferta(ctx, quem, oferta.ID, application.OfertaEntrada{Nome: "Acesso", Descricao: "conta", UnidadeID: &u, ProdutoID: &produtosRepo.id}); err != nil {
		t.Fatal(err)
	}

	equipe, err := svc.CriarEquipe(ctx, quem, application.EquipeEntrada{Nome: "N1 " + uuid.NewString()[:6], UnidadeID: &u, Membros: []uuid.UUID{pessoa}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, equipe.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_id = $1`, equipe.ID.String())
	})
	if len(equipe.Membros) != 1 {
		t.Fatalf("membros: %+v", equipe.Membros)
	}
	if _, err := svc.AtualizarEquipe(ctx, quem, equipe.ID, application.EquipeEntrada{Nome: "N2", UnidadeID: &u}); err != nil {
		t.Fatal(err)
	}

	falha := application.NewService(pool, stubRepo{insertErr: errors.New("insert")}, fakeEventos{}, nil)
	if _, err := falha.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "X"}); err == nil {
		t.Fatal("insert oferta")
	}
	if _, err := falha.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "X"}); err == nil {
		t.Fatal("insert equipe")
	}
	falha = application.NewService(pool, stubRepo{}, fakeEventos{err: errors.New("outbox")}, nil)
	if _, err := falha.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "X"}); err == nil {
		t.Fatal("outbox oferta")
	}
	if _, err := falha.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "X"}); err == nil {
		t.Fatal("outbox equipe")
	}
	falha = application.NewService(pool, stubRepo{membrosErr: errors.New("membros")}, fakeEventos{}, nil)
	if _, err := falha.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "X"}); err == nil {
		t.Fatal("membros")
	}
	falha = application.NewService(pool, stubRepo{oferta: oferta, updateErr: errors.New("update")}, fakeEventos{}, fakeProdutos{unidade: oferta.UnidadeID})
	if _, err := falha.AtualizarOferta(ctx, admin(), oferta.ID, application.OfertaEntrada{Nome: "Outro"}); err == nil {
		t.Fatal("update oferta")
	}
	falha = application.NewService(pool, stubRepo{equipe: equipe, updateErr: errors.New("update")}, fakeEventos{}, nil)
	if _, err := falha.AtualizarEquipe(ctx, admin(), equipe.ID, application.EquipeEntrada{Nome: "Outra"}); err == nil {
		t.Fatal("update equipe")
	}
	falha = application.NewService(pool, stubRepo{oferta: oferta}, fakeEventos{poison: true}, nil)
	if _, err := falha.AtualizarOferta(ctx, admin(), oferta.ID, application.OfertaEntrada{Nome: "Auditoria"}); err == nil {
		t.Fatal("auditoria")
	}
	falha = application.NewService(pool, stubRepo{equipe: equipe, membrosErr: errors.New("membros")}, fakeEventos{}, nil)
	if _, err := falha.AtualizarEquipe(ctx, admin(), equipe.ID, application.EquipeEntrada{Nome: "Outra"}); err == nil {
		t.Fatal("membros na edição")
	}
	falha = application.NewService(pool, stubRepo{equipe: equipe}, fakeEventos{err: errors.New("outbox")}, nil)
	if _, err := falha.AtualizarEquipe(ctx, admin(), equipe.ID, application.EquipeEntrada{Nome: "Outra"}); err == nil {
		t.Fatal("outbox da edição")
	}
}

// produtoRepo aponta toda consulta para a unidade dada. O id é estável
// para a oferta de teste referenciar um produto que "existe".
type produtoRepo struct {
	id      uuid.UUID
	unidade *uuid.UUID
}

func (p produtoRepo) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, nil
}
