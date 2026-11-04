package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

var padraoNotaCSAT = regexp.MustCompile(`(?i)^(?:nota\s*[:=]?\s*)?([1-5])\b(?:\s*(?:/\s*5|\*+|estrelas?))?(?:(?:\s*[-–—:]\s*|\s+)(.*))?$`)

func extrairNotaCSAT(texto string) (int, string, bool) {
	t := strings.TrimSpace(numeroCorreio.ReplaceAllString(texto, ""))
	t = strings.TrimSpace(t)
	if t == "" {
		return 0, "", false
	}
	m := padraoNotaCSAT.FindStringSubmatch(t)
	if m == nil {
		return 0, "", false
	}
	nota, err := strconv.Atoi(m[1])
	if err != nil || nota < 1 || nota > 5 {
		return 0, "", false
	}
	comentario := ""
	if len(m) > 2 {
		comentario = strings.TrimSpace(m[2])
	}
	return nota, comentario, true
}

// Respostas da entrada pelo WhatsApp: a pessoa sabe o que aconteceu com a
// mensagem que mandou.
const (
	respostaVarios = "Você tem mais de um chamado em andamento. Responda começando pelo número do chamado, por exemplo: #123 a impressora voltou."
)

// ReceberWhatsApp grava a mensagem de quem pediu (ADR 078) e devolve a
// confirmação. Com "#N", ou com um único chamado em andamento, vira
// comentário. Sem chamado, ou respondendo "novo", abre um chamado pelo
// mesmo caminho da Central. Quem atende responde pela central, onde há a
// nota interna. A mesma mensagem entra uma vez só.
func (s *Service) ReceberWhatsApp(ctx context.Context, usuario uuid.UUID, texto, mensagemID string) (domain.RespostaWhatsApp, error) {
	texto = strings.TrimSpace(texto)
	chave := "whatsapp:" + strings.TrimSpace(mensagemID)
	if texto == "" || chave == "whatsapp:" || utf8.RuneCountInString(chave) > 200 {
		return domain.RespostaWhatsApp{}, nil
	}
	if ja, err := s.repo.MensagemWhatsAppJaEntrou(ctx, s.pool, chave); err != nil || ja {
		return domain.RespostaWhatsApp{}, err
	}
	if strings.HasPrefix(texto, "#") || strings.Contains(strings.ToLower(texto), "[#") {
		return s.comentarWhatsApp(ctx, usuario, texto, chave)
	}
	aberta, tratada, err := s.abrirPeloWhatsApp(ctx, usuario, texto, chave)
	if err != nil || tratada {
		return aberta, err
	}
	sol, _, err := s.chamadoDaMensagem(ctx, usuario, texto)
	if err != nil || sol == nil {
		return domain.RespostaWhatsApp{}, err
	}
	return s.gravarComentarioWhatsApp(ctx, usuario, sol, texto, chave)
}

// comentarWhatsApp entra no chamado citado por "#N".
func (s *Service) comentarWhatsApp(ctx context.Context, usuario uuid.UUID, texto, chave string) (domain.RespostaWhatsApp, error) {
	sol, resposta, err := s.chamadoDaMensagem(ctx, usuario, texto)
	if err != nil || sol == nil {
		return soTexto(resposta), err
	}
	return s.gravarComentarioWhatsApp(ctx, usuario, sol, texto, chave)
}

func (s *Service) gravarComentarioWhatsApp(ctx context.Context, usuario uuid.UUID, sol *domain.Solicitacao, texto, chave string) (domain.RespostaWhatsApp, error) {
	reservou, err := s.repo.ReservarCorreio(ctx, s.pool, chave, sol.ID)
	if err != nil || !reservou {
		// Repetida: o provedor reentregou, e a confirmação já foi.
		return domain.RespostaWhatsApp{}, err
	}
	if sol.Estado == domain.EstadoResolvido || sol.Estado == domain.EstadoAtendido {
		if nota, comentario, ok := extrairNotaCSAT(texto); ok {
			if _, err := s.Avaliar(ctx, auth.Identity{UserID: usuario}, sol.ID, nota, comentario); err != nil {
				if errors.Is(err, domain.ErrAvaliacaoExistente) {
					return soTexto(fmt.Sprintf("O chamado #%d já foi avaliado anteriormente. Agradecemos sua colaboração!", sol.Numero)), nil
				}
				_ = s.repo.LiberarCorreio(ctx, s.pool, chave)
				return domain.RespostaWhatsApp{}, err
			}
			return soTexto(fmt.Sprintf("Agradecemos a sua avaliação (nota %d) para o chamado #%d! Sua opinião ajuda a aprimorar nossos serviços de TI.", nota, sol.Numero)), nil
		}
		// Concluído ainda não avaliado: a mensagem pede a nota, em vez de
		// entrar como comentário. Outro assunto começa com "novo".
		return soTexto(fmt.Sprintf("O chamado #%d foi concluído. Para avaliá-lo, envie uma nota de 1 a 5 (ex: #%d 5 Excelente). Para outro assunto, responda novo ou use a Central de Atendimento.", sol.Numero, sol.Numero)), nil
	}
	if _, err := s.Comentar(ctx, auth.Identity{UserID: usuario}, sol.ID, "Resposta pelo WhatsApp:\n"+texto, false); err != nil {
		switch {
		case errors.Is(err, domain.ErrSolicitacaoFinalizada):
			if sol.Estado == domain.EstadoResolvido || sol.Estado == domain.EstadoAtendido {
				return soTexto(fmt.Sprintf("O chamado #%d foi concluído. Para avaliá-lo, envie uma nota de 1 a 5 (ex: #%d 5 Excelente). Para outro assunto, responda novo ou use a Central de Atendimento.", sol.Numero, sol.Numero)), nil
			}
			return soTexto(fmt.Sprintf("O chamado #%d já foi encerrado. Para outro assunto, responda novo ou use a Central de Atendimento.", sol.Numero)), nil
		case errors.Is(err, domain.ErrComentarioInvalido):
			return domain.RespostaWhatsApp{}, nil
		case errors.Is(err, domain.ErrEspelho):
			return soTexto(fmt.Sprintf("O chamado #%d é atendido no GLPI até a mudança de sistema. Responda por lá.", sol.Numero)), nil
		}
		_ = s.repo.LiberarCorreio(ctx, s.pool, chave)
		return domain.RespostaWhatsApp{}, err
	}
	return soTexto(fmt.Sprintf("Recebemos a sua mensagem no chamado #%d. A equipe responde por aqui e na Central de Atendimento.", sol.Numero)), nil
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
		return nil, "Você não tem chamado em andamento. Escreva aqui para abrir um, ou use a Central de Atendimento.", nil
	case len(abertos) > 1:
		return nil, respostaVarios, nil
	}
	return &abertos[0], "", nil
}
