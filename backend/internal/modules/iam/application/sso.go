package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// DiretorioSSO lista as pessoas do SSO (ADR 077). O app liga a API de
// administração do Keycloak; sem SSO configurado, devolve ErrSemSSO.
type DiretorioSSO interface {
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

// WithDiretorio liga a sincronização com o SSO.
func (s *Service) WithDiretorio(d DiretorioSSO, logger *slog.Logger) *Service {
	s.diretorio, s.logger = d, logger
	return s
}

// SincronizarSSO traz as pessoas do SSO agora. Só a administração global
// de contas: a sincronização mexe em todas.
func (s *Service) SincronizarSSO(ctx context.Context) (RelatorioSSO, error) {
	identity, _ := auth.IdentityFromContext(ctx)
	if !auth.CanGlobal(identity, auth.PermUsersManage) {
		return RelatorioSSO{}, apperrors.Forbidden("só a administração global de contas sincroniza com o SSO")
	}
	if s.diretorio == nil {
		return RelatorioSSO{}, apperrors.Validation("a sincronização com o SSO não está ligada")
	}
	rel, err := s.sincronizar(ctx)
	if errors.Is(err, ErrSemSSO) {
		return rel, apperrors.Validation("configure o client e o segredo do SSO, com a conta de serviço habilitada, antes de sincronizar")
	}
	return rel, err
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
		rel, err := s.sincronizar(ctx)
		switch {
		case err == nil:
			s.logger.Info("iam: usuários do SSO sincronizados", slog.Int("lidos", rel.Lidos), slog.Int("criados", rel.Criados),
				slog.Int("desativados", rel.Desativados), slog.Int("conflitos", len(rel.Conflitos)))
		case !errors.Is(err, ErrSemSSO) && ctx.Err() == nil:
			s.logger.Warn("iam: sincronização com o SSO", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
