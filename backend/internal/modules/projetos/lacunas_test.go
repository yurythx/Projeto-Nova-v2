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
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

func TestMapErrorCobreOsMotivos(t *testing.T) {
	for _, err := range []error{
		nil,
		domain.ErrProjetoNaoEncontrado, domain.ErrTarefaNaoEncontrada, domain.ErrTituloObrigatorio,
		domain.ErrTransicaoInvalida, domain.ErrEstadoInvalido, domain.ErrProjetoEncerrado, domain.ErrProjetoConcluido,
		domain.ErrTarefasAbertas, domain.ErrPredecessoraInvalida, domain.ErrPredecessoraAberta,
		domain.ErrSubtarefaInvalida, domain.ErrSubtarefasAbertas, domain.ErrResponsavelObrigatorio,
		domain.ErrExclusaoNaoConfirmada, domain.ErrRelatoInvalido, domain.ErrAtualizacaoInvalida,
		domain.ErrAtualizacaoNaoAutorizada, domain.ErrMesInvalido, domain.ErrAnexoNaoEncontrado,
		domain.ErrPrioridadeInvalida, domain.ErrApontamentoInvalido, domain.ErrApontamentoNaoEncontrado,
		domain.ErrForaDoEscopo, domain.ErrVinculoInvalido, domain.ErrVinculoInexistente, domain.ErrDeOutraUnidade,
		domain.ErrIntegracaoDesligada, domain.ErrPeriodoInvalido, domain.ErrOrcamentoInvalido, domain.ErrEtapaInvalida,
		domain.ErrCartaInvalida, domain.ErrPessoaInvalida, domain.ErrForaDoProjeto, domain.ErrPessoasDemais,
		domain.ErrGerentePermanece, domain.ErrParticipanteAusente, domain.ErrReaberturaExigeAprovacao,
		domain.ErrCancelamentoNaoAutorizado, domain.ErrReaberturaNaoAutorizada, domain.ErrMotivoObrigatorio,
		errors.New("banco"),
	} {
		_ = application.MapError(err)
	}
}

func TestFuncoesQueACatracaNaoVia(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	unidade := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE users SET roles = ARRAY['nova-admin']::text[] WHERE id = $1`, pessoa); err != nil {
		if _, err = pool.Exec(ctx, `UPDATE users SET roles = '{nova-admin}' WHERE id = $1`, pessoa); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO directory_profiles (user_id, unidade_id) VALUES ($1, $2)`, pessoa, unidade); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	loja := storagetest.New()
	mod := New(modkit.Deps{Pool: pool, Outbox: outbox.NewWriter("test"), Logger: logger})
	svc := mod.Service()
	svc.ComAvisos(notificacoes.NewService(pool, nil, logger))
	admin := auth.Identity{UserID: pessoa, Name: "Ana Souza", Roles: []string{auth.RoleAdmin}}

	r := chi.NewRouter()
	var current auth.Identity
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(auth.WithIdentity(req.Context(), current)))
		})
	})
	mod.RegisterRoutes(kernel.Routes{Authed: r})
	do := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		current = admin
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
	exige := func(rec *httptest.ResponseRecorder, code int) {
		t.Helper()
		if rec.Code != code {
			t.Fatalf("%d, esperava %d: %s", rec.Code, code, rec.Body.String())
		}
	}
	idDe := func(rec *httptest.ResponseRecorder) uuid.UUID {
		t.Helper()
		var env struct {
			Data struct {
				ID uuid.UUID `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Data.ID == uuid.Nil {
			t.Fatalf("id: %v %s", err, rec.Body.String())
		}
		return env.Data.ID
	}

	ontem := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	mes := time.Now().UTC().Format("2006-01")
	rec := do(http.MethodPost, "/projetos", `{"titulo":"Portal da carta","descricao":"Escopo","objetivo":"Publicar","unidade_id":"`+unidade.String()+`","carta":{"onde":"SEMA","patrocinador":"Secretário"},"etapas":["Mapear","Publicar"]}`)
	exige(rec, http.StatusCreated)
	proj := idDe(rec)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM projetos WHERE id = $1`, proj) })

	base := "/projetos/" + proj.String()
	exige(do(http.MethodPut, base, `{"titulo":"Portal da carta","descricao":"Escopo","objetivo":"Publicar","unidade_id":"`+unidade.String()+`","gerente_id":"`+pessoa.String()+`","data_fim_planejada":"`+ontem+`"}`), http.StatusOK)
	exige(do(http.MethodGet, base+"/carta", ""), http.StatusOK)
	exige(do(http.MethodPut, base+"/carta", `{"titulo":"Portal da carta","descricao":"Escopo","objetivo":"Publicar","onde":"SEMA","patrocinador":"Secretário","metodologia":"ágil","centro_custo":"TI","orcamento":1000,"gerente_id":"`+pessoa.String()+`"}`), http.StatusOK)
	exige(do(http.MethodGet, base+"/historico", ""), http.StatusOK)
	exige(do(http.MethodGet, base+"/participantes", ""), http.StatusOK)
	exige(do(http.MethodGet, base+"/responsaveis", ""), http.StatusOK)
	exige(do(http.MethodGet, "/projetos/minhas", ""), http.StatusOK)
	exige(do(http.MethodGet, "/projetos/carga", ""), http.StatusOK)
	exige(do(http.MethodGet, "/projetos/prazos?mes="+mes, ""), http.StatusOK)
	exige(do(http.MethodGet, "/projetos/prazos?mes=13", ""), http.StatusUnprocessableEntity)
	exige(do(http.MethodGet, "/projetos/nao", ""), http.StatusBadRequest)

	rec = do(http.MethodPost, base+"/tarefas", `{"titulo":"Levantar filas","estado":"a_fazer","ordem":1,"data_limite":"`+ontem+`","pessoas":[{"id":"`+pessoa.String()+`"}]}`)
	exige(rec, http.StatusCreated)
	tarefa := idDe(rec)
	caminho := base + "/tarefas/" + tarefa.String()

	exige(do(http.MethodPost, caminho+"/atualizacoes", `{"texto":"   "}`), http.StatusUnprocessableEntity)
	exige(do(http.MethodPost, caminho+"/atualizacoes", `{"texto":"A fila do protocolo já está mapeada."}`), http.StatusCreated)
	exige(do(http.MethodGet, caminho+"/atualizacoes", ""), http.StatusOK)
	exige(do(http.MethodGet, caminho+"/movimentos", ""), http.StatusOK)
	exige(do(http.MethodPost, caminho+"/anexos/uploads", `{"nome":"planta.pdf","content_type":"application/pdf"}`), http.StatusServiceUnavailable)
	svc.ComArquivos(application.Arquivos{Store: loja, Bucket: "nova", MaxBytes: 10 << 20, Expiry: time.Minute})
	exige(do(http.MethodPost, caminho+"/anexos/uploads", `{"nome":"","content_type":"application/pdf"}`), http.StatusUnprocessableEntity)
	rec = do(http.MethodPost, caminho+"/anexos/uploads", `{"nome":"planta.pdf","content_type":"application/pdf"}`)
	exige(rec, http.StatusCreated)
	var ticket struct {
		Data struct {
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ticket); err != nil || ticket.Data.ObjectKey == "" {
		t.Fatal(rec.Body.String())
	}
	if err := loja.Put(ctx, "nova", ticket.Data.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	rec = do(http.MethodPost, caminho+"/anexos", `{"nome":"planta.pdf","object_key":"`+ticket.Data.ObjectKey+`"}`)
	exige(rec, http.StatusCreated)
	anexo := idDe(rec)
	exige(do(http.MethodGet, caminho+"/anexos", ""), http.StatusOK)
	exige(do(http.MethodGet, caminho+"/anexos/"+anexo.String(), ""), http.StatusOK)
	loja.FailPresign = true
	exige(do(http.MethodGet, caminho+"/anexos/"+anexo.String(), ""), http.StatusServiceUnavailable)
	loja.FailPresign = false

	alvo := uuid.New()
	rec = do(http.MethodPost, base+"/vinculos", `{"tipo":"ativo","alvo_id":"`+alvo.String()+`","rotulo":"Firewall"}`)
	exige(rec, http.StatusCreated)
	vinculo := idDe(rec)
	exige(do(http.MethodGet, base+"/vinculos", ""), http.StatusOK)
	exige(do(http.MethodDelete, base+"/vinculos/"+vinculo.String(), ""), http.StatusNoContent)

	rec = do(http.MethodPost, base+"/apontamentos", `{"tarefa_id":"`+tarefa.String()+`","data_apontamento":"`+time.Now().UTC().Format("2006-01-02")+`","horas":2,"descricao":"Levantamento","custo_hora":40}`)
	exige(rec, http.StatusCreated)
	apontamento := idDe(rec)
	exige(do(http.MethodGet, base+"/apontamentos", ""), http.StatusOK)
	exige(do(http.MethodDelete, base+"/apontamentos/"+apontamento.String(), ""), http.StatusNoContent)

	if err := svc.VerificarPrazos(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	leitor := auth.Identity{UserID: uuid.New(), Permissions: []string{string(auth.PermProjetosRead)}}
	if _, err := svc.Cancelar(ctx, leitor, proj, "não cabe mais"); !errors.Is(err, domain.ErrCancelamentoNaoAutorizado) {
		t.Fatalf("cancelar: %v", err)
	}
	exige(do(http.MethodDelete, caminho+"?confirmar=outro", ""), http.StatusUnprocessableEntity)
	exige(do(http.MethodDelete, caminho+"?confirmar=Levantar%20filas", ""), http.StatusNoContent)
	exige(do(http.MethodPost, base+"/cancelamento", `{"motivo":"o portal saiu do plano"}`), http.StatusOK)
}
