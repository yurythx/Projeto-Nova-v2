package application

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

func TestArquivos(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	if s.ComArquivos(Arquivos{}).arquivos.MaxBytes == 0 {
		t.Fatal("limite")
	}
	campos := []domain.Campo{
		{ID: "obs", Tipo: "text", Rotulo: "Observação"},
		{ID: "doc", Tipo: "file", Rotulo: "Documento", Obrigatorio: true},
	}
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok", Campos: campos})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("sem armazenamento")
	}
	if _, err := s.LinkArquivo(ctx, uuid.New(), "doc"); err == nil {
		t.Fatal("sem armazenamento")
	}
	loja := storagetest.New()
	s.ComArquivos(Arquivos{Store: loja, Bucket: "nova"})
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/x-msdownload"); err == nil {
		t.Fatal("tipo")
	}
	ticket, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf")
	if err != nil || ticket.ObjectKey == "" {
		t.Fatal(err)
	}
	if err := loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	s.sorteio = func() int { return 4 }
	ped, err := s.Abrir(ctx, Abertura{
		ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900",
		Respostas: map[string]any{"doc": ticket.ObjectKey},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if url, err := s.LinkArquivo(ctx, ped.ID, "doc"); err != nil || url == "" {
		t.Fatal(err)
	}
	if _, err := s.LinkArquivo(ctx, uuid.New(), "doc"); err == nil {
		t.Fatal("pedido")
	}
	for _, respostas := range []map[string]any{nil, {"doc": "  "}, {"doc": 1}, {"doc": "fora"}} {
		if _, err := s.Abrir(ctx, Abertura{
			ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: respostas,
		}, false); err == nil {
			t.Fatal("arquivo")
		}
	}
	if _, err := s.LinkArquivo(ctx, ped.ID, "outro"); err == nil {
		t.Fatal("campo")
	}
	loja.FailPresign = true
	if _, err := s.LinkArquivo(ctx, ped.ID, "doc"); err == nil {
		t.Fatal("leitura")
	}
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("ticket")
	}
	loja.FailPresign = false
	guardado := m.forms[form.ID]
	guardado.Campos = []domain.Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome"}}
	m.forms[form.ID] = guardado
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("sem arquivo")
	}
	m.servs[serv.ID] = domain.Servico{ID: serv.ID, UnidadeID: un, FormularioID: form.ID, Nome: "Ok", Ativo: false}
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("inativo")
	}
	m.falha = "buscar-serv"
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("serviço")
	}
	m.falha = "buscar-form"
	m.servs[serv.ID] = domain.Servico{ID: serv.ID, UnidadeID: un, FormularioID: form.ID, Nome: "Ok", Ativo: true}
	if _, err := s.PedirArquivo(ctx, serv.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("formulário")
	}
}
