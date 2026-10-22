package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func oferta(tipo string) Oferta {
	u, p := uuid.New(), uuid.New()
	return Oferta{ID: uuid.New(), Nome: "E-mail", Tipo: tipo, UnidadeID: &u, ProdutoID: &p, Ativa: true}
}

func TestNovaSolicitacaoHerdaDaOferta(t *testing.T) {
	quem := uuid.New()
	o := oferta(TipoIncidente)
	s, err := NovaSolicitacao(o, "  Não entro no e-mail ", " desde cedo ", quem)
	if err != nil {
		t.Fatal(err)
	}
	if s.Tipo != TipoIncidente || s.Estado != EstadoAberto || s.Titulo != "Não entro no e-mail" || s.Descricao != "desde cedo" {
		t.Fatalf("campos: %+v", s)
	}
	if s.OfertaID != o.ID || s.UnidadeID != o.UnidadeID || s.ProdutoID != o.ProdutoID || s.SolicitanteID != quem || s.OfertaNome != "E-mail" {
		t.Fatalf("herança da oferta: %+v", s)
	}

	inativa := oferta(TipoRequisicao)
	inativa.Ativa = false
	casos := []struct {
		nome   string
		o      Oferta
		titulo string
		quem   uuid.UUID
		want   error
	}{
		{"oferta inativa", inativa, "x", quem, ErrOfertaInativa},
		{"tipo desconhecido", oferta("problema"), "x", quem, ErrTipoInvalido},
		{"sem título", oferta(TipoRequisicao), " ", quem, ErrSolicitacaoInvalida},
		{"título longo", oferta(TipoRequisicao), strings.Repeat("a", NomeMax+1), quem, ErrSolicitacaoInvalida},
		{"sem solicitante", oferta(TipoRequisicao), "x", uuid.Nil, ErrSolicitanteInvalido},
	}
	for _, c := range casos {
		if _, err := NovaSolicitacao(c.o, c.titulo, "", c.quem); err != c.want {
			t.Errorf("%s: %v", c.nome, err)
		}
	}
}

func aberta(t *testing.T, tipo string) Solicitacao {
	t.Helper()
	s, err := NovaSolicitacao(oferta(tipo), "Pedido", "", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func passar(t *testing.T, s Solicitacao, para string, papeis ...string) Solicitacao {
	t.Helper()
	next, mudou, err := s.Transicionar(para, papeis...)
	if err != nil || !mudou {
		t.Fatalf("%s -> %s (%v): %v %v", s.Estado, para, papeis, mudou, err)
	}
	return next
}

func TestCicloDoIncidente(t *testing.T) {
	s := aberta(t, TipoIncidente)
	if _, _, err := s.Transicionar(EstadoEmAtendimento, PapelTecnico); err != ErrSemResponsavel {
		t.Fatalf("atender sem responsável: %v", err)
	}
	s, _, _ = s.Atribuir(uuid.New())
	if s.Estado != EstadoEmAtendimento {
		t.Fatalf("assumir põe em atendimento: %s", s.Estado)
	}
	s = passar(t, s, EstadoAguardandoTerceiro, PapelTecnico)
	s = passar(t, s, EstadoEmAtendimento, PapelTecnico)
	if _, _, err := s.Transicionar(EstadoAtendido, PapelTecnico); err != ErrTransicaoInvalida {
		t.Fatalf("incidente não é atendido: %v", err)
	}
	s = passar(t, s, EstadoResolvido, PapelTecnico)
	if s.ResolvidaEm == nil {
		t.Fatal("marco de resolução")
	}
	s = passar(t, s, EstadoEmAtendimento, PapelSolicitante)
	if s.ResolvidaEm != nil {
		t.Fatal("reaberta perde a resolução anterior")
	}
	s = passar(t, s, EstadoResolvido, PapelTecnico)
	s = passar(t, s, EstadoEncerrado, PapelSolicitante)
	if s.EncerradaEm == nil || !s.Finalizada() {
		t.Fatal("encerramento")
	}
	if _, _, err := s.Transicionar(EstadoEmAtendimento, PapelTecnico); err != ErrSolicitacaoFinalizada {
		t.Fatalf("finalizada não volta: %v", err)
	}
	if got, mudou, err := s.Transicionar(EstadoEncerrado, PapelTecnico); err != nil || mudou || got.Estado != EstadoEncerrado {
		t.Fatalf("mesmo estado não é mudança: %v %v", mudou, err)
	}
}

func TestCicloDaRequisicao(t *testing.T) {
	s := aberta(t, TipoRequisicao)
	s, _, _ = s.Atribuir(uuid.New())
	if _, _, err := s.Transicionar(EstadoResolvido, PapelTecnico); err != ErrTransicaoInvalida {
		t.Fatalf("requisição não é resolvida: %v", err)
	}
	s = passar(t, s, EstadoAtendido, PapelTecnico)
	s = passar(t, s, EstadoEncerrado, PapelTecnico)
	if !s.Finalizada() {
		t.Fatal("encerrada")
	}
}

func TestPapelDoSolicitante(t *testing.T) {
	s := aberta(t, TipoRequisicao)
	if got := s.Proximos(PapelSolicitante); !slices.Equal(got, []string{EstadoCancelado}) {
		t.Fatalf("aberto: %v", got)
	}
	s, _, _ = s.Atribuir(uuid.New())
	if _, _, err := s.Transicionar(EstadoAguardandoServidor, PapelSolicitante); err != ErrTransicaoInvalida {
		t.Fatalf("solicitante não conduz o atendimento: %v", err)
	}
	if _, _, err := s.Transicionar(EstadoCancelado, PapelSolicitante); err != ErrTransicaoInvalida {
		t.Fatalf("depois de começar, só a equipe cancela: %v", err)
	}
	s = passar(t, s, EstadoAguardandoServidor, PapelTecnico)
	if got := s.Proximos(PapelSolicitante); len(got) != 0 {
		t.Fatalf("a espera se resolve respondendo, não por botão: %v", got)
	}
	volta, mudou := s.RetomarPorResposta()
	if !mudou || volta.Estado != EstadoEmAtendimento {
		t.Fatalf("resposta devolve à equipe: %+v", volta)
	}
	if _, mudou := volta.RetomarPorResposta(); mudou {
		t.Fatal("fora da espera, responder não muda o estado")
	}
	if got := s.Proximos(PapelTecnico, PapelSolicitante); !slices.Equal(got, []string{EstadoEmAtendimento, EstadoCancelado}) {
		t.Fatalf("união dos papéis sem repetir: %v", got)
	}
	if got := s.Proximos("visitante"); len(got) != 0 {
		t.Fatalf("papel desconhecido: %v", got)
	}
	if got := (Solicitacao{Tipo: "problema", Estado: EstadoAberto}).Proximos(PapelTecnico); len(got) != 0 {
		t.Fatalf("tipo desconhecido não tem passagens: %v", got)
	}
	cancelada := passar(t, aberta(t, TipoIncidente), EstadoCancelado, PapelSolicitante)
	if cancelada.EncerradaEm == nil {
		t.Fatal("cancelamento marca o fim")
	}
}

func TestAtribuir(t *testing.T) {
	s := aberta(t, TipoIncidente)
	ana := uuid.New()
	if _, _, err := s.Atribuir(uuid.Nil); err != ErrResponsavelInvalido {
		t.Fatal(err)
	}
	s, mudou, err := s.Atribuir(ana)
	if err != nil || !mudou || *s.ResponsavelID != ana {
		t.Fatalf("atribuir: %v %v", mudou, err)
	}
	if _, mudou, _ := s.Atribuir(ana); mudou {
		t.Fatal("o mesmo responsável não é mudança")
	}
	s = passar(t, s, EstadoCancelado, PapelTecnico)
	if _, _, err := s.Atribuir(uuid.New()); err != ErrSolicitacaoFinalizada {
		t.Fatal(err)
	}
	if _, mudou := aberta(t, TipoIncidente).RetomarPorResposta(); mudou {
		t.Fatal("sem espera não há retomada")
	}
}

func TestComentario(t *testing.T) {
	s := aberta(t, TipoRequisicao)
	autor := uuid.New()
	c, err := NovoComentario(s, autor, "  pode me ligar?  ", false, false)
	if err != nil || c.Texto != "pode me ligar?" || c.Interno || *c.AutorID != autor || c.SolicitacaoID != s.ID {
		t.Fatalf("resposta: %+v %v", c, err)
	}
	if _, err := NovoComentario(s, autor, "nota", true, false); err != ErrNotaInternaNegada {
		t.Fatal(err)
	}
	if nota, err := NovoComentario(s, autor, "nota", true, true); err != nil || !nota.Interno {
		t.Fatalf("nota da equipe: %+v %v", nota, err)
	}
	if _, err := NovoComentario(s, autor, " ", false, true); err != ErrComentarioInvalido {
		t.Fatal(err)
	}
	if _, err := NovoComentario(s, autor, strings.Repeat("á", TextoMax+1), false, true); err != ErrComentarioInvalido {
		t.Fatal(err)
	}
	fim := passar(t, s, EstadoCancelado, PapelTecnico)
	if _, err := NovoComentario(fim, autor, "ainda?", false, true); err != ErrSolicitacaoFinalizada {
		t.Fatal(err)
	}
	if Conclusao(TipoIncidente) != EstadoResolvido || Conclusao(TipoRequisicao) != EstadoAtendido {
		t.Fatal("conclusão por tipo")
	}
}
