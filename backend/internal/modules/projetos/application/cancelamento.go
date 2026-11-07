package application

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Cancelar tira o projeto de circulação. Só um administrador, com o motivo.
// A passagem comum de estado não cancela.
func (s *Service) Cancelar(ctx context.Context, idt auth.Identity, id uuid.UUID, motivo string) (*domain.Projeto, error) {
	motivo, err := domain.NormalizarMotivo(motivo)
	if err != nil {
		return nil, err
	}
	if !domain.PodeCancelar(idt) {
		return nil, domain.ErrCancelamentoNaoAutorizado
	}
	var atualizado *domain.Projeto
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if !s.podeLer(idt, p.UnidadeID) && !s.podeGerir(idt, p.UnidadeID) && !idt.IsAdmin() {
			return domain.ErrForaDoEscopo
		}
		mapa, err := s.mapa(ctx, tx, "projetos")
		if err != nil {
			return err
		}
		if !domain.ValidarTransicao(p.Estado, domain.EstadoCancelado, mapa[p.Estado]...) {
			return domain.ErrTransicaoInvalida
		}
		res, err := s.repo.Transicionar(ctx, tx, id, domain.EstadoCancelado)
		if err != nil {
			return err
		}
		aprovador := idt.UserID
		if aprovador == uuid.Nil {
			if parsed, parseErr := uuid.Parse(idt.Subject); parseErr == nil {
				aprovador = parsed
			}
		}
		reg := &domain.Cancelamento{
			ID: uuid.New(), ProjetoID: id, Motivo: motivo,
			AprovadorID: aprovador, AprovadorNome: nomeDe(idt),
		}
		if err := s.repo.SalvarCancelamento(ctx, tx, reg); err != nil {
			return err
		}
		res.Cancelamento = reg
		atualizado = res
		enviadas, err = s.avisarProjeto(ctx, tx, idt.UserID, res,
			"Projeto #"+strconv.FormatInt(res.Numero, 10)+" cancelado",
			res.Titulo+". Motivo: "+motivo, "cancelado")
		if err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"id": id, "motivo": motivo, "aprovador_id": aprovador}
			if err := s.outbox.Write(ctx, tx, "projetos.projeto.cancelled", "projeto", id.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.projeto.cancelled", "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	s.entregar(ctx, enviadas)
	return s.comProximos(ctx, s.pool, atualizado)
}

func (s *Service) comCancelamento(ctx context.Context, db database.DBTX, p *domain.Projeto) (*domain.Projeto, error) {
	if p == nil || p.Estado != domain.EstadoCancelado {
		return p, nil
	}
	reg, achou, err := s.repo.UltimoCancelamento(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	if achou {
		p.Cancelamento = &reg
	}
	return p, nil
}
