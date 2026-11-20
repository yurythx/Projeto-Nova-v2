package application_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/domain/pagination"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
)

// waCtrl é o repositório real com falhas e leituras combinadas da entrada.
type waCtrl struct {
	domain.Repository
	falha             map[string]bool
	semOfertas        bool
	semUnidade        bool
	secsVazias        bool
	reservaFalsa      bool
	etapa             string
	inativa           bool
	ausenteNaVez      int
	inativaNaVez      int
	nGet              int
	emailFalha        *uuid.UUID
	reservaCorreioNao bool
}

func controle(falhas ...string) *waCtrl {
	m := map[string]bool{}
	for _, nome := range falhas {
		m[nome] = true
	}
	return &waCtrl{Repository: infrastructure.NewRepository(), falha: m}
}

func (w *waCtrl) ruim(nome string) bool { return w.falha[nome] }

func (w *waCtrl) servico(c cenario) *application.Service {
	return application.NewService(c.pool, w, eventosFalham{}, nil)
}

func (w *waCtrl) RoteiroWhatsApp(ctx context.Context, db database.DBTX, usuario uuid.UUID) (domain.RoteiroWhatsApp, error) {
	if w.ruim("RoteiroWhatsApp") {
		return domain.RoteiroWhatsApp{}, errFalha
	}
	if w.etapa != "" {
		return domain.RoteiroWhatsApp{UsuarioID: usuario, Etapa: w.etapa, UpdatedAt: time.Now().UTC()}, nil
	}
	return w.Repository.RoteiroWhatsApp(ctx, db, usuario)
}

func (w *waCtrl) SalvarRoteiroWhatsApp(ctx context.Context, db database.DBTX, rot domain.RoteiroWhatsApp) error {
	if w.ruim("SalvarRoteiroWhatsApp") {
		return errFalha
	}
	return w.Repository.SalvarRoteiroWhatsApp(ctx, db, rot)
}

func (w *waCtrl) ApagarRoteiroWhatsApp(ctx context.Context, db database.DBTX, usuario uuid.UUID) error {
	if w.ruim("ApagarRoteiroWhatsApp") {
		return errFalha
	}
	return w.Repository.ApagarRoteiroWhatsApp(ctx, db, usuario)
}

func (w *waCtrl) ReservarMensagemWhatsApp(ctx context.Context, db database.DBTX, messageID string) (bool, error) {
	if w.ruim("ReservarMensagemWhatsApp") {
		return false, errFalha
	}
	if w.reservaFalsa {
		return false, nil
	}
	return w.Repository.ReservarMensagemWhatsApp(ctx, db, messageID)
}

func (w *waCtrl) ListOfertasAtivas(ctx context.Context, db database.DBTX) ([]domain.Oferta, error) {
	if w.ruim("ListOfertasAtivas") {
		return nil, errFalha
	}
	if w.semOfertas {
		return nil, nil
	}
	if w.semUnidade {
		return []domain.Oferta{{ID: uuid.New(), Nome: "Solta", Ativa: true, Tipo: domain.TipoIncidente}}, nil
	}
	return w.Repository.ListOfertasAtivas(ctx, db)
}

func (w *waCtrl) GetOferta(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Oferta, error) {
	if w.ruim("GetOferta") {
		return domain.Oferta{}, errFalha
	}
	w.nGet++
	if w.ausenteNaVez > 0 && w.nGet == w.ausenteNaVez {
		return domain.Oferta{}, domain.ErrNaoEncontrado
	}
	o, err := w.Repository.GetOferta(ctx, db, id)
	if err != nil {
		return o, err
	}
	if w.inativa || (w.inativaNaVez > 0 && w.nGet == w.inativaNaVez) {
		o.Ativa = false
	}
	return o, nil
}

func (w *waCtrl) ListSolicitacoes(ctx context.Context, db database.DBTX, f domain.FiltroSolicitacao) ([]domain.Solicitacao, int64, error) {
	if w.ruim("ListSolicitacoes") {
		return nil, 0, errFalha
	}
	return w.Repository.ListSolicitacoes(ctx, db, f)
}

func (w *waCtrl) InsertSolicitacao(ctx context.Context, db database.DBTX, s domain.Solicitacao) (domain.Solicitacao, error) {
	if w.ruim("InsertSolicitacao") {
		return domain.Solicitacao{}, errFalha
	}
	return w.Repository.InsertSolicitacao(ctx, db, s)
}

func (w *waCtrl) SecretariasDe(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID]domain.Secretaria, error) {
	if w.ruim("SecretariasDe") {
		return nil, errFalha
	}
	if w.secsVazias {
		return map[uuid.UUID]domain.Secretaria{}, nil
	}
	return w.Repository.SecretariasDe(ctx, db, ids)
}

func (w *waCtrl) EmailDe(ctx context.Context, db database.DBTX, user uuid.UUID) (string, error) {
	if w.ruim("EmailDe") || (w.emailFalha != nil && user == *w.emailFalha) {
		return "", errFalha
	}
	return w.Repository.EmailDe(ctx, db, user)
}

func (w *waCtrl) SolicitacaoPorNumero(ctx context.Context, db database.DBTX, n int64) (domain.Solicitacao, error) {
	if w.ruim("SolicitacaoPorNumero") {
		return domain.Solicitacao{}, errFalha
	}
	return w.Repository.SolicitacaoPorNumero(ctx, db, n)
}

func (w *waCtrl) ReservarCorreio(ctx context.Context, db database.DBTX, id string, sol uuid.UUID) (bool, error) {
	if w.ruim("ReservarCorreio") {
		return false, errFalha
	}
	if w.reservaCorreioNao {
		return false, nil
	}
	return w.Repository.ReservarCorreio(ctx, db, id, sol)
}

func msgID() string { return "wamid." + uuid.NewString() }

func emailDe(t *testing.T, c cenario, user uuid.UUID) string {
	t.Helper()
	var email string
	if err := c.pool.QueryRow(context.Background(), `SELECT email FROM users WHERE id = $1`, user).Scan(&email); err != nil {
		t.Fatal(err)
	}
	return email
}

func tituloNoMenu(t *testing.T, c cenario, quem, oferta uuid.UUID) string {
	t.Helper()
	ctx := context.Background()
	for pagina := 0; pagina < 200; pagina++ {
		msg := "novo"
		if pagina > 0 {
			msg = "mais"
		}
		r, err := c.base.ReceberWhatsApp(ctx, quem, msg, msgID())
		if err != nil {
			t.Fatal(err)
		}
		for _, op := range r.Opcoes {
			if op.ID == oferta.String() {
				return op.Titulo
			}
		}
		if !temMaisOfertas(r) {
			break
		}
	}
	t.Fatal("serviço fora das páginas")
	return ""
}

func naFrente(t *testing.T, c cenario, id uuid.UUID) {
	t.Helper()
	nome := "!" + strings.ReplaceAll(id.String(), "-", "")[:10]
	if _, err := c.pool.Exec(context.Background(), `UPDATE atendimento_ofertas SET nome = $2, ordem = 0, ativa = true WHERE id = $1`, id, nome); err != nil {
		t.Fatal(err)
	}
}

func gravarRoteiro(t *testing.T, c cenario, rot domain.RoteiroWhatsApp) {
	t.Helper()
	if rot.UsuarioID == uuid.Nil {
		rot.UsuarioID = c.servidor.UserID
	}
	if rot.UpdatedAt.IsZero() {
		rot.UpdatedAt = time.Now().UTC()
	}
	if err := infrastructure.NewRepository().SalvarRoteiroWhatsApp(context.Background(), c.pool, rot); err != nil {
		t.Fatal(err)
	}
}

func etapaDoRoteiro(t *testing.T, c cenario) string {
	t.Helper()
	var etapa string
	err := c.pool.QueryRow(context.Background(), `SELECT etapa FROM atendimento_whatsapp_roteiro WHERE usuario_id = $1`, c.servidor.UserID).Scan(&etapa)
	if err != nil {
		t.Fatal(err)
	}
	return etapa
}

func TestRamosDaAberturaWhatsApp(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quem := c.servidor.UserID
	t.Cleanup(func() {
		_ = infrastructure.NewRepository().ApagarRoteiroWhatsApp(context.Background(), c.pool, quem)
	})

	t.Run("roteiro vencido recomeça", func(t *testing.T) {
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{
			Etapa: domain.RoteiroTitulo, Titulo: "velho", UpdatedAt: time.Now().Add(-25 * time.Hour),
		})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "Impressora da sala", msgID())
		if err != nil || strings.Contains(r.Texto, "descreva") || len(r.Opcoes) == 0 {
			t.Fatalf("vencido: %+v %v", r, err)
		}
	})

	t.Run("roteiro já feito comenta o chamado", func(t *testing.T) {
		sol := c.abrir(t)
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroFeito})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "preciso de ajuda", msgID())
		if err != nil || !strings.Contains(r.Texto, "Recebemos") || len(comentarios(t, c, sol.ID)) != 1 {
			t.Fatalf("feito: %+v %v", r, err)
		}
	})

	t.Run("opção que não existe", func(t *testing.T) {
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroOferta})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "nenhuma-opcao", msgID())
		if err != nil || !strings.Contains(r.Texto, "Não encontrei") || len(r.Opcoes) == 0 {
			t.Fatalf("opção: %+v %v", r, err)
		}
	})

	t.Run("título e descrição longos", func(t *testing.T) {
		escolherServico(t, c, quem, c.oferta.ID)
		r, err := c.base.ReceberWhatsApp(ctx, quem, strings.Repeat("a", domain.NomeMax+1), msgID())
		if err != nil || !strings.Contains(r.Texto, "160") {
			t.Fatalf("título: %+v %v", r, err)
		}
		if _, err = c.base.ReceberWhatsApp(ctx, quem, "Impressora", msgID()); err != nil {
			t.Fatal(err)
		}
		r, err = c.base.ReceberWhatsApp(ctx, quem, strings.Repeat("b", domain.TextoMax+1), msgID())
		if err != nil || !strings.Contains(r.Texto, "4000") {
			t.Fatalf("descrição: %+v %v", r, err)
		}
	})

	t.Run("sigla da secretaria no menu", func(t *testing.T) {
		if _, err := c.pool.Exec(ctx, `UPDATE unidades SET sigla = 'TI' WHERE id = $1`, c.unidade); err != nil {
			t.Fatal(err)
		}
		if titulo := tituloNoMenu(t, c, quem, c.oferta.ID); !strings.Contains(titulo, " · TI") {
			t.Fatalf("sigla: %q", titulo)
		}
	})

	t.Run("oferta inativa na escolha volta ao menu", func(t *testing.T) {
		naFrente(t, c, c.oferta.ID)
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroOferta})
		w := controle()
		w.inativa = true
		r, err := w.servico(c).ReceberWhatsApp(ctx, quem, "1", msgID())
		if err != nil || strings.Contains(r.Texto, "título") || len(r.Opcoes) == 0 {
			t.Fatalf("inativa: %+v %v", r, err)
		}
	})

	t.Run("oferta ausente na escolha volta ao menu", func(t *testing.T) {
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroOferta})
		w := controle()
		w.ausenteNaVez = 1
		r, err := w.servico(c).ReceberWhatsApp(ctx, quem, "1", msgID())
		if err != nil || strings.Contains(r.Texto, "título") || len(r.Opcoes) == 0 {
			t.Fatalf("ausente: %+v %v", r, err)
		}
	})

	t.Run("lista longa vem em texto", func(t *testing.T) {
		opcoes := make([]string, 11)
		for i := range opcoes {
			opcoes[i] = fmt.Sprintf("sala%d", i+1)
		}
		campos := []domain.Campo{{Rotulo: "Sala", Tipo: "lista", Obrigatorio: true, Opcoes: opcoes}}
		o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{
			Nome: "Lista " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade, Campos: &campos,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, o.ID)
			_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		})
		escolherServico(t, c, quem, o.ID)
		if _, err := c.base.ReceberWhatsApp(ctx, quem, "Cadeira", msgID()); err != nil {
			t.Fatal(err)
		}
		r, err := c.base.ReceberWhatsApp(ctx, quem, "Quebrou o encosto.", msgID())
		if err != nil || len(r.Opcoes) != 0 || !strings.Contains(r.Texto, "Sala") {
			t.Fatalf("lista: %+v %v", r, err)
		}
		r, err = c.base.ReceberWhatsApp(ctx, quem, "sala3", msgID())
		if err != nil || !strings.Contains(r.Texto, "Abrimos o chamado") {
			t.Fatalf("aberto: %+v %v", r, err)
		}
	})

	t.Run("respostas que não fecham o chamado", func(t *testing.T) {
		campos := []domain.Campo{{Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true}}
		o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{
			Nome: "Campo " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade, Campos: &campos,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
		})
		id := o.ID
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{
			Etapa: domain.RoteiroCampo, OfertaID: &id, Titulo: "Ok", Descricao: "Ok", CampoIndice: 9,
		})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "1", msgID())
		if err != nil || !strings.Contains(r.Texto, "Não consegui guardar") {
			t.Fatalf("respostas: %+v %v", r, err)
		}
	})

	t.Run("título vazio na conclusão pede de novo", func(t *testing.T) {
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{
			Etapa: domain.RoteiroCampo, OfertaID: &c.oferta.ID, Descricao: "Ok", CampoIndice: 3,
		})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "x", msgID())
		if err != nil || !strings.Contains(r.Texto, "título de novo") {
			t.Fatalf("título vazio: %+v %v", r, err)
		}
	})

	t.Run("descrição sem oferta volta ao menu", func(t *testing.T) {
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroDescricao, Titulo: "Ok"})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "Aconteceu agora.", msgID())
		if err != nil || strings.Contains(r.Texto, "Abrimos") || len(r.Opcoes) == 0 {
			t.Fatalf("sem oferta: %+v %v", r, err)
		}
	})

	t.Run("oferta inativa na descrição volta ao menu", func(t *testing.T) {
		if _, err := c.pool.Exec(ctx, `UPDATE atendimento_ofertas SET ativa = false WHERE id = $1`, c.oferta.ID); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = c.pool.Exec(context.Background(), `UPDATE atendimento_ofertas SET ativa = true WHERE id = $1`, c.oferta.ID)
		})
		gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroDescricao, OfertaID: &c.oferta.ID, Titulo: "Ok"})
		r, err := c.base.ReceberWhatsApp(ctx, quem, "Aconteceu agora.", msgID())
		if err != nil || strings.Contains(r.Texto, "Abrimos") || len(r.Opcoes) == 0 {
			t.Fatalf("inativa: %+v %v", r, err)
		}
	})
}

func TestRamosDaAberturaComFalha(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	quem := c.servidor.UserID
	t.Cleanup(func() {
		_ = infrastructure.NewRepository().ApagarRoteiroWhatsApp(context.Background(), c.pool, quem)
	})

	for _, nome := range []string{"RoteiroWhatsApp", "ListOfertasAtivas", "ReservarMensagemWhatsApp", "ListSolicitacoes", "SalvarRoteiroWhatsApp"} {
		w := controle(nome)
		if _, err := w.servico(c).ReceberWhatsApp(ctx, quem, "oi", msgID()); !errors.Is(err, errFalha) {
			t.Errorf("%s: %v", nome, err)
		}
	}

	w := controle()
	w.semOfertas = true
	r, err := w.servico(c).ReceberWhatsApp(ctx, quem, "oi", msgID())
	if err != nil || !strings.Contains(r.Texto, "Não há serviço") {
		t.Fatalf("catálogo vazio: %+v %v", r, err)
	}

	w = controle()
	w.etapa = "outra"
	r, err = w.servico(c).ReceberWhatsApp(ctx, quem, "oi", msgID())
	if err != nil || strings.Contains(r.Texto, "Não encontrei") || len(r.Opcoes) == 0 {
		t.Fatalf("etapa desconhecida: %+v %v", r, err)
	}

	w = controle()
	w.reservaFalsa = true
	r, err = w.servico(c).ReceberWhatsApp(ctx, quem, "oi", msgID())
	if err != nil || r.Texto != "" || len(r.Opcoes) != 0 {
		t.Fatalf("reserva recusada: %+v %v", r, err)
	}

	// A gravação que falha devolve a mensagem: a mesma entra em seguida.
	mensagem := msgID()
	if _, err := controle("SalvarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "novo", mensagem); err == nil {
		t.Fatal("salvar")
	}
	if r, err := c.base.ReceberWhatsApp(ctx, quem, "oi", mensagem); err != nil || len(r.Opcoes) == 0 {
		t.Fatalf("reentrega: %+v %v", r, err)
	}

	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroTitulo})
	if _, err := controle("SalvarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "Cadeira quebrada", msgID()); err == nil {
		t.Fatal("salvar o título")
	}
	if _, err := controle("ApagarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "cancelar", msgID()); err == nil {
		t.Fatal("apagar ao cancelar")
	}

	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroOferta, UpdatedAt: time.Now().Add(-25 * time.Hour)})
	if _, err := controle("ApagarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "oi", msgID()); err == nil {
		t.Fatal("apagar o vencido")
	}

	naFrente(t, c, c.oferta.ID)
	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroOferta})
	if _, err := controle("ListOfertasAtivas").servico(c).ReceberWhatsApp(ctx, quem, "1", msgID()); err == nil {
		t.Fatal("catálogo na escolha")
	}
	if _, err := controle("GetOferta").servico(c).ReceberWhatsApp(ctx, quem, "1", msgID()); err == nil {
		t.Fatal("oferta na escolha")
	}
	if _, err := controle("SalvarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "1", msgID()); err == nil {
		t.Fatal("salvar a escolha")
	}

	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroDescricao, OfertaID: &c.oferta.ID, Titulo: "Ok"})
	if _, err := controle("GetOferta").servico(c).ReceberWhatsApp(ctx, quem, "Parou agora.", msgID()); err == nil {
		t.Fatal("oferta na descrição")
	}
	if _, err := controle("InsertSolicitacao").servico(c).ReceberWhatsApp(ctx, quem, "Parou agora.", msgID()); err == nil {
		t.Fatal("abrir")
	}

	w = controle()
	w.ausenteNaVez = 2
	r, err = w.servico(c).ReceberWhatsApp(ctx, quem, "Parou agora.", msgID())
	if err != nil || strings.Contains(r.Texto, "Abrimos") || len(r.Opcoes) == 0 {
		t.Fatalf("sumiu ao abrir: %+v %v", r, err)
	}

	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroDescricao, OfertaID: &c.oferta.ID, Titulo: "Ok"})
	w = controle()
	w.inativaNaVez = 2
	r, err = w.servico(c).ReceberWhatsApp(ctx, quem, "Parou agora.", msgID())
	if err != nil || strings.Contains(r.Texto, "Abrimos") || len(r.Opcoes) == 0 {
		t.Fatalf("inativa ao abrir: %+v %v", r, err)
	}

	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroDescricao, OfertaID: &c.oferta.ID, Titulo: "Cadeira"})
	r, err = controle("ApagarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "O encosto quebrou.", msgID())
	if err != nil || !strings.Contains(r.Texto, "Abrimos o chamado") || etapaDoRoteiro(t, c) != domain.RoteiroFeito {
		t.Fatalf("feito quando não apaga: %+v %s %v", r, etapaDoRoteiro(t, c), err)
	}

	campos := []domain.Campo{
		{Rotulo: "Ramal", Tipo: "numero", Obrigatorio: true},
		{Rotulo: "Sala", Tipo: "texto", Obrigatorio: true},
	}
	o, err := c.base.CriarOferta(ctx, admin(), application.OfertaEntrada{
		Nome: "Dois " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade, Campos: &campos,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})
	gravarRoteiro(t, c, domain.RoteiroWhatsApp{
		Etapa: domain.RoteiroCampo, OfertaID: &o.ID, Titulo: "Ramal", Descricao: "Mudo", CampoIndice: 0,
	})
	if _, err := controle("SalvarRoteiroWhatsApp").servico(c).ReceberWhatsApp(ctx, quem, "3450", msgID()); err == nil {
		t.Fatal("salvar entre campos")
	}
	gravarRoteiro(t, c, domain.RoteiroWhatsApp{Etapa: domain.RoteiroCampo, Titulo: "Ramal", Descricao: "Mudo"})
	r, err = c.base.ReceberWhatsApp(ctx, quem, "3450", msgID())
	if err != nil || strings.Contains(r.Texto, "Abrimos") || len(r.Opcoes) == 0 {
		t.Fatalf("campo sem oferta: %+v %v", r, err)
	}
}

func TestRamosDoCorreio(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.abrir(t)
	email := emailDe(t, c, c.servidor.UserID)
	assunto := fmt.Sprintf("Re: [#%d] Pedido", sol.Numero)
	id := func() string { return "<" + uuid.NewString() + "@correio.test>" }

	for _, caso := range []struct{ de, assunto, corpo, id string }{
		{email, assunto, "oi", ""},
		{email, assunto, "oi", strings.Repeat("a", 201)},
		{email, "sem número", "oi", id()},
		{email, "#99999999999999999999", "oi", id()},
		{email, "Re: [#999999991] sumiu", "oi", id()},
	} {
		if err := c.base.ReceberCorreio(ctx, caso.de, caso.assunto, caso.corpo, caso.id); err != nil {
			t.Fatalf("ignorado %q: %v", caso.assunto, err)
		}
	}
	if n := comentarios(t, c, sol.ID); len(n) != 0 {
		t.Fatalf("não devia comentar: %v", n)
	}

	if err := c.base.ReceberCorreio(ctx, strings.ToUpper(email), assunto, "  ", id()); err != nil {
		t.Fatal(err)
	}
	if got := comentarios(t, c, sol.ID); len(got) != 1 || !strings.Contains(got[0], "(mensagem sem texto)") {
		t.Fatalf("vazio: %v", got)
	}
	if err := c.base.ReceberCorreio(ctx, "outro@nova.test", assunto, "oi", id()); err != nil {
		t.Fatal(err)
	}
	if len(comentarios(t, c, sol.ID)) != 1 {
		t.Fatal("remetente desconhecido entrou")
	}

	dup := id()
	if err := c.base.ReceberCorreio(ctx, email, assunto, "obrigado", dup); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ReceberCorreio(ctx, email, assunto, "obrigado de novo", dup); err != nil {
		t.Fatal(err)
	}
	if len(comentarios(t, c, sol.ID)) != 2 {
		t.Fatal("reentrega do correio")
	}

	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	emailTec := emailDe(t, c, c.tecnico.UserID)
	if err := c.base.ReceberCorreio(ctx, emailTec, assunto, "da equipe", id()); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ReceberCorreio(ctx, "ninguem@nova.test", assunto, "alheio", id()); err != nil {
		t.Fatal(err)
	}
	if len(comentarios(t, c, sol.ID)) != 3 {
		t.Fatalf("responsável: %v", comentarios(t, c, sol.ID))
	}

	if _, err := c.pool.Exec(ctx, `UPDATE atendimento_solicitacoes SET estado = 'encerrado' WHERE id = $1`, sol.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.base.ReceberCorreio(ctx, email, assunto, "tarde", id()); err != nil {
		t.Fatal(err)
	}
	longo := c.abrir(t)
	if err := c.base.ReceberCorreio(ctx, email, fmt.Sprintf("Re: [#%d] Pedido", longo.Numero), strings.Repeat("a", domain.TextoMax), id()); err != nil {
		t.Fatal(err)
	}
	if len(comentarios(t, c, sol.ID)) != 3 || len(comentarios(t, c, longo.ID)) != 0 {
		t.Fatal("encerrado ou longo entrou")
	}

	svc := c.com(falhando(), eventosFalham{application.EventoSolicitacaoComentada})
	outra := id()
	if err := svc.ReceberCorreio(ctx, email, fmt.Sprintf("Re: [#%d] Pedido", longo.Numero), "de novo", outra); err == nil {
		t.Fatal("evento do comentário")
	}
	if err := c.base.ReceberCorreio(ctx, email, fmt.Sprintf("Re: [#%d] Pedido", longo.Numero), "de novo", outra); err != nil {
		t.Fatalf("reentrega depois da falha: %v", err)
	}
	if len(comentarios(t, c, longo.ID)) != 1 {
		t.Fatalf("comentário liberado: %v", comentarios(t, c, longo.ID))
	}

	if err := controle("SolicitacaoPorNumero").servico(c).ReceberCorreio(ctx, email, assunto, "x", id()); !errors.Is(err, errFalha) {
		t.Fatalf("número: %v", err)
	}
	if err := controle("EmailDe").servico(c).ReceberCorreio(ctx, email, assunto, "x", id()); !errors.Is(err, errFalha) {
		t.Fatalf("e-mail de quem abriu: %v", err)
	}
	w := controle()
	w.emailFalha = &c.tecnico.UserID
	if err := w.servico(c).ReceberCorreio(ctx, emailTec, assunto, "x", id()); !errors.Is(err, errFalha) {
		t.Fatalf("e-mail de quem atende: %v", err)
	}
	if err := controle("ReservarCorreio").servico(c).ReceberCorreio(ctx, email, fmt.Sprintf("[#%d] x", longo.Numero), "x", id()); !errors.Is(err, errFalha) {
		t.Fatalf("reserva: %v", err)
	}
	w = controle()
	w.reservaCorreioNao = true
	antes := len(comentarios(t, c, longo.ID))
	if err := w.servico(c).ReceberCorreio(ctx, email, fmt.Sprintf("[#%d] x", longo.Numero), "x", id()); err != nil || len(comentarios(t, c, longo.ID)) != antes {
		t.Fatalf("reserva recusada: %v", err)
	}

	if err := c.com(falhando(), eventosFalham{}).RunCorreio(ctx); err != nil {
		t.Fatal(err)
	}
	carta := mail.Carta{UID: "1", De: email, Assunto: fmt.Sprintf("Re: [#%d] Pedido", longo.Numero), Corpo: "pela caixa", MessageID: id()}
	correrCorreio(t, c.com(falhando(), eventosFalham{}), &caixaTeste{cartas: []mail.Carta{carta}}, false)
	if len(comentarios(t, c, longo.ID)) != 2 {
		t.Fatalf("caixa: %v", comentarios(t, c, longo.ID))
	}
	correrCorreio(t, c.com(falhando(), eventosFalham{}), &caixaTeste{}, true)
	correrCorreio(t, c.com(falhando(), eventosFalham{}), &caixaTeste{lerErr: errFalha}, true)
	correrCorreio(t, c.com(falhando(), eventosFalham{}), &caixaTeste{
		cartas:   []mail.Carta{{UID: "2", De: email, Assunto: fmt.Sprintf("#%d", longo.Numero), Corpo: "marca", MessageID: id()}},
		marcaErr: errFalha,
	}, true)
	correrCorreio(t, controle("SolicitacaoPorNumero").servico(c), &caixaTeste{
		cartas:     []mail.Carta{{UID: "3", De: email, Assunto: fmt.Sprintf("#%d", longo.Numero), Corpo: "falha", MessageID: id()}},
		avisaAoLer: true,
	}, true)

	ctxCancel, cancel := context.WithCancel(ctx)
	cancel()
	if err := c.com(falhando(), eventosFalham{}).WithCaixa(&caixaTeste{lerErr: errFalha}).RunCorreio(ctxCancel); err != nil {
		t.Fatal(err)
	}
}

// caixaTeste entrega as cartas uma vez. pronta avisa o teste para cancelar o laço.
type caixaTeste struct {
	cartas     []mail.Carta
	lerErr     error
	marcaErr   error
	avisaAoLer bool
	pronta     chan struct{}
	lidas      []string
}

func (c *caixaTeste) avisar() {
	if c.pronta == nil {
		return
	}
	select {
	case c.pronta <- struct{}{}:
	default:
	}
}

func (c *caixaTeste) NaoLidas(context.Context) ([]mail.Carta, error) {
	if c.lerErr != nil || c.avisaAoLer {
		c.avisar()
	}
	if c.lerErr != nil {
		return nil, c.lerErr
	}
	out := c.cartas
	c.cartas = nil
	if len(out) == 0 {
		c.avisar()
	}
	return out, nil
}

func (c *caixaTeste) MarcarLida(_ context.Context, uid string) error {
	if c.marcaErr != nil {
		c.avisar()
		return c.marcaErr
	}
	c.lidas = append(c.lidas, uid)
	c.avisar()
	return nil
}

// correrCorreio lê a caixa uma vez e encerra. depoisDeLer espera o aviso
// sair antes de cancelar, senão o cancelamento come o erro da leitura.
func correrCorreio(t *testing.T, svc *application.Service, caixa *caixaTeste, depoisDeLer bool) {
	t.Helper()
	caixa.pronta = make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-caixa.pronta
		if depoisDeLer {
			time.Sleep(100 * time.Millisecond)
		}
		cancel()
	}()
	if err := svc.WithCaixa(caixa).WithLogger(slog.New(slog.DiscardHandler)).RunCorreio(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRamosDaListaEDoCatalogo(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	pg := pagination.New(1, 20, 100)
	gestor := admin()
	gestor.UserID = c.servidor.UserID
	sol := c.abrir(t)

	lista, _, err := c.base.ListarFila(ctx, c.tecnico, application.FiltroFila{Responsavel: application.ResponsavelNenhum, Unidade: c.unidade.String()}, pg)
	if err != nil || !temSolicitacao(lista, sol.ID) {
		t.Fatalf("sem dono: %+v %v", lista, err)
	}
	if _, _, err := c.base.ListarFila(ctx, c.tecnico, application.FiltroFila{Responsavel: "ninguem"}, pg); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("responsável: %v", err)
	}
	if _, _, err := c.base.ListarMinhas(ctx, c.servidor, application.FiltroFila{Prioridade: "urgente"}, pg); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("minhas: %v", err)
	}

	if _, err := c.base.Atribuir(ctx, c.tecnico, sol.ID, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	lista, _, err = c.base.ListarFila(ctx, c.tecnico, application.FiltroFila{Responsavel: application.ResponsavelNenhum, Unidade: c.unidade.String()}, pg)
	if err != nil || temSolicitacao(lista, sol.ID) {
		t.Fatalf("já tem dono: %+v %v", lista, err)
	}

	titulo := "MeuPedido" + uuid.NewString()[:6]
	propria, err := c.base.Abrir(ctx, c.tecnico, application.AberturaEntrada{OfertaID: c.oferta.ID, Titulo: titulo})
	if err != nil {
		t.Fatal(err)
	}
	achadas, err := c.base.Search(ctx, c.tecnico, titulo, 5)
	if err != nil || len(achadas) != 1 || achadas[0].ID != propria.ID {
		t.Fatalf("busca duplicada: %+v %v", achadas, err)
	}
	uma, err := c.base.Search(ctx, c.servidor, "Pedido", 1)
	if err != nil || len(uma) != 1 {
		t.Fatalf("limite: %+v %v", uma, err)
	}
	if longa, err := c.base.Search(ctx, c.servidor, strings.Repeat("á", 81), 0); err != nil || len(longa) != 0 {
		t.Fatalf("consulta longa: %+v %v", longa, err)
	}
	if larga, err := c.base.Search(ctx, c.servidor, "Pedido", 50); err != nil || len(larga) == 0 {
		t.Fatalf("limite alto: %+v %v", larga, err)
	}

	if _, err := c.pool.Exec(ctx, `UPDATE unidades SET sigla = 'TI' WHERE id = $1`, c.unidade); err != nil {
		t.Fatal(err)
	}
	outra, err := c.base.CriarOferta(ctx, gestor, application.OfertaEntrada{Nome: "Segunda " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, outra.ID)
	})
	catalogo, err := c.base.Catalogo(ctx)
	if err != nil || siglaDe(catalogo, c.oferta.ID) != "TI" || siglaDe(catalogo, outra.ID) != "TI" {
		t.Fatalf("sigla: %q %q %v", siglaDe(catalogo, c.oferta.ID), siglaDe(catalogo, outra.ID), err)
	}
	semSec, err := controle("SecretariasDe").servico(c).Catalogo(ctx)
	if err != nil || siglaDe(semSec, c.oferta.ID) != "" {
		t.Fatalf("secretaria indisponível: %q %v", siglaDe(semSec, c.oferta.ID), err)
	}
	w := controle()
	w.secsVazias = true
	semMapa, err := w.servico(c).Catalogo(ctx)
	if err != nil || siglaDe(semMapa, c.oferta.ID) != "" {
		t.Fatalf("sem mapa: %q %v", siglaDe(semMapa, c.oferta.ID), err)
	}
	w = controle()
	w.semUnidade = true
	soltas, err := w.servico(c).Catalogo(ctx)
	if err != nil || len(soltas) != 1 || soltas[0].SecretariaSigla != "" {
		t.Fatalf("sem unidade: %+v %v", soltas, err)
	}

	cat, err := c.base.CriarCategoria(ctx, gestor, application.CategoriaEntrada{Nome: "Rede " + uuid.NewString()[:6]})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_categorias WHERE id = $1`, cat.ID)
	})
	comCat, err := c.base.CriarOferta(ctx, gestor, application.OfertaEntrada{
		Nome: "Link " + uuid.NewString()[:6], Tipo: domain.TipoIncidente, UnidadeID: &c.unidade, CategoriaID: &cat.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE oferta_id = $1`, comCat.ID)
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, comCat.ID)
	})
	aberta, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: comCat.ID, Titulo: "Link caiu"})
	if err != nil || aberta.MetaRespostaMin != nil {
		t.Fatalf("categoria sem SLA: %+v %v", aberta, err)
	}
	sla, err := c.base.CriarSLA(ctx, gestor, application.SLAEntrada{
		Nome: "Prazo " + uuid.NewString()[:6], TempoRespostaMinutos: 15, TempoSolucaoMinutos: 60,
		CalendarioTipo: domain.CalendarioExpediente, UnidadeID: &c.unidade,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.pool.Exec(context.Background(), `DELETE FROM atendimento_slas WHERE id = $1`, sla.ID)
	})
	if err := c.base.AssociarCategoriaSLA(ctx, gestor, cat.ID, &sla.ID); err != nil {
		t.Fatal(err)
	}
	comPrazo, err := c.base.Abrir(ctx, c.servidor, application.AberturaEntrada{OfertaID: comCat.ID, Titulo: "Link caiu de novo"})
	if err != nil || comPrazo.MetaRespostaMin == nil || *comPrazo.MetaRespostaMin != 15 || comPrazo.MetaSolucaoMin == nil || *comPrazo.MetaSolucaoMin != 60 || comPrazo.Prioridade == nil || *comPrazo.Prioridade != domain.PrioridadeMedia {
		t.Fatalf("SLA: %+v %v", comPrazo, err)
	}
}

func temSolicitacao(lista []domain.Solicitacao, id uuid.UUID) bool {
	for _, s := range lista {
		if s.ID == id {
			return true
		}
	}
	return false
}

func siglaDe(lista []domain.Oferta, id uuid.UUID) string {
	for _, o := range lista {
		if o.ID == id {
			return o.SecretariaSigla
		}
	}
	return ""
}
