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
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (f falhaRepo) ListarDoPeriodo(ctx context.Context, db database.DBTX, r bool, u []uuid.UUID, d time.Time, n int) ([]domain.Solicitacao, error) {
	if f.falha["ListarDoPeriodo"] {
		return nil, errFalha
	}
	return f.Repository.ListarDoPeriodo(ctx, db, r, u, d, n)
}
func (f falhaRepo) SecretariasDe(ctx context.Context, db database.DBTX, u []uuid.UUID) (map[uuid.UUID]domain.Secretaria, error) {
	if f.falha["SecretariasDe"] {
		return nil, errFalha
	}
	return f.Repository.SecretariasDe(ctx, db, u)
}
func (f falhaRepo) AvaliacoesDe(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID]domain.Avaliacao, error) {
	if f.falha["AvaliacoesDe"] {
		return nil, errFalha
	}
	return f.Repository.AvaliacoesDe(ctx, db, ids)
}
func (f falhaRepo) TarefasDoPeriodo(ctx context.Context, db database.DBTX, r bool, u []uuid.UUID, d time.Time) ([]domain.TarefaMedida, error) {
	if f.falha["TarefasDoPeriodo"] {
		return nil, errFalha
	}
	return f.Repository.TarefasDoPeriodo(ctx, db, r, u, d)
}

func TestRelatorioPorSecretaria(t *testing.T) {
	ctx := context.Background()
	c := novoComTarefas(t)
	semMetas(t, c.cenario)
	ola := 48
	if _, err := c.base.AtualizarEquipe(ctx, admin(), c.rede.ID, application.EquipeEntrada{Nome: "Rede", UnidadeID: &c.unidade, Membros: c.rede.Membros, OLAHoras: &ola}); err != nil {
		t.Fatal(err)
	}
	sol := c.pedir(t)
	d, _ := c.base.Obter(ctx, c.tecnico, sol.ID)
	if _, err := c.base.FazerTarefa(ctx, c.daRede, sol.ID, d.Tarefas[1].ID, true, ""); err != nil {
		t.Fatal(err)
	}
	em := c.emAtendimento(t)
	concluir(t, c.cenario, em)
	if _, err := c.base.Avaliar(ctx, c.servidor, em.ID, 5, ""); err != nil {
		t.Fatal(err)
	}

	rel, err := c.base.RelatorioPorSecretaria(ctx, c.tecnico, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Dias != application.DiasDoRelatorio || rel.Truncado || rel.Total.Nome != "Total" || rel.Total.Total != 2 || rel.Total.Concluidas != 1 {
		t.Fatalf("total: %+v", rel)
	}
	if len(rel.Secretarias) != 1 || rel.Secretarias[0].UnidadeID == nil || *rel.Secretarias[0].UnidadeID != c.unidade || *rel.Secretarias[0].Experiencia.CSAT.Valor != 5 {
		t.Fatalf("a unidade de topo é a secretaria: %+v", rel.Secretarias)
	}
	if len(rel.Equipes) != 2 {
		t.Fatalf("OLA das equipes: %+v", rel.Equipes)
	}
	for _, e := range rel.Equipes {
		switch e.Nome {
		case "Rede":
			if *e.OLAHoras != 48 || e.Concluidas != 1 || *e.NoPrazoPercentual != 100 {
				t.Errorf("rede: %+v", e)
			}
		case "Suporte":
			if e.Pendentes != 1 || e.OLAHoras != nil {
				t.Errorf("suporte: %+v", e)
			}
		}
	}
	if rel, _ := c.base.RelatorioPorSecretaria(ctx, c.tecnico, 5000); rel.Dias != application.MaxDiasDoRelatorio {
		t.Fatalf("teto da janela: %d", rel.Dias)
	}

	antes := application.TetoGovernanca
	application.TetoGovernanca = 1
	t.Cleanup(func() { application.TetoGovernanca = antes })
	if rel, err := c.base.RelatorioPorSecretaria(ctx, c.tecnico, 30); err != nil || !rel.Truncado || rel.Total.Total != 1 {
		t.Fatalf("amostra truncada: %+v %v", rel, err)
	}
	application.TetoGovernanca = antes

	if _, err := c.base.RelatorioPorSecretaria(ctx, c.servidor, 0); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("sem atender: %v", err)
	}
	for _, m := range []string{"ListarDoPeriodo", "PassagensDe", "AvaliacoesDe", "SecretariasDe", "MetasXLA", "TarefasDoPeriodo"} {
		if _, err := c.com(falhando(m), eventosFalham{}).RelatorioPorSecretaria(ctx, c.tecnico, 0); !errors.Is(err, errFalha) {
			t.Errorf("falha em %s: %v", m, err)
		}
	}
}

func TestOLADaEquipe(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	zero, doze := 0, 12
	if _, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "X", UnidadeID: &c.unidade, OLAHoras: &zero}); !errors.Is(err, domain.ErrOLAInvalido) {
		t.Fatalf("criar com OLA zero: %v", err)
	}
	e, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "Patrimônio", UnidadeID: &c.unidade, OLAHoras: &doze})
	if err != nil || *e.OLAHoras != 12 {
		t.Fatalf("criar com OLA: %+v %v", e, err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, e.ID)
	})
	if _, err := c.base.AtualizarEquipe(ctx, admin(), e.ID, application.EquipeEntrada{Nome: "Patrimônio", UnidadeID: &c.unidade, OLAHoras: &zero}); !errors.Is(err, domain.ErrOLAInvalido) {
		t.Fatalf("atualizar com OLA zero: %v", err)
	}
	sem, err := c.base.AtualizarEquipe(ctx, admin(), e.ID, application.EquipeEntrada{Nome: "Patrimônio", UnidadeID: &c.unidade})
	if err != nil || sem.OLAHoras != nil {
		t.Fatalf("tirar o OLA: %+v %v", sem, err)
	}
}

func TestHandlerDoRelatorio(t *testing.T) {
	c := novoCenario(t)
	h := transport.NewHandlers(c.base, silencio)
	for nome, k := range map[string]struct {
		alvo   string
		espera int
	}{
		"padrão":        {"/x", http.StatusOK},
		"com dias":      {"/x?dias=30", http.StatusOK},
		"dias inválido": {"/x?dias=abc", http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		h.RelatorioPorSecretaria(rec, pedido(t, http.MethodGet, k.alvo, "", nil, c.tecnico))
		if rec.Code != k.espera {
			t.Errorf("%s: %d %s", nome, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	h.RelatorioPorSecretaria(rec, pedido(t, http.MethodGet, "/x", "", nil, c.servidor))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sem atender: %d", rec.Code)
	}
}
