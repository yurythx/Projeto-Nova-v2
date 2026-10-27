package application_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (f falhaRepo) MetasXLA(ctx context.Context, db database.DBTX) (domain.MetasXLA, error) {
	if f.falha["MetasXLA"] {
		return domain.MetasXLA{}, errFalha
	}
	return f.Repository.MetasXLA(ctx, db)
}
func (f falhaRepo) SalvarMetasXLA(ctx context.Context, db database.DBTX, m domain.MetasXLA, por uuid.UUID, agora time.Time) error {
	if f.falha["SalvarMetasXLA"] {
		return errFalha
	}
	return f.Repository.SalvarMetasXLA(ctx, db, m, por, agora)
}

// semMetas zera as metas da instituição antes e depois do teste.
func semMetas(t *testing.T, c cenario) {
	t.Helper()
	limpa := func() { _, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_xla`) }
	limpa()
	t.Cleanup(limpa)
}

func TestExperienciaNaGovernanca(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	semMetas(t, c)
	gestorGlobal := adminDe(t, c)

	// Uma resolvida de primeira; outra que voltou ao servidor e foi reaberta.
	direta := c.emAtendimento(t)
	concluir(t, c, direta)
	devolvida := c.emAtendimento(t)
	if _, err := c.base.Transicionar(ctx, c.tecnico, devolvida.ID, domain.EstadoAguardandoServidor); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Comentar(ctx, c.servidor, devolvida.ID, "Segue o print", false); err != nil {
		t.Fatal(err)
	}
	concluir(t, c, devolvida)
	if _, err := c.base.Transicionar(ctx, c.servidor, devolvida.ID, domain.EstadoEmAtendimento); err != nil {
		t.Fatal(err)
	}
	concluir(t, c, devolvida)

	dois, cinco := 2, 5
	if _, err := c.base.AvaliarCom(ctx, c.servidor, direta.ID, 5, &cinco, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.AvaliarCom(ctx, c.servidor, devolvida.ID, 3, &dois, "demorou"); err != nil {
		t.Fatal(err)
	}
	nota, horas := 4.5, 1
	if _, err := c.base.SalvarMetasXLA(ctx, gestorGlobal, domain.MetasXLA{CSATMin: &nota, EsperaMaxHoras: &horas}); err != nil {
		t.Fatal(err)
	}

	g, err := c.base.Governanca(ctx, c.tecnico)
	if err != nil || g.Experiencia == nil {
		t.Fatalf("governança: %+v %v", g, err)
	}
	x := g.Experiencia
	if *x.CSAT.Valor != 4 || *x.CSAT.Atingida || *x.Facilidade.Valor != 3.5 || x.Facilidade.Atingida != nil {
		t.Fatalf("nota e facilidade: %+v %+v", x.CSAT, x.Facilidade)
	}
	if *x.Reabertura.Valor != 50 || *x.IdasEVoltas.Valor != 0.5 || x.Reabertura.Amostra != 2 {
		t.Fatalf("reabertura e idas: %+v %+v", x.Reabertura, x.IdasEVoltas)
	}
	if x.EsperaHoras.Amostra != 2 || !*x.EsperaHoras.Atingida {
		t.Fatalf("espera de minutos cumpre a meta de uma hora: %+v", x.EsperaHoras)
	}
	if m, err := c.base.MetasXLA(ctx, c.tecnico); err != nil || *m.CSATMin != 4.5 {
		t.Fatalf("quem atende lê as metas: %+v %v", m, err)
	}
}

func TestExperienciaRegras(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	semMetas(t, c)
	em := c.emAtendimento(t)
	concluir(t, c, em)
	seis := 6
	if _, err := c.base.AvaliarCom(ctx, c.servidor, em.ID, 5, &seis, ""); !errors.Is(err, domain.ErrFacilidadeInvalida) {
		t.Fatalf("facilidade fora: %v", err)
	}
	fora := 9.0
	gestorGlobal := adminDe(t, c)
	for nome, k := range map[string]struct {
		err, espera error
	}{
		"metas por gestor de unidade": {func() error { _, err := c.base.SalvarMetasXLA(ctx, gestor(c.unidade), domain.MetasXLA{}); return err }(), domain.ErrForaDoEscopo},
		"metas fora da faixa": {func() error {
			_, err := c.base.SalvarMetasXLA(ctx, gestorGlobal, domain.MetasXLA{CSATMin: &fora})
			return err
		}(), domain.ErrMetasInvalidas},
		"ler sem atender": {func() error { _, err := c.base.MetasXLA(ctx, c.servidor); return err }(), domain.ErrForaDoEscopo},
		"falha ao ler as metas": {func() error {
			_, err := c.com(falhando("MetasXLA"), eventosFalham{}).MetasXLA(ctx, c.tecnico)
			return err
		}(), errFalha},
		"falha na governança": {func() error {
			_, err := c.com(falhando("MetasXLA"), eventosFalham{}).Governanca(ctx, c.tecnico)
			return err
		}(), errFalha},
		"falha ao ler antes de salvar": {func() error {
			_, err := c.com(falhando("MetasXLA"), eventosFalham{}).SalvarMetasXLA(ctx, gestorGlobal, domain.MetasXLA{})
			return err
		}(), errFalha},
		"falha ao salvar": {func() error {
			_, err := c.com(falhando("SalvarMetasXLA"), eventosFalham{}).SalvarMetasXLA(ctx, gestorGlobal, domain.MetasXLA{})
			return err
		}(), errFalha},
	} {
		if !errors.Is(k.err, k.espera) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}
	salvas, err := c.base.SalvarMetasXLA(ctx, gestorGlobal, domain.MetasXLA{})
	if err != nil || salvas.UpdatedAt == nil {
		t.Fatalf("salvar sem metas: %+v %v", salvas, err)
	}
}

func TestHandlersDaExperiencia(t *testing.T) {
	c := novoCenario(t)
	semMetas(t, c)
	em := c.emAtendimento(t)
	concluir(t, c, em)
	h := transport.NewHandlers(c.base, silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/x", id, corpo, quem))
		return rec
	}
	global := adminDe(t, c)
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"ler as metas":           {chama(h.MetasXLA, http.MethodGet, "", nil, c.tecnico), http.StatusOK},
		"ler sem atender":        {chama(h.MetasXLA, http.MethodGet, "", nil, c.servidor), http.StatusForbidden},
		"salvar as metas":        {chama(h.SalvarMetasXLA, http.MethodPut, "", map[string]any{"csat_min": 4, "espera_max_horas": 24}, global), http.StatusOK},
		"salvar fora da faixa":   {chama(h.SalvarMetasXLA, http.MethodPut, "", map[string]any{"reabertura_max": 150}, global), http.StatusUnprocessableEntity},
		"salvar sem ser global":  {chama(h.SalvarMetasXLA, http.MethodPut, "", map[string]any{}, gestor(c.unidade)), http.StatusForbidden},
		"avaliar com facilidade": {chama(h.Avaliar, http.MethodPost, em.ID.String(), map[string]any{"nota": 4, "facilidade": 5}, c.servidor), http.StatusCreated},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
}
