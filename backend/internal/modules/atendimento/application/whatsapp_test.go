package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// waFalho é o repositório real com falhas nas leituras da entrada.
type waFalho struct {
	domain.Repository
	numero, lista, reserva bool
}

func (f waFalho) SolicitacaoPorNumero(ctx context.Context, db database.DBTX, n int64) (domain.Solicitacao, error) {
	if f.numero {
		return domain.Solicitacao{}, errFalha
	}
	return f.Repository.SolicitacaoPorNumero(ctx, db, n)
}

func (f waFalho) ListSolicitacoes(ctx context.Context, db database.DBTX, filtro domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	if f.lista {
		return nil, 0, errFalha
	}
	return f.Repository.ListSolicitacoes(ctx, db, filtro)
}

func (f waFalho) ReservarCorreio(ctx context.Context, db database.DBTX, id string, sol uuid.UUID) (bool, error) {
	if f.reserva {
		return false, errFalha
	}
	return f.Repository.ReservarCorreio(ctx, db, id, sol)
}

func comentarios(t *testing.T, c cenario, sol uuid.UUID) []string {
	t.Helper()
	rows, err := c.pool.Query(context.Background(), `SELECT texto FROM atendimento_comentarios WHERE solicitacao_id = $1 ORDER BY created_at`, sol)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestRespostaPeloWhatsApp(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quem := c.servidor.UserID
	id := func() string { return "wamid." + uuid.NewString() }

	// Sem chamado em andamento, a pessoa sabe como abrir.
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "oi", id()); err != nil || !strings.Contains(r, "não tem chamado") {
		t.Fatalf("sem chamado: %q %v", r, err)
	}

	// Um só em andamento: a mensagem entra nele, uma vez.
	sol := c.abrir(t)
	mensagem := id()
	r, err := c.base.ReceberWhatsApp(ctx, quem, "  voltou a funcionar  ", mensagem)
	if err != nil || r != fmt.Sprintf("Recebemos a sua mensagem no chamado #%d. A equipe responde por aqui e na Central de Atendimento.", sol.Numero) {
		t.Fatalf("único: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "voltou a funcionar", mensagem); err != nil || r != "" {
		t.Fatalf("reentrega: %q %v", r, err)
	}
	if got := comentarios(t, c, sol.ID); len(got) != 1 || got[0] != "Resposta pelo WhatsApp:\nvoltou a funcionar" {
		t.Fatalf("comentário: %v", got)
	}

	// Dois em andamento: sem número, pede o número; com ele, entra no certo.
	outro := c.abrir(t)
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "e agora?", id()); err != nil || !strings.Contains(r, "mais de um chamado") {
		t.Fatalf("vários: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d obrigado", outro.Numero), id()); err != nil ||
		!strings.Contains(r, fmt.Sprintf("#%d", outro.Numero)) || len(comentarios(t, c, outro.ID)) != 1 {
		t.Fatalf("pelo número: %q %v", r, err)
	}

	// Quem atende responde pela central; quem não participa, não responde.
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, c.tecnico.UserID, fmt.Sprintf("#%d já resolvo", sol.Numero), id()); err != nil || !strings.Contains(r, "responda pela Central") {
		t.Fatalf("responsável: %q %v", r, err)
	}
	estranho := dbtest.User(t, c.pool)
	if r, err := c.base.ReceberWhatsApp(ctx, estranho, fmt.Sprintf("#%d", sol.Numero), id()); err != nil || !strings.Contains(r, "não é seu") {
		t.Fatalf("estranho: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "#999999999 oi", id()); err != nil || !strings.Contains(r, "Não encontramos o chamado #999999999") {
		t.Fatalf("inexistente: %q %v", r, err)
	}

	// Encerrado: avisa; mensagem vazia, sem id ou grande demais: nada.
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET estado = 'encerrado' WHERE id = $1`, outro.ID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d e isso?", outro.Numero), id()); err != nil || !strings.Contains(r, "já foi encerrado") {
		t.Fatalf("encerrado: %q %v", r, err)
	}
	for _, caso := range []struct{ texto, id string }{
		{" ", id()}, {"oi", " "}, {"oi", strings.Repeat("x", 300)}, {"#99999999999999999999999 oi", id()},
		{fmt.Sprintf("#%d %s", sol.Numero, strings.Repeat("a", domain.TextoMax)), id()},
	} {
		if r, err := c.base.ReceberWhatsApp(ctx, quem, caso.texto, caso.id); err != nil || r != "" {
			t.Errorf("%.30q: %q %v", caso.texto, r, err)
		}
	}
}

func TestRespostaPeloWhatsAppComFalhas(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	quem := c.servidor.UserID
	real := infrastructure.NewRepository()
	for nome, repo := range map[string]domain.Repository{
		"número":  waFalho{Repository: real, numero: true},
		"lista":   waFalho{Repository: real, lista: true},
		"reserva": waFalho{Repository: real, reserva: true},
	} {
		s := application.NewService(c.pool, repo, eventosFalham{}, nil)
		texto := "oi"
		if nome == "número" {
			texto = fmt.Sprintf("#%d oi", sol.Numero)
		}
		if _, err := s.ReceberWhatsApp(ctx, quem, texto, "wamid."+uuid.NewString()); !errors.Is(err, errFalha) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	// O comentário que falha libera a reserva: a reentrega do provedor grava.
	mensagem := "wamid." + uuid.NewString()
	falho := application.NewService(c.pool, real, eventosFalham{tipo: application.EventoSolicitacaoComentada}, nil)
	if _, err := falho.ReceberWhatsApp(ctx, quem, "oi", mensagem); err == nil {
		t.Fatal("falha ao comentar volta")
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "oi", mensagem); err != nil || !strings.Contains(r, "Recebemos") {
		t.Fatalf("reentrega depois da falha: %q %v", r, err)
	}
}
