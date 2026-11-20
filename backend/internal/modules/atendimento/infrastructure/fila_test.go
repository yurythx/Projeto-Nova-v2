package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type semLinha struct{}

func (semLinha) Scan(...any) error { return pgx.ErrNoRows }

func TestFilaQuandoOBancoFalha(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	id := uuid.New()
	agora := time.Now().UTC()
	regra := domain.Regra{ID: id, Nome: "Senha", Prioridade: domain.PrioridadeBaixa, CreatedAt: agora, UpdatedAt: agora}
	resposta := domain.RespostaPronta{OfertaID: id, Publico: "oi"}
	mensal := domain.ChamadoMensal{ID: id, OfertaID: id, Titulo: "Backup", Dia: 1, CriadoPor: id, CreatedAt: agora, UpdatedAt: agora}

	consultas := map[string]func(database.DBTX) error{
		"regras":       func(db database.DBTX) error { _, err := r.ListarRegras(ctx, db, domain.Filtro{}); return err },
		"da unidade":   func(db database.DBTX) error { _, err := r.ListarRegrasDaUnidade(ctx, db, &id); return err },
		"recorrencias": func(db database.DBTX) error { _, err := r.ListarRecorrencias(ctx, db, domain.Filtro{}); return err },
		"ativas":       func(db database.DBTX) error { _, err := r.ListarRecorrenciasAtivas(ctx, db); return err },
	}
	for nome, call := range consultas {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Fatalf("%s consulta: %v", nome, err)
		}
		if err := call(dbtest.ScanFail{}); err == nil {
			t.Fatalf("%s leitura", nome)
		}
		if err := call(dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Fatalf("%s rows: %v", nome, err)
		}
	}

	linhas := map[string]func(database.DBTX) error{
		"contar": func(db database.DBTX) error { _, err := r.ContarRegras(ctx, db, &id); return err },
		"obter":  func(db database.DBTX) error { _, err := r.ObterRegra(ctx, db, id); return err },
		"mensal": func(db database.DBTX) error { _, err := r.ObterRecorrencia(ctx, db, id); return err },
		"deflexao": func(db database.DBTX) error {
			_, _, err := r.MedirDeflexao(ctx, db, "2026-01-01", false, nil)
			return err
		},
		"resposta":    func(db database.DBTX) error { _, err := r.ObterResposta(ctx, db, id); return err },
		"reivindicar": func(db database.DBTX) error { _, err := r.ReivindicarAbertura(ctx, db, id, 2026, 1); return err },
		"do mes": func(db database.DBTX) error {
			_, err := r.SolicitacaoDoMes(ctx, db, id, id, "Backup", agora)
			return err
		},
	}
	for nome, call := range linhas {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Fatalf("%s: %v", nome, err)
		}
	}
	if _, err := r.ObterRegra(ctx, &dbtest.Seq{Row: semLinha{}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("regra ausente: %v", err)
	}
	if item, err := r.ObterResposta(ctx, &dbtest.Seq{Row: semLinha{}}, id); err != nil || item.OfertaID != id {
		t.Fatalf("resposta vazia: %+v %v", item, err)
	}
	if ok, err := r.ReivindicarAbertura(ctx, &dbtest.Seq{Row: semLinha{}}, id, 2026, 1); err != nil || ok {
		t.Fatalf("mês já aberto: %v %v", ok, err)
	}
	if achado, err := r.SolicitacaoDoMes(ctx, &dbtest.Seq{Row: semLinha{}}, id, id, "Backup", agora); err != nil || achado != uuid.Nil {
		t.Fatalf("mês sem chamado: %v %v", achado, err)
	}

	gravacoes := map[string]func(database.DBTX) error{
		"inserir regra":    func(db database.DBTX) error { return r.InserirRegra(ctx, db, regra) },
		"salvar resposta":  func(db database.DBTX) error { return r.SalvarResposta(ctx, db, resposta) },
		"excluir resposta": func(db database.DBTX) error { return r.ExcluirResposta(ctx, db, id) },
		"inserir mensal":   func(db database.DBTX) error { return r.InserirRecorrencia(ctx, db, mensal) },
		"ligar":            func(db database.DBTX) error { return r.LigarAbertura(ctx, db, id, 2026, 1, id) },
		"deflexao": func(db database.DBTX) error {
			return r.RegistrarDeflexao(ctx, db, id, "senha", "2026-01-01", &id, true)
		},
	}
	for nome, call := range gravacoes {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Fatalf("%s: %v", nome, err)
		}
	}
	if err := r.ExcluirRegra(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if err := r.ExcluirRegra(ctx, dbtest.ScanFail{}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("regra já saída: %v", err)
	}
	if err := r.ExcluirRecorrencia(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Fatal(err)
	}
	if err := r.ExcluirRecorrencia(ctx, dbtest.ScanFail{}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("recorrência já saída: %v", err)
	}
}
