package canais

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Provedores do WhatsApp (ADR 078): um ativo de cada vez.
const (
	ProvedorMeta      = "meta"
	ProvedorEvolution = "evolution"
)

// MetaAPI é a Graph API da Meta (Cloud API do WhatsApp).
var MetaAPI = "https://graph.facebook.com/v21.0"

// ErrWhatsAppIncompleto: o teste pede o provedor configurado.
var ErrWhatsAppIncompleto = errors.New("canais: configure o provedor do WhatsApp antes de testar")

// WhatsApp é a configuração do canal. As credenciais nunca saem:
// TemToken e TemChave dizem se há.
type WhatsApp struct {
	Ativo    bool   `json:"ativo"`
	Provedor string `json:"provedor"`
	// Meta: o número de origem, o token e o modelo aprovado de 3 variáveis
	// (título, mensagem e link). Fora da janela de 24 horas, a Meta só
	// aceita modelo.
	MetaNumeroID string `json:"meta_numero_id"`
	MetaToken    string `json:"-"`
	TemToken     bool   `json:"tem_token"`
	MetaModelo   string `json:"meta_modelo"`
	MetaIdioma   string `json:"meta_idioma"`
	// Evolution: o endereço da API, a instância e a chave.
	EvolutionURL       string `json:"evolution_url"`
	EvolutionInstancia string `json:"evolution_instancia"`
	EvolutionChave     string `json:"-"`
	TemChave           bool   `json:"tem_chave"`
	// Entrada (fase 4): a resposta pelo WhatsApp vira comentário. O
	// webhook só aceita o token do endereço; na Meta, também a assinatura
	// com o App Secret.
	Receber       bool   `json:"receber"`
	WebhookToken  string `json:"-"`
	WebhookURL    string `json:"webhook_url,omitempty"`
	MetaAppSecret string `json:"-"`
	// NovoWebhook pede outro token (o endereço antigo deixa de valer).
	NovoWebhook  bool       `json:"-"`
	TemAppSecret bool       `json:"tem_app_secret"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	UpdatedBy    string     `json:"updated_by,omitempty"`
}

var (
	metaNumeroID = regexp.MustCompile(`^[0-9]{5,32}$`)
	metaModelo   = regexp.MustCompile(`^[a-z0-9_]{1,512}$`)
	metaIdioma   = regexp.MustCompile(`^[a-z]{2,3}(_[A-Z]{2})?$`)
	instancia    = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
)

// Normalizar apara os campos e põe os padrões.
func (w WhatsApp) Normalizar() WhatsApp {
	w.Provedor = strings.TrimSpace(w.Provedor)
	if w.Provedor == "" {
		w.Provedor = ProvedorMeta
	}
	w.MetaNumeroID, w.MetaModelo = strings.TrimSpace(w.MetaNumeroID), strings.TrimSpace(w.MetaModelo)
	w.MetaIdioma = strings.TrimSpace(w.MetaIdioma)
	if w.MetaIdioma == "" {
		w.MetaIdioma = "pt_BR"
	}
	w.EvolutionURL = strings.TrimRight(strings.TrimSpace(w.EvolutionURL), "/")
	w.EvolutionInstancia = strings.TrimSpace(w.EvolutionInstancia)
	return w
}

// Validar confere a configuração. Cada campo preenchido segue o formato;
// ligado, o provedor escolhido precisa estar completo.
func (w WhatsApp) Validar() error {
	switch {
	case w.Provedor != ProvedorMeta && w.Provedor != ProvedorEvolution:
		return InvalidaError{"o provedor do WhatsApp é meta ou evolution"}
	case w.MetaNumeroID != "" && !metaNumeroID.MatchString(w.MetaNumeroID):
		return InvalidaError{"o identificador do número (Phone number ID) da Meta é só de dígitos"}
	case w.MetaModelo != "" && !metaModelo.MatchString(w.MetaModelo):
		return InvalidaError{"o nome do modelo da Meta tem letras minúsculas, dígitos e _"}
	case !metaIdioma.MatchString(w.MetaIdioma):
		return InvalidaError{"o idioma do modelo é como pt_BR"}
	case w.EvolutionURL != "" && !urlValida(w.EvolutionURL):
		return InvalidaError{"o endereço da Evolution API é http(s)://servidor[:porta]"}
	case w.EvolutionInstancia != "" && !instancia.MatchString(w.EvolutionInstancia):
		return InvalidaError{"a instância da Evolution tem letras, dígitos, ponto, hífen e _"}
	case utf8.RuneCountInString(w.MetaToken) > 1000 || utf8.RuneCountInString(w.EvolutionChave) > 500 || utf8.RuneCountInString(w.MetaAppSecret) > 200:
		return InvalidaError{"o token ou a chave está longo demais"}
	}
	if w.Ativo {
		if err := w.completo(); err != nil {
			return InvalidaError{"para ligar o WhatsApp, " + err.Error()}
		}
	}
	switch {
	case w.Receber && !w.Ativo:
		return InvalidaError{"para receber respostas, ligue também o envio: a confirmação volta pelo WhatsApp"}
	case w.Receber && w.Provedor == ProvedorMeta && w.MetaAppSecret == "" && !w.TemAppSecret:
		return InvalidaError{"para receber pela API da Meta, informe o App Secret: é com ele que se confere a assinatura"}
	}
	return nil
}

// completo diz o que falta ao provedor escolhido.
func (w WhatsApp) completo() error {
	if w.Provedor == ProvedorMeta {
		if w.MetaNumeroID == "" || w.MetaModelo == "" || (w.MetaToken == "" && !w.TemToken) {
			return errors.New("informe o identificador do número, o token e o modelo aprovado da Meta")
		}
		return nil
	}
	if w.EvolutionURL == "" || w.EvolutionInstancia == "" || (w.EvolutionChave == "" && !w.TemChave) {
		return errors.New("informe o endereço, a instância e a chave da Evolution API")
	}
	return nil
}

func urlValida(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.RawQuery == ""
}

// Aviso é o que vai pelo WhatsApp: as três partes do modelo da Meta, ou o
// texto montado com elas na Evolution.
type Aviso struct {
	Para, Titulo, Mensagem, Link string
}

// texto é o aviso em texto livre (Evolution).
func (a Aviso) texto() string {
	t := "*" + a.Titulo + "*\n" + a.Mensagem
	if a.Link != "" {
		t += "\n\nAcompanhe: " + a.Link
	}
	return t
}

// clienteHTTP fala com os provedores, com tempo limite curto: o aviso é
// de melhor esforço e não pode segurar quem agiu.
var clienteHTTP = &http.Client{Timeout: 10 * time.Second}

// enviarWhatsApp manda o aviso pelo provedor configurado.
func enviarWhatsApp(ctx context.Context, w WhatsApp, a Aviso) error {
	numero := strings.TrimPrefix(a.Para, "+")
	if w.Provedor == ProvedorMeta {
		corpo := map[string]any{
			"messaging_product": "whatsapp", "to": numero, "type": "template",
			"template": map[string]any{
				"name": w.MetaModelo, "language": map[string]string{"code": w.MetaIdioma},
				"components": []map[string]any{{"type": "body", "parameters": []map[string]string{
					{"type": "text", "text": limitar(a.Titulo, 200)},
					{"type": "text", "text": limitar(a.Mensagem, 700)},
					{"type": "text", "text": a.Link},
				}}},
			},
		}
		return postar(ctx, MetaAPI+"/"+url.PathEscape(w.MetaNumeroID)+"/messages", corpo, "Authorization", "Bearer "+w.MetaToken)
	}
	return postar(ctx, w.EvolutionURL+"/message/sendText/"+url.PathEscape(w.EvolutionInstancia),
		map[string]any{"number": numero, "text": a.texto()}, "apikey", w.EvolutionChave)
}

// limitar prepara o parâmetro do modelo: a Meta recusa quebra de linha,
// tabulação e mais de quatro espaços seguidos, e cada um tem limite.
func limitar(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func postar(ctx context.Context, endereco string, corpo any, cabecalho, valor string) error {
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endereco, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(cabecalho, valor)
	resp, err := clienteHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 == 2 {
		return nil
	}
	resposta, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("canais: o provedor do WhatsApp respondeu %d: %s", resp.StatusCode, motivo(resposta))
}

// motivo tira da resposta de erro a mensagem que ajuda (Meta e Evolution
// usam formatos diferentes); sem ela, o texto cru.
func motivo(b []byte) string {
	var r struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Response struct {
			Message any `json:"message"`
		} `json:"response"`
	}
	if json.Unmarshal(b, &r) == nil {
		if r.Error.Message != "" {
			return r.Error.Message
		}
		if r.Response.Message != nil {
			m, _ := json.Marshal(r.Response.Message)
			return string(m)
		}
	}
	return strings.TrimSpace(string(b))
}
