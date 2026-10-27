package application_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func (f falhaRepo) EtapasDaOferta(ctx context.Context, db database.DBTX, o uuid.UUID) ([]domain.EtapaAprovacao, error) {
	if f.falha["EtapasDaOferta"] {
		return nil, errFalha
	}
	return f.Repository.EtapasDaOferta(ctx, db, o)
}
func (f falhaRepo) SalvarEtapas(ctx context.Context, db database.DBTX, o uuid.UUID, e []domain.EtapaAprovacao) error {
	if f.falha["SalvarEtapas"] {
		return errFalha
	}
	return f.Repository.SalvarEtapas(ctx, db, o, e)
}
func (f falhaRepo) InsertAprovacao(ctx context.Context, db database.DBTX, a domain.Aprovacao) error {
	if f.falha["InsertAprovacao"] {
		return errFalha
	}
	return f.Repository.InsertAprovacao(ctx, db, a)
}
func (f falhaRepo) ListAprovacoes(ctx context.Context, db database.DBTX, s uuid.UUID) ([]domain.Aprovacao, error) {
	if f.falha["ListAprovacoes"] {
		return nil, errFalha
	}
	return f.Repository.ListAprovacoes(ctx, db, s)
}
func (f falhaRepo) AguardandoAprovacaoDe(ctx context.Context, db database.DBTX, a uuid.UUID, n int) ([]domain.Solicitacao, error) {
	if f.falha["AguardandoAprovacaoDe"] {
		return nil, errFalha
	}
	return f.Repository.AguardandoAprovacaoDe(ctx, db, a, n)
}
func (f falhaRepo) ContarAtivos(ctx context.Context, db database.DBTX, ids []uuid.UUID) (int, error) {
	if f.falha["ContarAtivos"] {
		return 0, errFalha
	}
	return f.Repository.ContarAtivos(ctx, db, ids)
}

// comAprovacao monta uma oferta de requisição com duas etapas (chefia e TI).
type comAprovacao struct {
	cenario
	oferta    domain.Oferta
	chefe, ti auth.Identity
}

func novoComAprovacao(t *testing.T) comAprovacao {
	t.Helper()
	ctx := context.Background()
	c := novoCenario(t)
	o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "Admissão " + uuid.NewString()[:6], Tipo: domain.TipoRequisicao, UnidadeID: &c.unidade})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})
	chefe, ti := auth.Identity{UserID: dbtest.User(t, c.pool)}, auth.Identity{UserID: dbtest.User(t, c.pool)}
	if _, err := c.base.SalvarEtapas(ctx, admin(), o.ID, []domain.EtapaAprovacao{
		{Nome: "Chefia", Aprovadores: []uuid.UUID{chefe.UserID}},
		{Nome: "TI", Aprovadores: []uuid.UUID{ti.UserID}},
	}); err != nil {
		t.Fatal(err)
	}
	return comAprovacao{cenario: c, oferta: o, chefe: chefe, ti: ti}
}

func (c comAprovacao) pedir(t *testing.T) domain.Solicitacao {
	t.Helper()
	s, err := c.base.Abrir(context.Background(), c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Admitir Ana"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAprovacaoEmDuasEtapasAteAFila(t *testing.T) {
	ctx := context.Background()
	c := novoComAprovacao(t)
	sol := c.pedir(t)
	if sol.Estado != domain.EstadoAguardandoAprovacao || *sol.EtapaAtual != 0 {
		t.Fatalf("abre aguardando: %+v", sol)
	}
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); !errors.Is(err, domain.ErrAguardandoAprovacao) {
		t.Fatalf("ninguém assume antes da aprovação: %v", err)
	}
	d, err := c.base.Obter(ctx, c.tecnico, sol.ID)
	if err != nil || d.EtapaAtual != "Chefia" || len(d.Proximos) != 1 || d.Proximos[0] != domain.EstadoCancelado {
		t.Fatalf("detalhe aguardando: %+v %v", d, err)
	}
	if lista, err := c.base.AguardandoMinhaAprovacao(ctx, c.chefe); err != nil || len(lista) != 1 {
		t.Fatalf("a chefia vê: %+v %v", lista, err)
	}
	if _, err := c.base.DecidirAprovacao(ctx, c.ti, sol.ID, true, ""); !errors.Is(err, domain.ErrNaoAprovador) {
		t.Fatalf("a TI ainda não decide: %v", err)
	}

	s1, err := c.base.DecidirAprovacao(ctx, c.chefe, sol.ID, true, "")
	if err != nil || *s1.EtapaAtual != 1 {
		t.Fatalf("chefia aprova: %+v %v", s1, err)
	}
	fim, err := c.base.DecidirAprovacao(ctx, c.ti, sol.ID, true, "ok")
	if err != nil || fim.Estado != domain.EstadoAberto || fim.EtapaAtual != nil {
		t.Fatalf("TI aprova e libera: %+v %v", fim, err)
	}
	d, _ = c.base.Obter(ctx, c.tecnico, sol.ID)
	if len(d.Aprovacoes) != 2 || d.EtapaAtual != "" || d.Passagens[len(d.Passagens)-1].Para != domain.EstadoAberto {
		t.Fatalf("detalhe liberado: %+v", d)
	}
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatalf("liberado, a equipe assume: %v", err)
	}
}

func TestRecusaDevolveOMotivo(t *testing.T) {
	ctx := context.Background()
	c := novoComAprovacao(t)
	sol := c.pedir(t)
	if _, err := c.base.DecidirAprovacao(ctx, c.chefe, sol.ID, false, " "); !errors.Is(err, domain.ErrMotivoObrigatorio) {
		t.Fatalf("recusa sem motivo: %v", err)
	}
	r, err := c.base.DecidirAprovacao(ctx, c.chefe, sol.ID, false, "Sem vaga aprovada")
	if err != nil || r.Estado != domain.EstadoCancelado {
		t.Fatalf("recusa: %+v %v", r, err)
	}
	d, _ := c.base.Obter(ctx, c.servidor, sol.ID)
	if len(d.Comentarios) != 1 || !strings.Contains(d.Comentarios[0].Texto, "Sem vaga aprovada") || d.Comentarios[0].Interno {
		t.Fatalf("quem pediu lê o motivo: %+v", d.Comentarios)
	}
}

func TestEtapasDaOfertaRegras(t *testing.T) {
	ctx := context.Background()
	c := novoComAprovacao(t)
	if etapas, err := c.base.EtapasDaOferta(ctx, admin(), c.oferta.ID); err != nil || len(etapas) != 2 {
		t.Fatalf("ler: %+v %v", etapas, err)
	}
	inativo := dbtest.User(t, c.pool)
	if _, err := c.pool.Exec(ctx, `UPDATE users SET active = false WHERE id = $1`, inativo); err != nil {
		t.Fatal(err)
	}
	outraUnidade := gestor(uuid.New())
	umaEtapa := []domain.EtapaAprovacao{{Nome: "x", Aprovadores: []uuid.UUID{c.chefe.UserID}}}
	casos := map[string]struct {
		err    error
		espera error
	}{}
	reg := func(n string, err, espera error) { casos[n] = struct{ err, espera error }{err, espera} }
	_, err := c.base.SalvarEtapas(ctx, admin(), c.cenario.oferta.ID, umaEtapa)
	reg("incidente não tem etapa", err, domain.ErrEtapasSoRequisicao)
	_, err = c.base.SalvarEtapas(ctx, admin(), c.oferta.ID, []domain.EtapaAprovacao{{Nome: "x", Aprovadores: []uuid.UUID{inativo}}})
	reg("aprovador inativo", err, domain.ErrEtapaInvalida)
	_, err = c.base.SalvarEtapas(ctx, admin(), c.oferta.ID, []domain.EtapaAprovacao{{Nome: " "}})
	reg("etapa sem nome", err, domain.ErrEtapaInvalida)
	_, err = c.base.SalvarEtapas(ctx, outraUnidade, c.oferta.ID, umaEtapa)
	reg("gestor de outra unidade salva", err, domain.ErrNaoEncontrado)
	_, err = c.base.EtapasDaOferta(ctx, outraUnidade, c.oferta.ID)
	reg("gestor de outra unidade lê", err, domain.ErrNaoEncontrado)
	_, err = c.base.EtapasDaOferta(ctx, admin(), uuid.New())
	reg("oferta inexistente", err, domain.ErrNaoEncontrado)
	_, err = c.base.SalvarEtapas(ctx, admin(), uuid.New(), umaEtapa)
	reg("salvar em oferta inexistente", err, domain.ErrNaoEncontrado)
	_, err = c.com(falhando("EtapasDaOferta"), eventosFalham{}).EtapasDaOferta(ctx, admin(), c.oferta.ID)
	reg("falha ao ler as etapas", err, errFalha)
	_, err = c.com(falhando("ContarAtivos"), eventosFalham{}).SalvarEtapas(ctx, admin(), c.oferta.ID, umaEtapa)
	reg("falha ao conferir os aprovadores", err, errFalha)
	_, err = c.com(falhando("SalvarEtapas"), eventosFalham{}).SalvarEtapas(ctx, admin(), c.oferta.ID, umaEtapa)
	reg("falha ao gravar", err, errFalha)
	_, err = c.com(falhando(), eventosFalham{application.EventoOfertaEtapas}).SalvarEtapas(ctx, admin(), c.oferta.ID, umaEtapa)
	reg("falha no outbox", err, errFalha)
	_, err = c.com(falhando("AguardandoAprovacaoDe"), eventosFalham{}).AguardandoMinhaAprovacao(ctx, c.chefe)
	reg("falha na lista de pendências", err, errFalha)
	for nome, k := range casos {
		if !errors.Is(k.err, k.espera) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}
	if vazio, err := c.base.SalvarEtapas(ctx, admin(), c.oferta.ID, nil); err != nil || len(vazio) != 0 {
		t.Fatalf("tirar a aprovação: %v %v", vazio, err)
	}
}

func TestAprovacaoCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoComAprovacao(t)
	com := func(m ...string) *application.Service { return c.com(falhando(m...), eventosFalham{}) }
	evento := func(nome string) *application.Service { return c.com(falhando(), eventosFalham{nome}) }

	if _, err := com("EtapasDaOferta").Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "x"}); !errors.Is(err, errFalha) {
		t.Fatalf("abrir, falha nas etapas: %v", err)
	}
	sol := c.pedir(t)
	for nome, k := range map[string]struct {
		svc     *application.Service
		quem    auth.Identity
		aprovar bool
		motivo  string
	}{
		"falha ao travar":             {com("TravarSolicitacao"), c.chefe, true, ""},
		"falha ao ler":                {com("GetSolicitacao"), c.chefe, true, ""},
		"falha nas etapas":            {com("EtapasDaOferta"), c.chefe, true, ""},
		"falha ao gravar a decisão":   {com("InsertAprovacao"), c.chefe, true, ""},
		"falha no evento":             {evento(application.EventoAprovacaoDecidida), c.chefe, true, ""},
		"falha ao avançar a etapa":    {com("UpdateSolicitacao"), c.chefe, true, ""},
		"falha na resposta da recusa": {com("InsertComentario"), c.chefe, false, "não"},
		"falha ao cancelar":           {com("InsertPassagem"), c.chefe, false, "não"},
	} {
		if _, err := k.svc.DecidirAprovacao(ctx, k.quem, sol.ID, k.aprovar, k.motivo); !errors.Is(err, errFalha) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := c.base.DecidirAprovacao(ctx, c.chefe, uuid.New(), true, ""); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("pedido inexistente: %v", err)
	}

	// Na última etapa: falha ao liberar e ao avisar a equipe.
	if _, err := c.base.DecidirAprovacao(ctx, c.chefe, sol.ID, true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := com("InsertPassagem").DecidirAprovacao(ctx, c.ti, sol.ID, true, ""); !errors.Is(err, errFalha) {
		t.Fatalf("falha ao liberar: %v", err)
	}
	if _, err := com("Responsaveis").DecidirAprovacao(ctx, c.ti, sol.ID, true, ""); !errors.Is(err, errFalha) {
		t.Fatalf("falha ao achar a equipe: %v", err)
	}

	// O detalhe propaga as falhas novas.
	if _, err := com("ListAprovacoes").Obter(ctx, c.tecnico, sol.ID); !errors.Is(err, errFalha) {
		t.Fatalf("detalhe, falha nas decisões: %v", err)
	}
	if _, err := com("EtapasDaOferta").Obter(ctx, c.tecnico, sol.ID); !errors.Is(err, errFalha) {
		t.Fatalf("detalhe, falha nas etapas: %v", err)
	}
	// Etapas tiradas da oferta: o pedido que esperava mostra sem nome.
	if _, err := c.base.SalvarEtapas(ctx, admin(), c.oferta.ID, nil); err != nil {
		t.Fatal(err)
	}
	if d, err := c.base.Obter(ctx, c.tecnico, sol.ID); err != nil || d.EtapaAtual != "" {
		t.Fatalf("etapa que deixou de existir: %+v %v", d, err)
	}
}

func TestHandlersDaAprovacao(t *testing.T) {
	c := novoComAprovacao(t)
	sol := c.pedir(t)
	h := transport.NewHandlers(c.base, silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/x", id, corpo, quem))
		return rec
	}
	etapas := map[string]any{"etapas": []map[string]any{{"nome": "Chefia", "aprovadores": []uuid.UUID{c.chefe.UserID}}, {"nome": "TI", "aprovadores": []uuid.UUID{c.ti.UserID}}}}
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"ler etapas":                {chama(h.EtapasDaOferta, http.MethodGet, c.oferta.ID.String(), nil, admin()), http.StatusOK},
		"ler etapas, id inválido":   {chama(h.EtapasDaOferta, http.MethodGet, "x", nil, admin()), http.StatusBadRequest},
		"ler etapas alheias":        {chama(h.EtapasDaOferta, http.MethodGet, c.oferta.ID.String(), nil, gestor(uuid.New())), http.StatusNotFound},
		"salvar etapas":             {chama(h.SalvarEtapas, http.MethodPut, c.oferta.ID.String(), etapas, admin()), http.StatusOK},
		"salvar, id inválido":       {chama(h.SalvarEtapas, http.MethodPut, "x", etapas, admin()), http.StatusBadRequest},
		"salvar sem aprovador":      {chama(h.SalvarEtapas, http.MethodPut, c.oferta.ID.String(), map[string]any{"etapas": []map[string]any{{"nome": "x"}}}, admin()), http.StatusUnprocessableEntity},
		"salvar em incidente":       {chama(h.SalvarEtapas, http.MethodPut, c.cenario.oferta.ID.String(), etapas, admin()), http.StatusUnprocessableEntity},
		"pendências da chefia":      {chama(h.AguardandoMinhaAprovacao, http.MethodGet, "", nil, c.chefe), http.StatusOK},
		"decidir, id inválido":      {chama(h.DecidirAprovacao, http.MethodPost, "x", map[string]any{"aprovar": true}, c.chefe), http.StatusBadRequest},
		"decidir sem o campo":       {chama(h.DecidirAprovacao, http.MethodPost, sol.ID.String(), map[string]any{}, c.chefe), http.StatusUnprocessableEntity},
		"decidir sem ser aprovador": {chama(h.DecidirAprovacao, http.MethodPost, sol.ID.String(), map[string]any{"aprovar": true}, c.ti), http.StatusForbidden},
		"chefia aprova":             {chama(h.DecidirAprovacao, http.MethodPost, sol.ID.String(), map[string]any{"aprovar": true}, c.chefe), http.StatusOK},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
	quebrado := transport.NewHandlers(c.com(falhando("AguardandoAprovacaoDe"), eventosFalham{}), silencio)
	if rec := chama(quebrado.AguardandoMinhaAprovacao, http.MethodGet, "", nil, c.chefe); rec.Code != http.StatusInternalServerError {
		t.Fatalf("pendências com o banco fora: %d", rec.Code)
	}
}
