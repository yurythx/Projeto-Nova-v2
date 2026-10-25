// Package application guarda, lista e revela segredos. A revelação e a
// gravação nascem na mesma transação que a auditoria. O payload dessa
// auditoria não leva a senha.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/caixa"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	EventoCriado         = "cofre.segredo.criado"
	EventoAtualizado     = "cofre.segredo.atualizado"
	EventoExcluido       = "cofre.segredo.excluido"
	EventoRevelado       = "cofre.segredo.revelado"
	EventoAcessoRecusado = "cofre.acesso.recusado"
)

// Eventos grava o outbox na transação de negócio.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Caixa cifra e abre o envelope. A senha não passa pelo repositório.
type Caixa interface {
	Selar(conteudo caixa.Conteudo, aad []byte) (caixa.Selo, error)
	Abrir(selo caixa.Selo, aad []byte) (caixa.Conteudo, error)
}

// Confirmador confere a senha de quem está autenticado. Não é a senha do cofre.
type Confirmador interface {
	Confirmar(ctx context.Context, userID uuid.UUID, username string, federada bool, senha string) error
}

// Freio é o bloqueio progressivo do login. Uma senha errada aqui conta nele.
type Freio interface {
	LockedFor(ctx context.Context, subject string) (time.Duration, error)
	RegisterFailure(ctx context.Context, subject string) (time.Duration, error)
	Reset(ctx context.Context, subject string) error
}

type correr interface {
	Em(ctx context.Context, fn func(context.Context, pgx.Tx) error) error
}

type trilha interface {
	Registrar(ctx context.Context, tx pgx.Tx, evento string, id uuid.UUID, payload map[string]any) error
}

type poolCorrer struct{ pool *pgxpool.Pool }

func (p poolCorrer) Em(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return database.WithTx(ctx, p.pool, fn)
}

type outboxTrilha struct{ eventos Eventos }

func (o outboxTrilha) Registrar(ctx context.Context, tx pgx.Tx, evento string, id uuid.UUID, payload map[string]any) error {
	if err := o.eventos.Write(ctx, tx, evento, "cofre_segredo", id.String(), uuid.Nil, payload); err != nil {
		return err
	}
	entrada := audit.FromContext(ctx)
	entrada.Action = evento
	entrada.ResourceType = "cofre_segredo"
	entrada.ResourceID = id.String()
	entrada.Metadata = payload
	return audit.NewWriter(tx).Record(ctx, entrada)
}

// Service implementa os casos de uso.
type Service struct {
	pool        *pgxpool.Pool
	repo        domain.Repository
	caixa       Caixa
	correr      correr
	trilha      trilha
	confirmador Confirmador
	freio       Freio
	agora       func() time.Time
}

// NewService cria o serviço. A caixa já está presa à chave mestra.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos, cifra Caixa) *Service {
	return &Service{
		pool:   pool,
		repo:   repo,
		caixa:  cifra,
		correr: poolCorrer{pool: pool},
		trilha: outboxTrilha{eventos: eventos},
		agora:  time.Now,
	}
}

// ComConfirmacao liga a reautenticação e o bloqueio do login.
func (s *Service) ComConfirmacao(c Confirmador, f Freio) { s.confirmador, s.freio = c, f }

// MapError traduz o domínio para a API.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNaoEncontrado):
		return comCausa(apperrors.NotFound("segredo não encontrado"), err)
	case errors.Is(err, domain.ErrForaDoEscopo):
		return comCausa(apperrors.Forbidden("você não alcança o cofre desta unidade"), err)
	case errors.Is(err, domain.ErrSemPermissao):
		return comCausa(apperrors.Forbidden("seu perfil não pode fazer isso neste segredo"), err)
	case errors.Is(err, domain.ErrSenhaRecusada):
		return comCausa(apperrors.Unauthorized("senha de acesso incorreta"), err)
	case errors.Is(err, domain.ErrBloqueado):
		return comCausa(apperrors.RateLimited("muitas tentativas de senha; aguarde para revelar novamente"), err)
	case errors.Is(err, domain.ErrSemConfirmacao):
		return comCausa(apperrors.DependencyUnavailable("não foi possível confirmar a sua senha"), err)
	case errors.Is(err, caixa.ErrAberto):
		return apperrors.Internal(err)
	}
	var invalida domain.EntradaInvalida
	if errors.As(err, &invalida) {
		return comCausa(apperrors.Validation(invalida.Msg), err)
	}
	return err
}

func comCausa(e *apperrors.Error, causa error) *apperrors.Error {
	e.Err = causa
	return e
}

func pode(identity auth.Identity, permissao auth.Permission, unidade *uuid.UUID) bool {
	alvo := auth.Target{}
	if unidade != nil {
		alvo = auth.InUnidade(*unidade)
	}
	return auth.Can(identity, permissao, alvo)
}

func (s *Service) filtro(identity auth.Identity) (domain.Filtro, error) {
	perms := []auth.Permission{auth.PermCofreRead, auth.PermCofreReveal, auth.PermCofreManage, auth.PermCofreGrant}
	vistos := map[uuid.UUID]bool{}
	var ids []uuid.UUID
	for _, perm := range perms {
		cob := auth.CoverageOf(identity, perm)
		if cob.All {
			return domain.Filtro{}, nil
		}
		for _, id := range cob.Unidades {
			if !vistos[id] {
				vistos[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return domain.Filtro{}, domain.ErrForaDoEscopo
	}
	return domain.Filtro{Restrito: true, Unidades: ids}, nil
}

// pista é o único corpo que vai para o outbox e para a auditoria.
func pista(id uuid.UUID, unidade *uuid.UUID) map[string]any {
	corpo := map[string]any{"id": id.String()}
	if unidade != nil {
		corpo["unidade_id"] = unidade.String()
	}
	return corpo
}

func metadados(itens []domain.Selado) []domain.Segredo {
	out := make([]domain.Segredo, 0, len(itens))
	for _, item := range itens {
		out = append(out, item.Segredo)
	}
	return out
}

// Listar devolve só o metadado do escopo. Quem não alcança nenhuma unidade
// recebe lista vazia.
func (s *Service) Listar(ctx context.Context, identity auth.Identity) ([]domain.Segredo, error) {
	f, err := s.filtro(identity)
	if errors.Is(err, domain.ErrForaDoEscopo) {
		return []domain.Segredo{}, nil
	}
	if err != nil {
		return nil, MapError(err)
	}
	var lista []domain.Selado
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		lista, err = s.repo.List(ctx, tx, f)
		return err
	})
	if err != nil {
		return nil, MapError(err)
	}
	return metadados(lista), nil
}

// Criar grava o envelope. A resposta não traz a senha de volta.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, nome, usuario, senha, notas string, unidade *uuid.UUID) (domain.Segredo, error) {
	nome, usuario, senha, notas, err := domain.Validar(nome, usuario, senha, notas, true)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	if !pode(identity, auth.PermCofreManage, unidade) {
		return domain.Segredo{}, MapError(domain.ErrForaDoEscopo)
	}
	agora := s.agora().UTC()
	item := domain.Selado{Segredo: domain.Segredo{
		ID: uuid.New(), Nome: nome, Usuario: usuario, UnidadeID: unidade,
		Versao: 1, CreatedAt: agora, UpdatedAt: agora,
	}}
	selo, err := s.caixa.Selar(caixa.Conteudo{Senha: senha, Notas: notas}, domain.AAD(item.ID, item.Versao, item.UnidadeID))
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	item.Dek, item.Cifra = selo.Dek, selo.Cifra
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Insert(ctx, tx, item); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, EventoCriado, item.ID, pista(item.ID, item.UnidadeID))
	})
	return item.Segredo, MapError(err)
}

// Mudanca altera o metadado. Senha e notas nulas ficam como estão.
// Mover troca a unidade e exige cofre:grant na origem e no destino.
type Mudanca struct {
	Nome    string
	Usuario string
	Senha   *string
	Notas   *string
	Unidade *uuid.UUID
	Mover   bool
}

// Atualizar troca nome, senha ou unidade. A senha nova também fica só no envelope.
func (s *Service) Atualizar(ctx context.Context, identity auth.Identity, id uuid.UUID, m Mudanca) (domain.Segredo, error) {
	f, err := s.filtro(identity)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	var item domain.Selado
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			return err
		}
		nome, usuario, _, _, err := domain.Validar(m.Nome, m.Usuario, "", "", false)
		if err != nil {
			return err
		}
		unidadeMudou := m.Mover && !mesmaUnidade(atual.UnidadeID, m.Unidade)
		mexeSegredo := m.Senha != nil || m.Notas != nil
		mexeNome := nome != atual.Nome || usuario != atual.Usuario
		if !mexeNome && !mexeSegredo && !unidadeMudou {
			item = atual
			return nil
		}
		if (mexeNome || mexeSegredo) && !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrSemPermissao
		}
		destino := atual.UnidadeID
		if unidadeMudou {
			if !pode(identity, auth.PermCofreGrant, atual.UnidadeID) || !pode(identity, auth.PermCofreGrant, m.Unidade) {
				return domain.ErrSemPermissao
			}
			destino = m.Unidade
		}
		senha, notas := "", ""
		if mexeSegredo || unidadeMudou {
			conteudo, err := s.caixa.Abrir(caixa.Selo{Dek: atual.Dek, Cifra: atual.Cifra}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
			if err != nil {
				return err
			}
			senha, notas = conteudo.Senha, conteudo.Notas
			if m.Senha != nil {
				senha = *m.Senha
			}
			if m.Notas != nil {
				notas = *m.Notas
			}
			if _, _, senha, notas, err = domain.Validar(nome, usuario, senha, notas, true); err != nil {
				return err
			}
		}
		atual.Nome = nome
		atual.Usuario = usuario
		atual.UnidadeID = destino
		atual.UpdatedAt = s.agora().UTC()
		if mexeSegredo || unidadeMudou {
			atual.Versao++
			selo, err := s.caixa.Selar(caixa.Conteudo{Senha: senha, Notas: notas}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
			if err != nil {
				return err
			}
			atual.Dek, atual.Cifra = selo.Dek, selo.Cifra
		}
		if err := s.repo.Update(ctx, tx, atual, f); err != nil {
			return err
		}
		item = atual
		return s.trilha.Registrar(ctx, tx, EventoAtualizado, atual.ID, pista(atual.ID, atual.UnidadeID))
	})
	return item.Segredo, MapError(err)
}

// Excluir remove o segredo do escopo de quem gerencia.
func (s *Service) Excluir(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	f, err := s.filtro(identity)
	if err != nil {
		return MapError(err)
	}
	return MapError(s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			return err
		}
		if !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrSemPermissao
		}
		if err := s.repo.Delete(ctx, tx, id, f); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, EventoExcluido, id, pista(id, atual.UnidadeID))
	}))
}

// Nome devolve o metadado de um segredo do alcance. Nunca a senha.
func (s *Service) Nome(ctx context.Context, identity auth.Identity, id uuid.UUID) (string, error) {
	f, err := s.filtro(identity)
	if err != nil {
		return "", MapError(err)
	}
	var nome string
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		item, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			return err
		}
		nome = item.Nome
		return nil
	})
	return nome, MapError(err)
}

// Revelar devolve a senha depois de confirmar a senha de acesso de quem pediu.
// Quem só lê o cofre não chega a essa confirmação.
func (s *Service) Revelar(ctx context.Context, identity auth.Identity, id uuid.UUID, senhaAcesso string) (domain.Revelado, error) {
	if identity.Source == auth.SourceAgente {
		return domain.Revelado{}, MapError(domain.ErrSemPermissao)
	}
	f, err := s.filtro(identity)
	if err != nil {
		return domain.Revelado{}, MapError(err)
	}
	var out domain.Revelado
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			return err
		}
		if !pode(identity, auth.PermCofreReveal, atual.UnidadeID) && !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrSemPermissao
		}
		if err := s.confirmarAcesso(ctx, identity, atual.ID, atual.UnidadeID, senhaAcesso); err != nil {
			return err
		}
		conteudo, err := s.caixa.Abrir(caixa.Selo{Dek: atual.Dek, Cifra: atual.Cifra}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
		if err != nil {
			return err
		}
		out = domain.Revelado{Segredo: atual.Segredo, Senha: conteudo.Senha, Notas: conteudo.Notas}
		return s.trilha.Registrar(ctx, tx, EventoRevelado, atual.ID, pista(atual.ID, atual.UnidadeID))
	})
	return out, MapError(err)
}

func (s *Service) confirmarAcesso(ctx context.Context, identity auth.Identity, segredo uuid.UUID, unidade *uuid.UUID, senha string) error {
	if strings.TrimSpace(senha) == "" {
		return domain.EntradaInvalida{"informe a sua senha de acesso"}
	}
	if s.confirmador == nil {
		return domain.ErrSemConfirmacao
	}
	chave := "user:" + strings.ToLower(identity.Username)
	if s.freio != nil {
		if left, err := s.freio.LockedFor(ctx, chave); err == nil && left > 0 {
			return domain.ErrBloqueado
		}
	}
	if err := s.confirmador.Confirmar(ctx, identity.UserID, identity.Username, identity.Source == auth.SourceKeycloak, senha); err != nil {
		if errors.Is(err, domain.ErrSenhaRecusada) {
			if s.freio != nil {
				_, _ = s.freio.RegisterFailure(ctx, chave)
			}
			s.auditarFalha(ctx, segredo, unidade)
		}
		return err
	}
	if s.freio != nil {
		_ = s.freio.Reset(ctx, chave)
	}
	return nil
}

func (s *Service) auditarFalha(ctx context.Context, id uuid.UUID, unidade *uuid.UUID) {
	payload := pista(id, unidade)
	if s.pool != nil {
		entrada := audit.FromContext(ctx)
		entrada.Action = EventoAcessoRecusado
		entrada.ResourceType = "cofre_segredo"
		entrada.ResourceID = id.String()
		entrada.Metadata = payload
		_ = audit.NewWriter(s.pool).Record(ctx, entrada)
		return
	}
	_ = s.trilha.Registrar(ctx, nil, EventoAcessoRecusado, id, payload)
}

func mesmaUnidade(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
