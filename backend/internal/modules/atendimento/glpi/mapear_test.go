package glpi

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

func TestTraducao(t *testing.T) {
	estados := map[[2]int]string{
		{1, 1}: domain.EstadoAberto,
		{2, 1}: domain.EstadoEmAtendimento,
		{3, 2}: domain.EstadoEmAtendimento,
		{4, 1}: domain.EstadoAguardandoTerceiro,
		{5, 1}: domain.EstadoResolvido,
		{5, 2}: domain.EstadoAtendido,
		{6, 2}: domain.EstadoEncerrado,
		{9, 1}: domain.EstadoAberto,
	}
	for k, espera := range estados {
		if got := Estado(k[0], k[1]); got != espera {
			t.Errorf("status %d tipo %d: %s", k[0], k[1], got)
		}
	}
	if Tipo(2) != domain.TipoRequisicao || Tipo(1) != domain.TipoIncidente || Tipo(0) != domain.TipoIncidente {
		t.Fatal("tipo")
	}
	for n, espera := range map[int]int{0: 1, 1: 1, 2: 1, 3: 2, 4: 3, 5: 3, 6: 3} {
		if Nivel(n) != espera {
			t.Errorf("nível %d", n)
		}
	}
	for n, espera := range map[int]int{0: 1, 1: 1, 3: 3, 5: 5, 9: 5} {
		if Nota(n) != espera {
			t.Errorf("nota %d", n)
		}
	}
}

func TestTexto(t *testing.T) {
	html := "&lt;p&gt;Bom dia,&lt;/p&gt;&lt;p&gt;o   &lt;b&gt;e-mail&lt;/b&gt; caiu &amp;amp; não volta.&lt;br /&gt;Obrigada&lt;/p&gt;&lt;p&gt;&lt;/p&gt;&lt;p&gt;&lt;/p&gt;&lt;p&gt;Ana&lt;/p&gt;"
	if got := Texto(html, 4000); got != "Bom dia,\no e-mail caiu & não volta.\nObrigada\n\nAna" {
		t.Fatalf("texto: %q", got)
	}
	longo := Texto(strings.Repeat("á", 300), 160)
	if utf8.RuneCountInString(longo) != 160 || !strings.HasSuffix(longo, "…") {
		t.Fatalf("corte: %d", utf8.RuneCountInString(longo))
	}
	if Cortar("curto", 10) != "curto" {
		t.Fatal("sem corte")
	}
}
