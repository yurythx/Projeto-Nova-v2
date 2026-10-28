package application_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/domain/pagination"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

var errFalha = errors.New("falha injetada")

// falhaRepo delega ao repositório real e falha só nos métodos marcados:
// os caminhos de erro rodam dentro de transações de verdade.
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

func (f falhaRepo) GetOferta(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Oferta, error) {
	if f.falha["GetOferta"] {
		return domain.Oferta{}, errFalha
	}
	return f.Repository.GetOferta(ctx, db, id)
}
func (f falhaRepo) InsertSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) (domain.Solicitacao, error) {
	if f.falha["InsertSolicitacao"] {
		return domain.Solicitacao{}, errFalha
	}
	return f.Repository.InsertSolicitacao(ctx, db, s)
}
func (f falhaRepo) InsertPassagem(ctx context.Context, db database.DBTX, p domain.Passagem) (domain.Passagem, error) {
	if f.falha["InsertPassagem"] {
		return domain.Passagem{}, errFalha
	}
	return f.Repository.InsertPassagem(ctx, db, p)
}
func (f falhaRepo) Responsaveis(ctx context.Context, db database.DBTX, u *uuid.UUID) ([]domain.Candidato, error) {
	if f.falha["Responsaveis"] {
		return nil, errFalha
	}
	return f.Repository.Responsaveis(ctx, db, u)
}
func (f falhaRepo) ListSolicitacoes(ctx context.Context, db database.DBTX, fs domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	if f.falha["ListSolicitacoes"] {
		return nil, 0, errFalha
	}
	return f.Repository.ListSolicitacoes(ctx, db, fs)
}
func (f falhaRepo) ListPassagens(ctx context.Context, db database.DBTX, id uuid.UUID) ([]domain.Passagem, error) {
	if f.falha["ListPassagens"] {
		return nil, errFalha
	}
	return f.Repository.ListPassagens(ctx, db, id)
}
func (f falhaRepo) ListComentarios(ctx context.Context, db database.DBTX, id uuid.UUID, internos bool) ([]domain.Comentario, error) {
	if f.falha["ListComentarios"] {
		return nil, errFalha
	}
	return f.Repository.ListComentarios(ctx, db, id, internos)
}
func (f falhaRepo) ResponsavelValido(ctx context.Context, db database.DBTX, user uuid.UUID, u *uuid.UUID) (bool, error) {
	if f.falha["ResponsavelValido"] {
		return false, errFalha
	}
	return f.Repository.ResponsavelValido(ctx, db, user, u)
}
func (f falhaRepo) UpdateSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) error {
	if f.falha["UpdateSolicitacao"] {
		return errFalha
	}
	return f.Repository.UpdateSolicitacao(ctx, db, s)
}
func (f falhaRepo) InsertComentario(ctx context.Context, db database.DBTX, c domain.Comentario) (domain.Comentario, error) {
	if f.falha["InsertComentario"] {
		return domain.Comentario{}, errFalha
	}
	return f.Repository.InsertComentario(ctx, db, c)
}
func (f falhaRepo) TravarSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["TravarSolicitacao"] {
		return errFalha
	}
	return f.Repository.TravarSolicitacao(ctx, db, id)
}
func (f falhaRepo) GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Solicitacao, error) {
	if f.falha["GetSolicitacao"] {
		return domain.Solicitacao{}, errFalha
	}
	return f.Repository.GetSolicitacao(ctx, db, id)
}
func (f falhaRepo) GetAvaliacao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Avaliacao, error) {
	if f.falha["GetAvaliacao"] {
		return domain.Avaliacao{}, errFalha
	}
	return f.Repository.GetAvaliacao(ctx, db, id)
}
func (f falhaRepo) InsertAvaliacao(ctx context.Context, db database.DBTX, a domain.Avaliacao) (domain.Avaliacao, error) {
	if f.falha["InsertAvaliacao"] {
		return domain.Avaliacao{}, errFalha
	}
	return f.Repository.InsertAvaliacao(ctx, db, a)
}
func (f falhaRepo) ResumoAvaliacoes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, recentes int) (domain.ResumoAvaliacoes, error) {
	if f.falha["ResumoAvaliacoes"] {
		return domain.ResumoAvaliacoes{}, errFalha
	}
	return f.Repository.ResumoAvaliacoes(ctx, db, restrito, unidades, recentes)
}
func (f falhaRepo) ListarParaGovernanca(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, limite int) ([]domain.Solicitacao, error) {
	if f.falha["ListarParaGovernanca"] {
		return nil, errFalha
	}
	return f.Repository.ListarParaGovernanca(ctx, db, restrito, unidades, limite)
}

// eventosFalham recusa só o evento do tipo dado.
type eventosFalham struct{ tipo string }

func (e eventosFalham) Write(_ context.Context, _ pgx.Tx, eventType, _, _ string, _ uuid.UUID, _ any) error {
	if eventType == e.tipo {
		return errFalha
	}
	return nil
}

type cenario struct {
	pool              *pgxpool.Pool
	unidade           uuid.UUID
	tecnico, servidor auth.Identity
	oferta            domain.Oferta
	base              *application.Service
}

// novoCenario monta uma unidade com oferta de incidente, uma equipe com o
// técnico e um servidor que abre. O serviço base não avisa ninguém.
func novoCenario(t *testing.T) cenario {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	tec, serv := dbtest.User(t, pool), dbtest.User(t, pool)
	base := application.NewService(pool, infrastructure.NewRepository(), eventosFalham{}, nil)
	o, err := base.CriarOferta(ctx, admin(), application.OfertaEntrada{Nome: "E-mail " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &u})
	if err != nil {
		t.Fatal(err)
	}
	eq, err := base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "N1", UnidadeID: &u, Membros: []uuid.UUID{tec}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE id = $1`, eq.ID)
	})
	tecnico := atendente(u)
	tecnico.UserID = tec
	return cenario{pool: pool, unidade: u, tecnico: tecnico, servidor: auth.Identity{UserID: serv}, oferta: o, base: base}
}

func TestBuscaDaSolicitacao(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	outro := auth.Identity{UserID: dbtest.User(t, c.pool)}

	doSolicitante, err := c.base.Search(ctx, c.servidor, "Pedido", 5)
	if err != nil || len(doSolicitante) != 1 || doSolicitante[0].ID != sol.ID {
		t.Fatalf("quem abriu: %+v %v", doSolicitante, err)
	}
	if doTecnico, err := c.base.Search(ctx, c.tecnico, "Ped", 5); err != nil || len(doTecnico) != 1 {
		t.Fatalf("fila: %+v %v", doTecnico, err)
	}
	if alheio, err := c.base.Search(ctx, outro, "Pedido", 5); err != nil || len(alheio) != 0 {
		t.Fatalf("fora do escopo: %+v %v", alheio, err)
	}
	if curinga, err := c.base.Search(ctx, c.servidor, "%", 5); err != nil || len(curinga) != 0 {
		t.Fatalf("curinga literal: %+v %v", curinga, err)
	}
	if porNumero, err := c.base.Search(ctx, c.servidor, fmtNumero(sol.Numero), 5); err != nil || len(porNumero) != 1 {
		t.Fatalf("número: %+v %v", porNumero, err)
	}
}

func fmtNumero(n int64) string {
	return strconv.FormatInt(n, 10)
}

func TestBuscaPropagaFalha(t *testing.T) {
	svc := application.NewService(nil, falhando("ListSolicitacoes"), eventosFalham{}, nil)
	if _, err := svc.Search(context.Background(), auth.Identity{UserID: uuid.New()}, "pedido", 5); err == nil {
		t.Fatal("falha da lista")
	}
	if lista, err := svc.Search(context.Background(), auth.Identity{}, "  ", 5); err != nil || lista != nil {
		t.Fatalf("consulta em branco: %+v %v", lista, err)
	}
}

func (c cenario) abrir(t *testing.T) domain.Solicitacao {
	t.Helper()
	s, err := c.base.Abrir(context.Background(), c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Pedido"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// emAtendimento abre e assume pelo técnico.
func (c cenario) emAtendimento(t *testing.T) domain.Solicitacao {
	t.Helper()
	s := c.abrir(t)
	if _, err := c.base.Atribuir(context.Background(), c.tecnico, s.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	return s
}

func (c cenario) com(repo domain.Repository, ev application.Eventos) *application.Service {
	return application.NewService(c.pool, repo, ev, nil)
}

func TestSolicitacaoCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	pg := pagination.New(1, 20, 100)
	abertura := application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Pedido"}

	// Abertura: cada passo da transação desfaz tudo.
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"oferta", c.com(falhando("GetOferta"), eventosFalham{})},
		{"insert", c.com(falhando("InsertSolicitacao"), eventosFalham{})},
		{"passagem", c.com(falhando("InsertPassagem"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoSolicitacaoAberta})},
		{"equipe", c.com(falhando("Responsaveis"), eventosFalham{})},
	} {
		if _, err := caso.svc.Abrir(ctx, c.servidor, abertura); err == nil {
			t.Errorf("abertura com falha em %s", caso.nome)
		}
	}
	minhas, _, err := c.base.ListarMinhas(ctx, c.servidor, application.FiltroFila{Finalizadas: true}, pg)
	if err != nil || len(minhas) != 0 {
		t.Fatalf("nenhuma abertura com falha ficou gravada: %v %v", minhas, err)
	}

	// Fila.
	if lista, _, err := c.base.ListarFila(ctx, c.servidor, application.FiltroFila{}, pg); err != nil || len(lista) != 0 {
		t.Fatalf("sem atendimento, fila vazia: %v %v", lista, err)
	}
	sol := c.emAtendimento(t)
	if lista, _, err := c.base.ListarFila(ctx, c.tecnico, application.FiltroFila{Responsavel: application.ResponsavelEu}, pg); err != nil || len(lista) != 1 {
		t.Fatalf("fila do técnico: %v %v", lista, err)
	}
	if lista, _, err := c.base.ListarFila(ctx, c.tecnico, application.FiltroFila{Responsavel: uuid.NewString()}, pg); err != nil || len(lista) != 0 {
		t.Fatalf("fila de outra pessoa: %v %v", lista, err)
	}
	if _, _, err := c.com(falhando("ListSolicitacoes"), eventosFalham{}).ListarFila(ctx, c.tecnico, application.FiltroFila{}, pg); err == nil {
		t.Fatal("falha na fila")
	}

	// Leitura do detalhe.
	for _, m := range []string{"ListPassagens", "ListComentarios"} {
		if _, err := c.com(falhando(m), eventosFalham{}).Obter(ctx, c.tecnico, sol.ID); err == nil {
			t.Errorf("detalhe com falha em %s", m)
		}
	}

	// Atribuição.
	if _, err := c.base.Atribuir(ctx, c.servidor, sol.ID, c.tecnico.UserID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("quem abriu não atribui: %v", err)
	}
	if _, err := c.base.Responsaveis(ctx, c.servidor, sol.ID); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("quem abriu não lista responsáveis: %v", err)
	}
	aberta := c.abrir(t)
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"conferência", c.com(falhando("ResponsavelValido"), eventosFalham{})},
		{"gravação", c.com(falhando("UpdateSolicitacao"), eventosFalham{})},
		{"passagem", c.com(falhando("InsertPassagem"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoSolicitacaoAtribuida})},
		{"trava", c.com(falhando("TravarSolicitacao"), eventosFalham{})},
		{"leitura", c.com(falhando("GetSolicitacao"), eventosFalham{})},
	} {
		if _, err := caso.svc.Atribuir(ctx, c.tecnico, aberta.ID, c.tecnico.UserID); err == nil {
			t.Errorf("atribuição com falha em %s", caso.nome)
		}
	}

	// Mudança de estado.
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"gravação", c.com(falhando("UpdateSolicitacao"), eventosFalham{})},
		{"passagem", c.com(falhando("InsertPassagem"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoSolicitacaoEstado})},
	} {
		if _, err := caso.svc.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoAguardandoTerceiro); err == nil {
			t.Errorf("passagem com falha em %s", caso.nome)
		}
	}

	// Comentário.
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"gravação", c.com(falhando("InsertComentario"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoSolicitacaoComentada})},
	} {
		if _, err := caso.svc.Comentar(ctx, c.tecnico, sol.ID, "nota", true); err == nil {
			t.Errorf("comentário com falha em %s", caso.nome)
		}
	}
	if _, err := c.base.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoAguardandoServidor); err != nil {
		t.Fatal(err)
	}
	if _, err := c.com(falhando("UpdateSolicitacao"), eventosFalham{}).Comentar(ctx, c.servidor, sol.ID, "segue", false); err == nil {
		t.Fatal("a resposta que devolve à equipe falha junto com a passagem")
	}
	d, err := c.base.Obter(ctx, c.tecnico, sol.ID)
	if err != nil || d.Solicitacao.Estado != domain.EstadoAguardandoServidor || len(d.Comentarios) != 0 {
		t.Fatalf("nada da resposta com falha ficou gravado: %+v %v", d, err)
	}
}

func (f falhaRepo) GetPolitica(ctx context.Context, db database.DBTX, u *uuid.UUID) (domain.Politica, error) {
	if f.falha["GetPolitica"] {
		return domain.Politica{}, errFalha
	}
	return f.Repository.GetPolitica(ctx, db, u)
}
func (f falhaRepo) ResolverPolitica(ctx context.Context, db database.DBTX, u *uuid.UUID) (domain.Politica, error) {
	if f.falha["ResolverPolitica"] {
		return domain.Politica{}, errFalha
	}
	if f.falha["PoliticaIlegivel"] {
		return domain.Politica{Fuso: "Marte/Olimpo"}, nil
	}
	return f.Repository.ResolverPolitica(ctx, db, u)
}
func (f falhaRepo) SalvarPolitica(ctx context.Context, db database.DBTX, p domain.Politica) (domain.Politica, error) {
	if f.falha["SalvarPolitica"] {
		return domain.Politica{}, errFalha
	}
	return f.Repository.SalvarPolitica(ctx, db, p)
}
func (f falhaRepo) ListFeriados(ctx context.Context, db database.DBTX, u *uuid.UUID) ([]domain.Feriado, error) {
	if f.falha["ListFeriados"] {
		return nil, errFalha
	}
	return f.Repository.ListFeriados(ctx, db, u)
}
func (f falhaRepo) FeriadosVigentes(ctx context.Context, db database.DBTX, u *uuid.UUID, de, ate string) ([]domain.Feriado, error) {
	if f.falha["FeriadosVigentes"] {
		return nil, errFalha
	}
	return f.Repository.FeriadosVigentes(ctx, db, u, de, ate)
}
func (f falhaRepo) InsertFeriado(ctx context.Context, db database.DBTX, x domain.Feriado) (domain.Feriado, error) {
	if f.falha["InsertFeriado"] {
		return domain.Feriado{}, errFalha
	}
	return f.Repository.InsertFeriado(ctx, db, x)
}
func (f falhaRepo) DeleteFeriado(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["DeleteFeriado"] {
		return errFalha
	}
	return f.Repository.DeleteFeriado(ctx, db, id)
}
func (f falhaRepo) PassagensDe(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID][]domain.Passagem, error) {
	if f.falha["PassagensDe"] {
		return nil, errFalha
	}
	return f.Repository.PassagensDe(ctx, db, ids)
}
func (f falhaRepo) ComPrazoCorrendo(ctx context.Context, db database.DBTX) ([]domain.Solicitacao, error) {
	if f.falha["ComPrazoCorrendo"] {
		return nil, errFalha
	}
	return f.Repository.ComPrazoCorrendo(ctx, db)
}

func (f falhaRepo) GetVisita(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Visita, error) {
	if f.falha["GetVisita"] {
		return domain.Visita{}, errFalha
	}
	return f.Repository.GetVisita(ctx, db, id)
}
func (f falhaRepo) TravarVisita(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["TravarVisita"] {
		return errFalha
	}
	return f.Repository.TravarVisita(ctx, db, id)
}
func (f falhaRepo) InsertVisita(ctx context.Context, db database.DBTX, v domain.Visita) error {
	if f.falha["InsertVisita"] {
		return errFalha
	}
	return f.Repository.InsertVisita(ctx, db, v)
}
func (f falhaRepo) UpdateVisita(ctx context.Context, db database.DBTX, v domain.Visita) error {
	if f.falha["UpdateVisita"] {
		return errFalha
	}
	return f.Repository.UpdateVisita(ctx, db, v)
}
func (f falhaRepo) ListVisitas(ctx context.Context, db database.DBTX, id uuid.UUID) ([]domain.Visita, error) {
	if f.falha["ListVisitas"] {
		return nil, errFalha
	}
	return f.Repository.ListVisitas(ctx, db, id)
}
func (f falhaRepo) VisitasEmAndamento(ctx context.Context, db database.DBTX, x domain.Filtro) ([]domain.VisitaEmCampo, error) {
	if f.falha["VisitasEmAndamento"] {
		return nil, errFalha
	}
	return f.Repository.VisitasEmAndamento(ctx, db, x)
}
func (f falhaRepo) InsertEvidencia(ctx context.Context, db database.DBTX, e domain.Evidencia) (domain.Evidencia, error) {
	if f.falha["InsertEvidencia"] {
		return domain.Evidencia{}, errFalha
	}
	return f.Repository.InsertEvidencia(ctx, db, e)
}
func (f falhaRepo) ApagarLocalizacoes(ctx context.Context, db database.DBTX, agora time.Time) (int64, error) {
	if f.falha["ApagarLocalizacoes"] {
		return 0, errFalha
	}
	return f.Repository.ApagarLocalizacoes(ctx, db, agora)
}

func (f falhaRepo) DadosDoTitular(ctx context.Context, db database.DBTX, user uuid.UUID) (domain.DadosDoTitular, error) {
	if f.falha["DadosDoTitular"] {
		return domain.DadosDoTitular{}, errFalha
	}
	return f.Repository.DadosDoTitular(ctx, db, user)
}

// A lista como a do GLPI (ADR 080): busca, recortes, período e ordem por
// coluna; o que não se lê é recusado.
func TestListaComoOGLPI(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	pg := pagination.New(1, 20, 100)
	primeira, segunda := c.abrir(t), c.abrir(t)
	fila := func(f application.FiltroFila) []domain.Solicitacao {
		t.Helper()
		lista, _, err := c.base.ListarFila(ctx, c.tecnico, f, pg)
		if err != nil {
			t.Fatal(err)
		}
		return lista
	}
	agora := time.Now().UTC()
	ontem, amanha := agora.Add(-24*time.Hour).Format(time.RFC3339), agora.Add(24*time.Hour).Format(time.RFC3339)
	unidade := c.unidade.String()
	for nome, caso := range map[string]struct {
		f application.FiltroFila
		n int
	}{
		"número":          {application.FiltroFila{Texto: strconv.FormatInt(segunda.Numero, 10)}, 1},
		"unidade":         {application.FiltroFila{Unidade: unidade}, 2},
		"outra unidade":   {application.FiltroFila{Unidade: uuid.NewString()}, 0},
		"outra categoria": {application.FiltroFila{Unidade: unidade, Categoria: uuid.NewString()}, 0},
		"período":         {application.FiltroFila{Unidade: unidade, De: ontem, Ate: amanha}, 2},
		"depois":          {application.FiltroFila{Unidade: unidade, De: amanha}, 0},
		"prioridade":      {application.FiltroFila{Unidade: unidade, Prioridade: domain.PrioridadeCritica}, 0},
	} {
		if lista := fila(caso.f); len(lista) != caso.n {
			t.Errorf("%s: %d, esperado %d", nome, len(lista), caso.n)
		}
	}
	for _, ordem := range domain.OrdensDaLista {
		for _, desc := range []bool{false, true} {
			if lista := fila(application.FiltroFila{Unidade: unidade, Ordem: ordem, Desc: desc}); len(lista) != 2 {
				t.Fatalf("ordem %s: %d", ordem, len(lista))
			}
		}
	}
	if lista := fila(application.FiltroFila{Unidade: unidade, Ordem: "numero", Desc: true}); lista[0].ID != segunda.ID || lista[1].ID != primeira.ID ||
		lista[0].UnidadeNome == "" {
		t.Fatalf("mais nova primeiro, com a unidade: %+v", lista)
	}
	for _, f := range []application.FiltroFila{{Prioridade: "urgente"}, {Ordem: "cor"}, {Unidade: "x"}, {Categoria: "x"}, {De: "ontem"}, {Ate: "2026-01-01"}} {
		if _, _, err := c.base.ListarFila(ctx, c.tecnico, f, pg); !errors.Is(err, domain.ErrEntradaInvalida) {
			t.Errorf("%+v: %v", f, err)
		}
		if _, _, err := c.base.ListarMinhas(ctx, c.servidor, f, pg); !errors.Is(err, domain.ErrEntradaInvalida) {
			t.Errorf("minhas %+v: %v", f, err)
		}
	}
}
