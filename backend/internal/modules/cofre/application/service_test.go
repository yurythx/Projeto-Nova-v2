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

	criado, err := svc.Criar(context.Background(), admin(), "Correio", "ana", senha, "pista", &unidadeA)
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

	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreRead, unidadeA), criado.ID, "acesso"); !errors.Is(err, domain.ErrSemPermissao) {
		t.Fatalf("leitura revelou: %v", err)
	}
	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeB), criado.ID, "acesso"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("outra unidade: %v", err)
	}

	revelado, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeA), criado.ID, "acesso")
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
	criado, err := svc.Criar(context.Background(), naUnidade(auth.PermCofreManage, unidadeA), "Wifi", "rede", "segredo-wifi", "", &unidadeA)
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
	if _, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreReveal, unidadeA), criado.ID, "acesso"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("a origem ainda vê: %v", err)
	}
	revelado, err := svc.Revelar(context.Background(), naUnidade(auth.PermCofreManage, unidadeB), criado.ID, "acesso")
	if err != nil || revelado.Senha != "segredo-wifi" {
		t.Fatalf("destino: %+v %v", revelado, err)
	}
}

func TestTrocarSenhaNaoDeixaAAnteriorAbrir(t *testing.T) {
	repo := &memoria{}
	svc := novo(t, repo, &capta{})
	unidade := uuid.New()
	quem := naUnidade(auth.PermCofreManage, unidade)
	criado, err := svc.Criar(context.Background(), quem, "VPN", "ops", "antiga", "nota", &unidade)
	if err != nil {
		t.Fatal(err)
	}
	nova := "nova-senha"
	if _, err := svc.Atualizar(context.Background(), quem, criado.ID, Mudanca{Nome: "VPN", Usuario: "ops", Senha: &nova}); err != nil {
		t.Fatal(err)
	}
	revelado, err := svc.Revelar(context.Background(), quem, criado.ID, "acesso")
	if err != nil || revelado.Senha != nova || revelado.Notas != "nota" {
		t.Fatalf("troca: %+v %v", revelado, err)
	}
}

func TestSenhaDeAcessoErradaNaoRevela(t *testing.T) {
	repo := &memoria{}
	trilha := &capta{}
	svc := novo(t, repo, trilha)
	unidade := uuid.New()
	quem := naUnidade(auth.PermCofreReveal, unidade)
	quem.Username = "ana"
	criado, err := svc.Criar(context.Background(), naUnidade(auth.PermCofreManage, unidade), "VPN", "ops", "segredo-vpn", "", &unidade)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Revelar(context.Background(), quem, criado.ID, "errada"); !errors.Is(err, domain.ErrSenhaRecusada) {
		t.Fatalf("senha errada: %v", err)
	}
	if slices.ContainsFunc(trilha.itens, func(item captaItem) bool {
		return strings.Contains(item.texto(), "errada") || strings.Contains(item.texto(), "segredo-vpn")
	}) {
		t.Fatalf("a auditoria levou senha: %+v", trilha.itens)
	}
	if !slices.ContainsFunc(trilha.itens, func(item captaItem) bool { return item.evento == EventoAcessoRecusado }) {
		t.Fatal("falta o registro da recusa")
	}
}

func novo(t *testing.T, repo domain.Repository, trilha trilha) *Service {
	t.Helper()
	cifra, err := caixa.Nova("MTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTE=")
	if err != nil {
		t.Fatal(err)
	}
	return &Service{repo: repo, caixa: cifra, correr: direto{}, trilha: trilha, confirmador: aceitaSenha{}, agora: time.Now}
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
	itens map[uuid.UUID]domain.Selado
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
		if visivel(f, item.UnidadeID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *memoria) Get(_ context.Context, _ database.DBTX, id uuid.UUID, f domain.Filtro) (domain.Selado, error) {
	item, ok := m.itens[id]
	if !ok || !visivel(f, item.UnidadeID) {
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
	if !ok || !visivel(f, atual.UnidadeID) {
		return domain.ErrNaoEncontrado
	}
	m.guardar(item)
	return nil
}

func (m *memoria) Delete(_ context.Context, _ database.DBTX, id uuid.UUID, f domain.Filtro) error {
	atual, ok := m.itens[id]
	if !ok || !visivel(f, atual.UnidadeID) {
		return domain.ErrNaoEncontrado
	}
	delete(m.itens, id)
	return nil
}
