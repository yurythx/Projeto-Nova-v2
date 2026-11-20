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
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type trilhaQueFalha struct{}

func (trilhaQueFalha) Record(context.Context, audit.Entry) error {
	return errors.New("trilha")
}

func TestTrilhaQueNaoGrava(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	terceiro := dbtest.User(t, pool)
	for _, id := range []uuid.UUID{pessoa, outro, terceiro} {
		if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, id, unidade); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	repo := infrastructure.NewRepository()
	logger := slog.New(slog.DiscardHandler)
	svc := NewService(pool, repo, outbox.NewWriter("test"), logger)
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa}

	original := novaTrilha
	t.Cleanup(func() { novaTrilha = original })
	falha := func(pgx.Tx) registro { return trilhaQueFalha{} }
	novaTrilha = falha
	if _, err := svc.Criar(ctx, admin, entrada); err == nil {
		t.Fatal("criar")
	}
	if _, err := svc.CriarGuiado(ctx, admin, entrada, domain.EntradaCarta{Onde: "SEMA"}, []string{"Mapear"}); err == nil {
		t.Fatal("guiado")
	}

	novaTrilha = original
	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Única", "", &pessoa, "Ana", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "servidor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, outro); err != nil {
		t.Fatal(err)
	}
	horas, err := svc.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
	if err != nil {
		t.Fatal(err)
	}

	novaTrilha = falha
	if _, err := svc.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar"}); err == nil {
		t.Fatal("carta")
	}
	if _, err := svc.Atualizar(ctx, admin, p.ID, entrada); err == nil {
		t.Fatal("atualizar")
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err == nil {
		t.Fatal("passar")
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Outra", "", &pessoa, "Ana", domain.EstadoTarefaAFazer, 2, nil, nil, nil, nil); err == nil {
		t.Fatal("tarefa nova")
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &pessoa, "Ana", domain.EstadoTarefaEmAndamento, 1, nil, nil, nil); err == nil {
		t.Fatal("mover tarefa")
	}
	if _, err := svc.CriarAtualizacao(ctx, admin, p.ID, tarefa.ID, "nota de campo"); err == nil {
		t.Fatal("atualização")
	}
	if err := svc.ExcluirTarefa(ctx, admin, tarefa.ID, "Única"); err == nil {
		t.Fatal("excluir tarefa")
	}
	if _, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "outro"); err == nil {
		t.Fatal("vínculo")
	}
	if err := svc.Desvincular(ctx, admin, vinculo.ID); err == nil {
		t.Fatal("desvincular")
	}
	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, terceiro); err == nil {
		t.Fatal("incluir")
	}
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, outro); err == nil {
		t.Fatal("excluir pessoa")
	}
	if _, err := svc.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10); err == nil {
		t.Fatal("apontamento")
	}
	if err := svc.ExcluirApontamento(ctx, admin, horas.ID); err == nil {
		t.Fatal("apagar apontamento")
	}
	if _, err := svc.Cancelar(ctx, admin, p.ID, "saiu do plano"); err == nil {
		t.Fatal("cancelar")
	}
}

func TestAvisoQueNaoGrava(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	gerente := dbtest.User(t, pool)
	for _, id := range []uuid.UUID{pessoa, gerente} {
		if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, id, unidade); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, pessoa); err != nil {
		t.Fatal(err)
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	logger := slog.New(slog.DiscardHandler)
	svc := NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &gerente}

	original := gravarAviso
	t.Cleanup(func() { gravarAviso = original })

	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	limite := time.Now().Add(-48 * time.Hour).UTC()
	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Única", "", &gerente, "Gerente", domain.EstadoTarefaAFazer, 1, &limite, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fim := time.Now().Add(-72 * time.Hour).UTC()
	if _, err := pool.Exec(ctx, `UPDATE projetos SET data_fim_planejada = $2 WHERE id = $1`, p.ID, fim); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("aviso")
	}
	outroLimite := time.Now().Add(-96 * time.Hour).UTC()
	if _, err := pool.Exec(ctx, `UPDATE projetos_tarefas SET data_limite = $2 WHERE id = $1`, tarefa.ID, outroLimite); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos SET data_fim_planejada = $2 WHERE id = $1`, p.ID, time.Now().Add(240*time.Hour).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err == nil {
		t.Fatal("prazo da tarefa")
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos_tarefas SET data_limite = NULL WHERE id = $1`, tarefa.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos SET data_fim_planejada = $2 WHERE id = $1`, p.ID, time.Now().Add(-120*time.Hour).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err == nil {
		t.Fatal("prazo do projeto")
	}
	if _, err := svc.Atualizar(ctx, admin, p.ID, entrada); err == nil {
		t.Fatal("atualizar")
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err == nil {
		t.Fatal("passar")
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Outra", "", &gerente, "Gerente", domain.EstadoTarefaAFazer, 2, nil, nil, nil, nil); err == nil {
		t.Fatal("tarefa")
	}
	if _, err := svc.CriarAtualizacao(ctx, admin, p.ID, tarefa.ID, "nota de campo"); err == nil {
		t.Fatal("atualização")
	}
	if _, err := svc.Cancelar(ctx, admin, p.ID, "saiu do plano"); err == nil {
		t.Fatal("cancelar")
	}
}
