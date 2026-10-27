package application_test

import (
	"context"
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
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func (f falhaRepo) OfertaDosEventos(ctx context.Context, db database.DBTX) (*uuid.UUID, error) {
	if f.falha["OfertaDosEventos"] {
		return nil, errFalha
	}
	return f.Repository.OfertaDosEventos(ctx, db)
}
func (f falhaRepo) SalvarOfertaDosEventos(ctx context.Context, db database.DBTX, o *uuid.UUID, por uuid.UUID, agora time.Time) error {
	if f.falha["SalvarOfertaDosEventos"] {
		return errFalha
	}
	return f.Repository.SalvarOfertaDosEventos(ctx, db, o, por, agora)
}
func (f falhaRepo) TravarEvento(ctx context.Context, db database.DBTX, fp string) (domain.EventoVinculado, error) {
	if f.falha["TravarEvento"] {
		return domain.EventoVinculado{}, errFalha
	}
	return f.Repository.TravarEvento(ctx, db, fp)
}
func (f falhaRepo) SalvarEvento(ctx context.Context, db database.DBTX, e domain.EventoVinculado) error {
	if f.falha["SalvarEvento"] {
		return errFalha
	}
	return f.Repository.SalvarEvento(ctx, db, e)
}

func alerta(fp, status, severidade string) domain.Alerta {
	return domain.Alerta{
		Status: status, Fingerprint: fp,
		Labels:      map[string]string{"alertname": "NovaServiceDown", "severity": severidade},
		Annotations: map[string]string{"summary": "API sem responder"},
		StartsAt:    time.Date(2026, 10, 27, 14, 0, 0, 0, time.UTC),
		EndsAt:      time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC),
	}
}

// adminDe é um administrador que existe em users (updated_by tem FK).
func adminDe(t *testing.T, c cenario) auth.Identity {
	t.Helper()
	a := admin()
	a.UserID = dbtest.User(t, c.pool)
	return a
}

// eventosNoCenario liga a oferta do cenário à entrada de eventos.
func eventosNoCenario(t *testing.T, c cenario) {
	t.Helper()
	if _, err := c.base.ConfigurarEventos(context.Background(), adminDe(t, c), &c.oferta.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_eventos_config`)
	})
}

func chamadoDoAlerta(t *testing.T, c cenario, fp string) domain.Solicitacao {
	t.Helper()
	var id uuid.UUID
	if err := c.pool.QueryRow(context.Background(), `SELECT solicitacao_id FROM atendimento_eventos WHERE fingerprint = $1`, fp).Scan(&id); err != nil {
		t.Fatal(err)
	}
	d, err := c.base.Obter(context.Background(), c.tecnico, id)
	if err != nil {
		t.Fatal(err)
	}
	return d.Solicitacao
}

func TestAlertaViraIncidenteUmaVez(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	eventosNoCenario(t, c)
	fp := "fp-" + uuid.NewString()

	r, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta(fp, domain.AlertaDisparando, "critical")})
	if err != nil || r.Abertos != 1 {
		t.Fatalf("abrir: %+v %v", r, err)
	}
	sol := chamadoDoAlerta(t, c, fp)
	if sol.SolicitanteID != domain.ContaMonitoramento || sol.Titulo != "API sem responder" || sol.Impacto != 3 || sol.Urgencia != 3 ||
		!strings.Contains(sol.Descricao, "alerta NovaServiceDown") {
		t.Fatalf("chamado: %+v", sol)
	}

	// O mesmo alerta de novo e a volta ao normal viram notas, não chamados.
	if r, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta(fp, domain.AlertaDisparando, "critical")}); err != nil || r.Anotados != 1 || r.Abertos != 0 {
		t.Fatalf("dispara de novo: %+v %v", r, err)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET responsavel_id = $1 WHERE id = $2`, c.tecnico.UserID, sol.ID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta(fp, domain.AlertaNormalizou, "critical")}); err != nil || r.Anotados != 1 {
		t.Fatalf("normalizou: %+v %v", r, err)
	}
	d, _ := c.base.Obter(ctx, c.tecnico, sol.ID)
	if d.Solicitacao.Finalizada() || len(d.Comentarios) != 2 || !d.Comentarios[0].Interno ||
		!strings.Contains(d.Comentarios[1].Texto, "normalizou") || !strings.Contains(d.Comentarios[1].Texto, "Confira a causa") {
		t.Fatalf("normalizou não conclui: %+v", d.Comentarios)
	}

	// Normalizado sem chamado aberto: nada a fazer.
	if r, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta("fp-"+uuid.NewString(), domain.AlertaNormalizou, "warning")}); err != nil || r.Ignorados != 1 {
		t.Fatalf("normalizou sem chamado: %+v %v", r, err)
	}

	// Fechado o chamado, o próximo disparo abre outro, com severidade média.
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET estado = 'cancelado' WHERE id = $1`, sol.ID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta(fp, domain.AlertaDisparando, "warning")}); err != nil || r.Abertos != 1 {
		t.Fatalf("reabrir depois de fechado: %+v %v", r, err)
	}
	if outro := chamadoDoAlerta(t, c, fp); outro.ID == sol.ID || outro.Impacto != 2 {
		t.Fatalf("chamado novo, impacto médio: %+v", outro)
	}
}

func TestEventosCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	ok := []domain.Alerta{alerta("fp-"+uuid.NewString(), domain.AlertaDisparando, "critical")}

	if _, err := c.base.ReceberAlertas(ctx, ok); !errors.Is(err, domain.ErrEventosSemOferta) {
		t.Fatalf("sem oferta escolhida: %v", err)
	}
	eventosNoCenario(t, c)
	com := func(m ...string) *application.Service { return c.com(falhando(m...), eventosFalham{}) }
	casos := map[string]struct {
		svc    *application.Service
		in     []domain.Alerta
		espera error
	}{
		"alerta inválido":           {c.base, []domain.Alerta{{Status: "x"}}, domain.ErrAlertaInvalido},
		"falha ao ler a oferta":     {com("OfertaDosEventos"), ok, errFalha},
		"falha na oferta":           {com("GetOferta"), ok, errFalha},
		"falha ao travar":           {com("TravarEvento"), ok, errFalha},
		"falha ao abrir":            {com("InsertSolicitacao"), ok, errFalha},
		"falha ao ligar o alerta":   {com("SalvarEvento"), ok, errFalha},
		"falha no evento do outbox": {c.com(falhando(), eventosFalham{application.EventoSolicitacaoAberta}), ok, errFalha},
	}
	for nome, k := range casos {
		if _, err := k.svc.ReceberAlertas(ctx, k.in); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", nome, err)
		}
	}

	// Com chamado aberto: falha ao lê-lo e falha na nota.
	fp := "fp-" + uuid.NewString()
	if _, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta(fp, domain.AlertaDisparando, "critical")}); err != nil {
		t.Fatal(err)
	}
	repete := []domain.Alerta{alerta(fp, domain.AlertaDisparando, "critical")}
	if _, err := com("GetSolicitacao").ReceberAlertas(ctx, repete); !errors.Is(err, errFalha) {
		t.Fatalf("ler o chamado do alerta: %v", err)
	}
	if _, err := com("InsertComentario").ReceberAlertas(ctx, repete); !errors.Is(err, errFalha) {
		t.Fatalf("gravar a nota: %v", err)
	}
	if _, err := c.com(falhando(), eventosFalham{application.EventoSolicitacaoComentada}).ReceberAlertas(ctx, repete); !errors.Is(err, errFalha) {
		t.Fatalf("evento da nota: %v", err)
	}

	// Oferta desligada depois de escolhida: o alerta não abre chamado.
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_ofertas SET ativa = false WHERE id = $1`, c.oferta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.ReceberAlertas(ctx, []domain.Alerta{alerta("fp-"+uuid.NewString(), domain.AlertaDisparando, "critical")}); !errors.Is(err, domain.ErrOfertaInativa) {
		t.Fatalf("oferta desligada: %v", err)
	}
}

func TestConfigurarEventos(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	t.Cleanup(func() { _, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_eventos_config`) })
	daUnidade := gestor(c.unidade)
	adm := adminDe(t, c)

	if _, err := c.base.ConfiguracaoDosEventos(ctx, daUnidade); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("gestor de unidade não lê: %v", err)
	}
	if _, err := c.base.ConfigurarEventos(ctx, daUnidade, &c.oferta.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("gestor de unidade não escolhe: %v", err)
	}
	if cfg, err := c.base.ConfigurarEventos(ctx, adm, &c.oferta.ID); err != nil || *cfg.OfertaID != c.oferta.ID {
		t.Fatalf("escolher: %+v %v", cfg, err)
	}
	if cfg, err := c.base.ConfiguracaoDosEventos(ctx, adm); err != nil || cfg.OfertaID == nil || *cfg.OfertaID != c.oferta.ID {
		t.Fatalf("ler: %+v %v", cfg, err)
	}
	if cfg, err := c.base.ConfigurarEventos(ctx, adm, nil); err != nil || cfg.OfertaID != nil {
		t.Fatalf("limpar: %+v %v", cfg, err)
	}
	pedido, err := c.base.CriarOferta(ctx, adm, application.OfertaEntrada{Nome: "Pedido " + uuid.NewString()[:6], Tipo: domain.TipoRequisicao, UnidadeID: &c.unidade})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, pedido.ID)
	})
	inexistente := uuid.New()
	for nome, k := range map[string]struct {
		svc    *application.Service
		quem   auth.Identity
		oferta *uuid.UUID
		espera error
	}{
		"oferta inexistente":    {c.base, adm, &inexistente, domain.ErrOfertaInexistente},
		"oferta de requisição":  {c.base, adm, &pedido.ID, domain.ErrEntradaInvalida},
		"falha ao ler a oferta": {c.com(falhando("GetOferta"), eventosFalham{}), adm, &c.oferta.ID, errFalha},
		"falha ao gravar":       {c.com(falhando("SalvarOfertaDosEventos"), eventosFalham{}), adm, nil, errFalha},
	} {
		if _, err := k.svc.ConfigurarEventos(ctx, k.quem, k.oferta); !errors.Is(err, k.espera) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := c.com(falhando("OfertaDosEventos"), eventosFalham{}).ConfiguracaoDosEventos(ctx, adm); !errors.Is(err, errFalha) {
		t.Fatalf("falha ao ler a configuração: %v", err)
	}
}

// webhook é o corpo que o Alertmanager manda, com os campos que não usamos.
func webhook(fp string) map[string]any {
	return map[string]any{
		"version": "4", "groupKey": "{}:{alertname=\"NovaServiceDown\"}", "truncatedAlerts": 0, "status": "firing",
		"receiver": "atendimento", "groupLabels": map[string]string{"alertname": "NovaServiceDown"},
		"commonLabels": map[string]string{}, "commonAnnotations": map[string]string{}, "externalURL": "http://alertmanager:9093",
		"alerts": []map[string]any{{
			"status": "firing", "fingerprint": fp, "generatorURL": "http://prometheus:9090/graph",
			"labels":      map[string]string{"alertname": "NovaServiceDown", "severity": "critical"},
			"annotations": map[string]string{"summary": "API sem responder"},
			"startsAt":    "2026-10-27T14:00:00Z", "endsAt": "0001-01-01T00:00:00Z",
		}},
	}
}

func TestHandlersDosEventos(t *testing.T) {
	c := novoCenario(t)
	eventosNoCenario(t, c)
	adm := adminDe(t, c)
	desligada := transport.NewHandlers(c.base, silencio)
	h := transport.NewHandlers(c.base, silencio).ComTokenDeEventos("segredo-da-maquina")
	recebe := func(hs *transport.Handlers, auth string, corpo any) *httptest.ResponseRecorder {
		req := pedido(t, http.MethodPost, "/eventos", "", corpo, adm)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		hs.ReceberAlertas(rec, req)
		return rec
	}
	fp := "fp-" + uuid.NewString()

	if rec := recebe(desligada, "Bearer segredo-da-maquina", webhook(fp)); rec.Code != http.StatusNotFound {
		t.Fatalf("sem token configurado, a entrada não existe: %d", rec.Code)
	}
	for _, ruim := range []string{"", "Bearer outro", "segredo-da-maquina"} {
		if rec := recebe(h, ruim, webhook(fp)); rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q: %d", ruim, rec.Code)
		}
	}
	rec := recebe(h, "Bearer segredo-da-maquina", webhook(fp))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"abertos":1`) {
		t.Fatalf("webhook: %d %s", rec.Code, rec.Body.String())
	}
	if rec := recebe(h, "Bearer segredo-da-maquina", map[string]any{"alerts": []any{}}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sem alertas: %d", rec.Code)
	}
	if rec := recebe(h, "Bearer segredo-da-maquina", map[string]any{"alerts": []map[string]any{{"status": "x"}}}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("alerta inválido: %d", rec.Code)
	}
	if rec := recebe(h, "Bearer segredo-da-maquina", map[string]any{"campo_estranho": 1}); rec.Code != http.StatusBadRequest {
		t.Fatalf("campo desconhecido: %d", rec.Code)
	}

	cfg := func(fn http.HandlerFunc, metodo string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/cfg", "", corpo, quem))
		return rec
	}
	if rec := cfg(h.ConfiguracaoDosEventos, http.MethodGet, nil, adm); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), c.oferta.ID.String()) {
		t.Fatalf("ler configuração: %d %s", rec.Code, rec.Body.String())
	}
	if rec := cfg(h.ConfiguracaoDosEventos, http.MethodGet, nil, gestor(c.unidade)); rec.Code != http.StatusForbidden {
		t.Fatalf("gestor de unidade lê: %d", rec.Code)
	}
	if rec := cfg(h.ConfigurarEventos, http.MethodPut, map[string]any{"oferta_id": c.oferta.ID}, adm); rec.Code != http.StatusOK {
		t.Fatalf("configurar: %d %s", rec.Code, rec.Body.String())
	}
	if rec := cfg(h.ConfigurarEventos, http.MethodPut, map[string]any{"oferta_id": "x"}, adm); rec.Code != http.StatusBadRequest {
		t.Fatalf("oferta inválida: %d", rec.Code)
	}
	if rec := cfg(h.ConfigurarEventos, http.MethodPut, map[string]any{"oferta_id": c.oferta.ID}, gestor(c.unidade)); rec.Code != http.StatusForbidden {
		t.Fatalf("gestor de unidade configura: %d", rec.Code)
	}
}
