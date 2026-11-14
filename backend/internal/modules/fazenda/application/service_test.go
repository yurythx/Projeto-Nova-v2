package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

type mem struct {
	itens    map[uuid.UUID]domain.Processo
	codigo   map[string]uuid.UUID
	falha    string
	semLista bool
}

func (m *mem) Inserir(_ context.Context, _ database.DBTX, p domain.Processo) error {
	if m.falha == "inserir" {
		return errors.New("inserir")
	}
	m.itens[p.ID] = p
	m.codigo[p.Codigo] = p.ID
	return nil
}

func (m *mem) Atualizar(_ context.Context, _ database.DBTX, p domain.Processo) error {
	if m.falha == "atualizar" {
		return errors.New("atualizar")
	}
	if _, ok := m.itens[p.ID]; !ok {
		return domain.ErrNaoEncontrado
	}
	m.itens[p.ID] = p
	return nil
}

func (m *mem) Buscar(_ context.Context, _ database.DBTX, id uuid.UUID) (domain.Processo, error) {
	if m.falha == "buscar" {
		return domain.Processo{}, errors.New("buscar")
	}
	p, ok := m.itens[id]
	if !ok {
		return domain.Processo{}, domain.ErrNaoEncontrado
	}
	return p, nil
}

func (m *mem) Listar(_ context.Context, _ database.DBTX, titular string) ([]domain.Processo, error) {
	if m.falha == "listar" {
		return nil, errors.New("listar")
	}
	if m.semLista {
		return nil, nil
	}
	var out []domain.Processo
	for _, p := range m.itens {
		if titular == "" || p.Titular == titular {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mem) Proximo(_ context.Context, _ database.DBTX, _ int) (int, error) {
	if m.falha == "proximo" {
		return 0, errors.New("proximo")
	}
	return len(m.itens) + 1, nil
}

func (m *mem) GuardarPeca(context.Context, database.DBTX, uuid.UUID, string, string) error {
	return nil
}

func (m *mem) ListarPecas(context.Context, database.DBTX, uuid.UUID) ([]domain.PecaArquivo, error) {
	return nil, nil
}

func (m *mem) PorCodigo(_ context.Context, _ database.DBTX, codigo string) (domain.Processo, error) {
	if m.falha == "codigo" {
		return domain.Processo{}, errors.New("codigo")
	}
	id, ok := m.codigo[codigo]
	if !ok {
		return domain.Processo{}, domain.ErrNaoEncontrado
	}
	return m.itens[id], nil
}

type trilhaFalsa struct {
	falha  bool
	avisos []string
}

func (t *trilhaFalsa) Registrar(_ context.Context, _ pgx.Tx, _, _ string, _ any, aviso string) error {
	if t.falha {
		return errors.New("trilha")
	}
	t.avisos = append(t.avisos, aviso)
	return nil
}

type govFalso struct {
	id  govbr.Identidade
	err error
}

func (g govFalso) Trocar(govbr.Config, string, string) (govbr.Identidade, error) {
	return g.id, g.err
}

func TestCaixaDoProtocolo(t *testing.T) {
	s, m, _ := novo(t)
	s.sino = notificacoes.NewService(nil, nil, nil)
	s.transacao = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, &txFalsa{})
	}
	quem := cidadao("ana")
	quem.UserID = uuid.New()
	var gravada notificacoes.Nova
	anterior := registrarCaixa
	registrarCaixa = func(_ context.Context, _ database.DBTX, _ []uuid.UUID, n notificacoes.Nova) ([]notificacoes.Enviada, error) {
		gravada = n
		if utf8.RuneCountInString(n.Mensagem) > 1000 {
			t.Fatal("mensagem longa")
		}
		return []notificacoes.Enviada{{Usuario: quem.UserID}}, nil
	}
	t.Cleanup(func() { registrarCaixa = anterior })
	avisoAnterior := avisoDeProtocolo
	avisoDeProtocolo = func(string) (string, error) { return strings.Repeat("á", 1200), nil }
	t.Cleanup(func() { avisoDeProtocolo = avisoAnterior })
	p, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()})
	if err != nil || p.Destinatario != quem.UserID || utf8.RuneCountInString(gravada.Mensagem) != 1000 || gravada.Modulo != "fazenda" {
		t.Fatalf("%+v %v %+v", p, err, gravada)
	}
	if _, err := s.caixa(context.Background(), nil, p, "x", "aviso"); err != nil {
		t.Fatal(err)
	}
	registrarCaixa = func(context.Context, database.DBTX, []uuid.UUID, notificacoes.Nova) ([]notificacoes.Enviada, error) {
		return nil, errors.New("caixa")
	}
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()}); err == nil {
		t.Fatal("caixa")
	}
	var id uuid.UUID
	for chave := range m.itens {
		id = chave
		break
	}
	if _, err := s.Avisar(context.Background(), fiscal(), id, "protocolo aberto", "acompanhe na área restrita"); err == nil {
		t.Fatal("caixa na alteração")
	}
	if len(m.itens) == 0 {
		t.Fatal("processo")
	}
	if s.ComSino(nil) != s || s.ComSino(notificacoes.NewService(nil, nil, nil)).sino == nil {
		t.Fatal("sino")
	}
}

func TestPainel(t *testing.T) {
	s, m, tr := novo(t)
	ctx := context.Background()
	if _, err := s.Painel(ctx, cidadao("ana")); err == nil {
		t.Fatal("painel do contribuinte")
	}
	lista, err := s.Painel(ctx, fiscal())
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	if n, err := s.Alertar(ctx, fiscal()); err != nil || n != 0 {
		t.Fatal(err)
	}
	despacho := time.Date(2026, 4, 1, 12, 0, 0, 0, domain.Cuiaba)
	s.agora = func() time.Time { return despacho.AddDate(0, 0, 70) }
	id := uuid.New()
	m.itens[id] = domain.Processo{ID: id, Numero: "000001/2026", Programa: domain.ProgramaAvanca, Titular: "ana", DespachoEm: despacho}
	m.itens[uuid.New()] = domain.Processo{ID: uuid.New(), Numero: "000002/2026", Programa: domain.ProgramaRegulariza, DespachoEm: despacho}
	lista, err = s.Painel(ctx, fiscal())
	if err != nil || !strings.Contains(fmtPrazos(lista), "marco de início") {
		t.Fatalf("%v %+v", err, lista)
	}
	if _, err := s.Alertar(ctx, cidadao("ana")); err == nil {
		t.Fatal("alerta do contribuinte")
	}
	n, err := s.Alertar(ctx, fiscal())
	if err != nil || n != 1 || len(tr.avisos) != 1 || strings.Contains(tr.avisos[0], "R$") {
		t.Fatalf("%d %v %v", n, err, tr.avisos)
	}
	if n, err = s.Alertar(ctx, fiscal()); err != nil || n != 0 {
		t.Fatal(err)
	}
	m.itens[id] = domain.Processo{ID: id, Numero: "123.456.789-01", Programa: domain.ProgramaAvanca, DespachoEm: despacho}
	if _, err := s.Alertar(ctx, fiscal()); err == nil {
		t.Fatal("alerta com CPF")
	}
	m.falha = "listar"
	if _, err := s.Painel(ctx, fiscal()); err == nil {
		t.Fatal("listar painel")
	}
	if _, err := s.Alertar(ctx, fiscal()); err == nil {
		t.Fatal("listar alerta")
	}
	s.intervalo = time.Millisecond
	encerrado, cancelar := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancelar()
	if err := s.RunAlertas(encerrado); err != nil {
		t.Fatal(err)
	}
	m.falha = ""
	s.intervalo = 0
	ctxParado, parar := context.WithCancel(context.Background())
	parar()
	if err := s.RunAlertas(ctxParado); err != nil {
		t.Fatal(err)
	}
}

func fmtPrazos(lista []domain.PrazoPainel) string {
	var nomes string
	for _, item := range lista {
		nomes += item.Nome
	}
	return nomes
}

func novo(t *testing.T) (*Service, *mem, *trilhaFalsa) {
	t.Helper()
	m := &mem{itens: map[uuid.UUID]domain.Processo{}, codigo: map[string]uuid.UUID{}}
	tr := &trilhaFalsa{}
	s := NewService(nil, m, tr, govbr.Config{}, nil)
	s.transacao = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	s.agora = func() time.Time { return time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC) }
	return s, m, tr
}

func cidadao(sub string) auth.Identity { return auth.Identity{Subject: sub} }

func fiscal() auth.Identity {
	return auth.Identity{Subject: "fiscal", Permissions: []string{string(auth.PermFazendaAnalisar), string(auth.PermFazendaDecidir), string(auth.PermFazendaProtocolar)}}
}

func docsAvanca() []domain.Documento {
	tipos := []string{"documento_unico_regularidade", "matricula", "orcamento_cronograma", "quadro_metas", "documento_pessoal", "declaracao_nao_inicio"}
	out := make([]domain.Documento, len(tipos))
	for i, tipo := range tipos {
		out[i] = domain.Documento{Tipo: tipo, Presente: true}
		if tipo == "matricula" {
			out[i].EmitidaEm = time.Date(2026, 6, 1, 0, 0, 0, 0, domain.Cuiaba)
		}
	}
	return out
}

func TestMapErrorEGov(t *testing.T) {
	if MapError(nil) != nil {
		t.Fatal("nil")
	}
	if _, ok := apperrors.As(MapError(domain.ErrNaoEncontrado)); !ok {
		t.Fatal("não encontrado")
	}
	if _, ok := apperrors.As(MapError(domain.Recusa{Msg: "x"})); !ok {
		t.Fatal("recusa")
	}
	if _, ok := apperrors.As(MapError(govbr.ErrSemCredencial)); !ok {
		t.Fatal("gov.br")
	}
	if MapError(apperrors.Forbidden("não")) == nil {
		t.Fatal("já traduzido")
	}
	if _, ok := apperrors.As(MapError(errors.New("banco"))); !ok {
		t.Fatal("interno")
	}
	s, _, _ := novo(t)
	if _, err := s.IniciarGovBR(); !errors.Is(err, govbr.ErrSemCredencial) {
		t.Fatal(err)
	}
	s.gov = govbr.Config{ClientID: "id", ClientSecret: "seg", RedirectURL: "https://x/cb"}
	if d, err := s.IniciarGovBR(); err != nil || d.URL == "" {
		t.Fatal(err)
	}
	if _, err := (govPadrao{}).Trocar(govbr.Config{}, "c", "v"); !errors.Is(err, govbr.ErrSemCredencial) {
		t.Fatal("troca padrão")
	}
	s.http = govFalso{err: errors.New("rede")}
	if _, err := s.TrocarGovBR("c", "v"); err == nil {
		t.Fatal("troca")
	}
	vazio := NewService(nil, &mem{itens: map[uuid.UUID]domain.Processo{}, codigo: map[string]uuid.UUID{}}, nil, govbr.Config{}, nil)
	vazio.transacao = s.transacao
	vazio.agora = s.agora
	if _, err := vazio.Abrir(context.Background(), cidadao("ana"), domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()}); err != nil {
		t.Fatal(err)
	}
	antes := avisoDeProtocolo
	avisoDeProtocolo = func(string) (string, error) { return "", errors.New("aviso") }
	t.Cleanup(func() { avisoDeProtocolo = antes })
	if _, err := vazio.Abrir(context.Background(), cidadao("ana"), domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()}); err == nil {
		t.Fatal("aviso do protocolo")
	}
	avisoDeProtocolo = antes
	s.http = govFalso{id: govbr.Identidade{Sub: "ana", Nome: "Ana", Email: "a@x", CPF: "00000000000"}}
	sess, err := s.TrocarGovBR("c", "v")
	if err != nil || sess.Sub != "ana" || sess.Nome != "Ana" {
		t.Fatal(err)
	}
}

func TestAbrirListarEBuscar(t *testing.T) {
	s, m, tr := novo(t)
	quem := cidadao("ana")
	p, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()})
	if err != nil || p.Titular != "ana" || p.Numero == "" || len(tr.avisos) != 1 {
		t.Fatalf("%+v %v avisos=%v", p, err, tr.avisos)
	}
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, Titular: "bia"}); err == nil {
		t.Fatal("em nome sem permissão")
	}
	fiscal := fiscal()
	if _, err := s.Abrir(context.Background(), fiscal, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, Titular: "bia", TemContrato: true, Documentos: docsAvanca()}); err != nil {
		t.Fatal(err)
	}
	lista, err := s.Listar(context.Background(), quem)
	if err != nil || len(lista) != 1 {
		t.Fatalf("contribuinte vê só o próprio: %d %v", len(lista), err)
	}
	todos, err := s.Listar(context.Background(), auth.Identity{Subject: "x", Permissions: []string{string(auth.PermFazendaDecidir)}})
	if err != nil || len(todos) < 2 {
		t.Fatalf("equipe: %d %v", len(todos), err)
	}
	m.semLista = true
	if lista, err = s.Listar(context.Background(), quem); err != nil || lista == nil || len(lista) != 0 {
		t.Fatal("lista vazia")
	}
	m.semLista = false
	m.falha = "listar"
	if _, err := s.Listar(context.Background(), quem); err == nil {
		t.Fatal("listar")
	}
	m.falha = ""
	if _, err := s.Buscar(context.Background(), quem, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Buscar(context.Background(), cidadao("outro"), p.ID); err == nil {
		t.Fatal("processo alheio")
	}
	if _, err := s.Buscar(context.Background(), fiscal, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Buscar(context.Background(), quem, uuid.New()); err == nil {
		t.Fatal("ausente")
	}
	m.falha = "buscar"
	if _, err := s.Buscar(context.Background(), quem, p.ID); err == nil {
		t.Fatal("busca")
	}
	m.falha = ""
	aut, err := s.Autenticidade(context.Background(), p.Codigo)
	if err != nil || !aut.Valido {
		t.Fatal(err)
	}
	if aut, err = s.Autenticidade(context.Background(), "nao"); err != nil || aut.Valido {
		t.Fatal(aut)
	}
	m.falha = "codigo"
	if _, err := s.Autenticidade(context.Background(), p.Codigo); err == nil {
		t.Fatal("código")
	}
	m.falha = "proximo"
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, Documentos: docsAvanca(), TemContrato: true}); err == nil {
		t.Fatal("sequência")
	}
	m.falha = "inserir"
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, Documentos: docsAvanca(), TemContrato: true}); err == nil {
		t.Fatal("inserir")
	}
	m.falha = ""
	tr.falha = true
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, Documentos: docsAvanca(), TemContrato: true}); err == nil {
		t.Fatal("trilha")
	}
	tr.falha = false
	m.falha = ""
	if _, err := s.Abrir(context.Background(), quem, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR}); err == nil {
		t.Fatal("checklist")
	}
}

func TestAtos(t *testing.T) {
	s, m, _ := novo(t)
	fiscal := fiscal()
	cid := cidadao("ana")
	p, err := s.Abrir(context.Background(), cid, domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.Exigir(ctx, cid, p.ID, false); err == nil {
		t.Fatal("exigência do contribuinte")
	}
	if _, err := s.Decidir(ctx, cid, p.ID); err == nil {
		t.Fatal("decisão do contribuinte")
	}
	if _, err := s.Recorrer(ctx, cid, p.ID, true); err == nil {
		t.Fatal("recurso do contribuinte")
	}
	if _, err := s.GuardarReducao(ctx, cid, p.ID, 2000, 5); err == nil {
		t.Fatal("redução do contribuinte")
	}
	if _, err := s.Parcelar(ctx, cid, p.ID, 10000, 1); err == nil {
		t.Fatal("parcela do contribuinte")
	}
	if _, err := s.Baixar(ctx, cid, p.ID, true); err == nil {
		t.Fatal("baixa do contribuinte")
	}
	if _, err := s.MarcarExecucao(ctx, cid, p.ID, true); err == nil {
		t.Fatal("execução do contribuinte")
	}
	if _, err := s.Avisar(ctx, cid, p.ID, "fato", "orientação"); err == nil {
		t.Fatal("aviso do contribuinte")
	}
	if _, err := s.Parcelar(ctx, fiscal, p.ID, 100, 2); err == nil {
		t.Fatal("mínimo")
	}
	salvo := m.itens[p.ID]
	salvo.Numero = "529.982.247-25"
	m.itens[p.ID] = salvo
	if _, err := s.Exigir(ctx, fiscal, p.ID, false); err == nil {
		t.Fatal("aviso com CPF")
	}
	salvo.Numero = p.Numero
	m.itens[p.ID] = salvo
	exigida, err := s.Exigir(ctx, fiscal, p.ID, false)
	if err != nil || exigida.Situacao != domain.SituacaoExigencia {
		t.Fatalf("%+v %v", exigida, err)
	}
	if _, err := s.Decidir(ctx, fiscal, p.ID); err == nil {
		t.Fatal("decisão com exigência aberta")
	}
	if _, err := s.Exigir(ctx, fiscal, p.ID, false); err == nil {
		t.Fatal("segunda exigência")
	}
	if _, err := s.Responder(ctx, cidadao("outro"), p.ID, true); err == nil {
		t.Fatal("resposta alheia")
	}
	respondida, err := s.Responder(ctx, cid, p.ID, true)
	if err != nil || respondida.Situacao != domain.SituacaoAnalise {
		t.Fatalf("%+v %v", respondida, err)
	}
	if _, err := s.Responder(ctx, cid, p.ID, true); err == nil {
		t.Fatal("sem exigência")
	}
	if _, err := s.Decidir(ctx, fiscal, uuid.New()); err == nil {
		t.Fatal("decidir ausente")
	}
	decidido, err := s.Decidir(ctx, fiscal, p.ID)
	if err != nil || decidido.Situacao != domain.SituacaoDecidido || decidido.DespachoEm.IsZero() {
		t.Fatal(err)
	}
	if _, err := s.Recorrer(ctx, fiscal, p.ID, false); err == nil {
		t.Fatal("rito")
	}
	if _, err := s.Recorrer(ctx, fiscal, p.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GuardarReducao(ctx, fiscal, p.ID, 0, 1); err == nil {
		t.Fatal("redução inválida")
	}
	if _, err := s.GuardarReducao(ctx, fiscal, p.ID, 2000, 5); err != nil {
		t.Fatal(err)
	}
	plano, err := s.Parcelar(ctx, fiscal, p.ID, 120000, 12)
	if err != nil || plano.Estado != domain.SituacaoAguardandoPagamento {
		t.Fatal(err)
	}
	if _, err := s.Baixar(ctx, fiscal, p.ID, false); err != nil {
		t.Fatal(err)
	}
	baixado, err := s.Baixar(ctx, fiscal, p.ID, true)
	if err != nil || baixado.Situacao != domain.SituacaoParcelado {
		t.Fatal(err)
	}
	marcado, err := s.MarcarExecucao(ctx, fiscal, p.ID, true)
	if err != nil || marcado.Execucao != "procuradoria" {
		t.Fatal(err)
	}
	if _, err := s.Avisar(ctx, fiscal, p.ID, "há débito", "veja"); err == nil {
		t.Fatal("aviso fiscal")
	}
	texto, err := s.Avisar(ctx, fiscal, p.ID, "registro informado", "acompanhe na área restrita")
	if err != nil || texto == "" {
		t.Fatal(err)
	}
	m.falha = "buscar"
	if _, err := s.Parcelar(ctx, fiscal, p.ID, 10000, 1); err == nil {
		t.Fatal("parcelar sem processo")
	}
	m.falha = "atualizar"
	if _, err := s.MarcarExecucao(ctx, fiscal, p.ID, false); err == nil {
		t.Fatal("atualizar")
	}
	m.falha = "buscar"
	if _, err := s.Decidir(ctx, fiscal, p.ID); err == nil {
		t.Fatal("buscar na alteração")
	}
	recibo, err := s.ReceberDUA(nil, 1)
	if err != nil || recibo.Multa != 20000 || strings.Contains(recibo.Recibo, "R$") {
		t.Fatal(err)
	}
	sim, err := s.Simular(domain.Simulacao{Inscricao: "1"})
	if err != nil || sim.Devido != 0 {
		t.Fatal(err)
	}
	if _, err := s.Simular(domain.Simulacao{}); err == nil {
		t.Fatal("simulação")
	}
}

type txFalsa struct{ falha bool }

func (t *txFalsa) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *txFalsa) Commit(context.Context) error          { return nil }
func (t *txFalsa) Rollback(context.Context) error        { return nil }
func (t *txFalsa) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *txFalsa) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *txFalsa) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *txFalsa) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *txFalsa) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if t.falha {
		return pgconn.CommandTag{}, errors.New("exec")
	}
	return pgconn.NewCommandTag("INSERT 1"), nil
}
func (t *txFalsa) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *txFalsa) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *txFalsa) Conn() *pgx.Conn                                         { return nil }

func TestTrilhaETransacaoPadrao(t *testing.T) {
	if err := (Nucleo{}).Registrar(context.Background(), nil, EventoAviso, "id", nil, ""); err == nil {
		t.Fatal("outbox ausente")
	}
	n := Nucleo{Out: outbox.NewWriter("nova.fazenda")}
	if err := n.Registrar(context.Background(), &txFalsa{falha: true}, EventoAviso, "id", nil, "acompanhe"); err == nil {
		t.Fatal("gravação")
	}
	if err := n.Registrar(context.Background(), &txFalsa{}, EventoAviso, "id", map[string]string{"n": "1"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := n.Registrar(context.Background(), &txFalsa{}, EventoAviso, "id", map[string]string{"n": "1"}, "acompanhe na área restrita"); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), "postgres://127.0.0.1:1/fazenda?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	semRede := NewService(pool, &mem{itens: map[uuid.UUID]domain.Processo{}, codigo: map[string]uuid.UUID{}}, &trilhaFalsa{}, govbr.Config{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := semRede.Abrir(ctx, cidadao("ana"), domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()}); err == nil {
		t.Fatal("banco ausente")
	}
	banco := dbtest.Pool(t)
	s := NewService(banco, &mem{itens: map[uuid.UUID]domain.Processo{}, codigo: map[string]uuid.UUID{}}, Nucleo{Out: outbox.NewWriter("nova.fazenda")}, govbr.Config{}, nil)
	s.agora = func() time.Time { return time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC) }
	if _, err := s.Abrir(context.Background(), cidadao("ana"), domain.Abertura{Programa: domain.ProgramaAvanca, Entrada: domain.EntradaGovBR, TemContrato: true, Documentos: docsAvanca()}); err != nil {
		t.Fatal(err)
	}
	if err := (Nucleo{Out: outbox.NewWriter("nova.fazenda")}).Registrar(context.Background(), nil, EventoAviso, "id", nil, "aviso"); err == nil {
		t.Fatal("tx nula")
	}
	err = database.WithTx(context.Background(), banco, func(ctx context.Context, tx pgx.Tx) error {
		n := Nucleo{Out: outbox.NewWriter("nova.fazenda")}
		if err := n.Registrar(ctx, tx, EventoAviso, "1", map[string]string{"n": "1"}, ""); err != nil {
			return err
		}
		return n.Registrar(ctx, tx, EventoAviso, "1", map[string]string{"n": "1"}, "acompanhe na área restrita")
	})
	if err != nil {
		t.Fatal(err)
	}
}
