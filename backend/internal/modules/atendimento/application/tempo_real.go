package application

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/domain/events"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/ws"
)

// Tempo real da fila (fase 3). O worker consome os eventos da solicitação
// que já saíram pelo outbox e avisa quem está com a tela aberta. O frame
// não carrega conteúdo: diz qual solicitação mudou e a tela busca de novo
// pela API, que aplica as regras de leitura.
//
// Tópicos:
//   - atendimento:todas                quem atende com concessão global
//   - atendimento:institucional        solicitações sem unidade (só global)
//   - atendimento:unidade:<id>         a fila de uma unidade
//   - atendimento:solicitacao:<id>     quem vê a solicitação (sem notas internas)
//   - atendimento:equipe:<id>          a equipe, inclusive as notas internas
//   - user:<solicitante>               a lista de quem abriu
const (
	TopicoTodas         = "atendimento:todas"
	TopicoInstitucional = "atendimento:institucional"
	prefixoUnidade      = "atendimento:unidade:"
	prefixoSolicitacao  = "atendimento:solicitacao:"
	prefixoEquipe       = "atendimento:equipe:"

	// FrameSolicitacao é o tipo do frame de mudança.
	FrameSolicitacao = "atendimento.solicitacao"
)

// Publicador entrega um frame aos assinantes do tópico em todas as
// réplicas (ws.Hub).
type Publicador interface {
	Publish(ctx context.Context, topic, frameType string, data any) error
}

// WithTempoReal liga a publicação dos frames. Sem ele, o consumidor só
// confirma os eventos.
func (s *Service) WithTempoReal(p Publicador) *Service {
	s.tempoReal = p
	return s
}

// TopicoUnidade é o tópico da fila de uma unidade (nil = institucional).
func TopicoUnidade(unidade *uuid.UUID) string {
	if unidade == nil {
		return TopicoInstitucional
	}
	return prefixoUnidade + unidade.String()
}

// TopicoSolicitacao e TopicoEquipe são os tópicos da tela de uma solicitação.
func TopicoSolicitacao(id uuid.UUID) string { return prefixoSolicitacao + id.String() }
func TopicoEquipe(id uuid.UUID) string      { return prefixoEquipe + id.String() }

// FrameMudanca é o que a tela recebe.
type FrameMudanca struct {
	ID     uuid.UUID `json:"id"`
	Numero int64     `json:"numero"`
	Estado string    `json:"estado"`
	Evento string    `json:"evento"`
}

type eventoSolicitacao struct {
	ID            uuid.UUID  `json:"id"`
	Numero        int64      `json:"numero"`
	Estado        string     `json:"estado"`
	UnidadeID     *uuid.UUID `json:"unidade_id"`
	SolicitanteID uuid.UUID  `json:"solicitante_id"`
	Interno       bool       `json:"interno"`
}

// HandleEvento publica a mudança nos tópicos de quem pode vê-la. A nota
// interna só vai para a equipe. Payload ilegível é confirmado e
// descartado (reprocessar não o conserta); falha do Redis volta para a
// política de retry.
func (s *Service) HandleEvento(ctx context.Context, ev events.Event) error {
	var p eventoSolicitacao
	if err := json.Unmarshal(ev.Payload, &p); err != nil || p.ID == uuid.Nil || s.tempoReal == nil {
		return nil
	}
	frame := FrameMudanca{ID: p.ID, Numero: p.Numero, Estado: p.Estado, Evento: ev.Type}
	topicos := []string{TopicoUnidade(p.UnidadeID), TopicoTodas, TopicoEquipe(p.ID)}
	if !p.Interno {
		topicos = append(topicos, TopicoSolicitacao(p.ID))
		if p.SolicitanteID != uuid.Nil {
			topicos = append(topicos, ws.UserTopic(p.SolicitanteID.String()))
		}
	}
	for _, t := range topicos {
		if err := s.tempoReal.Publish(ctx, t, FrameSolicitacao, frame); err != nil {
			return err
		}
	}
	return nil
}

// TopicosDaFila devolve os tópicos que a fila de identity assina.
func (s *Service) TopicosDaFila(identity auth.Identity) []string {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if cob.All {
		return []string{TopicoTodas}
	}
	out := []string{}
	for _, u := range cob.Unidades {
		t := TopicoUnidade(&u)
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// Authorize é o autorizador dos tópicos "atendimento:*": as mesmas regras
// da API, sobre a identidade da conexão.
func (s *Service) Authorize(ctx context.Context, client ws.ClientInfo, topic string) error {
	identity := client.Identity()
	switch {
	case topic == TopicoTodas, topic == TopicoInstitucional:
		if auth.CanGlobal(identity, auth.PermAtendimentoAtender) {
			return nil
		}
	case strings.HasPrefix(topic, prefixoUnidade):
		u, err := uuid.Parse(strings.TrimPrefix(topic, prefixoUnidade))
		if err == nil && auth.Can(identity, auth.PermAtendimentoAtender, auth.InUnidade(u)) {
			return nil
		}
	case strings.HasPrefix(topic, prefixoSolicitacao):
		if ps, ok := s.papeisEm(ctx, identity, strings.TrimPrefix(topic, prefixoSolicitacao)); ok && len(ps) > 0 {
			return nil
		}
	case strings.HasPrefix(topic, prefixoEquipe):
		if ps, ok := s.papeisEm(ctx, identity, strings.TrimPrefix(topic, prefixoEquipe)); ok && slices.Contains(ps, domain.PapelTecnico) {
			return nil
		}
	}
	return ws.ErrTopicForbidden
}

func (s *Service) papeisEm(ctx context.Context, identity auth.Identity, idTexto string) ([]string, bool) {
	id, err := uuid.Parse(idTexto)
	if err != nil {
		return nil, false
	}
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, id)
	if err != nil {
		return nil, false
	}
	return papeis(identity, sol), true
}
