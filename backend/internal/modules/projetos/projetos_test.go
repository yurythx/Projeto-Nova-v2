package projetos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type portasFalsas struct {
	disponivel bool
	alvo       domain.Alvo
	err        error
}

func (p portasFalsas) Disponivel(string) bool { return p.disponivel }
func (p portasFalsas) Consultar(context.Context, database.DBTX, string, uuid.UUID) (domain.Alvo, error) {
	return p.alvo, p.err
}

type produtosFalsos struct {
	unidade *uuid.UUID
	err     error
}

func (p produtosFalsos) Consultar(context.Context, database.DBTX, uuid.UUID) (*uuid.UUID, error) {
	return p.unidade, p.err
}

type passagensFalsas struct {
	mapa map[string][]string
	err  error
}

func (p passagensFalsas) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	return p.mapa, p.err
}

func TestModuloCicloCompleto(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	uA := dbtest.Unidade(t, pool)
	uB := dbtest.Unidade(t, pool)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mod := New(modkit.Deps{Pool: pool, Outbox: outbox.NewWriter("test"), Logger: logger})
	if mod.Manifest().Key != Key || mod.Service() == nil || len(mod.Workers()) != 1 {
		t.Fatal("manifesto, serviço ou worker inválido")
	}
	mod.ComPassagens(passagensFalsas{mapa: map[string][]string{}})
	alvoID := uuid.New()
	prodID := uuid.New()
	mod.ComPortas(
		portasFalsas{disponivel: true, alvo: domain.Alvo{Rotulo: "Ativo · Firewall SECITI", UnidadeID: &uA}},
		produtosFalsos{unidade: &uA},
	)

	r := chi.NewRouter()
	var currentIdentity auth.Identity
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(auth.WithIdentity(req.Context(), currentIdentity)))
		})
	})
	mod.RegisterRoutes(kernel.Routes{Authed: r})

	gestorA := auth.Identity{
		UserID:      uuid.New(),
		Subject:     uuid.NewString(),
		Email:       "gestor@seciti.test",
		Name:        "Gestor SECITI",
		Permissions: []string{string(auth.PermProjetosRead), string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID:   &uA,
			Unidades:    []uuid.UUID{uA},
			Permissions: []string{string(auth.PermProjetosRead), string(auth.PermProjetosManage)},
		}},
	}
	leitorB := auth.Identity{
		UserID:      uuid.New(),
		Subject:     uuid.NewString(),
		Email:       "leitor@sms.test",
		Permissions: []string{string(auth.PermProjetosRead)},
		Scopes: []auth.Scope{{
			UnidadeID:   &uB,
			Unidades:    []uuid.UUID{uB},
			Permissions: []string{string(auth.PermProjetosRead)},
		}},
	}
	gestorB := auth.Identity{
		UserID:      uuid.New(),
		Subject:     uuid.NewString(),
		Email:       "gestor@sms.test",
		Permissions: []string{string(auth.PermProjetosRead), string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID:   &uB,
			Unidades:    []uuid.UUID{uB},
			Permissions: []string{string(auth.PermProjetosRead), string(auth.PermProjetosManage)},
		}},
	}

	do := func(idt auth.Identity, method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		currentIdentity = idt
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
		return rec
	}

	// Gestor de A não cria projeto na unidade B.
	if rec := do(gestorA, http.MethodPost, "/projetos", `{"titulo":"Projeto B","unidade_id":"`+uB.String()+`"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 fora do escopo, obteve %d: %s", rec.Code, rec.Body.String())
	}

	// Cria projeto na unidade A (sem produto para não bater na FK de produtos, testamos a porta via service).
	rec := do(gestorA, http.MethodPost, "/projetos", `{"titulo":"Implantação ITIL v5","descricao":"Escopo","objetivo":"Padronizar","prioridade":"alta","orcamento":50000,"unidade_id":"`+uA.String()+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar projeto: %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data domain.Projeto `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	projID := env.Data.ID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, projID)
	})

	// Leitor de B não enxerga projeto de A.
	if rec := do(leitorB, http.MethodGet, "/projetos/"+projID.String(), ""); rec.Code != http.StatusForbidden {
		t.Fatalf("leitor de B deveria receber 403, obteve %d", rec.Code)
	}

	// Atualiza projeto.
	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String(),
		`{"titulo":"Implantação ITIL v5 - Fase 2","descricao":"Novo","objetivo":"Meta","prioridade":"critica","orcamento":60000,"unidade_id":"`+uA.String()+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("atualizar: %d %s", rec.Code, rec.Body.String())
	}

	// Lista com filtro de estado e busca.
	if rec := do(gestorA, http.MethodGet, "/projetos?estado=planejado&q=ITIL", ""); rec.Code != http.StatusOK {
		t.Fatalf("listar: %d", rec.Code)
	}

	// Tarefas com data_limite.
	prazo := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	recTar := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/tarefas",
		`{"titulo":"Mapear filas","descricao":"3 entidades","estado":"a_fazer","ordem":1,"data_limite":"`+prazo+`"}`)
	if recTar.Code != http.StatusCreated {
		t.Fatalf("criar tarefa: %d %s", recTar.Code, recTar.Body.String())
	}
	var envTar struct {
		Data domain.Tarefa `json:"data"`
	}
	_ = json.Unmarshal(recTar.Body.Bytes(), &envTar)
	if envTar.Data.DataLimite == nil {
		t.Fatal("data_limite deveria ser persistida")
	}
	if rec := do(gestorB, http.MethodPost, "/projetos/"+projID.String()+"/tarefas", `{"titulo":"De outra unidade","estado":"a_fazer"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("tarefa fora da unidade: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(gestorB, http.MethodGet, "/projetos/"+projID.String()+"/tarefas", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("listar tarefas fora da unidade: %d", rec.Code)
	}

	recSeg := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/tarefas",
		`{"titulo":"Depois do mapa","estado":"a_fazer","ordem":2,"predecessora_id":"`+envTar.Data.ID.String()+`"}`)
	if recSeg.Code != http.StatusCreated {
		t.Fatalf("criar tarefa dependente: %d %s", recSeg.Code, recSeg.Body.String())
	}
	var envSeg struct {
		Data domain.Tarefa `json:"data"`
	}
	_ = json.Unmarshal(recSeg.Body.Bytes(), &envSeg)
	pessoa := dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, pessoa, uA); err != nil {
		t.Fatal(err)
	}
	var perfil uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO perfis (slug, nome, permissoes) VALUES ($1, 'Gestão de projetos', $2) RETURNING id`, "gestao-"+pessoa.String(), []string{"projetos:manage"}).Scan(&perfil); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id, unidade_id) VALUES ($1, $2, $3)`, pessoa, perfil, uA); err != nil {
		t.Fatal(err)
	}
	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String()+"/tarefas/"+envSeg.Data.ID.String(),
		`{"titulo":"Depois do mapa","estado":"em_andamento","ordem":2,"predecessora_id":"`+envTar.Data.ID.String()+`"}`); rec.Code != http.StatusConflict {
		t.Fatalf("mover sem responsável: %d %s", rec.Code, rec.Body.String())
	}
	comPessoa := `"responsavel_id":"` + pessoa.String() + `"`
	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String()+"/tarefas/"+envSeg.Data.ID.String(),
		`{"titulo":"Depois do mapa","estado":"em_andamento","ordem":2,"predecessora_id":"`+envTar.Data.ID.String()+`",`+comPessoa+`}`); rec.Code != http.StatusOK {
		t.Fatalf("andamento em paralelo: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String()+"/tarefas/"+envSeg.Data.ID.String(),
		`{"titulo":"Depois do mapa","estado":"concluida","ordem":2,"predecessora_id":"`+envTar.Data.ID.String()+`",`+comPessoa+`}`); rec.Code != http.StatusConflict {
		t.Fatalf("concluir fora de ordem: %d %s", rec.Code, rec.Body.String())
	}

	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String()+"/tarefas/"+envTar.Data.ID.String(),
		`{"titulo":"Mapear filas GLPI","descricao":"Feito","estado":"concluida","ordem":1,`+comPessoa+`}`); rec.Code != http.StatusOK {
		t.Fatalf("atualizar tarefa: %d %s", rec.Code, rec.Body.String())
	}

	// Vínculo com preenchimento automático de rótulo via Portas.
	recVinc := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/vinculos",
		`{"tipo":"ativo","alvo_id":"`+alvoID.String()+`","rotulo":""}`)
	if recVinc.Code != http.StatusCreated {
		t.Fatalf("criar vínculo: %d %s", recVinc.Code, recVinc.Body.String())
	}
	var envVinc struct {
		Data domain.Vinculo `json:"data"`
	}
	_ = json.Unmarshal(recVinc.Body.Bytes(), &envVinc)
	if envVinc.Data.Rotulo != "Ativo · Firewall SECITI" {
		t.Fatalf("rótulo da porta não aplicado: %+v", envVinc.Data)
	}

	// Lista com filtros de rastreabilidade reversa (alvo_id, alvo_tipo e produto_id).
	if rec := do(gestorA, http.MethodGet, "/projetos?alvo_id="+alvoID.String(), ""); rec.Code != http.StatusOK {
		t.Fatalf("listar por alvo_id: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodGet, "/projetos?alvo_tipo=ativo&alvo_id="+alvoID.String()+"&produto_id="+prodID.String(), ""); rec.Code != http.StatusOK {
		t.Fatalf("listar por alvo_tipo e produto_id: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodGet, "/projetos?produto_id=invalido", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("produto_id inválido: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodGet, "/projetos?alvo_id=invalido", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("alvo_id inválido: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodGet, "/projetos/"+projID.String()+"/tarefas", ""); rec.Code != http.StatusOK {
		t.Fatalf("listar tarefas: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodGet, "/projetos/"+projID.String()+"/vinculos", ""); rec.Code != http.StatusOK {
		t.Fatalf("listar vínculos: %d", rec.Code)
	}

	// Apontamento de horas e resumo.
	recAp := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/apontamentos",
		`{"tarefa_id":"`+envTar.Data.ID.String()+`","horas":4.5,"custo_hora":100,"descricao":"Configuração das 3 entidades"}`)
	if recAp.Code != http.StatusCreated {
		t.Fatalf("criar apontamento: %d %s", recAp.Code, recAp.Body.String())
	}
	var envAp struct {
		Data domain.Apontamento `json:"data"`
	}
	_ = json.Unmarshal(recAp.Body.Bytes(), &envAp)

	if rec := do(gestorA, http.MethodGet, "/projetos/"+projID.String()+"/apontamentos", ""); rec.Code != http.StatusOK {
		t.Fatalf("listar apontamentos: %d", rec.Code)
	}

	// Exclui apontamento, vínculo e tarefa.
	if rec := do(gestorA, http.MethodDelete, "/projetos/"+projID.String()+"/apontamentos/"+envAp.Data.ID.String(), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("excluir apontamento: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodDelete, "/projetos/"+projID.String()+"/vinculos/"+envVinc.Data.ID.String(), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("excluir vínculo: %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodDelete, "/projetos/"+projID.String()+"/tarefas/"+envTar.Data.ID.String(), ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("apagar sem confirmação: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(gestorA, http.MethodDelete, "/projetos/"+projID.String()+"/tarefas/"+envTar.Data.ID.String()+"?confirmar="+url.QueryEscape("Mapear filas GLPI"), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("excluir tarefa: %d", rec.Code)
	}

	// O gestor não cancela. Só o administrador, com motivo.
	if rec := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/estado", `{"estado":"cancelado"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("gestor cancelando pela passagem: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/cancelamento", `{"motivo":"não cabe mais"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("gestor cancelando: %d %s", rec.Code, rec.Body.String())
	}
	admin := gestorA
	admin.Roles = []string{string(auth.RoleAdmin)}
	if rec := do(admin, http.MethodPost, "/projetos/"+projID.String()+"/cancelamento", `{"motivo":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("cancelar sem motivo: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(admin, http.MethodPost, "/projetos/"+projID.String()+"/cancelamento", `{"motivo":"o teste encerrou o projeto"}`); rec.Code != http.StatusOK {
		t.Fatalf("administrador cancelando: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String(), `{"titulo":"Editado","unidade_id":"`+uA.String()+`"}`); rec.Code != http.StatusConflict {
		t.Fatalf("editar cancelado deveria dar 409, obteve %d", rec.Code)
	}
	if rec := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/vinculos", `{"tipo":"ativo","alvo_id":"`+alvoID.String()+`"}`); rec.Code != http.StatusConflict {
		t.Fatalf("vincular em cancelado deveria dar 409, obteve %d", rec.Code)
	}

	// Testa os mapeamentos de erro e portas desligadas/outra unidade.
	mod.ComPortas(portasFalsas{disponivel: false}, produtosFalsos{unidade: &uB})
	if _, err := mod.Service().Criar(ctx, gestorA, domain.EntradaProjeto{Titulo: "Com produto de B", UnidadeID: &uA, ProdutoID: &prodID}); !errors.Is(err, domain.ErrDeOutraUnidade) {
		t.Fatalf("produto de outra unidade: %v", err)
	}
	for _, e := range []error{
		nil,
		domain.ErrProjetoNaoEncontrado, domain.ErrTarefaNaoEncontrada, domain.ErrTituloObrigatorio,
		domain.ErrTransicaoInvalida, domain.ErrEstadoInvalido, domain.ErrProjetoEncerrado,
		domain.ErrPrioridadeInvalida, domain.ErrApontamentoInvalido, domain.ErrApontamentoNaoEncontrado,
		domain.ErrForaDoEscopo, domain.ErrVinculoInvalido, domain.ErrVinculoInexistente,
		domain.ErrDeOutraUnidade, domain.ErrIntegracaoDesligada,
		domain.ErrProjetoConcluido, domain.ErrTarefasAbertas, domain.ErrPredecessoraInvalida, domain.ErrPredecessoraAberta,
		errors.New("outro"),
	} {
		_ = application.MapError(e)
	}

	// Rotas com UUIDs ou JSON inválidos.
	for _, m := range []struct{ metodo, rota, corpo string }{
		{http.MethodGet, "/projetos/invalido", ""},
		{http.MethodPut, "/projetos/invalido", `{}`},
		{http.MethodPut, "/projetos/" + projID.String(), `{`},
		{http.MethodPost, "/projetos", `{`},
		{http.MethodPost, "/projetos/invalido/estado", `{}`},
		{http.MethodPost, "/projetos/" + projID.String() + "/estado", `{`},
		{http.MethodGet, "/projetos/invalido/tarefas", ""},
		{http.MethodPost, "/projetos/invalido/tarefas", `{}`},
		{http.MethodPost, "/projetos/" + projID.String() + "/tarefas", `{`},
		{http.MethodPut, "/projetos/" + projID.String() + "/tarefas/invalido", `{}`},
		{http.MethodPut, "/projetos/" + projID.String() + "/tarefas/" + uuid.NewString(), `{`},
		{http.MethodDelete, "/projetos/" + projID.String() + "/tarefas/invalido", ""},
		{http.MethodGet, "/projetos/invalido/vinculos", ""},
		{http.MethodPost, "/projetos/invalido/vinculos", `{}`},
		{http.MethodPost, "/projetos/" + projID.String() + "/vinculos", `{`},
		{http.MethodDelete, "/projetos/" + projID.String() + "/vinculos/invalido", ""},
		{http.MethodGet, "/projetos/invalido/apontamentos", ""},
		{http.MethodPost, "/projetos/invalido/apontamentos", `{}`},
		{http.MethodPost, "/projetos/" + projID.String() + "/apontamentos", `{`},
		{http.MethodDelete, "/projetos/" + projID.String() + "/apontamentos/invalido", ""},
		{http.MethodGet, "/projetos/invalido/carta", ""},
		{http.MethodPut, "/projetos/invalido/carta", `{}`},
		{http.MethodPut, "/projetos/" + projID.String() + "/carta", `{`},
	} {
		if rec := do(gestorA, m.metodo, m.rota, m.corpo); rec.Code < 400 {
			t.Fatalf("esperava erro em %s %s, obteve %d", m.metodo, m.rota, rec.Code)
		}
	}
}

func TestRepositorioEServicoCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	repo := infrastructure.NewRepository()
	id := uuid.New()

	// Caminhos de falha do banco (Fail, ScanFail, RowsErr).
	_ = repo.Criar(ctx, dbtest.Fail{}, &domain.Projeto{ID: id, Titulo: "X"})
	_, _ = repo.Obter(ctx, dbtest.Fail{}, id)
	_, _ = repo.Listar(ctx, dbtest.Fail{}, domain.FiltroProjetos{})
	_, _ = repo.Listar(ctx, dbtest.ScanFail{}, domain.FiltroProjetos{})
	_, _ = repo.Listar(ctx, dbtest.RowsErr{}, domain.FiltroProjetos{})
	_, _ = repo.Atualizar(ctx, dbtest.Fail{}, id, domain.EntradaProjeto{Titulo: "X"})
	_, _ = repo.Transicionar(ctx, dbtest.Fail{}, id, domain.EstadoEmAndamento)
	_, _, _ = repo.PosicaoUnidade(ctx, dbtest.Fail{}, id)
	_, _ = repo.Aprovadores(ctx, dbtest.Fail{}, &id)
	_ = repo.SalvarReabertura(ctx, dbtest.Fail{}, &domain.Reabertura{ID: id, ProjetoID: id})
	_, _, _ = repo.UltimaReabertura(ctx, dbtest.Fail{}, id)
	_, _ = repo.EstadoDaTarefa(ctx, dbtest.Fail{}, id)
	_ = repo.CriarTarefa(ctx, dbtest.Fail{}, &domain.Tarefa{ID: id, ProjetoID: id, Titulo: "T"})
	_, _ = repo.ObterTarefa(ctx, dbtest.Fail{}, id)
	_, _ = repo.NomeDaPessoa(ctx, dbtest.Fail{}, id, nil)
	_, _ = repo.AtualizarTarefa(ctx, dbtest.Fail{}, id, "T", "", "", nil, "", "a_fazer", 1, nil, nil)
	_, _ = repo.ListarTarefas(ctx, dbtest.Fail{}, id)
	_, _ = repo.ListarTarefas(ctx, dbtest.ScanFail{}, id)
	_, _ = repo.ListarTarefas(ctx, dbtest.RowsErr{}, id)
	_ = repo.ExcluirTarefa(ctx, dbtest.Fail{}, id)
	_ = repo.CriarVinculo(ctx, dbtest.Fail{}, &domain.Vinculo{ID: id, ProjetoID: id})
	_, _ = repo.ListarVinculos(ctx, dbtest.Fail{}, id)
	_, _ = repo.ListarVinculos(ctx, dbtest.ScanFail{}, id)
	_, _ = repo.ListarVinculos(ctx, dbtest.RowsErr{}, id)
	_ = repo.ExcluirVinculo(ctx, dbtest.Fail{}, id)
	_ = repo.CriarApontamento(ctx, dbtest.Fail{}, &domain.Apontamento{ID: id, ProjetoID: id})
	_, _ = repo.ListarApontamentos(ctx, dbtest.Fail{}, id)
	_, _ = repo.ListarApontamentos(ctx, dbtest.ScanFail{}, id)
	_, _ = repo.ListarApontamentos(ctx, dbtest.RowsErr{}, id)
	_ = repo.ExcluirApontamento(ctx, dbtest.Fail{}, id)

	// Serviço com passagens nulas/com erro e identidades alternativas no apontamento.
	pool := dbtest.Pool(t)
	svc := application.NewService(pool, repo, nil, nil)
	admin := auth.Identity{
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes:      []auth.Scope{{Permissions: []string{string(auth.PermProjetosManage)}}},
	}
	p, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Projeto Base"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID)
	})
	for _, idt := range []auth.Identity{
		{Username: "u1", Subject: uuid.NewString(), Permissions: admin.Permissions, Scopes: admin.Scopes},
		{Email: "e1@test", Subject: "nao-uuid", Permissions: admin.Permissions, Scopes: admin.Scopes},
		{Permissions: admin.Permissions, Scopes: admin.Scopes},
	} {
		if _, err := svc.RegistrarApontamento(ctx, idt, p.ID, nil, "", 1, "Ativ", 10); err != nil {
			t.Fatalf("apontamento com %+v: %v", idt, err)
		}
	}
	if _, err := svc.RegistrarApontamento(ctx, admin, p.ID, nil, "", 0, "", 0); err == nil {
		t.Fatal("apontamento inválido deveria falhar")
	}
	if _, err := svc.RegistrarApontamento(ctx, admin, uuid.New(), nil, "", 1, "Ativ", 0); err == nil {
		t.Fatal("apontamento em projeto inexistente deveria falhar")
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "T", "", nil, "", "invalido", 1, nil, nil, nil); !errors.Is(err, domain.ErrEstadoInvalido) {
		t.Fatalf("estado de tarefa inválido: %v", err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, uuid.New(), "T", "", "", nil, "", "a_fazer", 1, nil, nil); err == nil {
		t.Fatal("atualizar tarefa inexistente deveria falhar")
	}
	if _, err := svc.Vincular(ctx, admin, p.ID, "invalido", uuid.New(), ""); !errors.Is(err, domain.ErrVinculoInvalido) {
		t.Fatalf("vínculo inválido: %v", err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); !errors.Is(err, domain.ErrTransicaoInvalida) {
		t.Fatalf("transição inválida: %v", err)
	}
	svc.ComPassagens(passagensFalsas{err: errors.New("falha quadro")})
	if _, err := svc.Listar(ctx, admin, "", ""); err == nil {
		t.Fatal("listar com erro de passagens deveria falhar")
	}
	if _, err := svc.Obter(ctx, admin, p.ID); err == nil {
		t.Fatal("obter com erro de passagens deveria falhar")
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err == nil {
		t.Fatal("transicionar com erro de passagens deveria falhar")
	}
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "T", "", nil, "", "a_fazer", 1, nil, nil, nil); err == nil {
		t.Fatal("criar tarefa com erro de passagens deveria falhar")
	}
}

func TestReaberturaSoPorSuperiorOuAdministrador(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	pai := dbtest.Unidade(t, pool)
	var entidade, filha uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT entidade_id FROM unidades WHERE id = $1`, pai).Scan(&entidade); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO unidades (entidade_id, parent_id, nome, slug) VALUES ($1, $2, 'Filha', $3) RETURNING id`,
		entidade, pai, "f-"+uuid.NewString()[:12]).Scan(&filha); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	svc.ComPassagens(passagensFalsas{})

	gestor := auth.Identity{
		UserID: uuid.New(), Name: "Gestor da filha",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &filha, Unidades: []uuid.UUID{filha},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	superior := auth.Identity{
		UserID: uuid.New(), Name: "Chefia",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &pai, Unidades: []uuid.UUID{pai, filha},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	daEntidade := auth.Identity{
		UserID: uuid.New(), Name: "Secretaria",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			EntidadeID:  &entidade,
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	admin := auth.Identity{
		UserID: uuid.New(), Name: "Administrador",
		Roles: []string{auth.RoleAdmin}, Permissions: []string{"*"},
	}

	abrir := func(quem auth.Identity) *domain.Projeto {
		t.Helper()
		p, err := svc.Criar(ctx, quem, domain.EntradaProjeto{Titulo: "Portal", UnidadeID: &filha})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID) })
		if _, err := svc.Transicionar(ctx, quem, p.ID, domain.EstadoEmAndamento); err != nil {
			t.Fatal(err)
		}
		fechado, err := svc.Transicionar(ctx, quem, p.ID, domain.EstadoConcluido)
		if err != nil {
			t.Fatal(err)
		}
		return fechado
	}

	p := abrir(gestor)
	if _, err := svc.Atualizar(ctx, superior, p.ID, domain.EntradaProjeto{Titulo: "Portal alterado", UnidadeID: &filha}); !errors.Is(err, domain.ErrProjetoConcluido) {
		t.Fatalf("editar concluído: %v", err)
	}
	if _, err := svc.Transicionar(ctx, gestor, p.ID, domain.EstadoEmAndamento); !errors.Is(err, domain.ErrReaberturaExigeAprovacao) {
		t.Fatalf("passagem comum: %v", err)
	}
	if _, err := svc.Reabrir(ctx, gestor, p.ID, "quero voltar"); !errors.Is(err, domain.ErrReaberturaNaoAutorizada) {
		t.Fatalf("gestor da própria unidade: %v", err)
	}
	if _, err := svc.Reabrir(ctx, superior, p.ID, "  "); !errors.Is(err, domain.ErrMotivoObrigatorio) {
		t.Fatalf("motivo vazio: %v", err)
	}
	reaberto, err := svc.Reabrir(ctx, superior, p.ID, "  O escopo voltou  ")
	if err != nil {
		t.Fatal(err)
	}
	if reaberto.Estado != domain.EstadoEmAndamento || reaberto.Reabertura == nil || reaberto.Reabertura.Motivo != "O escopo voltou" || reaberto.Reabertura.AprovadorNome != "Chefia" {
		t.Fatalf("reaberto: %+v", reaberto.Reabertura)
	}

	outro := abrir(gestor)
	if _, err := svc.Reabrir(ctx, daEntidade, outro.ID, "Decisão da secretaria"); err != nil {
		t.Fatal(err)
	}
	institucional, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Institucional"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, institucional.ID) })
	if _, err := svc.Transicionar(ctx, admin, institucional.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, institucional.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reabrir(ctx, superior, institucional.ID, "não é da unidade dele"); !errors.Is(err, domain.ErrReaberturaNaoAutorizada) {
		t.Fatalf("superior de outra árvore no institucional: %v", err)
	}
	if _, err := svc.Reabrir(ctx, admin, institucional.ID, "Determinação da administração"); err != nil {
		t.Fatal(err)
	}
}

func TestConclusaoAvisaQuemPodeReabrir(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	pai := dbtest.Unidade(t, pool)
	var entidade, filha uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT entidade_id FROM unidades WHERE id = $1`, pai).Scan(&entidade); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO unidades (entidade_id, parent_id, nome, slug) VALUES ($1, $2, 'Filha aviso', $3) RETURNING id`,
		entidade, pai, "fa-"+uuid.NewString()[:12]).Scan(&filha); err != nil {
		t.Fatal(err)
	}

	conta := func(nome string, roles []string) uuid.UUID {
		t.Helper()
		slug := "av-" + uuid.NewString()[:10]
		if roles == nil {
			roles = []string{}
		}
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO users (username, email, display_name, keycloak_subject, roles)
			VALUES ($1, $1 || '@nova.test', $2, $1, $3) RETURNING id`, slug, nome, roles).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM notificacoes WHERE user_id = $1`, id)
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		})
		return id
	}
	var perfil uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO perfis (slug, nome, permissoes) VALUES ($1, 'Gestão de projetos', '{projetos:manage}') RETURNING id`,
		"prj-"+uuid.NewString()[:12]).Scan(&perfil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM perfis WHERE id = $1`, perfil) })
	escopo := func(user, unidade uuid.UUID, entidade *uuid.UUID) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id) VALUES ($1, $2, $3, $4)`,
			user, perfil, entidade, unidade); err != nil {
			t.Fatal(err)
		}
	}

	superior := conta("Chefia", nil)
	daUnidade := conta("Gestor", nil)
	daEntidade := conta("Secretaria", nil)
	admin := conta("Administrador", []string{auth.RoleAdmin})
	escopo(superior, pai, nil)
	escopo(daUnidade, filha, nil)
	if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id, entidade_id) VALUES ($1, $2, $3)`, daEntidade, perfil, entidade); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	gestor := auth.Identity{
		UserID: daUnidade, Name: "Gestor",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &filha, Unidades: []uuid.UUID{filha},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	p, err := svc.Criar(ctx, gestor, domain.EntradaProjeto{Titulo: "Portal da secretaria", UnidadeID: &filha})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID) })
	if _, err := svc.Transicionar(ctx, gestor, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, gestor, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}

	avisos := func(user uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM notificacoes WHERE user_id = $1 AND modulo = 'projetos' AND link = $2`,
			user, "/projetos/"+p.ID.String()).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if avisos(superior) != 1 || avisos(daEntidade) != 1 || avisos(admin) != 1 {
		t.Fatalf("quem reabre: superior %d entidade %d admin %d", avisos(superior), avisos(daEntidade), avisos(admin))
	}
	if avisos(daUnidade) != 0 {
		t.Fatalf("gestor da própria unidade foi avisado")
	}

	gerente := conta("Gerente do portal", nil)
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2), ($3, $2)`, gerente, filha, daUnidade); err != nil {
		t.Fatal(err)
	}
	comGerente, err := svc.Criar(ctx, gestor, domain.EntradaProjeto{Titulo: "Com gerente", UnidadeID: &filha, GerenteID: &gerente})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, comGerente.ID) })
	if _, err := svc.Transicionar(ctx, gestor, comGerente.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, gestor, comGerente.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	doGerente := func(user uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM notificacoes WHERE user_id = $1 AND modulo = 'projetos' AND link = $2`,
			user, "/projetos/"+comGerente.ID.String()).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if doGerente(gerente) != 3 {
		t.Fatalf("gerente na conclusão: %d", doGerente(gerente))
	}
	if doGerente(daUnidade) != 0 {
		t.Fatalf("quem concluiu foi avisado no projeto do gerente")
	}

	proprio, err := svc.Criar(ctx, gestor, domain.EntradaProjeto{Titulo: "Gerente conclui", UnidadeID: &filha, GerenteID: &daUnidade})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, proprio.ID) })
	if _, err := svc.Transicionar(ctx, gestor, proprio.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, gestor, proprio.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	var nProprio int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notificacoes WHERE user_id = $1 AND link = $2`,
		daUnidade, "/projetos/"+proprio.ID.String()).Scan(&nProprio); err != nil {
		t.Fatal(err)
	}
	if nProprio != 2 {
		t.Fatalf("gerente no próprio projeto: %d", nProprio)
	}
}

func TestEtapaHowNasceComOLider(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	slug := "ld-" + uuid.NewString()[:10]
	var lider uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users (username, email, display_name, keycloak_subject, roles)
		VALUES ($1, $1 || '@nova.test', 'Líder da unidade', $1, '{}') RETURNING id`, slug).Scan(&lider); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, lider) })
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, lider, unidade); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	gestor := auth.Identity{
		UserID: uuid.New(), Name: "Gestor",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &unidade, Unidades: []uuid.UUID{unidade},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	repo := infrastructure.NewRepository()
	comLider, err := svc.CriarGuiado(ctx, gestor, domain.EntradaProjeto{
		Titulo: "Portal", UnidadeID: &unidade, GerenteID: &lider,
	}, domain.EntradaCarta{}, []string{"Planejar"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, comLider.ID) })
	tarefas, err := repo.ListarTarefas(ctx, pool, comLider.ID)
	if err != nil || len(tarefas) != 1 || tarefas[0].ResponsavelID == nil || *tarefas[0].ResponsavelID != lider || tarefas[0].ResponsavelNome != "Líder da unidade" {
		t.Fatalf("etapa com líder: %+v %v", tarefas, err)
	}

	semLider, err := svc.CriarGuiado(ctx, gestor, domain.EntradaProjeto{
		Titulo: "Sem líder", UnidadeID: &unidade,
	}, domain.EntradaCarta{}, []string{"Solta"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, semLider.ID) })
	livres, err := repo.ListarTarefas(ctx, pool, semLider.ID)
	if err != nil || len(livres) != 1 || livres[0].ResponsavelID != nil || livres[0].ResponsavelNome != "" {
		t.Fatalf("etapa sem líder: %+v %v", livres, err)
	}
}

func TestReaberturaAvisaGerente(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	pai := dbtest.Unidade(t, pool)
	var entidade, filha uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT entidade_id FROM unidades WHERE id = $1`, pai).Scan(&entidade); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO unidades (entidade_id, parent_id, nome, slug) VALUES ($1, $2, 'Filha reabre', $3) RETURNING id`,
		entidade, pai, "fr-"+uuid.NewString()[:12]).Scan(&filha); err != nil {
		t.Fatal(err)
	}
	conta := func(nome string) uuid.UUID {
		t.Helper()
		slug := "rb-" + uuid.NewString()[:10]
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO users (username, email, display_name, keycloak_subject, roles)
			VALUES ($1, $1 || '@nova.test', $2, $1, '{}') RETURNING id`, slug, nome).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM notificacoes WHERE user_id = $1`, id)
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		})
		return id
	}
	gerente := conta("Gerente do portal")
	superior := conta("Chefia")
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2), ($3, $2)`, gerente, filha, superior); err != nil {
		t.Fatal(err)
	}
	var perfil uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO perfis (slug, nome, permissoes) VALUES ($1, 'Reabre projetos', '{projetos:manage}') RETURNING id`,
		"rb-"+uuid.NewString()[:12]).Scan(&perfil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_scopes WHERE perfil_id = $1`, perfil)
		_, _ = pool.Exec(context.Background(), `DELETE FROM perfis WHERE id = $1`, perfil)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id, unidade_id) VALUES ($1, $2, $3)`, superior, perfil, pai); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	gestor := auth.Identity{
		UserID: uuid.New(), Name: "Gestor",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &filha, Unidades: []uuid.UUID{filha},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	quemReabre := auth.Identity{
		UserID: superior, Name: "Chefia",
		Permissions: []string{string(auth.PermProjetosManage)},
		Scopes: []auth.Scope{{
			UnidadeID: &pai, Unidades: []uuid.UUID{pai, filha},
			Permissions: []string{string(auth.PermProjetosManage)},
		}},
	}
	fechar := func(titulo string, lider *uuid.UUID) *domain.Projeto {
		t.Helper()
		p, err := svc.Criar(ctx, gestor, domain.EntradaProjeto{Titulo: titulo, UnidadeID: &filha, GerenteID: lider})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID) })
		if _, err := svc.Transicionar(ctx, gestor, p.ID, domain.EstadoEmAndamento); err != nil {
			t.Fatal(err)
		}
		fechado, err := svc.Transicionar(ctx, gestor, p.ID, domain.EstadoConcluido)
		if err != nil {
			t.Fatal(err)
		}
		return fechado
	}
	aviso := func(user uuid.UUID, titulo string) (int, string) {
		t.Helper()
		var n int
		var mensagem string
		err := pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(mensagem), '') FROM notificacoes
			WHERE user_id = $1 AND titulo = $2`, user, titulo).Scan(&n, &mensagem)
		if err != nil {
			t.Fatal(err)
		}
		return n, mensagem
	}

	p := fechar("Portal", &gerente)
	if _, err := svc.Reabrir(ctx, quemReabre, p.ID, "O escopo voltou"); err != nil {
		t.Fatal(err)
	}
	titulo := "Projeto #" + strconv.FormatInt(p.Numero, 10) + " reaberto"
	n, mensagem := aviso(gerente, titulo)
	if n != 1 || !strings.Contains(mensagem, "O escopo voltou") || !strings.Contains(mensagem, "Chefia") {
		t.Fatalf("aviso do gerente: %d %q", n, mensagem)
	}
	if n, _ := aviso(superior, titulo); n != 0 {
		t.Fatalf("quem reabriu também foi avisado: %d", n)
	}

	proprio := fechar("Portal da chefia", &superior)
	if _, err := svc.Reabrir(ctx, quemReabre, proprio.ID, "Eu mesmo reabro"); err != nil {
		t.Fatal(err)
	}
	tituloProprio := "Projeto #" + strconv.FormatInt(proprio.Numero, 10) + " reaberto"
	if n, _ := aviso(superior, tituloProprio); n != 0 {
		t.Fatalf("gerente que reabre a si mesmo foi avisado: %d", n)
	}
}

type horasFalsas struct{ total float64 }

func (h horasFalsas) Somar(context.Context, database.DBTX, []uuid.UUID) (float64, error) {
	return h.total, nil
}

func TestConclusaoTarefaPrazoEHoras(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(pool, infrastructure.NewRepository(), outbox.NewWriter("test"), logger)
	admin := auth.Identity{
		UserID:      uuid.New(),
		Name:        "Administrador",
		Roles:       []string{auth.RoleAdmin},
		Permissions: []string{"*"},
	}
	p, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Com tarefas"})
	if err != nil {
		t.Fatal(err)
	}
	pessoa := dbtest.User(t, pool)
	var perfil uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO perfis (slug, nome, permissoes) VALUES ($1, 'Gestão de projetos', $2) RETURNING id`, "gestao-"+pessoa.String(), []string{"projetos:manage"}).Scan(&perfil); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id) VALUES ($1, $2)`, pessoa, perfil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, p.ID) })
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoEmAndamento); err != nil {
		t.Fatal(err)
	}
	aberta, err := svc.CriarTarefa(ctx, admin, p.ID, "Aberta", "", nil, "", "a_fazer", 1, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); !errors.Is(err, domain.ErrTarefasAbertas) {
		t.Fatalf("concluir com tarefa aberta: %v", err)
	}
	segunda, err := svc.CriarTarefa(ctx, admin, p.ID, "Segunda", "", nil, "", "a_fazer", 2, nil, &aberta.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, segunda.ID, "Segunda", "", "", &pessoa, "", "em_andamento", 2, nil, &aberta.ID); err != nil {
		t.Fatalf("andamento em paralelo: %v", err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, segunda.ID, "Segunda", "", "", &pessoa, "", "concluida", 2, nil, &aberta.ID); !errors.Is(err, domain.ErrPredecessoraAberta) {
		t.Fatalf("concluir fora de ordem: %v", err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, aberta.ID, "Aberta", "", "", &pessoa, "", "concluida", 1, nil, &segunda.ID); !errors.Is(err, domain.ErrPredecessoraInvalida) {
		t.Fatalf("ciclo: %v", err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, aberta.ID, "Aberta", "", "", &pessoa, "", "concluida", 1, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, segunda.ID, "Segunda", "", "", &pessoa, "", "concluida", 2, nil, &aberta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transicionar(ctx, admin, p.ID, domain.EstadoConcluido); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegistrarApontamento(ctx, admin, p.ID, nil, "", 1, "depois", 10); err == nil || !strings.Contains(err.Error(), "reabertura") {
		t.Fatalf("hora em concluído: %v", err)
	}

	outro, err := svc.Criar(ctx, admin, domain.EntradaProjeto{Titulo: "Com chamado"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, outro.ID) })
	alvo := uuid.New()
	if err := infrastructure.NewRepository().CriarVinculo(ctx, pool, &domain.Vinculo{
		ID: uuid.New(), ProjetoID: outro.ID, Tipo: domain.VinculoSolicitacao, AlvoID: alvo, Rotulo: "Chamado",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CriarTarefa(ctx, admin, outro.ID, "Prazo", "", nil, "", "a_fazer", 1, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	svc.ComHoras(horasFalsas{total: 7.5})
	visao, err := svc.ObterCarta(ctx, admin, outro.ID)
	if err != nil {
		t.Fatal(err)
	}
	if visao.HorasChamados != 7.5 {
		t.Fatalf("horas dos chamados: %v", visao.HorasChamados)
	}

	slug := "pz-" + uuid.NewString()[:10]
	var gerente uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users (username, email, display_name, keycloak_subject, roles)
		VALUES ($1, $1 || '@nova.test', 'Gerente', $1, $2) RETURNING id`, slug, []string{}).Scan(&gerente); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notificacoes WHERE user_id = $1`, gerente)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, gerente)
	})
	limite := time.Now().UTC().Add(-time.Hour)
	if _, err := pool.Exec(ctx, `UPDATE projetos SET gerente_id = $2, data_fim_planejada = $3, estado = 'em_andamento' WHERE id = $1`, outro.ID, gerente, limite); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projetos_tarefas SET data_limite = $2, responsavel_id = $3 WHERE projeto_id = $1`, outro.ID, limite, gerente); err != nil {
		t.Fatal(err)
	}
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notificacoes WHERE user_id = $1 AND modulo = 'projetos'`, gerente).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("avisos de prazo: %d", n)
	}
}
