package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

func (s *Service) podeComentar(idt auth.Identity, projeto *domain.Projeto, tarefa *domain.Tarefa) bool {
	if projeto == nil || tarefa == nil {
		return false
	}
	if s.podeGerir(idt, projeto.UnidadeID) {
		return true
	}
	if projeto.GerenteID != nil && *projeto.GerenteID == idt.UserID {
		return true
	}
	return tarefa.ResponsavelID != nil && *tarefa.ResponsavelID == idt.UserID
}

func (s *Service) ListarAtualizacoes(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID) ([]domain.Atualizacao, error) {
	_, projeto, err := s.tarefaNoProjeto(ctx, s.pool, projetoID, tarefaID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	return s.repo.ListarAtualizacoes(ctx, s.pool, tarefaID)
}

func (s *Service) CriarAtualizacao(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID, texto string) (*domain.Atualizacao, error) {
	limpo, err := domain.ValidarAtualizacao(texto)
	if err != nil {
		return nil, err
	}
	var criada *domain.Atualizacao
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		tarefa, projeto, err := s.tarefaNoProjeto(ctx, tx, projetoID, tarefaID)
		if err != nil {
			return err
		}
		if !s.podeLer(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		if !s.podeComentar(idt, projeto, tarefa) {
			return domain.ErrAtualizacaoNaoAutorizada
		}
		autorNome := strings.TrimSpace(idt.Name)
		if autorNome == "" {
			autorNome = idt.Username
		}
		var autorID *uuid.UUID
		if idt.UserID != uuid.Nil {
			id := idt.UserID
			autorID = &id
		}
		a := &domain.Atualizacao{
			ID: uuid.New(), TarefaID: tarefa.ID, Texto: limpo,
			AutorID: autorID, AutorNome: autorNome, CreatedAt: time.Now().UTC(),
		}
		if err := s.repo.CriarAtualizacao(ctx, tx, a); err != nil {
			return err
		}
		if err := audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.tarefa.atualizacao", "projeto_tarefa", tarefa.ID.String(), nil, a)); err != nil {
			return err
		}
		parte, err := s.avisarAtualizacao(ctx, tx, idt.UserID, projeto, tarefa, limpo)
		if err != nil {
			return err
		}
		enviadas = append(enviadas, parte...)
		criada = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.entregar(ctx, enviadas)
	return criada, nil
}

func (s *Service) avisarAtualizacao(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto, t *domain.Tarefa, texto string) ([]notificacoes.Enviada, error) {
	if p == nil || t == nil {
		return nil, nil
	}
	destinos := append(idDe(t.ResponsavelID), idDe(p.GerenteID)...)
	return s.registrar(ctx, tx, autor, destinos, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Atualização na tarefa",
		Mensagem: cortar(t.Titulo+": "+texto, 1000),
		Link:     linkProjeto(p.ID),
		Chave:    chaveAviso("tarefa", t.ID.String(), "atualizacao"),
	})
}

func (s *Service) ListarPrazos(ctx context.Context, idt auth.Identity, mes string) ([]domain.PrazoAgenda, error) {
	if err := domain.MesValido(mes); err != nil {
		return nil, err
	}
	restrito, unidades := coberturaDe(idt)
	return s.repo.ListarPrazos(ctx, s.pool, mes, restrito, unidades)
}
