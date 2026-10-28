package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// Respostas da entrada pelo WhatsApp: a pessoa sabe o que aconteceu com a
// mensagem que mandou.
const (
	respostaSemChamado = "Você não tem chamado em andamento. Para abrir um, use a Central de Atendimento."
	respostaVarios     = "Você tem mais de um chamado em andamento. Responda começando pelo número do chamado, por exemplo: #123 a impressora voltou."
)

// ReceberWhatsApp grava como comentário a mensagem de quem pediu e
// respondeu pelo WhatsApp (ADR 078), e devolve a confirmação. Vale o
// chamado do "#N" da mensagem, se foi ela quem abriu; sem número, o único em
// andamento que ela abriu. Quem atende responde pela central, onde há a nota
// interna. A mesma mensagem entra uma vez só (a reserva do correio).
func (s *Service) ReceberWhatsApp(ctx context.Context, usuario uuid.UUID, texto, mensagemID string) (string, error) {
	texto = strings.TrimSpace(texto)
	chave := "whatsapp:" + strings.TrimSpace(mensagemID)
	if texto == "" || chave == "whatsapp:" || utf8.RuneCountInString(chave) > 200 {
		return "", nil
	}
	sol, resposta, err := s.chamadoDaMensagem(ctx, usuario, texto)
	if err != nil || sol == nil {
		return resposta, err
	}
	reservou, err := s.repo.ReservarCorreio(ctx, s.pool, chave, sol.ID)
	if err != nil || !reservou {
		// Repetida: o provedor reentregou, e a confirmação já foi.
		return "", err
	}
	if _, err := s.Comentar(ctx, auth.Identity{UserID: usuario}, sol.ID, "Resposta pelo WhatsApp:\n"+texto, false); err != nil {
		switch {
		case errors.Is(err, domain.ErrSolicitacaoFinalizada):
			return fmt.Sprintf("O chamado #%d já foi encerrado. Para outro assunto, abra um novo chamado na Central de Atendimento.", sol.Numero), nil
		case errors.Is(err, domain.ErrComentarioInvalido):
			return "", nil
		}
		_ = s.repo.LiberarCorreio(ctx, s.pool, chave)
		return "", err
	}
	return fmt.Sprintf("Recebemos a sua mensagem no chamado #%d. A equipe responde por aqui e na Central de Atendimento.", sol.Numero), nil
}

// chamadoDaMensagem acha o chamado; sem ele, devolve o que dizer à pessoa.
func (s *Service) chamadoDaMensagem(ctx context.Context, usuario uuid.UUID, texto string) (*domain.Solicitacao, string, error) {
	if achado := numeroCorreio.FindStringSubmatch(texto); achado != nil {
		numero, err := strconv.ParseInt(achado[1], 10, 64)
		if err != nil {
			return nil, "", nil
		}
		sol, err := s.repo.SolicitacaoPorNumero(ctx, s.pool, numero)
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return nil, fmt.Sprintf("Não encontramos o chamado #%d.", numero), nil
		}
		if err != nil {
			return nil, "", err
		}
		if sol.ResponsavelID != nil && *sol.ResponsavelID == usuario {
			return nil, fmt.Sprintf("Para o chamado #%d, responda pela Central de Atendimento: lá fica também a nota interna.", numero), nil
		}
		if sol.SolicitanteID != usuario {
			return nil, fmt.Sprintf("O chamado #%d não é seu. Confira o número.", numero), nil
		}
		return &sol, "", nil
	}
	abertos, _, err := s.repo.ListSolicitacoes(ctx, s.pool, domain.FiltroSolicitacao{Solicitante: &usuario, Limit: 2})
	switch {
	case err != nil:
		return nil, "", err
	case len(abertos) == 0:
		return nil, respostaSemChamado, nil
	case len(abertos) > 1:
		return nil, respostaVarios, nil
	}
	return &abertos[0], "", nil
}
