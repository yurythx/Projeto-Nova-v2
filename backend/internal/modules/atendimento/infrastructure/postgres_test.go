package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

type rowErr struct{ err error }

func (r rowErr) Scan(...any) error { return r.err }

type rowOK struct{}

func (rowOK) Scan(dest ...any) error {
	if len(dest) == 0 {
		return nil
	}
	if m, ok := dest[len(dest)-1].(*[]uuid.UUID); ok {
		*m = []uuid.UUID{}
	}
	return nil
}

type rowNota struct{ total, soma int }

func (n rowNota) Scan(dest ...any) error {
	if p, ok := dest[0].(*int); ok {
		*p = n.total
	}
	if len(dest) > 1 {
		if p, ok := dest[1].(*int); ok {
			*p = n.soma
		}
	}
	return nil
}

type rowBlank struct{}

func (rowBlank) Scan(...any) error { return nil }

type queryDB struct {
	rows pgx.Rows
	err  error
	row  pgx.Row
}

func (q queryDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, q.err
}
func (q queryDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return q.rows, q.err }
func (q queryDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if q.row != nil {
		return q.row
	}
	return rowErr{err: q.err}
}

type okRows struct{ left int }

func (r *okRows) Close()                                       {}
func (r *okRows) Err() error                                   { return nil }
func (r *okRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *okRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *okRows) Scan(dest ...any) error {
	if len(dest) > 0 {
		if m, ok := dest[len(dest)-1].(*[]uuid.UUID); ok && m != nil {
			*m = []uuid.UUID{uuid.New()}
		}
	}
	return nil
}
func (r *okRows) Values() ([]any, error) { return nil, nil }
func (r *okRows) RawValues() [][]byte    { return nil }
func (r *okRows) Conn() *pgx.Conn        { return nil }
func (r *okRows) Next() bool {
	if r.left == 0 {
		return false
	}
	r.left--
	return true
}

func TestRepositorioPropagaFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	o := domain.Oferta{ID: id, Nome: "E-mail", Ativa: true}
	e := domain.Equipe{ID: id, Nome: "N1"}
	calls := map[string]func(database.DBTX) error{
		"ListOfertas":  func(db database.DBTX) error { _, err := r.ListOfertas(ctx, db, domain.Filtro{}); return err },
		"GetOferta":    func(db database.DBTX) error { _, err := r.GetOferta(ctx, db, id); return err },
		"InsertOferta": func(db database.DBTX) error { _, err := r.InsertOferta(ctx, db, o); return err },
		"UpdateOferta": func(db database.DBTX) error { _, err := r.UpdateOferta(ctx, db, o); return err },
		"ListEquipes":  func(db database.DBTX) error { _, err := r.ListEquipes(ctx, db, domain.Filtro{}); return err },
		"GetEquipe":    func(db database.DBTX) error { _, err := r.GetEquipe(ctx, db, id); return err },
		"InsertEquipe": func(db database.DBTX) error { _, err := r.InsertEquipe(ctx, db, e); return err },
		"UpdateEquipe": func(db database.DBTX) error { _, err := r.UpdateEquipe(ctx, db, e); return err },
		"ContarAtivos": func(db database.DBTX) error { _, err := r.ContarAtivos(ctx, db, []uuid.UUID{id}); return err },
		"Candidatos":   func(db database.DBTX) error { _, err := r.Candidatos(ctx, db, id); return err },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	for _, name := range []string{"ListOfertas", "ListEquipes", "Candidatos"} {
		if err := calls[name](dbtest.ScanFail{}); err == nil {
			t.Errorf("%s com linha ilegível", name)
		}
		if err := calls[name](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s no fim da leitura: %v", name, err)
		}
		listaErr := calls[name](queryDB{rows: &okRows{left: 1}})
		if listaErr != nil {
			t.Errorf("%s com uma linha: %v", name, listaErr)
		}
		if err := calls[name](queryDB{rows: dbtest.NoRows()}); err != nil {
			t.Errorf("%s vazia: %v", name, err)
		}
	}
	fk := &pgconn.PgError{Code: "23503", ConstraintName: "atendimento_ofertas_unidade_id_fkey"}
	if _, err := r.InsertOferta(ctx, queryDB{row: rowErr{fk}}, o); !errors.Is(err, domain.ErrUnidadeInexistente) {
		t.Errorf("unidade: %v", err)
	}
	membro := &pgconn.PgError{Code: "23503", ConstraintName: "atendimento_equipe_membros_user_id_fkey"}
	seq := &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, {Err: membro}}}
	if err := r.SubstituirMembros(ctx, seq, id, []uuid.UUID{id}); !errors.Is(err, domain.ErrMembroInvalido) {
		t.Errorf("membro: %v", err)
	}
	if err := r.SubstituirMembros(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{{Err: dbtest.ErrInjected}}}, id, nil); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("delete: %v", err)
	}
	if err := r.SubstituirMembros(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK}}, id, nil); err != nil {
		t.Errorf("sem membros: %v", err)
	}
	if err := r.SubstituirMembros(ctx, &dbtest.Seq{Execs: []dbtest.ExecResult{dbtest.OK, dbtest.OK}}, id, []uuid.UUID{id}); err != nil {
		t.Errorf("grava membros: %v", err)
	}
	if _, err := r.GetOferta(ctx, queryDB{row: rowErr{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("oferta ausente: %v", err)
	}
	if _, err := r.GetEquipe(ctx, queryDB{row: rowErr{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("equipe ausente: %v", err)
	}
	if _, err := r.UpdateOferta(ctx, queryDB{row: rowErr{&pgconn.PgError{Code: "23514"}}}, o); !errors.Is(err, domain.ErrEntradaInvalida) {
		t.Errorf("check: %v", err)
	}
	if _, err := r.InsertOferta(ctx, queryDB{row: rowOK{}}, o); err != nil {
		t.Errorf("insert oferta: %v", err)
	}
	if _, err := r.UpdateOferta(ctx, queryDB{row: rowOK{}}, o); err != nil {
		t.Errorf("update oferta: %v", err)
	}
	salva, err := r.InsertEquipe(ctx, queryDB{row: rowOK{}}, domain.Equipe{ID: id, Nome: "N1"})
	if err != nil || salva.Membros == nil {
		t.Errorf("insert equipe: %+v %v", salva, err)
	}
	if _, err := r.UpdateEquipe(ctx, queryDB{row: rowOK{}}, domain.Equipe{ID: id, Nome: "N1", Membros: []uuid.UUID{id}}); err != nil {
		t.Errorf("update equipe: %v", err)
	}
	if _, err := r.GetEquipe(ctx, queryDB{row: rowOK{}}, id); err != nil {
		t.Errorf("get equipe: %v", err)
	}
	if _, err := r.ContarAtivos(ctx, queryDB{row: rowOK{}}, nil); err != nil {
		t.Errorf("contagem: %v", err)
	}
	if _, err := r.GetEquipe(ctx, queryDB{row: rowBlank{}}, id); err != nil {
		t.Errorf("equipe sem array de membros: %v", err)
	}
}

func TestPersistenciaReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	pessoa := dbtest.User(t, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO user_scopes (user_id, perfil_id, unidade_id)
		SELECT $1, id, $2 FROM perfis WHERE sistema LIMIT 1`, pessoa, u); err != nil {
		// Sem perfil de sistema o candidato não aparece; a oferta e a equipe seguem.
		t.Logf("lotação de teste não gravada: %v", err)
	}
	o, err := domain.NovaOferta("E-mail "+uuid.NewString()[:6], "acesso", domain.TipoRequisicao, nil, &u, true)
	if err != nil {
		t.Fatal(err)
	}
	salvo, err := r.InsertOferta(ctx, pool, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, salvo.ID)
	})
	e, err := domain.NovaEquipe("N1 "+uuid.NewString()[:6], &u, []uuid.UUID{pessoa})
	if err != nil {
		t.Fatal(err)
	}
	equipe, err := r.InsertEquipe(ctx, pool, e)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_equipes WHERE id = $1`, equipe.ID)
	})
	if err := r.SubstituirMembros(ctx, pool, equipe.ID, []uuid.UUID{pessoa}); err != nil {
		t.Fatal(err)
	}
	lida, err := r.GetEquipe(ctx, pool, equipe.ID)
	if err != nil || len(lida.Membros) != 1 {
		t.Fatalf("membros: %+v %v", lida, err)
	}
	if _, err := r.ContarAtivos(ctx, pool, []uuid.UUID{pessoa, uuid.New()}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Candidatos(ctx, pool, u); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetOferta(ctx, pool, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal(err)
	}
}

func TestSolicitacoesPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	s := domain.Solicitacao{ID: id}
	calls := map[string]func(database.DBTX) error{
		"ListOfertasAtivas": func(db database.DBTX) error { _, err := r.ListOfertasAtivas(ctx, db); return err },
		"ListSolicitacoes": func(db database.DBTX) error {
			_, _, err := r.ListSolicitacoes(ctx, db, domain.FiltroSolicitacao{})
			return err
		},
		"GetSolicitacao":       func(db database.DBTX) error { _, err := r.GetSolicitacao(ctx, db, id); return err },
		"TravarSolicitacao":    func(db database.DBTX) error { return r.TravarSolicitacao(ctx, db, id) },
		"InsertSolicitacao":    func(db database.DBTX) error { _, err := r.InsertSolicitacao(ctx, db, s); return err },
		"UpdateSolicitacao":    func(db database.DBTX) error { return r.UpdateSolicitacao(ctx, db, s) },
		"InsertPassagem":       func(db database.DBTX) error { _, err := r.InsertPassagem(ctx, db, domain.Passagem{}); return err },
		"ListPassagens":        func(db database.DBTX) error { _, err := r.ListPassagens(ctx, db, id); return err },
		"InsertComentario":     func(db database.DBTX) error { _, err := r.InsertComentario(ctx, db, domain.Comentario{}); return err },
		"ListComentarios":      func(db database.DBTX) error { _, err := r.ListComentarios(ctx, db, id, true); return err },
		"Responsaveis":         func(db database.DBTX) error { _, err := r.Responsaveis(ctx, db, &id); return err },
		"ResponsavelValido":    func(db database.DBTX) error { _, err := r.ResponsavelValido(ctx, db, id, nil); return err },
		"ListApontamentos":     func(db database.DBTX) error { _, err := r.ListApontamentos(ctx, db, id); return err },
		"InsertApontamento":    func(db database.DBTX) error { _, err := r.InsertApontamento(ctx, db, domain.Apontamento{}); return err },
		"DeleteApontamento":    func(db database.DBTX) error { return r.DeleteApontamento(ctx, db, id, id) },
		"SolicitacaoPorNumero": func(db database.DBTX) error { _, err := r.SolicitacaoPorNumero(ctx, db, 1); return err },
		"EmailDe":              func(db database.DBTX) error { _, err := r.EmailDe(ctx, db, id); return err },
		"ReservarCorreio":      func(db database.DBTX) error { _, err := r.ReservarCorreio(ctx, db, "m", id); return err },
		"LiberarCorreio":       func(db database.DBTX) error { return r.LiberarCorreio(ctx, db, "m") },
		"GetAvaliacao":         func(db database.DBTX) error { _, err := r.GetAvaliacao(ctx, db, id); return err },
		"InsertAvaliacao":      func(db database.DBTX) error { _, err := r.InsertAvaliacao(ctx, db, domain.Avaliacao{}); return err },
		"ResumoAvaliacoes": func(db database.DBTX) error {
			_, err := r.ResumoAvaliacoes(ctx, db, false, nil, 5)
			return err
		},
		"ListarParaGovernanca": func(db database.DBTX) error {
			_, err := r.ListarParaGovernanca(ctx, db, false, nil, 10)
			return err
		},
		"UnidadesComSolicitacao": func(db database.DBTX) error {
			_, err := r.UnidadesComSolicitacao(ctx, db)
			return err
		},
		"GravarRetrato": func(db database.DBTX) error {
			return r.GravarRetrato(ctx, db, domain.Retrato{Dia: time.Now()})
		},
		"ListarRetratos": func(db database.DBTX) error {
			_, err := r.ListarRetratos(ctx, db, time.Now(), false, nil)
			return err
		},
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := calls["ListOfertasAtivas"](dbtest.ScanFail{}); err == nil {
		t.Error("linha ilegível")
	}
	if err := calls["ListOfertasAtivas"](dbtest.RowsErr{}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("fim da leitura: %v", err)
	}
	if err := calls["ListSolicitacoes"](queryDB{row: rowOK{}, err: dbtest.ErrInjected}); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("contagem feita, página falha: %v", err)
	}
	if err := r.UpdateSolicitacao(ctx, dbtest.ScanFail{}, s); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("nenhuma linha atualizada: %v", err)
	}
	if _, err := r.InsertAvaliacao(ctx, queryDB{err: &pgconn.PgError{Code: "23505"}}, domain.Avaliacao{}); !errors.Is(err, domain.ErrAvaliacaoExistente) {
		t.Errorf("avaliação repetida: %v", err)
	}
	nota := rowNota{total: 2, soma: 9}
	if _, err := r.ResumoAvaliacoes(ctx, &dbtest.Seq{Row: nota, Queries: []dbtest.QueryResult{{Err: dbtest.ErrInjected}}}, false, nil, 5); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("resumo sem a lista: %v", err)
	}
	if resumo, err := r.ResumoAvaliacoes(ctx, &dbtest.Seq{Row: nota, Queries: []dbtest.QueryResult{{Rows: dbtest.NoRows()}}}, false, nil, 5); err != nil || resumo.Total != 2 || resumo.Recentes == nil {
		t.Errorf("resumo vazio: %+v %v", resumo, err)
	}
	if resumo, err := r.ResumoAvaliacoes(ctx, &dbtest.Seq{Row: nota}, false, nil, 0); err != nil || len(resumo.Recentes) != 0 {
		t.Errorf("sem recentes: %+v %v", resumo, err)
	}
	if err := r.TravarSolicitacao(ctx, queryDB{row: rowErr{pgx.ErrNoRows}}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("trava de solicitação ausente: %v", err)
	}
	fk := map[string]error{
		"atendimento_solicitacoes_oferta_id_fkey":      domain.ErrOfertaInexistente,
		"atendimento_comentarios_solicitacao_id_fkey":  domain.ErrNaoEncontrado,
		"atendimento_solicitacoes_responsavel_id_fkey": domain.ErrResponsavelInvalido,
		"atendimento_solicitacoes_solicitante_id_fkey": domain.ErrSolicitanteInvalido,
		"atendimento_comentarios_autor_id_fkey":        domain.ErrSolicitanteInvalido,
		"atendimento_solicitacoes_unidade_id_fkey":     domain.ErrUnidadeInexistente,
	}
	for constraint, want := range fk {
		err := r.UpdateSolicitacao(ctx, queryDB{err: &pgconn.PgError{Code: "23503", ConstraintName: constraint}}, s)
		if !errors.Is(err, want) {
			t.Errorf("%s: %v", constraint, err)
		}
	}
}

func TestPrazosPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	calls := map[string]func(database.DBTX) error{
		"GetPolitica":      func(db database.DBTX) error { _, err := r.GetPolitica(ctx, db, &id); return err },
		"ResolverPolitica": func(db database.DBTX) error { _, err := r.ResolverPolitica(ctx, db, &id); return err },
		"SalvarPolitica":   func(db database.DBTX) error { _, err := r.SalvarPolitica(ctx, db, domain.Politica{}); return err },
		"ListFeriados":     func(db database.DBTX) error { _, err := r.ListFeriados(ctx, db, nil); return err },
		"FeriadosVigentes": func(db database.DBTX) error {
			_, err := r.FeriadosVigentes(ctx, db, nil, "2026-01-01", "2026-12-31")
			return err
		},
		"GetFeriado":       func(db database.DBTX) error { _, err := r.GetFeriado(ctx, db, id); return err },
		"InsertFeriado":    func(db database.DBTX) error { _, err := r.InsertFeriado(ctx, db, domain.Feriado{}); return err },
		"DeleteFeriado":    func(db database.DBTX) error { return r.DeleteFeriado(ctx, db, id) },
		"PassagensDe":      func(db database.DBTX) error { _, err := r.PassagensDe(ctx, db, []uuid.UUID{id}); return err },
		"ComPrazoCorrendo": func(db database.DBTX) error { _, err := r.ComPrazoCorrendo(ctx, db); return err },
		"IncidentesRecorrentes": func(db database.DBTX) error {
			_, err := r.IncidentesRecorrentes(ctx, db, false, nil, time.Now(), 3)
			return err
		},
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := calls["PassagensDe"](dbtest.ScanFail{}); err == nil {
		t.Error("passagem ilegível")
	}
	if err := r.DeleteFeriado(ctx, dbtest.ScanFail{}, id); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("feriado ausente: %v", err)
	}
	if _, err := r.GetPolitica(ctx, queryDB{row: rowOK{}}, nil); err == nil {
		t.Error("política com JSON ilegível")
	}
	repetido := &pgconn.PgError{Code: "23505", ConstraintName: "atendimento_feriados_dia_uq"}
	if _, err := r.InsertFeriado(ctx, queryDB{row: rowErr{repetido}}, domain.Feriado{}); !errors.Is(err, domain.ErrFeriadoRepetido) {
		t.Errorf("feriado repetido: %v", err)
	}
}

func TestPrazosReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	topo := dbtest.Unidade(t, pool)
	var sub uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO unidades (entidade_id, parent_id, nome, slug)
		SELECT entidade_id, id, 'Sub', 'sub-' || substr(gen_random_uuid()::text, 1, 8) FROM unidades WHERE id = $1
		RETURNING id`, topo).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IN ($1, $2)`, topo, sub)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id IN ($1, $2)`, topo, sub)
		_, _ = pool.Exec(ctx, `DELETE FROM unidades WHERE id = $1`, sub)
	})
	if _, err := r.GetPolitica(ctx, pool, &sub); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("sem política própria: %v", err)
	}
	p := domain.Politica{ID: uuid.New(), UnidadeID: &topo, Fuso: "America/Sao_Paulo",
		Turnos: []domain.Turno{{Dia: 1, Inicio: "08:00", Fim: "12:00"}},
		Matriz: []domain.Celula{{Impacto: 1, Urgencia: 1, Prioridade: domain.PrioridadeBaixa}},
		Metas:  []domain.Meta{{Prioridade: domain.PrioridadeBaixa, RespostaMin: 60, SolucaoMin: 120}}}
	salva, err := r.SalvarPolitica(ctx, pool, p)
	if err != nil || salva.Turnos[0].Inicio != "08:00" {
		t.Fatalf("salvar: %+v %v", salva, err)
	}
	p.Fuso = "America/Cuiaba"
	if again, err := r.SalvarPolitica(ctx, pool, p); err != nil || again.ID != salva.ID || again.Fuso != "America/Cuiaba" {
		t.Fatalf("substitui a da unidade: %+v %v", again, err)
	}
	herdada, err := r.ResolverPolitica(ctx, pool, &sub)
	if err != nil || !herdada.Herdada || herdada.ID != salva.ID {
		t.Fatalf("a subunidade herda: %+v %v", herdada, err)
	}
	propria, err := r.ResolverPolitica(ctx, pool, &topo)
	if err != nil || propria.Herdada {
		t.Fatalf("a própria não é herdada: %+v %v", propria, err)
	}

	f, _ := domain.NovoFeriado(&topo, "2026-11-20", "Consciência Negra", domain.FeriadoNacional)
	if _, err := r.InsertFeriado(ctx, pool, f); err != nil {
		t.Fatal(err)
	}
	f.ID = uuid.New()
	if _, err := r.InsertFeriado(ctx, pool, f); !errors.Is(err, domain.ErrFeriadoRepetido) {
		t.Fatalf("mesma data na mesma unidade: %v", err)
	}
	vigentes, err := r.FeriadosVigentes(ctx, pool, &sub, "2026-11-01", "2026-11-30")
	if err != nil || len(vigentes) == 0 || vigentes[len(vigentes)-1].Data != "2026-11-20" {
		t.Fatalf("o feriado da unidade vale na subunidade: %+v %v", vigentes, err)
	}
	if lista, err := r.ListFeriados(ctx, pool, &sub); err != nil || len(lista) != 0 {
		t.Fatalf("a lista é só da própria unidade: %+v %v", lista, err)
	}
	if got, err := r.PassagensDe(ctx, pool, nil); err != nil || len(got) != 0 {
		t.Fatalf("sem ids: %v %v", got, err)
	}
	if _, err := r.ComPrazoCorrendo(ctx, pool); err != nil {
		t.Fatal(err)
	}
}

func TestVisitasPropagamFalhas(t *testing.T) {
	r := NewRepository()
	ctx := context.Background()
	id := uuid.New()
	v := domain.Visita{ID: id}
	calls := map[string]func(database.DBTX) error{
		"GetVisita":          func(db database.DBTX) error { _, err := r.GetVisita(ctx, db, id); return err },
		"TravarVisita":       func(db database.DBTX) error { return r.TravarVisita(ctx, db, id) },
		"InsertVisita":       func(db database.DBTX) error { return r.InsertVisita(ctx, db, v) },
		"UpdateVisita":       func(db database.DBTX) error { return r.UpdateVisita(ctx, db, v) },
		"ListVisitas":        func(db database.DBTX) error { _, err := r.ListVisitas(ctx, db, id); return err },
		"VisitasEmAndamento": func(db database.DBTX) error { _, err := r.VisitasEmAndamento(ctx, db, domain.Filtro{}); return err },
		"InsertEvidencia":    func(db database.DBTX) error { _, err := r.InsertEvidencia(ctx, db, domain.Evidencia{}); return err },
		"GetEvidencia":       func(db database.DBTX) error { _, err := r.GetEvidencia(ctx, db, id); return err },
		"ApagarLocalizacoes": func(db database.DBTX) error { _, err := r.ApagarLocalizacoes(ctx, db, time.Now()); return err },
	}
	for name, call := range calls {
		if err := call(dbtest.Fail{}); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("%s com o banco fora: %v", name, err)
		}
	}
	if err := r.UpdateVisita(ctx, dbtest.ScanFail{}, v); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("visita ausente: %v", err)
	}
	aberta := &pgconn.PgError{Code: "23505", ConstraintName: "uq_atendimento_visitas_aberta"}
	if err := r.InsertVisita(ctx, queryDB{err: aberta}, v); !errors.Is(err, domain.ErrVisitaAberta) {
		t.Errorf("segunda visita aberta: %v", err)
	}
	if _, err := r.DadosDoTitular(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("dados do titular: %v", err)
	}
	if err := r.ApagarLocalizacoesDe(ctx, dbtest.Fail{}, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("eliminação: %v", err)
	}
	for i := 1; i <= 3; i++ {
		qs := make([]dbtest.QueryResult, i)
		for j := range qs {
			qs[j] = dbtest.QueryResult{Rows: dbtest.NoRows()}
		}
		seq := &dbtest.Seq{Queries: append(qs, dbtest.QueryResult{Err: dbtest.ErrInjected})}
		if _, err := r.DadosDoTitular(ctx, seq, id); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("dados do titular, consulta %d: %v", i+1, err)
		}
	}
	for i := 0; i <= 3; i++ {
		qs := make([]dbtest.QueryResult, i)
		for j := range qs {
			qs[j] = dbtest.QueryResult{Rows: dbtest.NoRows()}
		}
		seq := &dbtest.Seq{Queries: append(qs, dbtest.QueryResult{Rows: dbtest.ErrRows()}, dbtest.QueryResult{Rows: dbtest.NoRows()})}
		if _, err := r.DadosDoTitular(ctx, seq, id); !errors.Is(err, dbtest.ErrInjected) {
			t.Errorf("leitura dos dados do titular, consulta %d: %v", i+1, err)
		}
	}
	// Visitas lidas, evidências falham.
	seq := &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: &okRows{left: 1}}, {Err: dbtest.ErrInjected}}}
	if _, err := r.ListVisitas(ctx, seq, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("evidências: %v", err)
	}
	seq = &dbtest.Seq{Queries: []dbtest.QueryResult{{Rows: &okRows{left: 1}}, {Rows: dbtest.ErrRows()}}}
	if _, err := r.ListVisitas(ctx, seq, id); !errors.Is(err, dbtest.ErrInjected) {
		t.Errorf("leitura das evidências: %v", err)
	}
}

func TestVisitasReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	tec, quem := dbtest.User(t, pool), dbtest.User(t, pool)
	o, _ := domain.NovaOferta("Rede "+uuid.NewString()[:6], "", domain.TipoIncidente, nil, &u, true)
	if _, err := r.InsertOferta(ctx, pool, o); err != nil {
		t.Fatal(err)
	}
	nova, _ := domain.NovaSolicitacao(o, "Sem rede", "", quem)
	sol, err := r.InsertSolicitacao(ctx, pool, nova)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_solicitacoes WHERE id = $1`, sol.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_ofertas WHERE id = $1`, o.ID)
	})
	agora := time.Now()
	destino := domain.Destino{Lat: -23.5512, Lng: -46.634, RaioM: 50}
	v, err := domain.NovaVisita(sol, uuid.New(), tec, "Escola", &destino, domain.Ponto{Lat: -23.5503, Lng: -46.6339, PrecisaoM: 8}, agora, agora)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.InsertVisita(ctx, pool, v); err != nil {
		t.Fatal(err)
	}
	outra, _ := domain.NovaVisita(sol, uuid.New(), tec, "", nil, domain.Ponto{}, agora, agora)
	if err := r.InsertVisita(ctx, pool, outra); !errors.Is(err, domain.ErrVisitaAberta) {
		t.Fatalf("uma visita aberta por técnico e solicitação: %v", err)
	}
	if err := r.TravarVisita(ctx, pool, v.ID); err != nil {
		t.Fatal(err)
	}
	lida, err := r.GetVisita(ctx, pool, v.ID)
	if err != nil || lida.Inicio == nil || lida.Destino == nil || lida.Destino.RaioM != 50 || lida.TecnicoNome == "" {
		t.Fatalf("leitura: %+v %v", lida, err)
	}
	em, err := r.VisitasEmAndamento(ctx, pool, domain.Filtro{Restrito: true, Unidades: []uuid.UUID{u}})
	if err != nil || len(em) != 1 || em[0].Numero != sol.Numero || em[0].Titulo != "Sem rede" || em[0].Inicio == nil {
		t.Fatalf("em campo: %+v %v", em, err)
	}
	e := domain.Evidencia{ID: uuid.New(), VisitaID: v.ID, Tipo: domain.EvidenciaFoto, ObjectKey: "atendimento/visitas/" + v.ID.String() + "/x.jpg",
		ContentType: "image/jpeg", Tamanho: 10, AutorID: &tec, CreatedAt: agora}
	gravada, err := r.InsertEvidencia(ctx, pool, e)
	if err != nil || gravada.ID != e.ID {
		t.Fatalf("evidência: %+v %v", gravada, err)
	}
	e2 := e
	e2.ID = uuid.New()
	if again, err := r.InsertEvidencia(ctx, pool, e2); err != nil || again.ID != e.ID {
		t.Fatalf("a mesma chave devolve a já gravada: %+v %v", again, err)
	}
	if got, err := r.GetEvidencia(ctx, pool, e.ID); err != nil || got.ObjectKey != e.ObjectKey {
		t.Fatalf("get evidência: %+v %v", got, err)
	}
	c, _, _ := lida.Concluir(domain.Ponto{Lat: -23.55, Lng: -46.63, PrecisaoM: 5}, agora.Add(time.Minute), agora.Add(2*time.Minute), "Maria", "ok")
	if err := r.UpdateVisita(ctx, pool, c); err != nil {
		t.Fatal(err)
	}
	visitas, err := r.ListVisitas(ctx, pool, sol.ID)
	if err != nil || len(visitas) != 1 || visitas[0].Fim == nil || len(visitas[0].Evidencias) != 1 {
		t.Fatalf("lista: %+v %v", visitas, err)
	}
	if vazias, err := r.ListVisitas(ctx, pool, uuid.New()); err != nil || len(vazias) != 0 {
		t.Fatalf("sem visitas: %v %v", vazias, err)
	}

	// Um ano e um minuto depois, as coordenadas somem; o resto fica.
	n, err := r.ApagarLocalizacoes(ctx, pool, c.LocalizacaoExpiraEm.Add(time.Minute))
	if err != nil || n < 1 {
		t.Fatalf("retenção: %d %v", n, err)
	}
	apagada, _ := r.GetVisita(ctx, pool, v.ID)
	if apagada.Inicio != nil || apagada.Fim != nil || apagada.Destino != nil || apagada.DistanciaM == nil || apagada.AssinanteNome != "Maria" {
		t.Fatalf("depois da retenção: %+v", apagada)
	}
	if n, _ := r.ApagarLocalizacoes(ctx, pool, c.LocalizacaoExpiraEm.Add(time.Minute)); n != 0 {
		t.Fatal("apagar de novo não toca em nada")
	}

	// Direitos do titular.
	pedidos, err := r.DadosDoTitular(ctx, pool, quem)
	if err != nil || len(pedidos.Solicitacoes) != 1 || len(pedidos.Visitas) != 0 {
		t.Fatalf("dados de quem abriu: %+v %v", pedidos, err)
	}
	campo, err := r.DadosDoTitular(ctx, pool, tec)
	if err != nil || len(campo.Visitas) != 1 || len(campo.Solicitacoes) != 0 {
		t.Fatalf("dados do técnico: %+v %v", campo, err)
	}
	if err := r.ApagarLocalizacoesDe(ctx, pool, tec); err != nil {
		t.Fatal(err)
	}
}

func TestSerieReal(t *testing.T) {
	pool := dbtest.Pool(t)
	r := NewRepository()
	ctx := context.Background()
	u := dbtest.Unidade(t, pool)
	dia := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	mttr := 1.5
	p := domain.Retrato{Dia: dia, UnidadeID: &u, MTTRHoras: &mttr, SomaMTTRMin: 90, MTTRN: 1}
	if err := r.GravarRetrato(ctx, pool, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM atendimento_indicadores_dia WHERE dia = $1`, dia)
	})
	mttr = 3
	p.MTTRHoras = &mttr
	p.SomaMTTRMin = 180
	if err := r.GravarRetrato(ctx, pool, p); err != nil {
		t.Fatal(err)
	}
	lista, err := r.ListarRetratos(ctx, pool, dia, true, []uuid.UUID{u})
	if err != nil || len(lista) != 1 || lista[0].MTTRHoras == nil || *lista[0].MTTRHoras != 3 || lista[0].SomaMTTRMin != 180 {
		t.Fatalf("regravou o dia: %+v %v", lista, err)
	}
	if err := r.GravarRetrato(ctx, pool, domain.Retrato{Dia: dia}); err != nil {
		t.Fatal(err)
	}
	// O banco é o da suíte inteira: outros testes gravam a série de hoje.
	// Vale só o dia que este teste gravou.
	tudo, err := r.ListarRetratos(ctx, pool, dia, false, nil)
	var doDia []domain.Retrato
	for _, p := range tudo {
		if p.Dia.Equal(dia) {
			doDia = append(doDia, p)
		}
	}
	if err != nil || len(doDia) != 1 || doDia[0].UnidadeID != nil {
		t.Fatalf("linha institucional: %+v %v", tudo, err)
	}
}
