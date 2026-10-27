package domain

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAlertaDoMonitoramento(t *testing.T) {
	a := Alerta{
		Status: AlertaDisparando, Fingerprint: "abc123",
		Labels:      map[string]string{"alertname": "NovaDiscoCritico", "severity": "critical", "instance": "vm-240"},
		Annotations: map[string]string{"summary": " Disco da VM acima de 95% ", "description": "O Postgres pode parar."},
	}
	if err := a.Validar(); err != nil || a.Nome() != "NovaDiscoCritico" || !a.Critico() || a.Titulo() != "Disco da VM acima de 95%" {
		t.Fatalf("alerta: %v %q %q", err, a.Nome(), a.Titulo())
	}
	d := a.Descricao()
	// Os rótulos vão em ordem, depois da descrição.
	if !strings.HasPrefix(d, "O Postgres pode parar.\n\nAberto pelo monitoramento (alerta NovaDiscoCritico).") ||
		strings.Index(d, "- alertname") > strings.Index(d, "- instance") || strings.Index(d, "- instance") > strings.Index(d, "- severity") {
		t.Fatalf("descrição: %q", d)
	}

	semNada := Alerta{Status: AlertaNormalizou, Fingerprint: "f"}
	if semNada.Nome() != "Alerta" || semNada.Titulo() != "Alerta" || semNada.Critico() {
		t.Fatalf("sem rótulos: %q %q", semNada.Nome(), semNada.Titulo())
	}
	if semNada.Descricao() != "Aberto pelo monitoramento (alerta Alerta)." {
		t.Fatalf("descrição mínima: %q", semNada.Descricao())
	}

	longo := Alerta{Status: AlertaDisparando, Fingerprint: "f",
		Annotations: map[string]string{"summary": strings.Repeat("s", NomeMax*2), "description": strings.Repeat("d", TextoMax*2)}}
	if utf8.RuneCountInString(longo.Titulo()) > NomeMax || utf8.RuneCountInString(longo.Descricao()) > TextoMax {
		t.Fatalf("cabe no chamado: %d %d", utf8.RuneCountInString(longo.Titulo()), utf8.RuneCountInString(longo.Descricao()))
	}

	gigante := Alerta{Labels: map[string]string{"alertname": strings.Repeat("n", 10*TextoMax)}}
	if utf8.RuneCountInString(gigante.Nome()) > maxNome+1 {
		t.Fatalf("nome cortado: %d", utf8.RuneCountInString(gigante.Nome()))
	}

	for nome, ruim := range map[string]Alerta{
		"estado inventado":  {Status: "pending", Fingerprint: "f"},
		"sem fingerprint":   {Status: AlertaDisparando},
		"fingerprint longa": {Status: AlertaDisparando, Fingerprint: strings.Repeat("f", maxFingerprint+1)},
	} {
		if err := ruim.Validar(); !errors.Is(err, ErrAlertaInvalido) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}
