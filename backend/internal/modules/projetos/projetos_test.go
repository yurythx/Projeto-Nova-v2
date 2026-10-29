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
	if mod.Manifest().Key != Key || mod.Service() == nil {
		t.Fatal("manifesto ou serviço inválido")
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
		`{"titulo":"Mapear filas","descricao":"3 entidades","responsavel_nome":"Yuri","estado":"a_fazer","ordem":1,"data_limite":"`+prazo+`"}`)
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

	if rec := do(gestorA, http.MethodPut, "/projetos/"+projID.String()+"/tarefas/"+envTar.Data.ID.String(),
		`{"titulo":"Mapear filas GLPI","descricao":"Feito","responsavel_nome":"Yuri","estado":"concluida","ordem":1}`); rec.Code != http.StatusOK {
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
	if rec := do(gestorA, http.MethodDelete, "/projetos/"+projID.String()+"/tarefas/"+envTar.Data.ID.String(), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("excluir tarefa: %d", rec.Code)
	}

	// Cancela o projeto e verifica que não aceita mais edições ou vínculos.
	if rec := do(gestorA, http.MethodPost, "/projetos/"+projID.String()+"/estado", `{"estado":"cancelado"}`); rec.Code != http.StatusOK {
		t.Fatalf("cancelar projeto: %d", rec.Code)
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
	_, _ = repo.EstadoDaTarefa(ctx, dbtest.Fail{}, id)
	_ = repo.CriarTarefa(ctx, dbtest.Fail{}, &domain.Tarefa{ID: id, ProjetoID: id, Titulo: "T"})
	_, _ = repo.AtualizarTarefa(ctx, dbtest.Fail{}, id, "T", "", "", "a_fazer", 1, nil)
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
		{Username: "u1", Subject: uuid.NewString()},
		{Email: "e1@test", Subject: "nao-uuid"},
		{},
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
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "T", "", "", "invalido", 1, nil); !errors.Is(err, domain.ErrEstadoInvalido) {
		t.Fatalf("estado de tarefa inválido: %v", err)
	}
	if _, err := svc.AtualizarTarefa(ctx, admin, uuid.New(), "T", "", "", "a_fazer", 1, nil); err == nil {
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
	if _, err := svc.CriarTarefa(ctx, admin, p.ID, "T", "", "", "a_fazer", 1, nil); err == nil {
		t.Fatal("criar tarefa com erro de passagens deveria falhar")
	}
}

