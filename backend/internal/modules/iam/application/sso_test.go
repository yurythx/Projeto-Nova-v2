package application_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// diretorio é o SSO do teste.
type diretorio struct {
	pessoas []domain.UsuarioSSO
	err     error
}

func (d diretorio) Usuarios(context.Context) ([]domain.UsuarioSSO, error) { return d.pessoas, d.err }

func TestSincronizarSSO(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx
	sfx := uuid.NewString()[:8]
	ana := domain.UsuarioSSO{Subject: "kc-ana-" + sfx, Username: "ana." + sfx, Email: "ana." + sfx + "@sso.test", Nome: "Ana Souza", Ativo: true,
		Grupos: []string{"Administrators"}}
	semEmail := domain.UsuarioSSO{Subject: "kc-sem-" + sfx, Ativo: true}
	// Uma conta local com o mesmo usuário: nunca se vincula sozinha.
	local := e.real()
	conta, err := local.CreateLocalUser(ctx, application.CreateLocalUserInput{Username: "local." + sfx, Email: "local." + sfx + "@x.test",
		DisplayName: "Local", Password: "Senha-Forte-123!"})
	e.must(err)
	conflito := domain.UsuarioSSO{Subject: "kc-local-" + sfx, Username: conta.Username, Email: "outro." + sfx + "@x.test", Ativo: true}
	d := &diretorio{pessoas: []domain.UsuarioSSO{ana, semEmail, conflito, {Subject: ""}}}
	invalidou := 0
	s := application.NewService(e.pool, infrastructure.NewRepository(), func(context.Context) { invalidou++ }).
		WithDiretorio(d, slog.New(slog.DiscardHandler))

	rel, err := s.SincronizarSSO(ctx)
	if err != nil || rel.Lidos != 3 || rel.Criados != 2 || len(rel.Conflitos) != 1 || rel.Conflitos[0] != conta.Username || invalidou != 1 {
		t.Fatalf("primeira: %+v %v", rel, err)
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
	d.pessoas = []domain.UsuarioSSO{ana}
	if rel, err := s.SincronizarSSO(ctx); err != nil || rel.Atualizados != 1 || rel.Desativados != 1 {
		t.Fatalf("desativar: %+v %v", rel, err)
	}
	ana.Ativo = true
	d.pessoas = []domain.UsuarioSSO{ana}
	if rel, err := s.SincronizarSSO(ctx); err != nil || rel.Desativados != 0 {
		t.Fatalf("reativar: %+v %v", rel, err)
	}
	e.must(e.pool.QueryRow(ctx, `SELECT display_name, active FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&nome, &ativo))
	if ativo || nome != "Ana Souza" {
		t.Fatalf("continua desativada e com o nome: %s %v", nome, ativo)
	}

	// Permissão, ligação e falhas.
	semPermissao := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New()})
	if _, err := s.SincronizarSSO(semPermissao); status(err) != http.StatusForbidden {
		t.Fatalf("sem users:manage global: %v", err)
	}
	if _, err := e.real().SincronizarSSO(ctx); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sem diretório: %v", err)
	}
	d.err = application.ErrSemSSO
	if _, err := s.SincronizarSSO(ctx); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("SSO sem conta de serviço: %v", err)
	}
	d.err = errors.New("keycloak fora")
	if _, err := s.SincronizarSSO(ctx); err == nil || !strings.Contains(err.Error(), "keycloak fora") {
		t.Fatalf("SSO fora: %v", err)
	}
	d.err = nil
	falha := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: 1}, nil).WithDiretorio(d, slog.New(slog.DiscardHandler))
	if _, err := falha.SincronizarSSO(ctx); err == nil {
		t.Fatal("falha ao gravar")
	}
	envenenado := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), poisonAt: 1, noTx: true}, nil).
		WithDiretorio(diretorio{}, slog.New(slog.DiscardHandler))
	if _, err := envenenado.SincronizarSSO(auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), Permissions: []string{"*"},
		Scopes: []auth.Scope{{Permissions: []string{"*"}}}})); err != nil {
		t.Fatalf("sem ninguém para gravar, só a auditoria: %v", err)
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
	rodar(e.real().WithDiretorio(diretorio{}, log))
	if !strings.Contains(registro.String(), "usuários do SSO sincronizados") {
		t.Fatalf("sucesso no registro: %s", registro.String())
	}
	registro.Reset()
	rodar(e.real().WithDiretorio(diretorio{err: application.ErrSemSSO}, log))
	if registro.Len() != 0 {
		t.Fatalf("sem SSO configurado, silêncio: %s", registro.String())
	}
	rodar(e.real().WithDiretorio(diretorio{err: errors.New("keycloak fora")}, log))
	if !strings.Contains(registro.String(), "keycloak fora") {
		t.Fatalf("falha no registro: %s", registro.String())
	}
}
