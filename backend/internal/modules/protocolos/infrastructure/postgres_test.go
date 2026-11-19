package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

func TestErrosDoBanco(t *testing.T) {
	r := NewPostgresRepository()
	ctx := context.Background()
	id := uuid.New()
	v := domain.Visao{Unidades: []uuid.UUID{}, Entidades: []uuid.UUID{}}
	f := domain.Formulario{ID: id, Campos: []domain.Campo{}}
	s := domain.Servico{ID: id, Requisitos: []domain.Orientacao{}, Documentos: []domain.Orientacao{}}
	p := domain.Pedido{ID: id, Respostas: map[string]any{}, Campos: []domain.Campo{}}
	chamadas := []func(database.DBTX) error{
		func(db database.DBTX) error { _, _, err := r.Unidade(ctx, db, id); return err },
		func(db database.DBTX) error { return r.InserirFormulario(ctx, db, f) },
		func(db database.DBTX) error { return r.AtualizarFormulario(ctx, db, f) },
		func(db database.DBTX) error { _, err := r.BuscarFormulario(ctx, db, id); return err },
		func(db database.DBTX) error { _, err := r.ListarFormularios(ctx, db, v); return err },
		func(db database.DBTX) error { return r.InserirServico(ctx, db, s) },
		func(db database.DBTX) error { return r.AtualizarServico(ctx, db, s) },
		func(db database.DBTX) error { _, err := r.BuscarServico(ctx, db, id); return err },
		func(db database.DBTX) error { _, err := r.ListarServicos(ctx, db, true, nil); return err },
		func(db database.DBTX) error { _, err := r.ListarServicosDaGestao(ctx, db, v); return err },
		func(db database.DBTX) error { return r.InserirPedido(ctx, db, p) },
		func(db database.DBTX) error { return r.AtualizarPedido(ctx, db, p) },
		func(db database.DBTX) error { _, err := r.BuscarPedido(ctx, db, id); return err },
		func(db database.DBTX) error { _, err := r.BuscarPorNumero(ctx, db, "X"); return err },
		func(db database.DBTX) error { _, err := r.ListarPedidos(ctx, db, v); return err },
		func(db database.DBTX) error { _, err := r.NumeroExiste(ctx, db, "X"); return err },
		func(db database.DBTX) error { _, err := r.CPFEmAberto(ctx, db, id, "1"); return err },
		func(db database.DBTX) error {
			return r.InserirHistorico(ctx, db, domain.Historico{ID: id, CreatedAt: time.Now()})
		},
		func(db database.DBTX) error { _, err := r.ListarHistorico(ctx, db, id); return err },
		func(db database.DBTX) error {
			return r.InserirResponsavel(ctx, db, domain.ResponsavelTecnico{ID: id})
		},
		func(db database.DBTX) error { _, err := r.ListarResponsaveis(ctx, db); return err },
		func(db database.DBTX) error { _, err := r.ResponsavelAtivo(ctx, db, "1"); return err },
		func(db database.DBTX) error { return r.InserirAgendamento(ctx, db, domain.Agendamento{ID: id}) },
		func(db database.DBTX) error { _, err := r.BuscarAgendamento(ctx, db, id); return err },
		func(db database.DBTX) error { _, err := r.ContarAgendamentos(ctx, db, id, "2026-11-16"); return err },
		func(db database.DBTX) error { _, _, err := r.LerFaltas(ctx, db, "1"); return err },
		func(db database.DBTX) error { return r.SomarFalta(ctx, db, "1", time.Now()) },
	}
	for i, ch := range chamadas {
		if err := ch(dbtest.Fail{}); err == nil {
			t.Fatalf("chamada %d", i)
		}
	}
	if err := r.AtualizarFormulario(ctx, dbtest.ScanFail{}, f); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if err := r.AtualizarServico(ctx, dbtest.ScanFail{}, s); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if err := r.AtualizarPedido(ctx, dbtest.ScanFail{}, p); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
	if _, err := jsonBytes(make(chan int)); err == nil {
		t.Fatal("json")
	}
	if err := lerJSON(nil, &f.Campos); err != nil {
		t.Fatal(err)
	}
	if err := lerJSON([]byte("{"), &f.Campos); err == nil {
		t.Fatal("json ruim")
	}
	if err := traduz(nil); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(traduz(&pgconn.PgError{Code: "23505", ConstraintName: "protocolos_pedidos_numero_key"}), domain.ErrNumeroRepetido) {
		t.Fatal("número")
	}
	if !errors.Is(traduz(&pgconn.PgError{Code: "23505", ConstraintName: "protocolos_responsaveis_cpf_key"}), domain.ErrCPFCadastrado) {
		t.Fatal("cpf")
	}
	if traduz(errors.New("outro")) == nil {
		t.Fatal("outro")
	}
}
