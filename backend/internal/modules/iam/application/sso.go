package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// DiretorioSSO lista as pessoas do SSO (ADR 077). O app liga a API de
// administração do Keycloak.
type DiretorioSSO interface {
	// Configurado diz se há client e segredo para ler o SSO.
	Configurado(ctx context.Context) bool
	// Usuarios devolve todas as pessoas; sem configuração, ErrSemSSO.
	Usuarios(ctx context.Context) ([]domain.UsuarioSSO, error)
}

// ErrSemSSO: o SSO não tem client e segredo para a sincronização.
var ErrSemSSO = errors.New("iam: SSO sem conta de serviço configurada")

// RelatorioSSO é o que a sincronização fez.
type RelatorioSSO struct {
	Lidos       int `json:"lidos"`
	Criados     int `json:"criados"`
	Atualizados int `json:"atualizados"`
	Desativados int `json:"desativados"`
	// Conflitos são as pessoas cujo usuário ou e-mail já é de uma conta
	// local: um administrador decide, como no primeiro login.
	Conflitos []string `json:"conflitos"`
}

// EstadoSSO é o que a tela mostra: se há uma sincronização rodando nesta
// réplica e a última terminada (de qualquer réplica ou do worker).
type EstadoSSO struct {
	Andamento bool                     `json:"andamento"`
	Ultima    *domain.SincronizacaoSSO `json:"ultima,omitempty"`
}

// ssoEmCurso guarda a sincronização pedida pela tela: uma por réplica.
type ssoEmCurso struct {
	ativa  atomic.Bool
	espera sync.WaitGroup
}

// TempoSSO limita a sincronização pedida pela tela.
var TempoSSO = 30 * time.Minute

// WithDiretorio liga a sincronização com o SSO.
func (s *Service) WithDiretorio(d DiretorioSSO, logger *slog.Logger) *Service {
	s.diretorio, s.logger = d, logger
	return s
}

func (s *Service) podeSincronizar(ctx context.Context) error {
	identity, _ := auth.IdentityFromContext(ctx)
	if !auth.CanGlobal(identity, auth.PermUsersManage) {
		return apperrors.Forbidden("só a administração global de contas sincroniza com o SSO")
	}
	return nil
}

// SincronizarSSO começa a sincronização em segundo plano: com milhares de
// pessoas ela leva minutos, mais do que uma requisição aguenta atrás do
// proxy. O resultado sai em EstadoDoSSO.
func (s *Service) SincronizarSSO(ctx context.Context) error {
	if err := s.podeSincronizar(ctx); err != nil {
		return err
	}
	switch {
	case s.diretorio == nil:
		return apperrors.Validation("a sincronização com o SSO não está ligada")
	case !s.diretorio.Configurado(ctx):
		return apperrors.Validation("configure o client e o segredo do SSO, com a conta de serviço habilitada, antes de sincronizar")
	case !s.sso.ativa.CompareAndSwap(false, true):
		return apperrors.Conflict("já há uma sincronização com o SSO em andamento")
	}
	s.sso.espera.Add(1)
	fundo, parar := context.WithTimeout(context.WithoutCancel(ctx), TempoSSO)
	go func() {
		defer func() { parar(); s.sso.ativa.Store(false); s.sso.espera.Done() }()
		s.registrarSSO(s.sincronizar(fundo))
	}()
	return nil
}

// AguardarSSO espera a sincronização pedida pela tela terminar.
func (s *Service) AguardarSSO() { s.sso.espera.Wait() }

// EstadoDoSSO diz se há sincronização em andamento e como foi a última.
func (s *Service) EstadoDoSSO(ctx context.Context) (EstadoSSO, error) {
	if err := s.podeSincronizar(ctx); err != nil {
		return EstadoSSO{}, err
	}
	e := EstadoSSO{Andamento: s.sso.ativa.Load()}
	ultima, ok, err := s.repo.UltimaSincronizacaoSSO(ctx, s.pool)
	if err != nil {
		return EstadoSSO{}, MapError(err)
	}
	if ok {
		e.Ultima = &ultima
	}
	return e, nil
}

// sincronizar grava cada pessoa na sua transação: o conflito de uma não
// desfaz as outras.
func (s *Service) sincronizar(ctx context.Context) (RelatorioSSO, error) {
	rel := RelatorioSSO{Conflitos: []string{}}
	pessoas, err := s.diretorio.Usuarios(ctx)
	if err != nil {
		return rel, err
	}
	for _, p := range pessoas {
		if p.Subject == "" {
			continue
		}
		rel.Lidos++
		var res domain.ResultadoSSO
		err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
			res, err = s.repo.UpsertSSO(ctx, tx, p)
			return err
		})
		switch {
		case errors.Is(err, domain.ErrConflict):
			rel.Conflitos = append(rel.Conflitos, p.Username)
			continue
		case err != nil:
			return rel, MapError(err)
		case res.Criado:
			rel.Criados++
		default:
			rel.Atualizados++
		}
		if res.Desativado {
			rel.Desativados++
		}
	}
	err = s.tx(ctx, true, func(ctx context.Context, _ pgx.Tx) (audit.Entry, error) {
		return audit.Meta(ctx, "iam.sso.sincronizado", "user", "sso", nil, rel), nil
	})
	return rel, err
}

// registrarSSO deixa no registro o fim de uma sincronização. Sem SSO
// configurado, silêncio: não é falha.
func (s *Service) registrarSSO(rel RelatorioSSO, err error) {
	switch {
	case err == nil:
		s.logger.Info("iam: usuários do SSO sincronizados", slog.Int("lidos", rel.Lidos), slog.Int("criados", rel.Criados),
			slog.Int("desativados", rel.Desativados), slog.Int("conflitos", len(rel.Conflitos)))
	case !errors.Is(err, ErrSemSSO) && !errors.Is(err, context.Canceled):
		s.logger.Warn("iam: sincronização com o SSO", slog.Any("error", err))
	}
}

// IntervaloSSO é de quanto em quanto o worker sincroniza.
var IntervaloSSO = 24 * time.Hour

// RunSincronizacaoSSO é o worker: sincroniza ao subir e depois a cada
// IntervaloSSO. Sem SSO configurado, só espera.
func (s *Service) RunSincronizacaoSSO(ctx context.Context) error {
	if s.diretorio == nil {
		<-ctx.Done()
		return nil
	}
	t := time.NewTicker(IntervaloSSO)
	defer t.Stop()
	for {
		s.registrarSSO(s.sincronizar(ctx))
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
