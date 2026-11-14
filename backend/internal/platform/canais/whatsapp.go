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
	ProvedorGove      = "gove"
)

// GoveAPI é a API V2 da Gove Digital, o mesmo provedor do Protocolo Digital.
var GoveAPI = "https://api.gove.digital/v2"

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
	// Gove: client credentials e o modelo (template_slug). O segredo não sai.
	GoveURL        string `json:"gove_url"`
	GoveClientID   string `json:"gove_client_id"`
	GoveSegredo    string `json:"-"`
	TemSegredoGove bool   `json:"tem_segredo_gove"`
	GoveModelo     string `json:"gove_modelo"`
	// RemoverSegredoGove apaga o segredo gravado.
	RemoverSegredoGove bool `json:"-"`
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
	w.GoveURL = strings.TrimRight(strings.TrimSpace(w.GoveURL), "/")
	if w.Provedor == ProvedorGove && w.GoveURL == "" {
		w.GoveURL = GoveAPI
	}
	w.GoveClientID = strings.TrimSpace(w.GoveClientID)
	w.GoveModelo = strings.TrimSpace(w.GoveModelo)
	return w
}

// Validar confere a configuração. Cada campo preenchido segue o formato;
// ligado, o provedor escolhido precisa estar completo.
func (w WhatsApp) Validar() error {
	switch {
	case w.Provedor != ProvedorMeta && w.Provedor != ProvedorEvolution && w.Provedor != ProvedorGove:
		return InvalidaError{"o provedor do WhatsApp é meta, evolution ou gove"}
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
	case w.GoveURL != "" && !urlValida(w.GoveURL):
		return InvalidaError{"o endereço da Gove é http(s)://servidor"}
	case w.GoveModelo != "" && !metaModelo.MatchString(w.GoveModelo):
		return InvalidaError{"o modelo da Gove tem letras minúsculas, dígitos e _"}
	case utf8.RuneCountInString(w.GoveClientID) > 200:
		return InvalidaError{"o client id da Gove cabe em 200 caracteres"}
	case utf8.RuneCountInString(w.MetaToken) > 1000 || utf8.RuneCountInString(w.EvolutionChave) > 500 || utf8.RuneCountInString(w.MetaAppSecret) > 200 || utf8.RuneCountInString(w.GoveSegredo) > 500:
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
	case w.Receber && w.Provedor == ProvedorGove:
		return InvalidaError{"a Gove envia o aviso; a resposta do contribuinte entra pela Meta ou pela Evolution"}
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
	if w.Provedor == ProvedorEvolution {
		if w.EvolutionURL == "" || w.EvolutionInstancia == "" || (w.EvolutionChave == "" && !w.TemChave) {
			return errors.New("informe o endereço, a instância e a chave da Evolution API")
		}
		return nil
	}
	if w.GoveClientID == "" || w.GoveModelo == "" || (w.GoveSegredo == "" && !w.TemSegredoGove) {
		return errors.New("informe o client id, o segredo e o modelo da Gove")
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
	if w.Provedor == ProvedorEvolution {
		return postar(ctx, w.EvolutionURL+"/message/sendText/"+url.PathEscape(w.EvolutionInstancia),
			map[string]any{"number": numero, "text": a.texto()}, "apikey", w.EvolutionChave)
	}
	numero = destinatarioGove(numero)
	tok, err := tokenGove(ctx, w)
	if err != nil {
		return err
	}
	return postar(ctx, w.GoveURL+"/chats/messages/send-template", map[string]any{
		"template_slug": w.GoveModelo,
		"to":            numero,
		"variables":     []string{limitar(a.Titulo, 200), limitar(a.Mensagem, 700), a.Link},
	}, "Authorization", "Bearer "+tok)
}

// destinatarioGove manda só dígitos e prefixa 55 quando o número é local.
func destinatarioGove(numero string) string {
	var b strings.Builder
	for _, r := range numero {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digitos := b.String()
	if len(digitos) <= 11 && !strings.HasPrefix(digitos, "55") {
		return "55" + digitos
	}
	return digitos
}

func tokenGove(ctx context.Context, w WhatsApp) (string, error) {
	corpo, err := postarLendo(ctx, w.GoveURL+"/oauth/token", map[string]any{
		"grant_type": "client_credentials", "scope": "*", "client_id": w.GoveClientID, "client_secret": w.GoveSegredo,
	}, "", "")
	if err != nil {
		return "", err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(corpo, &tok); err != nil || tok.AccessToken == "" {
		return "", errors.New("canais: a Gove não devolveu access_token")
	}
	return tok.AccessToken, nil
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
	_, err := postarLendo(ctx, endereco, corpo, cabecalho, valor)
	return err
}

func postarLendo(ctx context.Context, endereco string, corpo any, cabecalho, valor string) ([]byte, error) {
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endereco, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cabecalho != "" {
		req.Header.Set(cabecalho, valor)
	}
	resp, err := clienteHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resposta, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode/100 == 2 {
		return resposta, nil
	}
	return nil, fmt.Errorf("canais: o provedor do WhatsApp respondeu %d: %s", resp.StatusCode, motivo(resposta))
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
