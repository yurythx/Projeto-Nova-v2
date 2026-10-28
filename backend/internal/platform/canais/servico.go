package canais

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/mail"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

// Validade é quanto a configuração lida fica em memória: a troca feita em
// outra réplica vale aqui em até esse tempo; na mesma, na hora.
var Validade = 30 * time.Second

// Servico entrega os avisos pelos canais e guarda a configuração.
type Servico struct {
	store    Store
	ambiente Email
	enviar   func(context.Context, mail.SMTP, mail.Mensagem) error
	enviarWA func(context.Context, WhatsApp, Aviso) error
	// responderWA manda a confirmação da mensagem recebida (texto livre).
	responderWA func(context.Context, WhatsApp, string, string) error
	recebedor   Recebedor
	apiPublica  string

	mu      sync.Mutex
	cache   *Email
	lido    time.Time
	cacheWA *WhatsApp
	lidoWA  time.Time
}

// NovoServico cria o serviço. ambiente é o SMTP das variáveis de ambiente,
// que vale enquanto ninguém configurou pela tela.
func NovoServico(store Store, ambiente Email) *Servico {
	ambiente = ambiente.Normalizar()
	ambiente.Ativo = ambiente.Host != ""
	ambiente.TemSenha = ambiente.Senha != ""
	ambiente.Origem = OrigemAmbiente
	if !ambiente.Ativo {
		ambiente = Email{Porta: 587, Origem: OrigemNenhuma}
	}
	return &Servico{store: store, ambiente: ambiente, enviarWA: enviarWhatsApp, responderWA: responderTexto, enviar: func(ctx context.Context, s mail.SMTP, m mail.Mensagem) error {
		return s.Send(ctx, m)
	}}
}

var _ notificacoes.Correio = (*Servico)(nil)

// Email devolve a configuração vigente: a da tela ou, sem ela, a do ambiente.
func (s *Servico) Email(ctx context.Context) (Email, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil && time.Since(s.lido) < Validade {
		return *s.cache, nil
	}
	e, ok, err := s.store.Email(ctx)
	if err != nil {
		return Email{}, err
	}
	if !ok {
		e = s.ambiente
	}
	s.cache, s.lido = &e, time.Now()
	return e, nil
}

// SalvarEmail valida e grava a configuração da tela.
func (s *Servico) SalvarEmail(ctx context.Context, e Email, removerSenha bool, por string) (Email, error) {
	e = e.Normalizar()
	if err := e.Validar(); err != nil {
		return Email{}, err
	}
	out, err := s.store.SalvarEmail(ctx, e, removerSenha, por)
	if err != nil {
		return Email{}, err
	}
	s.mu.Lock()
	s.cache, s.lido = &out, time.Now()
	s.mu.Unlock()
	return out, nil
}

// Send entrega o aviso por e-mail, se o canal está ligado. É o correio do
// sino: desligado, não é erro (o aviso fica no sino).
func (s *Servico) Send(ctx context.Context, m notificacoes.Mensagem) error {
	e, err := s.Email(ctx)
	if err != nil || !e.Ativo {
		return err
	}
	return s.enviar(ctx, smtpDe(e), mail.Mensagem{Para: m.Para, Assunto: m.Assunto, Corpo: m.Corpo})
}

// Testar manda uma mensagem de teste com a configuração gravada, ligada ou
// não: é assim que se confere antes de ligar.
func (s *Servico) Testar(ctx context.Context, para string) error {
	e, err := s.Email(ctx)
	if err != nil {
		return err
	}
	if e.Host == "" || e.Remetente == "" {
		return ErrSemServidor
	}
	if _, err := NormalizarContato(Contato{Email: para}); err != nil || para == "" {
		return InvalidaError{"informe o e-mail que recebe o teste"}
	}
	return s.enviar(ctx, smtpDe(e), mail.Mensagem{
		Para:    para,
		Assunto: "Teste do e-mail da Central de Atendimento",
		Corpo:   "Se você recebeu esta mensagem, o envio de e-mail da Central de Atendimento está funcionando.",
	})
}

func smtpDe(e Email) mail.SMTP {
	return mail.SMTP{Host: e.Host, Porta: e.Porta, Usuario: e.Usuario, Senha: e.Senha, De: e.Remetente}
}

// Contato devolve onde a pessoa recebe os avisos.
func (s *Servico) Contato(ctx context.Context, usuario uuid.UUID) (Contato, error) {
	return s.store.Contato(ctx, usuario)
}

// SalvarContato grava o contato (vazio também: a pessoa respondeu que não).
func (s *Servico) SalvarContato(ctx context.Context, usuario uuid.UUID, c Contato) (Contato, error) {
	c, err := NormalizarContato(c)
	if err != nil {
		return Contato{}, err
	}
	return s.store.SalvarContato(ctx, usuario, c)
}
