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
// A fonte confirma o que a lista diz, a menos que fonte diga outra coisa
// (ou falhe, com errFonte).
type diretorio struct {
	mu       sync.Mutex
	pessoas  []domain.UsuarioSSO
	err      error
	semSSO   bool
	segurar  chan struct{}
	fonte    map[string]bool
	errFonte error
}

func (d *diretorio) Ativo(_ context.Context, subject string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.errFonte != nil {
		return false, d.errFonte
	}
	if v, ok := d.fonte[subject]; ok {
		return v, nil
	}
	for _, p := range d.pessoas {
		if p.Subject == subject {
			return p.Ativo, nil
		}
	}
	return false, nil
}

func (d *diretorio) contradizer(fonte map[string]bool, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fonte, d.errFonte = fonte, err
}

func (d *diretorio) Configurado(context.Context) bool { return !d.semSSO }

// Usuarios entrega uma pessoa por página.
func (d *diretorio) Usuarios(_ context.Context, lote func([]domain.UsuarioSSO) error) error {
	if d.segurar != nil {
		<-d.segurar
	}
	d.mu.Lock()
	pessoas, err := d.pessoas, d.err
	d.mu.Unlock()
	if err != nil {
		return err
	}
	for _, p := range pessoas {
		if err := lote([]domain.UsuarioSSO{p}); err != nil {
			return err
		}
	}
	return nil
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
	if err != nil || e.Andamento || e.Ultima == nil || e.Ultima.TerminadaEm == nil || e.Ultima.Erro != "" {
		t.Fatalf("estado: %+v %v", e, err)
	}
	var rel application.RelatorioSSO
	if err := json.Unmarshal(e.Ultima.Relatorio, &rel); err != nil {
		t.Fatal(err)
	}
	return rel
}

// limparSSO apaga as sincronizações: cada teste começa sem nenhuma.
func limparSSO(t *testing.T, e *env) {
	t.Helper()
	_, err := e.pool.Exec(context.Background(), `DELETE FROM iam_sso_sincronizacoes`)
	e.must(err)
}

func TestSincronizarSSO(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx
	limparSSO(t, e)
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

	if est, err := s.EstadoDoSSO(ctx); err != nil || est.Andamento || est.Ultima != nil {
		t.Fatalf("nunca sincronizou: %+v %v", est, err)
	}
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

	// Desativada no SSO: desativa aqui. Reativada lá: volta, porque foi o
	// SSO quem desativou (o Keycloak federado às vezes informa estado velho).
	ana.Ativo, ana.Nome = false, ""
	d.trocar([]domain.UsuarioSSO{ana}, nil)
	if rel := sincronizar(t, s, ctx); rel.Atualizados != 1 || rel.Desativados != 1 {
		t.Fatalf("desativar: %+v", rel)
	}
	if rel := sincronizar(t, s, ctx); rel.Desativados != 0 || rel.Reativados != 0 {
		t.Fatalf("desativada de novo não conta: %+v", rel)
	}
	ana.Ativo = true
	d.trocar([]domain.UsuarioSSO{ana}, nil)
	if rel := sincronizar(t, s, ctx); rel.Desativados != 0 || rel.Reativados != 1 {
		t.Fatalf("reativar: %+v", rel)
	}
	e.must(e.pool.QueryRow(ctx, `SELECT display_name, active FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&nome, &ativo))
	if !ativo || nome != "Ana Souza" {
		t.Fatalf("reativada e com o nome: %s %v", nome, ativo)
	}

	// Desativada por um administrador: a sincronização não a reativa, mesmo
	// que o SSO a desative e reative.
	var anaID uuid.UUID
	e.must(e.pool.QueryRow(ctx, `SELECT id FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&anaID))
	e.must(infrastructure.NewRepository().UpdateUser(ctx, e.pool, anaID, "Ana Souza", false, []string{}))
	for _, ativa := range []bool{false, true} {
		ana.Ativo = ativa
		d.trocar([]domain.UsuarioSSO{ana}, nil)
		if rel := sincronizar(t, s, ctx); rel.Reativados != 0 || rel.Desativados != 0 {
			t.Fatalf("a decisão do administrador vale (%v): %+v", ativa, rel)
		}
	}
	e.must(e.pool.QueryRow(ctx, `SELECT active FROM users WHERE keycloak_subject = $1`, ana.Subject).Scan(&ativo))
	if ativo {
		t.Fatal("o administrador a desativou")
	}

	// A lista que oscila: diz desativada, a fonte diz ativa. Não desativa.
	firme := domain.UsuarioSSO{Subject: "kc-firme-" + sfx, Username: "firme." + sfx, Ativo: true}
	d.trocar([]domain.UsuarioSSO{firme}, nil)
	sincronizar(t, s, ctx)
	firme.Ativo = false
	d.trocar([]domain.UsuarioSSO{firme}, nil)
	d.contradizer(map[string]bool{firme.Subject: true}, nil)
	if rel := sincronizar(t, s, ctx); rel.Desativados != 0 {
		t.Fatalf("a fonte manda: %+v", rel)
	}
	// Sem conseguir confirmar, o estado fica como está.
	d.contradizer(nil, errors.New("keycloak fora"))
	if rel := sincronizar(t, s, ctx); rel.Desativados != 0 {
		t.Fatalf("sem confirmação, não muda: %+v", rel)
	}
	// Confirmada na fonte, desativa; e a reativação também se confirma.
	d.contradizer(nil, nil)
	if rel := sincronizar(t, s, ctx); rel.Desativados != 1 {
		t.Fatalf("confirmada, desativa: %+v", rel)
	}
	firme.Ativo = true
	d.trocar([]domain.UsuarioSSO{firme}, nil)
	d.contradizer(map[string]bool{firme.Subject: false}, nil)
	if rel := sincronizar(t, s, ctx); rel.Reativados != 0 {
		t.Fatalf("reativação que a fonte nega: %+v", rel)
	}
	d.contradizer(nil, nil)
	// Conta nova que a lista diz desativada: a fonte decide; sem ela, entra
	// ativa, como no primeiro login.
	nova := domain.UsuarioSSO{Subject: "kc-nova-" + sfx, Username: "nova." + sfx, Ativo: false}
	semFonte := domain.UsuarioSSO{Subject: "kc-semfonte-" + sfx, Username: "semfonte." + sfx, Ativo: false}
	d.trocar([]domain.UsuarioSSO{nova}, nil)
	d.contradizer(map[string]bool{nova.Subject: true}, nil)
	sincronizar(t, s, ctx)
	d.trocar([]domain.UsuarioSSO{semFonte}, nil)
	d.contradizer(nil, errors.New("keycloak fora"))
	sincronizar(t, s, ctx)
	d.contradizer(nil, nil)
	for _, sub := range []string{nova.Subject, semFonte.Subject} {
		e.must(e.pool.QueryRow(ctx, `SELECT active FROM users WHERE keycloak_subject = $1`, sub).Scan(&ativo))
		if !ativo {
			t.Fatalf("%s entra ativa", sub)
		}
	}

	// Quem nasce desativado pelo SSO volta quando ele o ativa.
	novo := domain.UsuarioSSO{Subject: "kc-novo-" + sfx, Username: "novo." + sfx, Ativo: false}
	d.trocar([]domain.UsuarioSSO{novo}, nil)
	if rel := sincronizar(t, s, ctx); rel.Criados != 1 {
		t.Fatalf("criado desativado: %+v", rel)
	}
	novo.Ativo = true
	d.trocar([]domain.UsuarioSSO{novo}, nil)
	if rel := sincronizar(t, s, ctx); rel.Reativados != 1 {
		t.Fatalf("nasceu desativado e voltou: %+v", rel)
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
	if err := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: 1}, nil).
		WithDiretorio(d, slog.New(slog.DiscardHandler)).SincronizarSSO(ctx); err == nil || status(err) == http.StatusConflict {
		t.Fatalf("abrir com o banco fora: %v", err)
	}
}

func TestSincronizacaoComFalhas(t *testing.T) {
	e := newEnv(t)
	limparSSO(t, e)
	var registro bytes.Buffer
	log := slog.New(slog.NewTextHandler(&registro, nil))
	d := &diretorio{err: errors.New("keycloak fora")}
	s := application.NewService(e.pool, infrastructure.NewRepository(), nil).WithDiretorio(d, log)
	e.must(s.SincronizarSSO(e.ctx))
	s.AguardarSSO()
	if !strings.Contains(registro.String(), "keycloak fora") {
		t.Fatalf("a falha vai ao registro: %s", registro.String())
	}
	if est, err := s.EstadoDoSSO(e.ctx); err != nil || est.Andamento || est.Ultima.Erro != "keycloak fora" || est.Ultima.TerminadaEm == nil {
		t.Fatalf("a falha fica na sincronização: %+v %v", est, err)
	}
	// Falha ao ler a situação (2ª chamada), gravar a pessoa (3ª), o
	// andamento (4ª) e o fim (5ª).
	d.trocar([]domain.UsuarioSSO{{Subject: "kc-" + uuid.NewString(), Ativo: true}}, nil)
	for _, em := range []int{2, 3, 4, 5} {
		registro.Reset()
		falha := application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: em}, nil).WithDiretorio(d, log)
		e.must(falha.SincronizarSSO(e.ctx))
		falha.AguardarSSO()
		if !strings.Contains(registro.String(), "sincronização com o SSO") {
			t.Fatalf("falha na chamada %d: %s", em, registro.String())
		}
		limparSSO(t, e)
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
		ctx, parar := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer parar()
		if err := s.RunSincronizacaoSSO(ctx); err != nil {
			t.Fatal(err)
		}
	}
	limparSSO(t, e)
	rodar(e.real())
	rodar(e.real().WithDiretorio(&diretorio{semSSO: true}, log))
	if registro.Len() != 0 {
		t.Fatalf("sem configuração, nada: %s", registro.String())
	}
	// Com outra viva (a da tela), espera a próxima rodada.
	_, err := e.pool.Exec(context.Background(), `INSERT INTO iam_sso_sincronizacoes DEFAULT VALUES`)
	e.must(err)
	rodar(e.real().WithDiretorio(&diretorio{}, log))
	if registro.Len() != 0 {
		t.Fatalf("com outra viva, nada: %s", registro.String())
	}
	limparSSO(t, e)
	rodar(application.NewService(e.pool, &faultRepo{inner: infrastructure.NewRepository(), failAt: 1}, nil).WithDiretorio(&diretorio{}, log))
	if !strings.Contains(registro.String(), "sincronização com o SSO") {
		t.Fatalf("abrir falhou: %s", registro.String())
	}
	// Uma rodada cortada no meio da abertura deixa a linha viva: limpa.
	limparSSO(t, e)
	registro.Reset()
	rodar(e.real().WithDiretorio(&diretorio{pessoas: []domain.UsuarioSSO{{Subject: "kc-" + uuid.NewString(), Ativo: true}}}, log))
	if !strings.Contains(registro.String(), "usuários do SSO sincronizados") || !strings.Contains(registro.String(), "sincronizando com o SSO") {
		t.Fatalf("andamento e sucesso no registro: %s", registro.String())
	}
}
