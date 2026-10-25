package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Avaliacao é a pesquisa de satisfação de quem abriu a solicitação.
// Uma por solicitação, imutável: a nota não se reescreve.
type Avaliacao struct {
	ID            uuid.UUID `json:"id"`
	SolicitacaoID uuid.UUID `json:"solicitacao_id"`
	AutorID       uuid.UUID `json:"autor_id"`
	AutorNome     string    `json:"autor_nome"`
	Nota          int       `json:"nota"`
	Comentario    string    `json:"comentario"`
	CreatedAt     time.Time `json:"created_at"`
	Numero        int64     `json:"numero,omitempty"`
	Titulo        string    `json:"titulo,omitempty"`
}

// ResumoAvaliacoes é a média do escopo e as avaliações mais recentes.
type ResumoAvaliacoes struct {
	Total    int
	Soma     int
	Recentes []Avaliacao
}

// PodeAvaliar diz se a solicitação já tem conclusão para receber nota.
func PodeAvaliar(s Solicitacao) bool {
	switch s.Estado {
	case EstadoResolvido, EstadoAtendido, EstadoEncerrado:
		return true
	default:
		return false
	}
}

// NovaAvaliacao valida a nota de quem abriu. O comentário é opcional.
func NovaAvaliacao(s Solicitacao, autor uuid.UUID, nota int, comentario string) (Avaliacao, error) {
	if autor == uuid.Nil || autor != s.SolicitanteID {
		return Avaliacao{}, ErrAvaliacaoAlheia
	}
	if !PodeAvaliar(s) {
		return Avaliacao{}, ErrAvaliacaoForaDeHora
	}
	comentario = strings.TrimSpace(comentario)
	if nota < 1 || nota > 5 || utf8.RuneCountInString(comentario) > TextoMax {
		return Avaliacao{}, ErrAvaliacaoInvalida
	}
	return Avaliacao{
		ID: uuid.New(), SolicitacaoID: s.ID, AutorID: autor, Nota: nota, Comentario: comentario,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// NaoConcluivel explica por que uma mudança não pode concluir esta solicitação.
type NaoConcluivel struct {
	Numero int64
	Estado string
	Causa  error
}

func (e *NaoConcluivel) Error() string {
	if e == nil || e.Causa == nil {
		return "atendimento: solicitação não concluível"
	}
	return e.Causa.Error()
}

func (e *NaoConcluivel) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Causa
}
