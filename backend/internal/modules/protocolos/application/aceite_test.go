package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func ctxDe(usuario, un uuid.UUID, perms ...auth.Permission) context.Context {
	lista := make([]string, 0, len(perms))
	for _, p := range perms {
		lista = append(lista, string(p))
	}
	return auth.WithIdentity(context.Background(), auth.Identity{
		UserID: usuario,
		Scopes: []auth.Scope{{UnidadeID: &un, Permissions: lista}},
	})
}

// cenarioAceite monta um serviço com aceite e aprovação e abre um pedido.
func cenarioAceite(t *testing.T, exigeAceite, exigeAprovacao bool) (*mem, *Service, uuid.UUID, domain.Pedido) {
	t.Helper()
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{
		UnidadeID: un, Titulo: "Alvará",
		Campos: []domain.Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{
		UnidadeID: un, FormularioID: form.ID, Nome: "Alvará", ExigeAceite: exigeAceite, ExigeAprovacao: exigeAprovacao,
	})
	if err != nil {
		t.Fatal(err)
	}
	ped, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{"nome": "Maria"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	return m, s, un, ped
}

func ultimaDescricao(m *mem, acao string) string {
	for i := len(m.hist) - 1; i >= 0; i-- {
		if m.hist[i].Acao == acao {
			return m.hist[i].Descricao
		}
	}
	return ""
}

func TestAceiteTravaETransferencia(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, true, false)
	if !ped.ExigeAceite {
		t.Fatal("o pedido herda o aceite do serviço")
	}
	teo, ana, chefe := uuid.New(), uuid.New(), uuid.New()
	m.pessoas[teo], m.pessoas[ana], m.pessoas[chefe] = "Téo", "Ana", "Chefe"
	ctxTeo := ctxDe(teo, un, auth.PermProtocolosTratar)
	ctxChefe := ctxDe(chefe, un, auth.PermProtocolosTratar, auth.PermProtocolosGerir)

	// Sem dono, o serviço com aceite não deixa tratar.
	if _, err := s.Complementar(ctxTeo, ped.ID, "envie"); !errors.Is(err, domain.ErrSemResponsavel) {
		t.Fatal(err)
	}
	if _, err := s.Avancar(ctxTeo, ped.ID); !errors.Is(err, domain.ErrSemResponsavel) {
		t.Fatal(err)
	}
	if _, err := s.Anotar(ctxTeo, ped.ID, "nota"); !errors.Is(err, domain.ErrSemResponsavel) {
		t.Fatal(err)
	}
	if _, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoFavoravel, "ok"); !errors.Is(err, domain.ErrSemResponsavel) {
		t.Fatal(err)
	}
	m.falha = "buscar-serv"
	if _, err := s.exigirDonoDe(ctxTeo, ped.ID); err == nil {
		t.Fatal("serviço")
	}
	if _, err := s.Anotar(ctxTeo, ped.ID, "nota"); err == nil {
		t.Fatal("serviço na nota")
	}
	m.falha = ""

	// Quem só trata não indica outra pessoa; a pessoa tem de existir.
	if _, err := s.Aceitar(ctxTeo, ped.ID, &ana); err == nil {
		t.Fatal("indicar sem gerir")
	}
	desconhecido := uuid.New()
	if _, err := s.Aceitar(ctxChefe, ped.ID, &desconhecido); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	aceito, err := s.Aceitar(ctxTeo, ped.ID, nil)
	if err != nil || aceito.ResponsavelNome != "Téo" || aceito.Status != domain.StatusAndamento || ultimaDescricao(m, domain.AcaoAceite) != "assumido por Téo" {
		t.Fatalf("%+v %v", aceito, err)
	}
	if _, err := s.Aceitar(ctxTeo, ped.ID, &uuid.Nil); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("já tem dono", err)
	}
	if _, err := s.Anotar(ctxTeo, ped.ID, "com dono anota"); err != nil {
		t.Fatal(err)
	}
	m.falha = "buscar-serv"
	if _, err := s.Avancar(ctxTeo, ped.ID); err == nil {
		t.Fatal("serviço no avanço")
	}
	m.falha = ""

	// Transferir: o responsável ou quem gere.
	if _, err := s.Transferir(ctxDe(uuid.New(), un, auth.PermProtocolosTratar), ped.ID, ana); err == nil {
		t.Fatal("transferir sem ser o responsável")
	}
	if _, err := s.Transferir(ctxTeo, ped.ID, desconhecido); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	transferido, err := s.Transferir(ctxTeo, ped.ID, ana)
	if err != nil || transferido.ResponsavelNome != "Ana" || ultimaDescricao(m, domain.AcaoTransferencia) != "transferido para Ana" {
		t.Fatal(err)
	}
	s.sino = sinoFalso{erro: errors.New("sino")}
	if _, err := s.Transferir(ctxChefe, ped.ID, chefe); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transferir(ctxChefe, ped.ID, teo); err != nil {
		t.Fatal(err)
	}

	// O chefe indica alguém num pedido novo; a pessoa indicada é avisada.
	s.sino = sinoFalso{}
	outro, err := s.Abrir(ctxChefe, Abertura{ServicoID: ped.ServicoID, Nome: "João Lima", CPF: "98765432100", Respostas: map[string]any{"nome": "João"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	indicado, err := s.Aceitar(ctxChefe, outro.ID, &ana)
	if err != nil || *indicado.ResponsavelID != ana || ultimaDescricao(m, domain.AcaoAceite) != "atribuído a Ana" {
		t.Fatal(err)
	}

	// Quem pode assumir.
	pessoas, err := s.Pessoas(ctxTeo, ped.ID)
	if err != nil || len(pessoas) != 3 {
		t.Fatal(pessoas, err)
	}
	m.falha = "nulo"
	if lista, err := s.Pessoas(ctxTeo, ped.ID); err != nil || lista == nil {
		t.Fatal(err)
	}
	m.falha = "buscar-pedido"
	if _, err := s.Pessoas(ctxTeo, ped.ID); err == nil {
		t.Fatal("buscar")
	}
	m.falha = ""
	if _, err := s.Pessoas(ctxDe(teo, uuid.New(), auth.PermProtocolosTratar), ped.ID); err == nil {
		t.Fatal("fora da unidade")
	}
}

// exigirDonoDe expõe a trava para o teste do serviço que falha.
func (s *Service) exigirDonoDe(ctx context.Context, id uuid.UUID) (domain.Pedido, error) {
	p, err := s.repo.BuscarPedido(ctx, s.db, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	return p, s.exigirDono(ctx, p)
}

func TestAprovacaoDaConclusao(t *testing.T) {
	m, s, un, ped := cenarioAceite(t, false, true)
	chefe, teo := uuid.New(), uuid.New()
	ctxChefe := ctxDe(chefe, un, auth.PermProtocolosTratar, auth.PermProtocolosGerir)
	ctxTeo := ctxDe(teo, un, auth.PermProtocolosTratar)

	aguardando, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoFavoravel, "deferido")
	if err != nil || aguardando.Status != domain.StatusAguardando {
		t.Fatal(aguardando.Status, err)
	}
	if _, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoDesfavoravel, "outra"); !errors.Is(err, domain.ErrAprovacao) {
		t.Fatal(err)
	}
	if _, err := s.Aprovar(ctxTeo, ped.ID); err == nil {
		t.Fatal("aprovar sem gerir")
	}
	if _, err := s.Devolver(ctxChefe, ped.ID, "  "); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	devolvido, err := s.Devolver(ctxChefe, ped.ID, "falta o parecer")
	if err != nil || devolvido.Status != domain.StatusAndamento || devolvido.Decisao != "" {
		t.Fatal(err)
	}
	if _, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoFavoravel, "parecer anexado"); err != nil {
		t.Fatal(err)
	}
	aprovado, err := s.Aprovar(ctxChefe, ped.ID)
	if err != nil || aprovado.Status != domain.StatusConcluido || aprovado.Decisao != domain.DecisaoFavoravel {
		t.Fatal(err)
	}
	consulta, err := s.Consultar(ctxChefe, ped.Numero)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range consulta.Historico {
		if strings.Contains(h.Descricao, "falta o parecer") {
			t.Fatal("a devolução é interna", h)
		}
	}

	m.falha = "buscar-pedido"
	if _, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoFavoravel, "x"); err == nil {
		t.Fatal("buscar")
	}
	m.falha = "buscar-serv"
	if _, err := s.Concluir(ctxTeo, ped.ID, domain.DecisaoFavoravel, "x"); err == nil {
		t.Fatal("serviço")
	}
	m.falha = ""
}

func TestEncaminhamentoEntreUnidades(t *testing.T) {
	m, s, origem, ped := cenarioAceite(t, false, false)
	destino := uuid.New()
	teo, dani := uuid.New(), uuid.New()
	m.pessoas[teo] = "Téo"
	ctxTeo := ctxDe(teo, origem, auth.PermProtocolosTratar)
	ctxDani := ctxDe(dani, destino, auth.PermProtocolosTratar)

	if _, err := s.Aceitar(ctxTeo, ped.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Encaminhar(ctxTeo, ped.ID, origem, "mesma unidade", 0); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	m.falha = "unidade"
	if _, err := s.Encaminhar(ctxTeo, ped.ID, destino, "é da saúde", 3); err == nil {
		t.Fatal("unidade")
	}
	m.falha = ""
	enc, err := s.Encaminhar(ctxTeo, ped.ID, destino, "  é da saúde ", 3)
	if err != nil || enc.Encaminhamento == nil || enc.Encaminhamento.UnidadeNome != "Meio Ambiente" ||
		ultimaDescricao(m, domain.AcaoEncaminhamento) != "encaminhado a Meio Ambiente: é da saúde" {
		t.Fatalf("%+v %v", enc.Encaminhamento, err)
	}
	// Com o encaminhamento pendente, nada muda.
	if _, err := s.Complementar(ctxTeo, ped.ID, "envie"); !errors.Is(err, domain.ErrEncaminhamento) {
		t.Fatal(err)
	}
	if _, err := s.Encaminhar(ctxTeo, ped.ID, destino, "de novo", 0); !errors.Is(err, domain.ErrEncaminhamento) {
		t.Fatal(err)
	}
	// O destino enxerga a ficha e decide; a origem não decide.
	if _, err := s.BuscarPedido(ctxDani, ped.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BuscarPedido(ctxDe(uuid.New(), uuid.New(), auth.PermProtocolosTratar), ped.ID); err == nil {
		t.Fatal("terceira unidade")
	}
	if _, err := s.DecidirEncaminhamento(ctxTeo, ped.ID, true, ""); err == nil {
		t.Fatal("a origem não aceita")
	}
	if _, err := s.DecidirEncaminhamento(ctxDani, ped.ID, false, " "); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	recusado, err := s.DecidirEncaminhamento(ctxDani, ped.ID, false, "não é nosso")
	if err != nil || recusado.Encaminhamento != nil || recusado.UnidadeID != origem || recusado.ResponsavelID == nil {
		t.Fatal(err)
	}
	if _, err := s.DecidirEncaminhamento(ctxDani, ped.ID, true, ""); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal("sem encaminhamento", err)
	}
	if _, err := s.Encaminhar(ctxTeo, ped.ID, destino, "agora sim", 0); err != nil {
		t.Fatal(err)
	}
	recebido, err := s.DecidirEncaminhamento(ctxDani, ped.ID, true, "")
	if err != nil || recebido.UnidadeID != destino || recebido.ResponsavelID != nil || recebido.Encaminhamento != nil {
		t.Fatal(err)
	}

	m.falha = "buscar-pedido"
	if _, err := s.DecidirEncaminhamento(ctxDani, ped.ID, true, ""); err == nil {
		t.Fatal("buscar")
	}
	m.falha = ""
	// Pedido encerrado com encaminhamento esquecido: o domínio recusa as duas respostas.
	encerrado := m.pedidos[ped.ID]
	encerrado.Status = domain.StatusConcluido
	encerrado.Encaminhamento = &domain.Encaminhamento{UnidadeID: destino}
	m.pedidos[ped.ID] = encerrado
	if _, err := s.DecidirEncaminhamento(ctxDani, ped.ID, true, ""); !errors.Is(err, domain.ErrEstadoFinal) {
		t.Fatal(err)
	}
	if _, err := s.DecidirEncaminhamento(ctxDani, ped.ID, false, "x"); !errors.Is(err, domain.ErrEstadoFinal) {
		t.Fatal(err)
	}
}

func TestEdicaoAmpliada(t *testing.T) {
	_, s, un, ped := cenarioAceite(t, false, false)
	ctx := ctxGestao(un)
	ajustado, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "cidadão idoso", Prioridade: "urgente", Prazo: "2026-12-01"})
	if err != nil || ajustado.Prioridade != "urgente" || ajustado.PrazoEm == nil || ajustado.Respostas["nome"] != "Maria" {
		t.Fatalf("%+v %v", ajustado, err)
	}
	if _, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "x", Respostas: map[string]any{}}); err == nil {
		t.Fatal("resposta obrigatória")
	}
	if _, err := s.Editar(ctx, ped.ID, Edicao{Justificativa: "x", Prioridade: "maxima"}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
}

func TestMapErrorDoAceite(t *testing.T) {
	for _, err := range []error{domain.ErrSemResponsavel, domain.ErrEncaminhamento, domain.ErrAprovacao} {
		if MapError(err) == err {
			t.Fatal(err)
		}
	}
}
