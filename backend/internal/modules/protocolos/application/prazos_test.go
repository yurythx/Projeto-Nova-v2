package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage/storagetest"
)

// cenarioAgenda monta um serviço com horário (segunda a sexta, 08:00 e 08:30).
func cenarioAgenda(t *testing.T) (*mem, *Service, uuid.UUID, domain.Servico) {
	t.Helper()
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Vacina"})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{
		UnidadeID: un, FormularioID: form.ID, Nome: "Vacina", ExigeAgendamento: true,
		Agenda: domain.Agenda{Dias: []int{1, 2, 3, 4, 5}, Inicio: "08:00", Fim: "09:00", DuracaoMin: 30, VagasDia: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m, s, un, serv
}

func TestFeriados(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	gestor := ctxGestao(un)
	so := ctxDe(uuid.New(), un, auth.PermProtocolosTratar)

	if _, err := s.ListarFeriados(context.Background()); err == nil {
		t.Fatal("sem sessão")
	}
	if _, err := s.SalvarFeriado(context.Background(), "2026-12-25", "Natal"); err == nil {
		t.Fatal("sem sessão")
	}
	if _, err := s.SalvarFeriado(so, "2026-12-25", "Natal"); err == nil {
		t.Fatal("sem gerir")
	}
	if _, err := s.SalvarFeriado(gestor, "25/12", "Natal"); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	m.falha = "salvar-feriado"
	if _, err := s.SalvarFeriado(gestor, "2026-12-25", "Natal"); err == nil {
		t.Fatal("banco")
	}
	m.falha = ""
	if f, err := s.SalvarFeriado(gestor, "2026-12-25", "Natal"); err != nil || f.Nome != "Natal" {
		t.Fatal(err)
	}
	lista, err := s.ListarFeriados(so)
	if err != nil || len(lista) != 1 {
		t.Fatal(lista, err)
	}
	m.falha = "nulo"
	if lista, err := s.ListarFeriados(so); err != nil || lista == nil {
		t.Fatal(err)
	}
	m.falha = ""
	if err := s.RemoverFeriado(so, "2026-12-25"); err == nil {
		t.Fatal("remover sem gerir")
	}
	if err := s.RemoverFeriado(gestor, "natal"); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if err := s.RemoverFeriado(gestor, "2026-01-01"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if err := s.RemoverFeriado(gestor, "2026-12-25"); err != nil || len(m.feriados) != 0 {
		t.Fatal(err)
	}

	if pessoas, err := s.PessoasDaUnidade(gestor, un); err != nil || pessoas == nil {
		t.Fatal(err)
	}
	if _, err := s.PessoasDaUnidade(so, un); err == nil {
		t.Fatal("pessoas sem gerir")
	}
	m.falha = "nulo"
	if pessoas, err := s.PessoasDaUnidade(gestor, un); err != nil || pessoas == nil {
		t.Fatal(err)
	}
	m.falha = ""
}

func TestHorariosLivresEFeriadoNaAgenda(t *testing.T) {
	m, s, un, serv := cenarioAgenda(t)
	ctx := ctxGestao(un)
	// Hoje é segunda, 16/11/2026, 15:00; sexta, 20, é feriado.
	m.feriados["2026-11-20"] = "Feriado municipal"

	if livres, err := s.HorariosLivres(ctx, serv.ID, "2026-11-17"); err != nil || strings.Join(livres, ",") != "08:00,08:30" {
		t.Fatal(livres, err)
	}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, false); err != nil {
		t.Fatal(err)
	}
	if livres, _ := s.HorariosLivres(ctx, serv.ID, "2026-11-17"); strings.Join(livres, ",") != "08:30" {
		t.Fatal(livres)
	}
	for _, dia := range []string{"2026-11-15", "2026-11-16", "2026-11-20", "2026-11-21"} {
		if livres, err := s.HorariosLivres(ctx, serv.ID, dia); err != nil || len(livres) != 0 {
			t.Fatal(dia, livres, err)
		}
	}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "João Lima", CPF: "98765432100", Data: "2026-11-20", Hora: "08:00"}, false); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("feriado não marca", err)
	}
	if _, err := s.HorariosLivres(ctx, serv.ID, "17/11"); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.HorariosLivres(ctx, uuid.New(), "2026-11-17"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	semAgenda := serv
	semAgenda.ExigeAgendamento = false
	m.servs[serv.ID] = semAgenda
	if _, err := s.HorariosLivres(ctx, serv.ID, "2026-11-17"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	m.servs[serv.ID] = serv
	m.falha = "horas"
	if _, err := s.HorariosLivres(ctx, serv.ID, "2026-11-17"); err == nil {
		t.Fatal("horas")
	}
	m.falha = "feriados"
	if _, err := s.HorariosLivres(ctx, serv.ID, "2026-11-17"); err == nil {
		t.Fatal("feriados")
	}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "João Lima", CPF: "98765432100"}, false); err == nil {
		t.Fatal("feriados na abertura")
	}
	m.falha = ""
}

func TestCalendarioQueFalha(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, false)
	ctx := ctxGestao(un)
	destino := uuid.New()
	m.falha = "feriados"
	if _, err := s.ListarPedidos(ctx); err == nil {
		t.Fatal("lista")
	}
	if _, err := s.BuscarPedido(ctx, ped.ID); err == nil {
		t.Fatal("ficha")
	}
	if _, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "envie"}); err == nil {
		t.Fatal("complemento")
	}
	m.falha = ""
	serv := m.servs[ped.ServicoID]
	serv.Workflow = domain.Workflow{Ligado: true, Etapas: []domain.Etapa{{ID: "a", Ordem: 1}, {ID: "b", Ordem: 2}}}
	m.servs[ped.ServicoID] = serv
	p := m.pedidos[ped.ID]
	p.EtapaID = "a"
	m.pedidos[ped.ID] = p
	m.falha = "feriados"
	if _, err := s.Avancar(ctx, ped.ID); err == nil {
		t.Fatal("avanço")
	}
	m.falha = ""
	if _, err := s.Encaminhar(ctx, ped.ID, destino, "é de lá", 2); err != nil {
		t.Fatal(err)
	}
	m.falha = "feriados"
	if _, err := s.DecidirEncaminhamento(ctxGestao(destino), ped.ID, true, ""); err == nil {
		t.Fatal("encaminhamento")
	}
	m.falha = ""
}

func TestSEIPrazoEAnexosDaMesa(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, false)
	ctx := ctxGestao(un)
	sei := "12345.000001/2026-11"
	ruim := "x"

	if editado, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "abriu no SEI", ProcessoSEI: &sei}); err != nil || editado.ProcessoSEI != sei {
		t.Fatal(err)
	}
	if _, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "x", ProcessoSEI: &ruim}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}

	// Sem armazenamento, nenhum anexo passa.
	if _, err := s.PedirAnexoMesa(ctx, ped.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("sem armazenamento")
	}
	chave := []string{"protocolos/mesa/" + ped.ID.String() + "/a.pdf"}
	if _, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "envie", Anexos: chave}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.AnotarCom(ctx, ped.ID, "nota", chave); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.AvancarCom(ctx, ped.ID, chave); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.ConcluirCom(ctx, ped.ID, Conclusao{Decisao: domain.DecisaoFavoravel, Justificativa: "ok", Anexos: chave}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}

	loja := storagetest.New()
	s.ComArquivos(Arquivos{Store: loja, Bucket: "nova"})
	if _, err := s.PedirAnexoMesa(ctxDe(uuid.New(), uuid.New(), auth.PermProtocolosTratar), ped.ID, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("fora da unidade")
	}
	if _, err := s.PedirAnexoMesa(ctx, uuid.New(), "a.pdf", "application/pdf"); err == nil {
		t.Fatal("pedido")
	}
	enviar := func() []string {
		ticket, err := s.PedirAnexoMesa(ctx, ped.ID, "parecer.pdf", "application/pdf")
		if err != nil || !strings.HasPrefix(ticket.ObjectKey, "protocolos/mesa/"+ped.ID.String()+"/") {
			t.Fatal(ticket.ObjectKey, err)
		}
		if err := loja.Put(ctx, "nova", ticket.ObjectKey, strings.NewReader("%PDF"), 4, "application/pdf"); err != nil {
			t.Fatal(err)
		}
		return []string{ticket.ObjectKey}
	}

	if _, err := s.AnotarCom(ctx, ped.ID, "parecer anexado", enviar()); err != nil {
		t.Fatal(err)
	}
	anexos := enviar()
	comp, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "envie o RG", PrazoDias: 3, Anexos: anexos})
	if err != nil || comp.ComplementoPrazo == nil {
		t.Fatal(err)
	}
	if url, err := s.LinkArquivo(ctx, ped.ID, anexos[0]); err != nil || url == "" {
		t.Fatal("o anexo da mesa abre na ficha", err)
	}
	if _, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "de novo"}); !errors.Is(err, domain.ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := s.Responder(ctx, ped.ID, "recebido"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "outra", PrazoDias: 999}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.ConcluirCom(ctx, ped.ID, Conclusao{Decisao: domain.DecisaoFavoravel, Justificativa: "ok", ProcessoSEI: &ruim}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	outro := "54321.000002/2026-22"
	concluido, err := s.ConcluirCom(ctx, ped.ID, Conclusao{Decisao: domain.DecisaoFavoravel, Justificativa: "decisão no SEI", ProcessoSEI: &outro, Anexos: enviar()})
	if err != nil || concluido.ProcessoSEI != outro || concluido.Status != domain.StatusConcluido {
		t.Fatal(err)
	}
	if _, err := s.PedirAnexoMesa(ctx, ped.ID, "a.pdf", "application/pdf"); !errors.Is(err, domain.ErrEstadoFinal) {
		t.Fatal(err)
	}
	var comAnexo int
	for _, h := range m.hist {
		if len(h.Anexos) > 0 {
			comAnexo++
		}
	}
	if comAnexo != 3 {
		t.Fatal("nota, complementação e conclusão levam anexo", comAnexo)
	}
}

func TestAprovadorIndicado(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, true)
	ana := uuid.New()
	serv := m.servs[ped.ServicoID]
	serv.AprovadorID = &ana
	m.servs[ped.ServicoID] = serv
	chefe := ctxDe(uuid.New(), un, auth.PermProtocolosTratar, auth.PermProtocolosGerir)
	ctxAna := ctxDe(ana, uuid.New())

	if _, err := s.Concluir(chefe, ped.ID, domain.DecisaoFavoravel, "deferido"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Aprovar(chefe, ped.ID); err == nil {
		t.Fatal("o chefe não é o aprovador")
	}
	if _, err := s.Devolver(chefe, ped.ID, "x"); err == nil {
		t.Fatal("devolver também é do aprovador")
	}
	if p, err := s.Devolver(ctxAna, ped.ID, "falta parecer"); err != nil || p.Status != domain.StatusAndamento {
		t.Fatal(err)
	}
	if _, err := s.Concluir(chefe, ped.ID, domain.DecisaoFavoravel, "parecer anexado"); err != nil {
		t.Fatal(err)
	}
	if p, err := s.Aprovar(ctxAna, ped.ID); err != nil || p.Status != domain.StatusConcluido {
		t.Fatal(err)
	}
	m.falha = "buscar-pedido"
	if _, err := s.Aprovar(ctxAna, ped.ID); err == nil {
		t.Fatal("buscar")
	}
	m.falha = "buscar-serv"
	if _, err := s.Aprovar(ctxAna, ped.ID); err == nil {
		t.Fatal("serviço")
	}
	m.falha = ""
}

func TestRotinaDePrazos(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, false)
	s.sino = sinoComContato{}
	ctx := ctxGestao(un)
	teo := uuid.New()
	m.pessoas[teo] = "Téo"
	agora := s.agora()

	// A: complementação vencida, com responsável.
	if _, err := s.Aceitar(ctxDe(teo, un, auth.PermProtocolosTratar), ped.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ComplementarCom(ctx, ped.ID, Complementacao{Mensagem: "envie", PrazoDias: 1}); err != nil {
		t.Fatal(err)
	}
	// B: complementação que vence nas próximas horas.
	b, err := s.Abrir(ctx, Abertura{ServicoID: ped.ServicoID, Nome: "João Lima", CPF: "98765432100", Respostas: map[string]any{"nome": "João"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ComplementarCom(ctx, b.ID, Complementacao{Mensagem: "envie o RG", PrazoDias: 1}); err != nil {
		t.Fatal(err)
	}
	vence := agora.Add(3 * time.Hour)
	pb := m.pedidos[b.ID]
	pb.ComplementoPrazo = &vence
	m.pedidos[b.ID] = pb
	// C: prazo do protocolo vence amanhã, com responsável.
	c, err := s.Abrir(ctx, Abertura{ServicoID: ped.ServicoID, Nome: "Ana Paula", CPF: "11122233344", Respostas: map[string]any{"nome": "Ana"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	pc := m.pedidos[c.ID]
	amanha := agora.Add(20 * time.Hour)
	pc.PrazoEm, pc.ResponsavelID = &amanha, &teo
	m.pedidos[c.ID] = pc

	depois := m.pedidos[ped.ID].ComplementoPrazo.Add(time.Hour)
	pa := m.pedidos[ped.ID]
	pa.ComplementoPrazo = &[]time.Time{agora.Add(-time.Hour)}[0]
	m.pedidos[ped.ID] = pa
	_ = depois

	if err := s.VerificarPrazos(context.Background(), agora); err != nil {
		t.Fatal(err)
	}
	if m.pedidos[ped.ID].Status != domain.StatusCancelado {
		t.Fatal("a complementação vencida cancela", m.pedidos[ped.ID].Status)
	}
	ultima := m.hist[len(m.hist)-1]
	for _, h := range m.hist {
		if h.PedidoID == ped.ID && h.Acao == "cancelado" {
			ultima = h
		}
	}
	if ultima.UsuarioID != nil || ultima.Descricao != "o prazo da complementação venceu sem resposta" {
		t.Fatalf("%+v", ultima)
	}
	if len(m.avisados) != 1 || m.avisados[0] != b.ID {
		t.Fatal("o cidadão de B é lembrado", m.avisados)
	}
	if err := s.VerificarPrazos(context.Background(), agora); err != nil || len(m.avisados) != 1 {
		t.Fatal("o lembrete não repete", err, m.avisados)
	}

	// Pedido que o banco devolve sem estar vencido: a rotina pula.
	m.falha = "vencida-falsa"
	if err := s.VerificarPrazos(context.Background(), agora); err != nil {
		t.Fatal(err)
	}
	for _, falha := range []string{"vencidas", "vencendo"} {
		m.falha = falha
		if err := s.VerificarPrazos(context.Background(), agora); err == nil {
			t.Fatal(falha)
		}
	}
	m.falha = ""
	pb = m.pedidos[b.ID]
	pb.ComplementoAvisado = false
	m.pedidos[b.ID] = pb
	m.falha = "marcar"
	if err := s.VerificarPrazos(context.Background(), agora); err == nil {
		t.Fatal("marcar")
	}
	m.falha = ""
	vencida := agora.Add(-time.Minute)
	pb = m.pedidos[b.ID]
	pb.ComplementoPrazo = &vencida
	m.pedidos[b.ID] = pb
	m.falha = "atualizar-pedido"
	if err := s.VerificarPrazos(context.Background(), agora); err == nil {
		t.Fatal("gravar")
	}
	m.falha = ""

	if quemAge(auth.WithIdentity(context.Background(), auth.Identity{})) != nil {
		t.Fatal("sessão sem usuário")
	}

	antes := IntervaloPrazos
	IntervaloPrazos = time.Millisecond
	t.Cleanup(func() { IntervaloPrazos = antes })
	m.falha = "vencidas"
	ctxRotina, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelar()
	if err := s.RunPrazos(ctxRotina); err != nil {
		t.Fatal(err)
	}
	m.falha = ""
}
