package application_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func (f falhaRepo) ModeloDaOferta(ctx context.Context, db database.DBTX, o uuid.UUID) ([]domain.ModeloTarefa, error) {
	if f.falha["ModeloDaOferta"] {
		return nil, errFalha
	}
	return f.Repository.ModeloDaOferta(ctx, db, o)
}
func (f falhaRepo) SalvarModelo(ctx context.Context, db database.DBTX, o uuid.UUID, m []domain.ModeloTarefa) error {
	if f.falha["SalvarModelo"] {
		return errFalha
	}
	return f.Repository.SalvarModelo(ctx, db, o, m)
}
func (f falhaRepo) InsertTarefas(ctx context.Context, db database.DBTX, ts []domain.Tarefa) error {
	if f.falha["InsertTarefas"] {
		return errFalha
	}
	return f.Repository.InsertTarefas(ctx, db, ts)
}
func (f falhaRepo) ListTarefas(ctx context.Context, db database.DBTX, s uuid.UUID) ([]domain.Tarefa, error) {
	if f.falha["ListTarefas"] {
		return nil, errFalha
	}
	return f.Repository.ListTarefas(ctx, db, s)
}
func (f falhaRepo) GetTarefa(ctx context.Context, db database.DBTX, s, id uuid.UUID) (domain.Tarefa, error) {
	if f.falha["GetTarefa"] {
		return domain.Tarefa{}, errFalha
	}
	return f.Repository.GetTarefa(ctx, db, s, id)
}
func (f falhaRepo) UpdateTarefa(ctx context.Context, db database.DBTX, t domain.Tarefa) error {
	if f.falha["UpdateTarefa"] {
		return errFalha
	}
	return f.Repository.UpdateTarefa(ctx, db, t)
}
func (f falhaRepo) MembroDe(ctx context.Context, db database.DBTX, u uuid.UUID, es []uuid.UUID) ([]uuid.UUID, error) {
	if f.falha["MembroDe"] {
		return nil, errFalha
	}
	return f.Repository.MembroDe(ctx, db, u, es)
}
func (f falhaRepo) GetEquipe(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Equipe, error) {
	if f.falha["GetEquipe"] {
		return domain.Equipe{}, errFalha
	}
	return f.Repository.GetEquipe(ctx, db, id)
}

// comTarefas monta uma oferta de admissão com duas tarefas: uma do
// suporte (do técnico do cenário) e outra da rede (de outra pessoa).
type comTarefas struct {
	cenario
	oferta        domain.Oferta
	suporte, rede domain.Equipe
	daRede        auth.Identity
	modelo        []domain.ModeloTarefa
}

func novoComTarefas(t *testing.T) comTarefas {
	t.Helper()
	ctx := context.Background()
	c := novoCenario(t)
	o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "Admissão " + uuid.NewString()[:6], Tipo: domain.TipoRequisicao, UnidadeID: &c.unidade})
	if err != nil {
		t.Fatal(err)
	}
	daRede := atendente(c.unidade)
	daRede.UserID = dbtest.User(t, c.pool)
	suporte, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "Suporte", UnidadeID: &c.unidade, Membros: []uuid.UUID{c.tecnico.UserID}})
	if err != nil {
		t.Fatal(err)
	}
	rede, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "Rede", UnidadeID: &c.unidade, Membros: []uuid.UUID{daRede.UserID}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = c.pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = c.pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		_, _ = c.pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE id = ANY($1)`, []uuid.UUID{suporte.ID, rede.ID})
	})
	modelo := []domain.ModeloTarefa{{Titulo: "Criar conta", EquipeID: suporte.ID}, {Titulo: "Liberar VPN", EquipeID: rede.ID}}
	salvo, err := c.base.SalvarModelo(ctx, admin(), o.ID, modelo)
	if err != nil || len(salvo) != 2 || salvo[1].EquipeNome != "Rede" {
		t.Fatalf("modelo: %+v %v", salvo, err)
	}
	return comTarefas{cenario: c, oferta: o, suporte: suporte, rede: rede, daRede: daRede, modelo: modelo}
}

func (c comTarefas) pedir(t *testing.T) domain.Solicitacao {
	t.Helper()
	s, err := c.base.Abrir(context.Background(), c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Admitir Ana"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTarefasAteAtender(t *testing.T) {
	ctx := context.Background()
	c := novoComTarefas(t)
	sol := c.pedir(t)
	d, err := c.base.Obter(ctx, c.tecnico, sol.ID)
	if err != nil || len(d.Tarefas) != 2 || !d.Tarefas[0].Minha || d.Tarefas[1].Minha || d.Tarefas[1].EquipeNome != "Rede" {
		t.Fatalf("tarefas para quem atende: %+v %v", d.Tarefas, err)
	}
	if d, _ := c.base.Obter(ctx, c.servidor, sol.ID); len(d.Tarefas) != 0 {
		t.Fatalf("quem pediu não vê as tarefas: %+v", d.Tarefas)
	}
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoAtendido); !errors.Is(err, domain.ErrTarefasPendentes) {
		t.Fatalf("atendido com pendência: %v", err)
	}
	minha, daRede := d.Tarefas[0].ID, d.Tarefas[1].ID
	for nome, k := range map[string]struct {
		quem     auth.Identity
		tarefa   uuid.UUID
		concluir bool
		obs      string
		espera   error
	}{
		"de outra equipe":        {c.tecnico, daRede, true, "", domain.ErrNaoEhDaEquipe},
		"quem pediu":             {c.servidor, minha, true, "", domain.ErrNaoEhDaEquipe},
		"dispensa sem motivo":    {c.tecnico, daRede, false, " ", domain.ErrDispensaSemMotivo},
		"tarefa inexistente":     {c.tecnico, uuid.New(), true, "", domain.ErrNaoEncontrado},
		"de fora do atendimento": {atendente(uuid.New()), minha, true, "", domain.ErrNaoEncontrado},
	} {
		if _, err := c.base.FazerTarefa(ctx, k.quem, sol.ID, k.tarefa, k.concluir, k.obs); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	d, err = c.base.FazerTarefa(ctx, c.tecnico, sol.ID, minha, true, "Conta ana.silva")
	if err != nil || d.Tarefas[0].Estado != domain.TarefaConcluida || d.Tarefas[0].Observacao != "Conta ana.silva" {
		t.Fatalf("concluir a minha: %+v %v", d.Tarefas, err)
	}
	if _, err := c.base.FazerTarefa(ctx, c.tecnico, sol.ID, minha, true, ""); !errors.Is(err, domain.ErrTarefaFeita) {
		t.Fatalf("de novo: %v", err)
	}
	d, err = c.base.FazerTarefa(ctx, c.daRede, sol.ID, daRede, true, "")
	if err != nil || domain.Pendentes(d.Tarefas) != 0 {
		t.Fatalf("a rede conclui a dela: %+v %v", d.Tarefas, err)
	}
	if d, err := c.base.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoAtendido); err != nil || d.Solicitacao.Estado != domain.EstadoAtendido {
		t.Fatalf("sem pendência, atende: %+v %v", d.Solicitacao, err)
	}
}

func TestTarefasDispensadaECancelada(t *testing.T) {
	ctx := context.Background()
	c := novoComTarefas(t)
	sol := c.pedir(t)
	d, _ := c.base.Obter(ctx, c.tecnico, sol.ID)
	// Quem atende dispensa a tarefa de outra equipe, com motivo.
	d, err := c.base.FazerTarefa(ctx, c.tecnico, sol.ID, d.Tarefas[1].ID, false, "Ana não usa VPN")
	if err != nil || d.Tarefas[1].Estado != domain.TarefaDispensada {
		t.Fatalf("dispensar: %+v %v", d.Tarefas, err)
	}
	if _, err := c.base.Transicionar(ctx, c.servidor, sol.ID, domain.EstadoCancelado); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.FazerTarefa(ctx, c.tecnico, sol.ID, d.Tarefas[0].ID, true, ""); !errors.Is(err, domain.ErrSolicitacaoFinalizada) {
		t.Fatalf("pedido cancelado: %v", err)
	}
}

func TestTarefasDepoisDaAprovacao(t *testing.T) {
	ctx := context.Background()
	c := novoComAprovacao(t)
	if _, err := c.base.SalvarModelo(ctx, admin(), c.oferta.ID, []domain.ModeloTarefa{{Titulo: "Criar conta", EquipeID: equipeDe(t, c.cenario)}}); err != nil {
		t.Fatal(err)
	}
	sol := c.pedir(t)
	if d, _ := c.base.Obter(ctx, c.tecnico, sol.ID); len(d.Tarefas) != 0 {
		t.Fatal("antes da aprovação, nenhuma tarefa")
	}
	if _, err := c.base.DecidirAprovacao(ctx, c.chefe, sol.ID, true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.com(falhando("ModeloDaOferta"), eventosFalham{}).DecidirAprovacao(ctx, c.ti, sol.ID, true, ""); !errors.Is(err, errFalha) {
		t.Fatalf("falha ao criar as tarefas na liberação: %v", err)
	}
	if _, err := c.base.DecidirAprovacao(ctx, c.ti, sol.ID, true, ""); err != nil {
		t.Fatal(err)
	}
	if d, _ := c.base.Obter(ctx, c.tecnico, sol.ID); len(d.Tarefas) != 1 || !d.Tarefas[0].Minha {
		t.Fatalf("aprovado, as tarefas nascem: %+v", d.Tarefas)
	}
}

// equipeDe devolve a equipe N1 do cenário (a do técnico).
func equipeDe(t *testing.T, c cenario) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := c.pool.QueryRow(context.Background(), `SELECT e.id FROM atendimento_equipes e JOIN atendimento_equipe_membros m ON m.equipe_id = e.id
		WHERE e.unidade_id = $1 AND m.user_id = $2 LIMIT 1`, c.unidade, c.tecnico.UserID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestModeloDeTarefasRegras(t *testing.T) {
	ctx := context.Background()
	c := novoComTarefas(t)
	if m, err := c.base.ModeloDaOferta(ctx, admin(), c.oferta.ID); err != nil || len(m) != 2 {
		t.Fatalf("ler: %+v %v", m, err)
	}
	daUnidade := gestor(c.unidade)
	alheia, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "Fora", UnidadeID: ptr(dbtest.Unidade(t, c.pool))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, alheia.ID)
	})
	um := c.modelo[:1]
	casos := map[string]struct{ err, espera error }{}
	reg := func(n string, err, espera error) { casos[n] = struct{ err, espera error }{err, espera} }
	_, err = c.base.SalvarModelo(ctx, admin(), c.cenario.oferta.ID, um)
	reg("incidente não tem tarefa", err, domain.ErrTarefasSoRequisicao)
	_, err = c.base.SalvarModelo(ctx, admin(), c.oferta.ID, []domain.ModeloTarefa{{Titulo: "x", EquipeID: uuid.New()}})
	reg("equipe inexistente", err, domain.ErrModeloInvalido)
	_, err = c.base.SalvarModelo(ctx, daUnidade, c.oferta.ID, []domain.ModeloTarefa{{Titulo: "x", EquipeID: alheia.ID}})
	reg("equipe fora do alcance", err, domain.ErrModeloInvalido)
	_, err = c.base.SalvarModelo(ctx, admin(), c.oferta.ID, []domain.ModeloTarefa{{Titulo: " ", EquipeID: c.rede.ID}})
	reg("sem título", err, domain.ErrModeloInvalido)
	_, err = c.base.SalvarModelo(ctx, gestor(uuid.New()), c.oferta.ID, um)
	reg("gestor de outra unidade salva", err, domain.ErrNaoEncontrado)
	_, err = c.base.ModeloDaOferta(ctx, gestor(uuid.New()), c.oferta.ID)
	reg("gestor de outra unidade lê", err, domain.ErrNaoEncontrado)
	_, err = c.base.ModeloDaOferta(ctx, admin(), uuid.New())
	reg("ler de oferta inexistente", err, domain.ErrNaoEncontrado)
	_, err = c.base.SalvarModelo(ctx, admin(), uuid.New(), um)
	reg("salvar em oferta inexistente", err, domain.ErrNaoEncontrado)
	_, err = c.com(falhando("ModeloDaOferta"), eventosFalham{}).ModeloDaOferta(ctx, admin(), c.oferta.ID)
	reg("falha ao ler o modelo", err, errFalha)
	_, err = c.com(falhando("SalvarModelo"), eventosFalham{}).SalvarModelo(ctx, admin(), c.oferta.ID, um)
	reg("falha ao gravar", err, errFalha)
	_, err = c.com(falhando(), eventosFalham{application.EventoOfertaTarefas}).SalvarModelo(ctx, admin(), c.oferta.ID, um)
	reg("falha no outbox", err, errFalha)
	for nome, k := range casos {
		if !errors.Is(k.err, k.espera) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}
	if m, err := c.base.SalvarModelo(ctx, daUnidade, c.oferta.ID, um); err != nil || len(m) != 1 {
		t.Fatalf("gestor da unidade salva com a equipe dela: %v %v", m, err)
	}
	if vazio, err := c.base.SalvarModelo(ctx, admin(), c.oferta.ID, nil); err != nil || len(vazio) != 0 {
		t.Fatalf("tirar as tarefas: %v %v", vazio, err)
	}
}

func ptr[T any](v T) *T { return &v }

// comParam acrescenta outro parâmetro de rota ao pedido de teste.
func comParam(r *http.Request, chave, valor string) {
	chi.RouteContext(r.Context()).URLParams.Add(chave, valor)
}

func TestTarefasCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoComTarefas(t)
	com := func(m ...string) *application.Service { return c.com(falhando(m...), eventosFalham{}) }
	abertura := application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "x"}
	for _, m := range []string{"ModeloDaOferta", "InsertTarefas", "GetEquipe"} {
		if _, err := com(m).Abrir(ctx, c.servidor, abertura); !errors.Is(err, errFalha) {
			t.Errorf("abrir, falha em %s: %v", m, err)
		}
	}
	sol := c.pedir(t)
	d, _ := c.base.Obter(ctx, c.tecnico, sol.ID)
	minha := d.Tarefas[0].ID
	for nome, svc := range map[string]*application.Service{
		"ler a tarefa":        com("GetTarefa"),
		"conferir a equipe":   com("MembroDe"),
		"gravar a tarefa":     com("UpdateTarefa"),
		"evento":              c.com(falhando(), eventosFalham{application.EventoTarefaFeita}),
		"contar as pendentes": com("ListTarefas"),
	} {
		if _, err := svc.FazerTarefa(ctx, c.tecnico, sol.ID, minha, true, ""); !errors.Is(err, errFalha) {
			t.Errorf("fazer, falha ao %s: %v", nome, err)
		}
	}
	for _, m := range []string{"ListTarefas", "MembroDe"} {
		if _, err := com(m).Obter(ctx, c.tecnico, sol.ID); !errors.Is(err, errFalha) {
			t.Errorf("detalhe, falha em %s: %v", m, err)
		}
	}
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := com("ListTarefas").Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoAtendido); !errors.Is(err, errFalha) {
		t.Fatalf("conferir as pendentes: %v", err)
	}

	// A mudança não conclui o pedido com tarefa pendente.
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.base.ConcluirPorMudanca(ctx, tx, sol.ID, c.tecnico.UserID, 9, "Fonte", "")
	_ = tx.Rollback(ctx)
	var n *domain.NaoConcluivel
	if !errors.As(err, &n) || !errors.Is(n.Causa, domain.ErrTarefasPendentes) {
		t.Fatalf("mudança com pendência: %v", err)
	}

	// A equipe apagada deixa a tarefa sem dona: ninguém conclui, quem atende dispensa.
	if _, err := c.pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE id = $1`, c.rede.ID); err != nil {
		t.Fatal(err)
	}
	d, _ = c.base.Obter(ctx, c.tecnico, sol.ID)
	if d.Tarefas[1].EquipeID != nil || d.Tarefas[1].EquipeNome != "Rede" {
		t.Fatalf("tarefa sem dona: %+v", d.Tarefas[1])
	}
	if _, err := c.base.FazerTarefa(ctx, c.tecnico, sol.ID, d.Tarefas[1].ID, true, ""); !errors.Is(err, domain.ErrNaoEhDaEquipe) {
		t.Fatalf("sem dona, não se conclui: %v", err)
	}
}

func TestHandlersDasTarefas(t *testing.T) {
	c := novoComTarefas(t)
	sol := c.pedir(t)
	d, _ := c.base.Obter(context.Background(), c.tecnico, sol.ID)
	minha := d.Tarefas[0].ID.String()
	h := transport.NewHandlers(c.base, silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/x", id, corpo, quem))
		return rec
	}
	tarefa := func(sol, tarefa string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r := pedido(t, http.MethodPost, "/x", sol, corpo, quem)
		comParam(r, "tarefaId", tarefa)
		h.FazerTarefa(rec, r)
		return rec
	}
	modelo := map[string]any{"tarefas": []map[string]any{{"titulo": "Criar conta", "equipe_id": c.suporte.ID}}}
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"ler o modelo":            {chama(h.ModeloDaOferta, http.MethodGet, c.oferta.ID.String(), nil, admin()), http.StatusOK},
		"ler, id inválido":        {chama(h.ModeloDaOferta, http.MethodGet, "x", nil, admin()), http.StatusBadRequest},
		"ler o modelo alheio":     {chama(h.ModeloDaOferta, http.MethodGet, c.oferta.ID.String(), nil, gestor(uuid.New())), http.StatusNotFound},
		"salvar o modelo":         {chama(h.SalvarModelo, http.MethodPut, c.oferta.ID.String(), modelo, admin()), http.StatusOK},
		"salvar, id inválido":     {chama(h.SalvarModelo, http.MethodPut, "x", modelo, admin()), http.StatusBadRequest},
		"salvar sem equipe":       {chama(h.SalvarModelo, http.MethodPut, c.oferta.ID.String(), map[string]any{"tarefas": []map[string]any{{"titulo": "x"}}}, admin()), http.StatusUnprocessableEntity},
		"salvar em incidente":     {chama(h.SalvarModelo, http.MethodPut, c.cenario.oferta.ID.String(), modelo, admin()), http.StatusUnprocessableEntity},
		"fazer, pedido inválido":  {tarefa("x", minha, map[string]any{"concluir": true}, c.tecnico), http.StatusBadRequest},
		"fazer, tarefa inválida":  {tarefa(sol.ID.String(), "x", map[string]any{"concluir": true}, c.tecnico), http.StatusBadRequest},
		"fazer sem o campo":       {tarefa(sol.ID.String(), minha, map[string]any{}, c.tecnico), http.StatusUnprocessableEntity},
		"fazer a de outra equipe": {tarefa(sol.ID.String(), d.Tarefas[1].ID.String(), map[string]any{"concluir": true}, c.tecnico), http.StatusForbidden},
		"concluir a minha":        {tarefa(sol.ID.String(), minha, map[string]any{"concluir": true}, c.tecnico), http.StatusOK},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
}
