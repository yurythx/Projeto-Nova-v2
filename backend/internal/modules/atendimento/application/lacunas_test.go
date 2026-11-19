package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

type ativosOk struct {
	unidade *uuid.UUID
	err     error
}

func (a ativosOk) Consultar(context.Context, database.DBTX, uuid.UUID) (string, *uuid.UUID, error) {
	if a.err != nil {
		return "", nil, a.err
	}
	return "Servidor de e-mail", a.unidade, nil
}

type cofreOk struct{}

func (cofreOk) Nome(context.Context, auth.Identity, uuid.UUID) (string, error) {
	return "Wi-Fi", nil
}

func TestMapErrorDoAtendimento(t *testing.T) {
	for _, err := range []error{
		nil,
		domain.ErrNaoEncontrado, domain.ErrEspelho, domain.ErrSemIA, domain.ErrRespostaDaIA,
		domain.ErrSugestaoDecidida, domain.ErrAguardandoAprovacao, domain.ErrEtapasSoRequisicao,
		domain.ErrEtapaInvalida, domain.ErrSemAprovacaoPendente, domain.ErrNaoAprovador, domain.ErrAutoaprovacao,
		domain.ErrContratosDesligado, domain.ErrContratoAusente, domain.ErrOLAInvalido, domain.ErrFacilidadeInvalida,
		domain.ErrMetasInvalidas, domain.ErrTarefasSoRequisicao, domain.ErrModeloInvalido, domain.ErrTarefaFeita,
		domain.ErrNaoEhDaEquipe, domain.ErrDispensaSemMotivo, domain.ErrTarefasPendentes, domain.ErrMotivoObrigatorio,
		domain.ErrEventosSemOferta, domain.ErrAlertaInvalido, domain.ErrApresentacaoInvalida, domain.ErrCategoriaRepetida,
		domain.ErrCategoriaInexistente, domain.ErrEntradaInvalida, domain.ErrForaDoEscopo, domain.ErrProdutoInexistente,
		domain.ErrProdutoIndisponivel, domain.ErrProdutoDeOutraUnidade, domain.ErrMembroInvalido, domain.ErrUnidadeInexistente,
		domain.ErrTipoInvalido, domain.ErrOfertaInexistente, domain.ErrOfertaInativa, domain.ErrSolicitacaoInvalida,
		domain.ErrSolicitanteInvalido, domain.ErrTransicaoInvalida, domain.ErrSemResponsavel, domain.ErrResponsavelInvalido,
		domain.ErrIncidentePaiInvalido, domain.ErrSolicitacaoFinalizada, domain.ErrCofreDesligado, domain.ErrSegredoAusente,
		domain.ErrAtivosDesligado, domain.ErrAtivoAusente, domain.ErrAtivoDeOutraUnidade, domain.ErrComentarioInvalido,
		domain.ErrNotaInternaNegada, domain.ErrAvaliacaoInvalida, domain.ErrAvaliacaoForaDeHora, domain.ErrAvaliacaoAlheia,
		domain.ErrAvaliacaoExistente, domain.ErrPoliticaInvalida, domain.ErrFeriadoInvalido, domain.ErrFeriadoRepetido,
		domain.ErrNivelInvalido, domain.ErrVisitaInvalida, domain.ErrLocalizacao, domain.ErrCapturaForaPrazo,
		domain.ErrVisitaAberta, domain.ErrVisitaEncerrada, domain.ErrVisitaDeOutro, domain.ErrEvidenciaInvalida,
		domain.CampoInvalido{Msg: "informe o título"}, domain.Recusa{Msg: "informe o artigo"},
		errors.New("banco"),
	} {
		_ = application.MapError(err)
	}
}

func TestFuncoesQueACatracaNaoVia(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	gestor := admin()
	gestor.UserID = c.servidor.UserID
	ativa := false
	sla, err := c.base.CriarSLA(ctx, gestor, application.SLAEntrada{
		Nome: "Resposta em quatro horas", Descricao: "Incidente", TempoRespostaMinutos: 30, TempoSolucaoMinutos: 240,
		PausarEmEspera: true, CalendarioTipo: domain.CalendarioExpediente, UnidadeID: &c.unidade, Ativa: &ativa,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_slas WHERE id = $1`, sla.ID)
	})
	ativa = true
	if sla, err = c.base.AtualizarSLA(ctx, gestor, sla.ID, application.SLAEntrada{
		Nome: "Resposta em quatro horas", TempoRespostaMinutos: 15, TempoSolucaoMinutos: 120,
		CalendarioTipo: domain.CalendarioCorrido, UnidadeID: &c.unidade, Ativa: &ativa,
	}); err != nil {
		t.Fatal(err)
	}
	cat, err := c.base.CriarCategoria(ctx, gestor, application.CategoriaEntrada{Nome: "Rede " + uuid.NewString()[:6], Descricao: "Link e Wi-Fi"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_categorias WHERE id = $1`, cat.ID)
	})
	if err := c.base.AssociarCategoriaSLA(ctx, gestor, cat.ID, &sla.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.base.AssociarCategoriaSLA(ctx, gestor, cat.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ExcluirSLA(ctx, gestor, sla.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := c.base.SalvarResposta(ctx, gestor, c.oferta.ID, "Reinicie o aparelho.", "Ver o ramal antes."); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.SalvarResposta(ctx, gestor, c.oferta.ID, " ", " "); err != nil {
		t.Fatal(err)
	}

	marca, err := c.base.CriarRecorrencia(ctx, gestor, domain.ChamadoMensal{
		OfertaID: c.oferta.ID, Titulo: "Backup mensal", Descricao: "Conferir a cópia", Dia: 1, CriadoPor: c.servidor.UserID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_recorrencias WHERE id = $1`, marca.ID)
	})
	if _, err := c.base.ListarRecorrencias(ctx, gestor); err != nil {
		t.Fatal(err)
	}
	if err := c.base.AbrirRecorrencias(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ExcluirRecorrencia(ctx, gestor, marca.ID); err != nil {
		t.Fatal(err)
	}

	if err := c.base.RegistrarDeflexao(ctx, gestor, []string{"senha", " ", "senha", strings.Repeat("a", 81)}, false); err == nil {
		t.Fatal("artigo longo")
	}
	if err := c.base.RegistrarDeflexao(ctx, gestor, []string{" "}, false); err == nil {
		t.Fatal("artigo vazio")
	}
	if err := c.base.RegistrarDeflexao(ctx, gestor, []string{"senha", "wifi"}, true); err != nil {
		t.Fatal(err)
	}
	if err := c.base.RegistrarDeflexao(ctx, auth.Identity{}, []string{"senha"}, false); err == nil {
		t.Fatal("sem pessoa")
	}

	sol := c.emAtendimento(t)
	filho := c.abrir(t)
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, sol.ID, &sol.ID); err == nil {
		t.Fatal("ativos desligados")
	}
	alheia := uuid.New()
	c.base.WithAtivos(ativosOk{unidade: &alheia})
	outra := uuid.New()
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, sol.ID, &outra); err == nil {
		t.Fatal("outra unidade")
	}
	c.base.WithAtivos(ativosOk{err: domain.ErrAtivoAusente})
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, sol.ID, &outra); err == nil {
		t.Fatal("ativo ausente")
	}
	c.base.WithAtivos(ativosOk{})
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, sol.ID, &outra); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularAtivo(ctx, c.tecnico, sol.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularSegredo(ctx, c.tecnico, sol.ID, &outra); err == nil {
		t.Fatal("cofre desligado")
	}
	c.base.WithSegredo(cofreOk{})
	if _, err := c.base.VincularSegredo(ctx, c.tecnico, sol.ID, &outra); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularSegredo(ctx, c.tecnico, sol.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, filho.ID, &filho.ID); err == nil {
		t.Fatal("pai de si")
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, filho.ID, &sol.ID); err != nil {
		t.Fatal(err)
	}
	if lista, err := c.base.ListarFilhosDoIncidente(ctx, c.tecnico, sol.ID); err != nil || len(lista) != 1 {
		t.Fatalf("filhos %+v %v", lista, err)
	}
	if _, err := c.base.VincularIncidentePai(ctx, c.tecnico, filho.ID, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := c.base.Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoResolvido); err != nil {
		t.Fatal(err)
	}
	parada := c.emAtendimento(t)
	terceiro := c.emAtendimento(t)
	antigo := time.Now().Add(-10 * 24 * time.Hour)
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET resolvida_em = $2 WHERE id = $1`, sol.ID, antigo); err != nil {
		t.Fatal(err)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET updated_at = $2, responsavel_id = $3 WHERE id = $1`, parada.ID, antigo, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET estado = $2, updated_at = $3, responsavel_id = NULL WHERE id = $1`, terceiro.ID, domain.EstadoAguardandoTerceiro, antigo); err != nil {
		t.Fatal(err)
	}
	c.base.WithNotificacoes(notificacoes.NewService(c.pool, nil, slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err := c.base.CuidarDaFila(ctx); err != nil {
		t.Fatal(err)
	}
}
