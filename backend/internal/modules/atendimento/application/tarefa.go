package application

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Eventos das tarefas por equipe (ADR 068).
const (
	EventoOfertaTarefas = "atendimento.oferta.tarefas"
	EventoTarefaFeita   = "atendimento.solicitacao.tarefa"
)

// ModeloDaOferta devolve o modelo de tarefas da oferta (quem gere a unidade).
func (s *Service) ModeloDaOferta(ctx context.Context, identity auth.Identity, oferta uuid.UUID) ([]domain.ModeloTarefa, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return nil, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return nil, MapError(domain.ErrNaoEncontrado)
	}
	modelo, err := s.repo.ModeloDaOferta(ctx, s.pool, oferta)
	return modelo, MapError(err)
}

// SalvarModelo troca o modelo de tarefas da oferta. Cada equipe precisa
// existir e estar ao alcance de quem gere. Vale para os pedidos que
// chegarem à fila daqui em diante.
func (s *Service) SalvarModelo(ctx context.Context, identity auth.Identity, oferta uuid.UUID, modelo []domain.ModeloTarefa) ([]domain.ModeloTarefa, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return nil, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return nil, MapError(domain.ErrNaoEncontrado)
	}
	validas, err := domain.ValidarModelo(o.Tipo, modelo)
	if err != nil {
		return nil, MapError(err)
	}
	for i, m := range validas {
		e, err := s.repo.GetEquipe(ctx, s.pool, m.EquipeID)
		if err != nil || !ver(identity, e.UnidadeID) {
			return nil, MapError(domain.ErrModeloInvalido)
		}
		validas[i].EquipeNome = e.Nome
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SalvarModelo(ctx, tx, oferta, validas); err != nil {
			return err
		}
		return s.emitir(ctx, tx, EventoOfertaTarefas, "oferta", oferta, validas, nil)
	})
	if err != nil {
		return nil, MapError(err)
	}
	return validas, nil
}

// criarTarefas cria as tarefas do modelo quando a requisição chega à fila
// e avisa cada equipe da sua.
func (s *Service) criarTarefas(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, autor uuid.UUID) ([]notificacoes.Enviada, error) {
	if sol.Tipo != domain.TipoRequisicao {
		return nil, nil
	}
	modelo, err := s.repo.ModeloDaOferta(ctx, tx, sol.OfertaID)
	if err != nil || len(modelo) == 0 {
		return nil, err
	}
	if err := s.repo.InsertTarefas(ctx, tx, domain.TarefasDoModelo(sol.ID, modelo, sol.UpdatedAt)); err != nil {
		return nil, err
	}
	var enviadas []notificacoes.Enviada
	for _, m := range modelo {
		e, err := s.repo.GetEquipe(ctx, tx, m.EquipeID)
		if err != nil {
			return nil, err
		}
		avisos, err := s.avisar(ctx, tx, sol, e.Membros, autor, fmt.Sprintf("tarefa:%d", m.Ordem),
			fmt.Sprintf("Tarefa no pedido #%d: %s", sol.Numero, m.Titulo), sol.OfertaNome+": "+sol.Titulo)
		if err != nil {
			return nil, err
		}
		enviadas = append(enviadas, avisos...)
	}
	return enviadas, nil
}

// FazerTarefa conclui ou dispensa uma tarefa do pedido. Só quem atende o
// chamado age; concluir é de quem é da equipe da tarefa, e dispensar
// pede motivo. A última tarefa feita avisa o responsável.
func (s *Service) FazerTarefa(ctx context.Context, identity auth.Identity, id, tarefa uuid.UUID, concluir bool, observacao string) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrNaoEhDaEquipe
		}
		if sol.Finalizada() {
			return nil, domain.ErrSolicitacaoFinalizada
		}
		prev, err := s.repo.GetTarefa(ctx, tx, id, tarefa)
		if err != nil {
			return nil, err
		}
		daEquipe := false
		if prev.EquipeID != nil {
			membro, err := s.repo.MembroDe(ctx, tx, identity.UserID, []uuid.UUID{*prev.EquipeID})
			if err != nil {
				return nil, err
			}
			daEquipe = len(membro) > 0
		}
		next, err := prev.Fazer(identity.UserID, daEquipe, concluir, observacao, s.agora().UTC())
		if err != nil {
			return nil, err
		}
		if err := s.repo.UpdateTarefa(ctx, tx, next); err != nil {
			return nil, err
		}
		if err := s.registrar(ctx, tx, EventoTarefaFeita, "solicitacao", id,
			map[string]any{"id": id.String(), "tarefa_id": tarefa.String(), "estado": next.Estado}, next, prev); err != nil {
			return nil, err
		}
		tarefas, err := s.repo.ListTarefas(ctx, tx, id)
		if err != nil || domain.Pendentes(tarefas) > 0 || sol.ResponsavelID == nil {
			return nil, err
		}
		return s.avisar(ctx, tx, sol, []uuid.UUID{*sol.ResponsavelID}, identity.UserID, "tarefas",
			fmt.Sprintf("Tarefas do pedido #%d feitas", sol.Numero), "Todas as equipes terminaram; o pedido pode ser dado como atendido.")
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}

// conferirTarefas impede dar o pedido por atendido com tarefa pendente.
func (s *Service) conferirTarefas(ctx context.Context, tx pgx.Tx, prev, next domain.Solicitacao) error {
	if next.Tipo != domain.TipoRequisicao || next.Estado != domain.EstadoAtendido || prev.Estado == domain.EstadoAtendido {
		return nil
	}
	tarefas, err := s.repo.ListTarefas(ctx, tx, next.ID)
	if err != nil {
		return err
	}
	if domain.Pendentes(tarefas) > 0 {
		return domain.ErrTarefasPendentes
	}
	return nil
}

// tarefasPara devolve as tarefas do pedido para quem atende, marcando as
// que são da equipe de quem consulta.
func (s *Service) tarefasPara(ctx context.Context, identity auth.Identity, id uuid.UUID) ([]domain.Tarefa, error) {
	tarefas, err := s.repo.ListTarefas(ctx, s.pool, id)
	if err != nil || len(tarefas) == 0 {
		return tarefas, err
	}
	equipes := []uuid.UUID{}
	for _, t := range tarefas {
		if t.EquipeID != nil && !slices.Contains(equipes, *t.EquipeID) {
			equipes = append(equipes, *t.EquipeID)
		}
	}
	minhas, err := s.repo.MembroDe(ctx, s.pool, identity.UserID, equipes)
	if err != nil {
		return nil, err
	}
	for i, t := range tarefas {
		tarefas[i].Minha = t.EquipeID != nil && slices.Contains(minhas, *t.EquipeID)
	}
	return tarefas, nil
}
