package canais

import (
	"context"
	"time"

	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

var _ notificacoes.WhatsApp = (*Servico)(nil)

// WhatsApp devolve a configuração vigente (com o mesmo prazo de cache do
// e-mail).
func (s *Servico) WhatsApp(ctx context.Context) (WhatsApp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cacheWA == nil || time.Since(s.lidoWA) >= Validade {
		w, _, err := s.store.WhatsApp(ctx)
		if err != nil {
			return WhatsApp{}, err
		}
		s.cacheWA, s.lidoWA = &w, time.Now()
	}
	w := *s.cacheWA
	w.WebhookURL = s.webhookURL(w)
	return w, nil
}

// SalvarWhatsApp valida e grava. Para ligar, vale a credencial já gravada,
// a menos que ela esteja sendo removida.
func (s *Servico) SalvarWhatsApp(ctx context.Context, w WhatsApp, removerToken, removerChave bool, por string) (WhatsApp, error) {
	atual, _, err := s.store.WhatsApp(ctx)
	if err != nil {
		return WhatsApp{}, err
	}
	w = w.Normalizar()
	w.TemToken = atual.TemToken && !removerToken
	w.TemChave = atual.TemChave && !removerChave
	w.TemAppSecret = atual.TemAppSecret
	if err := w.Validar(); err != nil {
		return WhatsApp{}, err
	}
	// O endereço do webhook nasce na primeira vez que a entrada liga; trocar
	// invalida o antigo.
	w.WebhookToken = ""
	if w.Receber && (atual.WebhookToken == "" || w.NovoWebhook) {
		w.WebhookToken = novoToken()
	}
	out, err := s.store.SalvarWhatsApp(ctx, w, removerToken, removerChave, por)
	if err != nil {
		return WhatsApp{}, err
	}
	s.mu.Lock()
	s.cacheWA, s.lidoWA = &out, time.Now()
	s.mu.Unlock()
	out.WebhookURL = s.webhookURL(out)
	return out, nil
}

// SendWhatsApp entrega o aviso, se o canal está ligado. É o WhatsApp do
// sino: desligado, não é erro.
func (s *Servico) SendWhatsApp(ctx context.Context, m notificacoes.MensagemWhatsApp) error {
	w, err := s.WhatsApp(ctx)
	if err != nil || !w.Ativo {
		return err
	}
	return s.enviarWA(ctx, w, Aviso(m))
}

// TestarWhatsApp manda um aviso de teste com a configuração gravada,
// ligada ou não. Na API oficial, vai pelo modelo aprovado.
func (s *Servico) TestarWhatsApp(ctx context.Context, para string) error {
	w, err := s.WhatsApp(ctx)
	if err != nil {
		return err
	}
	if w.completo() != nil {
		return ErrWhatsAppIncompleto
	}
	c, err := NormalizarContato(Contato{WhatsApp: para})
	if err != nil || c.WhatsApp == "" {
		return InvalidaError{"informe o WhatsApp que recebe o teste, com DDD"}
	}
	return s.enviarWA(ctx, w, Aviso{
		Para:     c.WhatsApp,
		Titulo:   "Teste da Central de Atendimento",
		Mensagem: "Se você recebeu esta mensagem, o WhatsApp dos avisos está funcionando.",
		Link:     "",
	})
}
