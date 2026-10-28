package application

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

func silencio() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// pronta cria a conexão direta com credenciais, um chamado e a tabela.
func pronta(t *testing.T, s *Service) domain.Conexao {
	t.Helper()
	ctx := context.Background()
	c := conexao(t, s, domain.ModoDireto)
	if err := s.DefinirCredenciais(ctx, admin(), c.ID, domain.Credenciais{Login: "yuri", Senha: "s"}); err != nil {
		t.Fatal(err)
	}
	d := json.RawMessage(`{}`)
	if err := (preparo{s, c.ID}).Gravar(ctx, []domain.Item{{Tipo: domain.ItemChamado, ID: 1, Dados: d}, {Tipo: domain.ItemPessoa, ID: 9, Dados: d}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SalvarTabela(ctx, admin(), c.ID, []domain.Linha{{Secretaria: "S", Categoria: -1, Oferta: "O", Tipo: "incidente", Prioridade: "alta"}}); err != nil {
		t.Fatal(err)
	}
	return c
}

// enfileirar põe a tarefa direto no banco, sem a conferência do pedido.
func enfileirar(t *testing.T, s *Service, c uuid.UUID, tipo string) domain.Tarefa {
	t.Helper()
	tf, _ := domain.NovaTarefa(c, tipo, "teste")
	if err := infrastructure.NewRepository().InserirTarefa(context.Background(), s.pool, tf); err != nil {
		t.Fatal(err)
	}
	return tf
}

func estado(t *testing.T, s *Service, id uuid.UUID) domain.Tarefa {
	t.Helper()
	tf, err := infrastructure.NewRepository().GetTarefa(context.Background(), s.pool, id)
	if err != nil {
		t.Fatal(err)
	}
	return tf
}

func TestWorkerExtraiSimulaECarrega(t *testing.T) {
	ctx := context.Background()
	s, ext, car := novoServico(t, infrastructure.NewRepository())
	s.WithLogger(silencio())
	c := pronta(t, s)

	ex, err := s.PedirTarefa(ctx, admin(), c.ID, domain.TarefaExtracao, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if tf := estado(t, s, ex.ID); tf.Estado != domain.EstadoConcluida || !strings.Contains(string(tf.Relatorio), `"yuri"`) || tf.Feitos != 2 || ext.chamou != 1 {
		t.Fatalf("extração: %+v", tf)
	}
	if g, _ := s.ObterConexao(ctx, admin(), c.ID); g.ExtraidoEm == nil {
		t.Fatal("a data da extração")
	}

	sim, _ := s.PedirTarefa(ctx, admin(), c.ID, domain.TarefaSimulacao, "")
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if tf := estado(t, s, sim.ID); tf.Estado != domain.EstadoConcluida || car.gravou || car.chamados != 2 || len(car.tabela) != 1 {
		t.Fatalf("simulação: %+v %+v", tf, car)
	}

	// A carga falha: o erro (cortado) fica na tarefa.
	car.err = errFalha
	if _, err := s.repo.Contagens(ctx, s.pool, c.ID); err != nil {
		t.Fatal(err)
	}
	carga := enfileirar(t, s, c.ID, domain.TarefaCarga)
	car.err = &erroLongo{}
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if tf := estado(t, s, carga.ID); tf.Estado != domain.EstadoFalhou || len(tf.Erro) != 2000 || !car.gravou {
		t.Fatalf("carga que falhou: %+v", tf)
	}

	// O relatório que não vira JSON também é falha.
	car.err, car.rel = nil, map[string]any{"canal": make(chan int)}
	ruim := enfileirar(t, s, c.ID, domain.TarefaSimulacao)
	_ = s.Passo(ctx)
	if tf := estado(t, s, ruim.ID); tf.Estado != domain.EstadoFalhou {
		t.Fatalf("relatório ilegível: %+v", tf)
	}

	// O desfazer vai ao carregador, simulando ou gravando.
	car.rel = nil
	for _, tipo := range []string{domain.TarefaDesfazerSimulacao, domain.TarefaDesfazer} {
		des := enfileirar(t, s, c.ID, tipo)
		if err := s.Passo(ctx); err != nil {
			t.Fatal(err)
		}
		if tf := estado(t, s, des.ID); tf.Estado != domain.EstadoConcluida || !car.desfez || car.gravou != (tipo == domain.TarefaDesfazer) ||
			!strings.Contains(string(tf.Relatorio), `"chamados": 3`) {
			t.Fatalf("%s: %+v %+v", tipo, tf, car)
		}
	}
}

type erroLongo struct{}

func (*erroLongo) Error() string { return strings.Repeat("e", 3000) }

func TestWorkerCanceladaEParada(t *testing.T) {
	ctx := context.Background()
	s, ext, car := novoServico(t, infrastructure.NewRepository())
	s.WithLogger(silencio())
	c := pronta(t, s)

	// Cancelada na tela durante a carga: para e fica cancelada.
	tf := enfileirar(t, s, c.ID, domain.TarefaSimulacao)
	car.esperar = true
	car.antes = func() { _ = s.CancelarTarefa(ctx, admin(), tf.ID) }
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if got := estado(t, s, tf.ID); got.Estado != domain.EstadoCancelada {
		t.Fatalf("cancelada: %+v", got)
	}

	// O serviço desligando no meio: a tarefa fica em andamento, e o Passo
	// seguinte a devolve à fila quando parar de dar sinal de vida.
	car.antes, car.esperar = nil, false
	ext.esperar = true
	ex := enfileirar(t, s, c.ID, domain.TarefaExtracao)
	desligar, parar := context.WithCancel(ctx)
	go func() { time.Sleep(100 * time.Millisecond); parar() }()
	if err := s.Passo(desligar); err != nil {
		t.Fatal(err)
	}
	if got := estado(t, s, ex.ID); got.Estado != domain.EstadoEmAndamento {
		t.Fatalf("parada no meio: %+v", got)
	}
	ext.esperar = false
	s.Parada = -time.Minute
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if got := estado(t, s, ex.ID); got.Estado != domain.EstadoConcluida {
		t.Fatalf("retomada e concluída: %+v", got)
	}
}

func TestWorkerFalhas(t *testing.T) {
	ctx := context.Background()
	base, _, _ := novoServico(t, infrastructure.NewRepository())
	c := pronta(t, base)
	semCred := conexao(t, base, domain.ModoDireto)
	com := func(metodos ...string) *Service {
		return NewService(base.pool, falhando(metodos...), eventos{}, base.cipher).WithExtrator(&extrator{}).
			WithCarregador(&carregador{}).WithLogger(silencio())
	}
	falhou := func(nome string, s *Service, conexao uuid.UUID, tipo string) {
		t.Helper()
		tf := enfileirar(t, s, conexao, tipo)
		if err := s.Passo(ctx); err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if got := estado(t, s, tf.ID); got.Estado != domain.EstadoFalhou {
			t.Fatalf("%s: %+v", nome, got)
		}
	}
	falhou("conexão", com("GetConexao"), c.ID, domain.TarefaSimulacao)
	falhou("sem credenciais", com(), semCred.ID, domain.TarefaExtracao)
	falhou("tabela", com("ListarTabela"), c.ID, domain.TarefaCarga)
	semPortas := NewService(base.pool, infrastructure.NewRepository(), eventos{}, base.cipher).WithLogger(silencio())
	falhou("sem extrator", semPortas, c.ID, domain.TarefaExtracao)
	falhou("sem carregador", semPortas, c.ID, domain.TarefaCarga)

	// O fim que não grava só fica no registro; a tarefa sobra em andamento.
	tf := enfileirar(t, base, c.ID, domain.TarefaSimulacao)
	if err := com("Terminar").Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if got := estado(t, base, tf.ID); got.Estado != domain.EstadoEmAndamento {
		t.Fatalf("fim sem gravar: %+v", got)
	}
	_ = base.CancelarTarefa(ctx, admin(), tf.ID)

	for _, m := range []string{"Retomar", "Pegar"} {
		if err := com(m).Passo(ctx); err == nil {
			t.Errorf("%s: sem erro", m)
		}
	}
	// Cancelar grava a auditoria na mesma transação.
	outra := enfileirar(t, base, c.ID, domain.TarefaSimulacao)
	if err := NewService(base.pool, infrastructure.NewRepository(), eventos{errFalha}, nil).CancelarTarefa(ctx, admin(), outra.ID); err == nil {
		t.Fatal("cancelar sem a auditoria")
	}
	if _, err := base.ObterConexao(ctx, auth.Identity{}, c.ID); status(err) == 0 {
		t.Fatal("obter sem permissão")
	}
}

func TestRunTarefas(t *testing.T) {
	base, _, _ := novoServico(t, infrastructure.NewRepository())
	s := NewService(base.pool, falhando("Retomar"), eventos{}, nil).WithLogger(silencio())
	s.Intervalo = 10 * time.Millisecond
	ctx, parar := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer parar()
	if err := s.RunTarefas(ctx); err != nil {
		t.Fatal(err)
	}
}
