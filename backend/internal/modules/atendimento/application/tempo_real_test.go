package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/domain/events"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/ws"
)

type publicado struct{ topico, tipo string }

type fakeHub struct {
	saida []publicado
	err   error
}

func (f *fakeHub) Publish(_ context.Context, topic, frameType string, _ any) error {
	f.saida = append(f.saida, publicado{topic, frameType})
	return f.err
}

func (f *fakeHub) topicos() []string {
	out := []string{}
	for _, p := range f.saida {
		out = append(out, p.topico)
	}
	return out
}

// solRepo devolve uma solicitação fixa; o resto vem do stub.
type solRepo struct {
	stubRepo
	sol domain.Solicitacao
	err error
}

func (r solRepo) GetSolicitacao(context.Context, database.DBTX, uuid.UUID) (domain.Solicitacao, error) {
	return r.sol, r.err
}

func evento(t *testing.T, tipo string, payload map[string]any) events.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return events.Event{ID: uuid.New(), Type: tipo, Payload: raw}
}

func atendente(unidade uuid.UUID) auth.Identity {
	return auth.Identity{UserID: uuid.New(), Scopes: []auth.Scope{{
		UnidadeID: &unidade, Permissions: []string{"atendimento:atender"}, Unidades: []uuid.UUID{unidade},
	}}}
}

func TestHandleEventoPublicaParaQuemVe(t *testing.T) {
	ctx := context.Background()
	u, id, quem := uuid.New(), uuid.New(), uuid.New()
	hub := &fakeHub{}
	s := application.NewService(nil, stubRepo{}, fakeEventos{}, nil).WithTempoReal(hub)

	base := map[string]any{"id": id.String(), "numero": 7, "estado": "em_atendimento", "unidade_id": u.String(), "solicitante_id": quem.String()}
	if err := s.HandleEvento(ctx, evento(t, application.EventoSolicitacaoEstado, base)); err != nil {
		t.Fatal(err)
	}
	want := []string{
		application.TopicoUnidade(&u), application.TopicoTodas, application.TopicoEquipe(id),
		application.TopicoSolicitacao(id), ws.UserTopic(quem.String()),
	}
	if !slices.Equal(hub.topicos(), want) || hub.saida[0].tipo != application.FrameSolicitacao {
		t.Fatalf("mudança comum: %+v", hub.saida)
	}

	hub.saida = nil
	nota := map[string]any{"id": id.String(), "estado": "em_atendimento", "solicitante_id": quem.String(), "interno": true}
	if err := s.HandleEvento(ctx, evento(t, application.EventoSolicitacaoComentada, nota)); err != nil {
		t.Fatal(err)
	}
	if want := []string{application.TopicoInstitucional, application.TopicoTodas, application.TopicoEquipe(id)}; !slices.Equal(hub.topicos(), want) {
		t.Fatalf("nota interna só para a equipe: %v", hub.topicos())
	}

	hub.saida = nil
	if err := s.HandleEvento(ctx, events.Event{Payload: []byte("{")}); err != nil || len(hub.saida) != 0 {
		t.Fatalf("payload ilegível é descartado: %v %v", err, hub.saida)
	}
	if err := application.NewService(nil, stubRepo{}, fakeEventos{}, nil).HandleEvento(ctx, evento(t, "x", base)); err != nil {
		t.Fatalf("sem tempo real, só confirma: %v", err)
	}
	falha := application.NewService(nil, stubRepo{}, fakeEventos{}, nil).WithTempoReal(&fakeHub{err: errors.New("redis")})
	if err := falha.HandleEvento(ctx, evento(t, "x", base)); err == nil {
		t.Fatal("falha do Redis volta para o retry")
	}
}

func TestTopicosEAutorizacao(t *testing.T) {
	ctx := context.Background()
	u, outra := uuid.New(), uuid.New()
	quem := uuid.New()
	sol := domain.Solicitacao{ID: uuid.New(), UnidadeID: &u, SolicitanteID: quem}
	s := application.NewService(nil, solRepo{sol: sol}, fakeEventos{}, nil)

	if got := s.TopicosDaFila(admin()); !slices.Equal(got, []string{application.TopicoTodas}) {
		t.Fatalf("global: %v", got)
	}
	if got := s.TopicosDaFila(atendente(u)); !slices.Equal(got, []string{application.TopicoUnidade(&u)}) {
		t.Fatalf("unidade: %v", got)
	}
	if got := s.TopicosDaFila(auth.Identity{}); len(got) != 0 {
		t.Fatalf("sem atendimento: %v", got)
	}

	cliente := func(id auth.Identity) ws.ClientInfo {
		return ws.ClientInfo{UserID: id.UserID.String(), Roles: id.Roles, Scopes: id.Scopes}
	}
	tecnico, servidor := cliente(atendente(u)), ws.ClientInfo{UserID: quem.String()}
	casos := []struct {
		nome   string
		quem   ws.ClientInfo
		topico string
		ok     bool
	}{
		{"técnico na unidade", tecnico, application.TopicoUnidade(&u), true},
		{"técnico fora da unidade", tecnico, application.TopicoUnidade(&outra), false},
		{"unidade ilegível", tecnico, "atendimento:unidade:x", false},
		{"todas só global", tecnico, application.TopicoTodas, false},
		{"todas para o global", cliente(admin()), application.TopicoTodas, true},
		{"institucional só global", tecnico, application.TopicoInstitucional, false},
		{"técnico vê a solicitação", tecnico, application.TopicoSolicitacao(sol.ID), true},
		{"técnico na equipe", tecnico, application.TopicoEquipe(sol.ID), true},
		{"quem abriu vê a solicitação", servidor, application.TopicoSolicitacao(sol.ID), true},
		{"quem abriu não entra na equipe", servidor, application.TopicoEquipe(sol.ID), false},
		{"estranho", ws.ClientInfo{UserID: uuid.NewString()}, application.TopicoSolicitacao(sol.ID), false},
		{"id ilegível", servidor, "atendimento:solicitacao:x", false},
		{"tópico desconhecido", tecnico, "atendimento:outro", false},
	}
	for _, c := range casos {
		err := s.Authorize(ctx, c.quem, c.topico)
		if (err == nil) != c.ok {
			t.Errorf("%s: %v", c.nome, err)
		}
	}
	sumida := application.NewService(nil, solRepo{err: domain.ErrNaoEncontrado}, fakeEventos{}, nil)
	if err := sumida.Authorize(ctx, servidor, application.TopicoSolicitacao(sol.ID)); err == nil {
		t.Fatal("solicitação inexistente")
	}
}
