package application

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// IntervaloPrazos é de quanto em quanto o worker confere limite de tarefa
// e fim planejado do projeto. A chave do aviso inclui a data: a mesma
// data não repete; uma data nova avisa de novo.
var IntervaloPrazos = 15 * time.Minute

// RunPrazos confere os prazos até ctx acabar.
func (s *Service) RunPrazos(ctx context.Context) error {
	t := time.NewTicker(IntervaloPrazos)
	defer t.Stop()
	for {
		if err := s.VerificarPrazos(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil && s.logger != nil {
			s.logger.WarnContext(ctx, "projetos: verificação de prazos falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// VerificarPrazos avisa o responsável da tarefa e, se for outro, o gerente.
// O fim do projeto avisa o gerente. Sem destinatário, o prazo fica em silêncio.
func (s *Service) VerificarPrazos(ctx context.Context, agora time.Time) error {
	if s.avisos == nil {
		return nil
	}
	tarefas, projetos, err := s.repo.PrazosVencidos(ctx, s.pool, agora)
	if err != nil {
		return err
	}
	var enviadas []notificacoes.Enviada
	for _, t := range tarefas {
		destinos := destinosDaTarefa(t)
		if len(destinos) == 0 {
			continue
		}
		e, err := notificacoes.Registrar(ctx, s.pool, destinos, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Tarefa fora do prazo no projeto #" + strconv.FormatInt(t.ProjetoNumero, 10),
			Mensagem: cortar(t.Titulo+" · "+t.ProjetoTitulo, 1000),
			Link:     "/projetos/" + t.ProjetoID.String(),
			Chave:    "projetos:tarefa:" + t.TarefaID.String() + ":limite:" + t.DataLimite.UTC().Format(time.DateOnly),
		})
		if err != nil {
			return err
		}
		enviadas = append(enviadas, e...)
	}
	for _, p := range projetos {
		if p.GerenteID == nil || *p.GerenteID == uuid.Nil {
			continue
		}
		e, err := notificacoes.Registrar(ctx, s.pool, []uuid.UUID{*p.GerenteID}, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Fim planejado do projeto #" + strconv.FormatInt(p.Numero, 10) + " passou",
			Mensagem: cortar(p.Titulo, 1000),
			Link:     "/projetos/" + p.ID.String(),
			Chave:    "projetos:projeto:" + p.ID.String() + ":fim:" + p.Fim.UTC().Format(time.DateOnly),
		})
		if err != nil {
			return err
		}
		enviadas = append(enviadas, e...)
	}
	s.avisos.Entregar(ctx, enviadas)
	return nil
}

func destinosDaTarefa(t domain.PrazoTarefa) []uuid.UUID {
	if len(t.Pessoas) == 0 {
		return destinosDoPrazo(t.ResponsavelID, t.GerenteID)
	}
	visto := map[uuid.UUID]struct{}{}
	var out []uuid.UUID
	for _, id := range t.Pessoas {
		if id == uuid.Nil {
			continue
		}
		if _, ok := visto[id]; ok {
			continue
		}
		visto[id] = struct{}{}
		out = append(out, id)
	}
	if t.GerenteID != nil && *t.GerenteID != uuid.Nil {
		if _, ok := visto[*t.GerenteID]; !ok {
			out = append(out, *t.GerenteID)
		}
	}
	return out
}

func destinosDoPrazo(responsavel, gerente *uuid.UUID) []uuid.UUID {
	var out []uuid.UUID
	if responsavel != nil && *responsavel != uuid.Nil {
		out = append(out, *responsavel)
	}
	if gerente != nil && *gerente != uuid.Nil && (responsavel == nil || *gerente != *responsavel) {
		out = append(out, *gerente)
	}
	return out
}
