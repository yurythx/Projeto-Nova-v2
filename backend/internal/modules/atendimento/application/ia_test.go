package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// iaFalsa responde o que o teste mandar e guarda o que recebeu.
type iaFalsa struct {
	disponivel bool
	errDisp    error
	texto      string
	err        error
	sistema    string
	pergunta   string
}

func (f *iaFalsa) Disponivel(context.Context) (bool, error) { return f.disponivel, f.errDisp }

func (f *iaFalsa) Perguntar(_ context.Context, sistema, pergunta string, _ int) (string, string, error) {
	f.sistema, f.pergunta = sistema, pergunta
	return f.texto, "IA local", f.err
}

func (f falhaRepo) InsertSugestao(ctx context.Context, db database.DBTX, s domain.Sugestao) error {
	if f.falha["InsertSugestao"] {
		return errFalha
	}
	return f.Repository.InsertSugestao(ctx, db, s)
}
func (f falhaRepo) TravarSugestao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Sugestao, error) {
	if f.falha["TravarSugestao"] {
		return domain.Sugestao{}, errFalha
	}
	return f.Repository.TravarSugestao(ctx, db, id)
}
func (f falhaRepo) DecidirSugestao(ctx context.Context, db database.DBTX, s domain.Sugestao) error {
	if f.falha["DecidirSugestao"] {
		return errFalha
	}
	return f.Repository.DecidirSugestao(ctx, db, s)
}
func (f falhaRepo) IndicadoresIA(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]domain.IndicadorIA, error) {
	if f.falha["IndicadoresIA"] {
		return nil, errFalha
	}
	return f.Repository.IndicadoresIA(ctx, db, desde, restrito, unidades)
}

const classeOK = `{"impacto": 3, "urgencia": 3, "motivo": "o protocolo da prefeitura parou"}`

func TestEstadoDaIA(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	if e, err := c.base.EstadoDaIA(ctx, c.tecnico); err != nil || e.Disponivel {
		t.Fatalf("sem IA ligada: %+v %v", e, err)
	}
	ia := &iaFalsa{disponivel: true}
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(ia)
	if e, _ := svc.EstadoDaIA(ctx, c.servidor); e.Disponivel {
		t.Fatal("quem não atende não fica sabendo da IA")
	}
	if e, err := svc.EstadoDaIA(ctx, c.tecnico); err != nil || !e.Disponivel {
		t.Fatalf("quem atende: %+v %v", e, err)
	}
	ia.errDisp = errFalha
	if _, err := svc.EstadoDaIA(ctx, c.tecnico); err == nil {
		t.Fatal("falha ao ler a configuração")
	}
}

func TestSugerirClassificacao(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	ia := &iaFalsa{texto: "Segue: " + classeOK}
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(ia)

	s, err := svc.SugerirClassificacao(ctx, c.tecnico, sol.ID)
	if err != nil || s.Tipo != domain.SugestaoClassificacao || s.Classificacao.Impacto != 3 || s.Conexao != "IA local" || *s.PedidaPor != c.tecnico.UserID {
		t.Fatalf("sugestão: %+v %v", s, err)
	}
	if !strings.Contains(ia.pergunta, "Título: Pedido") || !strings.Contains(ia.pergunta, "Oferta: "+c.oferta.Nome) || !strings.Contains(ia.sistema, "JSON") {
		t.Fatalf("pergunta: %q", ia.pergunta)
	}
	// A IA só sugere: o chamado continua como estava.
	depois, _ := c.base.Obter(ctx, c.tecnico, sol.ID)
	if depois.Solicitacao.Impacto != sol.Impacto || depois.Solicitacao.Urgencia != sol.Urgencia {
		t.Fatalf("a IA mudou o chamado: %+v", depois.Solicitacao)
	}

	// Sem quem pediu (conta de serviço): a sugestão fica sem autor.
	semAutor := c.tecnico
	semAutor.UserID = uuid.Nil
	if s, err := svc.SugerirClassificacao(ctx, semAutor, sol.ID); err != nil || s.PedidaPor != nil {
		t.Fatalf("sem autor: %+v %v", s, err)
	}

	casos := []struct {
		nome   string
		svc    *application.Service
		quem   auth.Identity
		id     uuid.UUID
		espera error
	}{
		{"sem IA ligada", c.base, c.tecnico, sol.ID, domain.ErrSemIA},
		{"chamado inexistente", svc, c.tecnico, uuid.New(), domain.ErrNaoEncontrado},
		{"quem não vê o chamado", svc, auth.Identity{UserID: uuid.New()}, sol.ID, domain.ErrNaoEncontrado},
		{"quem abriu não pede à IA", svc, c.servidor, sol.ID, domain.ErrForaDoEscopo},
		{"falha ao ler o chamado", c.com(falhando("GetSolicitacao"), eventosFalham{}).WithAssistente(ia), c.tecnico, sol.ID, errFalha},
		{"falha ao gravar", c.com(falhando("InsertSugestao"), eventosFalham{}).WithAssistente(ia), c.tecnico, sol.ID, errFalha},
		{"IA fora do ar", c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{err: errFalha}), c.tecnico, sol.ID, domain.ErrRespostaDaIA},
		{"IA desligada na tela", c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{err: domain.ErrSemIA}), c.tecnico, sol.ID, domain.ErrSemIA},
		{"resposta fora do formato", c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{texto: `{"impacto": 9}`}), c.tecnico, sol.ID, domain.ErrRespostaDaIA},
	}
	for _, k := range casos {
		if _, err := k.svc.SugerirClassificacao(ctx, k.quem, k.id); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", k.nome, err)
		}
	}
}

func TestResumirLevaAConversaSemNomes(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	if _, err := c.base.Comentar(ctx, c.servidor, sol.ID, "Continua parado desde cedo.", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Comentar(ctx, c.tecnico, sol.ID, "Reiniciei o serviço.", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Comentar(ctx, c.tecnico, sol.ID, "Suspeito do certificado.", true); err != nil {
		t.Fatal(err)
	}
	ia := &iaFalsa{texto: "  O protocolo parou; reiniciado; falta ver o certificado.  "}
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(ia)

	s, err := svc.Resumir(ctx, c.tecnico, sol.ID)
	if err != nil || s.Tipo != domain.SugestaoResumo || s.Resumo != "O protocolo parou; reiniciado; falta ver o certificado." {
		t.Fatalf("resumo: %+v %v", s, err)
	}
	for _, trecho := range []string{"Quem abriu (", "Continua parado", "Resposta (", "Reiniciei", "Nota interna da equipe (", "certificado"} {
		if !strings.Contains(ia.pergunta, trecho) {
			t.Errorf("a pergunta não leva %q: %q", trecho, ia.pergunta)
		}
	}
	if strings.Contains(ia.pergunta, "Usuário u_") {
		t.Errorf("a pergunta leva nome de pessoa: %q", ia.pergunta)
	}

	casos := []struct {
		nome   string
		svc    *application.Service
		quem   auth.Identity
		espera error
	}{
		{"quem abriu não pede", svc, c.servidor, domain.ErrForaDoEscopo},
		{"falha ao ler a conversa", c.com(falhando("ListComentarios"), eventosFalham{}).WithAssistente(ia), c.tecnico, errFalha},
		{"IA fora do ar", c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{err: errFalha}), c.tecnico, domain.ErrRespostaDaIA},
		{"resposta em branco", c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{texto: "  "}), c.tecnico, domain.ErrRespostaDaIA},
	}
	for _, k := range casos {
		if _, err := k.svc.Resumir(ctx, k.quem, sol.ID); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", k.nome, err)
		}
	}
}

func TestDecidirSugestao(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{texto: classeOK})
	nova := func() domain.Sugestao {
		s, err := svc.SugerirClassificacao(ctx, c.tecnico, sol.ID)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	s := nova()
	d, err := svc.DecidirSugestao(ctx, c.tecnico, s.ID, true)
	if err != nil || *d.Decisao != domain.DecisaoAceita || *d.DecididaPor != c.tecnico.UserID {
		t.Fatalf("aceitar: %+v %v", d, err)
	}
	if _, err := svc.DecidirSugestao(ctx, c.tecnico, s.ID, false); !errors.Is(err, domain.ErrSugestaoDecidida) {
		t.Fatalf("segunda decisão: %v", err)
	}
	outra := nova()
	if d, err := svc.DecidirSugestao(ctx, c.tecnico, outra.ID, false); err != nil || *d.Decisao != domain.DecisaoRecusada {
		t.Fatalf("recusar: %+v %v", d, err)
	}

	pendente := nova()
	casos := []struct {
		nome   string
		svc    *application.Service
		quem   auth.Identity
		id     uuid.UUID
		espera error
	}{
		{"sugestão inexistente", svc, c.tecnico, uuid.New(), domain.ErrNaoEncontrado},
		{"quem abriu não decide", svc, c.servidor, pendente.ID, domain.ErrNaoEncontrado},
		{"outra unidade não decide", svc, atendente(uuid.New()), pendente.ID, domain.ErrNaoEncontrado},
		{"falha ao travar", c.com(falhando("TravarSugestao"), eventosFalham{}), c.tecnico, pendente.ID, errFalha},
		{"falha ao ler o chamado", c.com(falhando("GetSolicitacao"), eventosFalham{}), c.tecnico, pendente.ID, errFalha},
		{"falha ao gravar", c.com(falhando("DecidirSugestao"), eventosFalham{}), c.tecnico, pendente.ID, errFalha},
	}
	for _, k := range casos {
		if _, err := k.svc.DecidirSugestao(ctx, k.quem, k.id, true); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", k.nome, err)
		}
	}
}

func TestIndicadoresIA(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{texto: classeOK})
	s, _ := svc.SugerirClassificacao(ctx, c.tecnico, sol.ID)
	if _, err := svc.DecidirSugestao(ctx, c.tecnico, s.ID, true); err != nil {
		t.Fatal(err)
	}
	ind, err := svc.IndicadoresIA(ctx, c.tecnico)
	if err != nil || len(ind) != 2 || ind[0].Tipo != domain.SugestaoClassificacao || ind[0].Total != 1 || ind[0].Aceitas != 1 {
		t.Fatalf("indicadores: %+v %v", ind, err)
	}
	if _, err := svc.IndicadoresIA(ctx, c.servidor); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("quem não atende: %v", err)
	}
	if _, err := c.com(falhando("IndicadoresIA"), eventosFalham{}).IndicadoresIA(ctx, c.tecnico); !errors.Is(err, errFalha) {
		t.Fatalf("falha do banco: %v", err)
	}
}

func TestHandlersDaIA(t *testing.T) {
	c := novoCenario(t)
	sol := c.abrir(t)
	ia := &iaFalsa{disponivel: true, texto: classeOK}
	h := transport.NewHandlers(c.com(falhando(), eventosFalham{}).WithAssistente(ia), silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/x", id, corpo, quem))
		return rec
	}

	if rec := chama(h.EstadoDaIA, http.MethodGet, "", nil, c.tecnico); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"disponivel":true`) {
		t.Fatalf("estado: %d %s", rec.Code, rec.Body.String())
	}
	rec := chama(h.SugerirClassificacao, http.MethodPost, sol.ID.String(), nil, c.tecnico)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"impacto":3`) {
		t.Fatalf("classificação: %d %s", rec.Code, rec.Body.String())
	}
	var criada struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &criada); err != nil || criada.Data.ID == "" {
		t.Fatalf("id da sugestão: %s %v", rec.Body.String(), err)
	}
	ia.texto = "Resumo curto."
	if rec := chama(h.Resumir, http.MethodPost, sol.ID.String(), nil, c.tecnico); rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), "Resumo curto.") {
		t.Fatalf("resumo: %d %s", rec.Code, rec.Body.String())
	}
	if rec := chama(h.DecidirSugestao, http.MethodPost, criada.Data.ID, map[string]any{"aceita": true}, c.tecnico); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"decisao":"aceita"`) {
		t.Fatalf("decisão: %d %s", rec.Code, rec.Body.String())
	}
	if rec := chama(h.DecidirSugestao, http.MethodPost, criada.Data.ID, map[string]any{"aceita": false}, c.tecnico); rec.Code != http.StatusConflict {
		t.Fatalf("segunda decisão: %d", rec.Code)
	}
	if rec := chama(h.IndicadoresIA, http.MethodGet, "", nil, c.tecnico); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"aceitas":1`) {
		t.Fatalf("indicadores: %d %s", rec.Code, rec.Body.String())
	}

	// Entradas inválidas e falhas chegam a quem chamou com o status certo.
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"classificação com id inválido": {chama(h.SugerirClassificacao, http.MethodPost, "x", nil, c.tecnico), http.StatusBadRequest},
		"resumo com id inválido":        {chama(h.Resumir, http.MethodPost, "x", nil, c.tecnico), http.StatusBadRequest},
		"decisão com id inválido":       {chama(h.DecidirSugestao, http.MethodPost, "x", map[string]any{"aceita": true}, c.tecnico), http.StatusBadRequest},
		"decisão sem o campo":           {chama(h.DecidirSugestao, http.MethodPost, criada.Data.ID, map[string]any{}, c.tecnico), http.StatusUnprocessableEntity},
		"quem abriu pede resumo":        {chama(h.Resumir, http.MethodPost, sol.ID.String(), nil, c.servidor), http.StatusForbidden},
		"quem abriu pede classificação": {chama(h.SugerirClassificacao, http.MethodPost, sol.ID.String(), nil, c.servidor), http.StatusForbidden},
		"indicadores fora do escopo":    {chama(h.IndicadoresIA, http.MethodGet, "", nil, c.servidor), http.StatusForbidden},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
	ia.errDisp = errFalha
	if rec := chama(h.EstadoDaIA, http.MethodGet, "", nil, c.tecnico); rec.Code != http.StatusInternalServerError {
		t.Fatalf("estado com falha: %d", rec.Code)
	}
	ia.err = errFalha
	if rec := chama(h.SugerirClassificacao, http.MethodPost, sol.ID.String(), nil, c.tecnico); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("IA fora do ar: %d", rec.Code)
	}
	sem := transport.NewHandlers(c.base, silencio)
	if rec := chama(sem.Resumir, http.MethodPost, sol.ID.String(), nil, c.tecnico); rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "não está configurada") {
		t.Fatalf("sem IA: %d %s", rec.Code, rec.Body.String())
	}
}
