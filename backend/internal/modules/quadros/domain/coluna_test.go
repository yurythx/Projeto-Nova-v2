package domain

import (
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func fabrica() []Coluna {
	return []Coluna{
		{Quadro: "problemas", Chave: "registrado", Nome: "Registrados", Ordem: 0, Entrada: true, Ativa: true},
		{Quadro: "problemas", Chave: "em_analise", Nome: "Em análise", Ordem: 1, Ativa: true},
		{Quadro: "problemas", Chave: "encerrado", Nome: "Encerrados", Ordem: 2, Ativa: true},
	}
}

func str(s string) *string { return &s }
func n(v int) *int         { return &v }
func b(v bool) *bool       { return &v }

func chaveDe(cols []Coluna, chave string) Coluna {
	for _, c := range cols {
		if c.Chave == chave {
			return c
		}
	}
	return Coluna{}
}

func TestCriarColunaSoNesteQuadro(t *testing.T) {
	pronta, passagens, out, err := Criar(fabrica(), "c01234567", Nova{
		Nome: "Aguardando fornecedor", Cor: "#2563eb", Ordem: 1,
		De: []string{"em_analise"}, Para: []string{"encerrado"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pronta.Chave != "c01234567" || pronta.Entrada || pronta.Ordem != 1 {
		t.Fatalf("coluna = %+v", pronta)
	}
	if chaveDe(out, "registrado").Ordem != 0 || chaveDe(out, "em_analise").Ordem != 2 {
		t.Fatalf("ordem = %+v", Sequencia(out))
	}
	if len(passagens) != 2 || passagens[0].De != "em_analise" || passagens[0].Para != "c01234567" || passagens[1].Para != "encerrado" {
		t.Fatalf("passagens = %+v", passagens)
	}
	if _, _, _, err := Criar(fabrica(), "c01234567", Nova{Nome: "Sem entrada", Cor: "#2563eb", De: nil}); err != ErrSemEntrada {
		t.Fatalf("sem entrada: %v", err)
	}
	if _, _, _, err := Criar(fabrica(), "c01234567", Nova{Nome: "Do fim", Cor: "#2563eb", De: []string{"encerrado"}}); err != ErrPassagem {
		t.Fatalf("terminal: %v", err)
	}
}

func TestRenomearNaoTrocaAChave(t *testing.T) {
	out, err := Aplicar(fabrica(), "em_analise", 0, Patch{Nome: str("Triagem")})
	if err != nil {
		t.Fatal(err)
	}
	col := chaveDe(out, "em_analise")
	if col.Nome != "Triagem" || col.Chave != "em_analise" || col.Ordem != 1 {
		t.Fatalf("coluna = %+v", col)
	}
}

func TestOrdemContinuaDoZero(t *testing.T) {
	out, err := Aplicar(fabrica(), "encerrado", 0, Patch{Ordem: n(0)})
	if err != nil {
		t.Fatal(err)
	}
	if got := Sequencia(out); len(got) != 3 || got[0] != "encerrado" || got[1] != "registrado" || got[2] != "em_analise" {
		t.Fatalf("sequencia = %v", got)
	}
	for i, chave := range Sequencia(out) {
		if chaveDe(out, chave).Ordem != i {
			t.Fatalf("ordem de %s = %d", chave, chaveDe(out, chave).Ordem)
		}
	}
}

func TestNaoRetiraAColunaInicial(t *testing.T) {
	_, err := Aplicar(fabrica(), "registrado", 0, Patch{Ativa: b(false)})
	if err != ErrInicial {
		t.Fatalf("err = %v", err)
	}
}

func TestNaoRetiraColunaComRegistro(t *testing.T) {
	_, err := Aplicar(fabrica(), "em_analise", 2, Patch{Ativa: b(false)})
	if err != ErrComRegistro {
		t.Fatalf("err = %v", err)
	}
}

func TestRetirarCompactaEReativarVaiParaOFim(t *testing.T) {
	out, err := Aplicar(fabrica(), "em_analise", 0, Patch{Ativa: b(false)})
	if err != nil {
		t.Fatal(err)
	}
	if chaveDe(out, "em_analise").Ativa {
		t.Fatal("coluna deveria estar retirada")
	}
	if got := Sequencia(out); len(got) != 2 || got[0] != "registrado" || got[1] != "encerrado" || chaveDe(out, "encerrado").Ordem != 1 {
		t.Fatalf("sequencia = %v ordem encerrado %d", got, chaveDe(out, "encerrado").Ordem)
	}
	devolta, err := Aplicar(out, "em_analise", 0, Patch{Ativa: b(true)})
	if err != nil {
		t.Fatal(err)
	}
	if got := Sequencia(devolta); got[len(got)-1] != "em_analise" || chaveDe(devolta, "em_analise").Ordem != 2 {
		t.Fatalf("sequencia = %v", got)
	}
}

func TestNaoReordenaColunaRetirada(t *testing.T) {
	out, err := Aplicar(fabrica(), "encerrado", 0, Patch{Ativa: b(false)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Aplicar(out, "encerrado", 0, Patch{Ordem: n(0)})
	if err != ErrRetirada {
		t.Fatalf("err = %v", err)
	}
}

func TestCorENome(t *testing.T) {
	if _, err := Aplicar(fabrica(), "em_analise", 0, Patch{Cor: str("azul")}); err != ErrCor {
		t.Fatalf("cor: %v", err)
	}
	if _, err := Aplicar(fabrica(), "em_analise", 0, Patch{Nome: str("  ")}); err != ErrNome {
		t.Fatalf("nome: %v", err)
	}
	out, err := Aplicar(fabrica(), "em_analise", 0, Patch{Cor: str("#336699")})
	if err != nil || chaveDe(out, "em_analise").Cor != "#336699" {
		t.Fatalf("out err=%v cor=%s", err, chaveDe(out, "em_analise").Cor)
	}
}

func TestPermissoesDoQuadro(t *testing.T) {
	gestor := auth.Identity{UserID: uuid.New(), Permissions: []string{"problemas:manage"}}
	leitor := auth.Identity{UserID: uuid.New(), Permissions: []string{"problemas:read"}}
	fila := auth.Identity{UserID: uuid.New(), Permissions: []string{"atendimento:atender"}}
	if !PodeEscrever(gestor, "problemas") || PodeEscrever(leitor, "problemas") {
		t.Fatal("problemas:manage escreve; read não")
	}
	if !PodeLer(leitor, "problemas") || !PodeLer(fila, "fila") || PodeEscrever(fila, "fila") {
		t.Fatal("leitura da fila é atender; escrita é manage")
	}
	if PodeLer(gestor, "inexistente") || Conhecido("inexistente") {
		t.Fatal("quadro desconhecido")
	}
}
