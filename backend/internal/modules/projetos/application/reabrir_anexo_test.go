package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

func TestReabrirEAnexo(t *testing.T) {
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
	leitor := auth.Identity{UserID: pessoa, Name: "Ana Souza"}
	svc := NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), slog.New(slog.DiscardHandler))
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro}

	if nomeDe(auth.Identity{Username: "ana"}) != "ana" || nomeDe(auth.Identity{Email: "a@b.c"}) != "a@b.c" || nomeDe(auth.Identity{}) != "Usuário" {
		t.Fatal("nome")
	}
	if cortar(strings.Repeat("á", 5), 2) != "áá" {
		t.Fatal("corte")
	}
	if _, unidades := coberturaDe(auth.Identity{}); len(unidades) != 0 {
		t.Fatal("cobertura")
	}
	if ids, err := svc.responsaveisDoProjeto(ctx, pool, nil); err != nil || ids != nil {
		t.Fatal("sem projeto")
	}
	if _, err := svc.avisarSucessoras(ctx, nil, pessoa, &domain.Projeto{}, domain.Tarefa{}); err != nil {
		t.Fatal(err)
	}
	grupo, err := svc.fixarGrupo(ctx, pool, &domain.Projeto{GerenteID: &pessoa, GerenteNome: "Ana"}, &domain.Tarefa{
		ResponsavelNome: "Ana", Pessoas: []domain.Pessoa{{ID: pessoa, Nome: "Ana"}},
	}, nil, "Ana", nil)
	if err != nil || len(grupo) != 1 {
		t.Fatal(err)
	}
	grupo, err = svc.fixarGrupo(ctx, pool, &domain.Projeto{GerenteID: &pessoa, GerenteNome: "Ana"}, &domain.Tarefa{
		ResponsavelNome: "Ana", ResponsavelID: &pessoa,
	}, nil, "Ana", nil)
	if err != nil || len(grupo) != 1 {
		t.Fatal(err)
	}
	ruins := []domain.Pessoa{{Nome: "solta"}}
	if _, err := svc.fixarGrupo(ctx, pool, &domain.Projeto{}, nil, nil, "", &ruins); err == nil {
		t.Fatal("pessoa sem id")
	}
	repetidas := []domain.Pessoa{{ID: pessoa, Nome: "Ana"}, {ID: pessoa, Nome: "Ana"}, {}}
	if _, err := svc.fixarGrupo(ctx, pool, &domain.Projeto{GerenteID: &pessoa, GerenteNome: "Ana"}, nil, nil, "", &repetidas); err != nil {
		t.Fatal(err)
	}

	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Única", "", &outro, "Outro", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	filha, err := svc.CriarTarefa(ctx, admin, p.ID, "Filha", "", nil, "", domain.EstadoTarefaAFazer, 2, nil, nil, &tarefa.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	nada := uuid.Nil
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Solta", "", nil, "", "", 3, nil, &nada, &nada, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Pai sumido", "", nil, "", domain.EstadoTarefaAFazer, 4, nil, nil, &pessoa, nil); err == nil {
		t.Fatal("pai")
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Depois da filha", "", nil, "", domain.EstadoTarefaAFazer, 5, nil, &filha.ID, nil, nil); err == nil {
		t.Fatal("predecessora")
	}
	if _, err := svc.ListarAnexos(ctx, admin, uuid.New(), tarefa.ID); err == nil {
		t.Fatal("tarefa de outro projeto")
	}

	loja := storagetest.New()
	svc.ComArquivos(Arquivos{Store: loja, Bucket: "nova", MaxBytes: 10 << 20, Expiry: time.Minute})
	if _, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", "ausente"); err == nil {
		t.Fatal("arquivo ausente")
	}
	if _, err := svc.RegistrarAnexo(ctx, leitor, p.ID, tarefa.ID, "planta.pdf", "ausente"); err == nil {
		t.Fatal("leitor")
	}
	ticket, err := svc.PedirAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	semNome := admin
	semNome.Name = ""
	semNome.Username = "ana"
	anexo, err := svc.RegistrarAnexo(ctx, semNome, p.ID, tarefa.ID, "planta.pdf", ticket.ObjectKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListarAnexos(ctx, leitor, p.ID, tarefa.ID); err == nil {
		t.Fatal("lista")
	}
	if _, err := svc.URLAnexo(ctx, leitor, p.ID, tarefa.ID, anexo.ID); err == nil {
		t.Fatal("url")
	}
	if _, err := svc.URLAnexo(ctx, admin, p.ID, filha.ID, anexo.ID); err == nil {
		t.Fatal("anexo de outra tarefa")
	}
	if _, err := svc.ListarMovimentos(ctx, leitor, p.ID, tarefa.ID); err == nil {
		t.Fatal("movimentos")
	}

	original := novaTrilha
	t.Cleanup(func() { novaTrilha = original })
	novaTrilha = func(pgx.Tx) registro { return trilhaQueFalha{} }
	if _, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", ticket.ObjectKey); err == nil {
		t.Fatal("trilha do anexo")
	}
	novaTrilha = original
	svc.ComAvisos(notificacoes.NewService(pool, nil, slog.New(slog.DiscardHandler)))
	aviso := gravarAviso
	t.Cleanup(func() { gravarAviso = aviso })
	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("aviso")
	}
	if _, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "copia.pdf", ticket.ObjectKey); err == nil {
		t.Fatal("aviso do anexo")
	}
	gravarAviso = aviso

	if _, err := svc.AtualizarTarefa(ctx, admin, filha.ID, "Filha", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 2, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	abertas, err := svc.ListarTarefas(ctx, admin, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tk := range abertas {
		if tk.Estado == domain.EstadoTarefaConcluida {
			continue
		}
		if _, err := svc.AtualizarTarefa(ctx, admin, tk.ID, tk.Titulo, "", "", &outro, "Outro", domain.EstadoTarefaConcluida, tk.Ordem, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, pessoa); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, terceiro); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, outro); err == nil {
		t.Fatal("gerente")
	}
	naTarefa := []domain.Pessoa{{ID: terceiro, Nome: "Terceiro"}, {ID: outro, Nome: "Outro"}}
	if _, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, &naTarefa); err != nil {
		t.Fatal(err)
	}
	novaTrilha = func(pgx.Tx) registro { return trilhaQueFalha{} }
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, terceiro); err == nil {
		t.Fatal("saída")
	}
	novaTrilha = original
	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("aviso")
	}
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, terceiro); err == nil {
		t.Fatal("aviso da saída")
	}
	gravarAviso = aviso

	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "Tarde", "", nil, "", domain.EstadoTarefaAFazer, 9, nil, nil, nil, nil); err == nil {
		t.Fatal("encerrado")
	}
	if _, err := svc.ExcluirParticipante(ctx, leitor, p.ID, outro); err == nil {
		t.Fatal("leitor exclui")
	}
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, outro); err == nil {
		t.Fatal("gerente permanece")
	}

	svc.outbox = eventosQueFalham{}
	if _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); err == nil {
		t.Fatal("evento")
	}
	svc.outbox = outbox.NewWriter("test")
	novaTrilha = func(pgx.Tx) registro { return trilhaQueFalha{} }
	if _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); err == nil {
		t.Fatal("trilha")
	}
	novaTrilha = original
	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("aviso")
	}
	if _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); err == nil {
		t.Fatal("aviso")
	}
	gravarAviso = aviso
	semID := admin
	semID.UserID = uuid.Nil
	semID.Subject = "nao-e-uuid"
	semID.Name = ""
	semID.Email = "a@b.c"
	if _, err := svc.Reabrir(ctx, semID, p.ID, "voltou o escopo"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	semID.Subject = pessoa.String()
	semID.Username = ""
	if _, err := svc.Reabrir(ctx, semID, p.ID, "voltou de novo"); err != nil {
		t.Fatal(err)
	}
}
