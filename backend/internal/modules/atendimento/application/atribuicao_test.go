package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func (f falhaRepo) AbrirAtribuicao(ctx context.Context, db database.DBTX, s, r uuid.UUID, u *uuid.UUID, em time.Time) error {
	if f.falha["AbrirAtribuicao"] {
		return errFalha
	}
	return f.Repository.AbrirAtribuicao(ctx, db, s, r, u, em)
}
func (f falhaRepo) FecharAtribuicao(ctx context.Context, db database.DBTX, s uuid.UUID, em time.Time) error {
	if f.falha["FecharAtribuicao"] {
		return errFalha
	}
	return f.Repository.FecharAtribuicao(ctx, db, s, em)
}
func (f falhaRepo) AtribuicoesDoPeriodo(ctx context.Context, db database.DBTX, r bool, u []uuid.UUID, d time.Time) ([]domain.Atribuicao, error) {
	if f.falha["AtribuicoesDoPeriodo"] {
		return nil, errFalha
	}
	return f.Repository.AtribuicoesDoPeriodo(ctx, db, r, u, d)
}

func trechos(t *testing.T, c cenario, sol uuid.UUID) (abertos, fechados int) {
	t.Helper()
	if err := c.pool.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE fim IS NULL), count(*) FILTER (WHERE fim IS NOT NULL)
		FROM atendimento_atribuicoes WHERE solicitacao_id = $1`, sol).Scan(&abertos, &fechados); err != nil {
		t.Fatal(err)
	}
	return abertos, fechados
}

func TestAtribuicoesAcompanhamOChamado(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.emAtendimento(t)
	if a, f := trechos(t, c, sol.ID); a != 1 || f != 0 {
		t.Fatalf("assumir abre o trecho: %d %d", a, f)
	}
	outro := atendente(c.unidade)
	outro.UserID = dbtest.User(t, c.pool)
	eq, err := c.base.CriarEquipe(ctx, admin(), application.EquipeEntrada{Nome: "N2", UnidadeID: &c.unidade, Membros: []uuid.UUID{outro.UserID}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, eq.ID)
	})
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, outro.UserID); err != nil {
		t.Fatal(err)
	}
	if a, f := trechos(t, c, sol.ID); a != 1 || f != 1 {
		t.Fatalf("passar adiante fecha um e abre outro: %d %d", a, f)
	}
	concluir(t, c, sol)
	if a, f := trechos(t, c, sol.ID); a != 0 || f != 2 {
		t.Fatalf("concluir fecha: %d %d", a, f)
	}
	if _, err := c.base.Transicionar(ctx, c.servidor, sol.ID, domain.EstadoEmAtendimento); err != nil {
		t.Fatal(err)
	}
	if a, _ := trechos(t, c, sol.ID); a != 1 {
		t.Fatal("reabrir abre de novo")
	}
	rel, err := c.base.RelatorioPorSecretaria(ctx, c.tecnico, 0)
	if err != nil {
		t.Fatal(err)
	}
	porNome := map[string]domain.LinhaEquipe{}
	for _, e := range rel.Equipes {
		porNome[e.Nome] = e
	}
	if porNome["N1"].Chamados != 1 || porNome["N2"].Chamados != 1 || porNome["N2"].ComAEquipe != 1 {
		t.Fatalf("OLA dos chamados: %+v", rel.Equipes)
	}

	for nome, k := range map[string]struct {
		err error
	}{
		"falha ao fechar na troca": {func() error {
			_, err := c.com(falhando("FecharAtribuicao"), eventosFalham{}).Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID)
			return err
		}()},
		"falha ao abrir na troca": {func() error {
			_, err := c.com(falhando("AbrirAtribuicao"), eventosFalham{}).Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID)
			return err
		}()},
		"falha ao fechar na conclusão": {func() error {
			_, err := c.com(falhando("FecharAtribuicao"), eventosFalham{}).Transicionar(ctx, c.tecnico, sol.ID, domain.EstadoResolvido)
			return err
		}()},
		"falha no relatório": {func() error {
			_, err := c.com(falhando("AtribuicoesDoPeriodo"), eventosFalham{}).RelatorioPorSecretaria(ctx, c.tecnico, 0)
			return err
		}()},
	} {
		if !errors.Is(k.err, errFalha) {
			t.Errorf("%s: %v", nome, k.err)
		}
	}
	// Com trechos, a leitura das passagens também pode falhar.
	if _, err := c.com(falhando("PassagensDe"), eventosFalham{}).RelatorioPorSecretaria(ctx, c.tecnico, 0); !errors.Is(err, errFalha) {
		t.Fatalf("passagens dos trechos: %v", err)
	}
}
