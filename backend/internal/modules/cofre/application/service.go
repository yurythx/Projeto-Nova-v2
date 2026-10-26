// Package application guarda, lista e revela segredos. A revelação e a
// gravação nascem na mesma transação que a auditoria. O payload dessa
// auditoria não leva a senha.
package application

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

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
	EventoCriado          = "cofre.segredo.criado"
	EventoAtualizado      = "cofre.segredo.atualizado"
	EventoExcluido        = "cofre.segredo.excluido"
	EventoRevelado        = "cofre.segredo.revelado"
	EventoRetirado        = "cofre.segredo.retirado"
	EventoDevolvido       = "cofre.segredo.devolvido"
	EventoRetiradaVencida = "cofre.segredo.retirada_vencida"
	EventoRotacionado     = "cofre.segredo.rotacionado"
	EventoRotacaoAplicada = "cofre.segredo.rotacao_aplicada"
	EventoAcessoRecusado  = "cofre.acesso.recusado"
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
	gerar       func() (string, error)
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
		gerar:  gerarSenha,
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
	case errors.Is(err, domain.ErrOcupado):
		return comCausa(apperrors.Conflict("este segredo está retirado"), err)
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
			f := domain.Filtro{FiltrarPastas: true, Usuario: identity.UserID, GereTudo: auth.CoverageOf(identity, auth.PermCofreManage).All}
			if !f.GereTudo {
				f.GereUnidades = auth.CoverageOf(identity, auth.PermCofreManage).Unidades
			}
			return f, nil
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
	manage := auth.CoverageOf(identity, auth.PermCofreManage)
	return domain.Filtro{
		Restrito: true, Unidades: ids, FiltrarPastas: true, Usuario: identity.UserID,
		GereTudo: manage.All, GereUnidades: manage.Unidades,
	}, nil
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
		if err != nil {
			return err
		}
		if err := s.anexar(ctx, tx, f, lista); err != nil {
			return err
		}
		return s.anexarRetiradas(ctx, tx, f, lista)
	})
	if err != nil {
		return nil, MapError(err)
	}
	return metadados(lista), nil
}

// Entrada é o cadastro de um segredo. Administra só vale para um painel.
type Entrada struct {
	Nome       string
	Usuario    string
	Senha      string
	Notas      string
	Tipo       string
	Endereco   string
	Unidade    *uuid.UUID
	Administra []uuid.UUID
	Rotaciona  bool
	Pasta      *uuid.UUID
}

// Criar grava o envelope. A resposta não traz a senha de volta.
func (s *Service) Criar(ctx context.Context, identity auth.Identity, e Entrada) (domain.Segredo, error) {
	nome, usuario, senha, notas, err := domain.Validar(e.Nome, e.Usuario, e.Senha, e.Notas, true)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	tipo, err := domain.TipoDe(e.Tipo)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	endereco, err := domain.EnderecoDe(e.Endereco)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	if tipo != domain.TipoPainel && len(e.Administra) > 0 {
		return domain.Segredo{}, MapError(domain.EntradaInvalida{Msg: "só um painel administra máquinas"})
	}
	if !pode(identity, auth.PermCofreManage, e.Unidade) {
		return domain.Segredo{}, MapError(domain.ErrForaDoEscopo)
	}
	f, err := s.filtro(identity)
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	agora := s.agora().UTC()
	item := domain.Selado{Segredo: domain.Segredo{
		ID: uuid.New(), Nome: nome, Usuario: usuario, Tipo: tipo, Endereco: endereco, UnidadeID: e.Unidade,
		Rotaciona: e.Rotaciona, Versao: 1, CreatedAt: agora, UpdatedAt: agora,
	}}
	if item.Rotaciona {
		proxima := agora.Add(domain.IntervaloRotacao)
		item.ProximaRotacao = &proxima
	}
	selo, err := s.caixa.Selar(caixa.Conteudo{Senha: senha, Notas: notas}, domain.AAD(item.ID, item.Versao, item.UnidadeID))
	if err != nil {
		return domain.Segredo{}, MapError(err)
	}
	item.Dek, item.Cifra = selo.Dek, selo.Cifra
	err = s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if e.Pasta != nil {
			if err := s.conferirPasta(ctx, tx, *e.Pasta, item.UnidadeID); err != nil {
				return err
			}
			item.PastaID = e.Pasta
		}
		if err := s.repo.Insert(ctx, tx, item); err != nil {
			return err
		}
		ligadas, err := s.aplicarAdministracao(ctx, tx, f, item.ID, tipo, e.Administra)
		if err != nil {
			return err
		}
		item.Administra = ligadas
		return s.trilha.Registrar(ctx, tx, EventoCriado, item.ID, pista(item.ID, item.UnidadeID))
	})
	return item.Segredo, MapError(err)
}

// Mudanca altera o metadado. Senha e notas nulas ficam como estão.
// Mover troca a unidade e exige cofre:grant na origem e no destino.
type Mudanca struct {
	Nome       string
	Usuario    string
	Senha      *string
	Notas      *string
	Tipo       *string
	Endereco   *string
	Administra *[]uuid.UUID
	Unidade    *uuid.UUID
	Mover      bool
	Rotaciona  *bool
	MoverPasta bool
	Pasta      *uuid.UUID
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
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		um := []domain.Selado{atual}
		if err := s.anexar(ctx, tx, f, um); err != nil {
			return err
		}
		atual = um[0]
		nome, usuario, _, _, err := domain.Validar(m.Nome, m.Usuario, "", "", false)
		if err != nil {
			return err
		}
		tipo := atual.Tipo
		if m.Tipo != nil {
			tipo, err = domain.TipoDe(*m.Tipo)
			if err != nil {
				return err
			}
		}
		endereco := atual.Endereco
		if m.Endereco != nil {
			endereco, err = domain.EnderecoDe(*m.Endereco)
			if err != nil {
				return err
			}
		}
		administra := atual.Administra
		if tipo != domain.TipoPainel {
			administra = nil
		} else if m.Administra != nil {
			administra = *m.Administra
		}
		rotaciona := atual.Rotaciona
		if m.Rotaciona != nil {
			rotaciona = *m.Rotaciona
		}
		unidadeMudou := m.Mover && !mesmaUnidade(atual.UnidadeID, m.Unidade)
		mexeSegredo := m.Senha != nil || m.Notas != nil
		mexeNome := nome != atual.Nome || usuario != atual.Usuario
		mexeFicha := tipo != atual.Tipo || endereco != atual.Endereco || !mesmosIDs(administra, atual.Administra) || rotaciona != atual.Rotaciona
		pasta := atual.PastaID
		if unidadeMudou {
			pasta = nil
		} else if m.MoverPasta {
			pasta = m.Pasta
		}
		mexePasta := !mesmaUnidade(pasta, atual.PastaID)
		if !mexeNome && !mexeSegredo && !mexeFicha && !unidadeMudou && !mexePasta {
			item = atual
			return nil
		}
		if (mexeNome || mexeSegredo || mexeFicha || mexePasta) && !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
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
		if mexePasta && pasta != nil {
			if err := s.conferirPasta(ctx, tx, *pasta, destino); err != nil {
				return err
			}
		}
		atual.Nome = nome
		atual.Usuario = usuario
		atual.Tipo = tipo
		atual.Endereco = endereco
		atual.PastaID = pasta
		atual.Rotaciona = rotaciona
		if !rotaciona {
			atual.ProximaRotacao = nil
		} else if atual.ProximaRotacao == nil {
			proxima := s.agora().UTC().Add(domain.IntervaloRotacao)
			atual.ProximaRotacao = &proxima
		}
		atual.UnidadeID = destino
		atual.UpdatedAt = s.agora().UTC()
		if m.Senha != nil {
			atual.RotacaoPendente = false
			if atual.Rotaciona {
				proxima := s.agora().UTC().Add(domain.IntervaloRotacao)
				atual.ProximaRotacao = &proxima
			}
		}
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
		if mexeFicha || tipo != domain.TipoPainel {
			ligadas, err := s.aplicarAdministracao(ctx, tx, f, atual.ID, tipo, administra)
			if err != nil {
				return err
			}
			atual.Administra = ligadas
		}
		if mexeSegredo || unidadeMudou {
			if err := s.soltar(ctx, tx, atual.ID, atual.UnidadeID); err != nil {
				return err
			}
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

// Revelar devolve a senha de quem já está autenticado e pode revelar.
// Quem só lê o cofre não chega aqui. A senha de acesso não é pedida de novo.
func (s *Service) Revelar(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Revelado, error) {
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
		if err := s.autorizaRevelar(ctx, tx, identity, atual); err != nil {
			return err
		}
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		if err := s.recusarSeOcupado(ctx, tx, id, identity.UserID); err != nil {
			return err
		}
		um := []domain.Selado{atual}
		if err := s.anexar(ctx, tx, f, um); err != nil {
			return err
		}
		if err := s.anexarRetiradas(ctx, tx, f, um); err != nil {
			return err
		}
		atual = um[0]
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
		return domain.EntradaInvalida{Msg: "informe a sua senha de acesso"}
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

// Retirar reserva o segredo e devolve a senha. Outra pessoa não revela
// enquanto o prazo não acaba ou alguém devolve.
func (s *Service) Retirar(ctx context.Context, identity auth.Identity, id uuid.UUID, senhaAcesso string, minutos int) (domain.Revelado, error) {
	if identity.Source == auth.SourceAgente || identity.UserID == uuid.Nil {
		return domain.Revelado{}, MapError(domain.ErrSemPermissao)
	}
	minutos, err := domain.MinutosDeRetirada(minutos)
	if err != nil {
		return domain.Revelado{}, MapError(err)
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
		if err := s.autorizaRevelar(ctx, tx, identity, atual); err != nil {
			return err
		}
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		aberta, err := s.repo.RetiradaAberta(ctx, tx, id)
		if errors.Is(err, domain.ErrNaoEncontrado) {
			aberta, err = domain.RetiradaGravada{}, nil
		}
		if err != nil {
			return err
		}
		if aberta.ID != uuid.Nil && !aberta.Ate.After(s.agora()) {
			if err := s.encerrarERotacionar(ctx, tx, aberta, &atual, f, EventoRetiradaVencida); err != nil {
				return err
			}
			aberta = domain.RetiradaGravada{}
		}
		if aberta.ID != uuid.Nil && aberta.UsuarioID != identity.UserID {
			return domain.ErrOcupado
		}
		if err := s.confirmarAcesso(ctx, identity, atual.ID, atual.UnidadeID, senhaAcesso); err != nil {
			return err
		}
		um := []domain.Selado{atual}
		if err := s.anexar(ctx, tx, f, um); err != nil {
			return err
		}
		atual = um[0]
		conteudo, err := s.caixa.Abrir(caixa.Selo{Dek: atual.Dek, Cifra: atual.Cifra}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
		if err != nil {
			return err
		}
		ate := s.agora().UTC().Add(time.Duration(minutos) * time.Minute)
		nome := nomeDaPessoa(identity)
		if aberta.ID != uuid.Nil {
			if err := s.repo.RenovarRetirada(ctx, tx, aberta.ID, ate, nome); err != nil {
				return err
			}
		} else if err := s.repo.InserirRetirada(ctx, tx, domain.RetiradaGravada{
			ID: uuid.New(), SegredoID: atual.ID, UsuarioID: identity.UserID, Nome: nome, Ate: ate,
		}); err != nil {
			return err
		}
		atual.Retirada = &domain.Retirada{UsuarioID: identity.UserID, Nome: nome, Ate: ate}
		out = domain.Revelado{Segredo: atual.Segredo, Senha: conteudo.Senha, Notas: conteudo.Notas}
		return s.trilha.Registrar(ctx, tx, EventoRetirado, atual.ID, pistaCom(atual.ID, atual.UnidadeID, identity.UserID, ate))
	})
	return out, MapError(err)
}

// Devolver encerra a reserva. Quem retirou devolve; quem administra também.
// Se o segredo rotaciona, a senha guardada muda. O destino não.
func (s *Service) Devolver(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if identity.Source == auth.SourceAgente {
		return MapError(domain.ErrSemPermissao)
	}
	f, err := s.filtro(identity)
	if err != nil {
		return MapError(err)
	}
	return MapError(s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			return err
		}
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		aberta, err := s.repo.RetiradaAberta(ctx, tx, id)
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return nil
		}
		if err != nil {
			return err
		}
		if !aberta.Ate.After(s.agora()) {
			return s.encerrarERotacionar(ctx, tx, aberta, &atual, f, EventoRetiradaVencida)
		}
		if aberta.UsuarioID != identity.UserID && !pode(identity, auth.PermCofreManage, atual.UnidadeID) {
			return domain.ErrSemPermissao
		}
		return s.encerrarERotacionar(ctx, tx, aberta, &atual, f, EventoDevolvido)
	}))
}

func (s *Service) recusarSeOcupado(ctx context.Context, tx pgx.Tx, id, quem uuid.UUID) error {
	aberta, err := s.repo.RetiradaAberta(ctx, tx, id)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil
	}
	if err != nil {
		return err
	}
	if !aberta.Ate.After(s.agora()) || aberta.UsuarioID == quem {
		return nil
	}
	return domain.ErrOcupado
}

func (s *Service) soltar(ctx context.Context, tx pgx.Tx, id uuid.UUID, unidade *uuid.UUID) error {
	aberta, err := s.repo.RetiradaAberta(ctx, tx, id)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil
	}
	if err != nil {
		return err
	}
	evento := EventoDevolvido
	if !aberta.Ate.After(s.agora()) {
		evento = EventoRetiradaVencida
	}
	return s.encerrar(ctx, tx, aberta, unidade, evento)
}

func (s *Service) encerrar(ctx context.Context, tx pgx.Tx, aberta domain.RetiradaGravada, unidade *uuid.UUID, evento string) error {
	if err := s.repo.EncerrarRetirada(ctx, tx, aberta.ID, s.agora().UTC()); err != nil {
		return err
	}
	return s.trilha.Registrar(ctx, tx, evento, aberta.SegredoID, pistaCom(aberta.SegredoID, unidade, aberta.UsuarioID, aberta.Ate))
}

func (s *Service) encerrarERotacionar(ctx context.Context, tx pgx.Tx, aberta domain.RetiradaGravada, atual *domain.Selado, f domain.Filtro, evento string) error {
	if err := s.encerrar(ctx, tx, aberta, atual.UnidadeID, evento); err != nil {
		return err
	}
	if !atual.Rotaciona {
		return nil
	}
	return s.rotacionar(ctx, tx, atual, f)
}

// rotacionar troca a senha do envelope e marca que o endereço ainda usa a antiga.
func (s *Service) rotacionar(ctx context.Context, tx pgx.Tx, atual *domain.Selado, f domain.Filtro) error {
	conteudo, err := s.caixa.Abrir(caixa.Selo{Dek: atual.Dek, Cifra: atual.Cifra}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
	if err != nil {
		return err
	}
	gerar := s.gerar
	if gerar == nil {
		gerar = gerarSenha
	}
	senha, err := gerar()
	if err != nil {
		return err
	}
	if _, _, senha, _, err = domain.Validar(atual.Nome, atual.Usuario, senha, conteudo.Notas, true); err != nil {
		return err
	}
	atual.Versao++
	atual.UpdatedAt = s.agora().UTC()
	atual.RotacaoPendente = true
	proxima := s.agora().UTC().Add(domain.IntervaloRotacao)
	atual.ProximaRotacao = &proxima
	selo, err := s.caixa.Selar(caixa.Conteudo{Senha: senha, Notas: conteudo.Notas}, domain.AAD(atual.ID, atual.Versao, atual.UnidadeID))
	if err != nil {
		return err
	}
	atual.Dek, atual.Cifra = selo.Dek, selo.Cifra
	if err := s.repo.Update(ctx, tx, *atual, f); err != nil {
		return err
	}
	corpo := pista(atual.ID, atual.UnidadeID)
	corpo["versao"] = atual.Versao
	return s.trilha.Registrar(ctx, tx, EventoRotacionado, atual.ID, corpo)
}

// RotacionarDevidas troca a senha dos segredos marcados cujo prazo chegou.
// Segredo retirado ou com aplicação pendente fica para a próxima passagem.
func (s *Service) RotacionarDevidas(ctx context.Context) error {
	var lista []domain.Selado
	if err := s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		lista, err = s.repo.ListarParaRotacionar(ctx, tx, s.agora())
		return err
	}); err != nil {
		return err
	}
	var falhas []error
	for _, item := range lista {
		if err := s.correr.Em(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return s.rotacionarSeDevida(ctx, tx, item.ID)
		}); err != nil {
			falhas = append(falhas, err)
		}
	}
	return errors.Join(falhas...)
}

func (s *Service) rotacionarSeDevida(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	f := domain.Filtro{}
	if err := s.repo.Travar(ctx, tx, id); err != nil {
		return err
	}
	atual, err := s.repo.Get(ctx, tx, id, f)
	if err != nil {
		return err
	}
	if !atual.Rotaciona || atual.RotacaoPendente || atual.ProximaRotacao == nil || atual.ProximaRotacao.After(s.agora()) {
		return nil
	}
	aberta, err := s.repo.RetiradaAberta(ctx, tx, id)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return s.rotacionar(ctx, tx, &atual, f)
	}
	if err != nil {
		return err
	}
	if aberta.Ate.After(s.agora()) {
		return nil
	}
	return s.encerrarERotacionar(ctx, tx, aberta, &atual, f, EventoRetiradaVencida)
}

// MarcarAplicada tira o aviso depois que a senha nova foi colocada no endereço.
func (s *Service) MarcarAplicada(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
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
		if !atual.RotacaoPendente {
			return nil
		}
		if err := s.repo.Travar(ctx, tx, id); err != nil {
			return err
		}
		atual.RotacaoPendente = false
		atual.UpdatedAt = s.agora().UTC()
		if err := s.repo.Update(ctx, tx, atual, f); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, EventoRotacaoAplicada, atual.ID, pista(atual.ID, atual.UnidadeID))
	}))
}

func (s *Service) anexarRetiradas(ctx context.Context, tx pgx.Tx, f domain.Filtro, itens []domain.Selado) error {
	rel, err := s.repo.ListarRetiradas(ctx, tx, f, s.agora())
	if err != nil {
		return err
	}
	for i := range itens {
		if r, ok := rel[itens[i].ID]; ok {
			copia := r
			itens[i].Retirada = &copia
		}
	}
	return nil
}

func nomeDaPessoa(identity auth.Identity) string {
	nome := strings.TrimSpace(identity.Name)
	if nome == "" {
		nome = strings.TrimSpace(identity.Username)
	}
	if utf8.RuneCountInString(nome) > 160 {
		nome = string([]rune(nome)[:160])
	}
	return nome
}

func pistaCom(id uuid.UUID, unidade *uuid.UUID, usuario uuid.UUID, ate time.Time) map[string]any {
	corpo := pista(id, unidade)
	corpo["usuario_id"] = usuario.String()
	corpo["ate"] = ate.UTC().Format(time.RFC3339)
	return corpo
}

func (s *Service) anexar(ctx context.Context, tx pgx.Tx, f domain.Filtro, itens []domain.Selado) error {
	rel, err := s.repo.ListarAdministracao(ctx, tx, f)
	if err != nil {
		return err
	}
	for i := range itens {
		if ids := rel[itens[i].ID]; len(ids) > 0 {
			itens[i].Administra = ids
		}
	}
	return nil
}

func (s *Service) aplicarAdministracao(ctx context.Context, tx pgx.Tx, f domain.Filtro, id uuid.UUID, tipo string, maquinas []uuid.UUID) ([]uuid.UUID, error) {
	if tipo != domain.TipoPainel {
		if len(maquinas) > 0 {
			return nil, domain.EntradaInvalida{Msg: "só um painel administra máquinas"}
		}
		return nil, s.repo.SubstituirAdministracao(ctx, tx, id, nil, f)
	}
	limpas, err := s.conferirMaquinas(ctx, tx, f, id, maquinas)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SubstituirAdministracao(ctx, tx, id, limpas, f); err != nil {
		return nil, err
	}
	return limpas, nil
}

func (s *Service) conferirMaquinas(ctx context.Context, tx pgx.Tx, f domain.Filtro, painel uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	visto := map[uuid.UUID]struct{}{}
	var limpas []uuid.UUID
	for _, id := range ids {
		if id == painel {
			return nil, domain.EntradaInvalida{Msg: "o painel não administra a si mesmo"}
		}
		if _, ok := visto[id]; ok {
			continue
		}
		visto[id] = struct{}{}
		item, err := s.repo.Get(ctx, tx, id, f)
		if err != nil {
			if errors.Is(err, domain.ErrNaoEncontrado) {
				return nil, domain.EntradaInvalida{Msg: "máquina não encontrada no seu cofre"}
			}
			return nil, err
		}
		if item.Tipo != domain.TipoMaquina {
			return nil, domain.EntradaInvalida{Msg: "o painel só administra máquinas"}
		}
		limpas = append(limpas, id)
	}
	return limpas, nil
}

func mesmosIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for _, id := range a {
		if !slices.Contains(b, id) {
			return false
		}
	}
	return true
}

func mesmaUnidade(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
