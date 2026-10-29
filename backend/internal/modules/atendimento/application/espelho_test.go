package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// Sincronizando com o GLPI, o chamado importado muda lá até a virada
// (ADR 081): aqui é só leitura, e as respostas por e-mail e WhatsApp não
// entram nele.
func TestEspelhoDoGLPI(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	dbtest.Exclusivo(t, c.pool, "glpi")
	sol := c.emAtendimento(t)
	ativo := true
	var falha error
	svc := c.com(falhando(), eventosFalham{}).WithAssistente(&iaFalsa{disponivel: true}).
		WithEspelho(func(context.Context) (bool, error) { return ativo, falha })

	// Nascido aqui, segue normal.
	if _, err := svc.Comentar(ctx, c.tecnico, sol.ID, "Olhando", false); err != nil {
		t.Fatalf("chamado daqui: %v", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO atendimento_importacao (origem, tipo, id_origem, destino) VALUES ('glpi', 'solicitacao', $1, $2)`,
		"espelho-"+uuid.NewString()[:8], sol.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_importacao WHERE destino = $1`, sol.ID)
	})

	if _, err := svc.Comentar(ctx, c.tecnico, sol.ID, "Resolvi", false); !errors.Is(err, domain.ErrEspelho) {
		t.Fatalf("comentar no espelho: %v", err)
	}
	if _, err := svc.RegistrarApontamento(ctx, c.tecnico, sol.ID, domain.EntradaApontamento{Horas: 1, Descricao: "x"}); !errors.Is(err, domain.ErrEspelho) {
		t.Fatalf("apontar no espelho: %v", err)
	}
	if _, err := svc.SugerirClassificacao(ctx, c.tecnico, sol.ID); !errors.Is(err, domain.ErrEspelho) {
		t.Fatalf("IA no espelho: %v", err)
	}
	err := database.WithTx(ctx, c.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := svc.ConcluirPorMudanca(ctx, tx, sol.ID, c.tecnico.UserID, 1, "Mudança", "feita")
		return err
	})
	if !errors.Is(err, domain.ErrEspelho) {
		t.Fatalf("mudança no espelho: %v", err)
	}
	d, err := svc.Obter(ctx, c.tecnico, sol.ID)
	if err != nil || !d.Espelho || len(d.Proximos) != 0 {
		t.Fatalf("detalhe do espelho: %+v %v", d, err)
	}

	// A resposta de quem pediu não entra; pelo WhatsApp, ela sabe por quê.
	quem := c.servidor.UserID
	r, err := svc.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d e então?", sol.Numero), "wamid."+uuid.NewString())
	if err != nil || !strings.Contains(r, "atendido no GLPI") {
		t.Fatalf("WhatsApp no espelho: %q %v", r, err)
	}
	var email string
	if err := c.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, quem).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReceberCorreio(ctx, email, fmt.Sprintf("Re: [#%d] Pedido", sol.Numero), "e então?", "<"+uuid.NewString()+"@x>"); err != nil {
		t.Fatalf("e-mail no espelho: %v", err)
	}
	if got := comentarios(t, c, sol.ID); len(got) != 1 {
		t.Fatalf("só o comentário de antes: %v", got)
	}

	// A leitura do estado da sincronização que falha não deixa mudar.
	falha = errFalha
	if _, err := svc.Comentar(ctx, c.tecnico, sol.ID, "x", false); !errors.Is(err, errFalha) {
		t.Fatalf("falha ao ler a sincronização: %v", err)
	}
	if _, err := svc.Obter(ctx, c.tecnico, sol.ID); !errors.Is(err, errFalha) {
		t.Fatalf("detalhe com falha: %v", err)
	}
	// Na virada, a sincronização desliga e o chamado volta a ser daqui.
	ativo, falha = false, nil
	if _, err := svc.Comentar(ctx, c.tecnico, sol.ID, "Agora é aqui", false); err != nil {
		t.Fatalf("depois da virada: %v", err)
	}
	if d, _ := svc.Obter(ctx, c.tecnico, sol.ID); d.Espelho {
		t.Fatal("depois da virada não é espelho")
	}
}
