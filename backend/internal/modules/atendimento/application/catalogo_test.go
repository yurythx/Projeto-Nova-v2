package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

// catalogoFalho é o repositório real com falhas nas categorias.
type catalogoFalho struct {
	domain.Repository
	getErr, delErr error
}

func (c catalogoFalho) GetCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Categoria, error) {
	if c.getErr != nil {
		return domain.Categoria{}, c.getErr
	}
	return c.Repository.GetCategoria(ctx, db, id)
}

func (c catalogoFalho) DeleteCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if c.delErr != nil {
		return c.delErr
	}
	return c.Repository.DeleteCategoria(ctx, db, id)
}

func TestCatalogoEmBlocos(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	repo := infrastructure.NewRepository()
	s := application.NewService(pool, repo, outbox.NewWriter("nova.atendimento"), nil)
	quem := admin()
	sfx := uuid.NewString()[:8]

	c, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: " Acesso " + sfx, Descricao: "Contas e senhas", Icone: "key-round", Ordem: 1})
	if err != nil || !c.Ativa || c.Nome != "Acesso "+sfx {
		t.Fatalf("criar: %+v %v", c, err)
	}
	inativa, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: "Antiga " + sfx, Ativa: ptr(false)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_categorias WHERE id = ANY($1)`, []uuid.UUID{c.ID, inativa.ID})
	})
	todas, err := s.ListarCategorias(ctx, quem)
	if err != nil || !temCategoria(todas, c.ID) || !temCategoria(todas, inativa.ID) {
		t.Fatalf("todas: %v", err)
	}
	ativas, err := s.CategoriasDoCatalogo(ctx)
	if err != nil || !temCategoria(ativas, c.ID) || temCategoria(ativas, inativa.ID) {
		t.Fatalf("ativas: %v", err)
	}
	if c, err = s.AtualizarCategoria(ctx, quem, c.ID, application.CategoriaEntrada{Nome: "Acesso " + sfx, Icone: "lock", Ordem: 2}); err != nil || c.Icone != "lock" || !c.Ativa {
		t.Fatalf("atualizar: %+v %v", c, err)
	}

	// A oferta entra na categoria, com ícone e ordem; depois sai.
	o, err := s.CriarOferta(ctx, gestor(u), application.OfertaEntrada{Nome: "Senha " + sfx, UnidadeID: &u, CategoriaID: &c.ID, Icone: ptr("key"), Ordem: ptr(3)})
	if err != nil || o.CategoriaID == nil || *o.CategoriaID != c.ID || o.Icone != "key" || o.Ordem != 3 {
		t.Fatalf("oferta: %+v %v", o, err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID) })
	if o, err = s.AtualizarOferta(ctx, gestor(u), o.ID, application.OfertaEntrada{Nome: o.Nome, UnidadeID: &u}); err != nil || o.CategoriaID == nil || o.Icone != "key" {
		t.Fatalf("sem mexer no bloco, conserva: %+v %v", o, err)
	}
	if o, err = s.AtualizarOferta(ctx, gestor(u), o.ID, application.OfertaEntrada{Nome: o.Nome, UnidadeID: &u, CategoriaID: &uuid.Nil}); err != nil || o.CategoriaID != nil {
		t.Fatalf("tira da categoria: %+v %v", o, err)
	}
	if _, err := s.AtualizarOferta(ctx, gestor(u), o.ID, application.OfertaEntrada{Nome: o.Nome, UnidadeID: &u, CategoriaID: &c.ID}); err != nil {
		t.Fatal(err)
	}

	// Apagar a categoria leva a oferta para "Outros".
	if err := s.ApagarCategoria(ctx, quem, c.ID); err != nil {
		t.Fatal(err)
	}
	if o, err = repo.GetOferta(ctx, pool, o.ID); err != nil || o.CategoriaID != nil {
		t.Fatalf("oferta sem categoria: %+v %v", o, err)
	}

	// Regras.
	for nome, caso := range map[string]struct {
		err error
		run func() error
	}{
		"categoria é da instituição": {domain.ErrForaDoEscopo, func() error {
			_, err := s.CriarCategoria(ctx, gestor(u), application.CategoriaEntrada{Nome: "X"})
			return err
		}},
		"editar fora do escopo": {domain.ErrForaDoEscopo, func() error {
			_, err := s.AtualizarCategoria(ctx, gestor(u), inativa.ID, application.CategoriaEntrada{Nome: "X"})
			return err
		}},
		"apagar fora do escopo": {domain.ErrForaDoEscopo, func() error { return s.ApagarCategoria(ctx, gestor(u), inativa.ID) }},
		"listar sem gerir": {domain.ErrForaDoEscopo, func() error {
			_, err := s.ListarCategorias(ctx, auth.Identity{UserID: uuid.New()})
			return err
		}},
		"nome vazio": {domain.ErrApresentacaoInvalida, func() error {
			_, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: " "})
			return err
		}},
		"ícone inválido": {domain.ErrApresentacaoInvalida, func() error {
			_, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: "X", Icone: "<svg>"})
			return err
		}},
		"nome repetido": {domain.ErrCategoriaRepetida, func() error {
			_, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: "antiga " + sfx})
			return err
		}},
		"editar para nome repetido": {domain.ErrCategoriaRepetida, func() error {
			outra, err := s.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: "Outra " + sfx})
			if err != nil {
				return err
			}
			defer func() { _ = s.ApagarCategoria(ctx, quem, outra.ID) }()
			_, err = s.AtualizarCategoria(ctx, quem, outra.ID, application.CategoriaEntrada{Nome: "Antiga " + sfx})
			return err
		}},
		"editar inválida": {domain.ErrApresentacaoInvalida, func() error {
			_, err := s.AtualizarCategoria(ctx, quem, inativa.ID, application.CategoriaEntrada{Nome: "X", Ordem: 1000})
			return err
		}},
		"editar inexistente": {domain.ErrNaoEncontrado, func() error {
			_, err := s.AtualizarCategoria(ctx, quem, uuid.New(), application.CategoriaEntrada{Nome: "X"})
			return err
		}},
		"apagar inexistente": {domain.ErrNaoEncontrado, func() error { return s.ApagarCategoria(ctx, quem, uuid.New()) }},
		"oferta em categoria que não há": {domain.ErrCategoriaInexistente, func() error {
			_, err := s.CriarOferta(ctx, quem, application.OfertaEntrada{Nome: "X", CategoriaID: ptr(uuid.New())})
			return err
		}},
		"oferta com ícone inválido": {domain.ErrApresentacaoInvalida, func() error {
			_, err := s.AtualizarOferta(ctx, gestor(u), o.ID, application.OfertaEntrada{Nome: o.Nome, UnidadeID: &u, Icone: ptr("Key")})
			return err
		}},
		"oferta com o banco fora": {errBanco, func() error {
			falho := application.NewService(pool, catalogoFalho{Repository: repo, getErr: errBanco}, outbox.NewWriter("nova.atendimento"), nil)
			_, err := falho.CriarOferta(ctx, quem, application.OfertaEntrada{Nome: "X", CategoriaID: &inativa.ID})
			return err
		}},
		"apagar com o banco fora": {errBanco, func() error {
			falho := application.NewService(pool, catalogoFalho{Repository: repo, delErr: errBanco}, outbox.NewWriter("nova.atendimento"), nil)
			return falho.ApagarCategoria(ctx, quem, inativa.ID)
		}},
		"evento da edição falha": {errBanco, func() error {
			falho := application.NewService(pool, repo, fakeEventos{err: errBanco}, nil)
			_, err := falho.AtualizarCategoria(ctx, quem, inativa.ID, application.CategoriaEntrada{Nome: "Antiga " + sfx})
			return err
		}},
		"evento da criação falha": {errBanco, func() error {
			falho := application.NewService(pool, repo, fakeEventos{err: errBanco}, nil)
			_, err := falho.CriarCategoria(ctx, quem, application.CategoriaEntrada{Nome: "Nova " + sfx})
			return err
		}},
	} {
		if err := caso.run(); !errors.Is(err, caso.err) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

var errBanco = errors.New("banco fora")

func temCategoria(lista []domain.Categoria, id uuid.UUID) bool {
	for _, c := range lista {
		if c.ID == id {
			return true
		}
	}
	return false
}
