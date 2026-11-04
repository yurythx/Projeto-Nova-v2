package domain

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Etapas do roteiro de abertura pelo WhatsApp. Feito é o chamado que já
// nasceu quando não deu para apagar o roteiro na hora.
const (
	RoteiroOferta    = "oferta"
	RoteiroTitulo    = "titulo"
	RoteiroDescricao = "descricao"
	RoteiroCampo     = "campo"
	RoteiroFeito     = "feito"

	// OfertasPorPagina cabe numa lista da Meta (no máximo 10 linhas) e
	// ainda deixa a linha "Ver mais".
	OfertasPorPagina = 9
	// ValidadeRoteiroWhatsApp acompanha a janela em que a resposta livre
	// ainda pode sair. Depois disso a pessoa recomeça.
	ValidadeRoteiroWhatsApp = 24 * time.Hour
)

// OpcaoWhatsApp é uma escolha do roteiro. O mesmo id vale na lista da
// Meta e no número que a pessoa digita na Evolution.
type OpcaoWhatsApp struct {
	ID, Titulo, Descricao string
}

// RespostaWhatsApp é o que volta para quem escreveu. Sem opções, é texto.
type RespostaWhatsApp struct {
	Texto  string
	Opcoes []OpcaoWhatsApp
}

// RoteiroWhatsApp é a abertura em andamento de uma pessoa.
type RoteiroWhatsApp struct {
	UsuarioID   uuid.UUID
	Etapa       string
	OfertaID    *uuid.UUID
	Titulo      string
	Descricao   string
	Respostas   map[string]string
	CampoIndice int
	Pagina      int
	UpdatedAt   time.Time
}

// CamposObrigatorios são as perguntas que a abertura não pode pular.
// As opcionais ficam para a Central.
func CamposObrigatorios(campos []Campo) []Campo {
	out := make([]Campo, 0, len(campos))
	for _, c := range campos {
		if c.Obrigatorio {
			out = append(out, c)
		}
	}
	return out
}

// PaginaDeOfertas recorta o catálogo. temMais pede a linha "Ver mais".
// usada é a página de fato, porque um número alto demais volta ao início.
func PaginaDeOfertas(ofertas []Oferta, pagina int) ([]Oferta, bool, int) {
	if len(ofertas) == 0 {
		return nil, false, 0
	}
	if pagina < 0 || pagina*OfertasPorPagina >= len(ofertas) {
		pagina = 0
	}
	ini := pagina * OfertasPorPagina
	fim := ini + OfertasPorPagina
	if fim > len(ofertas) {
		fim = len(ofertas)
	}
	return ofertas[ini:fim], fim < len(ofertas), pagina
}

// EscolherOpcao aceita o número da lista, o id da Meta ou o título.
func EscolherOpcao(texto string, opcoes []OpcaoWhatsApp) (OpcaoWhatsApp, bool) {
	t := strings.TrimSpace(texto)
	if t == "" {
		return OpcaoWhatsApp{}, false
	}
	if n, err := strconv.Atoi(t); err == nil && n >= 1 && n <= len(opcoes) {
		return opcoes[n-1], true
	}
	for _, o := range opcoes {
		if strings.EqualFold(t, o.ID) || strings.EqualFold(t, o.Titulo) {
			return o, true
		}
	}
	return OpcaoWhatsApp{}, false
}

// RespostaDeCampo confere a resposta de um campo obrigatório da oferta.
func RespostaDeCampo(c Campo, texto string) (string, bool) {
	t := strings.TrimSpace(texto)
	if t == "" || utf8.RuneCountInString(t) > 400 {
		return "", false
	}
	switch c.Tipo {
	case "numero":
		for _, r := range t {
			if r < '0' || r > '9' {
				return "", false
			}
		}
		return t, true
	case "lista":
		for _, op := range c.Opcoes {
			if strings.EqualFold(op, t) {
				return op, true
			}
		}
		if n, err := strconv.Atoi(t); err == nil && n >= 1 && n <= len(c.Opcoes) {
			return c.Opcoes[n-1], true
		}
		return "", false
	default:
		return t, true
	}
}

// ComandoCancelar é desistir da abertura em andamento.
func ComandoCancelar(texto string) bool {
	switch strings.ToLower(strings.TrimSpace(texto)) {
	case "cancelar", "sair", "parar":
		return true
	default:
		return false
	}
}

// ComandoNovo começa outra abertura, mesmo com chamado em andamento.
func ComandoNovo(texto string) bool {
	return strings.EqualFold(strings.TrimSpace(texto), "novo")
}
