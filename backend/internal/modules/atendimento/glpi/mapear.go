// Package glpi importa o histórico do GLPI para o atendimento (ADR 072):
// categorias viram ofertas, chamados viram solicitações com a conversa e
// a avaliação. Lê a API REST do GLPI e grava direto no banco, sem avisar
// ninguém: o passado não toca o sino de quem atende.
package glpi

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

// ContaImportacao assina o que veio de uma pessoa que não existe aqui.
var ContaImportacao = uuid.MustParse("696d706f-7274-4163-a000-676c70690000")

// Origem é o nome da fonte na tabela de importação.
const Origem = "glpi"

// Estados e tipos do GLPI.
const (
	statusNovo        = 1
	statusAtribuido   = 2
	statusPlanejado   = 3
	statusPendente    = 4
	statusSolucionado = 5
	statusFechado     = 6

	tipoIncidente  = 1
	tipoRequisicao = 2
)

// Tipo traduz o tipo do chamado. O GLPI só tem os dois; o resto vira
// incidente, que é o caso comum.
func Tipo(t int) string {
	if t == tipoRequisicao {
		return domain.TipoRequisicao
	}
	return domain.TipoIncidente
}

// Estado traduz o status do GLPI. Pendente vira "aguardando terceiro":
// no GLPI ele é a espera genérica, e aqui não para o prazo de quem pediu
// sem explicação. Solucionado vira a conclusão do tipo.
func Estado(status, tipo int) string {
	switch status {
	case statusAtribuido, statusPlanejado:
		return domain.EstadoEmAtendimento
	case statusPendente:
		return domain.EstadoAguardandoTerceiro
	case statusSolucionado:
		return domain.Conclusao(Tipo(tipo))
	case statusFechado:
		return domain.EstadoEncerrado
	default:
		return domain.EstadoAberto
	}
}

// Nivel traduz urgência ou impacto do GLPI (1 muito baixo a 5 muito alto,
// 6 maior) para a escala daqui (1 baixo, 2 médio, 3 alto).
func Nivel(n int) int {
	switch {
	case n <= 2:
		return 1
	case n == 3:
		return 2
	default:
		return 3
	}
}

// Nota traduz a satisfação do GLPI (0 a 5) para a nota daqui (1 a 5).
func Nota(n int) int {
	return min(max(n, 1), 5)
}

var (
	tags    = regexp.MustCompile(`(?s)<[^>]*>`)
	quebras = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/tr)\s*/?>`)
	brancos = regexp.MustCompile(`[ \t\r\f\v]+`)
	linhas  = regexp.MustCompile(`\n{3,}`)
)

// Texto limpa o conteúdo do GLPI, que vem em HTML escapado, e corta no
// limite. As quebras de parágrafo viram linha; o resto das marcas some.
func Texto(s string, limite int) string {
	s = html.UnescapeString(s)
	s = quebras.ReplaceAllString(s, "\n")
	s = tags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = brancos.ReplaceAllString(s, " ")
	partes := strings.Split(s, "\n")
	for i, p := range partes {
		partes[i] = strings.TrimSpace(p)
	}
	s = linhas.ReplaceAllString(strings.Join(partes, "\n"), "\n\n")
	return Cortar(strings.TrimSpace(s), limite)
}

// Cortar limita em runas, marcando o corte com reticências.
func Cortar(s string, limite int) string {
	if utf8.RuneCountInString(s) <= limite {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:limite-1])) + "…"
}
