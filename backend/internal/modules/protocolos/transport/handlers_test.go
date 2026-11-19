package transport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

type falso struct{ erro error }

func (f falso) SalvarFormulario(context.Context, domain.Formulario) (domain.Formulario, error) {
	return domain.Formulario{Titulo: "Alvará"}, f.erro
}
func (f falso) ListarFormularios(context.Context) ([]domain.Formulario, error) {
	return []domain.Formulario{{Titulo: "Alvará"}}, f.erro
}
func (f falso) SalvarServico(context.Context, domain.Servico) (domain.Servico, error) {
	return domain.Servico{Nome: "Alvará"}, f.erro
}
func (f falso) ListarServicosGestao(context.Context) ([]domain.Servico, error) {
	return []domain.Servico{{Nome: "Alvará"}}, f.erro
}
func (f falso) Carta(context.Context, *uuid.UUID) ([]domain.Servico, error) {
	return []domain.Servico{{Nome: "Alvará"}}, f.erro
}
func (f falso) ServicoPublico(context.Context, uuid.UUID) (domain.Servico, domain.Formulario, error) {
	return domain.Servico{Nome: "Alvará"}, domain.Formulario{Titulo: "Alvará"}, f.erro
}
func (f falso) PedirArquivo(context.Context, uuid.UUID, string, string) (modkit.UploadTicket, error) {
	return modkit.UploadTicket{ObjectKey: "protocolos/carta/a.pdf", Method: "PUT"}, f.erro
}
func (f falso) PedirArquivoComplemento(context.Context, string, string, string, string) (modkit.UploadTicket, error) {
	return modkit.UploadTicket{ObjectKey: "protocolos/complemento/a.pdf", Method: "PUT"}, f.erro
}
func (f falso) LinkArquivo(context.Context, uuid.UUID, string) (string, error) {
	return "https://minio.test/a.pdf", f.erro
}
func (f falso) CadastrarResponsavel(context.Context, string, string, uuid.UUID) (domain.ResponsavelTecnico, error) {
	return domain.ResponsavelTecnico{Nome: "Ana", CPF: "12345678900"}, f.erro
}
func (f falso) ListarResponsaveis(context.Context) ([]domain.ResponsavelTecnico, error) {
	return []domain.ResponsavelTecnico{{Nome: "Ana"}}, f.erro
}
func (f falso) RegistrarFalta(context.Context, uuid.UUID) error { return f.erro }
func (f falso) Abrir(context.Context, application.Abertura, bool) (domain.Pedido, error) {
	return domain.Pedido{Numero: "SEMA202600001"}, f.erro
}
func (f falso) Consultar(context.Context, string) (domain.Consulta, error) {
	return domain.Consulta{Numero: "SEMA202600001"}, f.erro
}
func (f falso) ResponderPublico(context.Context, string, string, string, []string) (domain.Consulta, error) {
	return domain.Consulta{Numero: "SEMA202600001", Status: domain.StatusAndamento}, f.erro
}
func (f falso) ListarPedidos(context.Context) ([]domain.Pedido, error) {
	return []domain.Pedido{{Numero: "SEMA202600001"}}, f.erro
}
func (f falso) BuscarPedido(context.Context, uuid.UUID) (application.Ficha, error) {
	return application.Ficha{Pedido: domain.Pedido{Numero: "SEMA202600001"}}, f.erro
}
func (f falso) Complementar(context.Context, uuid.UUID, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusComplementacao}, f.erro
}
func (f falso) Responder(context.Context, uuid.UUID, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Anotar(context.Context, uuid.UUID, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Editar(context.Context, uuid.UUID, string, map[string]any) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Avancar(context.Context, uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{EtapaID: "b"}, f.erro
}
func (f falso) Concluir(context.Context, uuid.UUID, string, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusConcluido}, f.erro
}
func (f falso) Cancelar(context.Context, uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusCancelado}, f.erro
}

func pedido(method, alvo, body, id string) *http.Request {
	req := httptest.NewRequest(method, alvo, strings.NewReader(body))
	rctx := chi.NewRouteContext()
	if id != "" {
		rctx.URLParams.Add("id", id)
		rctx.URLParams.Add("numero", id)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestRotas(t *testing.T) {
	h := NewHandlers(falso{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	id := uuid.NewString()
	un := uuid.NewString()
	ok := []struct {
		fn     http.HandlerFunc
		method string
		body   string
		param  string
	}{
		{h.SalvarFormulario, http.MethodPost, `{"unidade_id":"` + un + `","titulo":"Alvará"}`, ""},
		{h.ListarFormularios, http.MethodGet, "", ""},
		{h.SalvarServico, http.MethodPost, `{"unidade_id":"` + un + `","nome":"Alvará"}`, ""},
		{h.ListarServicos, http.MethodGet, "", ""},
		{h.Responsavel, http.MethodPost, `{"nome":"Ana Lima","cpf":"12345678900","unidade_id":"` + un + `"}`, ""},
		{h.ListarResponsaveis, http.MethodGet, "", ""},
		{h.Falta, http.MethodPost, `{"pedido_id":"` + id + `"}`, ""},
		{h.AbrirPublico, http.MethodPost, `{"nome":"Maria","cpf":"12345678900"}`, id},
		{h.AbrirBalcao, http.MethodPost, `{"servico_id":"` + id + `","nome":"Maria","cpf":"12345678900"}`, ""},
		{h.Carta, http.MethodGet, "", ""},
		{h.CartaItem, http.MethodGet, "", id},
		{h.PedirArquivo, http.MethodPost, `{"nome":"doc.pdf","content_type":"application/pdf"}`, id},
		{h.LinkArquivo, http.MethodGet, "", id},
		{h.Consultar, http.MethodGet, "", "SEMA202600001"},
		{h.PedirArquivoComplemento, http.MethodPost, `{"cpf":"12345678900","nome":"doc.pdf","content_type":"application/pdf"}`, "SEMA202600001"},
		{h.ResponderPublico, http.MethodPost, `{"cpf":"12345678900","mensagem":"segue"}`, "SEMA202600001"},
		{h.ListarPedidos, http.MethodGet, "", ""},
		{h.BuscarPedido, http.MethodGet, "", id},
		{h.Complementar, http.MethodPost, `{"mensagem":"envie"}`, id},
		{h.Responder, http.MethodPost, `{"mensagem":"recebido no balcão"}`, id},
		{h.Anotar, http.MethodPost, `{"mensagem":"ligou e confirmou"}`, id},
		{h.Editar, http.MethodPost, `{"justificativa":"endereço corrigido","respostas":{}}`, id},
		{h.Avancar, http.MethodPost, "", id},
		{h.Cancelar, http.MethodPost, "", id},
		{h.Concluir, http.MethodPost, `{"decisao":"favoravel","justificativa":"ok"}`, id},
	}
	for _, c := range ok {
		rec := httptest.NewRecorder()
		c.fn(rec, pedido(c.method, "/", c.body, c.param))
		if rec.Code >= 400 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	h.Carta(rec, httptest.NewRequest(http.MethodGet, "/?unidade="+un, nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Carta(rec, httptest.NewRequest(http.MethodGet, "/?unidade=nao", nil))
	if rec.Code < 400 {
		t.Fatal("unidade")
	}
	rec = httptest.NewRecorder()
	h.CartaItem(rec, pedido(http.MethodGet, "/", "", "nao"))
	if rec.Code < 400 {
		t.Fatal("id")
	}
	rec = httptest.NewRecorder()
	h.SalvarFormulario(rec, pedido(http.MethodPost, "/", "{", ""))
	if rec.Code < 400 {
		t.Fatal("json")
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ruim := NewHandlers(falso{erro: domain.ErrNaoEncontrado}, log)
	falhas := []struct {
		fn     http.HandlerFunc
		method string
		body   string
		param  string
	}{
		{ruim.SalvarFormulario, http.MethodPost, `{"unidade_id":"` + un + `","titulo":"Alvará"}`, ""},
		{ruim.ListarFormularios, http.MethodGet, "", ""},
		{ruim.SalvarServico, http.MethodPost, `{"unidade_id":"` + un + `","nome":"Alvará"}`, ""},
		{ruim.ListarServicos, http.MethodGet, "", ""},
		{ruim.Responsavel, http.MethodPost, `{"nome":"Ana Lima","cpf":"12345678900","unidade_id":"` + un + `"}`, ""},
		{ruim.ListarResponsaveis, http.MethodGet, "", ""},
		{ruim.Falta, http.MethodPost, `{"pedido_id":"` + id + `"}`, ""},
		{ruim.AbrirPublico, http.MethodPost, `{"nome":"Maria","cpf":"12345678900"}`, id},
		{ruim.AbrirBalcao, http.MethodPost, `{"servico_id":"` + id + `","nome":"Maria","cpf":"12345678900"}`, ""},
		{ruim.Carta, http.MethodGet, "", ""},
		{ruim.CartaItem, http.MethodGet, "", id},
		{ruim.PedirArquivo, http.MethodPost, `{"nome":"doc.pdf","content_type":"application/pdf"}`, id},
		{ruim.LinkArquivo, http.MethodGet, "", id},
		{ruim.Consultar, http.MethodGet, "", "SEMA202600001"},
		{ruim.PedirArquivoComplemento, http.MethodPost, `{"cpf":"12345678900","nome":"doc.pdf","content_type":"application/pdf"}`, "SEMA202600001"},
		{ruim.ResponderPublico, http.MethodPost, `{"cpf":"12345678900","mensagem":"segue"}`, "SEMA202600001"},
		{ruim.ListarPedidos, http.MethodGet, "", ""},
		{ruim.BuscarPedido, http.MethodGet, "", id},
		{ruim.Complementar, http.MethodPost, `{"mensagem":"envie"}`, id},
		{ruim.Responder, http.MethodPost, `{"mensagem":"recebido no balcão"}`, id},
		{ruim.Anotar, http.MethodPost, `{"mensagem":"ligou e confirmou"}`, id},
		{ruim.Editar, http.MethodPost, `{"justificativa":"endereço corrigido","respostas":{}}`, id},
		{ruim.Avancar, http.MethodPost, "", id},
		{ruim.Cancelar, http.MethodPost, "", id},
		{ruim.Concluir, http.MethodPost, `{"decisao":"favoravel","justificativa":"ok"}`, id},
	}
	for _, c := range falhas {
		rec = httptest.NewRecorder()
		c.fn(rec, pedido(c.method, "/", c.body, c.param))
		if rec.Code < 400 {
			t.Fatalf("esperava erro em %s", c.method)
		}
	}
	for _, c := range []struct {
		fn    http.HandlerFunc
		body  string
		param string
	}{
		{h.SalvarServico, "{", ""},
		{h.Responsavel, "{", ""},
		{h.Falta, "{", ""},
		{h.AbrirPublico, "{", ""},
		{h.AbrirBalcao, "{", ""},
		{h.Complementar, "{", id},
		{h.Concluir, "{", id},
		{h.PedirArquivo, "{", id},
		{h.PedirArquivoComplemento, "{", "SEMA202600001"},
		{h.ResponderPublico, "{", "SEMA202600001"},
		{h.PedirArquivo, `{"nome":"doc.pdf"}`, "nao"},
		{h.LinkArquivo, "", "nao"},
		{h.BuscarPedido, "", "nao"},
		{h.Complementar, `{"mensagem":"x"}`, "nao"},
		{h.Responder, "{", id},
		{h.Responder, "", "nao"},
		{h.Anotar, "{", id},
		{h.Editar, "{", id},
		{h.Avancar, "", "nao"},
		{h.Cancelar, "", "nao"},
		{h.Concluir, `{"decisao":"favoravel","justificativa":"ok"}`, "nao"},
	} {
		rec = httptest.NewRecorder()
		c.fn(rec, pedido(http.MethodPost, "/", c.body, c.param))
		if rec.Code < 400 {
			t.Fatal("entrada")
		}
	}
	r := chi.NewRouter()
	RegisterPublic(r, h)
	RegisterAuthed(r, h, nil)
	rec = httptest.NewRecorder()
	abrir := httptest.NewRequest(http.MethodPost, "/protocolos/carta/"+id+"/pedido", strings.NewReader(`{"nome":"Maria","cpf":"12345678900"}`))
	abrir.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, abrir)
	if rec.Code != http.StatusCreated {
		t.Fatalf("abertura pública %d %s", rec.Code, rec.Body.String())
	}
}
