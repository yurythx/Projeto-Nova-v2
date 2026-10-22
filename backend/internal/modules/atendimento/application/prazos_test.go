package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/domain/pagination"
	"github.com/yurythx/sismarq/internal/modules/atendimento/application"
	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/sismarq/internal/platform/database"
	"github.com/yurythx/sismarq/internal/platform/notificacoes"
)

var silencio = slog.New(slog.NewTextHandler(io.Discard, nil))

// politica24h: expediente o tempo todo (as contas do teste ficam em
// minutos corridos), média com 60 min de resposta e 120 de solução.
func politica24h(unidade *uuid.UUID) application.PoliticaEntrada {
	in := application.PoliticaEntrada{UnidadeID: unidade, Fuso: "America/Sao_Paulo"}
	for dia := 0; dia <= 6; dia++ {
		in.Turnos = append(in.Turnos, domain.Turno{Dia: dia, Inicio: "00:00", Fim: "24:00"})
	}
	for i := 1; i <= 3; i++ {
		for u := 1; u <= 3; u++ {
			p := domain.PrioridadeMedia
			if i == 3 && u == 3 {
				p = domain.PrioridadeCritica
			}
			in.Matriz = append(in.Matriz, domain.Celula{Impacto: i, Urgencia: u, Prioridade: p})
		}
	}
	in.Metas = []domain.Meta{
		{Prioridade: domain.PrioridadeCritica, RespostaMin: 10, SolucaoMin: 30},
		{Prioridade: domain.PrioridadeMedia, RespostaMin: 60, SolucaoMin: 120},
	}
	return in
}

func (c cenario) comPolitica(t *testing.T) {
	t.Helper()
	if _, err := c.base.SalvarPolitica(context.Background(), admin(), politica24h(&c.unidade)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = c.pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id = $1`, c.unidade)
		_, _ = c.pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id = $1`, c.unidade)
	})
}

func (c cenario) avisos(t *testing.T, sol uuid.UUID, user uuid.UUID, fato string) int {
	t.Helper()
	var n int
	if err := c.pool.QueryRow(context.Background(), `SELECT count(*) FROM notificacoes WHERE user_id = $1 AND chave = $2`,
		user, "atendimento:"+sol.String()+":prazo:"+fato).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPrazoNaPratica(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)

	if _, err := c.base.SalvarPolitica(ctx, c.servidor, politica24h(&c.unidade)); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("política fora do escopo: %v", err)
	}
	if _, err := c.base.ObterPolitica(ctx, c.servidor, &c.unidade); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("leitura fora do escopo: %v", err)
	}
	if p, err := c.base.ObterPolitica(ctx, admin(), &c.unidade); err != nil || (p != nil && !p.Herdada) {
		t.Fatalf("antes de cadastrar, nada próprio: %+v %v", p, err)
	}
	semPrazo := c.abrir(t) // aberta antes da política: fica sem prazo
	c.comPolitica(t)
	if _, err := c.base.SalvarPolitica(ctx, admin(), politica24h(&c.unidade)); err != nil {
		t.Fatalf("substituir a política: %v", err)
	}
	if p, err := c.base.ObterPolitica(ctx, admin(), &c.unidade); err != nil || p == nil || p.Herdada || len(p.Metas) != 2 {
		t.Fatalf("política própria: %+v %v", p, err)
	}

	inicio := time.Now()
	agora := inicio
	svc := c.com(infrastructure.NewRepository(), eventosFalham{}).
		WithNotificacoes(notificacoes.NewService(c.pool, nil, silencio)).
		WithRelogio(func() time.Time { return agora })
	sol, err := svc.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Sem e-mail"})
	if err != nil || sol.Prioridade == nil || *sol.Prioridade != domain.PrioridadeMedia || *sol.MetaRespostaMin != 60 {
		t.Fatalf("a abertura já classifica pela política: %+v %v", sol, err)
	}
	outra, err := svc.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "Sem agenda"})
	if err != nil {
		t.Fatal(err)
	}

	// 50 min sem resposta: 83%, aviso de 75% para a equipe (sem responsável).
	agora = inicio.Add(50 * time.Minute)
	for range 2 {
		if err := svc.VerificarPrazos(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if c.avisos(t, sol.ID, c.tecnico.UserID, "resposta:75") != 1 || c.avisos(t, outra.ID, c.tecnico.UserID, "resposta:75") != 1 {
		t.Fatal("o alerta de 75% sai uma vez para a equipe, por solicitação")
	}
	d, err := svc.Obter(ctx, c.tecnico, sol.ID)
	if err != nil || d.Solicitacao.Prazo == nil || d.Solicitacao.Prazo.Resposta.Situacao != domain.SituacaoAtencao {
		t.Fatalf("a equipe vê o prazo: %+v %v", d.Solicitacao.Prazo, err)
	}
	if d, _ := svc.Obter(ctx, c.servidor, sol.ID); d.Solicitacao.Prazo != nil {
		t.Fatal("quem abriu não vê o prazo")
	}

	// 65 min: a resposta venceu.
	agora = inicio.Add(65 * time.Minute)
	if err := svc.VerificarPrazos(ctx); err != nil || c.avisos(t, sol.ID, c.tecnico.UserID, "resposta:100") != 1 {
		t.Fatalf("aviso de vencimento: %v", err)
	}

	// Assumida (a passagem leva o horário real, logo depois da abertura):
	// a resposta foi cumprida e o aviso da solução vai ao responsável.
	if _, err := svc.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	agora = inicio.Add(100 * time.Minute)
	if err := svc.VerificarPrazos(ctx); err != nil || c.avisos(t, sol.ID, c.tecnico.UserID, "solucao:75") != 1 {
		t.Fatalf("aviso da solução ao responsável: %v", err)
	}
	lista, _, err := svc.ListarFila(ctx, c.tecnico, application.FiltroFila{}, pagination.New(1, 20, 100))
	if err != nil || len(lista) != 3 {
		t.Fatalf("fila: %+v %v", lista, err)
	}
	for _, item := range lista {
		switch item.ID {
		case semPrazo.ID:
			if item.Prazo != nil {
				t.Fatal("sem prioridade, sem prazo")
			}
		case sol.ID:
			if item.Prazo == nil || item.Prazo.Resposta.Situacao != domain.SituacaoCumprido {
				t.Fatalf("a fila mostra o prazo: %+v", item.Prazo)
			}
		}
	}

	// Reclassificar como crítica aperta a meta.
	if _, err := svc.Classificar(ctx, c.servidor, sol.ID, 3, 3); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("quem abriu não classifica: %v", err)
	}
	d, err = svc.Classificar(ctx, c.tecnico, sol.ID, 3, 3)
	if err != nil || *d.Solicitacao.Prioridade != domain.PrioridadeCritica || d.Solicitacao.Prazo.Solucao.Situacao != domain.SituacaoVencido {
		t.Fatalf("crítica: %+v %v", d.Solicitacao, err)
	}
	if _, err := svc.Classificar(ctx, c.tecnico, sol.ID, 3, 3); err != nil {
		t.Fatalf("mesma classificação: %v", err)
	}
	if _, err := svc.Classificar(ctx, c.tecnico, sol.ID, 4, 1); !errors.Is(err, domain.ErrNivelInvalido) {
		t.Fatalf("nível inválido: %v", err)
	}

	// Feriados.
	u := c.unidade
	if _, err := c.base.ListarFeriados(ctx, c.servidor, &u); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	if _, err := c.base.CriarFeriado(ctx, admin(), application.FeriadoEntrada{UnidadeID: &u, Data: "20/11", Nome: "x", Tipo: domain.FeriadoMunicipal}); !errors.Is(err, domain.ErrFeriadoInvalido) {
		t.Fatal(err)
	}
	entrada := application.FeriadoEntrada{UnidadeID: &u, Data: "2026-11-20", Nome: "Consciência Negra", Tipo: domain.FeriadoNacional}
	if _, err := c.base.CriarFeriado(ctx, c.servidor, entrada); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatal(err)
	}
	f, err := c.base.CriarFeriado(ctx, admin(), entrada)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.CriarFeriado(ctx, admin(), entrada); !errors.Is(err, domain.ErrFeriadoRepetido) {
		t.Fatalf("data repetida: %v", err)
	}
	if lista, err := c.base.ListarFeriados(ctx, admin(), &u); err != nil || len(lista) != 1 {
		t.Fatalf("feriados da unidade: %v %v", lista, err)
	}
	if err := c.base.ExcluirFeriado(ctx, atendente(uuid.New()), f.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("de outra unidade: %v", err)
	}
	if err := c.base.ExcluirFeriado(ctx, admin(), f.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ExcluirFeriado(ctx, admin(), f.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("já excluído: %v", err)
	}
}

func TestPrazoCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	c.comPolitica(t)
	u := c.unidade
	pg := pagination.New(1, 20, 100)
	in := politica24h(&u)

	invalida := in
	invalida.Fuso = ""
	if _, err := c.base.SalvarPolitica(ctx, admin(), invalida); !errors.Is(err, domain.ErrPoliticaInvalida) {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"leitura", c.com(falhando("GetPolitica"), eventosFalham{})},
		{"gravação", c.com(falhando("SalvarPolitica"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoPoliticaSalva})},
	} {
		if _, err := caso.svc.SalvarPolitica(ctx, admin(), in); err == nil {
			t.Errorf("política com falha em %s", caso.nome)
		}
	}
	if _, err := c.com(falhando("ResolverPolitica"), eventosFalham{}).ObterPolitica(ctx, admin(), &u); err == nil {
		t.Error("leitura da política")
	}
	if _, err := c.com(falhando("ListFeriados"), eventosFalham{}).ListarFeriados(ctx, admin(), &u); err == nil {
		t.Error("lista de feriados")
	}
	feriado := application.FeriadoEntrada{UnidadeID: &u, Data: "2026-12-25", Nome: "Natal", Tipo: domain.FeriadoNacional}
	if _, err := c.com(falhando("InsertFeriado"), eventosFalham{}).CriarFeriado(ctx, admin(), feriado); err == nil {
		t.Error("gravação do feriado")
	}
	if _, err := c.com(falhando(), eventosFalham{application.EventoFeriadoCriado}).CriarFeriado(ctx, admin(), feriado); err == nil {
		t.Error("evento do feriado")
	}
	natal, err := c.base.CriarFeriado(ctx, admin(), feriado)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.com(falhando("DeleteFeriado"), eventosFalham{}).ExcluirFeriado(ctx, admin(), natal.ID); err == nil {
		t.Error("exclusão do feriado")
	}
	if err := c.com(falhando(), eventosFalham{application.EventoFeriadoExcluido}).ExcluirFeriado(ctx, admin(), natal.ID); err == nil {
		t.Error("evento da exclusão")
	}

	if _, err := c.com(falhando("ResolverPolitica"), eventosFalham{}).Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: "x"}); err == nil {
		t.Error("abertura sem conseguir ler a política")
	}
	sol := c.abrir(t)
	for _, caso := range []struct {
		nome string
		svc  *application.Service
	}{
		{"política", c.com(falhando("ResolverPolitica"), eventosFalham{})},
		{"gravação", c.com(falhando("UpdateSolicitacao"), eventosFalham{})},
		{"evento", c.com(falhando(), eventosFalham{application.EventoSolicitacaoClassificada})},
	} {
		if _, err := caso.svc.Classificar(ctx, c.tecnico, sol.ID, 1, 1); err == nil {
			t.Errorf("classificação com falha em %s", caso.nome)
		}
	}

	// A medição do prazo falha junto com a leitura.
	for _, m := range []string{"PassagensDe", "FeriadosVigentes", "PoliticaIlegivel"} {
		if _, err := c.com(falhando(m), eventosFalham{}).Obter(ctx, c.tecnico, sol.ID); err == nil {
			t.Errorf("detalhe com falha em %s", m)
		}
	}
	for _, m := range []string{"ResolverPolitica", "PassagensDe"} {
		if _, _, err := c.com(falhando(m), eventosFalham{}).ListarFila(ctx, c.tecnico, application.FiltroFila{}, pg); err == nil {
			t.Errorf("fila com falha em %s", m)
		}
	}

	// Alertas.
	comAvisos := func(r domain.Repository) *application.Service {
		return c.com(r, eventosFalham{}).WithNotificacoes(notificacoes.NewService(c.pool, nil, silencio)).
			WithRelogio(func() time.Time { return time.Now().Add(55 * time.Minute) })
	}
	if err := c.base.VerificarPrazos(ctx); err != nil {
		t.Fatalf("sem avisos ligados, nada a fazer: %v", err)
	}
	for _, m := range []string{"ComPrazoCorrendo", "PassagensDe", "Responsaveis"} {
		if err := comAvisos(falhando(m)).VerificarPrazos(ctx); err == nil {
			t.Errorf("alerta com falha em %s", m)
		}
	}
	cancelado, cancel := context.WithCancel(ctx)
	cancel()
	if err := comAvisos(enlatado{sol: sol}).VerificarPrazos(cancelado); err == nil {
		t.Error("aviso que não grava")
	}

	antes := application.IntervaloAlertas
	application.IntervaloAlertas = 5 * time.Millisecond
	t.Cleanup(func() { application.IntervaloAlertas = antes })
	curto, pare := context.WithTimeout(ctx, 30*time.Millisecond)
	defer pare()
	if err := comAvisos(falhando("ComPrazoCorrendo")).WithLogger(silencio).RunAlertas(curto); err != nil {
		t.Fatalf("o laço só termina com o contexto: %v", err)
	}
}

// enlatado responde o que o alerta precisa sem ir ao banco, para a falha
// cair só na gravação do aviso.
type enlatado struct {
	stubRepo
	sol domain.Solicitacao
}

func (e enlatado) ComPrazoCorrendo(context.Context, database.DBTX) ([]domain.Solicitacao, error) {
	return []domain.Solicitacao{e.sol}, nil
}
func (e enlatado) PassagensDe(context.Context, database.DBTX, []uuid.UUID) (map[uuid.UUID][]domain.Passagem, error) {
	return map[uuid.UUID][]domain.Passagem{e.sol.ID: {{Para: domain.EstadoAberto, Em: e.sol.CreatedAt}}}, nil
}
func (e enlatado) ResolverPolitica(context.Context, database.DBTX, *uuid.UUID) (domain.Politica, error) {
	in := politica24h(nil)
	return domain.NovaPolitica(nil, in.Fuso, in.Turnos, in.Matriz, in.Metas)
}
func (e enlatado) FeriadosVigentes(context.Context, database.DBTX, *uuid.UUID, string, string) ([]domain.Feriado, error) {
	return nil, nil
}
func (e enlatado) Responsaveis(context.Context, database.DBTX, *uuid.UUID) ([]domain.Candidato, error) {
	return []domain.Candidato{{ID: uuid.New()}}, nil
}
