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

type falhaRepo struct {
	domain.Repository
	op string
}

func (f falhaRepo) nega(nome string) bool { return f.op == nome }

func (f falhaRepo) SalvarCarta(ctx context.Context, db database.DBTX, projetoID uuid.UUID, c domain.EntradaCarta) (domain.Carta, error) {
	if f.nega("carta") {
		return domain.Carta{}, errors.New("carta")
	}
	return f.Repository.SalvarCarta(ctx, db, projetoID, c)
}

func (f falhaRepo) SubstituirPessoasDaTarefa(ctx context.Context, db database.DBTX, tarefaID uuid.UUID, pessoas []domain.Pessoa) error {
	if f.nega("pessoas") {
		return errors.New("pessoas")
	}
	return f.Repository.SubstituirPessoasDaTarefa(ctx, db, tarefaID, pessoas)
}

func (f falhaRepo) ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	if f.nega("tarefas") {
		return nil, errors.New("tarefas")
	}
	return f.Repository.ListarTarefas(ctx, db, projetoID)
}

func (f falhaRepo) Aprovadores(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]uuid.UUID, error) {
	if f.nega("aprovadores") {
		return nil, errors.New("aprovadores")
	}
	return f.Repository.Aprovadores(ctx, db, unidade)
}

func (f falhaRepo) ListarVinculos(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Vinculo, error) {
	if f.nega("vinculos") {
		return nil, errors.New("vinculos")
	}
	return f.Repository.ListarVinculos(ctx, db, projetoID)
}

func (f falhaRepo) PosicaoUnidade(ctx context.Context, db database.DBTX, unidade uuid.UUID) (*uuid.UUID, []uuid.UUID, error) {
	if f.nega("posicao") {
		return nil, nil, errors.New("posicao")
	}
	return f.Repository.PosicaoUnidade(ctx, db, unidade)
}

func (f falhaRepo) SalvarReabertura(ctx context.Context, db database.DBTX, r *domain.Reabertura) error {
	if f.nega("reabertura") {
		return errors.New("reabertura")
	}
	return f.Repository.SalvarReabertura(ctx, db, r)
}

func (f falhaRepo) SalvarCancelamento(ctx context.Context, db database.DBTX, c *domain.Cancelamento) error {
	if f.nega("cancelamento") {
		return errors.New("cancelamento")
	}
	return f.Repository.SalvarCancelamento(ctx, db, c)
}

func (f falhaRepo) UltimoCancelamento(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (domain.Cancelamento, bool, error) {
	if f.nega("ultimo") {
		return domain.Cancelamento{}, false, errors.New("ultimo")
	}
	return f.Repository.UltimoCancelamento(ctx, db, projetoID)
}

func (f falhaRepo) Transicionar(ctx context.Context, db database.DBTX, id uuid.UUID, novoEstado string) (*domain.Projeto, error) {
	if f.nega("transicao") {
		return nil, errors.New("transicao")
	}
	return f.Repository.Transicionar(ctx, db, id, novoEstado)
}

func (f falhaRepo) Obter(ctx context.Context, db database.DBTX, id uuid.UUID) (*domain.Projeto, error) {
	if f.nega("obter") {
		return nil, errors.New("obter")
	}
	return f.Repository.Obter(ctx, db, id)
}

func (f falhaRepo) ExcluirTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.nega("excluir") {
		return errors.New("excluir")
	}
	return f.Repository.ExcluirTarefa(ctx, db, id)
}

func (f falhaRepo) NomeDaPessoa(ctx context.Context, db database.DBTX, id uuid.UUID, unidade *uuid.UUID) (string, error) {
	if f.nega("nome") {
		return "", errors.New("nome")
	}
	return f.Repository.NomeDaPessoa(ctx, db, id, unidade)
}

func (f falhaRepo) EhParticipante(ctx context.Context, db database.DBTX, projetoID, id uuid.UUID) (bool, error) {
	if f.nega("participante") {
		return false, errors.New("participante")
	}
	return f.Repository.EhParticipante(ctx, db, projetoID, id)
}

// tarefasQueSomem devolve a pessoa na primeira lista e some na segunda,
// para o ramo em que a tarefa já não está no projeto depois da saída.
type tarefasQueSomem struct {
	domain.Repository
	pessoa uuid.UUID
	n      int
}

func (t *tarefasQueSomem) ListarTarefas(ctx context.Context, db database.DBTX, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	t.n++
	if t.n == 1 {
		return []domain.Tarefa{{ID: uuid.New(), Titulo: "Fantasma", ResponsavelID: &t.pessoa}}, nil
	}
	return nil, nil
}

type produtoQueFalha struct{}

func (produtoQueFalha) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return nil, errors.New("produto")
}

type mapaQueFalha struct{}

func (mapaQueFalha) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	return nil, errors.New("mapa")
}

type horasFixas struct{}

func (horasFixas) Somar(context.Context, database.DBTX, []uuid.UUID) (float64, error) {
	return 1, nil
}

func querErro(t *testing.T, err error, oque string) {
	t.Helper()
	if err == nil {
		t.Fatal(oque)
	}
}

func TestRamosRestantes(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	fora := dbtest.User(t, pool)
	for _, id := range []uuid.UUID{pessoa, outro} {
		if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, id, unidade); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	leitor := auth.Identity{UserID: pessoa, Name: "Ana Souza"}
	real := infrastructure.NewRepository()
	logger := slog.New(slog.DiscardHandler)
	svc := NewService(pool, real, outbox.NewWriter("test"), logger)
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro}
	nada := uuid.Nil
	guardado := pessoa
	if _, nome, err := svc.fixarPessoa(ctx, pool, nil, &nada, "  ", "", nil); err != nil || nome != "" {
		t.Fatal(err, nome)
	}
	if id, _, err := svc.fixarPessoa(ctx, pool, nil, nil, "Ana", "Ana", &guardado); err != nil || id == nil || *id != pessoa {
		t.Fatal(err)
	}
	if svc.podeComentar(leitor, nil, &domain.Tarefa{}) {
		t.Fatal("comentar")
	}
	if _, err := svc.avisarTrocaDeGerente(ctx, nil, pessoa, nil, nil, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.avisarTarefa(ctx, nil, pessoa, nil, domain.Tarefa{}, domain.Tarefa{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.avisarAtualizacao(ctx, nil, pessoa, nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	if p, err := svc.comProximos(ctx, pool, nil); err != nil || p != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.podeReabrir(ctx, pool, admin, nil); err != nil {
		t.Fatal(err)
	}

	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	querErro(t, func() error { _, err := svc.ObterCarta(ctx, leitor, p.ID); return err }(), "carta")
	querErro(t, func() error {
		_, err := svc.SalvarCarta(ctx, leitor, p.ID, domain.Plano5W2H{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar"})
		return err
	}(), "salvar")
	querErro(t, func() error { _, err := svc.Atualizar(ctx, leitor, p.ID, entrada); return err }(), "atualizar")
	querErro(t, func() error {
		_, err := svc.Vincular(ctx, leitor, p.ID, domain.VinculoSolicitacao, uuid.New(), "chamado")
		return err
	}(), "vínculo")
	querErro(t, func() error { _, err := svc.ListarVinculos(ctx, leitor, p.ID); return err }(), "lista vínculo")
	querErro(t, func() error { _, _, err := svc.ListarApontamentos(ctx, leitor, p.ID); return err }(), "lista horas")
	querErro(t, func() error { _, err := svc.IncluirParticipante(ctx, leitor, p.ID, outro); return err }(), "incluir")
	querErro(t, func() error { _, err := svc.ListarTarefas(ctx, leitor, p.ID); return err }(), "tarefas")
	querErro(t, func() error { _, err := svc.ListarResponsaveis(ctx, leitor, p.ID); return err }(), "responsáveis")

	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Única", "", &outro, "Outro", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	querErro(t, func() error {
		_, err := svc.CriarTarefa(ctx, admin, p.ID, "Andando", "", nil, "", domain.EstadoTarefaEmAndamento, 2, nil, nil, nil, nil)
		return err
	}(), "sem responsável")
	querErro(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, leitor, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaAFazer, 1, nil, nil, nil)
		return err
	}(), "editar")
	querErro(t, svc.ExcluirTarefa(ctx, leitor, tarefa.ID, "Única"), "apagar")
	querErro(t, func() error { _, err := svc.ListarAtualizacoes(ctx, leitor, p.ID, tarefa.ID); return err }(), "notas")
	querErro(t, func() error {
		_, err := svc.CriarAtualizacao(ctx, leitor, p.ID, tarefa.ID, "nota")
		return err
	}(), "nota")
	querErro(t, func() error {
		_, err := svc.RegistrarApontamento(ctx, leitor, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
		return err
	}(), "horas")

	seguinte, err := svc.CriarTarefa(ctx, admin, p.ID, "Depois", "", &outro, "Outro", domain.EstadoTarefaAFazer, 3, nil, &tarefa.ID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	aviso := gravarAviso
	t.Cleanup(func() { gravarAviso = aviso })
	vezes := 0
	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		vezes++
		if vezes > 1 {
			return nil, errors.New("aviso")
		}
		return nil, nil
	}
	querErro(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, nil)
		return err
	}(), "sucessora")
	gravarAviso = aviso
	if _, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("aviso")
	}
	querErro(t, func() error {
		_, err := svc.CriarTarefa(ctx, admin, p.ID, "Nova", "", nil, "", domain.EstadoTarefaAFazer, 4, nil, nil, nil, nil)
		return err
	}(), "aviso da tarefa")
	querErro(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Renomeada", "", "", &outro, "Outro", domain.EstadoTarefaAFazer, 3, nil, nil, nil)
		return err
	}(), "aviso da mudança")
	querErro(t, func() error { _, err := svc.Atualizar(ctx, admin, p.ID, entrada); return err }(), "aviso do gerente")
	outroGerente := entrada
	outroGerente.GerenteID = &pessoa
	querErro(t, func() error { _, err := svc.Atualizar(ctx, admin, p.ID, outroGerente); return err }(), "troca")
	gravarAviso = aviso

	if _, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoSolicitacao, uuid.New(), "chamado"); err != nil {
		t.Fatal(err)
	}
	svc.ComHoras(horasFixas{})
	svc.repo = falhaRepo{Repository: real, op: "vinculos"}
	querErro(t, func() error { _, err := svc.ObterCarta(ctx, admin, p.ID); return err }(), "horas")
	svc.repo = real
	if _, err := svc.ObterCarta(ctx, admin, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos_tarefas SET responsavel_id = $2 WHERE id = $1`, seguinte.ID, fora); err != nil {
		t.Fatal(err)
	}
	if err := database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		longo := *p
		longo.Titulo = strings.Repeat("a", 180)
		if _, err := svc.avisarAprovadores(ctx, tx, &longo, pessoa); err != nil {
			return err
		}
		svc.repo = falhaRepo{Repository: real, op: "aprovadores"}
		_, err := svc.avisarAprovadores(ctx, tx, p, pessoa)
		svc.repo = real
		if err == nil {
			return errors.New("aprovadores")
		}
		svc.repo = falhaRepo{Repository: real, op: "tarefas"}
		_, err = svc.avisarAprovadores(ctx, tx, p, pessoa)
		svc.repo = real
		if err == nil {
			return errors.New("tarefas")
		}
		svc.repo = falhaRepo{Repository: real, op: "vinculos"}
		_, err = svc.avisarAprovadores(ctx, tx, p, pessoa)
		svc.repo = real
		if err == nil {
			return errors.New("vínculos")
		}
		if _, err := svc.avisarReabertura(ctx, tx, p, "voltou", pessoa, ""); err != nil {
			return err
		}
		svc.repo = falhaRepo{Repository: real, op: "tarefas"}
		_, err = svc.avisarReabertura(ctx, tx, p, "voltou", pessoa, "Ana")
		svc.repo = real
		if err == nil {
			return errors.New("reabertura")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoProblema, uuid.New(), "problema"); err != nil {
		t.Fatal(err)
	}
	if err := database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.avisarAprovadores(ctx, tx, p, pessoa)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	svc.ComPassagens(mapaQueFalha{})
	querErro(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Depois", "", "", &outro, "Outro", domain.EstadoTarefaAFazer, 3, nil, nil, nil)
		return err
	}(), "mapa")
	extra, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Extra", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, extra.ID)
	})
	querErro(t, func() error { _, err := svc.Cancelar(ctx, admin, extra.ID, "saiu do plano"); return err }(), "mapa do cancelamento")
	svc.ComPassagens(nil)
	svc.repo = falhaRepo{Repository: real, op: "transicao"}
	querErro(t, func() error { _, err := svc.Cancelar(ctx, admin, extra.ID, "saiu do plano"); return err }(), "transição")
	svc.repo = falhaRepo{Repository: real, op: "cancelamento"}
	querErro(t, func() error { _, err := svc.Cancelar(ctx, admin, extra.ID, "saiu do plano"); return err }(), "gravação")
	svc.repo = real
	cancelado := *extra
	cancelado.Estado = domain.EstadoCancelado
	svc.repo = falhaRepo{Repository: real, op: "ultimo"}
	if _, err := svc.comCancelamento(ctx, pool, &cancelado); err == nil {
		t.Fatal("último")
	}
	svc.repo = real

	svc.repo = falhaRepo{Repository: real, op: "carta"}
	querErro(t, func() error {
		_, err := svc.CriarGuiado(ctx, admin, entrada, domain.EntradaCarta{Onde: "SEMA"}, []string{"Mapear"})
		return err
	}(), "carta guiada")
	svc.repo = falhaRepo{Repository: real, op: "pessoas"}
	querErro(t, func() error {
		_, err := svc.CriarGuiado(ctx, admin, entrada, domain.EntradaCarta{Onde: "SEMA"}, []string{"Mapear"})
		return err
	}(), "pessoas guiadas")
	svc.repo = real

	svc.repo = falhaRepo{Repository: real, op: "obter"}
	querErro(t, svc.ExcluirTarefa(ctx, admin, tarefa.ID, "Única"), "obter na exclusão")
	svc.repo = falhaRepo{Repository: real, op: "excluir"}
	querErro(t, svc.ExcluirTarefa(ctx, admin, tarefa.ID, "Única"), "excluir")
	svc.repo = real
	loja := storagetest.New()
	svc.ComArquivos(Arquivos{Store: loja, Bucket: "nova", MaxBytes: 10 << 20, Expiry: time.Minute})
	ticket, err := svc.PedirAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	querErro(t, func() error {
		_, err := svc.RegistrarAnexo(ctx, leitor, p.ID, tarefa.ID, "planta.pdf", ticket.ObjectKey)
		return err
	}(), "leitor anexa")

	querErro(t, func() error { _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); return err }(), "abertas")
	if _, err := svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Depois", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 3, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Obter(ctx, admin, p.ID); err != nil {
		t.Fatal(err)
	}
	svc.repo = falhaRepo{Repository: real, op: "posicao"}
	querErro(t, func() error { _, err := svc.Obter(ctx, admin, p.ID); return err }(), "posição")
	querErro(t, func() error { _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); return err }(), "posição na reabertura")
	svc.repo = falhaRepo{Repository: real, op: "reabertura"}
	querErro(t, func() error { _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); return err }(), "grava a reabertura")
	svc.repo = real

	querErro(t, func() error {
		_, err := svc.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar"})
		return err
	}(), "carta encerrada")
	querErro(t, func() error { _, err := svc.Atualizar(ctx, admin, p.ID, entrada); return err }(), "ficha encerrada")
	querErro(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Única", "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, nil)
		return err
	}(), "tarefa encerrada")
	querErro(t, svc.ExcluirTarefa(ctx, admin, tarefa.ID, "Única"), "exclusão encerrada")
	querErro(t, func() error {
		_, err := svc.CriarAtualizacao(ctx, admin, p.ID, tarefa.ID, "tarde")
		return err
	}(), "nota encerrada")
	querErro(t, func() error { _, err := svc.IncluirParticipante(ctx, admin, p.ID, pessoa); return err }(), "inclusão encerrada")
	querErro(t, func() error {
		_, err := svc.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
		return err
	}(), "horas encerradas")
	horas, err := svc.RegistrarApontamento(ctx, admin, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
	if err == nil {
		t.Fatal("apontamento em encerrado")
	}
	_ = horas
	vinculo, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "servidor")
	if err == nil {
		querErro(t, svc.Desvincular(ctx, leitor, vinculo.ID), "soltar")
	}
	querErro(t, func() error {
		_, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "servidor")
		return err
	}(), "vínculo encerrado")

	querErro(t, func() error {
		_, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", ticket.ObjectKey)
		return err
	}(), "anexo encerrado")
	svc.arquivos = nil
	querErro(t, func() error { _, err := svc.URLAnexo(ctx, admin, p.ID, tarefa.ID, uuid.New()); return err }(), "sem arquivo")

	if _, err := svc.nomeNoProjeto(ctx, pool, &domain.Projeto{ID: p.ID, GerenteID: &outro, GerenteNome: "  "}, outro); err != nil && !errors.Is(err, domain.ErrForaDoProjeto) && !errors.Is(err, domain.ErrPessoaInvalida) {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM projetos_participantes WHERE projeto_id = $1 AND user_id = $2`, p.ID, outro); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListarMembros(ctx, admin, p.ID); err != nil {
		t.Fatal(err)
	}
	querErro(t, func() error { _, err := svc.ListarMembros(ctx, leitor, p.ID); return err }(), "membros")

	if _, err := pool.Exec(ctx, `
		UPDATE projetos SET gerente_id = NULL, data_fim_planejada = now() - interval '4 days' WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE projetos_tarefas SET responsavel_id = NULL, data_limite = now() - interval '4 days' WHERE projeto_id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM projetos_tarefa_pessoas WHERE tarefa_id IN (SELECT id FROM projetos_tarefas WHERE projeto_id = $1)`, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// avisoAte devolve erro a partir da chamada seguinte à indicada.
func avisoAte(t *testing.T, ate int) func() {
	t.Helper()
	anterior := gravarAviso
	vezes := 0
	gravarAviso = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		vezes++
		if vezes > ate {
			return nil, errors.New("aviso")
		}
		return nil, nil
	}
	return func() { gravarAviso = anterior }
}

func TestRamosQueFaltam(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	fora := dbtest.User(t, pool)
	for _, id := range []uuid.UUID{pessoa, outro, fora} {
		if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, id, unidade); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	leitor := auth.Identity{UserID: pessoa, Name: "Ana Souza"}
	real := infrastructure.NewRepository()
	logger := slog.New(slog.DiscardHandler)
	svc := NewService(pool, real, outbox.NewWriter("test"), logger)
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	aviso := gravarAviso
	t.Cleanup(func() { gravarAviso = aviso })
	entrada := domain.EntradaProjeto{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro}

	if _, err := svc.avisarGerenteSaiu(ctx, nil, pessoa, nil, &outro); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.comReabertura(ctx, pool, admin, nil); err != nil {
		t.Fatal(err)
	}
	svc.repo = falhaRepo{Repository: real, op: "nome"}
	if svc.nomeDoAutor(ctx, pool, pessoa) != "" {
		t.Fatal("nome")
	}
	svc.repo = real

	restaurar := avisoAte(t, 0)
	querErro(t, func() error { _, err := svc.Criar(ctx, admin, entrada); return err }(), "aviso na criação")
	restaurar()

	p, err := svc.Criar(ctx, admin, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	svc.repo = falhaRepo{Repository: real, op: "participante"}
	querErro(t, func() error {
		_, err := svc.nomeNoProjeto(ctx, pool, &domain.Projeto{ID: p.ID, GerenteID: &outro, GerenteNome: "  "}, outro)
		return err
	}(), "participante")
	svc.repo = real
	querErro(t, func() error {
		_, err := svc.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Fonte", Descricao: "Escopo", Objetivo: "Publicar", GerenteNome: "Ninguém"})
		return err
	}(), "pessoa da carta")
	comProduto := entrada
	produto := uuid.New()
	comProduto.ProdutoID = &produto
	svc.ComPortas(nil, produtoQueFalha{})
	querErro(t, func() error { _, err := svc.Atualizar(ctx, admin, p.ID, comProduto); return err }(), "produto")
	svc.ComPortas(nil, nil)

	if _, err := svc.IncluirParticipante(ctx, admin, p.ID, fora); err != nil {
		t.Fatal(err)
	}
	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Única", "", &outro, "Outro", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doFora, err := svc.CriarTarefa(ctx, admin, p.ID, "Do fora", "", &fora, "Fora", domain.EstadoTarefaAFazer, 2, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	descartavel, err := svc.CriarTarefa(ctx, admin, p.ID, "Descartável", "", &outro, "Outro", domain.EstadoTarefaAFazer, 3, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoSolicitacao, uuid.New(), "chamado")
	if err != nil {
		t.Fatal(err)
	}
	apontamento, err := svc.RegistrarApontamento(ctx, admin, p.ID, nil, time.Now().Format("2006-01-02"), 1, "revisão", 10)
	if err != nil {
		t.Fatal(err)
	}
	querErro(t, svc.Desvincular(ctx, leitor, vinculo.ID), "soltar")
	querErro(t, svc.ExcluirApontamento(ctx, leitor, apontamento.ID), "horas do leitor")

	restaurar = avisoAte(t, 0)
	querErro(t, svc.ExcluirTarefa(ctx, admin, descartavel.ID, "Descartável"), "aviso ao apagar")
	restaurar()
	if err := svc.ExcluirTarefa(ctx, admin, descartavel.ID, "Descartável"); err != nil {
		t.Fatal(err)
	}

	restaurar = avisoAte(t, 1)
	antes := *p
	antes.GerenteID = &pessoa
	depois := *p
	depois.GerenteID = &outro
	querErro(t, database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.avisarTrocaDeGerente(ctx, tx, uuid.New(), &antes, &depois, "título", "mensagem")
		return err
	}), "novo gerente")
	restaurar()
	restaurar = avisoAte(t, 2)
	querErro(t, database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.avisarTrocaDeGerente(ctx, tx, uuid.New(), &antes, &depois, "título", "mensagem")
		return err
	}), "aviso da troca")
	restaurar()

	svc.repo = falhaRepo{Repository: real, op: "tarefas"}
	querErro(t, database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.avisarSucessoras(ctx, tx, pessoa, p, domain.Tarefa{ID: tarefa.ID, Estado: domain.EstadoTarefaConcluida, Titulo: "Única"})
		return err
	}), "sucessoras")
	svc.repo = real

	restaurar = avisoAte(t, 1)
	querErro(t, database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.avisarTarefa(ctx, tx, uuid.Nil, p,
			domain.Tarefa{ID: tarefa.ID, Titulo: "Única", ResponsavelID: &outro},
			domain.Tarefa{ID: tarefa.ID, Titulo: "Única", ResponsavelID: &fora})
		return err
	}), "saída da tarefa")
	restaurar()

	some := &tarefasQueSomem{Repository: real, pessoa: fora}
	svc.repo = some
	if _, err := svc.ExcluirParticipante(ctx, admin, p.ID, fora); err != nil {
		t.Fatal(err)
	}
	svc.repo = real

	silencio, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Silêncio", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, silencio.ID)
	})
	muda, err := svc.CriarTarefa(ctx, admin, silencio.ID, "Muda", "", nil, "", domain.EstadoTarefaAFazer, 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos SET gerente_id = NULL, data_fim_planejada = now() - interval '4 days' WHERE id = $1`, silencio.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos_tarefas SET responsavel_id = NULL, data_limite = now() - interval '4 days' WHERE id = $1`, muda.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM projetos_tarefa_pessoas WHERE tarefa_id = $1`, muda.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	svc.repo = falhaRepo{Repository: real, op: "tarefas"}
	querErro(t, func() error { _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); return err }(), "tarefas na conclusão")
	svc.repo = real
	for _, item := range []struct {
		id     uuid.UUID
		titulo string
	}{{tarefa.ID, "Única"}, {doFora.ID, "Do fora"}} {
		if _, err := svc.AtualizarTarefa(ctx, admin, item.id, item.titulo, "", "", &outro, "Outro", domain.EstadoTarefaConcluida, 1, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	restaurar = avisoAte(t, 0)
	querErro(t, func() error { _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); return err }(), "aviso da conclusão")
	restaurar()
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}

	svc.repo = falhaRepo{Repository: real, op: "transicao"}
	querErro(t, func() error { _, err := svc.Reabrir(ctx, admin, p.ID, "voltou o escopo"); return err }(), "transição da reabertura")
	svc.repo = real
	querErro(t, svc.Desvincular(ctx, admin, vinculo.ID), "soltar encerrado")
	querErro(t, svc.ExcluirApontamento(ctx, admin, apontamento.ID), "horas encerradas")
	querErro(t, func() error { _, err := svc.Cancelar(ctx, admin, p.ID, "saiu do plano"); return err }(), "cancelar concluído")
}
