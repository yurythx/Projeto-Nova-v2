package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/domain/pagination"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

func (f falhaRepo) SetSegredo(ctx context.Context, db database.DBTX, id uuid.UUID, segredo *uuid.UUID, agora time.Time) error {
	if f.falha["SetSegredo"] {
		return errFalha
	}
	return f.Repository.SetSegredo(ctx, db, id, segredo, agora)
}

func (f falhaRepo) SetAtivo(ctx context.Context, db database.DBTX, id uuid.UUID, ativo *uuid.UUID, nome string, agora time.Time) error {
	if f.falha["SetAtivo"] {
		return errFalha
	}
	return f.Repository.SetAtivo(ctx, db, id, ativo, nome, agora)
}

func (f falhaRepo) SetIncidentePai(ctx context.Context, db database.DBTX, id uuid.UUID, pai *uuid.UUID, agora time.Time) error {
	if f.falha["SetIncidentePai"] {
		return errFalha
	}
	return f.Repository.SetIncidentePai(ctx, db, id, pai, agora)
}

func (f falhaRepo) InsertSLA(ctx context.Context, db database.DBTX, s domain.SLA) (domain.SLA, error) {
	if f.falha["InsertSLA"] {
		return domain.SLA{}, errFalha
	}
	return f.Repository.InsertSLA(ctx, db, s)
}

func (f falhaRepo) UpdateSLA(ctx context.Context, db database.DBTX, s domain.SLA) (domain.SLA, error) {
	if f.falha["UpdateSLA"] {
		return domain.SLA{}, errFalha
	}
	return f.Repository.UpdateSLA(ctx, db, s)
}

func (f falhaRepo) DeleteSLA(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["DeleteSLA"] {
		return errFalha
	}
	return f.Repository.DeleteSLA(ctx, db, id)
}

func (f falhaRepo) GetSLA(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.SLA, error) {
	if f.falha["GetSLA"] {
		return domain.SLA{}, errFalha
	}
	return f.Repository.GetSLA(ctx, db, id)
}

func (f falhaRepo) ListSLAs(ctx context.Context, db database.DBTX, u *uuid.UUID) ([]domain.SLA, error) {
	if f.falha["ListSLAs"] {
		return nil, errFalha
	}
	return f.Repository.ListSLAs(ctx, db, u)
}

func (f falhaRepo) VincularCategoriaSLA(ctx context.Context, db database.DBTX, categoria uuid.UUID, sla *uuid.UUID) error {
	if f.falha["VincularCategoriaSLA"] {
		return errFalha
	}
	return f.Repository.VincularCategoriaSLA(ctx, db, categoria, sla)
}

func (f falhaRepo) GetCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Categoria, error) {
	if f.falha["GetCategoria"] {
		return domain.Categoria{}, errFalha
	}
	return f.Repository.GetCategoria(ctx, db, id)
}

func (f falhaRepo) UnidadesComSolicitacao(ctx context.Context, db database.DBTX) ([]uuid.UUID, error) {
	if f.falha["UnidadesComSolicitacao"] {
		return nil, errFalha
	}
	return f.Repository.UnidadesComSolicitacao(ctx, db)
}

func (f falhaRepo) GravarRetrato(ctx context.Context, db database.DBTX, r domain.Retrato) error {
	if f.falha["GravarRetrato"] {
		return errFalha
	}
	return f.Repository.GravarRetrato(ctx, db, r)
}

func (f falhaRepo) ListarRetratos(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]domain.Retrato, error) {
	if f.falha["ListarRetratos"] {
		return nil, errFalha
	}
	return f.Repository.ListarRetratos(ctx, db, desde, restrito, unidades)
}

// filaCtl devolve só os chamados do teste. A consulta global da fila
// pega até 40 registros antigos e não garante que o cenário entre.
type filaCtl struct {
	falhaRepo
	encerrar, paradas       []domain.Solicitacao
	estado, tipo            string
	espelho, importadoFalha bool
}

func (f filaCtl) ParaEncerrar(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if f.falha["ParaEncerrar"] {
		return nil, errFalha
	}
	return f.encerrar, nil
}

func (f filaCtl) Paradas(ctx context.Context, db database.DBTX, ate time.Time, limite int) ([]domain.Solicitacao, error) {
	if f.falha["Paradas"] {
		return nil, errFalha
	}
	return f.paradas, nil
}

func (f filaCtl) GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Solicitacao, error) {
	if f.falha["GetSolicitacao"] {
		return domain.Solicitacao{}, errFalha
	}
	s, err := f.Repository.GetSolicitacao(ctx, db, id)
	if err != nil {
		return s, err
	}
	if f.estado != "" {
		s.Estado = f.estado
	}
	if f.tipo != "" {
		s.Tipo = f.tipo
	}
	return s, nil
}

func (f filaCtl) Importado(ctx context.Context, db database.DBTX, id uuid.UUID) (bool, error) {
	if f.importadoFalha {
		return false, errFalha
	}
	if f.espelho {
		return true, nil
	}
	return f.Repository.Importado(ctx, db, id)
}

func (f filaCtl) servico(c cenario, ev application.Eventos) *application.Service {
	if ev == nil {
		ev = eventosFalham{}
	}
	return application.NewService(c.pool, f, ev, nil).WithLogger(slog.New(slog.DiscardHandler))
}

func (c cenario) resolvida(t *testing.T) domain.Solicitacao {
	t.Helper()
	s := c.emAtendimento(t)
	d, err := c.base.Transicionar(context.Background(), c.tecnico, s.ID, domain.EstadoResolvido)
	if err != nil {
		t.Fatal(err)
	}
	return d.Solicitacao
}

func gestorDo(c cenario) auth.Identity {
	g := admin()
	g.UserID = c.servidor.UserID
	return g
}

type mapaRuim struct{}

func (mapaRuim) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	return nil, errFalha
}

type cofreDesligado struct{}

func (cofreDesligado) Nome(context.Context, auth.Identity, uuid.UUID) (string, error) {
	return "", domain.ErrCofreDesligado
}

// getNaVez falha só na leitura indicada. A alteração lê o chamado e,
// no vínculo de pai, lê o pai em seguida.
type getNaVez struct {
	falhaRepo
	n, falhaNa int
}

func (g *getNaVez) GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Solicitacao, error) {
	g.n++
	if g.n == g.falhaNa {
		return domain.Solicitacao{}, errFalha
	}
	return g.Repository.GetSolicitacao(ctx, db, id)
}

// listaNaVez falha só a consulta indicada. A busca lê primeiro as
// próprias e, se ainda couber, a fila de quem atende.
type listaNaVez struct {
	falhaRepo
	n, falhaNa int
}

func (l *listaNaVez) ListSolicitacoes(ctx context.Context, db database.DBTX, f domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	l.n++
	if l.n == l.falhaNa {
		return nil, 0, errFalha
	}
	return l.Repository.ListSolicitacoes(ctx, db, f)
}

type governancaFixa struct {
	falhaRepo
	lista []domain.Solicitacao
}

func (g governancaFixa) ListarParaGovernanca(context.Context, database.DBTX, bool, []uuid.UUID, int) ([]domain.Solicitacao, error) {
	if g.falha["ListarParaGovernanca"] {
		return nil, errFalha
	}
	return g.lista, nil
}

func TestRamosDoCuidadoDaFila(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	calado := slog.New(slog.DiscardHandler)

	engole := func(t *testing.T, f filaCtl, ev application.Eventos) {
		t.Helper()
		if err := f.servico(c, ev).WithLogger(calado).CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("trava", func(t *testing.T) {
		sol := c.resolvida(t)
		f := filaCtl{falhaRepo: falhando("TravarSolicitacao"), encerrar: []domain.Solicitacao{sol}}
		engole(t, f, nil)
		engole(t, f, nil)
		if err := f.servico(c, nil).WithLogger(nil).CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("leitura", func(t *testing.T) {
		sol := c.resolvida(t)
		engole(t, filaCtl{falhaRepo: falhando("GetSolicitacao"), encerrar: []domain.Solicitacao{sol}}, nil)
	})

	t.Run("ja andou", func(t *testing.T) {
		sol := c.resolvida(t)
		engole(t, filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}, estado: domain.EstadoAberto}, nil)
		if atual, err := c.base.Obter(ctx, c.tecnico, sol.ID); err != nil || atual.Solicitacao.Estado != domain.EstadoResolvido {
			t.Fatalf("não devia encerrar: %+v %v", atual.Solicitacao.Estado, err)
		}
	})

	t.Run("espelho", func(t *testing.T) {
		sol := c.resolvida(t)
		f := filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}, espelho: true}
		svc := f.servico(c, nil).WithEspelho(func(context.Context) (bool, error) { return true, nil })
		if err := svc.CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
		f.importadoFalha = true
		f.espelho = false
		svc = f.servico(c, nil).WithEspelho(func(context.Context) (bool, error) { return false, errFalha })
		if err := svc.CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("transicao", func(t *testing.T) {
		sol := c.resolvida(t)
		engole(t, filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}, tipo: "x"}, nil)
	})

	t.Run("gravacao", func(t *testing.T) {
		sol := c.resolvida(t)
		for _, metodo := range []string{"UpdateSolicitacao", "InsertPassagem", "InsertComentario"} {
			engole(t, filaCtl{falhaRepo: falhando(metodo), encerrar: []domain.Solicitacao{sol}}, nil)
		}
		engole(t, filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}}, eventosFalham{application.EventoSolicitacaoEstado})
	})

	t.Run("encerra sem avisar o antigo", func(t *testing.T) {
		sol := c.resolvida(t)
		if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET resolvida_em = $2 WHERE id = $1`, sol.ID, time.Now().Add(-40*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
		f := filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}}
		svc := f.servico(c, nil).WithNotificacoes(&notificacoes.Service{})
		if err := svc.CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
		if atual, err := c.base.Obter(ctx, c.tecnico, sol.ID); err != nil || atual.Solicitacao.Estado != domain.EstadoEncerrado {
			t.Fatalf("encerrado: %+v %v", atual.Solicitacao.Estado, err)
		}
	})

	t.Run("encerra sem relógio de conclusão", func(t *testing.T) {
		sol := c.resolvida(t)
		if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET resolvida_em = NULL WHERE id = $1`, sol.ID); err != nil {
			t.Fatal(err)
		}
		engole(t, filaCtl{falhaRepo: falhando(), encerrar: []domain.Solicitacao{sol}}, nil)
		if atual, err := c.base.Obter(ctx, c.tecnico, sol.ID); err != nil || atual.Solicitacao.Estado != domain.EstadoEncerrado {
			t.Fatalf("encerrado: %+v %v", atual.Solicitacao.Estado, err)
		}
	})

	t.Run("parada sem destino", func(t *testing.T) {
		parada := domain.Solicitacao{ID: uuid.New(), UnidadeID: &c.unidade, Estado: domain.EstadoAguardandoTerceiro, Titulo: "Parada", OfertaNome: "E-mail", Numero: 1}
		f := filaCtl{falhaRepo: falhando("Responsaveis"), paradas: []domain.Solicitacao{parada}}
		svc := f.servico(c, nil).WithNotificacoes(&notificacoes.Service{})
		if err := svc.CuidarDaFila(ctx); !errors.Is(err, errFalha) {
			t.Fatalf("destinos: %v", err)
		}
		if err := f.servico(c, nil).CuidarDaFila(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("consulta da fila", func(t *testing.T) {
		f := filaCtl{falhaRepo: falhando("ParaEncerrar")}
		if err := f.servico(c, nil).CuidarDaFila(ctx); !errors.Is(err, errFalha) {
			t.Fatalf("para encerrar: %v", err)
		}
		f = filaCtl{falhaRepo: falhando("Paradas")}
		svc := f.servico(c, nil).WithNotificacoes(&notificacoes.Service{})
		if err := svc.CuidarDaFila(ctx); !errors.Is(err, errFalha) {
			t.Fatalf("paradas: %v", err)
		}
	})
}

func TestRamosDaSerie(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.resolvida(t)
	if _, err := c.base.Avaliar(ctx, c.servidor, sol.ID, 4, "rápido"); err != nil {
		t.Fatal(err)
	}

	if vazio, err := c.base.Serie(ctx, auth.Identity{UserID: c.servidor.UserID}, 30); err != nil || len(vazio) != 0 {
		t.Fatalf("sem cobertura: %+v %v", vazio, err)
	}
	if _, err := c.base.Serie(ctx, c.tecnico, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.Serie(ctx, c.tecnico, 400); err != nil {
		t.Fatal(err)
	}
	if _, err := c.com(falhando("ListarRetratos"), eventosFalham{}).Serie(ctx, c.tecnico, 7); err == nil {
		t.Fatal("retratos")
	}
	for _, marca := range []string{"PoliticaIlegivel", "FusoVazio", "ResolverPolitica"} {
		if _, err := c.com(falhando(marca), eventosFalham{}).Serie(ctx, admin(), 3); marca == "ResolverPolitica" && err != nil {
			t.Fatal(err)
		} else if marca != "ResolverPolitica" && err != nil {
			t.Fatal(err)
		}
	}

	if err := c.com(falhando("UnidadesComSolicitacao"), eventosFalham{}).RegistrarSerie(ctx); err == nil {
		t.Fatal("unidades")
	}

	antes := application.TetoGovernanca
	application.TetoGovernanca = 0
	t.Cleanup(func() { application.TetoGovernanca = antes })
	if err := c.base.RegistrarSerie(ctx); err != nil {
		t.Fatal(err)
	}
	application.TetoGovernanca = antes

	var csat *float64
	if err := c.pool.QueryRow(ctx, `SELECT csat_media FROM atendimento_indicadores_dia WHERE unidade_id = $1 ORDER BY dia DESC LIMIT 1`, c.unidade).Scan(&csat); err != nil || csat == nil || *csat != 4 {
		t.Fatalf("csat: %v %v", csat, err)
	}
	if pontos, err := c.base.Serie(ctx, admin(), 5); err != nil || len(pontos) == 0 {
		t.Fatalf("institucional: %+v %v", pontos, err)
	}

	prioridade := domain.PrioridadeMedia
	comPrazo := governancaFixa{falhaRepo: falhando("PassagensDe"), lista: []domain.Solicitacao{{ID: sol.ID, Prioridade: &prioridade}}}
	if err := c.com(comPrazo, eventosFalham{}).RegistrarSerie(ctx); err == nil {
		t.Fatal("medir")
	}
	concluida := governancaFixa{falhaRepo: falhando("PassagensDe"), lista: []domain.Solicitacao{{ID: sol.ID, Estado: domain.EstadoResolvido}}}
	if err := c.com(concluida, eventosFalham{}).RegistrarSerie(ctx); err == nil {
		t.Fatal("passagens")
	}
	if err := c.com(falhando("ResumoAvaliacoes"), eventosFalham{}).RegistrarSerie(ctx); err == nil {
		t.Fatal("avaliações")
	}

	ctxCancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := c.base.RunSerie(ctxCancelado); err != nil {
		t.Fatal(err)
	}

	svc := c.com(falhando("UnidadesComSolicitacao"), eventosFalham{}).WithLogger(slog.New(slog.DiscardHandler))
	vivo, parar := context.WithCancel(ctx)
	saiu := make(chan struct{})
	go func() {
		_ = svc.RunSerie(vivo)
		close(saiu)
	}()
	time.Sleep(150 * time.Millisecond)
	parar()
	select {
	case <-saiu:
	case <-time.After(3 * time.Second):
		t.Fatal("série não parou")
	}
}

func TestRamosDaSLA(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	gestor := gestorDo(c)
	entrada := application.SLAEntrada{
		Nome: "Prazo " + uuid.NewString()[:6], TempoRespostaMinutos: 15, TempoSolucaoMinutos: 60,
		CalendarioTipo: domain.CalendarioExpediente, UnidadeID: &c.unidade,
	}

	if _, err := c.base.ListarSLAs(ctx, c.servidor, nil); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("lista: %v", err)
	}
	if _, err := c.com(falhando("ListSLAs"), eventosFalham{}).ListarSLAs(ctx, gestor, &c.unidade); err == nil {
		t.Fatal("consulta")
	}
	if _, err := c.base.ListarSLAs(ctx, gestor, &c.unidade); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.ObterSLA(ctx, gestor, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("ausente: %v", err)
	}

	sla, err := c.base.CriarSLA(ctx, gestor, entrada)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_slas WHERE id = $1`, sla.ID)
	})
	if _, err := c.base.ObterSLA(ctx, c.servidor, sla.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("obter: %v", err)
	}
	if _, err := c.base.CriarSLA(ctx, c.servidor, entrada); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("criar: %v", err)
	}
	if _, err := c.base.CriarSLA(ctx, gestor, application.SLAEntrada{Nome: " ", TempoRespostaMinutos: 1, TempoSolucaoMinutos: 1}); !errors.Is(err, domain.ErrSLAInvalida) {
		t.Fatalf("nome: %v", err)
	}
	if _, err := c.com(falhando("InsertSLA"), eventosFalham{}).CriarSLA(ctx, gestor, entrada); err == nil {
		t.Fatal("insert")
	}
	if _, err := c.com(falhando(), eventosFalham{application.EventoSlasCriada}).CriarSLA(ctx, gestor, entrada); err == nil {
		t.Fatal("evento de criação")
	}

	if _, err := c.base.AtualizarSLA(ctx, gestor, uuid.New(), entrada); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("atualizar ausente: %v", err)
	}
	if _, err := c.base.AtualizarSLA(ctx, c.servidor, sla.ID, entrada); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("atualizar: %v", err)
	}
	if _, err := c.base.AtualizarSLA(ctx, gestor, sla.ID, application.SLAEntrada{Nome: " ", TempoRespostaMinutos: 1, TempoSolucaoMinutos: 1}); !errors.Is(err, domain.ErrSLAInvalida) {
		t.Fatalf("atualizar nome: %v", err)
	}
	if _, err := c.com(falhando("UpdateSLA"), eventosFalham{}).AtualizarSLA(ctx, gestor, sla.ID, entrada); err == nil {
		t.Fatal("update")
	}
	if _, err := c.com(falhando(), eventosFalham{application.EventoSlasAtualizada}).AtualizarSLA(ctx, gestor, sla.ID, entrada); err == nil {
		t.Fatal("evento de atualização")
	}

	if err := c.base.ExcluirSLA(ctx, gestor, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("excluir ausente: %v", err)
	}
	if err := c.base.ExcluirSLA(ctx, c.servidor, sla.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("excluir: %v", err)
	}
	if err := c.com(falhando("DeleteSLA"), eventosFalham{}).ExcluirSLA(ctx, gestor, sla.ID); err == nil {
		t.Fatal("delete")
	}
	if err := c.com(falhando(), eventosFalham{application.EventoSlasExcluida}).ExcluirSLA(ctx, gestor, sla.ID); err == nil {
		t.Fatal("evento de exclusão")
	}

	if err := c.base.AssociarCategoriaSLA(ctx, c.servidor, uuid.New(), nil); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("associar: %v", err)
	}
	if err := c.base.AssociarCategoriaSLA(ctx, gestor, uuid.New(), nil); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("categoria: %v", err)
	}
	cat, err := c.base.CriarCategoria(ctx, gestor, application.CategoriaEntrada{Nome: "Bloco " + uuid.NewString()[:6]})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_categorias WHERE id = $1`, cat.ID)
	})
	ausente := uuid.New()
	if err := c.base.AssociarCategoriaSLA(ctx, gestor, cat.ID, &ausente); !errors.Is(err, domain.ErrSLANaoEncontrada) {
		t.Fatalf("sla: %v", err)
	}
	if err := c.com(falhando("VincularCategoriaSLA"), eventosFalham{}).AssociarCategoriaSLA(ctx, gestor, cat.ID, &sla.ID); err == nil {
		t.Fatal("vínculo")
	}
	if err := c.com(falhando(), eventosFalham{application.EventoCategoriaSLA}).AssociarCategoriaSLA(ctx, gestor, cat.ID, nil); err == nil {
		t.Fatal("evento do vínculo")
	}

	h := transport.NewHandlers(c.base, slog.New(slog.DiscardHandler))
	corpo := `{"nome":"Prazo HTTP","tempo_resposta_minutos":10,"tempo_solucao_minutos":40,"calendario_tipo":"expediente"}`
	if codigo(t, h.ListarSLAs, pedidoQuery(http.MethodGet, "", &gestor, nil, "unidade_id=nao")) < 400 {
		t.Fatal("unidade")
	}
	if codigo(t, h.ListarSLAs, pedidoFila(http.MethodGet, "", &gestor, nil)) != http.StatusOK {
		t.Fatal("listar")
	}
	if codigo(t, h.CriarSLA, pedidoJSON(http.MethodPost, "{", &gestor, nil)) < 400 {
		t.Fatal("json")
	}
	if codigo(t, h.CriarSLA, pedidoJSON(http.MethodPost, corpo, &gestor, nil)) != http.StatusCreated {
		t.Fatal("criar http")
	}
	if codigo(t, h.ObterSLA, pedidoFila(http.MethodGet, "", &gestor, map[string]string{"id": "nao"})) < 400 {
		t.Fatal("id")
	}
	if codigo(t, h.ObterSLA, pedidoFila(http.MethodGet, "", &gestor, map[string]string{"id": sla.ID.String()})) != http.StatusOK {
		t.Fatal("obter http")
	}
	if codigo(t, h.AtualizarSLA, pedidoJSON(http.MethodPut, corpo, &gestor, map[string]string{"id": sla.ID.String()})) != http.StatusOK {
		t.Fatal("atualizar http")
	}
	if codigo(t, h.VincularCategoriaSLA, pedidoJSON(http.MethodPut, `{"sla_id":"`+sla.ID.String()+`"}`, &gestor, map[string]string{"id": cat.ID.String()})) != http.StatusOK {
		t.Fatal("vincular http")
	}
	if codigo(t, h.ExcluirSLA, pedidoFila(http.MethodDelete, "", &gestor, map[string]string{"id": sla.ID.String()})) != http.StatusNoContent {
		t.Fatal("excluir http")
	}
}

func TestRamosDasSolicitacoes(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	pg := pagination.New(1, 20, 100)
	sol := c.abrir(t)
	segredo := uuid.New()
	ativo := uuid.New()

	campos := []domain.Campo{{Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true}}
	comCampo, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{
		Nome: "Com campo " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade, Campos: &campos,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, comCampo.ID)
	})
	if _, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: comCampo.ID, Titulo: "Sem ramal"}); err == nil {
		t.Fatal("resposta obrigatória")
	}

	svc := c.com(falhando(), eventosFalham{})
	svc.ComPassagens(mapaRuim{})
	if _, _, err := svc.ListarFila(ctx, c.tecnico, application.FiltroFila{}, pg); err == nil {
		t.Fatal("mapa da fila")
	}
	if _, err := svc.Obter(ctx, c.tecnico, sol.ID); err == nil {
		t.Fatal("mapa do detalhe")
	}
	if _, err := svc.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoEmAtendimento); err == nil {
		t.Fatal("mapa da passagem")
	}

	if _, err := c.base.VincularSegredo(ctx, c.servidor, sol.ID, &segredo); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("segredo do servidor: %v", err)
	}
	if _, err := c.base.VincularAtivo(ctx, c.servidor, sol.ID, &ativo); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("ativo do servidor: %v", err)
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.servidor, sol.ID, &sol.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("pai do servidor: %v", err)
	}

	cancelada := c.abrir(t)
	if _, err := c.base.Transicionar(ctx, c.tecnico, cancelada.ID, domain.EstadoCancelado); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularSegredo(ctx, c.tecnico, cancelada.ID, nil); !errors.Is(err, domain.ErrSolicitacaoFinalizada) {
		t.Fatalf("segredo finalizado: %v", err)
	}
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, cancelada.ID, nil); !errors.Is(err, domain.ErrSolicitacaoFinalizada) {
		t.Fatalf("ativo finalizado: %v", err)
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, cancelada.ID, nil); !errors.Is(err, domain.ErrSolicitacaoFinalizada) {
		t.Fatalf("pai finalizado: %v", err)
	}

	cofre := c.com(falhando(), eventosFalham{}).WithSegredo(cofreDesligado{})
	if _, err := cofre.VincularSegredo(ctx, c.tecnico, sol.ID, &segredo); !errors.Is(err, domain.ErrCofreDesligado) {
		t.Fatalf("cofre: %v", err)
	}
	marcado := c.com(falhando(), eventosFalham{}).WithSegredo(cofreOk{}).WithRelogio(func() time.Time {
		return time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	})
	if _, err := marcado.VincularSegredo(ctx, c.tecnico, sol.ID, &segredo); err != nil {
		t.Fatal(err)
	}
	if _, err := c.com(falhando("SetSegredo"), eventosFalham{}).WithSegredo(cofreOk{}).VincularSegredo(ctx, c.tecnico, sol.ID, &segredo); err == nil {
		t.Fatal("gravar segredo")
	}
	if _, err := c.com(falhando(), eventosFalham{application.EventoSolicitacaoSegredo}).WithSegredo(cofreOk{}).VincularSegredo(ctx, c.tecnico, sol.ID, &segredo); err == nil {
		t.Fatal("evento do segredo")
	}

	if _, err := c.com(falhando("SetAtivo"), eventosFalham{}).WithAtivos(ativosOk{unidade: &c.unidade}).VincularAtivo(ctx, c.tecnico, sol.ID, &ativo); err == nil {
		t.Fatal("gravar ativo")
	}

	pai := c.abrir(t)
	filho := c.abrir(t)
	neto := c.abrir(t)
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, filho.ID, &pai.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, neto.ID, &filho.ID); !errors.Is(err, domain.ErrIncidentePaiInvalido) {
		t.Fatalf("pai que já é filho: %v", err)
	}
	leitura := &getNaVez{falhaRepo: falhando(), falhaNa: 2}
	if _, err := c.com(leitura, eventosFalham{}).VincularIncidentePai(ctx, c.tecnico, neto.ID, &pai.ID); err == nil {
		t.Fatal("leitura do pai")
	}
	alheio := auth.Identity{UserID: dbtest.User(t, c.pool)}
	if _, err := c.base.ListarFilhosDoIncidente(ctx, alheio, pai.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("filhos: %v", err)
	}

	busca := &listaNaVez{falhaRepo: falhando(), falhaNa: 2}
	if _, err := c.com(busca, eventosFalham{}).Search(ctx, c.tecnico, "Pedido", 5); err == nil {
		t.Fatal("segunda lista")
	}

	repo := infrastructure.NewRepository()
	if _, _, err := repo.ListSolicitacoes(ctx, c.pool, domain.FiltroSolicitacao{Quadro: true, Limit: 0}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.ListSolicitacoes(ctx, c.pool, domain.FiltroSolicitacao{Quadro: true, Limit: 9999}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET respostas = 'null' WHERE id = $1`, sol.ID); err != nil {
		t.Fatal(err)
	}
	if lida, err := repo.GetSolicitacao(ctx, c.pool, sol.ID); err != nil || lida.Respostas == nil {
		t.Fatalf("respostas nulas: %+v %v", lida.Respostas, err)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET respostas = '"x"' WHERE id = $1`, sol.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetSolicitacao(ctx, c.pool, sol.ID); err == nil {
		t.Fatal("resposta que não é objeto")
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET respostas = '{}'::jsonb WHERE id = $1`, sol.ID); err != nil {
		t.Fatal(err)
	}
	if email, err := repo.EmailDe(ctx, c.pool, uuid.New()); err != nil || email != "" {
		t.Fatalf("e-mail ausente: %q %v", email, err)
	}

	h := transport.NewHandlers(c.base.WithSegredo(cofreOk{}).WithAtivos(ativosOk{unidade: &c.unidade}), slog.New(slog.DiscardHandler))
	if codigo(t, h.VincularSegredo, pedidoFila(http.MethodPut, "", &c.tecnico, map[string]string{"id": "nao"})) < 400 {
		t.Fatal("segredo id")
	}
	if codigo(t, h.VincularSegredo, pedidoJSON(http.MethodPut, "{", &c.tecnico, map[string]string{"id": sol.ID.String()})) < 400 {
		t.Fatal("segredo json")
	}
	if codigo(t, h.VincularSegredo, pedidoJSON(http.MethodPut, `{"segredo_id":null}`, &c.tecnico, map[string]string{"id": sol.ID.String()})) != http.StatusOK {
		t.Fatal("segredo http")
	}
	if codigo(t, h.VincularAtivo, pedidoJSON(http.MethodPut, "{", &c.tecnico, map[string]string{"id": sol.ID.String()})) < 400 {
		t.Fatal("ativo json")
	}
	if codigo(t, h.VincularAtivo, pedidoJSON(http.MethodPut, `{"ativo_id":"`+ativo.String()+`"}`, &c.tecnico, map[string]string{"id": sol.ID.String()})) != http.StatusOK {
		t.Fatal("ativo http")
	}
	if codigo(t, h.Avaliar, pedidoFila(http.MethodPost, "", &c.servidor, map[string]string{"id": "nao"})) < 400 {
		t.Fatal("avaliar id")
	}
	if codigo(t, h.Avaliar, pedidoJSON(http.MethodPost, "{", &c.servidor, map[string]string{"id": sol.ID.String()})) < 400 {
		t.Fatal("avaliar json")
	}
	if codigo(t, h.Avaliar, pedidoJSON(http.MethodPost, `{"nota":4}`, &c.tecnico, map[string]string{"id": sol.ID.String()})) < 400 {
		t.Fatal("avaliar alheia")
	}
	if codigo(t, h.Serie, pedidoQuery(http.MethodGet, "", &c.tecnico, nil, "dias=abc")) != http.StatusOK {
		t.Fatal("série")
	}
	if codigo(t, h.Serie, pedidoQuery(http.MethodGet, "", &c.tecnico, nil, "dias=12")) != http.StatusOK {
		t.Fatal("série com dias")
	}
	ruim := transport.NewHandlers(c.com(falhando("ListarRetratos"), eventosFalham{}), slog.New(slog.DiscardHandler))
	if codigo(t, ruim.Serie, pedidoFila(http.MethodGet, "", &c.tecnico, nil)) < 400 {
		t.Fatal("série com falha")
	}
	if codigo(t, h.VincularIncidentePai, pedidoFila(http.MethodPut, "", &c.tecnico, map[string]string{"id": "nao"})) < 400 {
		t.Fatal("pai id")
	}
	if codigo(t, h.VincularIncidentePai, pedidoJSON(http.MethodPut, "{", &c.tecnico, map[string]string{"id": neto.ID.String()})) < 400 {
		t.Fatal("pai json")
	}
	if codigo(t, h.VincularIncidentePai, pedidoJSON(http.MethodPut, `{"incidente_pai_id":null}`, &c.tecnico, map[string]string{"id": filho.ID.String()})) != http.StatusOK {
		t.Fatal("pai http")
	}
	if codigo(t, h.ListarFilhosDoIncidente, pedidoFila(http.MethodGet, "", &c.tecnico, map[string]string{"id": "nao"})) < 400 {
		t.Fatal("filhos id")
	}
	if codigo(t, h.ListarFilhosDoIncidente, pedidoFila(http.MethodGet, "", &alheio, map[string]string{"id": pai.ID.String()})) < 400 {
		t.Fatal("filhos alheios")
	}
	if codigo(t, h.ListarFilhosDoIncidente, pedidoFila(http.MethodGet, "", &c.tecnico, map[string]string{"id": pai.ID.String()})) != http.StatusOK {
		t.Fatal("filhos http")
	}
}

func pedidoJSON(method, body string, idt *auth.Identity, params map[string]string) *http.Request {
	req := pedidoFila(method, body, idt, params)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func pedidoQuery(method, body string, idt *auth.Identity, params map[string]string, query string) *http.Request {
	req := pedidoFila(method, body, idt, params)
	req.URL.RawQuery = query
	return req
}

func codigo(t *testing.T, fn func(http.ResponseWriter, *http.Request), req *http.Request) int {
	t.Helper()
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec.Code
}
