package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Aprovação em etapas da requisição (ADR 068).
const (
	DecisaoAprovada = "aprovada"
	DecisaoRecusa   = "recusada"

	MaxEtapas      = 5
	MaxAprovadores = 20
	maxNomeEtapa   = 80
	// MaxMotivoRecusa deixa a resposta a quem pediu ("Pedido recusado na
	// etapa …: motivo") sempre dentro do limite do comentário.
	MaxMotivoRecusa = 1000
)

var (
	ErrAguardandoAprovacao  = errors.New("atendimento: o pedido ainda espera aprovação")
	ErrEtapasSoRequisicao   = errors.New("atendimento: só requisição tem etapas de aprovação")
	ErrEtapaInvalida        = errors.New("atendimento: etapa de aprovação inválida")
	ErrSemAprovacaoPendente = errors.New("atendimento: este pedido não espera aprovação")
	ErrNaoAprovador         = errors.New("atendimento: você não aprova esta etapa")
	ErrAutoaprovacao        = errors.New("atendimento: quem pediu não aprova o próprio pedido")
	ErrMotivoObrigatorio    = errors.New("atendimento: a recusa precisa de motivo, e o motivo cabe em 1000 caracteres")
)

// EtapaAprovacao é um passo da aprovação: basta um dos aprovadores.
type EtapaAprovacao struct {
	Ordem       int         `json:"ordem"`
	Nome        string      `json:"nome"`
	Aprovadores []uuid.UUID `json:"aprovadores"`
}

// Aprovacao é uma decisão gravada.
type Aprovacao struct {
	ID            uuid.UUID  `json:"id"`
	SolicitacaoID uuid.UUID  `json:"solicitacao_id"`
	Etapa         int        `json:"etapa"`
	EtapaNome     string     `json:"etapa_nome"`
	AprovadorID   *uuid.UUID `json:"aprovador_id,omitempty"`
	AprovadorNome string     `json:"aprovador_nome,omitempty"`
	Decisao       string     `json:"decisao"`
	Motivo        string     `json:"motivo,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ValidarEtapas confere as etapas de uma oferta e as numera em ordem.
// Lista vazia tira a aprovação da oferta.
func ValidarEtapas(tipo string, etapas []EtapaAprovacao) ([]EtapaAprovacao, error) {
	if len(etapas) == 0 {
		return []EtapaAprovacao{}, nil
	}
	if tipo != TipoRequisicao {
		return nil, ErrEtapasSoRequisicao
	}
	if len(etapas) > MaxEtapas {
		return nil, ErrEtapaInvalida
	}
	out := make([]EtapaAprovacao, 0, len(etapas))
	for i, e := range etapas {
		nome := strings.TrimSpace(e.Nome)
		if nome == "" || utf8.RuneCountInString(nome) > maxNomeEtapa || len(e.Aprovadores) == 0 || len(e.Aprovadores) > MaxAprovadores {
			return nil, ErrEtapaInvalida
		}
		vistos := []uuid.UUID{}
		for _, a := range e.Aprovadores {
			if a == uuid.Nil || slices.Contains(vistos, a) {
				return nil, ErrEtapaInvalida
			}
			vistos = append(vistos, a)
		}
		out = append(out, EtapaAprovacao{Ordem: i, Nome: nome, Aprovadores: vistos})
	}
	return out, nil
}

// AguardarAprovacao põe a requisição na primeira etapa.
func (s Solicitacao) AguardarAprovacao() Solicitacao {
	zero := 0
	s.Estado, s.EtapaAtual = EstadoAguardandoAprovacao, &zero
	return s
}

// Decidir aplica a decisão de quem aprova a etapa atual. Aprovar a última
// libera para a fila (liberada); recusar, com motivo, cancela.
func (s Solicitacao) Decidir(etapas []EtapaAprovacao, quem uuid.UUID, aprovar bool, motivo string, agora time.Time) (Solicitacao, Aprovacao, bool, error) {
	if s.Estado != EstadoAguardandoAprovacao || s.EtapaAtual == nil || *s.EtapaAtual >= len(etapas) {
		return Solicitacao{}, Aprovacao{}, false, ErrSemAprovacaoPendente
	}
	etapa := etapas[*s.EtapaAtual]
	if !slices.Contains(etapa.Aprovadores, quem) {
		return Solicitacao{}, Aprovacao{}, false, ErrNaoAprovador
	}
	if quem == s.SolicitanteID {
		return Solicitacao{}, Aprovacao{}, false, ErrAutoaprovacao
	}
	motivo = strings.TrimSpace(motivo)
	if utf8.RuneCountInString(motivo) > MaxMotivoRecusa {
		return Solicitacao{}, Aprovacao{}, false, ErrMotivoObrigatorio
	}
	a := Aprovacao{ID: uuid.New(), SolicitacaoID: s.ID, Etapa: etapa.Ordem, EtapaNome: etapa.Nome, AprovadorID: &quem,
		Decisao: DecisaoAprovada, Motivo: motivo, CreatedAt: agora}
	s.UpdatedAt = agora
	if !aprovar {
		if motivo == "" {
			return Solicitacao{}, Aprovacao{}, false, ErrMotivoObrigatorio
		}
		a.Decisao = DecisaoRecusa
		s.Estado, s.EtapaAtual, s.EncerradaEm = EstadoCancelado, nil, &agora
		return s, a, false, nil
	}
	if *s.EtapaAtual+1 < len(etapas) {
		proxima := *s.EtapaAtual + 1
		s.EtapaAtual = &proxima
		return s, a, false, nil
	}
	s.Estado, s.EtapaAtual = EstadoAberto, nil
	return s, a, true, nil
}
