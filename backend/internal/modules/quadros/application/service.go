// Package application grava o catálogo na mesma transação do outbox e da
// auditoria. O nome novo não substitui o estado do registro.
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const (
	eventoAtualizada = "quadros.coluna.atualizada"
	eventoCriada     = "quadros.coluna.criada"
)

// Eventos grava o evento no outbox da transação.
type Eventos interface {
	Write(ctx context.Context, tx pgx.Tx, eventType, aggregateType, aggregateID string, correlationID uuid.UUID, payload any) error
}

// Service implementa a leitura e a alteração do catálogo.
type Service struct {
	pool    *pgxpool.Pool
	repo    domain.Repository
	eventos Eventos
}

// NewService cria o serviço.
func NewService(pool *pgxpool.Pool, repo domain.Repository, eventos Eventos) *Service {
	return &Service{pool: pool, repo: repo, eventos: eventos}
}

// MapError traduz erros de domínio para a API.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrQuadro), errors.Is(err, domain.ErrNaoEncontrada):
		return causa(err, apperrors.NotFound("coluna não encontrada"))
	case errors.Is(err, domain.ErrProibido):
		return causa(err, apperrors.Forbidden("você não gere as colunas deste quadro"))
	case errors.Is(err, domain.ErrPatchVazio):
		return causa(err, apperrors.Validation("informe o nome, a cor, a ordem ou se a coluna fica no quadro"))
	case errors.Is(err, domain.ErrNome):
		return causa(err, apperrors.Validation("informe o nome da coluna (até 80 caracteres)"))
	case errors.Is(err, domain.ErrCor):
		return causa(err, apperrors.Validation("a cor usa o formato #rrggbb"))
	case errors.Is(err, domain.ErrInicial):
		return causa(err, apperrors.Conflict("a coluna inicial do fluxo não pode ser retirada"))
	case errors.Is(err, domain.ErrComRegistro):
		return causa(err, apperrors.Conflict("esta coluna ainda tem registro; esvazie antes de retirar"))
	case errors.Is(err, domain.ErrRetirada):
		return causa(err, apperrors.Conflict("devolva a coluna ao quadro antes de mudar a ordem"))
	case errors.Is(err, domain.ErrSemEntrada):
		return causa(err, apperrors.Validation("escolha ao menos uma coluna de onde o card entra"))
	case errors.Is(err, domain.ErrPassagem):
		return causa(err, apperrors.Validation("a passagem usa só colunas ativas deste quadro; a inicial do fim do fluxo e a aprovação não entram"))
	default:
		return apperrors.Internal(err)
	}
}

func causa(err error, api *apperrors.Error) error {
	api.Err = err
	return api
}

// Listar devolve o catálogo de quem já vê o quadro.
func (s *Service) Listar(ctx context.Context, id auth.Identity, quadro string) ([]domain.Coluna, error) {
	if !domain.Conhecido(quadro) {
		return nil, domain.ErrQuadro
	}
	if !domain.PodeLer(id, quadro) {
		return nil, domain.ErrProibido
	}
	cols, err := s.repo.Listar(ctx, s.pool, quadro)
	if err != nil {
		return nil, err
	}
	mapa, err := s.repo.Mapa(ctx, s.pool, quadro)
	if err != nil {
		return nil, err
	}
	return preencher(cols, mapa), nil
}

// Criar acrescenta uma coluna só neste quadro. A chave é gerada.
func (s *Service) Criar(ctx context.Context, id auth.Identity, quadro string, n domain.Nova) (domain.Coluna, error) {
	if !domain.Conhecido(quadro) {
		return domain.Coluna{}, domain.ErrQuadro
	}
	if !domain.PodeEscrever(id, quadro) {
		return domain.Coluna{}, domain.ErrProibido
	}
	var out domain.Coluna
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		cols, err := s.repo.Listar(ctx, tx, quadro)
		if err != nil {
			return err
		}
		chave, err := chaveLivre(cols)
		if err != nil {
			return err
		}
		pronta, passagens, proximas, err := domain.Criar(cols, chave, n)
		if err != nil {
			return err
		}
		out = pronta
		if err := s.repo.SoltarOrdens(ctx, tx, quadro); err != nil {
			return err
		}
		if err := s.repo.Inserir(ctx, tx, pronta); err != nil {
			return err
		}
		if err := s.repo.Salvar(ctx, tx, proximas); err != nil {
			return err
		}
		if err := s.repo.SubstituirPassagens(ctx, tx, quadro, chave, passagens); err != nil {
			return err
		}
		payload := map[string]any{"quadro": quadro, "chave": chave, "ordem": pronta.Ordem}
		if err := s.eventos.Write(ctx, tx, eventoCriada, "quadro_coluna", quadro+":"+chave, uuid.Nil, payload); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, eventoCriada, "quadro_coluna", quadro+":"+chave, nil, fotoNova(pronta, proximas)))
	})
	return out, err
}

// Atualizar altera nome, cor, ordem ou presença. A chave permanece.
func (s *Service) Atualizar(ctx context.Context, id auth.Identity, quadro, chave string, p domain.Patch) (domain.Coluna, error) {
	if !domain.Conhecido(quadro) {
		return domain.Coluna{}, domain.ErrQuadro
	}
	if !domain.PodeEscrever(id, quadro) {
		return domain.Coluna{}, domain.ErrProibido
	}
	var out domain.Coluna
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		cols, err := s.repo.Listar(ctx, tx, quadro)
		if err != nil {
			return err
		}
		antes, ok := achar(cols, chave)
		if !ok {
			return domain.ErrNaoEncontrada
		}
		n, err := s.repo.Contar(ctx, tx, quadro, chave)
		if err != nil {
			return err
		}
		proximas, err := domain.Aplicar(cols, chave, n, p)
		if err != nil {
			return err
		}
		depois, _ := achar(proximas, chave)
		out = depois
		if domain.Iguais(cols, proximas) {
			return nil
		}
		if err := s.repo.SoltarOrdens(ctx, tx, quadro); err != nil {
			return err
		}
		if err := s.repo.Salvar(ctx, tx, proximas); err != nil {
			return err
		}
		payload := map[string]any{
			"quadro": quadro, "chave": chave, "ordem": depois.Ordem, "ativa": depois.Ativa,
		}
		if err := s.eventos.Write(ctx, tx, eventoAtualizada, "quadro_coluna", quadro+":"+chave, uuid.Nil, payload); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, eventoAtualizada, "quadro_coluna", quadro+":"+chave, foto(antes, cols), foto(depois, proximas)))
	})
	return out, err
}

func achar(cols []domain.Coluna, chave string) (domain.Coluna, bool) {
	for _, c := range cols {
		if c.Chave == chave {
			return c, true
		}
	}
	return domain.Coluna{}, false
}

type fotoColuna struct {
	Nome      string   `json:"nome"`
	Cor       string   `json:"cor"`
	Ordem     int      `json:"ordem"`
	Ativa     bool     `json:"ativa"`
	Sequencia []string `json:"sequencia"`
}

func foto(c domain.Coluna, cols []domain.Coluna) fotoColuna {
	return fotoColuna{Nome: c.Nome, Cor: c.Cor, Ordem: c.Ordem, Ativa: c.Ativa, Sequencia: domain.Sequencia(cols)}
}

type fotoNovaColuna struct {
	Nome      string   `json:"nome"`
	Cor       string   `json:"cor"`
	Ordem     int      `json:"ordem"`
	De        []string `json:"de"`
	Para      []string `json:"para"`
	Sequencia []string `json:"sequencia"`
}

func fotoNova(c domain.Coluna, cols []domain.Coluna) fotoNovaColuna {
	return fotoNovaColuna{Nome: c.Nome, Cor: c.Cor, Ordem: c.Ordem, De: c.De, Para: c.Para, Sequencia: domain.Sequencia(cols)}
}

func chaveLivre(cols []domain.Coluna) (string, error) {
	ocupadas := map[string]bool{}
	for _, c := range cols {
		ocupadas[c.Chave] = true
	}
	for range 5 {
		var buf [4]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return "", err
		}
		chave := "c" + hex.EncodeToString(buf[:])
		if !ocupadas[chave] {
			return chave, nil
		}
	}
	return "", domain.ErrPassagem
}

func preencher(cols []domain.Coluna, mapa map[string][]string) []domain.Coluna {
	out := slices.Clone(cols)
	for i := range out {
		para := append([]string{}, mapa[out[i].Chave]...)
		slices.Sort(para)
		var de []string
		for origem, destinos := range mapa {
			if slices.Contains(destinos, out[i].Chave) {
				de = append(de, origem)
			}
		}
		slices.Sort(de)
		out[i].De, out[i].Para = de, para
	}
	return out
}
