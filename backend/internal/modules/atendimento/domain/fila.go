package domain

import (
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Recusa explica o cadastro da fila que não entra.
type Recusa struct{ Msg string }

func (e Recusa) Error() string { return e.Msg }

// Regra é a triagem curta de uma unidade. A primeira que casa vale.
// Sem condição, casa com todo chamado da unidade.
type Regra struct {
	ID                   uuid.UUID  `json:"id"`
	UnidadeID            *uuid.UUID `json:"unidade_id,omitempty"`
	Ordem                int        `json:"ordem"`
	Nome                 string     `json:"nome"`
	OfertaID             *uuid.UUID `json:"oferta_id,omitempty"`
	Palavra              string     `json:"palavra,omitempty"`
	UnidadeSolicitanteID *uuid.UUID `json:"unidade_solicitante_id,omitempty"`
	Prioridade           string     `json:"prioridade,omitempty"`
	EquipeID             *uuid.UUID `json:"equipe_id,omitempty"`
	Ativa                bool       `json:"ativa"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Casa diz se esta regra pega o chamado. Unidades são as lotações de quem pede.
func (r Regra) Casa(oferta uuid.UUID, titulo string, unidades []uuid.UUID) bool {
	if !r.Ativa {
		return false
	}
	if r.OfertaID != nil && *r.OfertaID != oferta {
		return false
	}
	if r.Palavra != "" && !strings.Contains(strings.ToLower(titulo), r.Palavra) {
		return false
	}
	if r.UnidadeSolicitanteID != nil && !slices.Contains(unidades, *r.UnidadeSolicitanteID) {
		return false
	}
	return true
}

// Primeira devolve a regra de menor ordem que casa. Nenhuma é ausência, não erro.
func Primeira(regras []Regra, oferta uuid.UUID, titulo string, unidades []uuid.UUID) *Regra {
	for i := range regras {
		if regras[i].Casa(oferta, titulo, unidades) {
			return &regras[i]
		}
	}
	return nil
}

// NovaRegra valida a triagem. A ação é prioridade, equipe, ou as duas.
func NovaRegra(in Regra) (Regra, error) {
	in.Nome = strings.TrimSpace(in.Nome)
	in.Palavra = strings.ToLower(strings.TrimSpace(in.Palavra))
	in.Prioridade = strings.TrimSpace(in.Prioridade)
	if in.Nome == "" || utf8.RuneCountInString(in.Nome) > NomeMax {
		return Regra{}, Recusa{"informe o nome da regra, com até 160 caracteres"}
	}
	if in.Ordem < 0 || in.Ordem > 999 {
		return Regra{}, Recusa{"a ordem vai de 0 a 999"}
	}
	if utf8.RuneCountInString(in.Palavra) > 80 {
		return Regra{}, Recusa{"a palavra cabe em 80 caracteres"}
	}
	if in.Prioridade != "" && !slices.Contains(Prioridades, in.Prioridade) {
		return Regra{}, Recusa{"a prioridade é critica, alta, media ou baixa"}
	}
	if in.Prioridade == "" && in.EquipeID == nil {
		return Regra{}, Recusa{"a regra precisa de uma prioridade ou de uma equipe"}
	}
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	agora := time.Now().UTC()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = agora
	}
	in.UpdatedAt = agora
	in.Ativa = true
	return in, nil
}

// RespostaPronta é o texto que a equipe cola. Não é enviada sozinha.
type RespostaPronta struct {
	OfertaID uuid.UUID `json:"oferta_id"`
	Publico  string    `json:"publico"`
	Interna  string    `json:"interna"`
}

// NovaResposta valida os dois textos. Os dois vazios não são resposta.
func NovaResposta(oferta uuid.UUID, publico, interna string) (RespostaPronta, error) {
	publico, interna = strings.TrimSpace(publico), strings.TrimSpace(interna)
	if oferta == uuid.Nil {
		return RespostaPronta{}, Recusa{"informe a oferta"}
	}
	if utf8.RuneCountInString(publico) > TextoMax || utf8.RuneCountInString(interna) > TextoMax {
		return RespostaPronta{}, Recusa{"a resposta cabe em 4000 caracteres"}
	}
	if publico == "" && interna == "" {
		return RespostaPronta{}, Recusa{"escreva a resposta pública ou a nota interna"}
	}
	return RespostaPronta{OfertaID: oferta, Publico: publico, Interna: interna}, nil
}

// ChamadoMensal abre um chamado no dia do mês, uma vez. O dia para em 28.
// Recorrencia continua sendo o grupo de incidentes da mesma oferta.
type ChamadoMensal struct {
	ID        uuid.UUID  `json:"id"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
	OfertaID  uuid.UUID  `json:"oferta_id"`
	Titulo    string     `json:"titulo"`
	Descricao string     `json:"descricao,omitempty"`
	Dia       int        `json:"dia"`
	CriadoPor uuid.UUID  `json:"criado_por"`
	Ativa     bool       `json:"ativa"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NovoChamadoMensal valida o chamado que se repete.
func NovoChamadoMensal(in ChamadoMensal) (ChamadoMensal, error) {
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao = strings.TrimSpace(in.Descricao)
	if in.OfertaID == uuid.Nil {
		return ChamadoMensal{}, Recusa{"informe a oferta"}
	}
	if in.Titulo == "" || utf8.RuneCountInString(in.Titulo) > NomeMax || utf8.RuneCountInString(in.Descricao) > TextoMax {
		return ChamadoMensal{}, Recusa{"informe o título, com até 160 caracteres; a descrição cabe em 4000"}
	}
	if in.Dia < 1 || in.Dia > 28 {
		return ChamadoMensal{}, Recusa{"o dia do mês vai de 1 a 28"}
	}
	if in.CriadoPor == uuid.Nil {
		return ChamadoMensal{}, Recusa{"informe quem cria a recorrência"}
	}
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	agora := time.Now().UTC()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = agora
	}
	in.UpdatedAt = agora
	in.Ativa = true
	return in, nil
}

// Deflexao é quantas pessoas viram artigo e não abriram pedido.
// Sem vista, o ponteiro fica nulo: zero não é amostra.
type Deflexao struct {
	Viram     int     `json:"viram"`
	SemPedido int     `json:"sem_pedido"`
	Taxa      float64 `json:"taxa"`
}

// ResumirDeflexao devolve nil quando ninguém viu artigo.
func ResumirDeflexao(viram, semPedido int) *Deflexao {
	if viram <= 0 {
		return nil
	}
	if semPedido < 0 {
		semPedido = 0
	}
	if semPedido > viram {
		semPedido = viram
	}
	taxa := math.Round(float64(semPedido)/float64(viram)*1000) / 10
	return &Deflexao{Viram: viram, SemPedido: semPedido, Taxa: taxa}
}
