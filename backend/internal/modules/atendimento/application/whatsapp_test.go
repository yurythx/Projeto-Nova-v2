package application_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// waFalho Ã© o repositÃ³rio real com falhas nas leituras da entrada.
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

	// Sem chamado em andamento, a mensagem abre o roteiro. Cancelar devolve
	// a conversa para o comentário do chamado que nascer em seguida.
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "oi", id()); err != nil || !strings.Contains(r.Texto, "Central de Atendimento") || len(r.Opcoes) == 0 {
		t.Fatalf("sem chamado: %+v %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "cancelar", id()); err != nil || !strings.Contains(r.Texto, "cancelada") {
		t.Fatalf("cancelar: %+v %v", r, err)
	}

	// Um só em andamento: a mensagem entra nele, uma vez.
	sol := c.abrir(t)
	mensagem := id()
	r, err := c.base.ReceberWhatsApp(ctx, quem, "  voltou a funcionar  ", mensagem)
	if err != nil || r.Texto != fmt.Sprintf("Recebemos a sua mensagem no chamado #%d. A equipe responde por aqui e na Central de Atendimento.", sol.Numero) {
		t.Fatalf("Ãºnico: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "voltou a funcionar", mensagem); err != nil || r.Texto != "" {
		t.Fatalf("reentrega: %q %v", r, err)
	}
	if got := comentarios(t, c, sol.ID); len(got) != 1 || got[0] != "Resposta pelo WhatsApp:\nvoltou a funcionar" {
		t.Fatalf("comentÃ¡rio: %v", got)
	}

	// Dois em andamento: sem nÃºmero, pede o nÃºmero; com ele, entra no certo.
	outro := c.abrir(t)
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "e agora?", id()); err != nil || !strings.Contains(r.Texto, "mais de um chamado") {
		t.Fatalf("vÃ¡rios: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d obrigado", outro.Numero), id()); err != nil ||
		!strings.Contains(r.Texto, fmt.Sprintf("#%d", outro.Numero)) || len(comentarios(t, c, outro.ID)) != 1 {
		t.Fatalf("pelo nÃºmero: %q %v", r, err)
	}

	// Quem atende responde pela central; quem nÃ£o participa, nÃ£o responde.
	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, c.tecnico.UserID, fmt.Sprintf("#%d jÃ¡ resolvo", sol.Numero), id()); err != nil || !strings.Contains(r.Texto, "responda pela Central") {
		t.Fatalf("responsÃ¡vel: %q %v", r, err)
	}
	estranho := dbtest.User(t, c.pool)
	if r, err := c.base.ReceberWhatsApp(ctx, estranho, fmt.Sprintf("#%d", sol.Numero), id()); err != nil || !strings.Contains(r.Texto, "Confira o") {
		t.Fatalf("estranho: %q %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "#999999999 oi", id()); err != nil || !strings.Contains(r.Texto, "encontramos o chamado #999999999") {
		t.Fatalf("inexistente: %q %v", r, err)
	}

	// Encerrado: avisa; mensagem vazia, sem id ou grande demais: nada.
	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET estado = 'encerrado' WHERE id = $1`, outro.ID); err != nil {
		t.Fatal(err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d e isso?", outro.Numero), id()); err != nil || !strings.Contains(r.Texto, "foi encerrado") {
		t.Fatalf("encerrado: %q %v", r, err)
	}
	for _, caso := range []struct{ texto, id string }{
		{" ", id()}, {"oi", " "}, {"oi", strings.Repeat("x", 300)}, {"#99999999999999999999999 oi", id()},
		{fmt.Sprintf("#%d %s", sol.Numero, strings.Repeat("a", domain.TextoMax)), id()},
	} {
		if r, err := c.base.ReceberWhatsApp(ctx, quem, caso.texto, caso.id); err != nil || r.Texto != "" {
			t.Errorf("%.30q: %q %v", caso.texto, r, err)
		}
	}
}

func TestRespostaPeloWhatsAppCSAT(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	quem := c.servidor.UserID
	id := func() string { return "wamid." + uuid.NewString() }

	if _, err := c.pool.Exec(ctx, "UPDATE atendimento_solicitacoes SET estado = 'resolvido' WHERE id = $1", sol.ID); err != nil {
		t.Fatal(err)
	}

	r, err := c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d ainda falta algo?", sol.Numero), id())
	if err != nil || !strings.Contains(r.Texto, "foi concluído. Para avaliá-lo, envie uma nota de 1 a 5") {
		t.Fatalf("orientacao CSAT: %q %v", r, err)
	}

	r, err = c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d 5 excelente atendimento", sol.Numero), id())
	if err != nil || !strings.Contains(r.Texto, "Agradecemos a sua avaliação (nota 5)") {
		t.Fatalf("avaliacao CSAT: %q %v", r, err)
	}

	r, err = c.base.ReceberWhatsApp(ctx, quem, fmt.Sprintf("#%d nota 4", sol.Numero), id())
	if err != nil || !strings.Contains(r.Texto, "já foi avaliado anteriormente") {
		t.Fatalf("avaliacao duplicada: %q %v", r, err)
	}
}

func TestRespostaPeloWhatsAppComFalhas(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	quem := c.servidor.UserID
	real := infrastructure.NewRepository()
	for nome, repo := range map[string]domain.Repository{
		"nÃºmero": waFalho{Repository: real, numero: true},
		"lista":   waFalho{Repository: real, lista: true},
		"reserva": waFalho{Repository: real, reserva: true},
	} {
		s := application.NewService(c.pool, repo, eventosFalham{}, nil)
		texto := "oi"
		if nome == "nÃºmero" {
			texto = fmt.Sprintf("#%d oi", sol.Numero)
		}
		if _, err := s.ReceberWhatsApp(ctx, quem, texto, "wamid."+uuid.NewString()); !errors.Is(err, errFalha) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	// O comentÃ¡rio que falha libera a reserva: a reentrega do provedor grava.
	mensagem := "wamid." + uuid.NewString()
	falho := application.NewService(c.pool, real, eventosFalham{tipo: application.EventoSolicitacaoComentada}, nil)
	if _, err := falho.ReceberWhatsApp(ctx, quem, "oi", mensagem); err == nil {
		t.Fatal("falha ao comentar volta")
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "oi", mensagem); err != nil || !strings.Contains(r.Texto, "Recebemos") {
		t.Fatalf("reentrega depois da falha: %q %v", r, err)
	}
}

func TestAberturaPeloWhatsApp(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quem := c.servidor.UserID
	id := func() string { return "wamid." + uuid.NewString() }

	// Com chamado em andamento, "novo" abre outro em vez de comentar.
	sol := c.abrir(t)
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "novo", id()); err != nil || len(r.Opcoes) == 0 || !strings.Contains(r.Texto, "Central de Atendimento") {
		t.Fatalf("novo: %+v %v", r, err)
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "cancelar", id()); err != nil || !strings.Contains(r.Texto, "cancelada") {
		t.Fatalf("cancelar o novo: %+v %v", r, err)
	}
	if n := comentarios(t, c, sol.ID); len(n) != 0 {
		t.Fatalf("novo não comentou: %v", n)
	}

	titulo := "Impressora da sala"
	escolherServico(t, c, quem, c.oferta.ID)
	if r, err := c.base.ReceberWhatsApp(ctx, quem, titulo, id()); err != nil || !strings.Contains(r.Texto, "descreva") {
		t.Fatalf("título: %+v %v", r, err)
	}
	descricaoID := id()
	r, err := c.base.ReceberWhatsApp(ctx, quem, "Não imprime desde cedo.", descricaoID)
	if err != nil || !strings.Contains(r.Texto, "Abrimos o chamado #") || !strings.Contains(r.Texto, titulo) {
		t.Fatalf("abertura: %+v %v", r, err)
	}
	if deNovo, err := c.base.ReceberWhatsApp(ctx, quem, "Não imprime desde cedo.", descricaoID); err != nil || deNovo.Texto != "" {
		t.Fatalf("reentrega da abertura: %+v %v", deNovo, err)
	}
	var qtd int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM atendimento_solicitacoes WHERE solicitante_id = $1 AND titulo = $2`, quem, titulo).Scan(&qtd); err != nil || qtd != 1 {
		t.Fatalf("um chamado: %d %v", qtd, err)
	}

	// A resposta seguinte entra no chamado que acabou de nascer, se for o único em andamento
	// além do que o teste já abriu pela Central. Aqui há dois, então pede o número.
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "e a tinta?", id()); err != nil || !strings.Contains(r.Texto, "mais de um chamado") || !strings.Contains(r.Texto, "novo") {
		t.Fatalf("dois chamados: %+v %v", r, err)
	}
}

func TestAberturaPeloWhatsAppComCampo(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quem := c.servidor.UserID
	id := func() string { return "wamid." + uuid.NewString() }
	campos := []domain.Campo{
		{Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true},
		{Rotulo: "Andar", Tipo: "lista", Obrigatorio: true, Opcoes: []string{"Térreo", "1"}},
		{Rotulo: "Obs", Tipo: "texto"},
	}
	o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{
		Nome: "Ramal " + uuid.NewString()[:6], Tipo: domain.TipoRequisicao, UnidadeID: &c.unidade, Campos: &campos,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})

	escolherServico(t, c, quem, o.ID)
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "Trocar ramal", id()); err != nil || !strings.Contains(r.Texto, "descreva") {
		t.Fatalf("título: %+v %v", r, err)
	}
	r, err := c.base.ReceberWhatsApp(ctx, quem, "O ramal não chama.", id())
	if err != nil || !strings.Contains(r.Texto, "Ramal") {
		t.Fatalf("campo: %+v %v", r, err)
	}
	if r, err = c.base.ReceberWhatsApp(ctx, quem, "3a", id()); err != nil || !strings.Contains(r.Texto, "Não entendi") {
		t.Fatalf("ramal inválido: %+v %v", r, err)
	}
	if r, err = c.base.ReceberWhatsApp(ctx, quem, "3450", id()); err != nil || !strings.Contains(r.Texto, "Andar") || len(r.Opcoes) != 2 {
		t.Fatalf("andar: %+v %v", r, err)
	}
	r, err = c.base.ReceberWhatsApp(ctx, quem, "Térreo", id())
	if err != nil || !strings.Contains(r.Texto, "Abrimos o chamado #") || !strings.Contains(r.Texto, "dados opcionais") {
		t.Fatalf("aberto: %+v %v", r, err)
	}
	var respostas string
	if err := c.pool.QueryRow(ctx, `SELECT respostas::text FROM atendimento_solicitacoes WHERE solicitante_id = $1 AND oferta_id = $2`, quem, o.ID).Scan(&respostas); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(respostas, "3450") || !strings.Contains(respostas, "Térreo") {
		t.Fatalf("respostas: %s", respostas)
	}
}

func escolherServico(t *testing.T, c cenario, quem, oferta uuid.UUID) domain.RespostaWhatsApp {
	t.Helper()
	ctx := context.Background()
	var r domain.RespostaWhatsApp
	var err error
	for pagina := 0; pagina < 8; pagina++ {
		msg := "novo"
		if pagina > 0 {
			msg = "mais"
		}
		r, err = c.base.ReceberWhatsApp(ctx, quem, msg, "wamid."+uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		for i, op := range r.Opcoes {
			if op.ID == oferta.String() {
				r, err = c.base.ReceberWhatsApp(ctx, quem, strconv.Itoa(i+1), "wamid."+uuid.NewString())
				if err != nil || !strings.Contains(r.Texto, "título") {
					t.Fatalf("serviço: %+v %v", r, err)
				}
				return r
			}
		}
	}
	t.Fatalf("serviço fora das páginas: %+v", r)
	return r
}
