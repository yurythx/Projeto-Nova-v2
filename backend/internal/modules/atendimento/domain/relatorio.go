package domain

import (
	"cmp"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Relatório por secretaria e OLA das equipes (ADR 070). A secretaria é a
// unidade de topo da árvore: o pedido de um departamento conta para a
// secretaria de que ele faz parte.

// Secretaria é a unidade de topo a que uma unidade pertence.
type Secretaria struct {
	ID    uuid.UUID
	Nome  string
	Sigla string
}

// LinhaSecretaria são os números de uma secretaria (ou do total) no período.
type LinhaSecretaria struct {
	UnidadeID     *uuid.UUID  `json:"unidade_id,omitempty"`
	Nome          string      `json:"nome"`
	Sigla         string      `json:"sigla,omitempty"`
	Total         int         `json:"total"`
	Abertas       int         `json:"abertas"`
	Concluidas    int         `json:"concluidas"`
	Canceladas    int         `json:"canceladas"`
	MTTRHoras     *float64    `json:"mttr_horas"`
	SLAPercentual *float64    `json:"sla_percentual"`
	FCRPercentual *float64    `json:"fcr_percentual"`
	Experiencia   Experiencia `json:"experiencia"`
}

// TarefaMedida é o que o OLA precisa de uma tarefa.
type TarefaMedida struct {
	EquipeID   uuid.UUID
	EquipeNome string
	OLAHoras   *int
	Estado     string
	CreatedAt  time.Time
	FeitaEm    *time.Time
}

// LinhaEquipe é o OLA de uma equipe no período: as tarefas da requisição
// e os trechos em que um chamado esteve com ela (ADR 073). A tarefa
// dispensada não conta: não houve trabalho a medir.
type LinhaEquipe struct {
	EquipeID          uuid.UUID `json:"equipe_id"`
	Nome              string    `json:"nome"`
	OLAHoras          *int      `json:"ola_horas"`
	Concluidas        int       `json:"concluidas"`
	NoPrazo           int       `json:"no_prazo"`
	NoPrazoPercentual *float64  `json:"no_prazo_percentual"`
	MedianaHoras      *float64  `json:"mediana_horas"`
	Pendentes         int       `json:"pendentes"`
	Atrasadas         int       `json:"atrasadas"`
	// Chamados são os trechos já encerrados (o chamado foi concluído ou
	// passou para outra equipe); ComAEquipe, os que ainda estão com ela.
	Chamados                  int      `json:"chamados"`
	ChamadosNoPrazo           int      `json:"chamados_no_prazo"`
	ChamadosNoPrazoPercentual *float64 `json:"chamados_no_prazo_percentual"`
	ChamadosMedianaHoras      *float64 `json:"chamados_mediana_horas"`
	ComAEquipe                int      `json:"com_a_equipe"`
	ChamadosAtrasados         int      `json:"chamados_atrasados"`
}

// LinhaDe resume um grupo de solicitações: volume, os indicadores da
// equipe e a experiência de quem pediu, com as metas da instituição.
func LinhaDe(lista []Solicitacao, passagens map[uuid.UUID][]Passagem, avaliacoes map[uuid.UUID]Avaliacao, metas MetasXLA) LinhaSecretaria {
	var l LinhaSecretaria
	var r ResumoAvaliacoes
	for _, s := range lista {
		l.Total++
		switch {
		case s.Estado == EstadoCancelado:
			l.Canceladas++
		case concluida(s):
			l.Concluidas++
		default:
			l.Abertas++
		}
		if a, ok := avaliacoes[s.ID]; ok {
			r.Total++
			r.Soma += a.Nota
			if a.Facilidade != nil {
				r.TotalFacilidade++
				r.SomaFacilidade += *a.Facilidade
			}
		}
	}
	ind := Resumir(lista, passagens)
	l.MTTRHoras, l.SLAPercentual, l.FCRPercentual = ind.MTTRHoras, ind.SLAPercentual, ind.FCRPercentual
	l.Experiencia = MedirExperiencia(lista, passagens, r, metas)
	return l
}

// PorSecretaria agrupa as solicitações pela secretaria da unidade. Sem
// unidade (ou com unidade fora da árvore), o pedido fica em "Institucional".
// As linhas vêm das que mais pediram para as que menos pediram.
func PorSecretaria(lista []Solicitacao, secretarias map[uuid.UUID]Secretaria, passagens map[uuid.UUID][]Passagem, avaliacoes map[uuid.UUID]Avaliacao, metas MetasXLA) []LinhaSecretaria {
	grupos := map[uuid.UUID][]Solicitacao{}
	for _, s := range lista {
		chave := uuid.Nil
		if s.UnidadeID != nil {
			if sec, ok := secretarias[*s.UnidadeID]; ok {
				chave = sec.ID
			}
		}
		grupos[chave] = append(grupos[chave], s)
	}
	porID := map[uuid.UUID]Secretaria{}
	for _, sec := range secretarias {
		porID[sec.ID] = sec
	}
	out := make([]LinhaSecretaria, 0, len(grupos))
	for chave, g := range grupos {
		l := LinhaDe(g, passagens, avaliacoes, metas)
		if chave == uuid.Nil {
			l.Nome = "Institucional"
		} else {
			id := chave
			l.UnidadeID, l.Nome, l.Sigla = &id, porID[chave].Nome, porID[chave].Sigla
		}
		out = append(out, l)
	}
	slices.SortFunc(out, func(a, b LinhaSecretaria) int {
		if c := cmp.Compare(b.Total, a.Total); c != 0 {
			return c
		}
		return cmp.Compare(a.Nome, b.Nome)
	})
	return out
}

// ResumirEquipes calcula o OLA de cada equipe. Pendente atrasada é a que
// passou do prazo da equipe e ainda não foi feita. No chamado, o tempo é
// o do trecho com a equipe, sem as esperas pelo servidor e pelo fornecedor.
func ResumirEquipes(tarefas []TarefaMedida, atribuicoes []Atribuicao, passagens map[uuid.UUID][]Passagem, agora time.Time) []LinhaEquipe {
	idx := map[uuid.UUID]int{}
	out := []LinhaEquipe{}
	horas := map[uuid.UUID][]float64{}
	horasChamados := map[uuid.UUID][]float64{}
	linha := func(id uuid.UUID, nome string, ola *int) *LinhaEquipe {
		i, ok := idx[id]
		if !ok {
			i = len(out)
			idx[id] = i
			out = append(out, LinhaEquipe{EquipeID: id, Nome: nome, OLAHoras: ola})
		}
		return &out[i]
	}
	for _, a := range atribuicoes {
		l := linha(a.EquipeID, a.EquipeNome, a.OLAHoras)
		tempo := TempoDaEquipe(a, passagens[a.SolicitacaoID], agora)
		fora := a.OLAHoras != nil && tempo > time.Duration(*a.OLAHoras)*time.Hour
		if a.Fim == nil {
			l.ComAEquipe++
			if fora {
				l.ChamadosAtrasados++
			}
			continue
		}
		l.Chamados++
		horasChamados[a.EquipeID] = append(horasChamados[a.EquipeID], tempo.Hours())
		if a.OLAHoras != nil && !fora {
			l.ChamadosNoPrazo++
		}
	}
	for _, t := range tarefas {
		l := linha(t.EquipeID, t.EquipeNome, t.OLAHoras)
		var limite *time.Time
		if t.OLAHoras != nil {
			v := t.CreatedAt.Add(time.Duration(*t.OLAHoras) * time.Hour)
			limite = &v
		}
		switch t.Estado {
		case TarefaConcluida:
			l.Concluidas++
			horas[t.EquipeID] = append(horas[t.EquipeID], t.FeitaEm.Sub(t.CreatedAt).Hours())
			if limite != nil && !t.FeitaEm.After(*limite) {
				l.NoPrazo++
			}
		case TarefaPendente:
			l.Pendentes++
			if limite != nil && agora.After(*limite) {
				l.Atrasadas++
			}
		}
	}
	for i := range out {
		l := &out[i]
		if l.OLAHoras != nil && l.Concluidas > 0 {
			l.NoPrazoPercentual = percentual(l.NoPrazo, l.Concluidas)
		}
		if h := horas[l.EquipeID]; len(h) > 0 {
			l.MedianaHoras = arredondar(mediana(h))
		}
		if l.OLAHoras != nil && l.Chamados > 0 {
			l.ChamadosNoPrazoPercentual = percentual(l.ChamadosNoPrazo, l.Chamados)
		}
		if h := horasChamados[l.EquipeID]; len(h) > 0 {
			l.ChamadosMedianaHoras = arredondar(mediana(h))
		}
	}
	slices.SortFunc(out, func(a, b LinhaEquipe) int { return cmp.Compare(a.Nome, b.Nome) })
	return out
}
