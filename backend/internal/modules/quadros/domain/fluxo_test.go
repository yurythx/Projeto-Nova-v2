package domain

import (
	"strings"
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func quadroDe(quadro string, chaves ...string) []Coluna {
	out := make([]Coluna, len(chaves))
	for i, chave := range chaves {
		out[i] = Coluna{Quadro: quadro, Chave: chave, Nome: chave, Ordem: i, Entrada: i == 0, Ativa: true}
	}
	return out
}

func pedido(de, para []string) Nova {
	return Nova{Nome: "Aguardando fornecedor", Cor: "#2563eb", Ordem: 1, De: de, Para: para}
}

func TestCriarRecusaPedidoInvalido(t *testing.T) {
	base := quadroDe("problemas", "registrado", "em_analise", "encerrado")
	inativa := quadroDe("problemas", "registrado", "em_analise")
	inativa[1].Ativa = false
	outroQuadro := quadroDe("problemas", "registrado", "em_analise")
	outroQuadro[1].Quadro = "fila"

	casos := []struct {
		nome  string
		cols  []Coluna
		chave string
		nova  Nova
		want  error
	}{
		{"chave curta", base, "c0123456", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"chave longa", base, "c012345678", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"chave com letra fora do hex", base, "c0123456g", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"chave maiúscula", base, "C01234567", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"quadro vazio", nil, "c01234567", pedido([]string{"em_analise"}, nil), ErrQuadro},
		{"nome vazio", base, "c01234567", Nova{Nome: "  ", Cor: "#2563eb", De: []string{"em_analise"}}, ErrNome},
		{"nome longo", base, "c01234567", Nova{Nome: strings.Repeat("á", 81), Cor: "#2563eb", De: []string{"em_analise"}}, ErrNome},
		{"cor vazia", base, "c01234567", Nova{Nome: "Fornecedor", De: []string{"em_analise"}}, ErrCor},
		{"cor curta", base, "c01234567", Nova{Nome: "Fornecedor", Cor: "#abc", De: []string{"em_analise"}}, ErrCor},
		{"sem entrada", base, "c01234567", pedido(nil, []string{"encerrado"}), ErrSemEntrada},
		{"entrada desconhecida", base, "c01234567", pedido([]string{"nao_existe"}, nil), ErrPassagem},
		{"entrada inativa", inativa, "c01234567", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"entrada de outro quadro", outroQuadro, "c01234567", pedido([]string{"em_analise"}, nil), ErrPassagem},
		{"entrada é a própria chave", base, "c01234567", pedido([]string{"c01234567"}, nil), ErrPassagem},
		{"entrada repetida", base, "c01234567", pedido([]string{"em_analise", "em_analise"}, nil), ErrPassagem},
		{"entrada em branco", base, "c01234567", pedido([]string{" "}, nil), ErrPassagem},
		{"saída desconhecida", base, "c01234567", pedido([]string{"em_analise"}, []string{"nao_existe"}), ErrPassagem},
		{"saída inativa", inativa, "c01234567", pedido([]string{"registrado"}, []string{"em_analise"}), ErrPassagem},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if _, _, _, err := Criar(c.cols, c.chave, c.nova); err != c.want {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestCriarRecusaTerminalEReservada(t *testing.T) {
	casos := []struct {
		quadro string
		chaves []string
		de     string
		para   string
	}{
		{"problemas", []string{"registrado", "encerrado"}, "encerrado", ""},
		{"mudancas", []string{"rascunho", "concluida", "falhou", "cancelada"}, "concluida", ""},
		{"mudancas", []string{"rascunho", "falhou"}, "falhou", ""},
		{"mudancas", []string{"rascunho", "cancelada"}, "cancelada", ""},
		{"mudancas", []string{"rascunho", "aprovada"}, "aprovada", ""},
		{"mudancas", []string{"rascunho", "rejeitada"}, "rejeitada", ""},
		{"mudancas", []string{"rascunho", "aprovada"}, "rascunho", "aprovada"},
		{"mudancas", []string{"rascunho", "rejeitada"}, "rascunho", "rejeitada"},
		{"projetos", []string{"planejado", "cancelado", "concluido"}, "cancelado", ""},
		{"fila", []string{"aberto", "encerrado", "cancelado"}, "encerrado", ""},
		{"fila", []string{"aberto", "cancelado"}, "cancelado", ""},
	}
	for _, c := range casos {
		nome := c.quadro + " " + c.de
		if c.para != "" {
			nome += " para " + c.para
		}
		t.Run(nome, func(t *testing.T) {
			de := []string{c.de}
			var para []string
			if c.para != "" {
				de = []string{"rascunho"}
				if c.de != "rascunho" && c.para != "" && c.de != "aprovada" && c.de != "rejeitada" {
					de = []string{c.de}
				}
				para = []string{c.para}
			}
			if _, _, _, err := Criar(quadroDe(c.quadro, c.chaves...), "c01234567", pedido(de, para)); err != ErrPassagem {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestCriarAceitaSaidaOpcionalENaoCopiaOQuadro(t *testing.T) {
	pronta, passagens, out, err := Criar(quadroDe("mudancas", "rascunho", "em_avaliacao"), "c01234567", pedido([]string{"em_avaliacao"}, nil))
	if err != nil || pronta.Quadro != "mudancas" || pronta.Entrada || pronta.Tom != "info" || len(passagens) != 1 || passagens[0].Para != "c01234567" {
		t.Fatalf("mudancas: %+v %+v %v", pronta, passagens, err)
	}
	if chaveDe(out, "rascunho").Quadro != "mudancas" {
		t.Fatal("a coluna nova ficou no quadro errado")
	}
	problemas, _, _, err := Criar(quadroDe("problemas", "registrado", "em_analise"), "c89abcdef", pedido([]string{"registrado"}, []string{"em_analise"}))
	if err != nil || problemas.Quadro != "problemas" || problemas.Chave == pronta.Chave {
		t.Fatalf("problemas: %+v %v", problemas, err)
	}

	fila, passagens, _, err := Criar(quadroDe("fila", "aberto", "concluida"), "c01234567", pedido([]string{"aberto"}, []string{"concluida"}))
	if err != nil || fila.Quadro != "fila" || len(passagens) != 2 {
		t.Fatalf("fila: %+v %+v %v", fila, passagens, err)
	}
	projetos, _, _, err := Criar(quadroDe("projetos", "planejado", "concluido", "cancelado"), "c01234567", pedido([]string{"concluido"}, nil))
	if err != nil || projetos.Quadro != "projetos" {
		t.Fatalf("projetos: %+v %v", projetos, err)
	}
}

func TestCriarOrdemForaDoIntervaloEncaixa(t *testing.T) {
	_, _, out, err := Criar(quadroDe("ativos", "planejado", "em_uso"), "c01234567", Nova{
		Nome: "Reserva", Cor: "#2563eb", Ordem: 99, De: []string{"planejado"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := Sequencia(out); got[len(got)-1] != "c01234567" {
		t.Fatalf("sequencia = %v", got)
	}
	_, _, noInicio, err := Criar(quadroDe("ativos", "planejado", "em_uso"), "c01234567", Nova{
		Nome: "Reserva", Cor: "#2563eb", Ordem: -3, De: []string{"planejado"},
	})
	if err != nil || Sequencia(noInicio)[0] != "c01234567" {
		t.Fatalf("sequencia = %v %v", Sequencia(noInicio), err)
	}
}

func TestAplicarRecusaPatchVazioEChaveDesconhecida(t *testing.T) {
	if _, err := Aplicar(fabrica(), "em_analise", 0, Patch{}); err != ErrPatchVazio {
		t.Fatalf("vazio: %v", err)
	}
	if _, err := Aplicar(fabrica(), "nao_existe", 0, Patch{Nome: str("x")}); err != ErrNaoEncontrada {
		t.Fatalf("chave: %v", err)
	}
	if _, err := Aplicar(fabrica(), "em_analise", 0, Patch{Nome: str(strings.Repeat("a", 81))}); err != ErrNome {
		t.Fatalf("nome: %v", err)
	}
}

func TestPermissoesDeTodosOsQuadros(t *testing.T) {
	casos := []struct{ quadro, ler, escrever string }{
		{"problemas", "problemas:read", "problemas:manage"},
		{"mudancas", "mudancas:aprovar", "mudancas:manage"},
		{"projetos", "projetos:read", "projetos:manage"},
		{"tarefas", "projetos:read", "projetos:manage"},
		{"ativos", "ativos:reportar", "ativos:manage"},
		{"fila", "atendimento:atender", "atendimento:manage"},
	}
	admin := auth.Identity{Roles: []string{string(auth.RoleAdmin)}}
	coringa := auth.Identity{Permissions: []string{"*"}}
	for _, c := range casos {
		t.Run(c.quadro, func(t *testing.T) {
			leitor := auth.Identity{Permissions: []string{c.ler}}
			gestor := auth.Identity{Permissions: []string{c.escrever}}
			if !PodeLer(leitor, c.quadro) || PodeEscrever(leitor, c.quadro) {
				t.Fatal("quem lê não escreve")
			}
			if !PodeLer(gestor, c.quadro) || !PodeEscrever(gestor, c.quadro) {
				t.Fatal("quem gere lê e escreve")
			}
			if !PodeEscrever(admin, c.quadro) || !PodeEscrever(coringa, c.quadro) {
				t.Fatal("admin cobre o quadro")
			}
		})
	}
}
