package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNovaConexao(t *testing.T) {
	c, err := NovaConexao(" glpi ", " GLPI da prefeitura ", "direto", "http://helpdesk/apirest.php/", "Super-Admin", "")
	if err != nil || c.Nome != "GLPI da prefeitura" || c.URL != "http://helpdesk/apirest.php" || c.Fuso != "America/Sao_Paulo" || c.ID == uuid.Nil {
		t.Fatalf("conexão: %+v %v", c, err)
	}
	if c, err := NovaConexao("glpi", "Conector", "conector", "", "", "America/Cuiaba"); err != nil || c.URL != "" {
		t.Fatalf("o conector dispensa o endereço: %+v %v", c, err)
	}
	for nome, args := range map[string][6]string{
		"origem":               {"otrs", "x", "direto", "http://h/a", "", ""},
		"nome":                 {"glpi", " ", "direto", "http://h/a", "", ""},
		"modo":                 {"glpi", "x", "manual", "http://h/a", "", ""},
		"perfil":               {"glpi", "x", "direto", "http://h/a", strings.Repeat("p", 81), ""},
		"url longa":            {"glpi", "x", "direto", "http://h/" + strings.Repeat("a", 500), "", ""},
		"sem url":              {"glpi", "x", "direto", "", "", ""},
		"url ftp":              {"glpi", "x", "direto", "ftp://h/a", "", ""},
		"url do conector ruim": {"glpi", "x", "conector", "h/a", "", ""},
		"fuso":                 {"glpi", "x", "direto", "http://h/a", "", "Lugar/Nenhum"},
	} {
		if _, err := NovaConexao(args[0], args[1], args[2], args[3], args[4], args[5]); !errors.As(err, new(Recusa)) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if (Recusa{"x"}).Error() != "x" {
		t.Fatal("mensagem da recusa")
	}
}

func TestTokenECredenciais(t *testing.T) {
	token, hash := EmitirToken()
	if !strings.HasPrefix(token, PrefixoToken) || hash != HashToken(token) || len(hash) != 64 {
		t.Fatalf("token: %q %q", token, hash)
	}
	if outro, _ := EmitirToken(); outro == token {
		t.Fatal("cada token é único")
	}
	for c, vazia := range map[Credenciais]bool{{}: true, {Login: "a"}: true, {Login: "a", Senha: "b"}: false, {UserToken: "u"}: false} {
		if c.Vazia() != vazia {
			t.Errorf("%+v: %v", c, c.Vazia())
		}
	}
}

func TestValidarItens(t *testing.T) {
	ok := Item{Tipo: ItemChamado, ID: 1, Classe: "incidente", Dados: json.RawMessage(`{"a":1}`)}
	if err := ValidarItens([]Item{ok, {Tipo: ItemEntidade, ID: 0, Dados: json.RawMessage(`{}`)}}); err != nil {
		t.Fatal(err)
	}
	muitos := make([]Item, LoteMax+1)
	for nome, itens := range map[string][]Item{
		"vazio":    nil,
		"muitos":   muitos,
		"tipo":     {{Tipo: "ticket", ID: 1, Dados: json.RawMessage(`{}`)}},
		"id":       {{Tipo: ItemPessoa, ID: -1, Dados: json.RawMessage(`{}`)}},
		"sem dado": {{Tipo: ItemPessoa, ID: 1}},
		"json":     {{Tipo: ItemPessoa, ID: 1, Dados: json.RawMessage(`{`)}},
		"grande":   {{Tipo: ItemPessoa, ID: 1, Dados: json.RawMessage(`"` + strings.Repeat("a", DadosMax) + `"`)}},
		"classe":   {{Tipo: ItemChamado, ID: 1, Classe: "problema", Dados: json.RawMessage(`{}`)}},
		"nome":     {{Tipo: ItemPessoa, ID: 1, Nome: strings.Repeat("n", 501), Dados: json.RawMessage(`{}`)}},
	} {
		if err := ValidarItens(itens); !errors.As(err, new(Recusa)) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func TestValidarTabela(t *testing.T) {
	linha := func(sec string, cat int) Linha {
		return Linha{Secretaria: sec, Categoria: cat, Oferta: " Computadores ", Tipo: "requisicao", Prioridade: "alta"}
	}
	tabela := []Linha{linha(" SMS ", 7), linha("SMS", CategoriaQualquer), linha("PREFEITURA", 7)}
	if err := ValidarTabela(tabela); err != nil || tabela[0].Secretaria != "SMS" || tabela[0].Oferta != "Computadores" {
		t.Fatalf("tabela: %+v %v", tabela, err)
	}
	ruim := func(f func(*Linha)) []Linha {
		l := linha("SMS", 1)
		f(&l)
		return []Linha{l}
	}
	for nome, linhas := range map[string][]Linha{
		"secretaria": ruim(func(l *Linha) { l.Secretaria = "" }),
		"categoria":  ruim(func(l *Linha) { l.Categoria = -2 }),
		"oferta":     ruim(func(l *Linha) { l.Oferta = strings.Repeat("o", 161) }),
		"catálogo":   ruim(func(l *Linha) { l.Catalogo = strings.Repeat("c", 81) }),
		"tipo":       ruim(func(l *Linha) { l.Tipo = "problema" }),
		"prioridade": ruim(func(l *Linha) { l.Prioridade = "urgente" }),
		"repetida":   {linha("sms", 1), linha("SMS", 1)},
	} {
		if err := ValidarTabela(linhas); !errors.As(err, new(Recusa)) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func TestTarefaESecretaria(t *testing.T) {
	c := uuid.New()
	tf, err := NovaTarefa(c, TarefaCarga, "ana")
	if err != nil || tf.Estado != EstadoPendente || tf.ConexaoID != c || tf.Terminada() {
		t.Fatalf("tarefa: %+v %v", tf, err)
	}
	if _, err := NovaTarefa(c, "limpeza", "ana"); !errors.As(err, new(Recusa)) {
		t.Fatalf("tipo: %v", err)
	}
	for _, e := range []string{EstadoConcluida, EstadoFalhou, EstadoCancelada} {
		if !(Tarefa{Estado: e}).Terminada() {
			t.Errorf("%s termina", e)
		}
	}
	for completo, quer := range map[string]string{"RAIZ > SMS > UPA": "SMS", "RAIZ": "RAIZ", " X ": "X"} {
		if Secretaria(completo) != quer {
			t.Errorf("%q", completo)
		}
	}
}
