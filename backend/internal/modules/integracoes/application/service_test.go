package application

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func status(err error) int {
	if ae, ok := apperrors.As(err); ok {
		return ae.Status
	}
	return 0
}

func TestMapError(t *testing.T) {
	for err, quer := range map[error]int{
		domain.ErrNaoEncontrado:   http.StatusNotFound,
		domain.ErrForaDoEscopo:    http.StatusForbidden,
		domain.ErrTarefaAtiva:     http.StatusConflict,
		domain.Recusa{Msg: "não"}: http.StatusUnprocessableEntity,
	} {
		if s := status(MapError(err)); s != quer {
			t.Errorf("%v: %d", err, s)
		}
	}
	if MapError(nil) != nil || !errors.Is(MapError(errFalha), errFalha) {
		t.Fatal("nil e erro desconhecido passam")
	}
}

func TestConexoesECredenciais(t *testing.T) {
	ctx := context.Background()
	s, _, _ := novoServico(t, infrastructure.NewRepository())
	c := conexao(t, s, domain.ModoDireto)
	if c.CreatedBy != "admin" {
		t.Fatalf("quem criou: %+v", c)
	}
	servidor := auth.Identity{UserID: uuid.New()}
	if _, err := s.ListarConexoes(ctx, servidor); status(err) != http.StatusForbidden {
		t.Fatalf("só quem administra tudo: %v", err)
	}
	if lista, err := s.ListarConexoes(ctx, admin()); err != nil || len(lista) == 0 {
		t.Fatalf("lista: %v", err)
	}
	if _, err := s.CriarConexao(ctx, servidor, ConexaoEntrada{}); status(err) != http.StatusForbidden {
		t.Fatalf("criar sem permissão: %v", err)
	}
	if _, err := s.CriarConexao(ctx, admin(), ConexaoEntrada{Tipo: "otrs"}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("criar inválida: %v", err)
	}
	// Tipo e modo não mudam na edição.
	nova, err := s.AtualizarConexao(ctx, admin(), c.ID, ConexaoEntrada{Tipo: "otrs", Modo: "conector", Nome: "Renomeada", URL: "https://glpi/apirest.php", Perfil: "Super-Admin"})
	if err != nil || nova.Nome != "Renomeada" || nova.Modo != domain.ModoDireto || nova.Perfil != "Super-Admin" {
		t.Fatalf("atualizar: %+v %v", nova, err)
	}
	if _, err := s.AtualizarConexao(ctx, admin(), c.ID, ConexaoEntrada{Nome: ""}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("atualizar inválida: %v", err)
	}
	if _, err := s.AtualizarConexao(ctx, admin(), uuid.New(), ConexaoEntrada{Nome: "x"}); status(err) != http.StatusNotFound {
		t.Fatalf("atualizar inexistente: %v", err)
	}

	cred := domain.Credenciais{AppToken: "a", Login: "yuri", Senha: "s"}
	if err := s.DefinirCredenciais(ctx, admin(), uuid.New(), cred); status(err) != http.StatusNotFound {
		t.Fatalf("credenciais de quem não há: %v", err)
	}
	if err := s.DefinirCredenciais(ctx, admin(), c.ID, domain.Credenciais{Login: "x"}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("credenciais vazias: %v", err)
	}
	if err := s.DefinirCredenciais(ctx, admin(), c.ID, cred); err != nil {
		t.Fatal(err)
	}
	g, _ := s.ObterConexao(ctx, admin(), c.ID)
	if got, err := s.credenciais(g); err != nil || got != cred || g.Cifradas == "" || g.Cifradas == "yuri" {
		t.Fatalf("cifradas e de volta: %+v %v", got, err)
	}
	semChave := NewService(s.pool, s.repo, eventos{}, nil)
	if err := semChave.DefinirCredenciais(ctx, admin(), c.ID, cred); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sem a chave de configuração: %v", err)
	}
	if _, err := semChave.credenciais(g); err == nil {
		t.Fatal("ler sem a chave")
	}
	g.Cifradas = "lixo"
	if _, err := s.credenciais(g); err == nil {
		t.Fatal("cifra ilegível")
	}
	g.Cifradas = s.cipher.Encrypt("não é json")
	if _, err := s.credenciais(g); err == nil {
		t.Fatal("conteúdo ilegível")
	}
	if _, err := s.EmitirToken(ctx, admin(), c.ID); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("token no modo direto: %v", err)
	}
	if _, err := s.EmitirToken(ctx, admin(), uuid.New()); status(err) != http.StatusNotFound {
		t.Fatalf("token de quem não há: %v", err)
	}

	k := conexao(t, s, domain.ModoConector)
	if err := s.DefinirCredenciais(ctx, admin(), k.ID, cred); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("no conector, as credenciais ficam com ele: %v", err)
	}
	if err := s.RevogarToken(ctx, servidor, k.ID); status(err) != http.StatusForbidden {
		t.Fatalf("revogar sem permissão: %v", err)
	}
	if err := s.ExcluirConexao(ctx, servidor, k.ID); status(err) != http.StatusForbidden {
		t.Fatalf("excluir sem permissão: %v", err)
	}
	if err := s.ExcluirConexao(ctx, admin(), k.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ObterConexao(ctx, admin(), k.ID); status(err) != http.StatusNotFound {
		t.Fatalf("excluída: %v", err)
	}
}

func TestConector(t *testing.T) {
	ctx := context.Background()
	s, _, _ := novoServico(t, infrastructure.NewRepository())
	c := conexao(t, s, domain.ModoConector)
	if _, ok, err := s.Reconhecer(ctx, "eyJhbGciOi"); ok || err != nil {
		t.Fatalf("JWT não é do conector: %v %v", ok, err)
	}
	if _, ok, err := s.Reconhecer(ctx, domain.PrefixoToken+"nenhum"); !ok || status(err) != http.StatusUnauthorized {
		t.Fatalf("token que não há: %v %v", ok, err)
	}
	if _, err := s.PedirTarefa(ctx, admin(), c.ID, domain.TarefaExtracao); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("extrair sem token: %v", err)
	}
	token, err := s.EmitirToken(ctx, admin(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	id, ok, err := s.Reconhecer(ctx, token)
	if err != nil || !ok || id.Subject != c.ID.String() || id.Source != auth.SourceAgente {
		t.Fatalf("conector: %+v %v %v", id, ok, err)
	}

	// Sem pedido na tela, o conector não tem o que fazer.
	if _, ok, err := s.ConectorPegar(ctx, id); ok || err != nil {
		t.Fatalf("nada pedido: %v %v", ok, err)
	}
	pedida, err := s.PedirTarefa(ctx, admin(), c.ID, domain.TarefaExtracao)
	if err != nil {
		t.Fatal(err)
	}
	tf, ok, err := s.ConectorPegar(ctx, id)
	if err != nil || !ok || tf.ID != pedida.ID {
		t.Fatalf("pegar: %+v %v %v", tf, ok, err)
	}
	if v, p, err := s.ConectorVersoes(ctx, id); err != nil || len(v) != 0 || len(p) != 0 {
		t.Fatalf("nada extraído ainda: %v %v %v", v, p, err)
	}
	itens := []domain.Item{{Tipo: domain.ItemChamado, ID: 1, Versao: "v1", Dados: json.RawMessage(`{}`)},
		{Tipo: domain.ItemPessoa, ID: 9, Dados: json.RawMessage(`{}`)}}
	if err := s.ConectorGravar(ctx, id, itens); err != nil {
		t.Fatal(err)
	}
	if err := s.ConectorGravar(ctx, id, nil); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("lote vazio: %v", err)
	}
	if v, p, _ := s.ConectorVersoes(ctx, id); v[1] != "v1" || !p[9] {
		t.Fatalf("o que já está lá: %v %v", v, p)
	}
	if estado, err := s.ConectorProgresso(ctx, id, tf.ID, 1, 2); err != nil || estado != domain.EstadoEmAndamento {
		t.Fatalf("progresso: %q %v", estado, err)
	}
	if err := s.ConectorTerminar(ctx, id, tf.ID, json.RawMessage(`{"chamados":1}`), ""); err != nil {
		t.Fatal(err)
	}
	if g, _ := s.ObterConexao(ctx, admin(), c.ID); g.ExtraidoEm == nil {
		t.Fatalf("a data da extração: %+v", g)
	}

	// O conector só mexe na própria conexão, e só na extração.
	outra := conexao(t, s, domain.ModoConector)
	tokenOutra, _ := s.EmitirToken(ctx, admin(), outra.ID)
	idOutra, _, _ := s.Reconhecer(ctx, tokenOutra)
	if _, err := s.ConectorProgresso(ctx, idOutra, tf.ID, 1, 1); status(err) != http.StatusNotFound {
		t.Fatalf("tarefa de outra conexão: %v", err)
	}
	semPermissao := auth.Identity{Subject: c.ID.String(), Source: auth.SourceAgente}
	for nome, ident := range map[string]auth.Identity{"subject": {Subject: "x"}, "pessoa": {Subject: c.ID.String()}, "permissão": semPermissao} {
		if _, _, err := s.ConectorVersoes(ctx, ident); status(err) != http.StatusForbidden {
			t.Errorf("%s: %v", nome, err)
		}
		if err := s.ConectorGravar(ctx, ident, itens); status(err) != http.StatusForbidden {
			t.Errorf("%s gravar: %v", nome, err)
		}
		if _, _, err := s.ConectorPegar(ctx, ident); status(err) != http.StatusForbidden {
			t.Errorf("%s pegar: %v", nome, err)
		}
		if err := s.ConectorTerminar(ctx, ident, tf.ID, nil, ""); status(err) != http.StatusForbidden {
			t.Errorf("%s terminar: %v", nome, err)
		}
	}
	if err := s.RevogarToken(ctx, admin(), c.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Reconhecer(ctx, token); status(err) != http.StatusUnauthorized {
		t.Fatalf("token revogado: %v", err)
	}
}

func TestTabelaEInventario(t *testing.T) {
	ctx := context.Background()
	s, _, _ := novoServico(t, infrastructure.NewRepository())
	c := conexao(t, s, domain.ModoDireto)
	if _, err := s.SugerirTabela(ctx, admin(), c.ID, nil); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sugerir sem extrair: %v", err)
	}
	ent, cat, sla := 1, 7, 2
	min := 240
	d := json.RawMessage(`{}`)
	if err := (preparo{s, c.ID}).Gravar(ctx, []domain.Item{
		{Tipo: domain.ItemEntidade, ID: 1, Nome: "RAIZ > SMS", Dados: d},
		{Tipo: domain.ItemCategoria, ID: 7, Nome: "Serviços > Computadores", Dados: d},
		{Tipo: domain.ItemSLA, ID: 2, Nome: "N1", Minutos: &min, Dados: d},
		{Tipo: domain.ItemChamado, ID: 1, Entidade: &ent, Categoria: &cat, SLA: &sla, Classe: "incidente", Dados: d},
	}); err != nil {
		t.Fatal(err)
	}
	inv, err := s.Inventario(ctx, admin(), c.ID)
	if err != nil || len(inv.Grupos) != 1 || inv.Secretarias["SMS"] != 1 {
		t.Fatalf("inventário: %+v %v", inv, err)
	}
	u := dbtest.Unidade(t, s.pool)
	linhas, err := s.SugerirTabela(ctx, admin(), c.ID, map[string]uuid.UUID{"SMS": u})
	if err != nil || len(linhas) != 2 || linhas[0].Prioridade != "alta" || linhas[0].UnidadeNome == "" {
		t.Fatalf("sugestão: %+v %v", linhas, err)
	}
	linhas[0].Oferta = "Equipamentos"
	if salvas, err := s.SalvarTabela(ctx, admin(), c.ID, linhas); err != nil || salvas[0].Oferta != "Equipamentos" {
		t.Fatalf("salvar: %+v %v", salvas, err)
	}
	if _, err := s.SalvarTabela(ctx, admin(), c.ID, []domain.Linha{{}}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("tabela inválida: %v", err)
	}
	nada := uuid.New()
	for nome, err := range map[string]error{
		"inventário": func() error { _, err := s.Inventario(ctx, admin(), nada); return err }(),
		"tabela":     func() error { _, err := s.Tabela(ctx, admin(), nada); return err }(),
		"sugerir":    func() error { _, err := s.SugerirTabela(ctx, admin(), nada, nil); return err }(),
		"salvar":     func() error { _, err := s.SalvarTabela(ctx, admin(), nada, nil); return err }(),
		"tarefas":    func() error { _, err := s.ListarTarefas(ctx, admin(), nada); return err }(),
		"pedir":      func() error { _, err := s.PedirTarefa(ctx, admin(), nada, "carga"); return err }(),
	} {
		if status(err) != http.StatusNotFound {
			t.Errorf("%s de quem não há: %v", nome, err)
		}
	}
}

func TestFalhasDoRepositorio(t *testing.T) {
	ctx := context.Background()
	base, _, _ := novoServico(t, infrastructure.NewRepository())
	c := conexao(t, base, domain.ModoDireto)
	k := conexao(t, base, domain.ModoConector)
	token, _ := base.EmitirToken(ctx, admin(), k.ID)
	id, _, _ := base.Reconhecer(ctx, token)
	d := json.RawMessage(`{}`)
	if err := (preparo{base, c.ID}).Gravar(ctx, []domain.Item{{Tipo: domain.ItemChamado, ID: 1, Dados: d}}); err != nil {
		t.Fatal(err)
	}
	if _, err := base.SalvarTabela(ctx, admin(), c.ID, []domain.Linha{{Secretaria: "S", Categoria: -1, Oferta: "O", Tipo: "incidente", Prioridade: "alta"}}); err != nil {
		t.Fatal(err)
	}
	tf, _ := base.PedirTarefa(ctx, admin(), k.ID, domain.TarefaExtracao)
	itens := []domain.Item{{Tipo: domain.ItemPessoa, ID: 1, Dados: d}}
	linha := []domain.Linha{{Secretaria: "S", Categoria: -1, Oferta: "O", Tipo: "incidente", Prioridade: "alta"}}
	com := func(metodos ...string) *Service {
		s := NewService(base.pool, falhando(metodos...), eventos{}, base.cipher).WithExtrator(base.extrator).WithCarregador(base.carregador)
		return s
	}
	casos := map[string]error{
		"listar": func() error { _, err := com("ListConexoes").ListarConexoes(ctx, admin()); return err }(),
		"criar": func() error {
			_, err := com("InsertConexao").CriarConexao(ctx, admin(), ConexaoEntrada{Tipo: "glpi", Nome: "x", Modo: "conector"})
			return err
		}(),
		"atualizar": func() error {
			_, err := com("UpdateConexao").AtualizarConexao(ctx, admin(), c.ID, ConexaoEntrada{Nome: "x", URL: "http://g/a"})
			return err
		}(),
		"excluir":     com("DeleteConexao").ExcluirConexao(ctx, admin(), uuid.New()),
		"credenciais": com("SetCredenciais").DefinirCredenciais(ctx, admin(), c.ID, domain.Credenciais{UserToken: "u"}),
		"token":       func() error { _, err := com("SetToken").EmitirToken(ctx, admin(), k.ID); return err }(),
		"revogar":     com("SetToken").RevogarToken(ctx, admin(), k.ID),
		"reconhecer":  func() error { _, _, err := com("PorToken").Reconhecer(ctx, token); return err }(),
		"versões":     func() error { _, _, err := com("Versoes").ConectorVersoes(ctx, id); return err }(),
		"pessoas":     func() error { _, _, err := com("IDs").ConectorVersoes(ctx, id); return err }(),
		"gravar":      com("GravarItens").ConectorGravar(ctx, id, itens),
		"progresso":   func() error { _, err := com("Progresso").ConectorProgresso(ctx, id, tf.ID, 1, 1); return err }(),
		"tarefa":      func() error { _, err := com("GetTarefa").ConectorProgresso(ctx, id, tf.ID, 1, 1); return err }(),
		"terminar":    com("Terminar").ConectorTerminar(ctx, id, tf.ID, nil, ""),
		"marcar":      com("MarcarExtracao").ConectorTerminar(ctx, id, tf.ID, nil, ""),
		"contagens":   func() error { _, err := com("Contagens").Inventario(ctx, admin(), c.ID); return err }(),
		"grupos":      func() error { _, err := com("Grupos").Inventario(ctx, admin(), c.ID); return err }(),
		"entidades":   func() error { _, err := com("Nomes:entidade").Inventario(ctx, admin(), c.ID); return err }(),
		"categorias":  func() error { _, err := com("Nomes:categoria").Inventario(ctx, admin(), c.ID); return err }(),
		"slas":        func() error { _, err := com("SLAs").Inventario(ctx, admin(), c.ID); return err }(),
		"sugerir":     func() error { _, err := com("Grupos").SugerirTabela(ctx, admin(), c.ID, nil); return err }(),
		"trocar":      func() error { _, err := com("TrocarTabela").SalvarTabela(ctx, admin(), c.ID, linha); return err }(),
		"listarTab":   func() error { _, err := com("ListarTabela").Tabela(ctx, admin(), c.ID); return err }(),
		"inserir": func() error {
			_, err := com("InserirTarefa").PedirTarefa(ctx, admin(), c.ID, domain.TarefaSimulacao)
			return err
		}(),
		"conferir": func() error {
			_, err := com("Contagens").PedirTarefa(ctx, admin(), c.ID, domain.TarefaCarga)
			return err
		}(),
		"conferirTab": func() error {
			_, err := com("ListarTabela").PedirTarefa(ctx, admin(), c.ID, domain.TarefaCarga)
			return err
		}(),
		"listarTar": func() error { _, err := com("ListarTarefas").ListarTarefas(ctx, admin(), c.ID); return err }(),
		"cancelar":  com("Cancelar").CancelarTarefa(ctx, admin(), tf.ID),
		"evento":    NewService(base.pool, infrastructure.NewRepository(), eventos{errFalha}, nil).RevogarToken(ctx, admin(), k.ID),
	}
	for nome, err := range casos {
		if !errors.Is(err, errFalha) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if err := com().CancelarTarefa(ctx, auth.Identity{}, tf.ID); status(err) != http.StatusForbidden {
		t.Fatalf("cancelar sem permissão: %v", err)
	}
}

func TestPedirTarefa(t *testing.T) {
	ctx := context.Background()
	s, _, _ := novoServico(t, infrastructure.NewRepository())
	c := conexao(t, s, domain.ModoDireto)
	pedir := func(svc *Service, tipo string) error { _, err := svc.PedirTarefa(ctx, admin(), c.ID, tipo); return err }
	if err := pedir(s, "limpar"); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("tipo: %v", err)
	}
	if err := pedir(s, domain.TarefaExtracao); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("extrair sem credenciais: %v", err)
	}
	semCarga := NewService(s.pool, s.repo, eventos{}, s.cipher)
	if err := pedir(semCarga, domain.TarefaSimulacao); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sem o atendimento: %v", err)
	}
	if err := pedir(s, domain.TarefaSimulacao); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("simular sem extrair: %v", err)
	}
	d := json.RawMessage(`{}`)
	_ = (preparo{s, c.ID}).Gravar(ctx, []domain.Item{{Tipo: domain.ItemChamado, ID: 1, Dados: d}})
	if err := pedir(s, domain.TarefaSimulacao); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("simular sem tabela: %v", err)
	}
	if _, err := s.SalvarTabela(ctx, admin(), c.ID, []domain.Linha{{Secretaria: "S", Categoria: -1, CategoriaNome: "(qualquer)", Oferta: "O", Tipo: "incidente", Prioridade: "alta"}}); err != nil {
		t.Fatal(err)
	}
	if err := pedir(s, domain.TarefaCarga); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("carregar sem unidade: %v", err)
	}
	if err := pedir(s, domain.TarefaSimulacao); err != nil {
		t.Fatal(err)
	}
	if err := pedir(s, domain.TarefaSimulacao); status(err) != http.StatusConflict {
		t.Fatalf("uma de cada vez: %v", err)
	}
	if lista, err := s.ListarTarefas(ctx, admin(), c.ID); err != nil || len(lista) != 1 {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	if err := s.CancelarTarefa(ctx, admin(), uuid.New()); status(err) != http.StatusNotFound {
		t.Fatalf("cancelar o que não há: %v", err)
	}
}
