package nome

import "testing"

func TestLimpar(t *testing.T) {
	casos := []struct{ primeiro, ultimo, esperado string }{
		{"Ana", "Souza", "Ana Souza"},
		{"Ana Souza", "", "Ana Souza"},
		{"YURI MIRANDA DE", "MIRANDA DE MENEZES", "YURI MIRANDA DE MENEZES"},
		{"ALEX WILLYAN RAMOS TEIXEIRA", "WILLYAN RAMOS TEIXEIRA SILVA", "ALEX WILLYAN RAMOS TEIXEIRA SILVA"},
		{"LAURO FRANK DA SILVA", "DA SILVA TEIXEIRA", "LAURO FRANK DA SILVA TEIXEIRA"},
		{"JENNIFER EVELIN", "EVELIN", "JENNIFER EVELIN"},
		{"DAVI RICARDO GONCALVES", "RICARDO GONCALVES FOGACA", "DAVI RICARDO GONCALVES FOGACA"},
		{"LEONARDO GABRIEL OLIVEIRA DE", "GABRIEL OLIVEIRA DE ALMEIDA", "LEONARDO GABRIEL OLIVEIRA DE ALMEIDA"},
		{"LUCAS DOS SANTOS", "DOS SANTOS SETE", "LUCAS DOS SANTOS SETE"},
		{"BRUNO MARQUES DA", "MARQUES DA SILVA", "BRUNO MARQUES DA SILVA"},
		{"", "", ""},
	}
	for _, c := range casos {
		if got := Limpar(c.primeiro, c.ultimo); got != c.esperado {
			t.Errorf("Limpar(%q, %q) = %q, esperado %q", c.primeiro, c.ultimo, got, c.esperado)
		}
	}
}

func TestSemRepeticaoDoQueJaFoiGravado(t *testing.T) {
	casos := map[string]string{
		"Ana Souza":                          "Ana Souza",
		"YURI MIRANDA DE MIRANDA DE MENEZES": "YURI MIRANDA DE MENEZES",
		"ALEX WILLYAN RAMOS TEIXEIRA WILLYAN RAMOS TEIXEIRA SILVA": "ALEX WILLYAN RAMOS TEIXEIRA SILVA",
		"LAURO FRANK DA SILVA DA SILVA TEIXEIRA":                   "LAURO FRANK DA SILVA TEIXEIRA",
		"JENNIFER EVELIN EVELIN":                                   "JENNIFER EVELIN",
		"DAVI RICARDO GONCALVES RICARDO GONCALVES FOGACA":          "DAVI RICARDO GONCALVES FOGACA",
		"LEONARDO GABRIEL OLIVEIRA DE GABRIEL OLIVEIRA DE ALMEIDA": "LEONARDO GABRIEL OLIVEIRA DE ALMEIDA",
		"LUCAS DOS SANTOS DOS SANTOS SETE":                         "LUCAS DOS SANTOS SETE",
		"BRUNO MARQUES DA MARQUES DA SILVA":                        "BRUNO MARQUES DA SILVA",
	}
	for in, esperado := range casos {
		if got := SemRepeticao(in); got != esperado {
			t.Errorf("SemRepeticao(%q) = %q, esperado %q", in, got, esperado)
		}
	}
}
