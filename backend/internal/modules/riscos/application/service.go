package application

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

const (
	EventoRiscoCriado     = "riscos.risco.criado"
	EventoRiscoAtualizado = "riscos.risco.atualizado"
	EventoRiscoExcluido   = "riscos.risco.excluido"
)

type RiscoEntrada struct {
	Codigo          string     `json:"codigo"`
	Titulo          string     `json:"titulo"`
	Descricao       string     `json:"descricao"`
	Categoria       string     `json:"categoria"`
	AtivoID         *uuid.UUID `json:"ativo_id"`
	UnidadeID       uuid.UUID  `json:"unidade_id"`
	Probabilidade   int        `json:"probabilidade"`
	Impacto         int        `json:"impacto"`
	Estrategia      string     `json:"estrategia"`
	PlanoMitigacao  string     `json:"plano_mitigacao"`
	MudancaID       *uuid.UUID `json:"mudanca_id"`
	ProjetoID       *uuid.UUID `json:"projeto_id"`
	ResponsavelID   *uuid.UUID `json:"responsavel_id"`
	ResponsavelNome string     `json:"responsavel_nome"`
	Status          string     `json:"status"`
}

type CelulaHeatmap struct {
	Probabilidade int `json:"probabilidade"`
	Impacto       int `json:"impacto"`
	Severidade    int `json:"severidade"`
	Quantidade    int `json:"quantidade"`
}

type ResumoHeatmap struct {
	Matriz      []CelulaHeatmap `json:"matriz"`
	TotalRiscos int             `json:"total_riscos"`
	Criticos    int             `json:"criticos"`
	Altos       int             `json:"altos"`
	Medios      int             `json:"medios"`
	Baixos      int             `json:"baixos"`
}

type Service struct {
	pool   *pgxpool.Pool
	repo   domain.Repository
	outbox *outbox.Writer
}

func NewService(pool *pgxpool.Pool, repo domain.Repository, outbox *outbox.Writer) *Service {
	return &Service{pool: pool, repo: repo, outbox: outbox}
}

func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrRiscoNaoEncontrado):
		return apperrors.NotFound("risco não encontrado")
	case errors.Is(err, domain.ErrRiscoInvalido):
		return apperrors.Validation("dados do risco inválidos")
	case errors.Is(err, domain.ErrForaDoEscopo):
		return apperrors.Forbidden("você não tem permissão para gerenciar riscos desta unidade")
	default:
		return err
	}
}

func cobre(cob auth.Coverage, unidadeID uuid.UUID) bool {
	return cob.All || slices.Contains(cob.Unidades, unidadeID)
}

func (s *Service) ListarRiscos(ctx context.Context, identity auth.Identity, unidade *uuid.UUID, categoria, status string) ([]domain.Risco, error) {
	cob := auth.CoverageOf(identity, "riscos:read")
	if !cob.All && len(cob.Unidades) == 0 {
		cob = auth.CoverageOf(identity, "riscos:manage")
	}

	var unidades []uuid.UUID
	if unidade != nil {
		unidades = []uuid.UUID{*unidade}
	} else if !cob.All {
		unidades = cob.Unidades
	}

	itens, err := s.repo.ListRiscos(ctx, s.pool, domain.Filtro{
		Unidades:  unidades,
		Categoria: categoria,
		Status:    status,
	})
	return itens, MapError(err)
}

func (s *Service) ObterRisco(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Risco, error) {
	item, err := s.repo.GetRisco(ctx, s.pool, id)
	if err != nil {
		return domain.Risco{}, MapError(err)
	}
	cob := auth.CoverageOf(identity, "riscos:read")
	if !cobre(cob, item.UnidadeID) {
		return domain.Risco{}, MapError(domain.ErrForaDoEscopo)
	}
	return item, nil
}

func (s *Service) CriarRisco(ctx context.Context, identity auth.Identity, in RiscoEntrada) (domain.Risco, error) {
	cob := auth.CoverageOf(identity, "riscos:manage")
	if !cobre(cob, in.UnidadeID) {
		return domain.Risco{}, MapError(domain.ErrForaDoEscopo)
	}

	r, err := domain.NovoRisco(
		in.Codigo, in.Titulo, in.Descricao, in.Categoria, in.UnidadeID,
		in.Probabilidade, in.Impacto, in.Estrategia, in.PlanoMitigacao,
		in.ResponsavelID, in.ResponsavelNome,
	)
	if err != nil {
		return domain.Risco{}, MapError(err)
	}
	r.AtivoID = in.AtivoID
	r.MudancaID = in.MudancaID
	r.ProjetoID = in.ProjetoID

	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if r, err = s.repo.InsertRisco(ctx, tx, r); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, EventoRiscoCriado, "risco", r.ID.String(), uuid.New(), r); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventoRiscoCriado, "risco", r.ID.String(), nil, r))
	})
	return r, MapError(err)
}

func (s *Service) AtualizarRisco(ctx context.Context, identity auth.Identity, id uuid.UUID, in RiscoEntrada) (domain.Risco, error) {
	prev, err := s.repo.GetRisco(ctx, s.pool, id)
	if err != nil {
		return domain.Risco{}, MapError(err)
	}
	cob := auth.CoverageOf(identity, "riscos:manage")
	if !cobre(cob, prev.UnidadeID) || !cobre(cob, in.UnidadeID) {
		return domain.Risco{}, MapError(domain.ErrForaDoEscopo)
	}

	next, err := prev.Aplicar(
		in.Codigo, in.Titulo, in.Descricao, in.Categoria, in.UnidadeID,
		in.Probabilidade, in.Impacto, in.Estrategia, in.PlanoMitigacao,
		in.ResponsavelID, in.ResponsavelNome, in.Status,
	)
	if err != nil {
		return domain.Risco{}, MapError(err)
	}
	next.AtivoID = in.AtivoID
	next.MudancaID = in.MudancaID
	next.ProjetoID = in.ProjetoID

	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if next, err = s.repo.UpdateRisco(ctx, tx, next); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, EventoRiscoAtualizado, "risco", next.ID.String(), uuid.New(), next); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventoRiscoAtualizado, "risco", next.ID.String(), prev, next))
	})
	return next, MapError(err)
}

func (s *Service) ExcluirRisco(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	r, err := s.repo.GetRisco(ctx, s.pool, id)
	if err != nil {
		return MapError(err)
	}
	cob := auth.CoverageOf(identity, "riscos:manage")
	if !cobre(cob, r.UnidadeID) {
		return MapError(domain.ErrForaDoEscopo)
	}

	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.DeleteRisco(ctx, tx, id); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, EventoRiscoExcluido, "risco", id.String(), uuid.New(), map[string]string{"id": id.String()}); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventoRiscoExcluido, "risco", id.String(), r, nil))
	})
	return MapError(err)
}

func (s *Service) Heatmap(ctx context.Context, identity auth.Identity, unidade *uuid.UUID) (ResumoHeatmap, error) {
	riscos, err := s.ListarRiscos(ctx, identity, unidade, "", "")
	if err != nil {
		return ResumoHeatmap{}, err
	}

	matrizMap := map[[2]int]int{}
	res := ResumoHeatmap{
		TotalRiscos: len(riscos),
		Matriz:      make([]CelulaHeatmap, 0, 25),
	}

	for _, r := range riscos {
		matrizMap[[2]int{r.Probabilidade, r.Impacto}]++
		switch r.NivelSeveridade() {
		case "critico":
			res.Criticos++
		case "alto":
			res.Altos++
		case "medio":
			res.Medios++
		case "baixo":
			res.Baixos++
		}
	}

	for p := 5; p >= 1; p-- {
		for i := 1; i <= 5; i++ {
			res.Matriz = append(res.Matriz, CelulaHeatmap{
				Probabilidade: p,
				Impacto:       i,
				Severidade:    p * i,
				Quantidade:    matrizMap[[2]int{p, i}],
			})
		}
	}

	return res, nil
}
