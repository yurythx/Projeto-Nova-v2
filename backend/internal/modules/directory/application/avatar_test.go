package application_test

import (
	"bytes"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/directory/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/directory/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/modules/directory/transport"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

// txAvatar derruba uma etapa da eliminação da foto depois que o perfil já foi apagado.
type txAvatar struct {
	database.DBTX
	falhaLeitura  bool
	falhaGravacao bool
}

func (t txAvatar) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if t.falhaLeitura && strings.Contains(sql, "avatar_key") && strings.HasPrefix(strings.TrimSpace(sql), "SELECT") {
		return linhaFalha{}
	}
	return t.DBTX.QueryRow(ctx, sql, args...)
}

type linhaFalha struct{}

func (linhaFalha) Scan(...any) error { return errors.New("não leu") }

func (t txAvatar) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if t.falhaGravacao && strings.Contains(sql, "avatar_key") {
		return pgconn.CommandTag{}, errors.New("não gravou")
	}
	return t.DBTX.Exec(ctx, sql, args...)
}

type memFoto struct {
	objs            map[string]storage.ObjectInfo
	corpo           map[string][]byte
	presignErr      error
	statErr, getErr error
	apagadas        []string
}

func (m *memFoto) Ping(context.Context) error { return nil }
func (m *memFoto) Put(_ context.Context, _, key string, r io.Reader, _ int64, ct string) error {
	b, _ := io.ReadAll(r)
	m.corpo[key] = b
	m.objs[key] = storage.ObjectInfo{Size: int64(len(b)), ContentType: ct}
	return nil
}
func (m *memFoto) Get(_ context.Context, _, key string) (io.ReadCloser, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return io.NopCloser(bytes.NewReader(m.corpo[key])), nil
}
func (m *memFoto) Delete(_ context.Context, _, key string) error {
	m.apagadas = append(m.apagadas, key)
	delete(m.objs, key)
	delete(m.corpo, key)
	return nil
}
func (m *memFoto) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	if m.presignErr != nil {
		return "", m.presignErr
	}
	return "http://armazenamento/enviar", nil
}
func (m *memFoto) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return "http://armazenamento/ver", nil
}
func (m *memFoto) Stat(_ context.Context, _, key string) (storage.ObjectInfo, error) {
	if m.statErr != nil {
		return storage.ObjectInfo{}, m.statErr
	}
	info, ok := m.objs[key]
	if !ok {
		return storage.ObjectInfo{}, storage.ErrObjectNotFound
	}
	return info, nil
}

func novaFoto() *memFoto {
	return &memFoto{objs: map[string]storage.ObjectInfo{}, corpo: map[string][]byte{}}
}

func TestAvatar(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	pessoa := dbtest.User(t, pool)
	idt := auth.Identity{UserID: pessoa}
	repo := infrastructure.NewRepository()
	svc := application.NewService(pool, repo)
	store := novaFoto()

	if _, err := svc.PedirAvatar(ctx, idt, "foto.png", "image/png"); err == nil {
		t.Fatal("sem armazenamento")
	}
	if err := svc.SalvarAvatar(ctx, idt, "avatares/x"); err == nil {
		t.Fatal("salvar sem armazenamento")
	}
	if _, _, err := svc.AbrirAvatar(ctx, pessoa); err == nil {
		t.Fatal("abrir sem armazenamento")
	}
	if _, err := svc.PedirAvatar(ctx, auth.Identity{}, "foto.png", "image/png"); err == nil {
		t.Fatal("sem pessoa")
	}

	svc.ComArquivos(application.Arquivos{Store: store, Bucket: "fotos"})
	if _, err := svc.PedirAvatar(ctx, auth.Identity{}, "foto.png", "image/png"); err == nil {
		t.Fatal("pedido sem pessoa")
	}
	if err := svc.SalvarAvatar(ctx, auth.Identity{}, "x"); err == nil {
		t.Fatal("salvar sem pessoa")
	}
	if err := svc.RemoverAvatar(ctx, auth.Identity{}); err == nil {
		t.Fatal("remover sem pessoa")
	}
	store.presignErr = errors.New("fora")
	if _, err := svc.PedirAvatar(ctx, idt, "foto.png", "image/png"); err == nil {
		t.Fatal("url")
	}
	store.presignErr = nil
	if _, err := svc.PedirAvatar(ctx, idt, "foto.png", "text/plain"); err == nil {
		t.Fatal("tipo")
	}
	ticket, err := svc.PedirAvatar(ctx, idt, "foto.png", "image/png")
	if err != nil || ticket.ObjectKey == "" {
		t.Fatal(err)
	}
	if err := svc.SalvarAvatar(ctx, idt, "../fora"); err == nil {
		t.Fatal("chave")
	}
	if err := svc.SalvarAvatar(ctx, idt, ticket.ObjectKey); err == nil {
		t.Fatal("arquivo ainda não enviado")
	}
	store.objs[ticket.ObjectKey] = storage.ObjectInfo{Size: 4, ContentType: "image/png"}
	store.corpo[ticket.ObjectKey] = []byte("foto")
	if err := svc.SalvarAvatar(ctx, idt, ticket.ObjectKey); err != nil {
		t.Fatal(err)
	}
	rc, tipo, err := svc.AbrirAvatar(ctx, pessoa)
	if err != nil || tipo != "image/png" {
		t.Fatalf("abrir: %v %s", err, tipo)
	}
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	store.objs[ticket.ObjectKey] = storage.ObjectInfo{Size: 4, ContentType: ""}
	if _, tipo, err = svc.AbrirAvatar(ctx, pessoa); err != nil || tipo != "application/octet-stream" {
		t.Fatalf("tipo vazio: %v %s", err, tipo)
	}
	store.getErr = errors.New("leitura")
	if _, _, err = svc.AbrirAvatar(ctx, pessoa); err == nil {
		t.Fatal("leitura")
	}
	store.getErr = nil
	store.statErr = errors.New("stat")
	if _, _, err = svc.AbrirAvatar(ctx, pessoa); err == nil {
		t.Fatal("stat")
	}
	store.statErr = nil

	outro, err := svc.PedirAvatar(ctx, idt, "nova.png", "image/png")
	if err != nil {
		t.Fatal(err)
	}
	store.objs[outro.ObjectKey] = storage.ObjectInfo{Size: 4, ContentType: "image/png"}
	store.corpo[outro.ObjectKey] = []byte("nova")
	if err := svc.SalvarAvatar(ctx, idt, outro.ObjectKey); err != nil {
		t.Fatal(err)
	}
	if len(store.apagadas) == 0 {
		t.Fatal("a foto anterior fica no armazenamento")
	}
	if err := svc.RemoverAvatar(ctx, idt); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.AbrirAvatar(ctx, pessoa); err == nil {
		t.Fatal("foto removida")
	}
	ausente := uuid.New()
	chaveAusente := "avatares/" + ausente.String() + "/2026/11/" + uuid.NewString() + ".png"
	store.objs[chaveAusente] = storage.ObjectInfo{Size: 4, ContentType: "image/png"}
	store.corpo[chaveAusente] = []byte("foto")
	if err := svc.SalvarAvatar(ctx, auth.Identity{UserID: ausente}, chaveAusente); err == nil {
		t.Fatal("pessoa inexistente")
	}
	if err := svc.RemoverAvatar(ctx, auth.Identity{UserID: ausente}); err == nil {
		t.Fatal("remover pessoa inexistente")
	}

	semFoto := application.NewService(pool, &faultRepo{inner: infrastructure.NewRepository()})
	semFoto.ComArquivos(application.Arquivos{Store: store, Bucket: "fotos"})
	if err := semFoto.SalvarAvatar(ctx, idt, ticket.ObjectKey); err == nil {
		t.Fatal("repositório sem foto")
	}
	if err := semFoto.RemoverAvatar(ctx, idt); err == nil {
		t.Fatal("remover sem foto")
	}
	if _, _, err := semFoto.AbrirAvatar(ctx, pessoa); err == nil {
		t.Fatal("abrir sem foto")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	if _, err := tx.Exec(ctx, `UPDATE users SET avatar_key = $2 WHERE id = $1`, pessoa, outro.ObjectKey); err != nil {
		t.Fatal(err)
	}
	if err := svc.ErasePersonalData(ctx, tx, pessoa); err != nil {
		t.Fatal(err)
	}
	if err := svc.ErasePersonalData(ctx, txAvatar{DBTX: pool, falhaLeitura: true}, pessoa); err == nil {
		t.Fatal("leitura da foto")
	}
	if err := svc.ErasePersonalData(ctx, txAvatar{DBTX: pool, falhaGravacao: true}, pessoa); err == nil {
		t.Fatal("gravação da foto")
	}
}

func TestHandlersDoAvatar(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()
	pessoa := dbtest.User(t, pool)
	store := novaFoto()
	svc := application.NewService(pool, infrastructure.NewRepository()).ComArquivos(application.Arquivos{Store: store, Bucket: "fotos"})
	h := transport.NewHandlers(svc, slog.New(slog.DiscardHandler), 50)
	r := chi.NewRouter()
	h.RegisterAuthedRoutes(r)
	chamar := func(method, path, corpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{UserID: pessoa}))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	if chamar(http.MethodPost, "/directory/me/avatar/uploads", "{").Code < 400 {
		t.Fatal("json")
	}
	if chamar(http.MethodPost, "/directory/me/avatar/uploads", `{"filename":"foto.png","content_type":"text/plain"}`).Code < 400 {
		t.Fatal("tipo")
	}
	sem := httptest.NewRequest(http.MethodDelete, "/directory/me/avatar", nil)
	recSem := httptest.NewRecorder()
	r.ServeHTTP(recSem, sem)
	if recSem.Code < 400 {
		t.Fatal("remover sem sessão")
	}
	rec := chamar(http.MethodPost, "/directory/me/avatar/uploads", `{"filename":"foto.png","content_type":"image/png"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "object_key") {
		t.Fatalf("pedido: %d %s", rec.Code, rec.Body.String())
	}
	chave := rec.Body.String()
	ini := strings.Index(chave, "avatares/")
	fim := strings.Index(chave[ini:], `"`)
	chave = chave[ini : ini+fim]
	if chamar(http.MethodPut, "/directory/me/avatar", "{").Code < 400 {
		t.Fatal("salvar json")
	}
	if chamar(http.MethodPut, "/directory/me/avatar", `{"object_key":"../x"}`).Code < 400 {
		t.Fatal("chave")
	}
	store.objs[chave] = storage.ObjectInfo{Size: 4, ContentType: "image/png"}
	store.corpo[chave] = []byte("foto")
	if chamar(http.MethodPut, "/directory/me/avatar", `{"object_key":"`+chave+`"}`).Code != http.StatusNoContent {
		t.Fatal("salvar")
	}
	aberto := chamar(http.MethodGet, "/directory/avatars/"+pessoa.String(), "")
	if aberto.Code != http.StatusOK || aberto.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("abrir: %d %s", aberto.Code, aberto.Body.String())
	}
	store.getErr = errors.New("leitura")
	if chamar(http.MethodGet, "/directory/avatars/"+pessoa.String(), "").Code == http.StatusNotFound {
		t.Fatal("falha de leitura")
	}
	store.getErr = nil
	if chamar(http.MethodGet, "/directory/avatars/nao-e-uuid", "").Code < 400 {
		t.Fatal("id")
	}
	if chamar(http.MethodGet, "/directory/avatars/"+uuid.NewString(), "").Code != http.StatusNotFound {
		t.Fatal("ausente")
	}
	if chamar(http.MethodDelete, "/directory/me/avatar", "").Code != http.StatusNoContent {
		t.Fatal("remover")
	}
	_ = ctx
}
