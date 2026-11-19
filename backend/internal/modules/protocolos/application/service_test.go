package application

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

func ctxGestao(un uuid.UUID) context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{
		UserID: uuid.New(),
		Scopes: []auth.Scope{{
			UnidadeID: &un,
			Permissions: []string{
				string(auth.PermProtocolosGerir), string(auth.PermProtocolosTratar), string(auth.PermProtocolosAbrir),
			},
		}},
	})
}

func TestMapErrorESorteio(t *testing.T) {
	casos := []error{
		domain.ErrNaoEncontrado, domain.ErrInvalido, domain.ErrEstadoFinal, domain.ErrComplemento,
		domain.ErrNumeroRepetido, domain.ErrCPFAberto, domain.ErrResponsavel, domain.ErrCPFCadastrado, domain.ErrBloqueio, domain.ErrSemVaga,
	}
	for _, err := range casos {
		if MapError(err) == err {
			t.Fatal(err)
		}
	}
	if MapError(errors.New("banco")) == nil {
		t.Fatal("erro cru")
	}
	if sortear() < 0 || sortear() > 99999 {
		t.Fatal(sortear())
	}
	antes := inteiroAleatorio
	inteiroAleatorio = func() (int, error) { return 0, errors.New("sem aleatorio") }
	if sortear() != 0 {
		t.Fatal("fallback")
	}
	inteiroAleatorio = antes
	if NewService(nil, novoMem(), nil, nil).ComSino(nil) == nil {
		t.Fatal("sino")
	}
}

func TestFluxoDoPedido(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	s.sino = sinoFalso{}
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{
		UnidadeID: un, Titulo: "Alvará",
		Campos: []domain.Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}, {ID: "anon", Tipo: "boolean", Rotulo: "Anônimo"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	form.Titulo = "Alvará revisto"
	form.Ativo = true
	if _, err := s.SalvarFormulario(ctx, form); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListarFormularios(ctx); err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{
		UnidadeID: un, FormularioID: form.ID, Nome: "Alvará", CampoAnonimo: "anon", ImpedirCPFAberto: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	serv.Nome = "Alvará novo"
	serv.Ativo = true
	if _, err := s.SalvarServico(ctx, serv); err != nil {
		t.Fatal(err)
	}
	carta, err := s.Carta(ctx, nil)
	if err != nil || len(carta) != 1 {
		t.Fatal(carta, err)
	}
	if _, _, err := s.ServicoPublico(ctx, serv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CadastrarResponsavel(ctx, "Ana Lima", "12345678900", un); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CadastrarResponsavel(ctx, "Ana Lima", "123.456.789-00", un); !errors.Is(err, domain.ErrCPFCadastrado) {
		t.Fatal(err)
	}
	tecnicos, err := s.ListarResponsaveis(ctx)
	if err != nil || len(tecnicos) != 1 || tecnicos[0].Nome != "Ana Lima" || tecnicos[0].CPF != "***.456.789-00" {
		t.Fatal(tecnicos, err)
	}
	if _, err := s.ListarResponsaveis(context.Background()); err == nil {
		t.Fatal("lista sem sessão")
	}
	ped, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "123.456.789-00", Respostas: map[string]any{"nome": "Maria", "anon": true}}, false)
	if err != nil || ped.Numero == "" || ped.Status != domain.StatusNovo {
		t.Fatalf("%+v %v", ped, err)
	}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{"nome": "Maria"}}, false); !errors.Is(err, domain.ErrCPFAberto) {
		t.Fatal(err)
	}
	balcao, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Joao Lima", CPF: "98765432100", Respostas: map[string]any{"nome": "Joao"}}, true)
	if err != nil || balcao.UsuarioID == nil {
		t.Fatal(balcao, err)
	}
	consulta, err := s.Consultar(ctx, ped.Numero)
	if err != nil || consulta.Nome != "Denúncia anônima" {
		t.Fatal(consulta, err)
	}
	lista, err := s.ListarPedidos(ctx)
	if err != nil || len(lista) < 2 || lista[0].SituacaoPrazo == "" {
		t.Fatal(len(lista), lista, err)
	}
	ficha, err := s.BuscarPedido(ctx, ped.ID)
	if err != nil || len(ficha.Historico) == 0 {
		t.Fatal(ficha, err)
	}
	if _, err := s.Complementar(ctx, ped.ID, "envie o comprovante"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Avancar(ctx, ped.ID); !errors.Is(err, domain.ErrComplemento) {
		t.Fatal(err)
	}
	if _, err := s.Responder(ctx, ped.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Concluir(ctx, ped.ID, domain.DecisaoFavoravel, "deferido"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancelar(ctx, balcao.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RegistrarFalta(ctx, "12345678900", un); err != nil {
		t.Fatal(err)
	}
	outro := uuid.New()
	if _, err := s.SalvarFormulario(ctxGestao(outro), domain.Formulario{UnidadeID: un, Titulo: "Fora"}); !errors.Is(err, apperrors.Forbidden("fora da unidade")) && err == nil {
		t.Fatal(err)
	}
	if _, err := s.ListarFormularios(context.Background()); err == nil {
		t.Fatal("sem sessão")
	}
}

func TestRespostaPublica(t *testing.T) {
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
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Alvará"})
	if err != nil {
		t.Fatal(err)
	}
	ped, err := s.Abrir(ctx, Abertura{
		ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900",
		Email: "maria@exemplo.gov.br", Telefone: "66999991234",
		Respostas: map[string]any{"nome": "Maria"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", nil); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	if _, err := s.Complementar(ctx, ped.ID, "envie o comprovante"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, ped.CPF, "a.pdf", "application/pdf"); err == nil {
		t.Fatal("sem armazenamento")
	}
	var dep *apperrors.Error
	if _, err := s.PedirArquivoComplemento(ctx, ped.Numero, ped.CPF, "a.pdf", "application/pdf"); !errors.As(err, &dep) || dep.Code != apperrors.CodeDependencyUnavailable {
		t.Fatal(err)
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, "00000000000", "segue", nil); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := s.ResponderPublico(ctx, ped.Numero, ped.CPF, "segue", []string{"protocolos/complemento/" + ped.ID.String() + "/a.pdf"}); !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	consulta, err := s.ResponderPublico(ctx, ped.Numero, "123.456.789-00", "segue o documento", nil)
	if err != nil || consulta.Status != domain.StatusAndamento || consulta.StatusRotulo != "Em andamento" {
		t.Fatal(consulta, err)
	}
	ficha, err := s.BuscarPedido(ctx, ped.ID)
	if err != nil || len(ficha.Historico) == 0 {
		t.Fatal(ficha, err)
	}
	ultimo := ficha.Historico[len(ficha.Historico)-1]
	publico := consulta.Historico[len(consulta.Historico)-1]
	if ultimo.Descricao != "segue o documento" || publico.Descricao != "segue o documento" || publico.Anexos != nil {
		t.Fatal(ultimo, publico)
	}
}

func TestAgendamentoEFalhas(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Hora", Campos: []domain.Campo{{ID: "nome", Tipo: "text", Rotulo: "Nome", Obrigatorio: true}}})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{
		UnidadeID: un, FormularioID: form.ID, Nome: "Visita", SoResponsavelTecnico: true, ExigeAgendamento: true,
		Agenda: domain.Agenda{Dias: []int{1, 2, 3, 4, 5}, Inicio: "08:00", Fim: "12:00", DuracaoMin: 30, VagasDia: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	abertura := Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{"nome": "Maria"}}
	if _, err := s.Abrir(ctx, abertura, false); !errors.Is(err, domain.ErrResponsavel) {
		t.Fatal(err)
	}
	m.resp["12345678900"] = true
	ped, err := s.Abrir(ctx, abertura, false)
	if err != nil || len(m.agenda) != 1 || m.agenda[0].Status != "agendado" {
		t.Fatalf("%+v %v %v", ped, m.agenda, err)
	}
	m.resp["11122233344"] = true
	marcado, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Pedro Alves", CPF: "11122233344", Data: "2026-11-16", Hora: "09:00", Respostas: map[string]any{"nome": "Pedro"}}, false)
	if err != nil || marcado.Numero == "" {
		t.Fatal(marcado, err)
	}
	m.resp["11122233344"] = true
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Pedro Alves", CPF: "11122233344", Data: "2026-11-16", Hora: "09:00", Respostas: map[string]any{"nome": "Pedro"}}, false); err != nil && !errors.Is(err, domain.ErrSemVaga) && !errors.Is(err, domain.ErrInvalido) {
		t.Fatal(err)
	}
	m.faltas, m.ultima = 2, s.agora()
	if _, err := s.Abrir(ctx, abertura, false); !errors.Is(err, domain.ErrBloqueio) {
		t.Fatal(err)
	}
	serv.Ativo = false
	m.servs[serv.ID] = serv
	if _, _, err := s.ServicoPublico(ctx, serv.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := s.Abrir(ctx, abertura, false); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	wf := m.servs[serv.ID]
	wf.Ativo = true
	wf.ExigeAgendamento = false
	wf.SoResponsavelTecnico = false
	wf.Workflow = domain.Workflow{Ligado: true, Etapas: []domain.Etapa{{ID: "a", Ordem: 1, Status: domain.StatusAndamento}, {ID: "b", Ordem: 2, Status: domain.StatusAnalise}}}
	m.servs[serv.ID] = wf
	anda, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Rita Dias", CPF: "55566677788", Respostas: map[string]any{"nome": "Rita"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Avancar(ctx, anda.ID); err != nil {
		t.Fatal(err)
	}
	s.run = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, &stubTx{falha: 1})
	}
	if _, err := s.Cancelar(ctx, anda.ID); err == nil {
		t.Fatal("evento")
	}
	if _, err := s.Consultar(ctx, "nao-existe"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := s.BuscarPedido(context.Background(), anda.ID); err == nil {
		t.Fatal("sem sessão na ficha")
	}
	s.sino = sinoFalso{erro: errors.New("canal")}
	s.run = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, &stubTx{})
	}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Ivo Neto", CPF: "22233344455", Respostas: map[string]any{"nome": "Ivo"}}, true); err != nil {
		t.Fatal(err)
	}
}

func TestRecusasDeCadastro(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "A"}); err == nil {
		t.Fatal("título")
	}
	m.nome = ""
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok"}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	m.nome = "Meio Ambiente"
	if _, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, Nome: "Ok"}); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := s.CadastrarResponsavel(ctx, "A", "1", un); err == nil {
		t.Fatal("responsável")
	}
	if err := s.RegistrarFalta(ctx, "123", un); err == nil {
		t.Fatal("cpf")
	}
	if _, err := s.SalvarFormulario(context.Background(), domain.Formulario{UnidadeID: un, Titulo: "Ok"}); err == nil {
		t.Fatal("sessão")
	}
}

func TestLacunas(t *testing.T) {
	defer func() { _ = recover() }()
	_ = NewService(nil, novoMem(), nil, nil).run(context.Background(), func(context.Context, pgx.Tx) error { return nil })
}

func TestListasEErros(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	if s.ComSino(notificacoes.NewService(nil, nil, slog.Default())) == nil {
		t.Fatal("sino")
	}
	s.sino = sinoFalso{}
	un := uuid.New()
	global := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		Permissions: []string{string(auth.PermProtocolosGerir)},
	}}})
	m.falha = "nulo"
	if _, err := s.ListarFormularios(global); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListarServicosGestao(global); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Carta(global, &un); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListarPedidos(global); err != nil {
		t.Fatal(err)
	}
	soGerir := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &un, Permissions: []string{string(auth.PermProtocolosGerir)},
	}}})
	m.falha = ""
	if _, err := s.ListarFormularios(soGerir); err != nil {
		t.Fatal(err)
	}
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	outra := uuid.New()
	m.forms[outra] = domain.Formulario{ID: outra, UnidadeID: outra, Titulo: "Outra", Ativo: true}
	if _, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: outra, Nome: "X"}); err == nil {
		t.Fatal("formulário de outra unidade")
	}
	m.falha = "inserir-form"
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Novo"}); err == nil {
		t.Fatal("insert")
	}
	m.falha = "buscar-form"
	if _, _, err := s.ServicoPublico(ctx, uuid.New()); err == nil {
		t.Fatal("formulario")
	}
	m.falha = ""
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	m.falha = "atualizar-serv"
	serv.Nome = "Ok 2"
	if _, err := s.SalvarServico(ctx, serv); err == nil {
		t.Fatal("update")
	}
	m.servs[serv.ID] = domain.Servico{ID: serv.ID, UnidadeID: un, FormularioID: form.ID, Nome: "Ok", Ativo: true, ImpedirCPFAberto: true}
	m.falha = "cpf"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		t.Fatal("cpf")
	}
	m.falha = "numero"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Respostas: map[string]any{}}, false); err == nil {
		t.Fatal("número")
	}
	m.falha = "buscar-numero"
	if _, err := s.Consultar(ctx, "x"); err == nil {
		t.Fatal("consulta")
	}
	m.falha = "buscar-pedido"
	if _, err := s.Complementar(ctx, uuid.New(), "texto"); err == nil {
		t.Fatal("pedido")
	}
	if _, err := s.BuscarPedido(ctxGestao(outra), form.ID); err == nil {
		t.Fatal("fora da ficha")
	}
	m.falha = ""
	s.sorteio = func() int { return -1 }
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		t.Fatal("sufixo")
	}
	s.sorteio = func() int { return 1 }
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "M", CPF: "12345678900"}, false); err == nil {
		t.Fatal("nome")
	}
	for _, falha := range []string{"buscar-serv", "buscar-form", "resp-ativo", "unidade", "inserir-pedido", "hist", "faltas", "contar"} {
		m.falha = falha
		_, _ = s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-15", Hora: "09:00"}, true)
	}
	m.falha = "listar-hist"
	_, _ = s.Consultar(ctx, "inexistente")
	m.falha = "buscar-agenda"
	if ped, err := s.Abrir(ctxGestao(un), Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		m.falha = "buscar-agenda"
		_, _ = s.Consultar(ctx, ped.Numero)
		_, _ = s.BuscarPedido(ctx, ped.ID)
	}
	m.falha = "somar"
	_ = s.RegistrarFalta(ctx, "12345678900", un)
	m.falha = "resp"
	_, _ = s.CadastrarResponsavel(ctx, "Ana Lima", "12345678900", un)
	m.falha = "listar-gestao"
	_, _ = s.ListarServicosGestao(ctx)
	m.falha = "atualizar-form"
	form.Titulo = "Ok de novo"
	_, _ = s.SalvarFormulario(ctx, form)
	m.falha = "inserir-serv"
	_, _ = s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Outro"})
}

func TestRamosRestantes(t *testing.T) {
	m := novoMem()
	s := servicoTeste(m)
	s.sino = sinoFalso{erro: errors.New("sino")}
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	s.sorteio = func() int { return 7 }
	m.numeros["SEMA202600007"] = true
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); !errors.Is(err, domain.ErrNumeroRepetido) {
		t.Fatal(err)
	}
	delete(m.numeros, "SEMA202600007")
	inativo := serv
	inativo.Ativo = false
	m.servs[serv.ID] = inativo
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, _, err := s.ServicoPublico(ctx, serv.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	agenda := domain.Agenda{Dias: []int{2}, Inicio: "08:00", Fim: "12:00", DuracaoMin: 30, VagasDia: 1}
	marcado := serv
	marcado.Ativo = true
	marcado.ExigeAgendamento = true
	marcado.Agenda = agenda
	m.servs[serv.ID] = marcado
	m.falha = "faltas"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		t.Fatal("faltas")
	}
	m.falha = "contar"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, false); err == nil {
		t.Fatal("contar")
	}
	m.falha = ""
	m.agenda = []domain.Agendamento{{}}
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, false); !errors.Is(err, domain.ErrSemVaga) {
		t.Fatal(err)
	}
	m.agenda = nil
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-15", Hora: "08:00"}, false); err == nil {
		t.Fatal("dia")
	}
	m.faltas, m.ultima = 2, time.Date(2026, 11, 16, 0, 0, 0, 0, domain.Cuiaba)
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); !errors.Is(err, domain.ErrBloqueio) {
		t.Fatal(err)
	}
	m.faltas = 0
	m.falha = "agenda"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, false); err == nil {
		t.Fatal("agenda")
	}
	m.falha = ""
	s.sorteio = func() int { return 8 }
	ped, err := s.Abrir(ctxGestao(un), Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m.falha = "listar-hist"
	if _, err := s.Consultar(ctx, ped.Numero); err == nil {
		t.Fatal("histórico")
	}
	if _, err := s.BuscarPedido(ctx, ped.ID); err == nil {
		t.Fatal("ficha")
	}
	m.falha = "unidade"
	if _, err := s.Cancelar(ctx, ped.ID); err == nil {
		t.Fatal("unidade")
	}
	m.falha = "atualizar-pedido"
	if _, err := s.Cancelar(ctx, ped.ID); err == nil {
		t.Fatal("atualizar")
	}
	m.falha = "buscar-serv"
	if _, err := s.Avancar(ctx, ped.ID); err == nil {
		t.Fatal("etapa")
	}
	m.falha = ""
	m.nome = ""
	if _, err := s.Cancelar(ctx, ped.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListarFormularios(context.Background()); err == nil {
		t.Fatal("sessão")
	}
	if _, err := s.BuscarPedido(context.Background(), ped.ID); err == nil {
		t.Fatal("sessão")
	}
	m.falha = "listar-pedidos"
	if _, err := s.ListarPedidos(ctx); err == nil {
		t.Fatal("lista")
	}
	m.falha = "listar-form"
	if _, err := s.ListarFormularios(ctx); err == nil {
		t.Fatal("forms")
	}
	m.falha = ""
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: " "}); err == nil {
		t.Fatal("título")
	}
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{ID: uuid.New(), UnidadeID: un, Titulo: "Ok"}); err == nil {
		t.Fatal("ausente")
	}
	m.falha = "unidade"
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok"}); err == nil {
		t.Fatal("unidade")
	}
	m.falha = ""
	m.nome = "Meio Ambiente"
	if _, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: uuid.New(), Nome: "Z"}); err == nil {
		t.Fatal("formulário")
	}
	m.falha = ""
	if _, err := s.CadastrarResponsavel(ctx, "A", "12345678900", un); err == nil {
		t.Fatal("nome")
	}
	if err := s.RegistrarFalta(ctx, "1", un); err == nil {
		t.Fatal("cpf")
	}
	if err := s.RegistrarFalta(context.Background(), "12345678900", un); err == nil {
		t.Fatal("sessão")
	}
	s.sorteio = func() int { return 9 }
	m.agenda = nil
	outro, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:30"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m.falha = "hist"
	if _, err := s.Cancelar(ctx, outro.ID); err == nil {
		t.Fatal("hist")
	}
}

func TestFechaRamos(t *testing.T) {
	cru := NewService(nil, novoMem(), outbox.NewWriter("nova.protocolos"), nil)
	if cru.agora().IsZero() || cru.sorteio() < 0 {
		t.Fatal("relogio")
	}
	antigo := inteiroAleatorio
	inteiroAleatorio = func() (int, error) { return 0, errors.New("rand") }
	if cru.sorteio() != 0 {
		t.Fatal("sorteio")
	}
	inteiroAleatorio = antigo

	m := novoMem()
	s := servicoTeste(m)
	un := uuid.New()
	ctx := ctxGestao(un)
	form, err := s.SalvarFormulario(ctx, domain.Formulario{UnidadeID: un, Titulo: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	serv, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Ok"})
	if err != nil {
		t.Fatal(err)
	}
	semAbrir := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &un, Permissions: []string{string(auth.PermProtocolosTratar)},
	}}})
	if _, err := s.Abrir(semAbrir, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, true); err == nil {
		t.Fatal("balcão")
	}
	tecnico := serv
	tecnico.SoResponsavelTecnico = true
	m.servs[serv.ID] = tecnico
	m.falha = "resp-ativo"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		t.Fatal("técnico")
	}
	m.servs[serv.ID] = serv
	for _, falha := range []string{"unidade", "inserir-pedido", "hist"} {
		m.falha = falha
		if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
			t.Fatal(falha)
		}
	}
	m.falha = ""
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{ID: uuid.New(), UnidadeID: un, Titulo: "Ok"}); err == nil {
		t.Fatal("ausente")
	}
	guardado := m.forms[form.ID]
	guardado.UnidadeID = uuid.New()
	m.forms[form.ID] = guardado
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{ID: form.ID, UnidadeID: un, Titulo: "Ok"}); err == nil {
		t.Fatal("outra unidade")
	}
	guardado.UnidadeID = un
	m.forms[form.ID] = guardado
	if _, err := s.SalvarFormulario(ctx, domain.Formulario{ID: form.ID, UnidadeID: un, Titulo: " "}); err == nil {
		t.Fatal("título")
	}
	if _, err := s.SalvarServico(context.Background(), domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Z"}); err == nil {
		t.Fatal("sessão")
	}
	m.falha = "unidade"
	if _, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: "Z"}); err == nil {
		t.Fatal("unidade")
	}
	m.falha = ""
	if _, err := s.SalvarServico(ctx, domain.Servico{UnidadeID: un, FormularioID: form.ID, Nome: " "}); err == nil {
		t.Fatal("nome")
	}
	if _, err := s.SalvarServico(ctx, domain.Servico{ID: uuid.New(), UnidadeID: un, FormularioID: form.ID, Nome: "Z"}); err == nil {
		t.Fatal("serviço")
	}
	atual := m.servs[serv.ID]
	atual.UnidadeID = uuid.New()
	m.servs[serv.ID] = atual
	if _, err := s.SalvarServico(ctx, domain.Servico{ID: serv.ID, UnidadeID: un, FormularioID: form.ID, Nome: "Z"}); err == nil {
		t.Fatal("dono")
	}
	atual.UnidadeID = un
	m.servs[serv.ID] = atual
	if _, err := s.SalvarServico(ctx, domain.Servico{ID: serv.ID, UnidadeID: un, FormularioID: form.ID, Nome: " "}); err == nil {
		t.Fatal("nome")
	}
	if _, err := s.ListarServicosGestao(context.Background()); err == nil {
		t.Fatal("sessão")
	}
	if _, err := s.ListarPedidos(context.Background()); err == nil {
		t.Fatal("sessão")
	}
	m.falha = "buscar-form"
	if _, _, err := s.ServicoPublico(ctx, serv.ID); err == nil {
		t.Fatal("formulário")
	}
	m.falha = ""
	if _, err := s.CadastrarResponsavel(context.Background(), "Ana Lima", "12345678900", un); err == nil {
		t.Fatal("sessão")
	}
	marcado := serv
	marcado.ExigeAgendamento = true
	marcado.Agenda = domain.Agenda{Dias: []int{0}, Inicio: "08:00", Fim: "12:00", DuracaoMin: 30, VagasDia: 1}
	m.servs[serv.ID] = marcado
	m.falha = "contar"
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900"}, false); err == nil {
		t.Fatal("busca")
	}
	m.falha = ""
	if _, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-15", Hora: "08:00"}, false); err == nil {
		t.Fatal("passado")
	}
	marcado.Agenda.Dias = []int{2}
	m.servs[serv.ID] = marcado
	s.sorteio = func() int { return 3 }
	ped, err := s.Abrir(ctx, Abertura{ServicoID: serv.ID, Nome: "Maria Souza", CPF: "12345678900", Data: "2026-11-17", Hora: "08:00"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if consulta, err := s.Consultar(ctx, ped.Numero); err != nil || consulta.Agendamento == nil {
		t.Fatal(err)
	}
	if ficha, err := s.BuscarPedido(ctx, ped.ID); err != nil || ficha.Agendamento == nil {
		t.Fatal(err)
	}
	m.falha = "buscar-serv"
	if _, err := s.Consultar(ctx, ped.Numero); err == nil {
		t.Fatal("serviço")
	}
	m.falha = "buscar-agenda"
	if _, err := s.Consultar(ctx, ped.Numero); err == nil {
		t.Fatal("agenda")
	}
	if _, err := s.BuscarPedido(ctx, ped.ID); err == nil {
		t.Fatal("agenda")
	}
	m.falha = "nulo"
	if ficha, err := s.BuscarPedido(ctx, ped.ID); err != nil || ficha.Historico == nil {
		t.Fatal(err)
	}
	m.falha = ""
	alheio := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &un, Permissions: []string{"protocolos:nenhuma"},
	}}})
	if _, err := s.BuscarPedido(alheio, ped.ID); err == nil {
		t.Fatal("visão")
	}
	if _, err := s.Cancelar(alheio, ped.ID); err == nil {
		t.Fatal("tratar")
	}
}
