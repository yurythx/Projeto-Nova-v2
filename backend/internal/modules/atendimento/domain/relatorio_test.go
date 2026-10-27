package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidarOLA(t *testing.T) {
	zero, um, demais := 0, 1, 9000
	if ValidarOLA(nil) != nil || ValidarOLA(&um) != nil {
		t.Fatal("válidos")
	}
	for _, h := range []*int{&zero, &demais} {
		if !errors.Is(ValidarOLA(h), ErrOLAInvalido) {
			t.Errorf("%d", *h)
		}
	}
	e := Equipe{ID: uuid.New(), Nome: "Rede", Membros: []uuid.UUID{}}
	com := e
	com.OLAHoras = &um
	if e.Igual(com) || !com.Igual(com) {
		t.Fatal("o prazo entra na comparação")
	}
}

func TestPorSecretaria(t *testing.T) {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	em := base.Add(3 * time.Hour)
	saude, depto, educacao, orfa := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	secSaude := Secretaria{ID: saude, Nome: "Saúde", Sigla: "SMS"}
	secretarias := map[uuid.UUID]Secretaria{saude: secSaude, depto: secSaude, educacao: {ID: educacao, Nome: "Educação"}}
	a, b, c, d, e := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	lista := []Solicitacao{
		{ID: a, UnidadeID: &saude, Estado: EstadoResolvido, CreatedAt: base, ResolvidaEm: &em},
		{ID: b, UnidadeID: &depto, Estado: EstadoAberto, CreatedAt: base},
		{ID: c, UnidadeID: &depto, Estado: EstadoCancelado, CreatedAt: base},
		{ID: d, UnidadeID: &educacao, Estado: EstadoEmAtendimento, CreatedAt: base},
		{ID: e, UnidadeID: &orfa, Estado: EstadoAberto, CreatedAt: base},
		{ID: uuid.New(), Estado: EstadoAberto, CreatedAt: base},
	}
	quatro := 4
	avaliacoes := map[uuid.UUID]Avaliacao{a: {Nota: 5, Facilidade: &quatro}, d: {Nota: 3}}
	linhas := PorSecretaria(lista, secretarias, nil, avaliacoes, MetasXLA{})
	if len(linhas) != 3 {
		t.Fatalf("linhas: %+v", linhas)
	}
	s := linhas[0]
	if s.Nome != "Saúde" || s.Sigla != "SMS" || *s.UnidadeID != saude || s.Total != 3 || s.Concluidas != 1 || s.Abertas != 1 || s.Canceladas != 1 {
		t.Fatalf("o departamento conta para a secretaria: %+v", s)
	}
	if *s.Experiencia.CSAT.Valor != 5 || *s.Experiencia.Facilidade.Valor != 4 || *s.Experiencia.EsperaHoras.Valor != 3 {
		t.Fatalf("experiência da secretaria: %+v", s.Experiencia)
	}
	// Sem unidade e com unidade fora da árvore vão juntos, em "Institucional".
	if linhas[1].Nome != "Institucional" || linhas[1].Total != 2 || linhas[1].UnidadeID != nil || linhas[2].Nome != "Educação" {
		t.Fatalf("ordem e institucional: %+v", linhas[1:])
	}
	total := LinhaDe(lista, nil, avaliacoes, MetasXLA{})
	if total.Total != 6 || *total.Experiencia.CSAT.Valor != 4 || total.Experiencia.Facilidade.Amostra != 1 {
		t.Fatalf("total: %+v", total)
	}
	// Empate no volume: a ordem é pelo nome.
	dois := PorSecretaria([]Solicitacao{lista[0], lista[3]}, secretarias, nil, nil, MetasXLA{})
	if dois[0].Nome != "Educação" || dois[1].Nome != "Saúde" {
		t.Fatalf("empate por nome: %+v", dois)
	}
}

func TestResumirEquipes(t *testing.T) {
	agora := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	rede, suporte := uuid.New(), uuid.New()
	ola := 24
	h := func(n int) *time.Time { v := agora.Add(time.Duration(-100+n) * time.Hour); return &v }
	criada := agora.Add(-100 * time.Hour)
	tarefas := []TarefaMedida{
		{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaConcluida, CreatedAt: criada, FeitaEm: h(10)},
		{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaConcluida, CreatedAt: criada, FeitaEm: h(30)},
		{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaPendente, CreatedAt: criada},
		{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaPendente, CreatedAt: agora.Add(-time.Hour)},
		{EquipeID: rede, EquipeNome: "Rede", OLAHoras: &ola, Estado: TarefaDispensada, CreatedAt: criada, FeitaEm: h(1)},
		{EquipeID: suporte, EquipeNome: "Atendimento", Estado: TarefaConcluida, CreatedAt: criada, FeitaEm: h(5)},
		{EquipeID: suporte, EquipeNome: "Atendimento", Estado: TarefaPendente, CreatedAt: criada},
	}
	linhas := ResumirEquipes(tarefas, agora)
	if len(linhas) != 2 || linhas[0].Nome != "Atendimento" {
		t.Fatalf("por nome: %+v", linhas)
	}
	sem := linhas[0]
	if sem.NoPrazoPercentual != nil || sem.Atrasadas != 0 || *sem.MedianaHoras != 5 || sem.Pendentes != 1 {
		t.Fatalf("sem OLA não há prazo: %+v", sem)
	}
	r := linhas[1]
	if r.Concluidas != 2 || r.NoPrazo != 1 || *r.NoPrazoPercentual != 50 || *r.MedianaHoras != 20 || r.Pendentes != 2 || r.Atrasadas != 1 {
		t.Fatalf("OLA da rede: %+v", r)
	}
	if out := ResumirEquipes(nil, agora); len(out) != 0 {
		t.Fatal("vazio")
	}
}
