package application

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

type sinoComContato struct{ sinoFalso }

func (sinoComContato) AvisarContato(context.Context, string, string, notificacoes.Nova) {}

func TestRamosQueACatracaAponta(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)

	if _, err := s.ListarResponsaveis(context.Background()); err == nil {
		t.Fatal("sessão")
	}
	sem := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New()})
	if _, err := s.ListarResponsaveis(sem); err == nil {
		t.Fatal("fora da unidade")
	}
	if _, err := s.CadastrarResponsavel(ctx, "Ana Lima", "12345678900", un); err != nil {
		t.Fatal(err)
	}
	lista, err := s.ListarResponsaveis(ctx)
	if err != nil || len(lista) != 1 || !strings.Contains(lista[0].CPF, "*") {
		t.Fatalf("máscara %+v %v", lista, err)
	}
	m.falha = "nulo"
	if lista, err = s.ListarResponsaveis(ctx); err != nil || lista == nil {
		t.Fatalf("lista vazia %+v %v", lista, err)
	}
	m.falha = "listar-resp"
	if _, err := s.ListarResponsaveis(ctx); err == nil {
		t.Fatal("lista")
	}
	m.falha = ""

	form, err := s.SalvarFormulario(ctx, domain.Formulario{
		UnidadeID: un, Titulo: "Alvará",
		Campos: []domain.Campo{
			{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true},
			{ID: "doc", Tipo: "file", Rotulo: "Documento"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Alvará"})
	if err != nil {
		t.Fatal(err)
	}
	ped, err := s.Abrir(ctx, Abertura{
		ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Email: "maria@sema.test",
		Respostas: map[string]any{"nome": "Maria", "doc": "protocolos/carta/ja-no-pedido"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, "12345678900", "a.pdf", "application/pdf"); err == nil {
		t.Fatal("sem armazenamento")
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, "12345678900", "segue o documento", nil); !errors.Is(err, domain.ErrInvalido) {
		t.Fatalf("ainda não está em complementação: %v", err)
	}
	loja := storagetest.New()
	s.ComArquivos(Arquivos{Store: loja, Bucket: "nova"})
	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, "12345678900", "a.pdf", "application/pdf"); !errors.Is(err, domain.ErrInvalido) {
		t.Fatalf("status: %v", err)
	}
	if _, err := s.Complementar(ctx, ped.ID, "envie o documento"); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.PedirArquivoComplemento(ctx, ped.Numero, ped.CPF, "a.pdf", "application/pdf")
	if err != nil || ticket.ObjectKey == "" {
		t.Fatal(err)
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "   ", nil); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("mensagem")
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"", "", "", "", "", "x"}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("muitos")
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"fora/do/prefixo"}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("prefixo")
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{prefixoComplemento(ped.ID) + "/ausente.pdf"}); err == nil {
		t.Fatal("arquivo ausente")
	}
	if err := loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	m.falha = "unidade"
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"", ticket.ObjectKey}); err == nil {
		t.Fatal("unidade")
	}
	m.falha = "atualizar-pedido"
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"", ticket.ObjectKey}); err == nil {
		t.Fatal("atualizar")
	}
	m.falha = "hist"
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"", ticket.ObjectKey}); err == nil {
		t.Fatal("histórico da resposta")
	}
	if guardado := m.pedidos[ped.ID]; guardado.Status != domain.StatusComplementacao {
		guardado.Status = domain.StatusComplementacao
		m.pedidos[ped.ID] = guardado
	}
	m.falha = "buscar-numero"
	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, ped.CPF, "doc.pdf", "application/pdf"); err == nil {
		t.Fatal("número")
	}
	m.falha = ""
	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, "00000000000", "doc.pdf", "application/pdf"); err == nil {
		t.Fatal("cpf")
	}
	s.sino = sinoComContato{}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue o documento", []string{"", ticket.ObjectKey}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: " ", Respostas: map[string]any{"nome": "Maria"}}); err == nil {
		t.Fatal("justificativa")
	}
	if _, err := s.LinkArquivo(ctx, ped.ID, prefixoComplemento(ped.ID)+"/nao-esta.pdf"); err == nil {
		t.Fatal("anexo fora do histórico")
	}
	chave := prefixoComplemento(ped.ID) + "/doc.pdf"
	m.hist = append(m.hist, domain.Historico{PedidoID: ped.ID, Anexos: []string{chave}})
	if err := loja.Put(ctx, "nova", chave, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	if url, err := s.LinkArquivo(ctx, ped.ID, chave); err != nil || url == "" {
		t.Fatal(err)
	}

	if got := preservarArquivos(domain.Pedido{
		Campos:    []domain.Campo{{ID: "doc", Tipo: "file"}, {ID: "nome", Tipo: "text"}},
		Respostas: map[string]any{"doc": "protocolos/carta/ja-no-pedido"},
	}, nil); got["doc"] != "protocolos/carta/ja-no-pedido" || got["nome"] != nil {
		t.Fatalf("arquivos %+v", got)
	}
	if _, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "o nome veio abreviado", Respostas: map[string]any{"nome": "Maria da Silva"}}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Anotar(sem, ped.ID, "nota da mesa"); err == nil {
		t.Fatal("tratar")
	}
	m.falha = "buscar-pedido"
	if _, err := s.Anotar(ctx, ped.ID, "nota da mesa"); err == nil {
		t.Fatal("pedido")
	}
	m.falha = ""
	guardado := m.pedidos[ped.ID]
	guardado.Status = domain.StatusConcluido
	m.pedidos[ped.ID] = guardado
	if _, err := s.Anotar(ctx, ped.ID, "nota da mesa"); !errors.Is(err, domain.ErrEstadoFinal) {
		t.Fatalf("encerrado: %v", err)
	}
	guardado.Status = domain.StatusAndamento
	m.pedidos[ped.ID] = guardado
	m.falha = "hist"
	if _, err := s.Anotar(ctx, ped.ID, "nota da mesa"); err == nil {
		t.Fatal("histórico")
	}
	m.falha = ""

	if err := s.RegistrarFalta(sem, ped.ID); err == nil {
		t.Fatal("permissão da falta")
	}
	if err := s.RegistrarFalta(ctx, ped.ID); !errors.Is(err, domain.ErrInvalido) {
		t.Fatalf("sem horário: %v", err)
	}
	m.agenda = []domain.Agendamento{{ID: uuid.New(), PedidoID: ped.ID, ServicoID: serv.ID, Status: "realizado"}}
	if err := s.RegistrarFalta(ctx, ped.ID); !errors.Is(err, domain.ErrInvalido) {
		t.Fatalf("status: %v", err)
	}
	m.falha = "buscar-agenda"
	if err := s.RegistrarFalta(ctx, ped.ID); err == nil || errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	m.falha = ""
	m.agenda[0].Status = domain.HorarioConfirmado
	m.falha = "atualizar-agenda"
	if err := s.RegistrarFalta(ctx, ped.ID); err == nil {
		t.Fatal("atualizar")
	}
}

func TestAvisoERespostaQueFalham(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{
		UnidadeID: un, Titulo: "Alvará",
		Campos: []domain.Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Alvará"})
	if err != nil {
		t.Fatal(err)
	}
	aviso := textoDeAviso
	textoDeAviso = func(string, string, string) (string, error) { return "", domain.ErrInvalido }
	t.Cleanup(func() { textoDeAviso = aviso })
	if _, err := s.Abrir(ctx, Abertura{
		ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{"nome": "Maria"},
	}, false); err == nil {
		t.Fatal("aviso da abertura")
	}
	textoDeAviso = aviso

	ped, err := s.Abrir(ctx, Abertura{
		ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{"nome": "Maria"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	original := m.pedidos[ped.ID]
	semNumero := original
	semNumero.Numero = "   "
	m.pedidos[ped.ID] = semNumero
	if _, err := s.Cancelar(ctx, ped.ID); err == nil {
		t.Fatal("aviso do cancelamento")
	}
	m.pedidos[ped.ID] = original
	if _, err := s.Complementar(ctx, ped.ID, "falta o documento"); err != nil {
		t.Fatal(err)
	}
	comp := m.pedidos[ped.ID]
	comp.Numero = "   "
	m.pedidos[ped.ID] = comp
	if _, err := s.ResponderPublico(ctx, "   ", "12345678900", "segue", nil); err == nil {
		t.Fatal("aviso da resposta")
	}
	m.pedidos[ped.ID] = comp
	resposta := responderComplemento
	responderComplemento = func(domain.Pedido, time.Time) (domain.Pedido, error) {
		return domain.Pedido{}, domain.ErrEstadoFinal
	}
	t.Cleanup(func() { responderComplemento = resposta })
	if _, err := s.ResponderPublico(ctx, "   ", "12345678900", "segue", nil); !errors.Is(err, domain.ErrEstadoFinal) {
		t.Fatalf("resposta: %v", err)
	}
}

func TestSorteioSemEntropia(t *testing.T) {
	orig := rand.Reader
	rand.Reader = falhaLeitor{}
	t.Cleanup(func() { rand.Reader = orig })
	if _, err := inteiroAleatorio(); err == nil {
		t.Fatal("entropia")
	}
}

type falhaLeitor struct{}

func (falhaLeitor) Read([]byte) (int, error) { return 0, errors.New("sem entropia") }
