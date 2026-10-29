package application

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// A sincronização até a virada (ADR 081): ligada, o worker pede a
// extração a cada 15 minutos e a carga vem logo depois dela.
func TestSincronizacao(t *testing.T) {
	ctx := context.Background()
	s, ext, car := novoServico(t, infrastructure.NewRepository())
	s.WithLogger(silencio())
	agora := time.Now()
	s.agora = func() time.Time { return agora }
	c := pronta(t, s)

	// Sem unidade na tabela, a carga não roda sozinha: recusa ligar.
	if _, err := s.Sincronizar(ctx, admin(), c.ID, true); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("sem unidade: %v", err)
	}
	u := dbtest.Unidade(t, s.pool)
	if _, err := s.SalvarTabela(ctx, admin(), c.ID, []domain.Linha{{Secretaria: "S", Categoria: -1, Oferta: "O", Tipo: "incidente",
		Prioridade: "alta", UnidadeID: &u}}); err != nil {
		t.Fatal(err)
	}
	if ativa, _ := s.SincronizacaoAtiva(ctx); ativa {
		t.Fatal("desligada de início")
	}
	ligada, err := s.Sincronizar(ctx, admin(), c.ID, true)
	if err != nil || !ligada.Sincronizar {
		t.Fatalf("ligar: %+v %v", ligada, err)
	}
	if ativa, err := s.SincronizacaoAtiva(ctx); !ativa || err != nil {
		t.Fatalf("ativa: %v", err)
	}

	// Um passo: a extração e, logo depois dela, a carga, as duas pela
	// sincronização.
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	lista, _ := s.ListarTarefas(ctx, admin(), c.ID)
	if len(lista) != 2 || lista[0].Tipo != domain.TarefaCarga || lista[1].Tipo != domain.TarefaExtracao ||
		lista[0].PedidaPor != domain.PedidaPelaSincronizacao || lista[0].Estado != domain.EstadoConcluida || ext.chamou != 1 || !car.gravou {
		t.Fatalf("tarefas: %+v", lista)
	}
	// Antes do intervalo, nada; depois dele, a próxima extração.
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if ext.chamou != 1 {
		t.Fatalf("antes do intervalo: %d", ext.chamou)
	}
	agora = agora.Add(domain.IntervaloSincronizacao + time.Minute)
	if err := s.Passo(ctx); err != nil {
		t.Fatal(err)
	}
	if ext.chamou != 2 {
		t.Fatalf("depois do intervalo: %d", ext.chamou)
	}

	// Desligada na virada: o worker não pede mais nada.
	if d, err := s.Sincronizar(ctx, admin(), c.ID, false); err != nil || d.Sincronizar {
		t.Fatalf("desligar: %+v %v", d, err)
	}
	agora = agora.Add(time.Hour)
	if err := s.Passo(ctx); err != nil || ext.chamou != 2 {
		t.Fatalf("desligada: %d %v", ext.chamou, err)
	}

	// As falhas de cada passo.
	com := func(metodos ...string) *Service {
		f := falhando(metodos...)
		f.alvo = c.ID
		svc := NewService(s.pool, f, eventos{}, s.cipher).WithExtrator(ext).WithCarregador(car).WithLogger(silencio())
		svc.agora = s.agora
		return svc
	}
	if _, err := com("SetSincronizar").Sincronizar(ctx, admin(), c.ID, true); !errors.Is(err, errFalha) {
		t.Fatalf("gravar a chave: %v", err)
	}
	if _, err := s.Sincronizar(ctx, admin(), uuid.New(), false); status(err) != http.StatusNotFound {
		t.Fatalf("conexão que não existe: %v", err)
	}
	if _, err := com("ListarTabela").Sincronizar(ctx, admin(), c.ID, true); !errors.Is(err, errFalha) {
		t.Fatalf("conferir: %v", err)
	}
	if err := com("ParaSincronizar").Passo(ctx); !errors.Is(err, errFalha) {
		t.Fatalf("procurar: %v", err)
	}
	if _, err := s.Sincronizar(ctx, admin(), c.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := com("InserirTarefa").Passo(ctx); !errors.Is(err, errFalha) {
		t.Fatalf("pedir a extração: %v", err)
	}
	// Uma tarefa pedida pela tela no meio tempo faz a vez da extração.
	pela := enfileirar(t, s, c.ID, domain.TarefaSimulacao)
	if err := com("ParaSincronizar:todas").agendar(ctx); err != nil {
		t.Fatalf("com tarefa ativa: %v", err)
	}
	_ = s.Passo(ctx)
	if tf := estado(t, s, pela.ID); tf.Estado != domain.EstadoConcluida {
		t.Fatalf("a da tela: %+v", tf)
	}
	// A carga que vem da extração falha junto com ela.
	ex := enfileirar(t, s, c.ID, domain.TarefaExtracao)
	tf := estado(t, s, ex.ID)
	for _, metodo := range []string{"GetConexao", "InserirTarefa"} {
		if err := com(metodo).terminar(ctx, tf, nil, ""); !errors.Is(err, errFalha) {
			t.Errorf("terminar com falha em %s: %v", metodo, err)
		}
	}
	_ = s.repo.Cancelar(ctx, s.pool, ex.ID)
	_, _ = s.Sincronizar(ctx, admin(), c.ID, false)
}
