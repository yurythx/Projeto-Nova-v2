package application

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

const (
	EventProjetoCreated      = "projetos.projeto.created"
	EventProjetoUpdated      = "projetos.projeto.updated"
	EventProjetoTransitioned = "projetos.projeto.transitioned"
)

type Service struct {
	pool   *pgxpool.Pool
	repo   domain.Repository
	outbox *outbox.Writer
	logger *slog.Logger
}

func NewService(pool *pgxpool.Pool, repo domain.Repository, ob *outbox.Writer, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, outbox: ob, logger: logger}
}

func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrProjetoNaoEncontrado):
		return apperrors.NotFound("projeto não encontrado")
	case errors.Is(err, domain.ErrTarefaNaoEncontrada):
		return apperrors.NotFound("tarefa não encontrada")
	case errors.Is(err, domain.ErrTituloObrigatorio):
		return apperrors.Validation("informe um título válido (até 160 caracteres)")
	case errors.Is(err, domain.ErrTransicaoInvalida):
		return apperrors.Conflict("passagem de estado não permitida para o estado atual")
	case errors.Is(err, domain.ErrProjetoEncerrado):
		return apperrors.Conflict("projeto cancelado não aceita alterações")
	case errors.Is(err, domain.ErrPrioridadeInvalida):
		return apperrors.Validation("prioridade inválida")
	case errors.Is(err, domain.ErrApontamentoInvalido):
		return apperrors.Validation("horas devem ser maiores que zero e até 24h, com descrição")
	case errors.Is(err, domain.ErrApontamentoNaoEncontrado):
		return apperrors.NotFound("apontamento não encontrado")
	}
	return err
}

func (s *Service) Criar(ctx context.Context, idt auth.Identity, e domain.EntradaProjeto) (*domain.Projeto, error) {
	if err := e.Validar(); err != nil {
		return nil, err
	}

	p := &domain.Projeto{
		ID:                  uuid.New(),
		Titulo:              e.Titulo,
		Descricao:           e.Descricao,
		Objetivo:            e.Objetivo,
		Estado:              domain.EstadoPlanejado,
		Prioridade:          e.Prioridade,
		DataInicioPlanejada: e.DataInicioPlanejada,
		DataFimPlanejada:    e.DataFimPlanejada,
		DataInicioReal:      e.DataInicioReal,
		DataFimReal:         e.DataFimReal,
		Orcamento:           e.Orcamento,
		UnidadeID:           e.UnidadeID,
		ProdutoID:           e.ProdutoID,
		GerenteID:           e.GerenteID,
		GerenteNome:         e.GerenteNome,
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Criar(ctx, tx, p); err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"id": p.ID, "numero": p.Numero, "titulo": p.Titulo}
			if err := s.outbox.Write(ctx, tx, EventProjetoCreated, "projeto", p.ID.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoCreated, "projeto", p.ID.String(), nil, p))
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Obter(ctx context.Context, idt auth.Identity, id uuid.UUID) (*domain.Projeto, error) {
	return s.repo.Obter(ctx, s.pool, id)
}

func (s *Service) Listar(ctx context.Context, idt auth.Identity, estado, busca string) ([]domain.Projeto, error) {
	f := domain.FiltroProjetos{
		Estado: estado,
		Busca:  busca,
	}
	return s.repo.Listar(ctx, s.pool, f)
}

func (s *Service) Atualizar(ctx context.Context, idt auth.Identity, id uuid.UUID, e domain.EntradaProjeto) (*domain.Projeto, error) {
	if err := e.Validar(); err != nil {
		return nil, err
	}

	var atualizado *domain.Projeto
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if p.Estado == domain.EstadoCancelado {
			return domain.ErrProjetoEncerrado
		}

		res, err := s.repo.Atualizar(ctx, tx, id, e)
		if err != nil {
			return err
		}
		atualizado = res

		if s.outbox != nil {
			payload := map[string]any{"id": id, "titulo": e.Titulo}
			_ = s.outbox.Write(ctx, tx, EventProjetoUpdated, "projeto", id.String(), uuid.Nil, payload)
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoUpdated, "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	return atualizado, nil
}

func (s *Service) Transicionar(ctx context.Context, idt auth.Identity, id uuid.UUID, novoEstado string) (*domain.Projeto, error) {
	var atualizado *domain.Projeto
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}

		if !domain.ValidarTransicao(p.Estado, novoEstado) {
			return domain.ErrTransicaoInvalida
		}

		res, err := s.repo.Transicionar(ctx, tx, id, novoEstado)
		if err != nil {
			return err
		}
		atualizado = res

		if s.outbox != nil {
			payload := map[string]any{"id": id, "estado_anterior": p.Estado, "novo_estado": novoEstado}
			_ = s.outbox.Write(ctx, tx, EventProjetoTransitioned, "projeto", id.String(), uuid.Nil, payload)
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoTransitioned, "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	return atualizado, nil
}

func (s *Service) CriarTarefa(ctx context.Context, idt auth.Identity, projetoID uuid.UUID, titulo, descricao, responsavel, estado string, ordem int) (*domain.Tarefa, error) {
	t := &domain.Tarefa{
		ID:              uuid.New(),
		ProjetoID:       projetoID,
		Titulo:          titulo,
		Descricao:       descricao,
		ResponsavelNome: responsavel,
		Estado:          estado,
		Ordem:           ordem,
	}
	if err := s.repo.CriarTarefa(ctx, s.pool, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) AtualizarTarefa(ctx context.Context, idt auth.Identity, id uuid.UUID, titulo, descricao, responsavel, estado string, ordem int) (*domain.Tarefa, error) {
	return s.repo.AtualizarTarefa(ctx, s.pool, id, titulo, descricao, responsavel, estado, ordem)
}

func (s *Service) ListarTarefas(ctx context.Context, idt auth.Identity, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	return s.repo.ListarTarefas(ctx, s.pool, projetoID)
}

func (s *Service) ExcluirTarefa(ctx context.Context, idt auth.Identity, id uuid.UUID) error {
	return s.repo.ExcluirTarefa(ctx, s.pool, id)
}

func (s *Service) ListarVinculos(ctx context.Context, idt auth.Identity, projetoID uuid.UUID) ([]domain.Vinculo, error) {
	return s.repo.ListarVinculos(ctx, s.pool, projetoID)
}

func (s *Service) Vincular(ctx context.Context, idt auth.Identity, projetoID uuid.UUID, tipo string, alvoID uuid.UUID, rotulo string) (*domain.Vinculo, error) {
	v := &domain.Vinculo{
		ID:           uuid.New(),
		ProjetoID:    projetoID,
		Tipo:         tipo,
		AlvoID:       alvoID,
		Rotulo:       rotulo,
		VinculadoPor: idt.Email,
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.CriarVinculo(ctx, tx, v); err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"projeto_id": projetoID, "tipo": tipo, "alvo_id": alvoID, "rotulo": rotulo}
			_ = s.outbox.Write(ctx, tx, "projetos.vinculo.created", "projeto", projetoID.String(), uuid.Nil, payload)
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.vinculo.created", "projeto", projetoID.String(), nil, v))
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Desvincular(ctx context.Context, idt auth.Identity, vinculoID uuid.UUID) error {
	return s.repo.ExcluirVinculo(ctx, s.pool, vinculoID)
}

func (s *Service) RegistrarApontamento(
	ctx context.Context,
	idt auth.Identity,
	projetoID uuid.UUID,
	tarefaID *uuid.UUID,
	dataApontamento string,
	horas float64,
	descricao string,
	custoHora float64,
) (*domain.Apontamento, error) {
	nomeUsuario := idt.Name
	if nomeUsuario == "" {
		nomeUsuario = idt.Username
	}
	if nomeUsuario == "" {
		nomeUsuario = idt.Email
	}
	if nomeUsuario == "" {
		nomeUsuario = "Usuário"
	}

	usuarioID := idt.UserID
	if usuarioID == uuid.Nil {
		if parsed, err := uuid.Parse(idt.Subject); err == nil {
			usuarioID = parsed
		} else {
			usuarioID = uuid.New()
		}
	}

	a := &domain.Apontamento{
		ID:              uuid.New(),
		ProjetoID:       projetoID,
		TarefaID:        tarefaID,
		UsuarioID:       usuarioID,
		UsuarioNome:     nomeUsuario,
		DataApontamento: dataApontamento,
		Horas:           horas,
		Descricao:       descricao,
		CustoHora:       custoHora,
		CustoTotal:      horas * custoHora,
	}
	if err := a.Validar(); err != nil {
		return nil, MapError(err)
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.repo.Obter(ctx, tx, projetoID); err != nil {
			return err
		}
		if err := s.repo.CriarApontamento(ctx, tx, a); err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{
				"projeto_id":  projetoID,
				"tarefa_id":   tarefaID,
				"usuario_id":  usuarioID,
				"horas":       horas,
				"custo_total": a.CustoTotal,
			}
			_ = s.outbox.Write(ctx, tx, "projetos.apontamento.created", "projeto", projetoID.String(), uuid.Nil, payload)
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.apontamento.created", "projeto", projetoID.String(), nil, a))
	})
	if err != nil {
		return nil, MapError(err)
	}
	return a, nil
}

func (s *Service) ListarApontamentos(
	ctx context.Context,
	idt auth.Identity,
	projetoID uuid.UUID,
) ([]domain.Apontamento, domain.ResumoHorasProjeto, error) {
	lista, err := s.repo.ListarApontamentos(ctx, s.pool, projetoID)
	if err != nil {
		return nil, domain.ResumoHorasProjeto{}, MapError(err)
	}

	resumo := domain.ResumoHorasProjeto{
		PorUsuario: make(map[string]float64),
	}
	for _, a := range lista {
		resumo.TotalHoras += a.Horas
		resumo.TotalCusto += a.CustoTotal
		resumo.PorUsuario[a.UsuarioNome] += a.Horas
	}

	return lista, resumo, nil
}

func (s *Service) ExcluirApontamento(ctx context.Context, idt auth.Identity, apontamentoID uuid.UUID) error {
	return s.repo.ExcluirApontamento(ctx, s.pool, apontamentoID)
}
