package application_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func relator(u uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &u, Permissions: []string{string(auth.PermAtivosReportar)}, Unidades: []uuid.UUID{u},
	}}}
}

func consulta(u uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &u, Permissions: []string{string(auth.PermAtendimentoAtender)}, Unidades: []uuid.UUID{u},
	}}}
}

type repoFalho struct {
	domain.Repository
	falha string
}

func (r repoFalho) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Ativo, error) {
	if r.falha == "List" {
		return nil, errFalha
	}
	return r.Repository.List(ctx, db, f)
}

func (r repoFalho) PorIdentificador(ctx context.Context, db database.DBTX, id string) (domain.Ativo, error) {
	if r.falha == "PorIdentificador" {
		return domain.Ativo{}, errFalha
	}
	return r.Repository.PorIdentificador(ctx, db, id)
}

func (r repoFalho) Insert(ctx context.Context, db database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	if r.falha == "Insert" {
		return domain.Ativo{}, errFalha
	}
	return r.Repository.Insert(ctx, db, a)
}

func (r repoFalho) Update(ctx context.Context, db database.DBTX, a domain.Ativo) (domain.Ativo, error) {
	if r.falha == "Update" {
		return domain.Ativo{}, errFalha
	}
	return r.Repository.Update(ctx, db, a)
}

func (r repoFalho) ListRelacoes(ctx context.Context, db database.DBTX, origem uuid.UUID) ([]domain.Relacao, error) {
	if r.falha == "ListRelacoes" {
		return nil, errFalha
	}
	return r.Repository.ListRelacoes(ctx, db, origem)
}

func (r repoFalho) InsertRelacao(ctx context.Context, db database.DBTX, item domain.Relacao) error {
	if r.falha == "InsertRelacao" {
		return errFalha
	}
	return r.Repository.InsertRelacao(ctx, db, item)
}

func (r repoFalho) DeleteRelacao(ctx context.Context, db database.DBTX, id, origem uuid.UUID) error {
	if r.falha == "DeleteRelacao" {
		return errFalha
	}
	return r.Repository.DeleteRelacao(ctx, db, id, origem)
}

func (r repoFalho) FechaCiclo(ctx context.Context, db database.DBTX, origem, destino uuid.UUID) (bool, error) {
	if r.falha == "FechaCiclo" {
		return false, errFalha
	}
	return r.Repository.FechaCiclo(ctx, db, origem, destino)
}

func (r repoFalho) InsertCredencial(ctx context.Context, db database.DBTX, c domain.Credencial) error {
	if r.falha == "InsertCredencial" {
		return errFalha
	}
	return r.Repository.InsertCredencial(ctx, db, c)
}

func (r repoFalho) ListCredenciais(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Credencial, error) {
	if r.falha == "ListCredenciais" {
		return nil, errFalha
	}
	return r.Repository.ListCredenciais(ctx, db, f)
}

func (r repoFalho) RevogarCredencial(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	if r.falha == "RevogarCredencial" {
		return errFalha
	}
	return r.Repository.RevogarCredencial(ctx, db, id, f)
}

func (r repoFalho) PorHash(ctx context.Context, db database.DBTX, hash []byte) (domain.Credencial, error) {
	if r.falha == "PorHash" {
		return domain.Credencial{}, errFalha
	}
	return r.Repository.PorHash(ctx, db, hash)
}

type quadro struct {
	err error
	m   map[string][]string
}

func (q quadro) Mapa(context.Context, database.DBTX, string) (map[string][]string, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.m, nil
}

func TestRamosDosAtivos(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	u, outra := dbtest.Unidade(t, pool), dbtest.Unidade(t, pool)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM ativo_relacoes WHERE origem_id IN (SELECT id FROM ativos WHERE unidade_id = ANY($1)) OR destino_id IN (SELECT id FROM ativos WHERE unidade_id = ANY($1))`, []uuid.UUID{u, outra})
		_, _ = pool.Exec(ctx, `DELETE FROM ativos_credenciais WHERE unidade_id = ANY($1)`, []uuid.UUID{u, outra})
		_, _ = pool.Exec(ctx, `DELETE FROM ativos WHERE unidade_id = ANY($1)`, []uuid.UUID{u, outra})
	})
	quem := admin()
	svc := application.NewService(pool, infrastructure.NewRepository(), eventos{}, nil)
	entrada := func(nome string, unidade uuid.UUID) domain.Entrada {
		return domain.Entrada{Nome: nome, Tipo: "hardware", Estado: "planejado", Identificador: nome, UnidadeID: &unidade}
	}

	duplicada := auth.Identity{Scopes: []auth.Scope{
		{UnidadeID: &u, Permissions: []string{"ativos:read"}, Unidades: []uuid.UUID{u}},
		{UnidadeID: &u, Permissions: []string{"ativos:manage"}, Unidades: []uuid.UUID{u}},
	}}
	if _, err := svc.Listar(ctx, duplicada, nil); err != nil {
		t.Fatal(err)
	}
	svc.ComPassagens(quadro{err: errFalha})
	if _, err := svc.Criar(ctx, quem, entrada("Mapa", u)); err == nil {
		t.Fatal("mapa")
	}
	svc.ComPassagens(nil)

	a, err := svc.Criar(ctx, gestor(u), entrada("Servidor "+uuid.NewString()[:6], u))
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Criar(ctx, quem, entrada("Link "+uuid.NewString()[:6], u))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Atualizar(ctx, quem, a.ID, domain.Entrada{Nome: a.Nome, Tipo: a.Tipo, Estado: a.Estado, Identificador: a.Identificador, UnidadeID: a.UnidadeID}); err != nil {
		t.Fatal(err)
	}
	coluna := "estoque"
	comColuna := application.NewService(pool, infrastructure.NewRepository(), eventos{}, nil)
	comColuna.ComPassagens(quadro{m: map[string][]string{"em_uso": {coluna}}})
	if _, err := comColuna.Atualizar(ctx, quem, a.ID, domain.Entrada{Nome: a.Nome, Tipo: a.Tipo, Estado: coluna, Identificador: a.Identificador, UnidadeID: a.UnidadeID}); !errors.Is(err, domain.ErrEstadoInvalido) {
		t.Fatalf("coluna de outra faixa: %v", err)
	}

	if _, err := svc.Relacionar(ctx, quem, a.ID, a.ID, "conectado"); !relacaoRuim(err) {
		t.Fatalf("par: %v", err)
	}
	if _, err := svc.Relacionar(ctx, consulta(u), a.ID, b.ID, "conectado"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("consulta: %v", err)
	}
	semMapa := application.NewService(pool, infrastructure.NewRepository(), eventos{}, nil)
	semMapa.ComPassagens(quadro{err: errFalha})
	if _, err := semMapa.Atualizar(ctx, quem, a.ID, domain.Entrada{Nome: a.Nome, Tipo: a.Tipo, Estado: a.Estado, Identificador: a.Identificador, UnidadeID: a.UnidadeID}); err == nil {
		t.Fatal("mapa da edição")
	}
	if _, err := svc.Relacionar(ctx, quem, uuid.New(), b.ID, "conectado"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("origem ausente: %v", err)
	}
	if _, err := svc.Relacionar(ctx, quem, a.ID, uuid.New(), "conectado"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("destino ausente: %v", err)
	}
	fora, err := svc.Criar(ctx, quem, entrada("Fora "+uuid.NewString()[:6], outra))
	if err != nil {
		t.Fatal(err)
	}
	misto := auth.Identity{Scopes: []auth.Scope{
		{UnidadeID: &u, Permissions: []string{string(auth.PermAtivosManage)}, Unidades: []uuid.UUID{u}},
		{UnidadeID: &outra, Permissions: []string{string(auth.PermAtendimentoAtender)}, Unidades: []uuid.UUID{outra}},
	}}
	if _, err := svc.Relacionar(ctx, misto, a.ID, fora.ID, "conectado"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("destino sem gestão: %v", err)
	}
	liga, err := svc.Relacionar(ctx, quem, a.ID, b.ID, "conectado")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Relacionar(ctx, quem, a.ID, b.ID, "depende"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Relacionar(ctx, quem, b.ID, a.ID, "depende"); !relacaoRuim(err) {
		t.Fatalf("ciclo: %v", err)
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "FechaCiclo"}, eventos{}, nil).Relacionar(ctx, quem, b.ID, a.ID, "depende"); err == nil {
		t.Fatal("ciclo ilegível")
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "InsertRelacao"}, eventos{}, nil).Relacionar(ctx, quem, a.ID, b.ID, "composto"); err == nil {
		t.Fatal("gravar relação")
	}
	if _, err := svc.ListarRelacoes(ctx, quem, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "ListRelacoes"}, eventos{}, nil).ListarRelacoes(ctx, quem, a.ID); err == nil {
		t.Fatal("ler relações")
	}
	if err := svc.Desrelacionar(ctx, consulta(u), a.ID, liga.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("desligar sem gestão: %v", err)
	}
	if err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "DeleteRelacao"}, eventos{}, nil).Desrelacionar(ctx, quem, a.ID, liga.ID); err == nil {
		t.Fatal("apagar relação")
	}
	if err := svc.Desrelacionar(ctx, quem, a.ID, liga.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Desrelacionar(ctx, quem, a.ID, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("relação ausente: %v", err)
	}

	maquina := relator(u)
	idMaquina := "PC-" + uuid.NewString()[:8]
	primeiro, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: idMaquina, Nome: "Balcão"})
	if err != nil || primeiro.Origem != domain.OrigemAgente || primeiro.Estado != "em_uso" {
		t.Fatalf("abrir: %+v %v", primeiro, err)
	}
	if repetido, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: " " + idMaquina, Nome: "Balcão"}); err != nil || repetido.ID != primeiro.ID {
		t.Fatalf("repetido: %+v %v", repetido, err)
	}
	if atualizado, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: idMaquina, Nome: "Balcão 2"}); err != nil || atualizado.Nome != "Balcão 2" {
		t.Fatalf("atualizar relato: %+v %v", atualizado, err)
	}
	if _, err := svc.Inventariar(ctx, quem, domain.Relato{Identificador: idMaquina, Nome: "X"}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("admin sem unidade: %v", err)
	}
	if _, err := svc.Inventariar(ctx, relator(outra), domain.Relato{Identificador: idMaquina, Nome: "X", UnidadeID: u}); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("unidade alheia: %v", err)
	}
	duas := auth.Identity{Scopes: []auth.Scope{{
		Permissions: []string{string(auth.PermAtivosReportar)}, Unidades: []uuid.UUID{u, outra},
	}}}
	if _, err := svc.Inventariar(ctx, duas, domain.Relato{Identificador: "outro", Nome: "Y"}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("duas unidades: %v", err)
	}
	manual, err := svc.Criar(ctx, quem, entrada("Manual "+uuid.NewString()[:6], u))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: manual.Identificador, Nome: "Agente"}); !errors.Is(err, domain.ErrCadastroManual) {
		t.Fatalf("manual: %v", err)
	}
	if _, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: "ruim", Inventario: &domain.Inventario{Sistema: "linux"}}); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("inventário: %v", err)
	}
	if _, err := svc.Inventariar(ctx, maquina, domain.Relato{Identificador: "tipo-ruim-" + uuid.NewString()[:4], Tipo: "nave"}); !errors.Is(err, domain.ErrTipoInvalido) {
		t.Fatalf("tipo do relato: %v", err)
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "Update"}, eventos{}, nil).Inventariar(ctx, maquina, domain.Relato{Identificador: idMaquina, Nome: "Balcão 3"}); err == nil {
		t.Fatal("regravar relato")
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "PorIdentificador"}, eventos{}, nil).Inventariar(ctx, maquina, domain.Relato{Identificador: "novo-pc", Nome: "Novo"}); err == nil {
		t.Fatal("leitura")
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "Insert"}, eventos{}, nil).Inventariar(ctx, maquina, domain.Relato{Identificador: "pc-novo-" + uuid.NewString()[:4], Nome: "Novo"}); err == nil {
		t.Fatal("insert do relato")
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), eventos{errFalha}, nil).Inventariar(ctx, maquina, domain.Relato{Identificador: "pc-evt-" + uuid.NewString()[:4], Nome: "Novo"}); err == nil {
		t.Fatal("evento do relato")
	}

	if _, err := svc.EmitirCredencial(ctx, gestor(outra), "Agente", u); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("credencial fora: %v", err)
	}
	if _, err := svc.EmitirCredencial(ctx, quem, " ", u); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Fatalf("nome: %v", err)
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "InsertCredencial"}, eventos{}, nil).EmitirCredencial(ctx, quem, "Agente", u); err == nil {
		t.Fatal("gravar credencial")
	}
	if _, err := application.NewService(pool, infrastructure.NewRepository(), eventos{errFalha}, nil).EmitirCredencial(ctx, quem, "Agente", u); err == nil {
		t.Fatal("evento da credencial")
	}
	emitida, err := svc.EmitirCredencial(ctx, quem, "Balcão", u)
	if err != nil || !strings.HasPrefix(emitida.Token, domain.PrefixoCredencial) {
		t.Fatalf("emitir: %+v %v", emitida, err)
	}
	if lista, err := svc.ListarCredenciais(ctx, gestor(u)); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	if lista, err := svc.ListarCredenciais(ctx, auth.Identity{}); err != nil || len(lista) != 0 {
		t.Fatalf("sem escopo: %+v %v", lista, err)
	}
	if _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "ListCredenciais"}, eventos{}, nil).ListarCredenciais(ctx, quem); err == nil {
		t.Fatal("ler credenciais")
	}
	id, ok, err := svc.Reconhecer(ctx, emitida.Token)
	if err != nil || !ok || id.Source != auth.SourceAgente || id.Subject != emitida.ID.String() {
		t.Fatalf("reconhecer: %+v %v %v", id, ok, err)
	}
	if _, ok, err := svc.Reconhecer(ctx, "curto"); ok || err != nil {
		t.Fatalf("não é credencial: %v %v", ok, err)
	}
	if _, ok, err := svc.Reconhecer(ctx, domain.PrefixoCredencial+"ausente"); err == nil || !ok {
		t.Fatalf("desconhecida: %v %v", ok, err)
	}
	if _, _, err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "PorHash"}, eventos{}, nil).Reconhecer(ctx, emitida.Token); err == nil {
		t.Fatal("hash")
	}
	if err := svc.RevogarCredencial(ctx, auth.Identity{}, emitida.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("revogar sem escopo: %v", err)
	}
	if err := application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "RevogarCredencial"}, eventos{}, nil).RevogarCredencial(ctx, quem, emitida.ID); err == nil {
		t.Fatal("revogar")
	}
	if err := svc.RevogarCredencial(ctx, quem, emitida.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.Reconhecer(ctx, emitida.Token); err == nil || !ok {
		t.Fatalf("revogada: %v %v", ok, err)
	}

	router := chi.NewRouter()
	transport.NewHandlers(svc, slog.New(slog.DiscardHandler)).RegisterRoutes(router)
	pedir := func(method, caminho, corpo string, idt auth.Identity) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, caminho, strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithIdentity(req.Context(), idt))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	deve := func(method, caminho, corpo string, idt auth.Identity, status int) {
		t.Helper()
		rec := pedir(method, caminho, corpo, idt)
		if rec.Code != status {
			t.Fatalf("%s %s: %d %s", method, caminho, rec.Code, rec.Body.String())
		}
	}
	deve(http.MethodGet, "/ativos?produto_id=nao", "", quem, http.StatusBadRequest)
	deve(http.MethodGet, "/ativos/"+a.ID.String(), "", quem, http.StatusOK)
	deve(http.MethodGet, "/ativos/nao", "", quem, http.StatusBadRequest)
	corpo := `{"nome":"Pelo HTTP","tipo":"rede","estado":"em_uso","identificador":"HTTP-` + uuid.NewString()[:8] + `","unidade_id":"` + u.String() + `"}`
	deve(http.MethodPost, "/ativos", "{", quem, http.StatusBadRequest)
	deve(http.MethodPost, "/ativos", corpo, quem, http.StatusCreated)
	edicao := `{"nome":"Servidor editado","tipo":"hardware","estado":"em_uso","identificador":"` + a.Identificador + `","unidade_id":"` + u.String() + `"}`
	deve(http.MethodPut, "/ativos/"+a.ID.String(), edicao, quem, http.StatusOK)
	deve(http.MethodGet, "/ativos/nao/relacoes", "", quem, http.StatusBadRequest)
	deve(http.MethodGet, "/ativos/"+uuid.NewString()+"/relacoes", "", quem, http.StatusNotFound)
	deve(http.MethodGet, "/ativos/"+a.ID.String()+"/relacoes", "", quem, http.StatusOK)
	deve(http.MethodPost, "/ativos/"+a.ID.String()+"/relacoes", "{", quem, http.StatusBadRequest)
	deve(http.MethodPost, "/ativos/"+a.ID.String()+"/relacoes", `{"destino_id":"`+a.ID.String()+`","tipo":"conectado"}`, quem, http.StatusUnprocessableEntity)
	deve(http.MethodPost, "/ativos/"+a.ID.String()+"/relacoes", `{"destino_id":"`+b.ID.String()+`","tipo":"composto"}`, quem, http.StatusCreated)
	deve(http.MethodPost, "/ativos/nao/relacoes", `{}`, quem, http.StatusBadRequest)
	deve(http.MethodDelete, "/ativos/nao/relacoes/"+uuid.NewString(), "", quem, http.StatusBadRequest)
	deve(http.MethodDelete, "/ativos/"+a.ID.String()+"/relacoes/nao", "", quem, http.StatusBadRequest)
	ligaHTTP, err := svc.Relacionar(ctx, quem, a.ID, b.ID, "conectado")
	if err != nil {
		t.Fatal(err)
	}
	deve(http.MethodDelete, "/ativos/"+a.ID.String()+"/relacoes/"+ligaHTTP.ID.String(), "", quem, http.StatusNoContent)
	deve(http.MethodPost, "/ativos/inventario", `{"identificador":"HTTP-`+uuid.NewString()[:6]+`","nome":"HTTP","unidade_id":"`+u.String()+`"}`, quem, http.StatusOK)
	deve(http.MethodPost, "/ativos/inventario", "{", quem, http.StatusBadRequest)
	deve(http.MethodPost, "/ativos/credenciais", "{", quem, http.StatusBadRequest)
	deve(http.MethodPost, "/ativos/credenciais", `{"nome":"HTTP","unidade_id":"`+u.String()+`"}`, quem, http.StatusCreated)
	deve(http.MethodGet, "/ativos/credenciais", "", quem, http.StatusOK)
	viva, err := svc.EmitirCredencial(ctx, quem, "Apagar", u)
	if err != nil {
		t.Fatal(err)
	}
	deve(http.MethodDelete, "/ativos/credenciais/"+viva.ID.String(), "", quem, http.StatusNoContent)
	deve(http.MethodDelete, "/ativos/credenciais/nao", "", quem, http.StatusBadRequest)
	deve(http.MethodDelete, "/ativos/credenciais/"+emitida.ID.String(), "", quem, http.StatusNotFound)
	recusa := transport.NewHandlers(svc, slog.New(slog.DiscardHandler))
	recusaRec := httptest.NewRecorder()
	recusaReq := httptest.NewRequest(http.MethodPost, "/ativos/credenciais", strings.NewReader(`{"nome":"X","unidade_id":"`+u.String()+`"}`))
	recusaReq.Header.Set("Content-Type", "application/json")
	recusaReq = recusaReq.WithContext(auth.WithIdentity(recusaReq.Context(), gestor(outra)))
	recusa.EmitirCredencial(recusaRec, recusaReq)
	if recusaRec.Code != http.StatusForbidden {
		t.Fatalf("emitir fora: %d %s", recusaRec.Code, recusaRec.Body.String())
	}
	falho := transport.NewHandlers(application.NewService(pool, repoFalho{Repository: infrastructure.NewRepository(), falha: "ListCredenciais"}, eventos{}, nil), slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ativos/credenciais", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), quem))
	falho.ListarCredenciais(rec, req)
	if rec.Code < 500 {
		t.Fatalf("lista: %d %s", rec.Code, rec.Body.String())
	}
}

func relacaoRuim(err error) bool {
	var e domain.RelacaoInvalida
	return errors.As(err, &e)
}
