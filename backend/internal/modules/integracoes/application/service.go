// Package application contém os casos de uso das integrações (ADR 076):
// conexão, área de preparo, tabela de-para e tarefas.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/secretcrypto"
)

// Eventos grava o outbox na transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Destino é a área de preparo de uma conexão, onde a extração grava.
type Destino interface {
	Versoes(ctx context.Context) (map[int]string, error)
	Pessoas(ctx context.Context) (map[int]bool, error)
	Gravar(ctx context.Context, itens []domain.Item) error
}

// Leitor lê a área de preparo de uma conexão, na ordem do id da origem.
type Leitor interface {
	Itens(ctx context.Context, tipo string, de, limite int) ([]domain.Item, int, error)
	Item(ctx context.Context, tipo string, id int) (domain.Item, bool, error)
}

// Progresso recebe o andamento da tarefa.
type Progresso func(feitos, total int)

// Extrator lê a origem a partir do servidor (modo direto). Fica no app,
// sobre o pacote da origem: este módulo não conhece o GLPI.
type Extrator interface {
	Extrair(ctx context.Context, c domain.Conexao, cred domain.Credenciais, destino Destino, progresso Progresso) (any, error)
}

// Carregador roda a carga sobre a área de preparo, com a tabela de-para.
type Carregador interface {
	Carregar(ctx context.Context, c domain.Conexao, tabela []domain.Linha, gravar bool, leitor Leitor, progresso Progresso) (any, error)
}

// Service implementa os casos de uso.
type Service struct {
	pool       *pgxpool.Pool
	repo       domain.Repository
	eventos    Eventos
	cipher     *secretcrypto.Cipher
	extrator   Extrator
	carregador Carregador
	logger     *slog.Logger
	agora      func() time.Time
	// Parada é quanto uma tarefa fica sem sinal de vida antes de voltar a
	// pendente; Intervalo é de quanto em quanto o worker procura tarefa.
	Parada, Intervalo time.Duration
}

// NewService cria o serviço. cipher nil: o modo direto não guarda
// credenciais.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, cipher *secretcrypto.Cipher) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos, cipher: cipher, logger: slog.Default(), agora: time.Now,
		Parada: 5 * time.Minute, Intervalo: 3 * time.Second}
}

// WithExtrator liga a extração direta.
func (s *Service) WithExtrator(e Extrator) *Service { s.extrator = e; return s }

// WithCarregador liga a simulação e a carga.
func (s *Service) WithCarregador(c Carregador) *Service { s.carregador = c; return s }

// WithLogger troca o registro.
func (s *Service) WithLogger(l *slog.Logger) *Service { s.logger = l; return s }

// MapError traduz o domínio para a API.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return apperrors.NotFound("conexão ou tarefa não encontrada")
	case errors.Is(err, domain.ErrForaDoEscopo):
		return apperrors.Forbidden("a migração é de quem administra a plataforma inteira")
	case errors.Is(err, domain.ErrTarefaAtiva):
		return apperrors.Conflict(domain.ErrTarefaAtiva.Error()[len("integracoes: "):])
	}
	var r domain.Recusa
	if errors.As(err, &r) {
		return apperrors.Validation(r.Msg)
	}
	return err
}

// soGlobal: a migração mexe em todas as unidades.
func soGlobal(identity auth.Identity) error {
	if !auth.CoverageOf(identity, auth.PermIntegracoesManage).All {
		return domain.ErrForaDoEscopo
	}
	return nil
}

func (s *Service) registrar(ctx context.Context, tx pgx.Tx, evento, tipo string, id uuid.UUID, payload map[string]any) error {
	if err := s.eventos.Write(ctx, tx, evento, tipo, id.String(), uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Entry{Action: evento, ResourceType: tipo, ResourceID: id.String(), Metadata: payload})
}

// ConexaoEntrada é o cadastro da conexão.
type ConexaoEntrada struct {
	Tipo, Nome, Modo, URL, Perfil, Fuso string
}

// ListarConexoes devolve as conexões.
func (s *Service) ListarConexoes(ctx context.Context, identity auth.Identity) ([]domain.Conexao, error) {
	if err := soGlobal(identity); err != nil {
		return nil, MapError(err)
	}
	lista, err := s.repo.ListConexoes(ctx, s.pool)
	return lista, MapError(err)
}

// ObterConexao devolve a conexão.
func (s *Service) ObterConexao(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Conexao, error) {
	if err := soGlobal(identity); err != nil {
		return domain.Conexao{}, MapError(err)
	}
	c, err := s.repo.GetConexao(ctx, s.pool, id)
	return c, MapError(err)
}

// CriarConexao grava a conexão e a auditoria.
func (s *Service) CriarConexao(ctx context.Context, identity auth.Identity, in ConexaoEntrada) (domain.Conexao, error) {
	if err := soGlobal(identity); err != nil {
		return domain.Conexao{}, MapError(err)
	}
	c, err := domain.NovaConexao(in.Tipo, in.Nome, in.Modo, in.URL, in.Perfil, in.Fuso)
	if err != nil {
		return domain.Conexao{}, MapError(err)
	}
	c.CreatedBy = identity.Username
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if c, err = s.repo.InsertConexao(ctx, tx, c); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.conexao.criada", "integracoes_conexao", c.ID, map[string]any{"nome": c.Nome, "tipo": c.Tipo, "modo": c.Modo})
	})
	return c, MapError(err)
}

// AtualizarConexao troca nome, endereço, perfil e fuso. Tipo e modo ficam.
func (s *Service) AtualizarConexao(ctx context.Context, identity auth.Identity, id uuid.UUID, in ConexaoEntrada) (domain.Conexao, error) {
	atual, err := s.ObterConexao(ctx, identity, id)
	if err != nil {
		return domain.Conexao{}, err
	}
	novo, err := domain.NovaConexao(atual.Tipo, in.Nome, atual.Modo, in.URL, in.Perfil, in.Fuso)
	if err != nil {
		return domain.Conexao{}, MapError(err)
	}
	atual.Nome, atual.URL, atual.Perfil, atual.Fuso = novo.Nome, novo.URL, novo.Perfil, novo.Fuso
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.UpdateConexao(ctx, tx, atual); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.conexao.alterada", "integracoes_conexao", id, map[string]any{"nome": atual.Nome})
	})
	if err != nil {
		return domain.Conexao{}, MapError(err)
	}
	return s.ObterConexao(ctx, identity, id)
}

// ExcluirConexao encerra a conexão: a área de preparo, a tabela e as
// tarefas vão junto (ON DELETE CASCADE). O que já foi carregado fica.
func (s *Service) ExcluirConexao(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if err := soGlobal(identity); err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteConexao(ctx, tx, id); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.conexao.excluida", "integracoes_conexao", id, map[string]any{})
	}))
}

// DefinirCredenciais grava, cifradas, as credenciais do modo direto.
func (s *Service) DefinirCredenciais(ctx context.Context, identity auth.Identity, id uuid.UUID, cred domain.Credenciais) error {
	c, err := s.ObterConexao(ctx, identity, id)
	if err != nil {
		return err
	}
	switch {
	case c.Modo != domain.ModoDireto:
		return MapError(domain.Recusa{Msg: "no modo conector, as credenciais ficam com o conector"})
	case s.cipher == nil:
		return MapError(domain.Recusa{Msg: "o servidor está sem a chave de configuração (CONFIG_ENCRYPTION_KEY)"})
	case cred.Vazia():
		return MapError(domain.Recusa{Msg: "informe o token de usuário, ou o login e a senha"})
	}
	b, _ := json.Marshal(cred)
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SetCredenciais(ctx, tx, id, s.cipher.Encrypt(string(b))); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.credenciais.definidas", "integracoes_conexao", id, map[string]any{"login": cred.Login != ""})
	}))
}

func (s *Service) credenciais(c domain.Conexao) (domain.Credenciais, error) {
	var cred domain.Credenciais
	if s.cipher == nil || c.Cifradas == "" {
		return cred, domain.Recusa{Msg: "a conexão está sem credenciais"}
	}
	texto, err := s.cipher.Decrypt(c.Cifradas)
	if err != nil {
		return cred, err
	}
	return cred, json.Unmarshal([]byte(texto), &cred)
}

// EmitirToken gera o token do conector. Ele só aparece nesta resposta;
// um novo substitui o anterior.
func (s *Service) EmitirToken(ctx context.Context, identity auth.Identity, id uuid.UUID) (string, error) {
	c, err := s.ObterConexao(ctx, identity, id)
	if err != nil {
		return "", err
	}
	if c.Modo != domain.ModoConector {
		return "", MapError(domain.Recusa{Msg: "o token é do modo conector"})
	}
	token, hash := domain.EmitirToken()
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SetToken(ctx, tx, id, &hash); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.token.emitido", "integracoes_conexao", id, map[string]any{})
	})
	return token, MapError(err)
}

// RevogarToken desliga o conector da conexão.
func (s *Service) RevogarToken(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if err := soGlobal(identity); err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SetToken(ctx, tx, id, nil); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.token.revogado", "integracoes_conexao", id, map[string]any{})
	}))
}

// Reconhecer autentica o bearer do conector. ok false: não é dele.
func (s *Service) Reconhecer(ctx context.Context, token string) (auth.Identity, bool, error) {
	if !strings.HasPrefix(token, domain.PrefixoToken) {
		return auth.Identity{}, false, nil
	}
	c, err := s.repo.PorToken(ctx, s.pool, domain.HashToken(token))
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return auth.Identity{}, true, apperrors.Unauthorized("invalid or expired access token")
	}
	if err != nil {
		return auth.Identity{}, true, err
	}
	return auth.Identity{
		Subject: c.ID.String(), Username: "conector", Name: "Conector: " + c.Nome, Source: auth.SourceAgente,
		Permissions: []string{string(auth.PermIntegracoesConector)},
	}, true, nil
}

// conexaoDoConector é a conexão do token que chamou.
func conexaoDoConector(identity auth.Identity) (uuid.UUID, error) {
	id, err := uuid.Parse(identity.Subject)
	if err != nil || identity.Source != auth.SourceAgente || !auth.HasPermission(identity, auth.PermIntegracoesConector) {
		return uuid.Nil, domain.ErrForaDoEscopo
	}
	return id, nil
}
