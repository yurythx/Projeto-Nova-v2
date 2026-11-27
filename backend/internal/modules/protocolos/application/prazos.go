package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// Eventos do calendário.
const (
	EventoFeriado        = "protocolos.feriado.salvo"
	EventoFeriadoRemovid = "protocolos.feriado.removido"
)

// IntervaloPrazos é de quanto em quanto o worker confere os prazos. O aviso
// ao responsável tem chave com a data do prazo: a mesma data não repete.
var IntervaloPrazos = 15 * time.Minute

// quemAge é o usuário da sessão; a rotina de prazos não tem sessão.
func quemAge(ctx context.Context) *uuid.UUID {
	quem, ok := auth.IdentityFromContext(ctx)
	if !ok || quem.UserID == uuid.Nil {
		return nil
	}
	return &quem.UserID
}

func (s *Service) calendario(ctx context.Context) (domain.Calendario, error) {
	lista, err := s.repo.ListarFeriados(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return domain.CalendarioDe(lista), nil
}

// exigirGestao pede protocolos:gerir em alguma unidade: o feriado é municipal.
func (s *Service) exigirGestao(ctx context.Context) error {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return apperrors.Unauthorized("sessão necessária")
	}
	c := auth.CoverageOf(id, auth.PermProtocolosGerir)
	if !c.All && len(c.Unidades) == 0 && len(c.Entidades) == 0 {
		return apperrors.Forbidden("o calendário pede protocolos:gerir")
	}
	return nil
}

// ListarFeriados devolve o calendário municipal a quem tem sessão.
func (s *Service) ListarFeriados(ctx context.Context) ([]domain.Feriado, error) {
	if _, ok := auth.IdentityFromContext(ctx); !ok {
		return nil, apperrors.Unauthorized("sessão necessária")
	}
	lista, err := s.repo.ListarFeriados(ctx, s.db)
	if lista == nil {
		lista = []domain.Feriado{}
	}
	return lista, err
}

// SalvarFeriado grava ou renomeia o feriado da data.
func (s *Service) SalvarFeriado(ctx context.Context, data, nome string) (domain.Feriado, error) {
	if err := s.exigirGestao(ctx); err != nil {
		return domain.Feriado{}, err
	}
	f, err := domain.NovoFeriado(data, nome)
	if err != nil {
		return domain.Feriado{}, err
	}
	err = s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SalvarFeriado(ctx, tx, f); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoFeriado, "protocolos_feriado", f.Data, nil, f, map[string]string{"data": f.Data})
	})
	return f, err
}

// RemoverFeriado tira a data do calendário.
func (s *Service) RemoverFeriado(ctx context.Context, data string) error {
	if err := s.exigirGestao(ctx); err != nil {
		return err
	}
	f, err := domain.NovoFeriado(data, "--")
	if err != nil {
		return err
	}
	return s.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.RemoverFeriado(ctx, tx, f.Data); err != nil {
			return err
		}
		return s.gravar(ctx, tx, EventoFeriadoRemovid, "protocolos_feriado", f.Data, map[string]string{"data": f.Data}, nil, map[string]string{"data": f.Data})
	})
}

// HorariosLivres são os horários da grade ainda não marcados no dia. Dia
// passado, fora da grade ou feriado não tem horário; hoje, só os que ainda vêm.
func (s *Service) HorariosLivres(ctx context.Context, servico uuid.UUID, dia string) ([]string, error) {
	serv, err := s.repo.BuscarServico(ctx, s.db, servico)
	if err != nil {
		return nil, err
	}
	if !serv.Ativo || !serv.ExigeAgendamento {
		return nil, domain.ErrNaoEncontrado
	}
	data, err := time.ParseInLocation(time.DateOnly, dia, domain.Cuiaba)
	if err != nil {
		return nil, domain.ErrInvalido
	}
	agora := s.agora().In(domain.Cuiaba)
	hoje := agora.Format(time.DateOnly)
	livres := []string{}
	if dia < hoje || !serv.Agenda.Cabe(data) {
		return livres, nil
	}
	cal, err := s.calendario(ctx)
	if err != nil {
		return nil, err
	}
	if cal.Feriado(data) {
		return livres, nil
	}
	tomadas, err := s.repo.ListarHoras(ctx, s.db, serv.ID, dia)
	if err != nil {
		return nil, err
	}
	agoraHora := agora.Format("15:04")
	for _, hora := range serv.Agenda.Grade() {
		if horaTomada(tomadas, hora) || (dia == hoje && hora <= agoraHora) {
			continue
		}
		livres = append(livres, hora)
	}
	return livres, nil
}

// PessoasDaUnidade lista quem pode ser o aprovador do serviço da unidade.
func (s *Service) PessoasDaUnidade(ctx context.Context, unidade uuid.UUID) ([]domain.Pessoa, error) {
	if err := s.exigir(ctx, auth.PermProtocolosGerir, unidade); err != nil {
		return nil, err
	}
	lista, err := s.repo.Pessoas(ctx, s.db, unidade)
	if lista == nil {
		lista = []domain.Pessoa{}
	}
	return lista, err
}

// RunPrazos confere os prazos até ctx acabar.
func (s *Service) RunPrazos(ctx context.Context) error {
	t := time.NewTicker(IntervaloPrazos)
	defer t.Stop()
	for {
		if err := s.VerificarPrazos(ctx, s.agora()); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "protocolos: verificação de prazos falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// VerificarPrazos cancela a complementação vencida sem resposta, lembra o
// cidadão na véspera do fim da complementação e avisa o responsável do
// protocolo que vence nas próximas 24 horas.
func (s *Service) VerificarPrazos(ctx context.Context, agora time.Time) error {
	vencidas, err := s.repo.ComplementacoesVencidas(ctx, s.db, agora)
	if err != nil {
		return err
	}
	for _, p := range vencidas {
		prox, err := p.ExpirarComplementacao(agora)
		if err != nil {
			continue
		}
		if _, err := s.persistir(ctx, p, prox, EventoCancelado, "cancelado", "pedido cancelado por falta de complementação",
			"o prazo da complementação venceu sem resposta", nil); err != nil {
			return err
		}
		if p.ResponsavelID != nil {
			s.avisarResponsavel(ctx, *p.ResponsavelID, prox, "Cancelado: o prazo da complementação venceu sem resposta.",
				"protocolos:complemento-vencido:"+p.ID.String())
		}
	}
	ate := agora.Add(24 * time.Hour)
	vencendo, err := s.repo.PrazosVencendo(ctx, s.db, agora, ate)
	if err != nil {
		return err
	}
	for _, p := range vencendo {
		if p.Status == domain.StatusComplementacao && dentro(p.ComplementoPrazo, agora, ate) && !p.ComplementoAvisado {
			s.avisar(ctx, nil, p.Email, p.Telefone, "protocolos:complemento-vencendo:"+p.ID.String(), "Pedido "+p.Numero,
				"O prazo para responder a complementação termina em "+p.ComplementoPrazo.In(domain.Cuiaba).Format("02/01 às 15:04")+
					". Responda pela consulta do número.")
			if err := s.repo.MarcarAvisoComplemento(ctx, s.db, p.ID); err != nil {
				return err
			}
		}
		if p.ResponsavelID != nil && dentro(p.PrazoEm, agora, ate) {
			s.avisarResponsavel(ctx, *p.ResponsavelID, p, "O prazo do protocolo vence em "+p.PrazoEm.In(domain.Cuiaba).Format("02/01")+".",
				"protocolos:prazo:"+p.ID.String()+":"+p.PrazoEm.UTC().Format(time.DateOnly))
		}
	}
	return nil
}

func dentro(t *time.Time, de, ate time.Time) bool {
	return t != nil && !t.Before(de) && !t.After(ate)
}
