package application

import (
	"slices"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

func TestExtrasDaFilaTraduzemConcluida(t *testing.T) {
	incidente := domain.Solicitacao{Tipo: domain.TipoIncidente, Estado: domain.EstadoAberto}
	got := extrasDaFila(incidente, map[string][]string{"aberto": {"concluida", "c01234567"}})
	if !slices.Contains(got, domain.EstadoResolvido) || slices.Contains(got, "concluida") || !slices.Contains(got, "c01234567") {
		t.Fatalf("incidente: %v", got)
	}

	requisicao := domain.Solicitacao{Tipo: domain.TipoRequisicao, Estado: domain.EstadoAtendido}
	got = extrasDaFila(requisicao, map[string][]string{"concluida": {"c89abcdef"}})
	if !slices.Contains(got, "c89abcdef") {
		t.Fatalf("requisição concluída: %v", got)
	}
}
