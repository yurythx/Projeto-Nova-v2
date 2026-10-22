package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
)

type solicitacaoResp struct {
	ID            string  `json:"id"`
	Numero        int64   `json:"numero"`
	Tipo          string  `json:"tipo"`
	Estado        string  `json:"estado"`
	UnidadeID     *string `json:"unidade_id"`
	ResponsavelID *string `json:"responsavel_id"`
	OfertaNome    string  `json:"oferta_nome"`
}

type detalheResp struct {
	Solicitacao solicitacaoResp `json:"solicitacao"`
	Papeis      []string        `json:"papeis"`
	Proximos    []string        `json:"proximos"`
	Passagens   []struct {
		Para string `json:"para"`
	} `json:"passagens"`
	Comentarios []struct {
		Texto   string `json:"texto"`
		Interno bool   `json:"interno"`
	} `json:"comentarios"`
}

// pessoaEm cria um usuário lotado na unidade com um perfil das permissões
// dadas; devolve o id e o token.
func (h *apiHarness) pessoaEm(t *testing.T, unidade string, permissoes ...string) (uuid.UUID, string) {
	t.Helper()
	admin := h.admin()
	perms, _ := json.Marshal(permissoes)
	perfil := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/iam/perfis", admin,
		`{"nome":"Perfil `+uuid.NewString()[:8]+`","permissoes":`+string(perms)+`}`)).ID
	id, tok := h.user("nova-user")
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/users/"+id.String()+"/lotacoes", admin,
		`{"perfil_id":"`+perfil+`","unidade_id":"`+unidade+`"}`)
	return id, tok
}

func avisosDe(t *testing.T, h *apiHarness, user uuid.UUID, solicitacao string) int {
	t.Helper()
	var n int
	if err := h.d.DB.QueryRow(context.Background(), `SELECT count(*) FROM notificacoes
		WHERE user_id = $1 AND modulo = 'atendimento' AND chave LIKE 'atendimento:' || $2 || ':%'`, user, solicitacao).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Fase 2: o servidor abre pela oferta, a equipe da unidade assume, conversa
// (com nota interna que o servidor não vê), pede informação e conclui; o
// servidor confirma. Cada passo deixa passagem, evento e auditoria.
func TestAtendimentoSolicitacao(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	o := h.orgEscopo(t)
	tecID, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	_, tecnicoB := h.pessoaEm(t, o.b, "atendimento:atender")
	servID, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	_, semPerm := h.user("nova-user")

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Não consigo acessar o e-mail","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	inativa := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Pedido antigo","unidade_id":"`+o.a+`","ativa":false}`))
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Tipo errado","tipo":"problema","unidade_id":"`+o.a+`"}`)
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Suporte A","unidade_id":"`+o.a+`","membros":["`+tecID.String()+`"]}`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/atendimento/equipes/"+uuid.NewString(), tecnico,
		`{"nome":"Ausente","unidade_id":"`+o.a+`"}`)

	catalogo := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/catalogo", servidor, "").Body.String()
	if !strings.Contains(catalogo, oferta.ID) || strings.Contains(catalogo, inativa.ID) {
		t.Fatalf("catálogo do servidor: %s", catalogo)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/catalogo", semPerm, "")
	h.expect(http.StatusForbidden, http.MethodPost, "/api/v1/atendimento/solicitacoes", semPerm,
		`{"oferta_id":"`+oferta.ID+`","titulo":"x"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+inativa.ID+`","titulo":"x"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+uuid.NewString()+`","titulo":"x"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":" "}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor, `{`)

	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Senha expirou","descricao":"desde hoje cedo"}`))
	if sol.Numero <= 0 || sol.Tipo != "incidente" || sol.Estado != "aberto" || sol.UnidadeID == nil || *sol.UnidadeID != o.a ||
		sol.OfertaNome != "Não consigo acessar o e-mail" {
		t.Fatalf("abertura: %+v", sol)
	}
	if outboxCount(t, h, "atendimento.solicitacao.aberta", sol.ID) != 1 {
		t.Fatal("evento de abertura")
	}
	if avisosDe(t, h, tecID, sol.ID) != 1 || avisosDe(t, h, servID, sol.ID) != 0 {
		t.Fatal("a equipe é avisada da abertura; quem abriu, não")
	}

	minhas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/minhas?situacao=abertas", servidor, "").Body.String()
	if !strings.Contains(minhas, sol.ID) {
		t.Fatalf("minhas: %s", minhas)
	}
	fila := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila?responsavel=nenhum", tecnico, "").Body.String()
	if !strings.Contains(fila, sol.ID) || !strings.Contains(fila, `"proximos":["em_atendimento","cancelado"]`) {
		t.Fatalf("fila de A, com as passagens do Kanban: %s", fila)
	}
	topicos := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila/topicos", tecnico, "").Body.String()
	if !strings.Contains(topicos, "atendimento:unidade:"+o.a) || strings.Contains(topicos, o.b) {
		t.Fatalf("tópicos da fila de A: %s", topicos)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/fila/topicos", servidor, "")
	if h.deliver("atendimento.solicitacao.aberta", sol.ID) != 1 {
		t.Fatal("o consumidor de tempo real recebe a abertura")
	}
	if outra := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila", tecnicoB, "").Body.String(); strings.Contains(outra, sol.ID) {
		t.Fatalf("a fila de B não vê A: %s", outra)
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/fila", servidor, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodGet, "/api/v1/atendimento/fila?responsavel=x", tecnico, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, tecnicoB, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, semPerm, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+uuid.NewString(), admin, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/solicitacoes/x", admin, "")

	visto := data[detalheResp](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, servidor, ""))
	if !slices.Equal(visto.Papeis, []string{"solicitante"}) || !slices.Equal(visto.Proximos, []string{"cancelado"}) {
		t.Fatalf("visão de quem abriu: %+v", visto)
	}

	estado := func(want int, tok, para string) detalheResp {
		t.Helper()
		rec := h.expect(want, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/estado", tok, `{"estado":"`+para+`"}`)
		if want != http.StatusOK {
			return detalheResp{}
		}
		return data[detalheResp](t, rec)
	}
	comentar := func(want int, tok, texto string, interno bool) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"texto": texto, "interno": interno})
		h.expect(want, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/comentarios", tok, string(body))
	}

	estado(http.StatusConflict, tecnico, "em_atendimento")
	estado(http.StatusConflict, servidor, "em_atendimento")
	estado(http.StatusNotFound, tecnicoB, "cancelado")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico,
		`{"responsavel_id":"`+servID.String()+`"}`)
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", servidor,
		`{"responsavel_id":"`+tecID.String()+`"}`)
	resp := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsaveis", tecnico, "").Body.String()
	if !strings.Contains(resp, tecID.String()) {
		t.Fatalf("responsáveis possíveis: %s", resp)
	}
	assumida := data[detalheResp](t, h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico,
		`{"responsavel_id":"`+tecID.String()+`"}`))
	if assumida.Solicitacao.Estado != "em_atendimento" || assumida.Solicitacao.ResponsavelID == nil || *assumida.Solicitacao.ResponsavelID != tecID.String() {
		t.Fatalf("assumir põe em atendimento: %+v", assumida.Solicitacao)
	}
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico,
		`{"responsavel_id":"`+tecID.String()+`"}`)
	if outboxCount(t, h, "atendimento.solicitacao.atribuida", sol.ID) != 1 {
		t.Fatal("atribuir de novo à mesma pessoa não emite evento")
	}

	comentar(http.StatusCreated, tecnico, "Conta bloqueada no AD", true)
	comentar(http.StatusForbidden, servidor, "nota", true)
	comentar(http.StatusUnprocessableEntity, servidor, " ", false)
	if avisosDe(t, h, servID, sol.ID) != 0 {
		t.Fatal("nota interna não avisa quem abriu")
	}
	doServidor := data[detalheResp](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, servidor, ""))
	if len(doServidor.Comentarios) != 0 {
		t.Fatalf("quem abriu não vê nota interna: %+v", doServidor.Comentarios)
	}
	daEquipe := data[detalheResp](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, tecnico, ""))
	if len(daEquipe.Comentarios) != 1 || !daEquipe.Comentarios[0].Interno {
		t.Fatalf("a equipe vê a nota: %+v", daEquipe.Comentarios)
	}

	estado(http.StatusOK, tecnico, "aguardando_servidor")
	estado(http.StatusOK, tecnico, "aguardando_servidor")
	comentar(http.StatusCreated, servidor, "Meu ramal é 1234", false)
	retomada := data[detalheResp](t, h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, tecnico, ""))
	if retomada.Solicitacao.Estado != "em_atendimento" {
		t.Fatalf("a resposta devolve à equipe: %s", retomada.Solicitacao.Estado)
	}
	estado(http.StatusConflict, tecnico, "atendido")
	resolvida := estado(http.StatusOK, tecnico, "resolvido")
	if !slices.Contains(resolvida.Proximos, "encerrado") {
		t.Fatalf("próximos da equipe: %v", resolvida.Proximos)
	}
	fim := estado(http.StatusOK, servidor, "encerrado")
	if len(fim.Proximos) != 0 {
		t.Fatalf("encerrada não tem próximos: %v", fim.Proximos)
	}
	comentar(http.StatusConflict, servidor, "obrigado", false)
	estado(http.StatusConflict, tecnico, "em_atendimento")

	var para []string
	for _, p := range fim.Passagens {
		para = append(para, p.Para)
	}
	if want := []string{"aberto", "em_atendimento", "aguardando_servidor", "em_atendimento", "resolvido", "encerrado"}; !slices.Equal(para, want) {
		t.Fatalf("passagens: %v", para)
	}
	if n := outboxCount(t, h, "atendimento.solicitacao.estado_alterado", sol.ID); n != 4 {
		t.Fatalf("eventos de estado: %d", n)
	}
	if h.deliver("atendimento.solicitacao.estado_alterado", sol.ID) != 4 || h.deliver("atendimento.solicitacao.comentada", sol.ID) != 2 {
		t.Fatal("o consumidor de tempo real recebe cada mudança")
	}
	var auditoria int
	if err := h.d.DB.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs
		WHERE resource_type = 'solicitacao' AND resource_id = $1`, sol.ID).Scan(&auditoria); err != nil {
		t.Fatal(err)
	}
	// abertura, atribuição, nota, aguardando, resposta, retomada, resolvido, encerrado
	if auditoria != 8 {
		t.Fatalf("auditoria: %d", auditoria)
	}
	if avisosDe(t, h, servID, sol.ID) == 0 {
		t.Fatal("quem abriu é avisado das mudanças")
	}

	encerradas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila", tecnico, "").Body.String()
	if strings.Contains(encerradas, sol.ID) {
		t.Fatal("a fila padrão não mostra encerradas")
	}
	todas := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila?situacao=todas&tipo=incidente", tecnico, "").Body.String()
	if !strings.Contains(todas, sol.ID) {
		t.Fatalf("situacao=todas: %s", todas)
	}
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes/x/estado", tecnico, `{"estado":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/estado", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes/x/comentarios", tecnico, `{"texto":"x"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/comentarios", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/solicitacoes/x/responsavel", tecnico, `{}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsavel", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/solicitacoes/x/responsaveis", tecnico, "")
	h.expect(http.StatusNotFound, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/responsaveis", tecnicoB, "")
}

const politicaA = `{"unidade_id":"%s","fuso":"America/Sao_Paulo",
	"turnos":[{"dia":1,"inicio":"08:00","fim":"18:00"},{"dia":2,"inicio":"08:00","fim":"18:00"},{"dia":3,"inicio":"08:00","fim":"18:00"},
		{"dia":4,"inicio":"08:00","fim":"18:00"},{"dia":5,"inicio":"08:00","fim":"18:00"}],
	"matriz":[{"impacto":1,"urgencia":1,"prioridade":"baixa"},{"impacto":1,"urgencia":2,"prioridade":"baixa"},{"impacto":1,"urgencia":3,"prioridade":"media"},
		{"impacto":2,"urgencia":1,"prioridade":"baixa"},{"impacto":2,"urgencia":2,"prioridade":"media"},{"impacto":2,"urgencia":3,"prioridade":"alta"},
		{"impacto":3,"urgencia":1,"prioridade":"media"},{"impacto":3,"urgencia":2,"prioridade":"alta"},{"impacto":3,"urgencia":3,"prioridade":"critica"}],
	"metas":[{"prioridade":"critica","resposta_min":30,"solucao_min":240},{"prioridade":"alta","resposta_min":60,"solucao_min":480},
		{"prioridade":"media","resposta_min":120,"solucao_min":960},{"prioridade":"baixa","resposta_min":480,"solucao_min":2400}]}`

// Fase 4: a gestão cadastra a política e os feriados da unidade; a
// subunidade herda; a solicitação nasce classificada e a equipe
// reclassifica; a fila e o detalhe da equipe mostram o prazo.
func TestAtendimentoPrazos(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	tecID, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	_, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	t.Cleanup(func() {
		_, _ = h.d.DB.Exec(context.Background(), `DELETE FROM atendimento_politicas WHERE unidade_id = $1`, o.a)
	})

	h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/politica?unidade_id="+o.a, tecnico, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/politica?unidade_id=x", tecnico, "")
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/politica?unidade_id="+o.b, tecnico, "")
	h.expect(http.StatusUnprocessableEntity, http.MethodPut, "/api/v1/atendimento/politica", tecnico,
		`{"unidade_id":"`+o.a+`","fuso":"America/Sao_Paulo","turnos":[],"matriz":[],"metas":[]}`)
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/atendimento/politica", tecnico, fmt.Sprintf(politicaA, o.b))
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/politica", tecnico, `{`)
	pol := data[idResp](t, h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/politica", tecnico, fmt.Sprintf(politicaA, o.a)))
	h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/politica", tecnico, fmt.Sprintf(politicaA, o.a))
	if outboxCount(t, h, "atendimento.politica.salva", pol.ID) != 2 {
		t.Fatal("cada gravação da política emite evento, com o mesmo id")
	}
	herdada := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/politica?unidade_id="+o.sub, tecnico, "").Body.String()
	if !strings.Contains(herdada, `"herdada":true`) {
		t.Fatalf("a subunidade herda a política de A: %s", herdada)
	}

	h.expect(http.StatusUnprocessableEntity, http.MethodPost, "/api/v1/atendimento/feriados", tecnico,
		`{"unidade_id":"`+o.a+`","data":"2026-11-20","nome":"x","tipo":"religioso"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/feriados", tecnico, `{`)
	feriado := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/feriados", tecnico,
		`{"unidade_id":"`+o.a+`","data":"2026-11-20","nome":"Consciência Negra","tipo":"nacional"}`))
	h.expect(http.StatusConflict, http.MethodPost, "/api/v1/atendimento/feriados", tecnico,
		`{"unidade_id":"`+o.a+`","data":"2026-11-20","nome":"De novo","tipo":"nacional"}`)
	lista := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/feriados?unidade_id="+o.a, tecnico, "").Body.String()
	if !strings.Contains(lista, feriado.ID) || !strings.Contains(lista, `"data":"2026-11-20"`) {
		t.Fatalf("feriados: %s", lista)
	}
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/feriados?unidade_id=x", tecnico, "")
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/feriados?unidade_id="+o.b, tecnico, "")
	h.expect(http.StatusNoContent, http.MethodDelete, "/api/v1/atendimento/feriados/"+feriado.ID, tecnico, "")
	h.expect(http.StatusNotFound, http.MethodDelete, "/api/v1/atendimento/feriados/"+feriado.ID, tecnico, "")
	h.expect(http.StatusBadRequest, http.MethodDelete, "/api/v1/atendimento/feriados/x", tecnico, "")

	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Rede fora","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Rede","unidade_id":"`+o.a+`","membros":["`+tecID.String()+`"]}`)
	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Sem rede no prédio"}`))
	visto := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, servidor, "").Body.String()
	if !strings.Contains(visto, `"prioridade":"media"`) || strings.Contains(visto, `"prazo"`) {
		t.Fatalf("nasce média; quem abriu não vê o prazo: %s", visto)
	}

	h.expect(http.StatusUnprocessableEntity, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/classificacao", tecnico,
		`{"impacto":0,"urgencia":2}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/solicitacoes/x/classificacao", tecnico, `{"impacto":1,"urgencia":1}`)
	h.expect(http.StatusNotFound, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+uuid.NewString()+"/classificacao", tecnico, `{"impacto":1,"urgencia":1}`)
	h.expect(http.StatusBadRequest, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/classificacao", tecnico, `{`)
	h.expect(http.StatusForbidden, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/classificacao", servidor,
		`{"impacto":3,"urgencia":3}`)
	critica := h.expect(http.StatusOK, http.MethodPut, "/api/v1/atendimento/solicitacoes/"+sol.ID+"/classificacao", tecnico,
		`{"impacto":3,"urgencia":3}`).Body.String()
	if !strings.Contains(critica, `"prioridade":"critica"`) || !strings.Contains(critica, `"meta_solucao_min":240`) || !strings.Contains(critica, `"prazo"`) {
		t.Fatalf("crítica, com prazo para a equipe: %s", critica)
	}
	if outboxCount(t, h, "atendimento.solicitacao.classificada", sol.ID) != 1 || h.deliver("atendimento.solicitacao.classificada", sol.ID) != 1 {
		t.Fatal("evento da classificação chega ao tempo real")
	}
	fila := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/fila", tecnico, "").Body.String()
	if !strings.Contains(fila, `"situacao"`) {
		t.Fatalf("a fila traz o semáforo: %s", fila)
	}

	// O alerta de prazo roda no worker, supervisionado pelo Kernel.
	var nomes []string
	for _, p := range h.d.Kernel.Plugins() {
		if wp, ok := p.(kernel.WorkerProvider); ok && p.Manifest().Key == "atendimento" {
			for _, w := range wp.Workers() {
				nomes = append(nomes, string(w.Process)+":"+w.Name)
			}
		}
	}
	if !slices.Equal(nomes, []string{"worker:alertas-de-prazo", "worker:retencao-da-localizacao"}) {
		t.Fatalf("workers do atendimento: %v", nomes)
	}
}

// Fase 5: o técnico faz check-in, junta foto e assinatura e faz check-out;
// a supervisão vê a posição só da visita em andamento; o detalhe e a
// auditoria nunca levam coordenadas.
func TestAtendimentoVisita(t *testing.T) {
	h := newHarness(t)
	o := h.orgEscopo(t)
	tecID, tecnico := h.pessoaEm(t, o.a, "atendimento:atender", "atendimento:manage")
	_, outroTec := h.pessoaEm(t, o.b, "atendimento:atender")
	_, servidor := h.pessoaEm(t, o.b, "atendimento:abrir")
	oferta := data[recursoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/ofertas", tecnico,
		`{"nome":"Rede no local","tipo":"incidente","unidade_id":"`+o.a+`"}`))
	h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/equipes", tecnico,
		`{"nome":"Campo","unidade_id":"`+o.a+`","membros":["`+tecID.String()+`"]}`)
	sol := data[solicitacaoResp](t, h.expect(http.StatusCreated, http.MethodPost, "/api/v1/atendimento/solicitacoes", servidor,
		`{"oferta_id":"`+oferta.ID+`","titulo":"Sem rede na escola"}`))

	visita := uuid.NewString()
	agora := time.Now().UTC().Format(time.RFC3339Nano)
	checkin := `{"id":"` + visita + `","local":"Escola","destino":{"lat":-23.5512,"lng":-46.634,"raio_m":50},
		"ponto":{"lat":-23.5503,"lng":-46.6339,"precisao_m":8},"capturada_em":"` + agora + `"}`
	base := "/api/v1/atendimento/solicitacoes/" + sol.ID + "/visitas"
	h.expect(http.StatusForbidden, http.MethodPost, base, servidor, checkin)
	h.expect(http.StatusNotFound, http.MethodPost, base, outroTec, checkin)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/solicitacoes/x/visitas", tecnico, checkin)
	h.expect(http.StatusBadRequest, http.MethodPost, base, tecnico, `{`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, base, tecnico,
		`{"id":"`+uuid.NewString()+`","ponto":{"lat":99,"lng":0},"capturada_em":"`+agora+`"}`)
	criada := h.expect(http.StatusCreated, http.MethodPost, base, tecnico, checkin).Body.String()
	if strings.Contains(criada, `"inicio"`) || !strings.Contains(criada, `"fora_da_area":true`) {
		t.Fatalf("check-in sem coordenadas na resposta: %s", criada)
	}
	h.expect(http.StatusCreated, http.MethodPost, base, tecnico, checkin)
	h.expect(http.StatusConflict, http.MethodPost, base, tecnico,
		`{"id":"`+uuid.NewString()+`","ponto":{"lat":-23.55,"lng":-46.63},"capturada_em":"`+agora+`"}`)

	campo := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/visitas/em-campo", tecnico, "").Body.String()
	if !strings.Contains(campo, visita) || !strings.Contains(campo, `"inicio"`) {
		t.Fatalf("a supervisão vê a posição: %s", campo)
	}
	if outra := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/visitas/em-campo", outroTec, "").Body.String(); strings.Contains(outra, visita) {
		t.Fatal("a supervisão de B não vê A")
	}
	h.expect(http.StatusForbidden, http.MethodGet, "/api/v1/atendimento/visitas/em-campo", servidor, "")

	vbase := "/api/v1/atendimento/visitas/" + visita
	ticket := data[struct {
		ObjectKey string `json:"object_key"`
	}](t, h.expect(http.StatusOK, http.MethodPost, vbase+"/evidencias/upload", tecnico, `{"nome":"assinatura.png","content_type":"image/png"}`))
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, vbase+"/evidencias/upload", tecnico, `{"nome":"x.exe","content_type":"application/x-msdownload"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, vbase+"/evidencias/upload", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/visitas/x/evidencias/upload", tecnico, `{"nome":"a.png","content_type":"image/png"}`)
	h.upload(ticket.ObjectKey, "image/png", []byte("png"))
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, vbase+"/evidencias", tecnico, `{"tipo":"video","object_key":"`+ticket.ObjectKey+`"}`)
	h.expect(http.StatusUnprocessableEntity, http.MethodPost, vbase+"/evidencias", tecnico,
		`{"tipo":"foto","object_key":"atendimento/visitas/`+visita+`/nao-enviada.png"}`)
	h.expect(http.StatusBadRequest, http.MethodPost, vbase+"/evidencias", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/visitas/x/evidencias", tecnico, `{"tipo":"foto","object_key":"k"}`)
	ev := data[idResp](t, h.expect(http.StatusCreated, http.MethodPost, vbase+"/evidencias", tecnico,
		`{"tipo":"assinatura","object_key":"`+ticket.ObjectKey+`"}`))
	url := h.expect(http.StatusOK, http.MethodGet, vbase+"/evidencias/"+ev.ID, servidor, "").Body.String()
	if !strings.Contains(url, `"url"`) {
		t.Fatalf("quem abriu vê a assinatura: %s", url)
	}
	h.expect(http.StatusNotFound, http.MethodGet, vbase+"/evidencias/"+ev.ID, outroTec, "")
	h.expect(http.StatusBadRequest, http.MethodGet, "/api/v1/atendimento/visitas/x/evidencias/"+ev.ID, tecnico, "")
	h.expect(http.StatusBadRequest, http.MethodGet, vbase+"/evidencias/x", tecnico, "")

	saida := `{"ponto":{"lat":-23.5512,"lng":-46.634,"precisao_m":5},"capturada_em":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","assinante_nome":"Diretora Ana"}`
	h.expect(http.StatusBadRequest, http.MethodPost, vbase+"/checkout", tecnico, `{`)
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/visitas/x/checkout", tecnico, saida)
	h.expect(http.StatusNotFound, http.MethodPost, "/api/v1/atendimento/visitas/"+uuid.NewString()+"/checkout", tecnico, saida)
	h.expect(http.StatusOK, http.MethodPost, vbase+"/checkout", tecnico, saida)
	h.expect(http.StatusOK, http.MethodPost, vbase+"/checkout", tecnico, saida)
	h.expect(http.StatusConflict, http.MethodPost, vbase+"/cancelar", tecnico, "")
	engano := uuid.NewString()
	h.expect(http.StatusCreated, http.MethodPost, base, tecnico,
		`{"id":"`+engano+`","ponto":{"lat":-23.55,"lng":-46.63},"capturada_em":"`+time.Now().UTC().Format(time.RFC3339Nano)+`"}`)
	h.expect(http.StatusOK, http.MethodPost, "/api/v1/atendimento/visitas/"+engano+"/cancelar", tecnico, "")
	h.expect(http.StatusBadRequest, http.MethodPost, "/api/v1/atendimento/visitas/x/cancelar", tecnico, "")

	detalhe := h.expect(http.StatusOK, http.MethodGet, "/api/v1/atendimento/solicitacoes/"+sol.ID, servidor, "").Body.String()
	if !strings.Contains(detalhe, `"assinante_nome":"Diretora Ana"`) || strings.Contains(detalhe, `"lat"`) {
		t.Fatalf("o detalhe mostra a visita sem coordenadas: %s", detalhe)
	}
	var trilha string
	if err := h.d.DB.QueryRow(context.Background(), `SELECT COALESCE(string_agg(diff_after::text, ' '), '') FROM audit_logs
		WHERE resource_type = 'visita' AND resource_id = $1`, visita).Scan(&trilha); err != nil {
		t.Fatal(err)
	}
	if trilha == "" || strings.Contains(trilha, `"lat"`) {
		t.Fatalf("a auditoria registra a visita sem coordenadas: %s", trilha)
	}
	if h.deliver("atendimento.visita.iniciada", visita) != 0 {
		t.Fatal("eventos de visita não vão para o tempo real da fila")
	}
}
