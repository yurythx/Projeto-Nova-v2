package canais

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// Erros do webhook.
var (
	// ErrWebhookDesconhecido: token errado ou entrada desligada (404, sem
	// dizer qual).
	ErrWebhookDesconhecido = errors.New("canais: webhook desconhecido")
	// ErrAssinatura: a assinatura da Meta não confere com o App Secret.
	ErrAssinatura = errors.New("canais: assinatura do webhook inválida")
)

// Recebedor grava no atendimento a mensagem de quem respondeu e devolve o
// texto da confirmação (vazio: não responde). A porta fica no app.
type Recebedor interface {
	ReceberWhatsApp(ctx context.Context, usuario uuid.UUID, texto, mensagemID string) (string, error)
}

// WithRecebedor liga a entrada pelo WhatsApp ao atendimento.
func (s *Servico) WithRecebedor(r Recebedor) *Servico {
	s.recebedor = r
	return s
}

// WithAPIPublica é o endereço público da API, para montar o do webhook.
func (s *Servico) WithAPIPublica(base string) *Servico {
	s.apiPublica = strings.TrimRight(base, "/")
	return s
}

// webhookURL é o endereço que se cadastra no provedor.
func (s *Servico) webhookURL(w WhatsApp) string {
	if !w.Receber || w.WebhookToken == "" {
		return ""
	}
	return s.apiPublica + "/api/v1/canais/whatsapp/webhook/" + url.PathEscape(w.WebhookToken)
}

// novoToken gera o segredo do endereço do webhook.
func novoToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Recebida é uma mensagem de texto que chegou.
type Recebida struct {
	De, ID, Texto string
}

// conferir valida o token do endereço contra o gravado, em tempo constante.
func (s *Servico) conferir(ctx context.Context, token string) (WhatsApp, error) {
	w, err := s.WhatsApp(ctx)
	if err != nil {
		return w, err
	}
	if !w.Receber || w.WebhookToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(w.WebhookToken)) != 1 {
		return w, ErrWebhookDesconhecido
	}
	return w, nil
}

// Verificar responde ao desafio da Meta ao cadastrar o webhook.
func (s *Servico) Verificar(ctx context.Context, token, modo, verificacao, desafio string) (string, error) {
	if _, err := s.conferir(ctx, token); err != nil {
		return "", err
	}
	if modo != "subscribe" || subtle.ConstantTimeCompare([]byte(verificacao), []byte(token)) != 1 {
		return "", ErrAssinatura
	}
	return desafio, nil
}

// Webhook trata o que o provedor mandou: confere o token e, na Meta, a
// assinatura; depois grava cada mensagem de texto. Um erro ao gravar volta
// para o provedor tentar de novo; a reserva por mensagem evita repetir.
func (s *Servico) Webhook(ctx context.Context, token, assinatura string, corpo []byte) error {
	w, err := s.conferir(ctx, token)
	if err != nil {
		return err
	}
	var msgs []Recebida
	if w.Provedor == ProvedorMeta {
		if !assinaturaValida(w.MetaAppSecret, assinatura, corpo) {
			return ErrAssinatura
		}
		msgs = mensagensDaMeta(corpo)
	} else {
		msgs = mensagensDaEvolution(corpo)
	}
	for _, m := range msgs {
		if err := s.receber(ctx, w, m); err != nil {
			return err
		}
	}
	return nil
}

// assinaturaValida confere X-Hub-Signature-256 (sha256=HMAC do corpo).
func assinaturaValida(segredo, assinatura string, corpo []byte) bool {
	hexa, ok := strings.CutPrefix(assinatura, "sha256=")
	if !ok || segredo == "" {
		return false
	}
	recebida, err := hex.DecodeString(hexa)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(segredo))
	m.Write(corpo)
	return hmac.Equal(recebida, m.Sum(nil))
}

func mensagensDaMeta(corpo []byte) []Recebida {
	var p struct {
		Entry []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From string `json:"from"`
						ID   string `json:"id"`
						Type string `json:"type"`
						Text struct {
							Body string `json:"body"`
						} `json:"text"`
					} `json:"messages"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	_ = json.Unmarshal(corpo, &p)
	var out []Recebida
	for _, e := range p.Entry {
		for _, c := range e.Changes {
			for _, m := range c.Value.Messages {
				if m.Type == "text" {
					out = append(out, Recebida{De: m.From, ID: m.ID, Texto: m.Text.Body})
				}
			}
		}
	}
	return out
}

func mensagensDaEvolution(corpo []byte) []Recebida {
	var p struct {
		Event string `json:"event"`
		Data  struct {
			Key struct {
				RemoteJid string `json:"remoteJid"`
				FromMe    bool   `json:"fromMe"`
				ID        string `json:"id"`
			} `json:"key"`
			Message struct {
				Conversation        string `json:"conversation"`
				ExtendedTextMessage struct {
					Text string `json:"text"`
				} `json:"extendedTextMessage"`
			} `json:"message"`
		} `json:"data"`
	}
	if json.Unmarshal(corpo, &p) != nil || strings.ReplaceAll(strings.ToLower(p.Event), "_", ".") != "messages.upsert" {
		return nil
	}
	numero, dominio, _ := strings.Cut(p.Data.Key.RemoteJid, "@")
	// Só conversa individual: grupo, lista e status ficam de fora.
	if p.Data.Key.FromMe || dominio != "s.whatsapp.net" {
		return nil
	}
	texto := p.Data.Message.Conversation
	if texto == "" {
		texto = p.Data.Message.ExtendedTextMessage.Text
	}
	return []Recebida{{De: numero, ID: p.Data.Key.ID, Texto: texto}}
}

// variantes são as formas do número: no Brasil, o WhatsApp às vezes manda
// o celular sem o nono dígito, e a pessoa o cadastrou com ele.
func variantes(de string) []string {
	d := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, de)
	out := []string{"+" + d}
	if strings.HasPrefix(d, "55") {
		switch len(d) {
		case 12:
			out = append(out, "+"+d[:4]+"9"+d[4:])
		case 13:
			if d[4] == '9' {
				out = append(out, "+"+d[:4]+d[5:])
			}
		}
	}
	return out
}

// receber grava a mensagem de quem tem o número cadastrado e manda a
// confirmação. Número desconhecido ou mensagem vazia não são erro.
func (s *Servico) receber(ctx context.Context, w WhatsApp, m Recebida) error {
	if s.recebedor == nil || strings.TrimSpace(m.Texto) == "" || m.ID == "" {
		return nil
	}
	usuario, ok, err := s.store.UsuarioPorWhatsApp(ctx, variantes(m.De))
	if err != nil || !ok {
		return err
	}
	resposta, err := s.recebedor.ReceberWhatsApp(ctx, usuario, m.Texto, m.ID)
	if err != nil || resposta == "" {
		return err
	}
	// A confirmação vai dentro da janela de 24 horas que a mensagem abriu:
	// texto livre, sem modelo. Se falhar, a resposta já está gravada.
	_ = s.responderWA(ctx, w, m.De, resposta)
	return nil
}

// responderTexto manda texto livre (dentro da janela de 24 horas).
func responderTexto(ctx context.Context, w WhatsApp, para, texto string) error {
	numero := strings.TrimPrefix(para, "+")
	if w.Provedor == ProvedorMeta {
		return postar(ctx, MetaAPI+"/"+url.PathEscape(w.MetaNumeroID)+"/messages", map[string]any{
			"messaging_product": "whatsapp", "to": numero, "type": "text", "text": map[string]string{"body": texto},
		}, "Authorization", "Bearer "+w.MetaToken)
	}
	return postar(ctx, w.EvolutionURL+"/message/sendText/"+url.PathEscape(w.EvolutionInstancia),
		map[string]any{"number": numero, "text": texto}, "apikey", w.EvolutionChave)
}
