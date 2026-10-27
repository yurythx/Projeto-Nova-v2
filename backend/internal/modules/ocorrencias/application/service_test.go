package application_test

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
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

var errFalha = errors.New("falha injetada")

// eventos guarda o que foi publicado; falha para os nomes pedidos.
type eventos struct {
	nomes []string
	falha map[string]bool
}

func (e *eventos) Write(_ context.Context, _ pgx.Tx, nome, _, _ string, _ uuid.UUID, _ any) error {
	if e.falha[nome] {
		return errFalha
	}
	e.nomes = append(e.nomes, nome)
	return nil
}

// falhaRepo falha nos métodos pedidos e delega o resto ao banco.
type falhaRepo struct {
	domain.Repository
	falha map[string]bool
}

func falhando(metodos ...string) falhaRepo {
	f := falhaRepo{Repository: infrastructure.NewRepository(), falha: map[string]bool{}}
	for _, m := range metodos {
		f.falha[m] = true
	}
	return f
}

func (f falhaRepo) Insert(ctx context.Context, db database.DBTX, o domain.Ocorrencia) error {
	if f.falha["Insert"] {
		return errFalha
	}
	return f.Repository.Insert(ctx, db, o)
}
func (f falhaRepo) InsertAtualizacao(ctx context.Context, db database.DBTX, a domain.Atualizacao) error {
	if f.falha["InsertAtualizacao"] {
		return errFalha
	}
	return f.Repository.InsertAtualizacao(ctx, db, a)
}
func (f falhaRepo) Update(ctx context.Context, db database.DBTX, o domain.Ocorrencia) error {
	if f.falha["Update"] {
		return errFalha
	}
	return f.Repository.Update(ctx, db, o)
}
func (f falhaRepo) Travar(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Ocorrencia, error) {
	if f.falha["Travar"] {
		return domain.Ocorrencia{}, errFalha
	}
	return f.Repository.Travar(ctx, db, id)
}
func (f falhaRepo) List(ctx context.Context, db database.DBTX, limite int) ([]domain.Ocorrencia, error) {
	if f.falha["List"] {
		return nil, errFalha
	}
	return f.Repository.List(ctx, db, limite)
}
func (f falhaRepo) ParaPagina(ctx context.Context, db database.DBTX, desde time.Time) ([]domain.Ocorrencia, error) {
	if f.falha["ParaPagina"] {
		return nil, errFalha
	}
	return f.Repository.ParaPagina(ctx, db, desde)
}

type cenario struct {
	svc     *application.Service
	ev      *eventos
	quem    auth.Identity
	outro   uuid.UUID
	novoSvc func(r domain.Repository, ev *eventos) *application.Service
}

func novoCenario(t *testing.T) cenario {
	t.Helper()
	pool := dbtest.Pool(t)
	ev := &eventos{falha: map[string]bool{}}
	quem, outro := dbtest.User(t, pool), dbtest.User(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ocorrencias WHERE coordenador_id = ANY($1)`, []uuid.UUID{quem, outro})
	})
	novo := func(r domain.Repository, e *eventos) *application.Service { return application.NewService(pool, r, e) }
	return cenario{svc: novo(infrastructure.NewRepository(), ev), ev: ev, quem: auth.Identity{UserID: quem}, outro: outro, novoSvc: novo}
}

var entrada = application.DeclaracaoEntrada{Titulo: "Protocolo fora do ar", Servicos: []string{"Protocolo"}, Impacto: domain.ImpactoFora, Texto: "Investigando."}

func TestCicloPeloServico(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	o, err := c.svc.Declarar(ctx, c.quem, entrada)
	if err != nil || o.Estado != domain.EstadoInvestigando || len(o.Atualizacoes) != 1 || *o.CoordenadorID != c.quem.UserID {
		t.Fatalf("declarar: %+v %v", o, err)
	}
	o, err = c.svc.Atualizar(ctx, c.quem, o.ID, application.AtualizacaoEntrada{Estado: domain.EstadoIdentificado, Impacto: domain.ImpactoParcial, Texto: "Certificado vencido."})
	if err != nil || o.Impacto != domain.ImpactoParcial || len(o.Atualizacoes) != 2 {
		t.Fatalf("atualizar: %+v %v", o, err)
	}
	if o, err = c.svc.TrocarCoordenador(ctx, o.ID, c.outro); err != nil || *o.CoordenadorID != c.outro {
		t.Fatalf("coordenador: %+v %v", o, err)
	}
	if _, err := c.svc.Revisar(ctx, o.ID, "cedo"); !errors.Is(err, domain.ErrAindaAberta) {
		t.Fatalf("revisão antes de resolver: %v", err)
	}
	o, err = c.svc.Atualizar(ctx, c.quem, o.ID, application.AtualizacaoEntrada{Estado: domain.EstadoResolvido, Impacto: domain.ImpactoParcial, Texto: "Voltou."})
	if err != nil || !o.Resolvida() {
		t.Fatalf("resolver: %+v %v", o, err)
	}
	if o, err = c.svc.Revisar(ctx, o.ID, "Causa: renovação manual."); err != nil || o.Revisao == "" {
		t.Fatalf("revisar: %+v %v", o, err)
	}
	quer := []string{application.EventoDeclarada, application.EventoAtualizada, application.EventoCoordenacao, application.EventoResolvida, application.EventoRevisada}
	if strings.Join(c.ev.nomes, ",") != strings.Join(quer, ",") {
		t.Fatalf("eventos: %v", c.ev.nomes)
	}

	s, err := c.svc.Situacao(ctx)
	if err != nil {
		t.Fatal(err)
	}
	achou := false
	for _, p := range s.Recentes {
		achou = achou || p.ID == o.ID
	}
	if !achou {
		t.Fatalf("a resolvida aparece entre as recentes: %+v", s)
	}
	if lista, err := c.svc.Listar(ctx); err != nil || len(lista) == 0 {
		t.Fatalf("painel: %v", err)
	}
}

func TestCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	aberta, err := c.svc.Declarar(ctx, c.quem, entrada)
	if err != nil {
		t.Fatal(err)
	}
	atualiza := application.AtualizacaoEntrada{Estado: domain.EstadoMonitorando, Impacto: domain.ImpactoFora, Texto: "ok"}
	comFalha := func(m ...string) *application.Service { return c.novoSvc(falhando(m...), &eventos{}) }
	eventoFalha := func(nome string) *application.Service {
		return c.novoSvc(infrastructure.NewRepository(), &eventos{falha: map[string]bool{nome: true}})
	}
	casos := map[string]struct {
		err    error
		espera error
	}{}
	run := func(nome string, err, espera error) { casos[nome] = struct{ err, espera error }{err, espera} }

	_, err = c.svc.Declarar(ctx, c.quem, application.DeclaracaoEntrada{Titulo: " ", Servicos: []string{"x"}, Impacto: domain.ImpactoFora, Texto: "x"})
	run("declarar sem título", err, domain.ErrEntradaInvalida)
	_, err = comFalha("Insert").Declarar(ctx, c.quem, entrada)
	run("declarar, falha ao gravar", err, errFalha)
	_, err = comFalha("InsertAtualizacao").Declarar(ctx, c.quem, entrada)
	run("declarar, falha na linha do tempo", err, errFalha)
	_, err = eventoFalha(application.EventoDeclarada).Declarar(ctx, c.quem, entrada)
	run("declarar, falha no outbox", err, errFalha)
	_, err = c.svc.Atualizar(ctx, c.quem, uuid.New(), atualiza)
	run("atualizar inexistente", err, domain.ErrNaoEncontrada)
	_, err = c.svc.Atualizar(ctx, c.quem, aberta.ID, application.AtualizacaoEntrada{Estado: "x", Impacto: domain.ImpactoFora, Texto: "x"})
	run("atualizar com estado inventado", err, domain.ErrEntradaInvalida)
	_, err = comFalha("Travar").Atualizar(ctx, c.quem, aberta.ID, atualiza)
	run("atualizar, falha ao travar", err, errFalha)
	_, err = comFalha("InsertAtualizacao").Atualizar(ctx, c.quem, aberta.ID, atualiza)
	run("atualizar, falha na linha do tempo", err, errFalha)
	_, err = comFalha("Update").Atualizar(ctx, c.quem, aberta.ID, atualiza)
	run("atualizar, falha ao regravar", err, errFalha)
	_, err = c.svc.TrocarCoordenador(ctx, aberta.ID, uuid.New())
	run("coordenador inexistente", err, domain.ErrCoordenadorInvalido)
	_, err = c.svc.Obter(ctx, uuid.New())
	run("obter inexistente", err, domain.ErrNaoEncontrada)
	_, err = comFalha("List").Listar(ctx)
	run("painel, falha", err, errFalha)
	_, err = comFalha("ParaPagina").Situacao(ctx)
	run("página, falha", err, errFalha)
	for nome, k := range casos {
		if !errors.Is(k.err, k.espera) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}

	if _, err := c.svc.Atualizar(ctx, c.quem, aberta.ID, application.AtualizacaoEntrada{Estado: domain.EstadoResolvido, Impacto: domain.ImpactoFora, Texto: "fim"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.svc.Atualizar(ctx, c.quem, aberta.ID, atualiza); !errors.Is(err, domain.ErrJaResolvida) {
		t.Fatalf("resolvida não recebe atualização: %v", err)
	}
}

// A página mostra a resolvida por 7 dias e depois a esquece.
func TestResolvidaSaiDaPaginaDepoisDeUmaSemana(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	o, err := c.svc.Declarar(ctx, c.quem, entrada)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.svc.Atualizar(ctx, c.quem, o.ID, application.AtualizacaoEntrada{Estado: domain.EstadoResolvido, Impacto: domain.ImpactoFora, Texto: "Voltou."}); err != nil {
		t.Fatal(err)
	}
	naPagina := func(s domain.Situacao) bool {
		for _, p := range s.Recentes {
			if p.ID == o.ID {
				return true
			}
		}
		return false
	}
	seisDias := c.svc.WithRelogio(func() time.Time { return time.Now().Add(6 * 24 * time.Hour) })
	if s, _ := seisDias.Situacao(ctx); !naPagina(s) {
		t.Fatal("seis dias depois, ainda aparece")
	}
	oitoDias := c.svc.WithRelogio(func() time.Time { return time.Now().Add(8 * 24 * time.Hour) })
	if s, _ := oitoDias.Situacao(ctx); naPagina(s) {
		t.Fatal("oito dias depois, saiu da página")
	}
}

func TestMapError(t *testing.T) {
	for err, status := range map[error]int{
		domain.ErrNaoEncontrada:       http.StatusNotFound,
		domain.ErrEntradaInvalida:     http.StatusUnprocessableEntity,
		domain.ErrJaResolvida:         http.StatusConflict,
		domain.ErrAindaAberta:         http.StatusConflict,
		domain.ErrCoordenadorInvalido: http.StatusUnprocessableEntity,
	} {
		api, ok := apperrors.As(application.MapError(err))
		if !ok || api.Status != status {
			t.Errorf("%v: %+v", err, api)
		}
	}
	if application.MapError(nil) != nil || !errors.Is(application.MapError(errFalha), errFalha) {
		t.Fatal("nil continua nil; erro desconhecido segue como veio")
	}
	pronto := apperrors.BadRequest("id inválido")
	if application.MapError(pronto) != pronto {
		t.Fatal("erro HTTP pronto não é trocado")
	}
}

var silencio = slog.New(slog.NewTextHandler(io.Discard, nil))

func pedido(t *testing.T, metodo, id string, corpo any, quem auth.Identity) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if corpo != nil {
		_ = json.NewEncoder(&buf).Encode(corpo)
	}
	req := httptest.NewRequest(metodo, "/x", &buf)
	rctx := chi.NewRouteContext()
	if id != "" {
		rctx.URLParams.Add("id", id)
	}
	return req.WithContext(auth.WithIdentity(context.WithValue(req.Context(), chi.RouteCtxKey, rctx), quem))
}

func TestHandlers(t *testing.T) {
	c := novoCenario(t)
	h := transport.NewHandlers(c.svc, silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, id, corpo, c.quem))
		return rec
	}
	rec := chama(h.Declarar, http.MethodPost, "", map[string]any{"titulo": "Rede fora", "servicos": []string{"Rede"}, "impacto": "fora", "texto": "Investigando."})
	if rec.Code != http.StatusCreated {
		t.Fatalf("declarar: %d %s", rec.Code, rec.Body.String())
	}
	var criada struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &criada)
	id := criada.Data.ID

	pub := chama(h.Situacao, http.MethodGet, "", nil)
	if pub.Code != http.StatusOK || pub.Header().Get("Cache-Control") != "public, max-age=30" || !strings.Contains(pub.Body.String(), `"geral":"fora"`) {
		t.Fatalf("página: %d %v %s", pub.Code, pub.Header(), pub.Body.String())
	}
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"painel":                    {chama(h.Listar, http.MethodGet, "", nil), http.StatusOK},
		"obter":                     {chama(h.Obter, http.MethodGet, id, nil), http.StatusOK},
		"atualizar":                 {chama(h.Atualizar, http.MethodPost, id, map[string]any{"estado": "identificado", "impacto": "fora", "texto": "Achamos."}), http.StatusOK},
		"coordenador":               {chama(h.TrocarCoordenador, http.MethodPut, id, map[string]any{"user_id": c.outro}), http.StatusOK},
		"revisão de aberta":         {chama(h.Revisar, http.MethodPut, id, map[string]any{"texto": "cedo"}), http.StatusConflict},
		"declarar sem campos":       {chama(h.Declarar, http.MethodPost, "", map[string]any{}), http.StatusUnprocessableEntity},
		"declarar com impacto ruim": {chama(h.Declarar, http.MethodPost, "", map[string]any{"titulo": "x", "servicos": []string{"y"}, "impacto": "leve", "texto": "z"}), http.StatusUnprocessableEntity},
		"serviço só com espaços":    {chama(h.Declarar, http.MethodPost, "", map[string]any{"titulo": "x", "servicos": []string{"  "}, "impacto": "fora", "texto": "z"}), http.StatusUnprocessableEntity},
		"obter com id inválido":     {chama(h.Obter, http.MethodGet, "x", nil), http.StatusBadRequest},
		"obter inexistente":         {chama(h.Obter, http.MethodGet, uuid.NewString(), nil), http.StatusNotFound},
		"atualizar com id inválido": {chama(h.Atualizar, http.MethodPost, "x", nil), http.StatusBadRequest},
		"atualizar sem campos":      {chama(h.Atualizar, http.MethodPost, id, map[string]any{}), http.StatusUnprocessableEntity},
		"atualizar inexistente":     {chama(h.Atualizar, http.MethodPost, uuid.NewString(), map[string]any{"estado": "identificado", "impacto": "fora", "texto": "x"}), http.StatusNotFound},
		"coordenador id inválido":   {chama(h.TrocarCoordenador, http.MethodPut, "x", nil), http.StatusBadRequest},
		"coordenador sem campo":     {chama(h.TrocarCoordenador, http.MethodPut, id, map[string]any{}), http.StatusUnprocessableEntity},
		"coordenador inexistente":   {chama(h.TrocarCoordenador, http.MethodPut, id, map[string]any{"user_id": uuid.New()}), http.StatusUnprocessableEntity},
		"revisão id inválido":       {chama(h.Revisar, http.MethodPut, "x", nil), http.StatusBadRequest},
		"revisão sem texto":         {chama(h.Revisar, http.MethodPut, id, map[string]any{}), http.StatusUnprocessableEntity},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
	if rec := chama(h.Atualizar, http.MethodPost, id, map[string]any{"estado": "resolvido", "impacto": "fora", "texto": "Voltou."}); rec.Code != http.StatusOK {
		t.Fatalf("resolver: %d", rec.Code)
	}
	if rec := chama(h.Revisar, http.MethodPut, id, map[string]any{"texto": "Causa."}); rec.Code != http.StatusOK {
		t.Fatalf("revisar: %d %s", rec.Code, rec.Body.String())
	}

	quebrado := transport.NewHandlers(c.novoSvc(falhando("List", "ParaPagina"), &eventos{}), silencio)
	for nome, fn := range map[string]http.HandlerFunc{"painel": quebrado.Listar, "página": quebrado.Situacao} {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, http.MethodGet, "", nil, c.quem))
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s com o banco fora: %d", nome, rec.Code)
		}
	}
}
