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
func (f falso) ComplementarCom(context.Context, uuid.UUID, application.Complementacao) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusComplementacao}, f.erro
}
func (f falso) Responder(context.Context, uuid.UUID, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) AnotarCom(context.Context, uuid.UUID, string, []string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Editar(context.Context, uuid.UUID, application.Edicao) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) AvancarCom(context.Context, uuid.UUID, []string) (domain.Pedido, error) {
	return domain.Pedido{EtapaID: "b"}, f.erro
}
func (f falso) ConcluirCom(context.Context, uuid.UUID, application.Conclusao) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusConcluido}, f.erro
}
func (f falso) Cancelar(context.Context, uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusCancelado}, f.erro
}
func (f falso) Pessoas(context.Context, uuid.UUID) ([]domain.Pessoa, error) {
	return []domain.Pessoa{{Nome: "Ana"}}, f.erro
}
func (f falso) Aceitar(context.Context, uuid.UUID, *uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Transferir(context.Context, uuid.UUID, uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Encaminhar(context.Context, uuid.UUID, uuid.UUID, string, int) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) DecidirEncaminhamento(context.Context, uuid.UUID, bool, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) Aprovar(context.Context, uuid.UUID) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusConcluido}, f.erro
}
func (f falso) Devolver(context.Context, uuid.UUID, string) (domain.Pedido, error) {
	return domain.Pedido{Status: domain.StatusAndamento}, f.erro
}
func (f falso) PedirAnexoMesa(context.Context, uuid.UUID, string, string) (modkit.UploadTicket, error) {
	return modkit.UploadTicket{ObjectKey: "protocolos/mesa/a.pdf", Method: "PUT"}, f.erro
}
func (f falso) ListarFeriados(context.Context) ([]domain.Feriado, error) {
	return []domain.Feriado{{Data: "2026-12-25", Nome: "Natal"}}, f.erro
}
func (f falso) SalvarFeriado(context.Context, string, string) (domain.Feriado, error) {
	return domain.Feriado{Data: "2026-12-25", Nome: "Natal"}, f.erro
}
func (f falso) RemoverFeriado(context.Context, string) error { return f.erro }
func (f falso) HorariosLivres(context.Context, uuid.UUID, string) ([]string, error) {
	return []string{"08:00"}, f.erro
}
func (f falso) PessoasDaUnidade(context.Context, uuid.UUID) ([]domain.Pessoa, error) {
	return []domain.Pessoa{{Nome: "Ana"}}, f.erro
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
		{h.Pessoas, http.MethodGet, "", id},
		{h.Aceitar, http.MethodPost, `{}`, id},
		{h.Aceitar, http.MethodPost, `{"responsavel_id":"` + un + `"}`, id},
		{h.Transferir, http.MethodPost, `{"responsavel_id":"` + un + `"}`, id},
		{h.Encaminhar, http.MethodPost, `{"unidade_id":"` + un + `","justificativa":"é da saúde","prazo_dias":5}`, id},
		{h.AceitarEncaminhamento, http.MethodPost, "", id},
		{h.RecusarEncaminhamento, http.MethodPost, `{"mensagem":"não é nosso"}`, id},
		{h.Aprovar, http.MethodPost, "", id},
		{h.Devolver, http.MethodPost, `{"mensagem":"refazer o parecer"}`, id},
		{h.Avancar, http.MethodPost, `{"anexos":["protocolos/mesa/a.pdf"]}`, id},
		{h.Complementar, http.MethodPost, `{"mensagem":"envie","prazo_dias":5,"anexos":[]}`, id},
		{h.Concluir, http.MethodPost, `{"decisao":"favoravel","justificativa":"ok","processo_sei":"12345.000001/2026-11"}`, id},
		{h.Editar, http.MethodPost, `{"justificativa":"vínculo","processo_sei":""}`, id},
		{h.PedirAnexoMesa, http.MethodPost, `{"nome":"parecer.pdf","content_type":"application/pdf"}`, id},
		{h.ListarFeriados, http.MethodGet, "", ""},
		{h.SalvarFeriado, http.MethodPost, `{"data":"2026-12-25","nome":"Natal"}`, ""},
		{h.RemoverFeriado, http.MethodDelete, "", ""},
		{h.HorariosLivres, http.MethodGet, "", id},
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
		{ruim.Pessoas, http.MethodGet, "", id},
		{ruim.Aceitar, http.MethodPost, `{}`, id},
		{ruim.Transferir, http.MethodPost, `{"responsavel_id":"` + un + `"}`, id},
		{ruim.Encaminhar, http.MethodPost, `{"unidade_id":"` + un + `","justificativa":"x"}`, id},
		{ruim.AceitarEncaminhamento, http.MethodPost, "", id},
		{ruim.RecusarEncaminhamento, http.MethodPost, `{"mensagem":"x"}`, id},
		{ruim.Aprovar, http.MethodPost, "", id},
		{ruim.Devolver, http.MethodPost, `{"mensagem":"x"}`, id},
		{ruim.PedirAnexoMesa, http.MethodPost, `{"nome":"a.pdf","content_type":"application/pdf"}`, id},
		{ruim.ListarFeriados, http.MethodGet, "", ""},
		{ruim.SalvarFeriado, http.MethodPost, `{"data":"2026-12-25","nome":"Natal"}`, ""},
		{ruim.RemoverFeriado, http.MethodDelete, "", ""},
		{ruim.HorariosLivres, http.MethodGet, "", id},
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
		{h.Falta, `{}`, ""},
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
		{h.AbrirPublico, `{"nome":"Maria","cpf":"12345678900"}`, "nao"},
		{h.Anotar, `{"mensagem":"nota"}`, "nao"},
		{h.Editar, `{"justificativa":"corrigir","respostas":{}}`, "nao"},
		{h.Avancar, "", "nao"},
		{h.Cancelar, "", "nao"},
		{h.Concluir, `{"decisao":"favoravel","justificativa":"ok"}`, "nao"},
		{h.Pessoas, "", "nao"},
		{h.Aceitar, `{}`, "nao"},
		{h.Aceitar, "{", id},
		{h.Transferir, `{}`, id},
		{h.Transferir, "{", id},
		{h.Encaminhar, "{", id},
		{h.AceitarEncaminhamento, "", "nao"},
		{h.RecusarEncaminhamento, "{", id},
		{h.Aprovar, "", "nao"},
		{h.Devolver, `{"mensagem":"x"}`, "nao"},
		{h.Avancar, "{", id},
		{h.PedirAnexoMesa, "{", id},
		{h.SalvarFeriado, "{", ""},
		{h.HorariosLivres, "", "nao"},
	} {
		rec = httptest.NewRecorder()
		c.fn(rec, pedido(http.MethodPost, "/", c.body, c.param))
		if rec.Code < 400 {
			t.Fatal("entrada")
		}
	}
	rec = httptest.NewRecorder()
	h.PessoasDaUnidade(rec, httptest.NewRequest(http.MethodGet, "/?unidade_id="+un, nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	for _, alvo := range []*Handlers{h, ruim} {
		rec = httptest.NewRecorder()
		alvo.PessoasDaUnidade(rec, httptest.NewRequest(http.MethodGet, "/?unidade_id=nao", nil))
		if rec.Code < 400 {
			t.Fatal("unidade")
		}
	}
	rec = httptest.NewRecorder()
	ruim.PessoasDaUnidade(rec, httptest.NewRequest(http.MethodGet, "/?unidade_id="+un, nil))
	if rec.Code < 400 {
		t.Fatal("pessoas")
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
