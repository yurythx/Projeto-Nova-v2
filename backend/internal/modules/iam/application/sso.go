package application

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
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
	// Usuarios entrega as pessoas página a página, a lote; sem
	// configuração, ErrSemSSO. Um erro de lote para a leitura.
	Usuarios(ctx context.Context, lote func([]domain.UsuarioSSO) error) error
}

// ErrSemSSO: o SSO não tem client e segredo para a sincronização.
var ErrSemSSO = errors.New("iam: SSO sem conta de serviço configurada")

// RelatorioSSO é o que a sincronização fez até agora.
type RelatorioSSO struct {
	Lidos       int `json:"lidos"`
	Criados     int `json:"criados"`
	Atualizados int `json:"atualizados"`
	Desativados int `json:"desativados"`
	// Reativados são os que o SSO tinha desativado e deu de novo como ativos.
	Reativados int `json:"reativados"`
	// Conflitos são as pessoas cujo usuário ou e-mail já é de uma conta
	// local: um administrador decide, como no primeiro login.
	Conflitos []string `json:"conflitos"`
}

// EstadoSSO é o que a tela mostra: a sincronização mais recente, de
// qualquer processo (a tela ou o worker), e se ela ainda está viva.
type EstadoSSO struct {
	Andamento bool                     `json:"andamento"`
	Ultima    *domain.SincronizacaoSSO `json:"ultima,omitempty"`
}

// ssoEmCurso deixa os testes esperarem a sincronização pedida pela tela.
type ssoEmCurso struct {
	espera sync.WaitGroup
}

// TempoSSO limita a sincronização pedida pela tela. A primeira, com o cache
// do Keycloak frio, pode levar horas num realm federado ao AD.
var TempoSSO = 6 * time.Hour

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
// pessoas ela leva minutos ou horas, mais do que uma requisição aguenta
// atrás do proxy. O andamento sai em EstadoDoSSO.
func (s *Service) SincronizarSSO(ctx context.Context) error {
	if err := s.podeSincronizar(ctx); err != nil {
		return err
	}
	switch {
	case s.diretorio == nil:
		return apperrors.Validation("a sincronização com o SSO não está ligada")
	case !s.diretorio.Configurado(ctx):
		return apperrors.Validation("configure o client e o segredo do SSO, com a conta de serviço habilitada, antes de sincronizar")
	}
	id, err := s.repo.IniciarSincronizacaoSSO(ctx, s.pool)
	if errors.Is(err, domain.ErrConflict) {
		return apperrors.Conflict("já há uma sincronização com o SSO em andamento")
	}
	if err != nil {
		return MapError(err)
	}
	s.sso.espera.Add(1)
	fundo, parar := context.WithTimeout(context.WithoutCancel(ctx), TempoSSO)
	go func() {
		defer func() { parar(); s.sso.espera.Done() }()
		s.registrarSSO(s.sincronizar(fundo, id))
	}()
	return nil
}

// AguardarSSO espera a sincronização pedida pela tela terminar.
func (s *Service) AguardarSSO() { s.sso.espera.Wait() }

// EstadoDoSSO traz a sincronização mais recente e se ela está viva: sem
// fim e com batida há menos de SemBatidaSSO.
func (s *Service) EstadoDoSSO(ctx context.Context) (EstadoSSO, error) {
	if err := s.podeSincronizar(ctx); err != nil {
		return EstadoSSO{}, err
	}
	ultima, ok, err := s.repo.UltimaSincronizacaoSSO(ctx, s.pool)
	if err != nil {
		return EstadoSSO{}, MapError(err)
	}
	if !ok {
		return EstadoSSO{}, nil
	}
	viva := ultima.TerminadaEm == nil && time.Since(ultima.BatidaEm) < domain.SemBatidaSSO
	return EstadoSSO{Andamento: viva, Ultima: &ultima}, nil
}

// sincronizar grava cada página assim que chega, cada pessoa na sua
// transação (o conflito de uma não desfaz as outras), e regrava o
// andamento. O fim é gravado mesmo com o contexto cancelado.
func (s *Service) sincronizar(ctx context.Context, id uuid.UUID) (RelatorioSSO, error) {
	rel := RelatorioSSO{Conflitos: []string{}}
	err := s.diretorio.Usuarios(ctx, func(pessoas []domain.UsuarioSSO) error {
		for _, p := range pessoas {
			if err := s.gravarPessoa(ctx, p, &rel); err != nil {
				return err
			}
		}
		if err := s.marcarSSO(ctx, id, rel, false, nil); err != nil {
			return err
		}
		s.logger.Info("iam: sincronizando com o SSO", slog.Int("lidos", rel.Lidos), slog.Int("criados", rel.Criados))
		return nil
	})
	if err == nil {
		err = s.tx(ctx, true, func(ctx context.Context, _ pgx.Tx) (audit.Entry, error) {
			return audit.Meta(ctx, "iam.sso.sincronizado", "user", "sso", nil, rel), nil
		})
	}
	fim, parar := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer parar()
	if errFim := s.marcarSSO(fim, id, rel, true, err); err == nil {
		err = errFim
	}
	return rel, err
}

// gravarPessoa grava uma pessoa e conta no relatório.
func (s *Service) gravarPessoa(ctx context.Context, p domain.UsuarioSSO, rel *RelatorioSSO) error {
	if p.Subject == "" {
		return nil
	}
	rel.Lidos++
	var res domain.ResultadoSSO
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		res, err = s.repo.UpsertSSO(ctx, tx, p)
		return err
	})
	switch {
	case errors.Is(err, domain.ErrConflict):
		rel.Conflitos = append(rel.Conflitos, p.Username)
		return nil
	case err != nil:
		return MapError(err)
	case res.Criado:
		rel.Criados++
	default:
		rel.Atualizados++
	}
	switch {
	case res.Desativado:
		rel.Desativados++
	case res.Reativado:
		rel.Reativados++
	}
	return nil
}

// marcarSSO regrava o andamento; terminada, com o erro (se houve).
func (s *Service) marcarSSO(ctx context.Context, id uuid.UUID, rel RelatorioSSO, terminada bool, falha error) error {
	corpo, _ := json.Marshal(rel)
	erro := ""
	if falha != nil {
		erro = falha.Error()
	}
	return MapError(s.repo.AtualizarSincronizacaoSSO(ctx, s.pool, id, corpo, terminada, erro))
}

// registrarSSO deixa no registro o fim de uma sincronização. Sem SSO
// configurado, silêncio: não é falha.
func (s *Service) registrarSSO(rel RelatorioSSO, err error) {
	switch {
	case err == nil:
		s.logger.Info("iam: usuários do SSO sincronizados", slog.Int("lidos", rel.Lidos), slog.Int("criados", rel.Criados),
			slog.Int("desativados", rel.Desativados), slog.Int("reativados", rel.Reativados), slog.Int("conflitos", len(rel.Conflitos)))
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
		s.rodarSSO(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// rodarSSO é uma rodada do worker: sem configuração, nada; com outra
// sincronização viva (a da tela), espera a próxima.
func (s *Service) rodarSSO(ctx context.Context) {
	if !s.diretorio.Configurado(ctx) {
		return
	}
	id, err := s.repo.IniciarSincronizacaoSSO(ctx, s.pool)
	if err != nil {
		if !errors.Is(err, domain.ErrConflict) && ctx.Err() == nil {
			s.logger.Warn("iam: sincronização com o SSO", slog.Any("error", err))
		}
		return
	}
	s.registrarSSO(s.sincronizar(ctx, id))
}
