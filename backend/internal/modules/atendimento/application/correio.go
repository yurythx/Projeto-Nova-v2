package application

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
)

var numeroCorreio = regexp.MustCompile(`(?i)(?:\[#|#)(\d+)`)

// WithCaixa liga a entrada de e-mail. Sem caixa, o worker não faz nada.
func (s *Service) WithCaixa(c mail.Caixa) *Service {
	s.caixa = c
	return s
}

// ReceberCorreio grava a resposta de quem abriu ou de quem atende, uma vez
// por mensagem. Remetente desconhecido, número ausente ou chamado já
// encerrado não geram comentário e não são erro: a caixa pode seguir.
func (s *Service) ReceberCorreio(ctx context.Context, de, assunto, corpo, messageID string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" || utf8.RuneCountInString(messageID) > 200 {
		return nil
	}
	achado := numeroCorreio.FindStringSubmatch(assunto)
	if achado == nil {
		return nil
	}
	numero, err := strconv.ParseInt(achado[1], 10, 64)
	if err != nil {
		return nil
	}
	sol, err := s.repo.SolicitacaoPorNumero(ctx, s.pool, numero)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil
	}
	if err != nil {
		return err
	}
	de = strings.ToLower(strings.TrimSpace(de))
	emailSol, err := s.repo.EmailDe(ctx, s.pool, sol.SolicitanteID)
	if err != nil {
		return err
	}
	autor := sol.SolicitanteID
	if !strings.EqualFold(de, emailSol) {
		if sol.ResponsavelID == nil {
			return nil
		}
		emailResp, err := s.repo.EmailDe(ctx, s.pool, *sol.ResponsavelID)
		if err != nil {
			return err
		}
		if !strings.EqualFold(de, emailResp) {
			return nil
		}
		autor = *sol.ResponsavelID
	}
	reservou, err := s.repo.ReservarCorreio(ctx, s.pool, messageID, sol.ID)
	if err != nil || !reservou {
		return err
	}
	texto := strings.TrimSpace(corpo)
	if texto == "" {
		texto = "(mensagem sem texto)"
	}
	if _, err := s.Comentar(ctx, auth.Identity{UserID: autor}, sol.ID, "Resposta por e-mail:\n"+texto, false); err != nil {
		if errors.Is(err, domain.ErrSolicitacaoFinalizada) || errors.Is(err, domain.ErrComentarioInvalido) {
			return nil
		}
		_ = s.repo.LiberarCorreio(ctx, s.pool, messageID)
		return err
	}
	return nil
}

// RunCorreio lê a caixa e grava as respostas. Sem caixa, encerra.
func (s *Service) RunCorreio(ctx context.Context) error {
	if s.caixa == nil {
		return nil
	}
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for {
		if err := s.lerCorreio(ctx); err != nil && ctx.Err() == nil && s.logger != nil {
			s.logger.WarnContext(ctx, "atendimento: leitura do correio falhou", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (s *Service) lerCorreio(ctx context.Context) error {
	cartas, err := s.caixa.NaoLidas(ctx)
	if err != nil {
		return err
	}
	for _, c := range cartas {
		if err := s.ReceberCorreio(ctx, c.De, c.Assunto, c.Corpo, c.MessageID); err != nil {
			return err
		}
		if err := s.caixa.MarcarLida(ctx, c.UID); err != nil {
			return err
		}
	}
	return nil
}
