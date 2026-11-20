package application

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

func TestRecusaAntesDoBanco(t *testing.T) {
	ctx := context.Background()
	svc := NewService(nil, &memoria{}, nil)
	quem := auth.Identity{}
	if _, err := svc.Listar(ctx, quem, "outro"); !errors.Is(err, domain.ErrQuadro) {
		t.Fatal(err)
	}
	if _, err := svc.Listar(ctx, quem, "problemas"); !errors.Is(err, domain.ErrProibido) {
		t.Fatal(err)
	}
	if _, err := svc.Criar(ctx, quem, "outro", domain.Nova{}); !errors.Is(err, domain.ErrQuadro) {
		t.Fatal(err)
	}
	if _, err := svc.Criar(ctx, quem, "problemas", domain.Nova{}); !errors.Is(err, domain.ErrProibido) {
		t.Fatal(err)
	}
	if _, err := svc.Atualizar(ctx, quem, "outro", "registrado", domain.Patch{}); !errors.Is(err, domain.ErrQuadro) {
		t.Fatal(err)
	}
	if _, err := svc.Atualizar(ctx, quem, "problemas", "registrado", domain.Patch{}); !errors.Is(err, domain.ErrProibido) {
		t.Fatal(err)
	}
}

func TestListarPreenchePassagens(t *testing.T) {
	repo := &memoria{
		cols: colunasDeProblema(),
		mapa: map[string][]string{"registrado": {"em_analise"}, "outra": {"registrado"}},
	}
	cols, err := NewService(nil, repo, nil).Listar(context.Background(), gestor(), "problemas")
	if err != nil || len(cols) != 2 {
		t.Fatalf("%v %#v", err, cols)
	}
	if len(cols[0].Para) != 1 || cols[0].Para[0] != "em_analise" || len(cols[0].De) != 1 || cols[0].De[0] != "outra" {
		t.Fatalf("%#v", cols[0])
	}
	repo.falha = "listar"
	if _, err := NewService(nil, repo, nil).Listar(context.Background(), gestor(), "problemas"); err == nil {
		t.Fatal("listar")
	}
	repo.falha = "mapa"
	if _, err := NewService(nil, repo, nil).Listar(context.Background(), gestor(), "problemas"); err == nil {
		t.Fatal("mapa")
	}
}

func TestCriarEAtualizar(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := &memoria{cols: colunasDeProblema()}
	svc := NewService(pool, repo, outbox.NewWriter("nova.quadros"))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_type = 'quadro_coluna' AND aggregate_id LIKE 'problemas:c%'`)
	})
	criada, err := svc.Criar(ctx, gestor(), "problemas", domain.Nova{
		Nome: " Triagem ", Cor: "#112233", Ordem: 1, De: []string{"registrado"}, Para: []string{"em_analise"},
	})
	if err != nil || criada.Nome != "Triagem" || criada.Chave == "" {
		t.Fatalf("%v %#v", err, criada)
	}
	igual, err := svc.Atualizar(ctx, gestor(), "problemas", criada.Chave, domain.Patch{Nome: &criada.Nome})
	if err != nil || igual.Nome != criada.Nome {
		t.Fatalf("igual: %v %#v", err, igual)
	}
	nome := "Triagem nova"
	atual, err := svc.Atualizar(ctx, gestor(), "problemas", criada.Chave, domain.Patch{Nome: &nome})
	if err != nil || atual.Nome != nome {
		t.Fatalf("atualizar: %v %#v", err, atual)
	}
}

func TestCriarQueFalha(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	pedido := domain.Nova{Nome: "Triagem", Cor: "#112233", Ordem: 1, De: []string{"registrado"}, Para: []string{"em_analise"}}
	for _, falha := range []string{"listar", "soltar", "inserir", "salvar", "passagens"} {
		repo := &memoria{cols: colunasDeProblema(), falha: falha}
		if _, err := NewService(pool, repo, outbox.NewWriter("nova.quadros")).Criar(ctx, gestor(), "problemas", pedido); err == nil {
			t.Fatal(falha)
		}
	}
	repo := &memoria{cols: colunasDeProblema()}
	if _, err := NewService(pool, repo, eventos{errors.New("outbox")}).Criar(ctx, gestor(), "problemas", pedido); err == nil {
		t.Fatal("outbox")
	}
	ruim := pedido
	ruim.Cor = "azul"
	if _, err := NewService(pool, &memoria{cols: colunasDeProblema()}, outbox.NewWriter("nova.quadros")).Criar(ctx, gestor(), "problemas", ruim); !errors.Is(err, domain.ErrCor) {
		t.Fatal(err)
	}
	orig := lerEntropia
	t.Cleanup(func() { lerEntropia = orig })
	lerEntropia = func([]byte) (int, error) { return 0, errors.New("sem entropia") }
	if _, err := NewService(pool, &memoria{cols: colunasDeProblema()}, outbox.NewWriter("nova.quadros")).Criar(ctx, gestor(), "problemas", pedido); err == nil {
		t.Fatal("entropia")
	}
	lerEntropia = func(p []byte) (int, error) {
		for i := range p {
			p[i] = 0
		}
		return len(p), nil
	}
	ocupada := append(colunasDeProblema(), domain.Coluna{Quadro: "problemas", Chave: "c00000000", Nome: "Já", Cor: "#112233", Ordem: 2, Ativa: true})
	if _, err := NewService(pool, &memoria{cols: ocupada}, outbox.NewWriter("nova.quadros")).Criar(ctx, gestor(), "problemas", pedido); !errors.Is(err, domain.ErrPassagem) {
		t.Fatal(err)
	}
}

func TestAtualizarQueFalha(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	nome := "Outro nome"
	patch := domain.Patch{Nome: &nome}
	for _, falha := range []string{"listar", "contar", "soltar", "salvar"} {
		repo := &memoria{cols: colunasDeProblema(), falha: falha}
		if _, err := NewService(pool, repo, outbox.NewWriter("nova.quadros")).Atualizar(ctx, gestor(), "problemas", "em_analise", patch); err == nil {
			t.Fatal(falha)
		}
	}
	if _, err := NewService(pool, &memoria{cols: colunasDeProblema()}, eventos{errors.New("outbox")}).Atualizar(ctx, gestor(), "problemas", "em_analise", patch); err == nil {
		t.Fatal("outbox")
	}
	if _, err := NewService(pool, &memoria{cols: colunasDeProblema()}, outbox.NewWriter("nova.quadros")).Atualizar(ctx, gestor(), "problemas", "nao-existe", patch); !errors.Is(err, domain.ErrNaoEncontrada) {
		t.Fatal(err)
	}
	if _, err := NewService(pool, &memoria{cols: colunasDeProblema()}, outbox.NewWriter("nova.quadros")).Atualizar(ctx, gestor(), "problemas", "em_analise", domain.Patch{}); !errors.Is(err, domain.ErrPatchVazio) {
		t.Fatal(err)
	}
}

func gestor() auth.Identity {
	return auth.Identity{Subject: "ana", Permissions: []string{"problemas:manage"}}
}

func colunasDeProblema() []domain.Coluna {
	return []domain.Coluna{
		{Quadro: "problemas", Chave: "registrado", Nome: "Registrados", Tom: "info", Ordem: 0, Entrada: true, Ativa: true},
		{Quadro: "problemas", Chave: "em_analise", Nome: "Em análise", Tom: "info", Ordem: 1, Ativa: true},
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type eventos struct{ err error }

func (e eventos) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return e.err
}

type memoria struct {
	cols  []domain.Coluna
	mapa  map[string][]string
	falha string
}

func (m *memoria) estoura(op string) error {
	if m.falha == op {
		return errors.New(op)
	}
	return nil
}

func (m *memoria) Listar(context.Context, database.DBTX, string) ([]domain.Coluna, error) {
	if err := m.estoura("listar"); err != nil {
		return nil, err
	}
	return append([]domain.Coluna{}, m.cols...), nil
}

func (m *memoria) Salvar(_ context.Context, _ database.DBTX, cols []domain.Coluna) error {
	if err := m.estoura("salvar"); err != nil {
		return err
	}
	m.cols = append([]domain.Coluna{}, cols...)
	return nil
}

func (m *memoria) Inserir(_ context.Context, _ database.DBTX, c domain.Coluna) error {
	if err := m.estoura("inserir"); err != nil {
		return err
	}
	m.cols = append(m.cols, c)
	return nil
}

func (m *memoria) SoltarOrdens(context.Context, database.DBTX, string) error {
	return m.estoura("soltar")
}

func (m *memoria) Contar(context.Context, database.DBTX, string, string) (int, error) {
	return 0, m.estoura("contar")
}

func (m *memoria) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	if err := m.estoura("mapa"); err != nil {
		return nil, err
	}
	return m.mapa, nil
}

func (m *memoria) SubstituirPassagens(context.Context, database.DBTX, string, string, []domain.Passagem) error {
	return m.estoura("passagens")
}
