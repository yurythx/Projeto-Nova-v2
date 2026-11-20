package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

func (f falhaRepo) ListarRegras(ctx context.Context, db database.DBTX, filtro domain.Filtro) ([]domain.Regra, error) {
	if f.falha["ListarRegras"] {
		return nil, errFalha
	}
	return f.Repository.ListarRegras(ctx, db, filtro)
}

func (f falhaRepo) ContarRegras(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (int, error) {
	if f.falha["ContarRegras"] {
		return 0, errFalha
	}
	return f.Repository.ContarRegras(ctx, db, unidade)
}

func (f falhaRepo) InserirRegra(ctx context.Context, db database.DBTX, regra domain.Regra) error {
	if f.falha["InserirRegra"] {
		return errFalha
	}
	return f.Repository.InserirRegra(ctx, db, regra)
}

func (f falhaRepo) ObterRegra(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Regra, error) {
	if f.falha["ObterRegra"] {
		return domain.Regra{}, errFalha
	}
	return f.Repository.ObterRegra(ctx, db, id)
}

func (f falhaRepo) ExcluirRegra(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["ExcluirRegra"] {
		return errFalha
	}
	return f.Repository.ExcluirRegra(ctx, db, id)
}

func (f falhaRepo) ObterResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) (domain.RespostaPronta, error) {
	if f.falha["ObterResposta"] {
		return domain.RespostaPronta{}, errFalha
	}
	return f.Repository.ObterResposta(ctx, db, oferta)
}

func (f falhaRepo) SalvarResposta(ctx context.Context, db database.DBTX, item domain.RespostaPronta) error {
	if f.falha["SalvarResposta"] {
		return errFalha
	}
	return f.Repository.SalvarResposta(ctx, db, item)
}

func (f falhaRepo) ExcluirResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) error {
	if f.falha["ExcluirResposta"] {
		return errFalha
	}
	return f.Repository.ExcluirResposta(ctx, db, oferta)
}

func (f falhaRepo) ListarRecorrencias(ctx context.Context, db database.DBTX, filtro domain.Filtro) ([]domain.ChamadoMensal, error) {
	if f.falha["ListarRecorrencias"] {
		return nil, errFalha
	}
	return f.Repository.ListarRecorrencias(ctx, db, filtro)
}

func (f falhaRepo) ListarRecorrenciasAtivas(ctx context.Context, db database.DBTX) ([]domain.ChamadoMensal, error) {
	if f.falha["ListarRecorrenciasAtivas"] {
		return nil, errFalha
	}
	return f.Repository.ListarRecorrenciasAtivas(ctx, db)
}

func (f falhaRepo) InserirRecorrencia(ctx context.Context, db database.DBTX, item domain.ChamadoMensal) error {
	if f.falha["InserirRecorrencia"] {
		return errFalha
	}
	return f.Repository.InserirRecorrencia(ctx, db, item)
}

func (f falhaRepo) ObterRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.ChamadoMensal, error) {
	if f.falha["ObterRecorrencia"] {
		return domain.ChamadoMensal{}, errFalha
	}
	return f.Repository.ObterRecorrencia(ctx, db, id)
}

func (f falhaRepo) ExcluirRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["ExcluirRecorrencia"] {
		return errFalha
	}
	return f.Repository.ExcluirRecorrencia(ctx, db, id)
}

func (f falhaRepo) ReivindicarAbertura(ctx context.Context, db database.DBTX, id uuid.UUID, ano, mes int) (bool, error) {
	if f.falha["ReivindicarAbertura"] {
		return false, errFalha
	}
	return f.Repository.ReivindicarAbertura(ctx, db, id, ano, mes)
}

func (f falhaRepo) RegistrarDeflexao(ctx context.Context, db database.DBTX, usuario uuid.UUID, artigo, dia string, unidade *uuid.UUID, resolveu bool) error {
	if f.falha["RegistrarDeflexao"] {
		return errFalha
	}
	return f.Repository.RegistrarDeflexao(ctx, db, usuario, artigo, dia, unidade, resolveu)
}

func (f falhaRepo) ListarRegrasDaUnidade(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Regra, error) {
	if f.falha["ListarRegrasDaUnidade"] {
		return nil, errFalha
	}
	return f.Repository.ListarRegrasDaUnidade(ctx, db, unidade)
}

func (f falhaRepo) SolicitacaoDoMes(ctx context.Context, db database.DBTX, oferta, solicitante uuid.UUID, titulo string, inicio time.Time) (uuid.UUID, error) {
	if f.falha["SolicitacaoDoMes"] {
		return uuid.Nil, errFalha
	}
	return f.Repository.SolicitacaoDoMes(ctx, db, oferta, solicitante, titulo, inicio)
}

func (f falhaRepo) ParaEncerrar(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if f.falha["ParaEncerrar"] {
		return nil, errFalha
	}
	return f.Repository.ParaEncerrar(ctx, db, ate, limite)
}

func (f falhaRepo) Paradas(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if f.falha["Paradas"] {
		return nil, errFalha
	}
	return f.Repository.Paradas(ctx, db, ate, limite)
}

func TestRamosDaFila(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	gestor := admin()
	gestor.UserID = c.servidor.UserID
	logger := slog.New(slog.DiscardHandler)
	c.base = application.NewService(c.pool, c.baseRepo(), eventosFalham{}, nil).WithLogger(logger)

	if lista, err := c.base.ListarRegras(ctx, auth.Identity{}); err != nil || len(lista) != 0 {
		t.Fatalf("sem escopo: %+v %v", lista, err)
	}
	if _, err := c.base.CriarRegra(ctx, c.servidor, domain.Regra{Nome: "Senha", Prioridade: domain.PrioridadeAlta, UnidadeID: &c.unidade}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("regra sem gestão: %v", err)
	}
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "", UnidadeID: &c.unidade}); err == nil {
		t.Fatal("regra sem nome")
	}
	outra := dbtest.Unidade(t, c.pool)
	alheia, err := c.base.CriarOferta(ctx, gestor, application.OfertaEntrada{Nome: "Outra " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &outra})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, alheia.ID)
	})
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "Alheia", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade, OfertaID: &alheia.ID}); err == nil {
		t.Fatal("oferta de outra unidade")
	}
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "Sem unidade", Prioridade: domain.PrioridadeBaixa, OfertaID: &c.oferta.ID}); err == nil {
		t.Fatal("oferta com unidade e regra sem")
	}

	regra, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "Senha", Palavra: "senha", Prioridade: domain.PrioridadeAlta, UnidadeID: &c.unidade})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_regras WHERE unidade_id = $1`, c.unidade)
	})
	if lista, err := c.base.ListarRegras(ctx, gestor); err != nil || len(lista) == 0 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	if err := c.base.ExcluirRegra(ctx, c.servidor, regra.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("excluir sem gestão: %v", err)
	}
	if err := c.base.ExcluirRegra(ctx, gestor, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("regra ausente: %v", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO atendimento_regras (id, unidade_id, ordem, nome, prioridade, created_at, updated_at)
		SELECT gen_random_uuid(), $1, 2, 'cheia ' || g, 'baixa', now(), now() FROM generate_series(1, 29) g`, c.unidade); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "A trigésima primeira", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade}); err == nil {
		t.Fatal("teto de 30")
	}
	if _, err := c.pool.Exec(ctx, `DELETE FROM atendimento_regras WHERE unidade_id = $1 AND nome LIKE 'cheia %'`, c.unidade); err != nil {
		t.Fatal(err)
	}

	if _, err := c.base.ObterResposta(ctx, c.servidor, c.oferta.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("resposta sem atendimento: %v", err)
	}
	if _, err := c.base.ObterResposta(ctx, c.tecnico, c.oferta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.SalvarResposta(ctx, c.servidor, c.oferta.ID, "oi", ""); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("salvar sem gestão: %v", err)
	}
	if _, err := c.base.SalvarResposta(ctx, gestor, c.oferta.ID, strings.Repeat("a", 4001), ""); err == nil {
		t.Fatal("resposta longa")
	}
	if _, err := c.base.CriarRecorrencia(ctx, c.servidor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Backup", Dia: 1}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("recorrência sem gestão: %v", err)
	}
	if _, err := c.base.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Backup", Dia: 0}); err == nil {
		t.Fatal("dia zero")
	}
	if err := c.base.RegistrarDeflexao(ctx, gestor, []string{"a", "b", "c", "d", "e"}, false); err == nil {
		t.Fatal("cinco artigos")
	}
	if lista, err := c.base.ListarRecorrencias(ctx, auth.Identity{}); err != nil || len(lista) != 0 {
		t.Fatalf("recorrências sem escopo: %+v %v", lista, err)
	}

	falho := application.NewService(c.pool, falhando("ContarRegras"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.CriarRegra(ctx, gestor, domain.Regra{Nome: "Contagem", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade}); err == nil {
		t.Fatal("contagem")
	}
	falho = application.NewService(c.pool, falhando("InserirRegra"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.CriarRegra(ctx, gestor, domain.Regra{Nome: "Gravação", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade}); err == nil {
		t.Fatal("gravação da regra")
	}
	falho = application.NewService(c.pool, falhando("ListarRegras"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.ListarRegras(ctx, gestor); err == nil {
		t.Fatal("lista de regras")
	}
	falho = application.NewService(c.pool, falhando("ObterResposta"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.ObterResposta(ctx, c.tecnico, c.oferta.ID); err == nil {
		t.Fatal("leitura da resposta")
	}
	falho = application.NewService(c.pool, falhando("SalvarResposta"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.SalvarResposta(ctx, gestor, c.oferta.ID, "oi", ""); err == nil {
		t.Fatal("gravação da resposta")
	}
	falho = application.NewService(c.pool, falhando("ExcluirResposta"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.SalvarResposta(ctx, gestor, c.oferta.ID, " ", " "); err == nil {
		t.Fatal("exclusão da resposta")
	}
	falho = application.NewService(c.pool, falhando("InserirRecorrencia"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Backup", Dia: 2}); err == nil {
		t.Fatal("gravação da recorrência")
	}
	falho = application.NewService(c.pool, falhando("ListarRecorrencias"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.ListarRecorrencias(ctx, gestor); err == nil {
		t.Fatal("lista de recorrências")
	}
	falho = application.NewService(c.pool, falhando("ObterRecorrencia"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.ExcluirRecorrencia(ctx, gestor, uuid.New()); err == nil {
		t.Fatal("leitura da recorrência")
	}
	falho = application.NewService(c.pool, falhando("RegistrarDeflexao"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.RegistrarDeflexao(ctx, gestor, []string{"senha"}, false); err == nil {
		t.Fatal("gravação da deflexão")
	}
	falho = application.NewService(c.pool, falhando("GetOferta"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Backup", Dia: 2}); err == nil {
		t.Fatal("oferta da recorrência")
	}
	evento := application.NewService(c.pool, c.baseRepo(), eventosFalham{tipo: "atendimento.regra.criada"}, nil).WithLogger(logger)
	if _, err := evento.CriarRegra(ctx, gestor, domain.Regra{Nome: "Evento", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade}); err == nil {
		t.Fatal("evento da regra")
	}
	falho = application.NewService(c.pool, falhando("ExcluirRegra"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.ExcluirRegra(ctx, gestor, regra.ID); err == nil {
		t.Fatal("exclusão da regra")
	}
	evento = application.NewService(c.pool, c.baseRepo(), eventosFalham{tipo: "atendimento.regra.excluida"}, nil).WithLogger(logger)
	if err := evento.ExcluirRegra(ctx, gestor, regra.ID); err == nil {
		t.Fatal("evento da exclusão")
	}
	falho = application.NewService(c.pool, falhando("GetOferta"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.CriarRegra(ctx, gestor, domain.Regra{Nome: "Oferta", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade, OfertaID: &c.oferta.ID}); err == nil {
		t.Fatal("oferta da regra")
	}
	var equipe uuid.UUID
	if err := c.pool.QueryRow(ctx, `SELECT id FROM atendimento_equipes WHERE unidade_id = $1`, c.unidade).Scan(&equipe); err != nil {
		t.Fatal(err)
	}
	eqOutra, err := c.base.CriarEquipe(ctx, gestor, application.EquipeEntrada{Nome: "N2", UnidadeID: &outra, Membros: []uuid.UUID{c.tecnico.UserID}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, eqOutra.ID)
	})
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "Equipe de fora", Prioridade: domain.PrioridadeBaixa, UnidadeID: &c.unidade, EquipeID: &eqOutra.ID}); err == nil {
		t.Fatal("equipe de outra unidade")
	}
	if _, err := c.base.CriarRegra(ctx, gestor, domain.Regra{Nome: "Impressora", Palavra: "impressora", EquipeID: &equipe, UnidadeID: &c.unidade}); err != nil {
		t.Fatal(err)
	}
	falho = application.NewService(c.pool, falhando("GetEquipe"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "A impressora parou"}); err == nil {
		t.Fatal("equipe da abertura")
	}
	falho = application.NewService(c.pool, falhando("ListarRegrasDaUnidade"), eventosFalham{}, nil).WithLogger(logger)
	if _, err := falho.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Pedido"}); err == nil {
		t.Fatal("regras da abertura")
	}
	comEscopo := c.tecnico
	comEscopo.Scopes = append(comEscopo.Scopes, auth.Scope{}, comEscopo.Scopes[0])
	if err := c.base.RegistrarDeflexao(ctx, comEscopo, []string{"artigo-fila"}, true); err != nil {
		t.Fatal(err)
	}

	c.comPolitica(t)
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_politicas SET fuso = 'Nao/Existe' WHERE unidade_id = $1`, c.unidade); err != nil {
		t.Fatal(err)
	}
	marca, err := c.base.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Backup do mês", Dia: 15})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_recorrencias WHERE id = $1`, marca.ID)
	})
	c.base.WithRelogio(func() time.Time { return time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC) })
	if err := c.base.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Backup do mês"}); err != nil {
		t.Fatal(err)
	}
	c.base.WithRelogio(func() time.Time { return time.Date(2026, 1, 20, 15, 0, 0, 0, time.UTC) })
	if err := c.base.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.base.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	aviso := application.NewService(c.pool, falhando("ReivindicarAbertura"), eventosFalham{}, nil).WithLogger(logger)
	if err := aviso.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	falho = application.NewService(c.pool, falhando("ExcluirRecorrencia"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.ExcluirRecorrencia(ctx, gestor, marca.ID); err == nil {
		t.Fatal("exclusão da recorrência")
	}
	if _, err := c.base.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{OfertaID: c.oferta.ID, Titulo: "Conferência", Dia: 1}); err != nil {
		t.Fatal(err)
	}
	falho = application.NewService(c.pool, falhando("SolicitacaoDoMes"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	falho = application.NewService(c.pool, falhando("ParaEncerrar"), eventosFalham{}, nil).WithLogger(logger)
	if err := falho.CuidarDaFila(ctx); err == nil {
		t.Fatal("fila para encerrar")
	}
	falho = application.NewService(c.pool, falhando("Paradas"), eventosFalham{}, nil).WithLogger(logger).
		WithNotificacoes(notificacoes.NewService(c.pool, nil, logger))
	if err := falho.CuidarDaFila(ctx); err == nil {
		t.Fatal("paradas")
	}
	parado, cancelarItem := context.WithCancel(ctx)
	cancelarItem()
	if err := c.base.AbrirRecorrencias(parado); err == nil {
		t.Fatal("contexto cancelado")
	}
	if err := c.base.ExcluirRecorrencia(ctx, gestor, marca.ID); err != nil {
		t.Fatal(err)
	}

	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := c.base.RunRecorrencias(cancelado); err != nil {
		t.Fatal(err)
	}

	filaHTTP(t, c, gestor)
}

func (c cenario) baseRepo() domain.Repository {
	return falhando().Repository
}

func filaHTTP(t *testing.T, c cenario, gestor auth.Identity) {
	t.Helper()
	h := transport.NewHandlers(c.base, slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	h.ListarRegras(rec, pedidoFila(http.MethodGet, "", &gestor, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("listar regras: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.CriarRegra(rec, pedidoFila(http.MethodPost, `{`, &gestor, nil))
	if rec.Code < 400 {
		t.Fatalf("json da regra: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ExcluirRegra(rec, pedidoFila(http.MethodDelete, "", &gestor, map[string]string{"id": "nao"}))
	if rec.Code < 400 {
		t.Fatalf("id da regra: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ObterResposta(rec, pedidoFila(http.MethodGet, "", &c.tecnico, map[string]string{"id": "nao"}))
	if rec.Code < 400 {
		t.Fatalf("id da resposta: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.SalvarResposta(rec, pedidoFila(http.MethodPut, `{`, &gestor, map[string]string{"id": c.oferta.ID.String()}))
	if rec.Code < 400 {
		t.Fatalf("json da resposta: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ListarRecorrencias(rec, pedidoFila(http.MethodGet, "", &gestor, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("listar recorrências: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.CriarRecorrencia(rec, pedidoFila(http.MethodPost, `{`, &gestor, nil))
	if rec.Code < 400 {
		t.Fatalf("json da recorrência: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ExcluirRecorrencia(rec, pedidoFila(http.MethodDelete, "", &gestor, map[string]string{"id": "nao"}))
	if rec.Code < 400 {
		t.Fatalf("id da recorrência: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.RegistrarDeflexao(rec, pedidoFila(http.MethodPost, `{`, &gestor, nil))
	if rec.Code < 400 {
		t.Fatalf("json da deflexão: %d", rec.Code)
	}

	corpo := `{"nome":"Pela HTTP","prioridade":"baixa","unidade_id":"` + c.unidade.String() + `"}`
	rec = httptest.NewRecorder()
	h.CriarRegra(rec, pedidoFila(http.MethodPost, corpo, &gestor, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar regra: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.SalvarResposta(rec, pedidoFila(http.MethodPut, `{"publico":"Reinicie.","interna":""}`, &gestor, map[string]string{"id": c.oferta.ID.String()}))
	if rec.Code != http.StatusOK {
		t.Fatalf("salvar resposta: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ObterResposta(rec, pedidoFila(http.MethodGet, "", &c.tecnico, map[string]string{"id": c.oferta.ID.String()}))
	if rec.Code != http.StatusOK {
		t.Fatalf("obter resposta: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.CriarRecorrencia(rec, pedidoFila(http.MethodPost, `{"oferta_id":"`+c.oferta.ID.String()+`","titulo":"Pela HTTP","dia":4}`, &gestor, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar recorrência: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.RegistrarDeflexao(rec, pedidoFila(http.MethodPost, `{"artigos":["http"],"resolveu":true}`, &gestor, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deflexão: %d %s", rec.Code, rec.Body.String())
	}
	var regraID, recorrenciaID uuid.UUID
	if err := c.pool.QueryRow(context.Background(), `SELECT id FROM atendimento_regras WHERE unidade_id = $1 LIMIT 1`, c.unidade).Scan(&regraID); err != nil {
		t.Fatal(err)
	}
	if err := c.pool.QueryRow(context.Background(), `SELECT id FROM atendimento_recorrencias WHERE oferta_id = $1 LIMIT 1`, c.oferta.ID).Scan(&recorrenciaID); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	h.ExcluirRegra(rec, pedidoFila(http.MethodDelete, "", &gestor, map[string]string{"id": regraID.String()}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("excluir regra: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ExcluirRecorrencia(rec, pedidoFila(http.MethodDelete, "", &gestor, map[string]string{"id": recorrenciaID.String()}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("excluir recorrência: %d %s", rec.Code, rec.Body.String())
	}
}

func pedidoFila(method, body string, idt *auth.Identity, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rota := chi.NewRouteContext()
	for chave, valor := range params {
		rota.URLParams.Add(chave, valor)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rota)
	if idt != nil {
		ctx = auth.WithIdentity(ctx, *idt)
	}
	return req.WithContext(ctx)
}
