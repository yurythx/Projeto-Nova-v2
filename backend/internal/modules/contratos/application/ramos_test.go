package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

var errFalha = errors.New("falha injetada")

type repoFalho struct {
	domain.Repository
	falha    string
	vencendo []domain.Vencimento
}

func falhando(metodo string) repoFalho {
	return repoFalho{Repository: infrastructure.NewRepository(), falha: metodo}
}

func (r repoFalho) ListFornecedores(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Fornecedor, error) {
	if r.falha == "ListFornecedores" {
		return nil, errFalha
	}
	return r.Repository.ListFornecedores(ctx, db, f)
}

func (r repoFalho) InsertFornecedor(ctx context.Context, db database.DBTX, item domain.Fornecedor) (domain.Fornecedor, error) {
	if r.falha == "InsertFornecedor" {
		return domain.Fornecedor{}, errFalha
	}
	return r.Repository.InsertFornecedor(ctx, db, item)
}

func (r repoFalho) DeleteFornecedor(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	if r.falha == "DeleteFornecedor" {
		return errFalha
	}
	return r.Repository.DeleteFornecedor(ctx, db, id, f)
}

func (r repoFalho) ListContratos(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Contrato, error) {
	if r.falha == "ListContratos" {
		return nil, errFalha
	}
	return r.Repository.ListContratos(ctx, db, f)
}

func (r repoFalho) InsertContrato(ctx context.Context, db database.DBTX, item domain.Contrato) (domain.Contrato, error) {
	if r.falha == "InsertContrato" {
		return domain.Contrato{}, errFalha
	}
	return r.Repository.InsertContrato(ctx, db, item)
}

func (r repoFalho) DeleteContrato(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	if r.falha == "DeleteContrato" {
		return errFalha
	}
	return r.Repository.DeleteContrato(ctx, db, id, f)
}

func (r repoFalho) ListLicencas(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Licenca, error) {
	if r.falha == "ListLicencas" {
		return nil, errFalha
	}
	return r.Repository.ListLicencas(ctx, db, f)
}

func (r repoFalho) InsertLicenca(ctx context.Context, db database.DBTX, item domain.Licenca) (domain.Licenca, error) {
	if r.falha == "InsertLicenca" {
		return domain.Licenca{}, errFalha
	}
	return r.Repository.InsertLicenca(ctx, db, item)
}

func (r repoFalho) DeleteLicenca(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	if r.falha == "DeleteLicenca" {
		return errFalha
	}
	return r.Repository.DeleteLicenca(ctx, db, id, f)
}

func (r repoFalho) Vencendo(ctx context.Context, db database.DBTX) ([]domain.Vencimento, error) {
	if r.falha == "Vencendo" {
		return nil, errFalha
	}
	if r.vencendo != nil {
		return r.vencendo, nil
	}
	return r.Repository.Vencendo(ctx, db)
}

type licencasFixas struct {
	repoFalho
	lista []domain.Licenca
}

func (l licencasFixas) ListLicencas(context.Context, database.DBTX, domain.Filtro) ([]domain.Licenca, error) {
	return l.lista, nil
}

func (r repoFalho) Destinatarios(ctx context.Context, db database.DBTX, u *uuid.UUID) ([]uuid.UUID, error) {
	if r.falha == "Destinatarios" {
		return nil, errFalha
	}
	return r.Repository.Destinatarios(ctx, db, u)
}

type programas struct{ err error }

func (p programas) Contar(context.Context, database.DBTX, bool, []uuid.UUID, string) (application.UsoDePrograma, error) {
	if p.err != nil {
		return application.UsoDePrograma{}, p.err
	}
	return application.UsoDePrograma{Instalados: 2, ComInventario: 4, Parcial: true}, nil
}

func TestRamosDoCadastro(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u, outra := dbtest.Unidade(t, pool), dbtest.Unidade(t, pool)
	svc := application.NewService(pool, infrastructure.NewRepository(), semEventos{})
	quem := admin()
	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	ninguém := auth.Identity{UserID: uuid.New()}

	if lista, err := svc.ListarFornecedores(ctx, ninguém); err != nil || len(lista) != 0 {
		t.Fatalf("sem escopo: %+v %v", lista, err)
	}
	if _, err := svc.CriarFornecedor(ctx, quem, " ", "", "", nil); err == nil {
		t.Fatal("nome")
	}
	if _, err := svc.CriarFornecedor(ctx, gestor(u), "Casa alheia", "", "", &outra); !recusa(err, "não gerencia") {
		t.Fatalf("fora: %v", err)
	}
	if _, err := application.NewService(pool, falhando("InsertFornecedor"), semEventos{}).CriarFornecedor(ctx, quem, "Casa", "", "", nil); err == nil {
		t.Fatal("insert")
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), semEventos{application.EventoFornecedor}).CriarFornecedor(ctx, quem, "Casa", "", "", nil); err == nil {
		t.Fatal("evento")
	}
	casa, err := svc.CriarFornecedor(ctx, gestor(u), "Casa "+uuid.NewString()[:6], "casa@org.gov.br", "11", &u)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_licencas WHERE fornecedor_id = $1`, casa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_contratos WHERE fornecedor_id = $1`, casa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_fornecedores WHERE id = $1`, casa.ID)
	})
	if lista, err := svc.ListarFornecedores(ctx, gestor(u)); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	if _, err := application.NewService(pool, falhando("ListFornecedores"), semEventos{}).ListarFornecedores(ctx, quem); err == nil {
		t.Fatal("consulta")
	}
	if err := svc.ExcluirFornecedor(ctx, ninguém, casa.ID); !recusa(err, "não gerencia") {
		t.Fatalf("excluir sem escopo: %v", err)
	}
	if err := svc.ExcluirFornecedor(ctx, quem, uuid.New()); !recusa(err, "não encontrado") {
		t.Fatalf("excluir ausente: %v", err)
	}
	if err := application.NewService(pool, falhando("DeleteFornecedor"), semEventos{}).ExcluirFornecedor(ctx, quem, casa.ID); err == nil {
		t.Fatal("delete")
	}
	if err := application.NewService(pool, infrastructure.NewRepository(), semEventos{"contratos.fornecedor.excluido"}).ExcluirFornecedor(ctx, quem, casa.ID); err == nil {
		t.Fatal("evento de exclusão")
	}

	if lista, err := svc.ListarContratos(ctx, ninguém); err != nil || len(lista) != 0 {
		t.Fatalf("contratos sem escopo: %+v %v", lista, err)
	}
	if _, err := svc.CriarContrato(ctx, quem, "", " ", casa.ID, hoje, hoje, 0, nil); err == nil {
		t.Fatal("título")
	}
	if _, err := svc.CriarContrato(ctx, gestor(u), "", "Link", casa.ID, hoje, hoje, 0, &outra); !recusa(err, "não gerencia") {
		t.Fatalf("contrato fora: %v", err)
	}
	if _, err := application.NewService(pool, falhando("InsertContrato"), semEventos{}).CriarContrato(ctx, quem, "", "Link", casa.ID, hoje, hoje, 0, nil); err == nil {
		t.Fatal("insert do contrato")
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), semEventos{application.EventoContrato}).CriarContrato(ctx, quem, "", "Link", casa.ID, hoje, hoje, 0, nil); err == nil {
		t.Fatal("evento do contrato")
	}
	contrato, err := svc.CriarContrato(ctx, quem, "9", "Link "+uuid.NewString()[:6], casa.ID, hoje, hoje.AddDate(0, 0, 40), 100, &u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.NewService(pool, falhando("ListContratos"), semEventos{}).ListarContratos(ctx, quem); err == nil {
		t.Fatal("consulta de contratos")
	}
	if err := svc.ExcluirContrato(ctx, ninguém, contrato.ID); !recusa(err, "não gerencia") {
		t.Fatalf("excluir contrato: %v", err)
	}
	if err := application.NewService(pool, falhando("DeleteContrato"), semEventos{}).ExcluirContrato(ctx, quem, contrato.ID); err == nil {
		t.Fatal("delete do contrato")
	}
	if err := application.NewService(pool, infrastructure.NewRepository(), semEventos{"contratos.contrato.excluido"}).ExcluirContrato(ctx, quem, contrato.ID); err == nil {
		t.Fatal("evento do contrato excluído")
	}

	if lista, err := svc.ListarLicencas(ctx, ninguém); err != nil || len(lista) != 0 {
		t.Fatalf("licenças sem escopo: %+v %v", lista, err)
	}
	if _, err := svc.CriarLicenca(ctx, quem, " ", casa.ID, nil, 1, "", nil); err == nil {
		t.Fatal("licença sem nome")
	}
	if _, err := svc.CriarLicenca(ctx, quem, "Office", casa.ID, nil, 1, strings.Repeat("a", 81), nil); err == nil {
		t.Fatal("programa")
	}
	if _, err := svc.CriarLicenca(ctx, gestor(u), "Office", casa.ID, nil, 1, "", &outra); !recusa(err, "não gerencia") {
		t.Fatalf("licença fora: %v", err)
	}
	if _, err := application.NewService(pool, falhando("InsertLicenca"), semEventos{}).CriarLicenca(ctx, quem, "Office", casa.ID, nil, 1, "", nil); err == nil {
		t.Fatal("insert da licença")
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), semEventos{application.EventoLicenca}).CriarLicenca(ctx, quem, "Office", casa.ID, nil, 1, "", nil); err == nil {
		t.Fatal("evento da licença")
	}
	fim := hoje.AddDate(0, 0, 5)
	daUnidade, err := svc.CriarLicenca(ctx, quem, "Office "+uuid.NewString()[:6], casa.ID, &fim, 3, "calc", &u)
	if err != nil {
		t.Fatal(err)
	}
	institucional, err := svc.CriarLicenca(ctx, quem, "Editor "+uuid.NewString()[:6], casa.ID, nil, 1, "writer", nil)
	if err != nil {
		t.Fatal(err)
	}
	semPrograma, err := svc.CriarLicenca(ctx, quem, "Sem programa "+uuid.NewString()[:6], casa.ID, nil, 1, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.NewService(pool, falhando("ListLicencas"), semEventos{}).ListarLicencas(ctx, quem); err == nil {
		t.Fatal("consulta de licenças")
	}
	semUso, err := svc.ListarLicencas(ctx, quem)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range semUso {
		if item.ID == daUnidade.ID && item.Uso != nil {
			t.Fatal("sem inventário não há uso")
		}
	}
	comUso := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).WithProgramas(programas{})
	achouUnidade, achouInst := false, false
	listaUso, errUso := comUso.ListarLicencas(ctx, quem)
	for _, item := range deveListar(t, listaUso, errUso) {
		if item.ID == daUnidade.ID && item.Uso != nil && item.Uso.Instalados == 2 && item.Uso.Parcial {
			achouUnidade = true
		}
		if item.ID == institucional.ID && item.Uso != nil {
			achouInst = true
		}
		if item.ID == semPrograma.ID && item.Uso != nil {
			t.Fatal("programa vazio não compara")
		}
	}
	if !achouUnidade || !achouInst {
		t.Fatal("uso da unidade e da instituição")
	}
	if _, err := comUso.ListarLicencas(ctx, gestor(u)); err != nil {
		t.Fatal(err)
	}
	institucionalNaLista := licencasFixas{repoFalho: falhando(""), lista: []domain.Licenca{{ID: uuid.New(), Nome: "Editor", Programa: "writer"}}}
	if lista, err := application.NewService(pool, institucionalNaLista, semEventos{}).WithProgramas(programas{}).ListarLicencas(ctx, gestor(u)); err != nil || len(lista) != 1 || lista[0].Uso == nil {
		t.Fatalf("licença sem unidade no recorte: %+v %v", lista, err)
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).WithProgramas(programas{err: application.ErrProgramasIndisponiveis}).ListarLicencas(ctx, quem); err != nil {
		t.Fatal(err)
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).WithProgramas(programas{err: errFalha}).ListarLicencas(ctx, quem); err == nil {
		t.Fatal("inventário")
	}
	if err := svc.ExcluirLicenca(ctx, ninguém, daUnidade.ID); !recusa(err, "não gerencia") {
		t.Fatalf("excluir licença: %v", err)
	}
	if err := application.NewService(pool, falhando("DeleteLicenca"), semEventos{}).ExcluirLicenca(ctx, quem, daUnidade.ID); err == nil {
		t.Fatal("delete da licença")
	}
	if err := svc.ExcluirLicenca(ctx, quem, daUnidade.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ExcluirLicenca(ctx, quem, institucional.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ExcluirLicenca(ctx, quem, semPrograma.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ExcluirContrato(ctx, quem, contrato.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ExcluirFornecedor(ctx, quem, casa.ID); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	transport.NewHandlers(svc, slog.New(slog.NewTextHandler(io.Discard, nil))).RegisterRoutes(router)
	pedir := func(method, caminho, corpo string, idt auth.Identity) int {
		t.Helper()
		req := httptest.NewRequest(method, caminho, strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithIdentity(req.Context(), idt))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	if pedir(http.MethodGet, "/contratos/fornecedores", "", quem) != http.StatusOK {
		t.Fatal("listar fornecedores")
	}
	if pedir(http.MethodPost, "/contratos/fornecedores", "{", quem) != http.StatusBadRequest {
		t.Fatal("json do fornecedor")
	}
	if pedir(http.MethodPost, "/contratos/fornecedores", `{"nome":" "}`, quem) != http.StatusUnprocessableEntity {
		t.Fatal("nome http")
	}
	if pedir(http.MethodPost, "/contratos/fornecedores", `{"nome":"Casa HTTP","unidade_id":"`+outra.String()+`"}`, gestor(u)) != http.StatusForbidden {
		t.Fatal("fornecedor de outra unidade")
	}
	if pedir(http.MethodPost, "/contratos/fornecedores", `{"nome":"Casa HTTP `+uuid.NewString()[:6]+`"}`, quem) != http.StatusCreated {
		t.Fatal("criar fornecedor")
	}
	if pedir(http.MethodDelete, "/contratos/fornecedores/nao", "", quem) != http.StatusBadRequest {
		t.Fatal("id")
	}
	if pedir(http.MethodDelete, "/contratos/fornecedores/"+uuid.NewString(), "", quem) != http.StatusNotFound {
		t.Fatal("fornecedor ausente")
	}
	if pedir(http.MethodGet, "/contratos/contratos", "", quem) != http.StatusOK {
		t.Fatal("listar contratos")
	}
	if pedir(http.MethodPost, "/contratos/contratos", "{", quem) != http.StatusBadRequest {
		t.Fatal("json do contrato")
	}
	corpo := `{"titulo":"Link HTTP","fornecedor_id":"` + casa.ID.String() + `","inicio":"` + hoje.Format(time.RFC3339) + `","fim":"` + hoje.Format(time.RFC3339) + `","valor_centavos":10}`
	if codigo := pedir(http.MethodPost, "/contratos/contratos", corpo, quem); codigo != http.StatusNotFound && codigo != http.StatusUnprocessableEntity {
		t.Fatalf("contrato de fornecedor já excluído: %d", codigo)
	}
	if pedir(http.MethodGet, "/contratos/licencas", "", quem) != http.StatusOK {
		t.Fatal("listar licenças")
	}
	if pedir(http.MethodPost, "/contratos/licencas", "{", quem) != http.StatusBadRequest {
		t.Fatal("json da licença")
	}
	if pedir(http.MethodDelete, "/contratos/licencas/nao", "", quem) != http.StatusBadRequest {
		t.Fatal("id da licença")
	}
	if pedir(http.MethodDelete, "/contratos/contratos/nao", "", quem) != http.StatusBadRequest {
		t.Fatal("id do contrato")
	}
	httpCasa, err := svc.CriarFornecedor(ctx, quem, "Casa rota "+uuid.NewString()[:6], "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_licencas WHERE fornecedor_id = $1`, httpCasa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_contratos WHERE fornecedor_id = $1`, httpCasa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_fornecedores WHERE id = $1`, httpCasa.ID)
	})
	contratoHTTP := `{"titulo":"Link da rota","fornecedor_id":"` + httpCasa.ID.String() + `","inicio":"` + hoje.Format(time.RFC3339) + `","fim":"` + hoje.AddDate(0, 0, 1).Format(time.RFC3339) + `","valor_centavos":10}`
	if pedir(http.MethodPost, "/contratos/contratos", contratoHTTP, quem) != http.StatusCreated {
		t.Fatal("criar contrato")
	}
	licencaHTTP := `{"nome":"Licença da rota","fornecedor_id":"` + httpCasa.ID.String() + `","quantidade":2,"programa":"calc"}`
	if pedir(http.MethodPost, "/contratos/licencas", licencaHTTP, quem) != http.StatusCreated {
		t.Fatal("criar licença")
	}
	criado, err := svc.CriarContrato(ctx, quem, "", "Para apagar", httpCasa.ID, hoje, hoje, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pedir(http.MethodDelete, "/contratos/contratos/"+criado.ID.String(), "", quem) != http.StatusNoContent {
		t.Fatal("apagar contrato")
	}
	licCriada, err := svc.CriarLicenca(ctx, quem, "Para apagar", httpCasa.ID, nil, 1, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pedir(http.MethodDelete, "/contratos/licencas/"+licCriada.ID.String(), "", quem) != http.StatusNoContent {
		t.Fatal("apagar licença")
	}
	sozinho, err := svc.CriarFornecedor(ctx, quem, "Sem vínculo "+uuid.NewString()[:6], "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pedir(http.MethodDelete, "/contratos/fornecedores/"+sozinho.ID.String(), "", quem) != http.StatusNoContent {
		t.Fatal("apagar fornecedor")
	}
	falhaContrato := chi.NewRouter()
	transport.NewHandlers(application.NewService(pool, falhando("ListContratos"), semEventos{}), slog.New(slog.DiscardHandler)).RegisterRoutes(falhaContrato)
	if pedirNo(t, falhaContrato, http.MethodGet, "/contratos/contratos", "", quem) < 500 {
		t.Fatal("lista de contratos")
	}
	falhaLicenca := chi.NewRouter()
	transport.NewHandlers(application.NewService(pool, falhando("ListLicencas"), semEventos{}), slog.New(slog.DiscardHandler)).RegisterRoutes(falhaLicenca)
	if pedirNo(t, falhaLicenca, http.MethodGet, "/contratos/licencas", "", quem) < 500 {
		t.Fatal("lista de licenças")
	}
	falhaCriar := chi.NewRouter()
	transport.NewHandlers(application.NewService(pool, falhando("InsertLicenca"), semEventos{}), slog.New(slog.DiscardHandler)).RegisterRoutes(falhaCriar)
	if pedirNo(t, falhaCriar, http.MethodPost, "/contratos/licencas", licencaHTTP, quem) < 400 {
		t.Fatal("licença não gravada")
	}

	quebrado := chi.NewRouter()
	transport.NewHandlers(application.NewService(pool, falhando("ListFornecedores"), semEventos{}), slog.New(slog.DiscardHandler)).RegisterRoutes(quebrado)
	req := httptest.NewRequest(http.MethodGet, "/contratos/fornecedores", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), quem))
	rec := httptest.NewRecorder()
	quebrado.ServeHTTP(rec, req)
	if rec.Code < 500 {
		t.Fatalf("lista com falha: %d", rec.Code)
	}
}

func pedirNo(t *testing.T, router http.Handler, method, caminho, corpo string, idt auth.Identity) int {
	t.Helper()
	req := httptest.NewRequest(method, caminho, strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), idt))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

func recusa(err error, trecho string) bool {
	return err != nil && strings.Contains(err.Error(), trecho)
}

func deveListar(t *testing.T, lista []domain.Licenca, err error) []domain.Licenca {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return lista
}

func TestRamosDoVencimento(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u := dbtest.Unidade(t, pool)
	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	calado := slog.New(slog.DiscardHandler)
	svc := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).
		WithNotificacoes(notificacoes.NewService(pool, nil, calado)).
		WithLogger(calado)
	casa, err := svc.CriarFornecedor(ctx, admin(), "Aviso "+uuid.NewString()[:6], "", "", &u)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_licencas WHERE fornecedor_id = $1`, casa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_contratos WHERE fornecedor_id = $1`, casa.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM contratos_fornecedores WHERE id = $1`, casa.ID)
	})
	criar := func(dias int) {
		t.Helper()
		fim := hoje.AddDate(0, 0, dias)
		if _, err := svc.CriarContrato(ctx, admin(), "", "Vence "+uuid.NewString()[:4], casa.ID, hoje.AddDate(0, 0, -10), fim, 0, &u); err != nil {
			t.Fatal(err)
		}
	}
	criar(30)
	criar(20)
	criar(5)
	criar(0)
	criar(-1)
	if _, err := svc.CriarLicenca(ctx, admin(), "Licença "+uuid.NewString()[:4], casa.ID, &hoje, 1, "", &u); err != nil {
		t.Fatal(err)
	}

	if err := application.NewService(pool, infrastructure.NewRepository(), semEventos{}).VerificarVencimentos(ctx); err != nil {
		t.Fatal(err)
	}
	if err := application.NewService(pool, falhando("Vencendo"), semEventos{}).WithNotificacoes(notificacoes.NewService(pool, nil, calado)).VerificarVencimentos(ctx); err == nil {
		t.Fatal("consulta")
	}
	semDestino := repoFalho{Repository: infrastructure.NewRepository(), falha: "Destinatarios", vencendo: []domain.Vencimento{{
		ID: uuid.New(), Titulo: "Parado", Fim: hoje, UnidadeID: &u,
	}}}
	if err := application.NewService(pool, semDestino, semEventos{}).WithNotificacoes(notificacoes.NewService(pool, nil, calado)).VerificarVencimentos(ctx); err == nil {
		t.Fatal("destinatários")
	}
	tituloLongo := repoFalho{Repository: infrastructure.NewRepository(), vencendo: []domain.Vencimento{{
		ID: uuid.New(), Titulo: strings.Repeat("a", 190), Fim: hoje,
	}}}
	if err := application.NewService(pool, tituloLongo, semEventos{}).WithNotificacoes(notificacoes.NewService(pool, nil, calado)).WithLogger(calado).VerificarVencimentos(ctx); err != nil {
		t.Fatal(err)
	}

	if err := svc.VerificarVencimentos(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.WithRelogio(func() time.Time { return hoje.AddDate(0, 0, -15) }).VerificarVencimentos(ctx); err != nil {
		t.Fatal(err)
	}

	ctxCancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := svc.RunVencimentos(ctxCancelado); err != nil {
		t.Fatal(err)
	}
	vivo, parar := context.WithCancel(ctx)
	saiu := make(chan struct{})
	go func() {
		_ = application.NewService(pool, falhando("Vencendo"), semEventos{}).
			WithNotificacoes(notificacoes.NewService(pool, nil, calado)).
			WithLogger(calado).
			RunVencimentos(vivo)
		close(saiu)
	}()
	time.Sleep(150 * time.Millisecond)
	parar()
	select {
	case <-saiu:
	case <-time.After(3 * time.Second):
		t.Fatal("vencimento não parou")
	}
}
