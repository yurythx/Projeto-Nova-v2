package application

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/caixa"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func TestListaNaoLevaSenhaEOutraUnidadeNaoRevela(t *testing.T) {
	repo := &memoria{}
	trilha := &capta{}
	svc := novo(t, repo, trilha)
	unidadeA := uuid.New()
	unidadeB := uuid.New()
	senha := "senha-que-nao-pode-vazar"

	criado, err := svc.Criar(context.Background(), admin(), Entrada{Nome: "Correio", Usuario: "ana", Senha: senha, Notas: "pista", Unidade: &unidadeA})
	if err != nil {
		t.Fatal(err)
	}
	corpo, _ := json.Marshal(criado)
	if strings.Contains(string(corpo), senha) || strings.Contains(repo.itens[criado.ID].Cifra, senha) {
		t.Fatal("a senha apareceu no metadado ou no texto cifrado")
	}

	lista, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidadeA))
	if err != nil {
		t.Fatal(err)
	}
	listaJSON, _ := json.Marshal(lista)
	if strings.Contains(string(listaJSON), senha) || strings.Contains(string(listaJSON), "dek") {
		t.Fatalf("lista vazou: %s", listaJSON)
	}
	if len(lista) != 1 || lista[0].Nome != "Correio" {
		t.Fatalf("lista: %+v", lista)
	}

	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreRead, unidadeA), criado.ID); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("leitura revelou: %v", err)
	}
	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeB), criado.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("outra unidade: %v", err)
	}

	revelado, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeA), criado.ID)
	if err != nil || revelado.Senha != senha || revelado.Notas != "pista" {
		t.Fatalf("revelar: %+v %v", revelado, err)
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool {
		return item.evento == EventoRevelado && !strings.Contains(item.texto(), senha) && item.payload["id"] == criado.ID.String()
	}) {
		t.Fatalf("auditoria da revelação: %+v", trilha.itens)
	}
}

func TestMoverExigeGrantEMantemASenha(t *testing.T) {
	repo := &memoria{}
	svc := novo(t, repo, &capta{})
	unidadeA := uuid.New()
	unidadeB := uuid.New()
	criado, err := svc.Criar(context.Background(), naUnidade(auth.PermCofreManage, unidadeA), Entrada{Nome: "Wifi", Usuario: "rede", Senha: "segredo-wifi", Unidade: &unidadeA})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Atualizar(context.Background(), nasUnidades(auth.PermCofreGrant, unidadeA, unidadeB), criado.ID, Mudanca{
		Nome: "Wifi novo", Usuario: "rede", Mover: true, Unidade: &unidadeB,
	}); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("grant não troca o nome: %v", err)
	}
	movido, err := svc.Atualizar(context.Background(), nasUnidades(auth.PermCofreGrant, unidadeA, unidadeB), criado.ID, Mudanca{
		Nome: "Wifi", Usuario: "rede", Mover: true, Unidade: &unidadeB,
	})
	if err != nil || movido.UnidadeID == nil || *movido.UnidadeID != unidadeB || movido.Versao != 2 {
		t.Fatalf("mover: %+v %v", movido, err)
	}
	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeA), criado.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("a origem ainda vê: %v", err)
	}
	revelado, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreManage, unidadeB), criado.ID)
	if err != nil || revelado.Senha != "segredo-wifi" {
		t.Fatalf("destino: %+v %v", revelado, err)
	}
}

func TestTrocarSenhaNaoDeixaAAnteriorAbrir(t *testing.T) {
	repo := &memoria{}
	svc := novo(t, repo, &capta{})
	unidade := uuid.New()
	quem := naUnidade(auth.PermCofreManage, unidade)
	criado, err := svc.Criar(context.Background(), quem, Entrada{Nome: "VPN", Usuario: "ops", Senha: "antiga", Notas: "nota", Unidade: &unidade})
	if err != nil {
		t.Fatal(err)
	}
	nova := "nova-senha"
	if _, err := svc.Atualizar(context.Background(), quem, criado.ID, Mudanca{Nome: "VPN", Usuario: "ops", Senha: &nova}); err != nil {
		t.Fatal(err)
	}
	revelado, err := svc.Revelar(context.Background(), quem, criado.ID)
	if err != nil || revelado.Senha != nova || revelado.Notas != "nota" {
		t.Fatalf("troca: %+v %v", revelado, err)
	}
}

func TestRevelarNaoPedeSenhaDeAcessoDeNovo(t *testing.T) {
	repo := &memoria{}
	trilha := &capta{}
	svc := novo(t, repo, trilha)
	unidade := uuid.New()
	quem := naUnidade(auth.PermCofreReveal, unidade)
	quem.Username = "ana"
	criado, err := svc.Criar(context.Background(), naUnidade(auth.PermCofreManage, unidade), Entrada{Nome: "VPN", Usuario: "ops", Senha: "segredo-vpn", Unidade: &unidade})
	if err != nil {
		t.Fatal(err)
	}
	revelado, err := svc.Revelar(context.Background(), quem, criado.ID)
	if err != nil || revelado.Senha != "segredo-vpn" {
		t.Fatalf("revelar: %+v %v", revelado, err)
	}
	if slices.ContainsFunc(trilha.itens, func(item captaItem) bool {
		return strings.Contains(item.texto(), "segredo-vpn")
	}) {
		t.Fatalf("a auditoria levou senha: %+v", trilha.itens)
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool { return item.evento == EventoRevelado }) {
		t.Fatal("falta o registro da revelação")
	}
}

func TestEnderecoFicaNaListaEPainelApontaAMaquina(t *testing.T) {
	repo := &memoria{}
	svc := novo(t, repo, &capta{})
	unidade := uuid.New()
	quem := naUnidade(auth.PermCofreManage, unidade)
	vm, err := svc.Criar(context.Background(), quem, Entrada{
		Nome: "vm-web", Usuario: "root", Senha: "senha-da-vm", Tipo: domain.TipoMaquina, Endereco: "10.0.0.8", Unidade: &unidade,
	})
	if err != nil {
		t.Fatal(err)
	}
	painel, err := svc.Criar(context.Background(), quem, Entrada{
		Nome: "aaPanel", Usuario: "admin", Senha: "senha-do-painel", Tipo: domain.TipoPainel,
		Endereco: "https://painel.exemplo", Unidade: &unidade, Administra: []uuid.UUID{vm.ID},
	})
	if err != nil || !slices.Contains(painel.Administra, vm.ID) {
		t.Fatalf("painel: %+v %v", painel, err)
	}
	app, err := svc.Criar(context.Background(), quem, Entrada{
		Nome: "Correio", Usuario: "admin", Senha: "senha-do-correio", Tipo: domain.TipoAplicacao,
		Endereco: "https://correio.exemplo", Unidade: &unidade,
	})
	if err != nil {
		t.Fatal(err)
	}
	lista, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil {
		t.Fatal(err)
	}
	corpo, _ := json.Marshal(lista)
	texto := string(corpo)
	if !strings.Contains(texto, "10.0.0.8") || !strings.Contains(texto, "https://painel.exemplo") || !strings.Contains(texto, vm.ID.String()) {
		t.Fatalf("a lista escondeu o endereço ou o vínculo: %s", texto)
	}
	if strings.Contains(texto, "senha-da-vm") || strings.Contains(texto, "senha-do-painel") || strings.Contains(texto, "senha-do-correio") {
		t.Fatalf("a lista levou senha: %s", texto)
	}
	if _, err := svc.Criar(context.Background(), quem, Entrada{
		Nome: "outro", Usuario: "a", Senha: "x", Tipo: domain.TipoAplicacao, Unidade: &unidade, Administra: []uuid.UUID{vm.ID},
	}); err == nil {
		t.Fatal("aplicação administrou máquina")
	}
	_ = app
}

func TestRetiradaReservaAteOPrazoENaoLevaASenha(t *testing.T) {
	repo := &memoria{}
	trilha := &capta{}
	svc := novo(t, repo, trilha)
	momento := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	svc.agora = func() time.Time { return momento }
	unidade := uuid.New()
	ana := naUnidade(auth.PermCofreReveal, unidade)
	ana.UserID = uuid.New()
	ana.Name = "Ana"
	bia := naUnidade(auth.PermCofreReveal, unidade)
	bia.UserID = uuid.New()
	bia.Name = "Bia"
	gestor := naUnidade(auth.PermCofreManage, unidade)
	gestor.UserID = uuid.New()
	gestor.Name = "Gestor"

	criado, err := svc.Criar(context.Background(), gestor, Entrada{
		Nome: "vm-web", Usuario: "root", Senha: "senha-da-vm", Tipo: domain.TipoMaquina, Endereco: "10.0.0.8", Unidade: &unidade,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Retirar(context.Background(), ana, criado.ID, "acesso", 10); err == nil {
		t.Fatal("aceitou retirada de 10 minutos")
	}
	agente := ana
	agente.Source = auth.SourceAgente
	if _, err := svc.Retirar(context.Background(), agente, criado.ID, "acesso", 60); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("agente retirou: %v", err)
	}

	retirado, err := svc.Retirar(context.Background(), ana, criado.ID, "acesso", 60)
	if err != nil || retirado.Senha != "senha-da-vm" || retirado.Retirada == nil || !retirado.Retirada.Ate.Equal(momento.Add(time.Hour)) {
		t.Fatalf("retirar: %+v %v", retirado, err)
	}
	if _, err := svc.Retirar(context.Background(), bia, criado.ID, "errada", 60); !errors.Is(err, domain.ErrOcupado) {
		t.Fatalf("outra pessoa, senha errada nem deveria contar: %v", err)
	}
	if _, err := svc.Revelar(context.Background(), bia, criado.ID); !errors.Is(err, domain.ErrOcupado) {
		t.Fatalf("revelou ocupado: %v", err)
	}
	if _, err := svc.Revelar(context.Background(), ana, criado.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Devolver(context.Background(), bia, criado.ID); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("quem não retirou devolveu: %v", err)
	}

	lista, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || len(lista) != 1 || lista[0].Retirada == nil || lista[0].Retirada.Nome != "Ana" {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	corpo, _ := json.Marshal(lista)
	if strings.Contains(string(corpo), "senha-da-vm") {
		t.Fatalf("lista levou a senha: %s", corpo)
	}
	for _, item := range trilha.itens {
		if strings.Contains(item.texto(), "senha-da-vm") {
			t.Fatalf("auditoria levou a senha: %s", item.texto())
		}
	}

	renovado, err := svc.Retirar(context.Background(), ana, criado.ID, "acesso", 15)
	if err != nil || renovado.Retirada == nil || !renovado.Retirada.Ate.Equal(momento.Add(15*time.Minute)) {
		t.Fatalf("renovar: %+v %v", renovado.Retirada, err)
	}
	if err := svc.Devolver(context.Background(), gestor, criado.ID); err != nil {
		t.Fatal(err)
	}
	livre, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || livre[0].Retirada != nil {
		t.Fatalf("ainda retirado: %+v %v", livre, err)
	}

	if _, err := svc.Retirar(context.Background(), ana, criado.ID, "acesso", 60); err != nil {
		t.Fatal(err)
	}
	momento = momento.Add(time.Hour)
	seguinte, err := svc.Retirar(context.Background(), bia, criado.ID, "acesso", 30)
	if err != nil || seguinte.Retirada == nil || seguinte.Retirada.Nome != "Bia" {
		t.Fatalf("depois do prazo: %+v %v", seguinte, err)
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool { return item.evento == EventoRetiradaVencida }) {
		t.Fatal("o vencimento não entrou na auditoria")
	}

	nova := "senha-nova"
	if _, err := svc.Atualizar(context.Background(), gestor, criado.ID, Mudanca{Nome: "vm-web", Usuario: "root", Senha: &nova}); err != nil {
		t.Fatal(err)
	}
	depois, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || depois[0].Retirada != nil {
		t.Fatalf("trocar a senha manteve a retirada: %+v %v", depois, err)
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool {
		return item.evento == EventoDevolvido && !strings.Contains(item.texto(), nova)
	}) {
		t.Fatal("a troca não devolveu, ou a senha nova entrou na auditoria")
	}
}

func TestRotacaoTrocaASenhaDoCofreEMantemODestinoPorContaDeQuemAplica(t *testing.T) {
	repo := &memoria{}
	trilha := &capta{}
	svc := novo(t, repo, trilha)
	momento := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	svc.agora = func() time.Time { return momento }
	unidade := uuid.New()
	ana := naUnidade(auth.PermCofreReveal, unidade)
	ana.UserID = uuid.New()
	ana.Name = "Ana"
	gestor := naUnidade(auth.PermCofreManage, unidade)
	gestor.UserID = uuid.New()

	criado, err := svc.Criar(context.Background(), gestor, Entrada{
		Nome: "vm-web", Usuario: "root", Senha: "senha-da-vm", Notas: "porta 22",
		Tipo: domain.TipoMaquina, Endereco: "10.0.0.8", Unidade: &unidade, Rotaciona: true,
	})
	if err != nil || criado.ProximaRotacao == nil || !criado.ProximaRotacao.Equal(momento.Add(domain.IntervaloRotacao)) {
		t.Fatalf("cadastro: %+v %v", criado.ProximaRotacao, err)
	}
	if _, err := svc.Retirar(context.Background(), ana, criado.ID, "acesso", 60); err != nil {
		t.Fatal(err)
	}
	passada := momento.Add(-time.Hour)
	guardado := repo.itens[criado.ID]
	guardado.ProximaRotacao = &passada
	guardado.RotacaoPendente = false
	repo.guardar(guardado)
	if err := svc.RotacionarDevidas(context.Background()); err != nil {
		t.Fatal(err)
	}
	durante, err := svc.Revelar(context.Background(), ana, criado.ID)
	if err != nil || durante.Senha != "senha-da-vm" {
		t.Fatalf("rotacionou com a retirada aberta: %+v %v", durante.Senha, err)
	}
	if err := svc.Devolver(context.Background(), ana, criado.ID); err != nil {
		t.Fatal(err)
	}
	lista, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || len(lista) != 1 || !lista[0].RotacaoPendente {
		t.Fatalf("aviso: %+v %v", lista, err)
	}
	corpo, _ := json.Marshal(lista)
	if strings.Contains(string(corpo), "senha-gerada-teste") || strings.Contains(string(corpo), "senha-da-vm") {
		t.Fatalf("lista levou senha: %s", corpo)
	}
	nova, err := svc.Revelar(context.Background(), gestor, criado.ID)
	if err != nil || nova.Senha != "senha-gerada-teste" || nova.Notas != "porta 22" {
		t.Fatalf("senha nova: %+v %v", nova, err)
	}
	for _, item := range trilha.itens {
		if strings.Contains(item.texto(), "senha-gerada-teste") || strings.Contains(item.texto(), "senha-da-vm") {
			t.Fatalf("auditoria levou senha: %s", item.texto())
		}
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool { return item.evento == EventoRotacionado }) {
		t.Fatal("rotação sem auditoria")
	}
	if err := svc.MarcarAplicada(context.Background(), ana, criado.ID); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("quem só revela marcou: %v", err)
	}
	if err := svc.MarcarAplicada(context.Background(), gestor, criado.ID); err != nil {
		t.Fatal(err)
	}
	livre, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || livre[0].RotacaoPendente {
		t.Fatalf("aviso ficou: %+v %v", livre, err)
	}

	momento = momento.Add(domain.IntervaloRotacao + time.Hour)
	if err := svc.RotacionarDevidas(context.Background()); err != nil {
		t.Fatal(err)
	}
	deNovo, err := svc.Listar(context.Background(), naUnidade(auth.PermCofreRead, unidade))
	if err != nil || !deNovo[0].RotacaoPendente {
		t.Fatalf("os 30 dias não rotacionaram: %+v %v", deNovo, err)
	}
}

func novo(t *testing.T, repo domain.Repository, trilha trilha) *Service {
	t.Helper()
	cifra, err := caixa.Nova("MTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTE=")
	if err != nil {
		t.Fatal(err)
	}
	return &Service{
		repo: repo, caixa: cifra, correr: direto{}, trilha: trilha, confirmador: aceitaSenha{}, agora: time.Now,
		gerar: func() (string, error) { return "senha-gerada-teste", nil },
	}
}

type aceitaSenha struct{}

func (aceitaSenha) Confirmar(_ context.Context, _ uuid.UUID, _ string, _ bool, senha string) error {
	if senha != "acesso" {
		return domain.ErrSenhaRecusada
	}
	return nil
}

func admin() auth.Identity {
	return auth.Identity{Roles: []string{auth.RoleAdmin}}
}

func naUnidade(perm auth.Permission, unidade uuid.UUID) auth.Identity {
	return nasUnidades(perm, unidade)
}

func nasUnidades(perm auth.Permission, unidades ...uuid.UUID) auth.Identity {
	escopos := make([]auth.Scope, 0, len(unidades))
	for _, unidade := range unidades {
		id := unidade
		escopos = append(escopos, auth.Scope{
			Permissions: []string{string(perm)},
			UnidadeID:   &id,
			Unidades:    []uuid.UUID{id},
		})
	}
	return auth.Identity{Scopes: escopos}
}

type direto struct{}

func (direto) Em(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return fn(ctx, nil)
}

type captaItem struct {
	evento  string
	payload map[string]any
}

func (c captaItem) texto() string {
	b, _ := json.Marshal(c.payload)
	return string(b)
}

type capta struct{ itens []captaItem }

func (c *capta) Registrar(_ context.Context, _ pgx.Tx, evento string, _ uuid.UUID, payload map[string]any) error {
	c.itens = append(c.itens, captaItem{evento: evento, payload: payload})
	return nil
}

type memoria struct {
	itens     map[uuid.UUID]domain.Selado
	vinculos  [][2]uuid.UUID
	retiradas []retiradaMem
	pastas    map[uuid.UUID]domain.Pasta
	membros   []membroMem
}

type membroMem struct {
	PastaID uuid.UUID
	domain.Membro
}

func (m *memoria) passaPasta(f domain.Filtro, item domain.Selado) bool {
	if !f.FiltrarPastas || item.PastaID == nil {
		return true
	}
	if f.GereTudo {
		return true
	}
	if item.UnidadeID != nil && slices.Contains(f.GereUnidades, *item.UnidadeID) {
		return true
	}
	for _, mb := range m.membros {
		if mb.PastaID == *item.PastaID && mb.UsuarioID == f.Usuario {
			return true
		}
	}
	return false
}

type retiradaMem struct {
	domain.RetiradaGravada
	devolvida *time.Time
}

func (m *memoria) guardar(item domain.Selado) {
	if m.itens == nil {
		m.itens = map[uuid.UUID]domain.Selado{}
	}
	m.itens[item.ID] = item
}

func visivel(f domain.Filtro, unidade *uuid.UUID) bool {
	if !f.Restrito {
		return true
	}
	if unidade == nil {
		return false
	}
	return slices.Contains(f.Unidades, *unidade)
}

func (m *memoria) List(_ context.Context, _ database.DBTX, f domain.Filtro) ([]domain.Selado, error) {
	out := []domain.Selado{}
	for _, item := range m.itens {
		if visivel(f, item.UnidadeID) && m.passaPasta(f, item) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *memoria) Get(_ context.Context, _ database.DBTX, id uuid.UUID, f domain.Filtro) (domain.Selado, error) {
	item, ok := m.itens[id]
	if !ok || !visivel(f, item.UnidadeID) || !m.passaPasta(f, item) {
		return domain.Selado{}, domain.ErrNaoEncontrado
	}
	return item, nil
}

func (m *memoria) Insert(_ context.Context, _ database.DBTX, item domain.Selado) error {
	m.guardar(item)
	return nil
}

func (m *memoria) Update(_ context.Context, _ database.DBTX, item domain.Selado, f domain.Filtro) error {
	atual, ok := m.itens[item.ID]
	if !ok || !visivel(f, atual.UnidadeID) || !m.passaPasta(f, atual) {
		return domain.ErrNaoEncontrado
	}
	m.guardar(item)
	return nil
}

func (m *memoria) Delete(_ context.Context, _ database.DBTX, id uuid.UUID, f domain.Filtro) error {
	atual, ok := m.itens[id]
	if !ok || !visivel(f, atual.UnidadeID) || !m.passaPasta(f, atual) {
		return domain.ErrNaoEncontrado
	}
	delete(m.itens, id)
	restantes := make([]retiradaMem, 0, len(m.retiradas))
	for _, r := range m.retiradas {
		if r.SegredoID != id {
			restantes = append(restantes, r)
		}
	}
	m.retiradas = restantes
	proximos := make([][2]uuid.UUID, 0, len(m.vinculos))
	for _, v := range m.vinculos {
		if v[0] != id && v[1] != id {
			proximos = append(proximos, v)
		}
	}
	m.vinculos = proximos
	return nil
}

func (m *memoria) ListarAdministracao(_ context.Context, _ database.DBTX, f domain.Filtro) (map[uuid.UUID][]uuid.UUID, error) {
	out := map[uuid.UUID][]uuid.UUID{}
	for _, v := range m.vinculos {
		painel, okP := m.itens[v[0]]
		maquina, okM := m.itens[v[1]]
		if okP && okM && visivel(f, painel.UnidadeID) && visivel(f, maquina.UnidadeID) {
			out[v[0]] = append(out[v[0]], v[1])
		}
	}
	return out, nil
}

func (m *memoria) SubstituirAdministracao(_ context.Context, _ database.DBTX, painel uuid.UUID, maquinas []uuid.UUID, f domain.Filtro) error {
	kept := make([][2]uuid.UUID, 0, len(m.vinculos)+len(maquinas))
	for _, v := range m.vinculos {
		if v[0] != painel {
			kept = append(kept, v)
			continue
		}
		maquina, ok := m.itens[v[1]]
		if ok && visivel(f, maquina.UnidadeID) {
			continue
		}
		kept = append(kept, v)
	}
	for _, id := range maquinas {
		kept = append(kept, [2]uuid.UUID{painel, id})
	}
	m.vinculos = kept
	return nil
}

func (m *memoria) Travar(_ context.Context, _ database.DBTX, id uuid.UUID) error {
	if _, ok := m.itens[id]; !ok {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (m *memoria) aberta(segredo uuid.UUID) (retiradaMem, bool) {
	for _, item := range m.retiradas {
		if item.SegredoID == segredo && item.devolvida == nil {
			return item, true
		}
	}
	return retiradaMem{}, false
}

func (m *memoria) RetiradaAberta(_ context.Context, _ database.DBTX, segredo uuid.UUID) (domain.RetiradaGravada, error) {
	item, ok := m.aberta(segredo)
	if !ok {
		return domain.RetiradaGravada{}, domain.ErrNaoEncontrado
	}
	return item.RetiradaGravada, nil
}

func (m *memoria) InserirRetirada(_ context.Context, _ database.DBTX, item domain.RetiradaGravada) error {
	if _, ok := m.aberta(item.SegredoID); ok {
		return domain.ErrOcupado
	}
	m.retiradas = append(m.retiradas, retiradaMem{RetiradaGravada: item})
	return nil
}

func (m *memoria) RenovarRetirada(_ context.Context, _ database.DBTX, id uuid.UUID, ate time.Time, nome string) error {
	for i := range m.retiradas {
		if m.retiradas[i].ID == id && m.retiradas[i].devolvida == nil {
			m.retiradas[i].Ate = ate
			m.retiradas[i].Nome = nome
			return nil
		}
	}
	return domain.ErrNaoEncontrado
}

func (m *memoria) EncerrarRetirada(_ context.Context, _ database.DBTX, id uuid.UUID, quando time.Time) error {
	for i := range m.retiradas {
		if m.retiradas[i].ID == id && m.retiradas[i].devolvida == nil {
			m.retiradas[i].devolvida = &quando
			return nil
		}
	}
	return domain.ErrNaoEncontrado
}

func (m *memoria) ListarRetiradas(_ context.Context, _ database.DBTX, f domain.Filtro, agora time.Time) (map[uuid.UUID]domain.Retirada, error) {
	out := map[uuid.UUID]domain.Retirada{}
	for _, item := range m.retiradas {
		segredo, ok := m.itens[item.SegredoID]
		if !ok || item.devolvida != nil || !item.Ate.After(agora) || !visivel(f, segredo.UnidadeID) {
			continue
		}
		out[item.SegredoID] = domain.Retirada{UsuarioID: item.UsuarioID, Nome: item.Nome, Ate: item.Ate}
	}
	return out, nil
}

func (m *memoria) ListarParaRotacionar(_ context.Context, _ database.DBTX, agora time.Time) ([]domain.Selado, error) {
	out := []domain.Selado{}
	for _, item := range m.itens {
		if item.Rotaciona && !item.RotacaoPendente && item.ProximaRotacao != nil && !item.ProximaRotacao.After(agora) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *memoria) guardarPasta(p domain.Pasta) {
	if m.pastas == nil {
		m.pastas = map[uuid.UUID]domain.Pasta{}
	}
	m.pastas[p.ID] = p
}

func (m *memoria) ListarPastas(_ context.Context, _ database.DBTX, f domain.Filtro) ([]domain.Pasta, error) {
	out := []domain.Pasta{}
	for _, p := range m.pastas {
		gerencia := f.GereTudo || (p.UnidadeID != nil && slices.Contains(f.GereUnidades, *p.UnidadeID))
		if gerencia && !f.Restrito {
			out = append(out, p)
			continue
		}
		if gerencia && visivel(f, p.UnidadeID) {
			out = append(out, p)
			continue
		}
		if m.membroDe(p.ID, f.Usuario) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *memoria) membroDe(pasta, usuario uuid.UUID) bool {
	for _, mb := range m.membros {
		if mb.PastaID == pasta && mb.UsuarioID == usuario {
			return true
		}
	}
	return false
}

func (m *memoria) ObterPasta(_ context.Context, _ database.DBTX, id uuid.UUID) (domain.Pasta, error) {
	p, ok := m.pastas[id]
	if !ok {
		return domain.Pasta{}, domain.ErrNaoEncontrado
	}
	p.Membros = m.membrosDa(id)
	return p, nil
}

func (m *memoria) membrosDa(pasta uuid.UUID) []domain.Membro {
	out := []domain.Membro{}
	for _, mb := range m.membros {
		if mb.PastaID == pasta {
			out = append(out, mb.Membro)
		}
	}
	return out
}

func (m *memoria) InserirPasta(_ context.Context, _ database.DBTX, p domain.Pasta) error {
	m.guardarPasta(p)
	return m.SubstituirMembros(context.Background(), nil, p.ID, p.Membros)
}

func (m *memoria) ContarNaPasta(_ context.Context, _ database.DBTX, pasta uuid.UUID) (int, error) {
	n := 0
	for _, item := range m.itens {
		if item.PastaID != nil && *item.PastaID == pasta {
			n++
		}
	}
	return n, nil
}

func (m *memoria) ExcluirPasta(_ context.Context, _ database.DBTX, id uuid.UUID) error {
	if _, ok := m.pastas[id]; !ok {
		return domain.ErrNaoEncontrado
	}
	delete(m.pastas, id)
	restantes := []membroMem{}
	for _, mb := range m.membros {
		if mb.PastaID != id {
			restantes = append(restantes, mb)
		}
	}
	m.membros = restantes
	for segredoID, item := range m.itens {
		if item.PastaID != nil && *item.PastaID == id {
			item.PastaID = nil
			m.itens[segredoID] = item
		}
	}
	return nil
}

func (m *memoria) SubstituirMembros(_ context.Context, _ database.DBTX, pasta uuid.UUID, membros []domain.Membro) error {
	restantes := []membroMem{}
	for _, mb := range m.membros {
		if mb.PastaID != pasta {
			restantes = append(restantes, mb)
		}
	}
	for _, mb := range membros {
		restantes = append(restantes, membroMem{PastaID: pasta, Membro: mb})
	}
	m.membros = restantes
	if p, ok := m.pastas[pasta]; ok {
		p.Membros = membros
		m.pastas[pasta] = p
	}
	return nil
}

func (m *memoria) Membro(_ context.Context, _ database.DBTX, pasta, usuario uuid.UUID) (domain.Membro, error) {
	for _, mb := range m.membros {
		if mb.PastaID == pasta && mb.UsuarioID == usuario {
			return mb.Membro, nil
		}
	}
	return domain.Membro{}, domain.ErrNaoEncontrado
}

func (m *memoria) Pessoas(context.Context, database.DBTX, *uuid.UUID) ([]domain.Pessoa, error) {
	return []domain.Pessoa{}, nil
}

func (m *memoria) UsuariosExistem(context.Context, database.DBTX, []uuid.UUID) (bool, error) {
	return true, nil
}

type contaSenha struct{ chamadas int }

func (c *contaSenha) Confirmar(context.Context, uuid.UUID, string, bool, string) error {
	c.chamadas++
	return nil
}

func TestPastaEscondeDeQuemNaoEstaNela(t *testing.T) {
	repo := &memoria{}
	svc := novo(t, repo, &capta{})
	unidade := uuid.New()
	gestor := naUnidade(auth.PermCofreManage, unidade)
	gestor.UserID = uuid.New()
	leitura := naUnidade(auth.PermCofreReveal, unidade)
	leitura.UserID = uuid.New()
	membroID := uuid.New()
	membro := naUnidade(auth.PermCofreReveal, unidade)
	membro.UserID = membroID
	soLe := naUnidade(auth.PermCofreReveal, unidade)
	soLe.UserID = uuid.New()

	aberto, err := svc.Criar(context.Background(), gestor, Entrada{Nome: "Aberto", Usuario: "a", Senha: "senha-aberta", Unidade: &unidade})
	if err != nil {
		t.Fatal(err)
	}
	pasta, err := svc.CriarPasta(context.Background(), gestor, "Painéis", &unidade, []domain.Membro{
		{UsuarioID: membroID, PodeRevelar: true},
		{UsuarioID: soLe.UserID, PodeRevelar: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	guardado, err := svc.Atualizar(context.Background(), gestor, aberto.ID, Mudanca{
		Nome: "Aberto", Usuario: "a", MoverPasta: true, Pasta: &pasta.ID,
	})
	if err != nil || guardado.PastaID == nil || *guardado.PastaID != pasta.ID {
		t.Fatalf("mover para a pasta: %+v %v", guardado, err)
	}
	outro, err := svc.Criar(context.Background(), gestor, Entrada{Nome: "Solto", Usuario: "b", Senha: "senha-solta", Unidade: &unidade})
	if err != nil {
		t.Fatal(err)
	}

	lista, err := svc.Listar(context.Background(), leitura)
	if err != nil || len(lista) != 1 || lista[0].ID != outro.ID {
		t.Fatalf("quem não está na pasta vê o segredo: %+v %v", lista, err)
	}
	confirma := &contaSenha{}
	svc.confirmador = confirma
	if _, err := svc.Revelar(context.Background(), leitura, guardado.ID); !errors.Is(err, domain.ErrNaoEncontrado) || confirma.chamadas != 0 {
		t.Fatalf("revelou sem estar na pasta: %v chamadas=%d", err, confirma.chamadas)
	}
	if _, err := svc.Revelar(context.Background(), soLe, guardado.ID); !errors.Is(err, domain.ErrSemPermissao) || confirma.chamadas != 0 {
		t.Fatalf("leu a senha sem poder revelar: %v chamadas=%d", err, confirma.chamadas)
	}
	svc.confirmador = aceitaSenha{}
	revelado, err := svc.Revelar(context.Background(), membro, guardado.ID)
	if err != nil || revelado.Senha != "senha-aberta" {
		t.Fatalf("membro: %+v %v", revelado, err)
	}
	var invalida domain.EntradaInvalida
	if err := svc.ExcluirPasta(context.Background(), gestor, pasta.ID); !errors.As(err, &invalida) {
		t.Fatalf("trava: %v", err)
	}
}
