package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestLerClassificacao(t *testing.T) {
	// Modelos pequenos põem texto em volta do JSON.
	c, err := LerClassificacao("Claro! {\"impacto\": 3, \"urgencia\": 2, \"motivo\": \"  sistema do protocolo parado  \"} Espero ter ajudado.")
	if err != nil || c.Impacto != 3 || c.Urgencia != 2 || c.Motivo != "sistema do protocolo parado" {
		t.Fatalf("com texto em volta: %+v %v", c, err)
	}
	longo, err := LerClassificacao(`{"impacto":1,"urgencia":1,"motivo":"` + strings.Repeat("á", maxMotivo+50) + `"}`)
	if err != nil || utf8.RuneCountInString(longo.Motivo) != maxMotivo+1 || !strings.HasSuffix(longo.Motivo, "…") {
		t.Fatalf("motivo longo é cortado: %d %v", utf8.RuneCountInString(longo.Motivo), err)
	}
	for nome, texto := range map[string]string{
		"sem JSON":           "impacto alto",
		"chave invertida":    "} {",
		"JSON quebrado":      `{"impacto": 3,`,
		"impacto zero":       `{"impacto":0,"urgencia":2,"motivo":"x"}`,
		"impacto quatro":     `{"impacto":4,"urgencia":2,"motivo":"x"}`,
		"urgência zero":      `{"impacto":2,"urgencia":0,"motivo":"x"}`,
		"urgência quatro":    `{"impacto":2,"urgencia":4,"motivo":"x"}`,
		"sem motivo":         `{"impacto":2,"urgencia":2,"motivo":"   "}`,
		"nível em texto":     `{"impacto":"alto","urgencia":2,"motivo":"x"}`,
		"resposta em branco": "",
	} {
		if _, err := LerClassificacao(texto); !errors.Is(err, ErrRespostaDaIA) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func TestLerResumo(t *testing.T) {
	if r, err := LerResumo("\n  O e-mail parou.\nFalta testar.  \n"); err != nil || r != "O e-mail parou.\nFalta testar." {
		t.Fatalf("resumo: %q %v", r, err)
	}
	if _, err := LerResumo("   \n "); !errors.Is(err, ErrRespostaDaIA) {
		t.Fatalf("em branco: %v", err)
	}
	if r, _ := LerResumo(strings.Repeat("a", maxResumo+10)); utf8.RuneCountInString(r) != maxResumo+1 {
		t.Fatalf("resumo longo é cortado: %d", utf8.RuneCountInString(r))
	}
}

func TestDecidirSugestaoUmaVez(t *testing.T) {
	quem, agora := uuid.New(), time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s := Sugestao{ID: uuid.New()}
	aceita, err := s.Decidir(true, quem, agora)
	if err != nil || *aceita.Decisao != DecisaoAceita || *aceita.DecididaPor != quem || !aceita.DecididaEm.Equal(agora) {
		t.Fatalf("aceitar: %+v %v", aceita, err)
	}
	if s.Decisao != nil {
		t.Fatal("decidir devolve outra sugestão; a original fica como estava")
	}
	if _, err := aceita.Decidir(false, quem, agora); !errors.Is(err, ErrSugestaoDecidida) {
		t.Fatalf("segunda decisão: %v", err)
	}
	recusada, err := s.Decidir(false, quem, agora)
	if err != nil || *recusada.Decisao != DecisaoRecusada {
		t.Fatalf("recusar: %+v %v", recusada, err)
	}
}
