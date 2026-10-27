package application_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (f falhaRepo) SetContratoApoio(ctx context.Context, db database.DBTX, id uuid.UUID, c *domain.ContratoDeApoio, agora time.Time) error {
	if f.falha["SetContratoApoio"] {
		return errFalha
	}
	return f.Repository.SetContratoApoio(ctx, db, id, c, agora)
}
func (f falhaRepo) ComContratoNoPeriodo(ctx context.Context, db database.DBTX, r bool, u []uuid.UUID, d time.Time) ([]domain.Solicitacao, error) {
	if f.falha["ComContratoNoPeriodo"] {
		return nil, errFalha
	}
	return f.Repository.ComContratoNoPeriodo(ctx, db, r, u, d)
}

// contratosFalsos faz o papel do plug-in Contratos.
type contratosFalsos struct {
	lista []domain.ContratoDeApoio
	err   error
}

func (c contratosFalsos) Listar(context.Context, auth.Identity) ([]domain.ContratoDeApoio, error) {
	return c.lista, c.err
}

func (c contratosFalsos) Obter(_ context.Context, _ auth.Identity, id uuid.UUID) (domain.ContratoDeApoio, error) {
	if c.err != nil {
		return domain.ContratoDeApoio{}, c.err
	}
	for _, x := range c.lista {
		if x.ID == id {
			return x, nil
		}
	}
	return domain.ContratoDeApoio{}, domain.ErrContratoAusente
}

func TestContratoDeApoio(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quatro := 4
	link := domain.ContratoDeApoio{ID: uuid.New(), Rotulo: "12/2026 — Link (Provedor)", Fornecedor: "Provedor", PrazoHoras: &quatro}
	porta := contratosFalsos{lista: []domain.ContratoDeApoio{link}}
	c.base.WithContratos(porta)
	em := c.emAtendimento(t)

	if lista, err := c.base.ContratosDeApoio(ctx, c.tecnico); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	d, err := c.base.VincularContrato(ctx, c.tecnico, em.ID, &link.ID)
	if err != nil || d.Solicitacao.ContratoApoio != link.Rotulo || *d.Solicitacao.ContratoPrazoHoras != 4 {
		t.Fatalf("vincular: %+v %v", d.Solicitacao, err)
	}
	if visto, _ := c.base.Obter(ctx, c.servidor, em.ID); visto.Solicitacao.ContratoApoioID != nil || visto.Solicitacao.ContratoApoio != "" {
		t.Fatalf("quem pediu não vê o contrato: %+v", visto.Solicitacao)
	}

	// Com o fornecedor e de volta: o relatório mede o contrato.
	if _, err := c.base.Transicionar(ctx, c.tecnico, em.ID, domain.EstadoAguardandoTerceiro); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Transicionar(ctx, c.tecnico, em.ID, domain.EstadoEmAtendimento); err != nil {
		t.Fatal(err)
	}
	rel, err := c.base.RelatorioPorSecretaria(ctx, c.tecnico, 0)
	if err != nil || len(rel.Contratos) != 1 || rel.Contratos[0].Medidos != 1 || *rel.Contratos[0].NoPrazoPercentual != 100 {
		t.Fatalf("relatório do contrato: %+v %v", rel.Contratos, err)
	}
	for _, m := range []string{"ComContratoNoPeriodo", "PassagensDe"} {
		if _, err := c.com(falhando(m), eventosFalham{}).RelatorioPorSecretaria(ctx, c.tecnico, 0); !errors.Is(err, errFalha) {
			t.Errorf("relatório, falha em %s: %v", m, err)
		}
	}

	if d, err := c.base.VincularContrato(ctx, c.tecnico, em.ID, nil); err != nil || d.Solicitacao.ContratoApoioID != nil {
		t.Fatalf("limpar: %+v %v", d.Solicitacao, err)
	}
	outro := uuid.New()
	desligado := c.com(falhando(), eventosFalham{})
	semPorta := c.com(falhando(), eventosFalham{})
	desligado.WithContratos(contratosFalsos{err: domain.ErrContratosDesligado})
	quebrado := c.com(falhando(), eventosFalham{}).WithContratos(contratosFalsos{err: errFalha})
	comFalha := func(m string) *application.Service { return c.com(falhando(m), eventosFalham{}).WithContratos(porta) }
	for nome, k := range map[string]struct {
		err, espera error
	}{
		"contrato fora da lista": {func() error { _, err := c.base.VincularContrato(ctx, c.tecnico, em.ID, &outro); return err }(), domain.ErrContratoAusente},
		"quem pediu não vincula": {func() error { _, err := c.base.VincularContrato(ctx, c.servidor, em.ID, &link.ID); return err }(), domain.ErrForaDoEscopo},
		"sem a porta":            {func() error { _, err := semPorta.VincularContrato(ctx, c.tecnico, em.ID, &link.ID); return err }(), domain.ErrContratosDesligado},
		"plug-in desligado":      {func() error { _, err := desligado.VincularContrato(ctx, c.tecnico, em.ID, &link.ID); return err }(), domain.ErrContratosDesligado},
		"porta com erro":         {func() error { _, err := quebrado.VincularContrato(ctx, c.tecnico, em.ID, &link.ID); return err }(), domain.ErrContratoAusente},
		"falha ao gravar": {func() error {
			_, err := comFalha("SetContratoApoio").VincularContrato(ctx, c.tecnico, em.ID, &link.ID)
			return err
		}(), errFalha},
		"falha no evento": {func() error {
			_, err := c.com(falhando(), eventosFalham{application.EventoSolicitacaoContrato}).WithContratos(porta).VincularContrato(ctx, c.tecnico, em.ID, &link.ID)
			return err
		}(), errFalha},
		"lista sem a porta": {func() error { _, err := semPorta.ContratosDeApoio(ctx, c.tecnico); return err }(), domain.ErrContratosDesligado},
		"lista sem atender": {func() error { _, err := c.base.ContratosDeApoio(ctx, c.servidor); return err }(), domain.ErrForaDoEscopo},
		"lista com erro":    {func() error { _, err := quebrado.ContratosDeApoio(ctx, c.tecnico); return err }(), errFalha},
	} {
		if !errors.Is(k.err, k.espera) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}
	concluir(t, c, em)
	if _, err := c.base.Transicionar(ctx, c.tecnico, em.ID, domain.EstadoEncerrado); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularContrato(ctx, c.tecnico, em.ID, &link.ID); !errors.Is(err, domain.ErrSolicitacaoFinalizada) {
		t.Fatalf("encerrado: %v", err)
	}
}

func TestHandlersDoContratoDeApoio(t *testing.T) {
	c := novoCenario(t)
	link := domain.ContratoDeApoio{ID: uuid.New(), Rotulo: "Link"}
	c.base.WithContratos(contratosFalsos{lista: []domain.ContratoDeApoio{link}})
	em := c.emAtendimento(t)
	h := transport.NewHandlers(c.base, silencio)
	chama := func(fn http.HandlerFunc, metodo, id string, corpo any, quem auth.Identity) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		fn(rec, pedido(t, metodo, "/x", id, corpo, quem))
		return rec
	}
	for nome, k := range map[string]struct {
		rec    *httptest.ResponseRecorder
		espera int
	}{
		"listar":                {chama(h.ContratosDeApoio, http.MethodGet, "", nil, c.tecnico), http.StatusOK},
		"listar sem atender":    {chama(h.ContratosDeApoio, http.MethodGet, "", nil, c.servidor), http.StatusForbidden},
		"vincular":              {chama(h.VincularContrato, http.MethodPut, em.ID.String(), map[string]any{"contrato_id": link.ID}, c.tecnico), http.StatusOK},
		"vincular, id inválido": {chama(h.VincularContrato, http.MethodPut, "x", map[string]any{}, c.tecnico), http.StatusBadRequest},
		"contrato ausente":      {chama(h.VincularContrato, http.MethodPut, em.ID.String(), map[string]any{"contrato_id": uuid.New()}, c.tecnico), http.StatusUnprocessableEntity},
		"corpo inválido":        {chama(h.VincularContrato, http.MethodPut, em.ID.String(), "x", c.tecnico), http.StatusBadRequest},
	} {
		if k.rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, k.rec.Code, k.rec.Body.String())
		}
	}
	semPorta := transport.NewHandlers(c.com(falhando(), eventosFalham{}), silencio)
	if rec := chama(semPorta.ContratosDeApoio, http.MethodGet, "", nil, c.tecnico); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("plug-in desligado: %d", rec.Code)
	}
}
