package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ContaMonitoramento é a conta de sistema que abre o incidente do alerta
// (ADR 067; criada pela migration 000022, inativa e sem senha).
var ContaMonitoramento = uuid.MustParse("6d6f6e69-746f-4072-a000-6d656e746f00")

// Estados do alerta no webhook do Alertmanager.
const (
	AlertaDisparando = "firing"
	AlertaNormalizou = "resolved"

	maxFingerprint = 128
	maxNome        = 120
)

var (
	// ErrEventosSemOferta: ninguém escolheu a oferta dos alertas.
	ErrEventosSemOferta = errors.New("atendimento: escolha a oferta em que os alertas abrem incidente")
	// ErrAlertaInvalido: o webhook trouxe alerta sem estado ou sem fingerprint.
	ErrAlertaInvalido = errors.New("atendimento: alerta fora do formato do Alertmanager")
)

// Alerta é um alerta do webhook do Alertmanager (versão 4).
type Alerta struct {
	Status      string            `json:"status"`
	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	// GeneratorURL aponta para o gráfico do alerta; não entra no chamado.
	GeneratorURL string `json:"generatorURL"`
}

// Validar confere o mínimo para deduplicar: estado conhecido e fingerprint.
func (a Alerta) Validar() error {
	if (a.Status != AlertaDisparando && a.Status != AlertaNormalizou) ||
		a.Fingerprint == "" || utf8.RuneCountInString(a.Fingerprint) > maxFingerprint {
		return ErrAlertaInvalido
	}
	return nil
}

// Nome é o alertname, ou "Alerta" quando não vem. Cortado: entra no
// título e nas notas, que têm limite, e um alerta que nunca cabe seria
// reenviado pelo Alertmanager para sempre.
func (a Alerta) Nome() string {
	if n := strings.TrimSpace(a.Labels["alertname"]); n != "" {
		return cortar(n, maxNome)
	}
	return "Alerta"
}

// Critico diz se o alerta pede impacto e urgência altos.
func (a Alerta) Critico() bool { return a.Labels["severity"] == "critical" }

// Titulo é o resumo do alerta, ou o nome, no limite do chamado.
func (a Alerta) Titulo() string {
	t := strings.TrimSpace(a.Annotations["summary"])
	if t == "" {
		t = a.Nome()
	}
	return cortar(t, NomeMax-1)
}

// Descricao junta a descrição do alerta e os rótulos, em ordem, no limite
// do chamado. Só o que o alerta trouxe entra.
func (a Alerta) Descricao() string {
	var b strings.Builder
	if d := strings.TrimSpace(a.Annotations["description"]); d != "" {
		b.WriteString(d)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Aberto pelo monitoramento (alerta %s).", a.Nome())
	chaves := make([]string, 0, len(a.Labels))
	for k := range a.Labels {
		chaves = append(chaves, k)
	}
	slices.Sort(chaves)
	if len(chaves) > 0 {
		b.WriteString("\nRótulos:")
		for _, k := range chaves {
			fmt.Fprintf(&b, "\n- %s: %s", k, a.Labels[k])
		}
	}
	return cortar(b.String(), TextoMax-1)
}

// EventoVinculado liga a fingerprint ao chamado que ela abriu.
type EventoVinculado struct {
	Fingerprint   string
	SolicitacaoID uuid.UUID
	Alerta        string
	UltimoDisparo time.Time
}
