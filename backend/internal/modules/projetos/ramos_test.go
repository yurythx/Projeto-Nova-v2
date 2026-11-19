package projetos

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

func promover(t *testing.T, pool *pgxpool.Pool, pessoa, unidade uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE users SET roles = ARRAY['nova-admin']::text[] WHERE id = $1`, pessoa); err != nil {
		if _, err = pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, pessoa); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, pessoa, unidade); err != nil {
		t.Fatal(err)
	}
}

func novoServico(pool *pgxpool.Pool, repo domain.Repository) *application.Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, repo, outbox.NewWriter("test"), logger)
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	return svc
}

func entradaDe(unidade uuid.UUID, titulo string) domain.EntradaProjeto {
	return domain.EntradaProjeto{Titulo: titulo, Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade}
}

func deve(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func espera(t *testing.T, err, alvo error) {
	t.Helper()
	if !errors.Is(err, alvo) {
		t.Fatalf("esperava %v, veio %v", alvo, err)
	}
}

type produtoFalha struct {
	unidade *uuid.UUID
	err     error
}

func (p produtoFalha) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, p.err
}

type portasFalha struct {
	off     bool
	err     error
	unidade *uuid.UUID
	rotulo  string
}

func (p portasFalha) Disponivel(string) bool { return !p.off }
func (p portasFalha) Consultar(context.Context, database.DBTX, string, uuid.UUID) (domain.Alvo, error) {
	if p.err != nil {
		return domain.Alvo{}, p.err
	}
	return domain.Alvo{Rotulo: p.rotulo, UnidadeID: p.unidade}, nil
}

type mapaFalha struct{}

func (mapaFalha) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	return nil, errors.New("mapa")
}

type horasFalha struct{ err error }

func (h horasFalha) Somar(context.Context, database.DBTX, []uuid.UUID) (float64, error) {
	if h.err != nil {
		return 0, h.err
	}
	return 3.5, nil
}

func leitorDe(id uuid.UUID) auth.Identity {
	return auth.Identity{UserID: id, Scopes: []auth.Scope{{
		Permissions: []string{string(auth.PermProjetosRead)},
	}}}
}

func TestHandlersSemCobertura(t *testing.T) {
	pool := dbtest.Pool(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, &cortaRepo{sempre: true}, nil, logger)
	h := transport.NewHandlers(svc, logger)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			idt := auth.Identity{UserID: uuid.New(), Name: "Ana", Roles: []string{auth.RoleAdmin}}
			next.ServeHTTP(w, req.WithContext(auth.WithIdentity(req.Context(), idt)))
		})
	})
	h.RegisterRoutes(r)

	id := uuid.New().String()
	tarefa := uuid.New().String()
	chama := func(method, target, body string) {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, target, reader)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code < 400 || !strings.Contains(rec.Body.String(), "error") {
			t.Fatalf("%s %s → %d %s", method, target, rec.Code, rec.Body.String())
		}
	}
	jsonTarefa := `{"titulo":"Levantar"}`
	jsonVinculo := `{"tipo":"ativo","alvo_id":"` + id + `"}`
	jsonHoras := `{"data_apontamento":"2026-11-18","horas":1,"descricao":"Levantamento"}`
	for _, c := range []struct{ m, p, b string }{
		{http.MethodGet, "/projetos?produto_id=nao", ""},
		{http.MethodGet, "/projetos?alvo_id=nao", ""},
		{http.MethodGet, "/projetos?produto_id=" + id, ""},
		{http.MethodGet, "/projetos/minhas", ""},
		{http.MethodGet, "/projetos/carga", ""},
		{http.MethodGet, "/projetos/prazos?mes=13", ""},
		{http.MethodGet, "/projetos/prazos?mes=2026-11", ""},
		{http.MethodGet, "/projetos/nao", ""},
		{http.MethodGet, "/projetos/" + id, ""},
		{http.MethodGet, "/projetos/nao/historico", ""},
		{http.MethodGet, "/projetos/" + id + "/historico", ""},
		{http.MethodGet, "/projetos/nao/tarefas", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/nao/anexos", ""},
		{http.MethodGet, "/projetos/nao/tarefas/" + tarefa + "/anexos", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/nao/movimentos", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/" + tarefa + "/movimentos", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/nao/atualizacoes", ""},
		{http.MethodGet, "/projetos/" + id + "/tarefas/" + tarefa + "/atualizacoes", ""},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/atualizacoes", "{"},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/atualizacoes", `{"texto":"nota"}`},
		{http.MethodGet, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos/nao", ""},
		{http.MethodGet, "/projetos/" + id + "/vinculos", ""},
		{http.MethodGet, "/projetos/nao/apontamentos", ""},
		{http.MethodGet, "/projetos/" + id + "/apontamentos", ""},
		{http.MethodGet, "/projetos/nao/carta", ""},
		{http.MethodGet, "/projetos/" + id + "/carta", ""},
		{http.MethodGet, "/projetos/nao/participantes", ""},
		{http.MethodGet, "/projetos/" + id + "/participantes", ""},
		{http.MethodPost, "/projetos", "{"},
		{http.MethodPost, "/projetos", `{"titulo":"Corte"}`},
		{http.MethodPut, "/projetos/nao", `{"titulo":"Corte"}`},
		{http.MethodPut, "/projetos/" + id, "{"},
		{http.MethodPut, "/projetos/" + id, `{"titulo":"Corte"}`},
		{http.MethodPost, "/projetos/nao/estado", `{"estado":"em_andamento"}`},
		{http.MethodPost, "/projetos/" + id + "/estado", "{"},
		{http.MethodPost, "/projetos/" + id + "/estado", `{"estado":"em_andamento"}`},
		{http.MethodPost, "/projetos/nao/reabertura", `{"motivo":"porque"}`},
		{http.MethodPost, "/projetos/" + id + "/reabertura", "{"},
		{http.MethodPost, "/projetos/" + id + "/reabertura", `{"motivo":"porque"}`},
		{http.MethodPost, "/projetos/nao/cancelamento", `{"motivo":"porque"}`},
		{http.MethodPost, "/projetos/" + id + "/cancelamento", "{"},
		{http.MethodPost, "/projetos/" + id + "/cancelamento", `{"motivo":"porque"}`},
		{http.MethodPost, "/projetos/nao/tarefas", jsonTarefa},
		{http.MethodPost, "/projetos/" + id + "/tarefas", "{"},
		{http.MethodPost, "/projetos/" + id + "/tarefas", jsonTarefa},
		{http.MethodPut, "/projetos/" + id + "/tarefas/nao", jsonTarefa},
		{http.MethodPut, "/projetos/" + id + "/tarefas/" + tarefa, "{"},
		{http.MethodPut, "/projetos/" + id + "/tarefas/" + tarefa, jsonTarefa},
		{http.MethodGet, "/projetos/nao/responsaveis", ""},
		{http.MethodGet, "/projetos/" + id + "/responsaveis", ""},
		{http.MethodPost, "/projetos/nao/participantes", `{"id":"` + id + `"}`},
		{http.MethodPost, "/projetos/" + id + "/participantes", "{"},
		{http.MethodPost, "/projetos/" + id + "/participantes", `{"id":"` + id + `"}`},
		{http.MethodDelete, "/projetos/" + id + "/participantes/nao", ""},
		{http.MethodDelete, "/projetos/nao/participantes/" + id, ""},
		{http.MethodDelete, "/projetos/" + id + "/participantes/" + id, ""},
		{http.MethodDelete, "/projetos/" + id + "/tarefas/nao", ""},
		{http.MethodDelete, "/projetos/" + id + "/tarefas/" + tarefa + "?confirmar=outro", ""},
		{http.MethodPost, "/projetos/" + id + "/tarefas/nao/anexos/uploads", `{"nome":"a.pdf"}`},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos/uploads", "{"},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos/uploads", `{"nome":"a.pdf","content_type":"application/pdf"}`},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos", "{"},
		{http.MethodPost, "/projetos/" + id + "/tarefas/" + tarefa + "/anexos", `{"nome":"a.pdf","object_key":"k"}`},
		{http.MethodPost, "/projetos/nao/vinculos", jsonVinculo},
		{http.MethodPost, "/projetos/" + id + "/vinculos", "{"},
		{http.MethodPost, "/projetos/" + id + "/vinculos", jsonVinculo},
		{http.MethodDelete, "/projetos/" + id + "/vinculos/nao", ""},
		{http.MethodDelete, "/projetos/" + id + "/vinculos/" + id, ""},
		{http.MethodPost, "/projetos/nao/apontamentos", jsonHoras},
		{http.MethodPost, "/projetos/" + id + "/apontamentos", "{"},
		{http.MethodPost, "/projetos/" + id + "/apontamentos", jsonHoras},
		{http.MethodDelete, "/projetos/" + id + "/apontamentos/nao", ""},
		{http.MethodDelete, "/projetos/" + id + "/apontamentos/" + id, ""},
		{http.MethodPut, "/projetos/nao/carta", `{"titulo":"Corte"}`},
		{http.MethodPut, "/projetos/" + id + "/carta", "{"},
		{http.MethodPut, "/projetos/" + id + "/carta", `{"titulo":"Corte"}`},
	} {
		chama(c.m, c.p, c.b)
	}
}

func TestRamosQueFaltam(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	tecnico := dbtest.User(t, pool)
	promover(t, pool, pessoa, unidade)
	promover(t, pool, outro, unidade)
	promover(t, pool, tecnico, unidade)
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	svc := novoServico(pool, infrastructure.NewRepository())
	var criados []uuid.UUID
	t.Cleanup(func() {
		for _, id := range criados {
			_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, id)
		}
	})
	abrir := func(titulo string) *domain.Projeto {
		t.Helper()
		p, err := svc.Criar(ctx, admin, entradaDe(unidade, titulo))
		deve(t, err)
		criados = append(criados, p.ID)
		return p
	}

	leitor := leitorDe(uuid.New())
	_, err := svc.Criar(ctx, leitor, entradaDe(unidade, "Sem permissão"))
	espera(t, err, domain.ErrForaDoEscopo)
	_, err = svc.Criar(ctx, admin, domain.EntradaProjeto{})
	espera(t, err, domain.ErrTituloObrigatorio)
	_, err = svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Ruim", Prioridade: "ruim"})
	espera(t, err, domain.ErrPrioridadeInvalida)

	inicio := time.Now().UTC()
	fim := inicio.Add(-time.Hour)
	_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Plano", Orcamento: -1}, domain.EntradaCarta{}, nil)
	espera(t, err, domain.ErrOrcamentoInvalido)
	_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Plano", DataInicioPlanejada: &inicio, DataFimPlanejada: &fim}, domain.EntradaCarta{}, nil)
	espera(t, err, domain.ErrPeriodoInvalido)
	_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Plano"}, domain.EntradaCarta{Onde: strings.Repeat("a", domain.CartaOndeMax+1)}, nil)
	espera(t, err, domain.ErrCartaInvalida)
	_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Plano"}, domain.EntradaCarta{}, []string{strings.Repeat("a", domain.EtapaMax+1)})
	espera(t, err, domain.ErrEtapaInvalida)
	muitas := make([]string, domain.EtapasMax+1)
	for i := range muitas {
		muitas[i] = "etapa"
	}
	_, _, err = domain.ValidarAbertura(domain.EntradaProjeto{Titulo: "Plano"}, domain.EntradaCarta{}, muitas)
	espera(t, err, domain.ErrEtapaInvalida)
	_, err = svc.CriarGuiado(ctx, admin, domain.EntradaProjeto{Titulo: "Plano", Orcamento: -1}, domain.EntradaCarta{}, []string{"  ", "Mapear"})
	espera(t, err, domain.ErrOrcamentoInvalido)

	produto := uuid.New()
	outra := uuid.New()
	comProduto := entradaDe(unidade, "Com produto")
	comProduto.ProdutoID = &produto
	svc.ComPortas(nil, produtoFalha{err: errors.New("produto")})
	_, err = svc.Criar(ctx, admin, comProduto)
	if err == nil {
		t.Fatal("produto")
	}
	svc.ComPortas(nil, produtoFalha{unidade: &outra})
	_, err = svc.Criar(ctx, admin, comProduto)
	espera(t, err, domain.ErrDeOutraUnidade)
	svc.ComPortas(nil, produtoFalha{})
	criado, err := svc.Criar(ctx, admin, comProduto)
	deve(t, err)
	criados = append(criados, criado.ID)
	svc.ComPortas(nil, nil)

	p := abrir("Portal dos ramos")
	_, err = svc.Atualizar(ctx, admin, p.ID, domain.EntradaProjeto{
		Titulo: "Portal dos ramos", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa,
	})
	deve(t, err)
	_, err = svc.Obter(ctx, admin, uuid.New())
	espera(t, err, domain.ErrProjetoNaoEncontrado)
	_, err = svc.Atualizar(ctx, admin, uuid.New(), entradaDe(unidade, "Portal dos ramos"))
	espera(t, err, domain.ErrProjetoNaoEncontrado)
	_, err = svc.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Portal dos ramos", Orcamento: -1})
	espera(t, err, domain.ErrOrcamentoInvalido)
	_, err = svc.Transicionar(ctx, admin, p.ID, "inexistente")
	espera(t, err, domain.ErrTransicaoInvalida)
	_, err = svc.Transicionar(ctx, admin, p.ID, domain.EstadoCancelado)
	espera(t, err, domain.ErrCancelamentoNaoAutorizado)
	_, err = svc.Transicionar(ctx, leitor, p.ID, domain.EstadoEmAndamento)
	espera(t, err, domain.ErrForaDoEscopo)
	_, err = svc.Reabrir(ctx, admin, p.ID, "ainda não")
	espera(t, err, domain.ErrReaberturaExigeAprovacao)
	_, err = svc.Reabrir(ctx, admin, p.ID, " ")
	espera(t, err, domain.ErrMotivoObrigatorio)

	svc.ComPassagens(mapaFalha{})
	_, err = svc.Listar(ctx, admin, "", "")
	if err == nil {
		t.Fatal("mapa")
	}
	_, err = svc.Obter(ctx, admin, p.ID)
	if err == nil {
		t.Fatal("mapa obter")
	}
	svc.ComPassagens(nil)
	_, err = svc.ListarMinhas(ctx, auth.Identity{})
	deve(t, err)
	_, err = svc.ListarPrazos(ctx, admin, "13")
	espera(t, err, domain.ErrMesInvalido)
	_, err = svc.ListarPrazos(ctx, leitor, time.Now().Format("2006-01"))
	deve(t, err)
	_, err = svc.ListarPorFiltro(ctx, leitor, domain.FiltroProjetos{Estado: "planejado", Busca: "Portal", AlvoID: &produto})
	deve(t, err)

	ontem := time.Now().Add(-48 * time.Hour).UTC()
	amanha := time.Now().Add(48 * time.Hour).UTC()
	grupo := []domain.Pessoa{{ID: pessoa, Nome: "Ana"}}
	tarefa, err := svc.CriarTarefa(ctx, admin, p.ID, "Levantar filas", "", &pessoa, "Ana", "", 1, &ontem, nil, nil, &grupo)
	deve(t, err)
	demais := make([]domain.Pessoa, domain.PessoasPorTarefaMax+1)
	for i := range demais {
		demais[i] = domain.Pessoa{ID: pessoa, Nome: "Ana"}
	}
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Levantar filas", "", "", nil, "", "a_fazer", 1, nil, nil, &demais)
	espera(t, err, domain.ErrPessoasDemais)
	ruim := []domain.Pessoa{{Nome: "sem conta"}}
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Levantar filas", "", "", nil, "", "a_fazer", 1, nil, nil, &ruim)
	espera(t, err, domain.ErrPessoaInvalida)
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Levantar filas", "", strings.Repeat("a", domain.RelatoMax+1), nil, "", "a_fazer", 1, nil, nil, nil)
	espera(t, err, domain.ErrRelatoInvalido)
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Levantar filas", "", "", nil, "", "coluna", 1, nil, nil, nil)
	espera(t, err, domain.ErrEstadoInvalido)
	nulo := uuid.Nil
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Filas levantadas", "detalhe", "feito", &pessoa, "Ana", domain.EstadoTarefaEmAndamento, 2, &amanha, &nulo, &grupo)
	deve(t, err)

	_, err = svc.IncluirParticipante(ctx, admin, p.ID, uuid.Nil)
	espera(t, err, domain.ErrPessoaInvalida)
	_, err = svc.IncluirParticipante(ctx, admin, p.ID, tecnico)
	deve(t, err)
	_, err = svc.IncluirParticipante(ctx, admin, p.ID, outro)
	deve(t, err)
	_, err = svc.ExcluirParticipante(ctx, admin, p.ID, pessoa)
	espera(t, err, domain.ErrGerentePermanece)
	_, err = svc.ExcluirParticipante(ctx, admin, p.ID, uuid.New())
	espera(t, err, domain.ErrParticipanteAusente)

	tecnicoID := tecnico
	grupoTecnico := []domain.Pessoa{{ID: tecnico, Nome: "Técnico"}}
	_, err = svc.CriarAtualizacao(ctx, leitor, p.ID, tarefa.ID, "não posso")
	espera(t, err, domain.ErrAtualizacaoNaoAutorizada)
	_, err = svc.CriarAtualizacao(ctx, admin, p.ID, tarefa.ID, "   ")
	espera(t, err, domain.ErrAtualizacaoInvalida)
	_, err = svc.ListarAtualizacoes(ctx, leitor, p.ID, tarefa.ID)
	deve(t, err)
	_, err = svc.ListarMovimentos(ctx, leitor, p.ID, tarefa.ID)
	deve(t, err)

	filha, err := svc.CriarTarefa(ctx, admin, p.ID, "Detalhar", "", &tecnicoID, "", domain.EstadoTarefaAFazer, 2, nil, nil, &tarefa.ID, &grupoTecnico)
	deve(t, err)
	_, err = svc.CriarAtualizacao(ctx, leitorDe(tecnico), p.ID, filha.ID, "sigo daqui")
	deve(t, err)
	_, err = svc.CriarTarefa(ctx, admin, p.ID, "Neta", "", nil, "", "", 3, nil, nil, &filha.ID, nil)
	espera(t, err, domain.ErrSubtarefaInvalida)
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Filas levantadas", "", "", &pessoa, "Ana", domain.EstadoTarefaConcluida, 2, nil, nil, &grupo)
	espera(t, err, domain.ErrSubtarefasAbertas)
	_, err = svc.CriarTarefa(ctx, admin, p.ID, "Depois da filha", "", &pessoa, "Ana", domain.EstadoTarefaAFazer, 4, nil, &tarefa.ID, &tarefa.ID, &grupo)
	espera(t, err, domain.ErrPredecessoraInvalida)
	seguinte, err := svc.CriarTarefa(ctx, admin, p.ID, "Publicar", "", &tecnicoID, "", domain.EstadoTarefaAFazer, 5, &ontem, &tarefa.ID, nil, &grupoTecnico)
	deve(t, err)
	_, err = svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Publicar", "", "", &tecnicoID, "", domain.EstadoTarefaConcluida, 5, nil, &tarefa.ID, &grupoTecnico)
	espera(t, err, domain.ErrPredecessoraAberta)

	semNome := admin
	semNome.Name = ""
	semNome.Username = "ana"
	_, err = svc.AtualizarTarefa(ctx, semNome, filha.ID, "Detalhar", "", "nota", &tecnicoID, "", domain.EstadoTarefaConcluida, 2, nil, nil, &grupoTecnico)
	deve(t, err)
	_, err = svc.AtualizarTarefa(ctx, admin, tarefa.ID, "Filas levantadas", "", "encerrada", &pessoa, "Ana", domain.EstadoTarefaConcluida, 2, nil, nil, &grupo)
	deve(t, err)
	_, err = svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Publicar", "agora", "", &outro, "", domain.EstadoTarefaBloqueada, 5, &ontem, &tarefa.ID, &[]domain.Pessoa{{ID: outro}})
	deve(t, err)
	_, err = svc.AtualizarTarefa(ctx, admin, seguinte.ID, "Publicado", "", "no ar", &tecnicoID, "", domain.EstadoTarefaConcluida, 6, &amanha, &tarefa.ID, &grupoTecnico)
	deve(t, err)

	_, err = svc.Vincular(ctx, admin, p.ID, "agenda", uuid.New(), "")
	espera(t, err, domain.ErrVinculoInvalido)
	svc.ComPortas(portasFalha{off: true}, nil)
	_, err = svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "")
	espera(t, err, domain.ErrIntegracaoDesligada)
	svc.ComPortas(portasFalha{err: errors.New("porta")}, nil)
	_, err = svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "")
	if err == nil {
		t.Fatal("porta")
	}
	svc.ComPortas(portasFalha{unidade: &outra, rotulo: "Servidor"}, nil)
	_, err = svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "")
	espera(t, err, domain.ErrDeOutraUnidade)
	svc.ComPortas(portasFalha{rotulo: "Servidor"}, nil)
	ligado, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "")
	deve(t, err)
	svc.ComPortas(nil, nil)
	deve(t, svc.Desvincular(ctx, admin, ligado.ID))
	deve(t, svc.Desvincular(ctx, admin, uuid.New()))
	for _, tipo := range []string{domain.VinculoProblema, domain.VinculoMudanca, domain.VinculoSolicitacao} {
		_, err = svc.Vincular(ctx, admin, p.ID, tipo, uuid.New(), tipo)
		deve(t, err)
	}
	svc.ComHoras(horasFalha{err: errors.New("horas")})
	_, err = svc.ObterCarta(ctx, admin, p.ID)
	if err == nil {
		t.Fatal("horas")
	}
	svc.ComHoras(horasFalha{})
	_, err = svc.ObterCarta(ctx, admin, p.ID)
	deve(t, err)
	svc.ComHoras(nil)

	err = svc.ExcluirTarefa(ctx, admin, seguinte.ID, "outro")
	espera(t, err, domain.ErrExclusaoNaoConfirmada)
	err = svc.ExcluirTarefa(ctx, leitor, seguinte.ID, "Publicado")
	espera(t, err, domain.ErrForaDoEscopo)

	loja := storagetest.New()
	svc.ComArquivos(application.Arquivos{Store: loja, Bucket: "nova", MaxBytes: 10 << 20, Expiry: time.Minute})
	_, err = svc.PedirAnexo(ctx, leitor, p.ID, tarefa.ID, "planta.pdf", "application/pdf")
	espera(t, err, domain.ErrForaDoEscopo)
	ticket, err := svc.PedirAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", "application/pdf")
	deve(t, err)
	deve(t, loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"))
	anexo, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "planta.pdf", ticket.ObjectKey)
	deve(t, err)
	if _, err = svc.URLAnexo(ctx, admin, p.ID, tarefa.ID, anexo.ID); err != nil {
		t.Fatal(err)
	}
	loja.FailPresign = true
	if _, err = svc.URLAnexo(ctx, admin, p.ID, tarefa.ID, anexo.ID); err == nil {
		t.Fatal("presign")
	}
	loja.FailPresign = false
	_, err = svc.RegistrarAnexo(ctx, admin, p.ID, tarefa.ID, "", ticket.ObjectKey)
	if err == nil {
		t.Fatal("nome")
	}

	atualizado, err := svc.Atualizar(ctx, admin, p.ID, domain.EntradaProjeto{
		Titulo: "Portal dos ramos", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &outro,
	})
	deve(t, err)
	_, err = svc.CriarAtualizacao(ctx, leitorDe(outro), p.ID, tarefa.ID, "o gerente anota")
	deve(t, err)
	_, err = svc.Atualizar(ctx, admin, p.ID, domain.EntradaProjeto{
		Titulo: "Portal dos ramos", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteNome: "ninguém",
	})
	espera(t, err, domain.ErrPessoaInvalida)
	_, err = svc.Atualizar(ctx, admin, p.ID, domain.EntradaProjeto{
		Titulo: "Portal dos ramos", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade,
	})
	deve(t, err)
	_ = atualizado
	_, err = svc.Atualizar(ctx, admin, p.ID, domain.EntradaProjeto{
		Titulo: "Portal dos ramos", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa,
	})
	deve(t, err)

	deve(t, svc.VerificarPrazos(ctx, time.Now().UTC()))
	ctxPronto, cancela := context.WithCancel(ctx)
	cancela()
	deve(t, svc.RunPrazos(ctxPronto))

	restantes, err := svc.ListarTarefas(ctx, admin, p.ID)
	deve(t, err)
	for _, tk := range restantes {
		if tk.Estado == domain.EstadoTarefaConcluida || tk.PaiID != nil {
			continue
		}
		quem := tk.ResponsavelID
		nome := tk.ResponsavelNome
		if quem == nil {
			quem = &pessoa
			nome = "Ana"
		}
		_, err = svc.AtualizarTarefa(ctx, admin, tk.ID, tk.Titulo, tk.Descricao, tk.Relato, quem, nome, domain.EstadoTarefaConcluida, tk.Ordem, tk.DataLimite, tk.PredecessoraID, nil)
		deve(t, err)
	}
	restantes, err = svc.ListarTarefas(ctx, admin, p.ID)
	deve(t, err)
	for _, tk := range restantes {
		if tk.Estado == domain.EstadoTarefaConcluida {
			continue
		}
		quem := tk.ResponsavelID
		if quem == nil {
			quem = &pessoa
		}
		_, err = svc.AtualizarTarefa(ctx, admin, tk.ID, tk.Titulo, tk.Descricao, tk.Relato, quem, tk.ResponsavelNome, domain.EstadoTarefaConcluida, tk.Ordem, tk.DataLimite, tk.PredecessoraID, nil)
		deve(t, err)
	}
	_, err = svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento)
	deve(t, err)
	_, err = svc.Transicionar(ctx, admin, p.ID, domain.EstadoSuspenso)
	deve(t, err)
	_, err = svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento)
	deve(t, err)
	_, err = svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido)
	deve(t, err)
	_, err = svc.Reabrir(ctx, leitor, p.ID, "quero reabrir")
	espera(t, err, domain.ErrReaberturaNaoAutorizada)
	soAssunto := admin
	soAssunto.UserID = uuid.Nil
	soAssunto.Name = ""
	soAssunto.Subject = pessoa.String()
	_, err = svc.Reabrir(ctx, soAssunto, p.ID, "o superior pediu")
	deve(t, err)
	_, err = svc.ListarHistorico(ctx, admin, p.ID)
	deve(t, err)

	menor := abrir("Um vínculo")
	_, err = svc.Atualizar(ctx, admin, menor.ID, domain.EntradaProjeto{
		Titulo: "Um vínculo", Descricao: "Escopo", Objetivo: "Publicar", UnidadeID: &unidade, GerenteID: &pessoa,
	})
	deve(t, err)
	unica, err := svc.CriarTarefa(ctx, admin, menor.ID, "Única", "", &pessoa, "Ana", domain.EstadoTarefaConcluida, 1, nil, nil, nil, &grupo)
	deve(t, err)
	_ = unica
	_, err = svc.Vincular(ctx, admin, menor.ID, domain.VinculoSolicitacao, uuid.New(), "chamado")
	deve(t, err)
	_, err = svc.Transicionar(ctx, admin, menor.ID, domain.EstadoEmAndamento)
	deve(t, err)
	_, err = svc.Transicionar(ctx, admin, menor.ID, domain.EstadoConcluido)
	deve(t, err)

	for _, idt := range []auth.Identity{
		{UserID: pessoa, Username: "ana", Roles: []string{auth.RoleAdmin}},
		{UserID: pessoa, Email: "ana@example.com", Roles: []string{auth.RoleAdmin}},
		{Name: "", Subject: pessoa.String(), Roles: []string{auth.RoleAdmin}},
	} {
		avulso := abrir("Para cancelar")
		_, err = svc.Cancelar(ctx, idt, avulso.ID, "saiu do plano")
		deve(t, err)
	}
	_, err = svc.Cancelar(ctx, admin, p.ID, " ")
	espera(t, err, domain.ErrMotivoObrigatorio)
	_, err = svc.Obter(ctx, admin, criados[len(criados)-1])
	deve(t, err)

	deve(t, svc.ExcluirApontamento(ctx, admin, uuid.New()))
	horas, err := svc.RegistrarApontamento(ctx, auth.Identity{UserID: pessoa, Email: "ana@example.com", Roles: []string{auth.RoleAdmin}}, p.ID, &tarefa.ID, time.Now().Format("2006-01-02"), 1, "revisão", 10)
	if err != nil && p.Estado != "" {
		// o projeto já foi reaberto; o lançamento tem de passar
		t.Fatal(err)
	}
	if horas != nil {
		deve(t, svc.ExcluirApontamento(ctx, admin, horas.ID))
	}
	_, err = svc.ExcluirParticipante(ctx, admin, p.ID, tecnico)
	deve(t, err)
}

func TestCortesDosRetornos(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	outro := dbtest.User(t, pool)
	promover(t, pool, pessoa, unidade)
	promover(t, pool, outro, unidade)
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}
	repo := &cortaRepo{Repository: infrastructure.NewRepository()}
	svc := novoServico(pool, repo)
	var criados []uuid.UUID
	t.Cleanup(func() {
		for _, id := range criados {
			_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, id)
		}
	})
	guardar := func(p *domain.Projeto, err error) error {
		if err == nil && p != nil {
			criados = append(criados, p.ID)
		}
		return err
	}
	comGerente := entradaDe(unidade, "Corte da catraca")
	comGerente.GerenteID = &pessoa
	repo.cortar(t, func() error {
		p, err := svc.Criar(ctx, admin, comGerente)
		return guardar(p, err)
	})
	p, err := svc.Criar(ctx, admin, comGerente)
	deve(t, err)
	criados = append(criados, p.ID)

	repo.cortar(t, func() error { _, err := svc.Obter(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, err := svc.Listar(ctx, admin, "planejado", "Corte"); return err })
	repo.cortar(t, func() error { _, err := svc.ListarMinhas(ctx, admin); return err })
	repo.cortar(t, func() error { _, err := svc.ListarCarga(ctx, admin); return err })
	repo.cortar(t, func() error { _, err := svc.ListarHistorico(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarTarefas(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarMembros(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarResponsaveis(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarVinculos(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error { _, _, err := svc.ListarApontamentos(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error {
		_, err := svc.SalvarCarta(ctx, admin, p.ID, domain.Plano5W2H{Titulo: "Corte da catraca", Descricao: "Escopo", Objetivo: "Publicar", Onde: "SEMA"})
		return err
	})
	repo.cortar(t, func() error { _, err := svc.ObterCarta(ctx, admin, p.ID); return err })
	repo.cortar(t, func() error {
		_, err := svc.Atualizar(ctx, admin, p.ID, comGerente)
		return err
	})
	var tarefaID uuid.UUID
	grupo := []domain.Pessoa{{ID: pessoa, Nome: "Ana"}}
	repo.cortar(t, func() error {
		tk, err := svc.CriarTarefa(ctx, admin, p.ID, "Medir", "", &pessoa, "Ana", "a_fazer", 1, nil, nil, nil, &grupo)
		if err == nil {
			tarefaID = tk.ID
		}
		return err
	})
	repo.cortar(t, func() error {
		_, err := svc.AtualizarTarefa(ctx, admin, tarefaID, "Medir de novo", "d", "r", &pessoa, "Ana", domain.EstadoTarefaEmAndamento, 1, nil, nil, &grupo)
		return err
	})
	repo.cortar(t, func() error {
		_, err := svc.CriarAtualizacao(ctx, admin, p.ID, tarefaID, "andamento")
		return err
	})
	repo.cortar(t, func() error { _, err := svc.ListarAtualizacoes(ctx, admin, p.ID, tarefaID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarMovimentos(ctx, admin, p.ID, tarefaID); return err })
	repo.cortar(t, func() error { _, err := svc.ListarAnexos(ctx, admin, p.ID, tarefaID); return err })
	var vinculoID uuid.UUID
	repo.cortar(t, func() error {
		v, err := svc.Vincular(ctx, admin, p.ID, domain.VinculoAtivo, uuid.New(), "Firewall")
		if err == nil {
			vinculoID = v.ID
		}
		return err
	})
	repo.cortar(t, func() error { return svc.Desvincular(ctx, admin, vinculoID) })
	var apontamentoID uuid.UUID
	repo.cortar(t, func() error {
		a, err := svc.RegistrarApontamento(ctx, admin, p.ID, &tarefaID, time.Now().Format("2006-01-02"), 1, "medida", 20)
		if err == nil {
			apontamentoID = a.ID
		}
		return err
	})
	repo.cortar(t, func() error { return svc.ExcluirApontamento(ctx, admin, apontamentoID) })
	repo.cortar(t, func() error { _, err := svc.IncluirParticipante(ctx, admin, p.ID, outro); return err })
	repo.cortar(t, func() error { _, err := svc.ExcluirParticipante(ctx, admin, p.ID, outro); return err })
	repo.cortar(t, func() error {
		_, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento)
		return err
	})
	repo.cortar(t, func() error {
		_, err := svc.ListarPrazos(ctx, admin, time.Now().Format("2006-01"))
		return err
	})
	repo.cortar(t, func() error { return svc.VerificarPrazos(ctx, time.Now().UTC()) })
	repo.cortar(t, func() error {
		p, err := svc.CriarGuiado(ctx, admin, entradaDe(unidade, "Corte guiado"), domain.EntradaCarta{Onde: "SEMA"}, []string{"Mapear", " "})
		return guardar(p, err)
	})

	loja := storagetest.New()
	svc.ComArquivos(application.Arquivos{Store: loja, Bucket: "nova", MaxBytes: 10 << 20, Expiry: time.Minute})
	var chave string
	repo.cortar(t, func() error {
		ticket, err := svc.PedirAnexo(ctx, admin, p.ID, tarefaID, "nota.pdf", "application/pdf")
		if err == nil {
			chave = ticket.ObjectKey
		}
		return err
	})
	deve(t, loja.Put(ctx, "nova", chave, strings.NewReader("%PDF"), 4, "application/pdf"))
	var anexoID uuid.UUID
	repo.cortar(t, func() error {
		a, err := svc.RegistrarAnexo(ctx, admin, p.ID, tarefaID, "nota.pdf", chave)
		if err == nil {
			anexoID = a.ID
		}
		return err
	})
	repo.cortar(t, func() error { _, err := svc.URLAnexo(ctx, admin, p.ID, tarefaID, anexoID); return err })

	curto := &cortaRepo{Repository: infrastructure.NewRepository(), na: 1}
	aviso := novoServico(pool, curto)
	ctxCurto, cancela := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancela()
	}()
	deve(t, aviso.RunPrazos(ctxCurto))

	repo.na = 0
	err = svc.ExcluirTarefa(ctx, admin, tarefaID, "Medir de novo")
	deve(t, err)
}
