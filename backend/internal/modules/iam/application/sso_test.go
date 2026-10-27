package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// diretorio é o SSO do teste. segurar faz a leitura esperar a liberação.
type diretorio struct {
	mu      sync.Mutex
	pessoas []domain.UsuarioSSO
	err     error
	semSSO  bool
	segurar chan struct{}
}

func (d *diretorio) Configurado(context.Context) bool { return !d.semSSO }

func (d *diretorio) Usuarios(context.Context) ([]domain.UsuarioSSO, error) {
	if d.segurar != nil {
		<-d.segurar
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pessoas, d.err
}

func (d *diretorio) trocar(p []domain.UsuarioSSO, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pessoas, d.err = p, err
}

// sincronizar pede, espera terminar e devolve a última sincronização.
func sincronizar(t *testing.T, s *application.Service, ctx context.Context) application.RelatorioSSO {
	t.Helper()
	if err := s.SincronizarSSO(ctx); err != nil {
		t.Fatal(err)
	}
	s.AguardarSSO()
	e, err := s.EstadoDoSSO(ctx)
	if err != nil || e.Andamento || e.Ultima == nil {
		t.Fatalf("estado: %+v %v", e, err)
	}
	var rel application.RelatorioSSO
	if err := json.Unmarshal(e.Ultima.Relatorio, &rel); err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestSincronizarSSO(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx
	sfx := uuid.NewString()[:8]
	ana := domain.UsuarioSSO{Subject: "kc-ana-" + sfx, Username: "ana." + sfx, Email: "ana." + sfx + "@sso.test", Nome: "Ana Souza", Ativo: true,
		Grupos: []string{"Administrators"}}
	semEmail := domain.UsuarioSSO{Subject: "kc-sem-" + sfx, Ativo: true}
	conta, err := e.real().CreateLocalUser(ctx, application.CreateLocalUserInput{Username: "local." + sfx, Email: "local." + sfx + "@x.test",
		DisplayName: "Local", Password: "Senha-Forte-123!"})
	e.must(err)
	conflito := domain.UsuarioSSO{Subject: "kc-local-" + sfx, Username: conta.Username, Email: "outro." + sfx + "@x.test", Ativo: true}
	d := &diretorio{pessoas: []domain.UsuarioSSO{ana, semEmail, conflito, {Subject: ""}}}
	invalidou := 0
	s := application.NewService(e.pool, infrastructure.NewRepository(), func(context.Context) { invalidou++ }).
		WithDiretorio(d, slog.New(slog.DiscardHandler))

	rel := sincronizar(t, s, ctx)
	if rel.Lidos != 3 || rel.Criados != 2 || len(rel.Conflitos) != 1 || rel.Conflitos[0] != conta.Username || invalidou != 1 {
		t.Fatalf("primeira: %+v", rel)
	}
	var nome, email string
	var ativo bool
	var grupos []string
	e.must(e.pool.QueryRow(ctx, `SELECT display_name, email, active, groups FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&nome, &email, &ativo, &grupos))
	if nome != "Ana Souza" || !ativo || len(grupos) != 1 {
		t.Fatalf("ana: %s %v %v", nome, ativo, grupos)
	}
	e.must(e.pool.QueryRow(ctx, `SELECT email FROM users WHERE keycloak_subject = $1`, semEmail.Subject).Scan(&email))
	if email != semEmail.Subject+"@sem-email.invalid" {
		t.Fatalf("sem e-mail: %s", email)
	}

	// Desativada no SSO: desativa aqui. Reativada lá: não volta sozinha.
	ana.Ativo, ana.Nome = false, ""
	d.trocar([]domain.UsuarioSSO{ana}, nil)
	if rel := sincronizar(t, s, ctx); rel.Atualizados != 1 || rel.Desativados != 1 {
		t.Fatalf("desativar: %+v", rel)
	}
	ana.Ativo = true
	d.trocar([]domain.UsuarioSSO{ana}, nil)
	if rel := sincronizar(t, s, ctx); rel.Desativados != 0 {
		t.Fatalf("reativar: %+v", rel)
	}
	e.must(e.pool.QueryRow(ctx, `SELECT display_name, active FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&nome, &ativo))
	if ativo || nome != "Ana Souza" {
		t.Fatalf("continua desativada e com o nome: %s %v", nome, ativo)
	}

	// Uma de cada vez: a segunda espera a primeira terminar.
	d.segurar = make(chan struct{})
	e.must(s.SincronizarSSO(ctx))
	if err := s.SincronizarSSO(ctx); status(err) != http.StatusConflict {
		t.Fatalf("duas ao mesmo tempo: %v", err)
	}
	if est, err := s.EstadoDoSSO(ctx); err != nil || !est.Andamento {
		t.Fatalf("em andamento: %+v %v", est, err)
	}
	close(d.segurar)
	s.AguardarSSO()
	d.segurar = nil

	// Permissão, ligação e configuração.
	semPermissao := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New()})
	if err := s.SincronizarSSO(semPermissao); status(err) != http.StatusForbidden {
		t.Fatalf("sem users:manage global: %v", err)
	}
	if _, err := s.EstadoDoSSO(semPermissao); status(err) != http.StatusForbidden {
		t.Fatalf("estado sem permissão: %v", err)
	}
	if err := e.real().SincronizarSSO(ctx); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sem diretório: %v", err)
	}
	if err := application.NewService(e.pool, infrastructure.NewRepository(), nil).WithDiretorio(&diretorio{semSSO: true}, slog.New(slog.DiscardHandler)).
		SincronizarSSO(ctx); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("SSO sem conta de serviço: %v", err)
	}
	falha := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: 1}, nil)
	if _, err := falha.EstadoDoSSO(ctx); err == nil {
		t.Fatal("estado com o banco fora")
	}
}

func TestSincronizacaoComFalhas(t *testing.T) {
	e := newEnv(t)
	var registro bytes.Buffer
	log := slog.New(slog.NewTextHandler(&registro, nil))
	d := &diretorio{err: errors.New("keycloak fora")}
	s := application.NewService(e.pool, infrastructure.NewRepository(), nil).WithDiretorio(d, log)
	e.must(s.SincronizarSSO(e.ctx))
	s.AguardarSSO()
	if !strings.Contains(registro.String(), "keycloak fora") {
		t.Fatalf("a falha vai ao registro: %s", registro.String())
	}
	registro.Reset()
	d.trocar([]domain.UsuarioSSO{{Subject: "kc-" + uuid.NewString(), Ativo: true}}, nil)
	falha := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: 1}, nil).WithDiretorio(d, log)
	e.must(falha.SincronizarSSO(e.ctx))
	falha.AguardarSSO()
	if !strings.Contains(registro.String(), "sincronização com o SSO") {
		t.Fatalf("falha ao gravar: %s", registro.String())
	}
	registro.Reset()
	d.trocar(nil, application.ErrSemSSO)
	e.must(s.SincronizarSSO(e.ctx))
	s.AguardarSSO()
	if registro.Len() != 0 {
		t.Fatalf("sem SSO, silêncio: %s", registro.String())
	}
}

func TestWorkerSSO(t *testing.T) {
	e := newEnv(t)
	antes := application.IntervaloSSO
	application.IntervaloSSO = 10 * time.Millisecond
	t.Cleanup(func() { application.IntervaloSSO = antes })
	var registro bytes.Buffer
	log := slog.New(slog.NewTextHandler(&registro, nil))
	rodar := func(s *application.Service) {
		t.Helper()
		ctx, parar := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer parar()
		if err := s.RunSincronizacaoSSO(ctx); err != nil {
			t.Fatal(err)
		}
	}
	rodar(e.real())
	rodar(e.real().WithDiretorio(&diretorio{}, log))
	if !strings.Contains(registro.String(), "usuários do SSO sincronizados") {
		t.Fatalf("sucesso no registro: %s", registro.String())
	}
}
