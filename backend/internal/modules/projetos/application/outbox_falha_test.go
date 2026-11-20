package application

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type eventosQueFalham struct{}

func (eventosQueFalham) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return errors.New("outbox")
}

func TestEventoQueNaoGrava(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, pessoa); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, pessoa, unidade); err != nil {
		t.Fatal(err)
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	repo := infrastructure.NewRepository()
	logger := slog.New(slog.DiscardHandler)
	bom := NewService(pool, repo, outbox.NewWriter("test"), logger)
	ruim := NewService(pool, repo, outbox.NewWriter("test"), logger)
	ruim.outbox = eventosQueFalham{}
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa}

	if _, err := ruim.Criar(ctx, admin, entrada); err == nil {
		t.Fatal("criar")
	}
	if _, err := ruim.CriarGuiado(ctx, admin, entrada, domain.EntradaCarta{Onde: "SEMA"}, []string{"Mapear"}); err == nil {
		t.Fatal("guiado")
	}

	p, err := bom.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	if _, err := ruim.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar"}); err == nil {
		t.Fatal("carta")
	}
	if _, err := ruim.Atualizar(ctx, admin, p.ID, entrada); err == nil {
		t.Fatal("atualizar")
	}
	if _, err := ruim.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err == nil {
		t.Fatal("passar")
	}
	tarefa, err := bom.CriarTarefa(ctx, admin, p.ID, "Única", "", nil, "", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ruim.ExcluirTarefa(ctx, admin, tarefa.ID, "Única"); err == nil {
		t.Fatal("tarefa")
	}
	if _, err := ruim.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "servidor"); err == nil {
		t.Fatal("vínculo")
	}
	vinculo, err := bom.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "servidor")
	if err != nil {
		t.Fatal(err)
	}
	if err := ruim.Desvincular(ctx, admin, vinculo.ID); err == nil {
		t.Fatal("desvincular")
	}
	if _, err := bom.IncluirParticipante(ctx, admin, p.ID, pessoa); err != nil {
		t.Fatal(err)
	}
	horas, err := bom.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ruim.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10); err == nil {
		t.Fatal("apontamento")
	}
	if err := ruim.ExcluirApontamento(ctx, admin, horas.ID); err == nil {
		t.Fatal("apagar apontamento")
	}
	if _, err := ruim.Cancelar(ctx, admin, p.ID, "saiu do plano"); err == nil {
		t.Fatal("cancelar")
	}
}
