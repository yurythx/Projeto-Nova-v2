package glpi

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Calendario é um calendário de expediente do GLPI, com os turnos e os
// feriados ligados a ele. É nele que os SLAs contam o tempo útil.
type Calendario struct {
	ID       int       `json:"id"`
	Nome     string    `json:"name"`
	Turnos   []Turno   `json:"turnos"`
	Feriados []Feriado `json:"feriados"`
}

// Turno é um trecho de expediente (CalendarSegment); Dia 0 é domingo.
type Turno struct {
	Calendario int    `json:"calendars_id"`
	Dia        int    `json:"day"`
	Inicio     string `json:"begin"`
	Fim        string `json:"end"`
}

// Feriado é um dia sem expediente; perpétuo repete todo ano.
type Feriado struct {
	ID       int    `json:"id"`
	Nome     string `json:"name"`
	Inicio   string `json:"begin_date"`
	Fim      string `json:"end_date"`
	Perpetuo int    `json:"is_perpetual"`
}

// Grupo é um grupo técnico do GLPI e as pessoas dele.
type Grupo struct {
	ID         int    `json:"id"`
	Nome       string `json:"completename"`
	EntidadeID int    `json:"entities_id"`
	Membros    []int  `json:"membros"`
	// Chamados é quantos chamados foram atribuídos ao grupo: o GLPI traz
	// do AD centenas de grupos, e só os que atendem viram equipe.
	Chamados int `json:"chamados"`
}

// Expediente lê o calendário e os grupos técnicos (ADR 078): é com eles
// que a carga monta a política de prazos, os feriados e as equipes.
type Expediente interface {
	Calendarios(ctx context.Context) ([]Calendario, error)
	Grupos(ctx context.Context) ([]Grupo, error)
}

// Calendarios lê os calendários com os turnos e os feriados de cada um.
func (c *Cliente) Calendarios(ctx context.Context) ([]Calendario, error) {
	var cals []Calendario
	var turnos []Turno
	var feriados []Feriado
	var ligas []struct {
		Calendario int `json:"calendars_id"`
		Feriado    int `json:"holidays_id"`
	}
	for _, l := range []struct {
		caminho string
		out     any
	}{
		{"/Calendar?range=0-999", &cals}, {"/CalendarSegment?range=0-9999", &turnos},
		{"/Holiday?range=0-9999", &feriados}, {"/Calendar_Holiday?range=0-9999", &ligas},
	} {
		if _, err := c.get(ctx, l.caminho, l.out); err != nil {
			return nil, err
		}
	}
	porID := map[int]Feriado{}
	for _, f := range feriados {
		porID[f.ID] = f
	}
	for i := range cals {
		cals[i].Turnos, cals[i].Feriados = []Turno{}, []Feriado{}
		for _, t := range turnos {
			if t.Calendario == cals[i].ID {
				cals[i].Turnos = append(cals[i].Turnos, t)
			}
		}
		for _, l := range ligas {
			if f, ok := porID[l.Feriado]; ok && l.Calendario == cals[i].ID {
				cals[i].Feriados = append(cals[i].Feriados, f)
			}
		}
	}
	return cals, nil
}

// Grupos lê os grupos técnicos e quem está em cada um.
func (c *Cliente) Grupos(ctx context.Context) ([]Grupo, error) {
	var grupos []Grupo
	var membros []struct {
		Grupo  int `json:"groups_id"`
		Pessoa int `json:"users_id"`
	}
	if _, err := c.get(ctx, "/Group?range=0-9999", &grupos); err != nil {
		return nil, err
	}
	if _, err := c.get(ctx, "/Group_User?range=0-99999", &membros); err != nil {
		return nil, err
	}
	// Tipo 2 é o grupo atribuído ao chamado (1 é o requerente, 3 observa).
	var ligas []struct {
		Grupo int `json:"groups_id"`
		Tipo  int `json:"type"`
	}
	if _, err := c.get(ctx, "/Group_Ticket?range=0-199999", &ligas); err != nil {
		return nil, err
	}
	atendidos := map[int]int{}
	for _, l := range ligas {
		if l.Tipo == 2 {
			atendidos[l.Grupo]++
		}
	}
	for i := range grupos {
		grupos[i].Chamados = atendidos[grupos[i].ID]
		grupos[i].Membros = []int{}
		for _, m := range membros {
			if m.Grupo == grupos[i].ID {
				grupos[i].Membros = append(grupos[i].Membros, m.Pessoa)
			}
		}
	}
	return grupos, nil
}

// RelatorioExpediente é o que a carga fez com o calendário e os grupos.
type RelatorioExpediente struct {
	// Politica: "criada", "existente" (não se sobrescreve a configurada) ou
	// "sem calendário" (o GLPI não deu turnos aos SLAs).
	Politica      string   `json:"politica"`
	Calendario    string   `json:"calendario,omitempty"`
	Metas         []string `json:"metas,omitempty"`
	Feriados      int      `json:"feriados"`
	Equipes       int      `json:"equipes"`
	Membros       int      `json:"membros"`
	MembrosSemPar int      `json:"membros_sem_par"`
	SemUnidade    []string `json:"grupos_sem_unidade,omitempty"`
	// GruposSemChamado são os grupos que nunca atenderam: ficam de fora.
	GruposSemChamado int `json:"grupos_sem_chamado"`
}

// matrizUsual é a matriz de impacto x urgência da tela (a soma dos dois).
func matrizUsual() []domain.Celula {
	out := []domain.Celula{}
	for impacto := 1; impacto <= 3; impacto++ {
		for urgencia := 1; urgencia <= 3; urgencia++ {
			p := domain.PrioridadeMedia
			switch soma := impacto + urgencia; {
			case soma == 6:
				p = domain.PrioridadeCritica
			case soma >= 5:
				p = domain.PrioridadeAlta
			case soma <= 2:
				p = domain.PrioridadeBaixa
			}
			out = append(out, domain.Celula{Impacto: impacto, Urgencia: urgencia, Prioridade: p})
		}
	}
	return out
}

// metasDosSLAs junta cada par do GLPI (o SLM: resposta e solução) e dá as
// prioridades pela solução, da mais curta para a mais longa: alta, média,
// baixa. A crítica, que o GLPI não tem, fica com metade da alta.
func metasDosSLAs(slas []SLA) []domain.Meta {
	pares := map[int]*domain.Meta{}
	for _, s := range slas {
		min, ok := MetaMin(s.Tempo, s.Unidade)
		if !ok || s.SLM <= 0 {
			continue
		}
		m := pares[s.SLM]
		if m == nil {
			m = &domain.Meta{}
			pares[s.SLM] = m
		}
		if s.Tipo == 1 {
			m.RespostaMin = min
		} else {
			m.SolucaoMin = min
		}
	}
	lista := []domain.Meta{}
	for _, m := range pares {
		if m.SolucaoMin > 0 {
			m.RespostaMin = min(cmp.Or(m.RespostaMin, m.SolucaoMin), m.SolucaoMin)
			lista = append(lista, *m)
		}
	}
	slices.SortFunc(lista, func(a, b domain.Meta) int { return cmp.Or(a.SolucaoMin-b.SolucaoMin, a.RespostaMin-b.RespostaMin) })
	if len(lista) == 0 {
		return nil
	}
	ordem := []string{domain.PrioridadeAlta, domain.PrioridadeMedia, domain.PrioridadeBaixa}
	out := []domain.Meta{}
	for i, p := range ordem {
		m := lista[min(i, len(lista)-1)]
		m.Prioridade = p
		out = append(out, m)
	}
	alta := out[0]
	return append([]domain.Meta{{Prioridade: domain.PrioridadeCritica, RespostaMin: max(alta.RespostaMin/2, 15),
		SolucaoMin: max(alta.SolucaoMin/2, 60)}}, out...)
}

// calendarioDosSLAs é o calendário que mais SLAs usam.
func calendarioDosSLAs(slas []SLA, cals []Calendario) (Calendario, bool) {
	uso := map[int]int{}
	for _, s := range slas {
		if s.Calendario > 0 {
			uso[s.Calendario]++
		}
	}
	var escolhido Calendario
	melhor := 0
	for _, c := range cals {
		if uso[c.ID] > melhor && len(c.Turnos) > 0 {
			escolhido, melhor = c, uso[c.ID]
		}
	}
	return escolhido, melhor > 0
}

// tipoDoFeriado separa os nacionais e os pontos facultativos pelo nome; o
// resto é municipal (o GLPI não guarda o tipo).
func tipoDoFeriado(nome string) string {
	n := strings.ToLower(nome)
	for _, p := range []string{"carnaval", "cinzas", "corpus christi"} {
		if strings.Contains(n, p) {
			return "ponto_facultativo"
		}
	}
	for _, p := range []string{"natal", "tiradentes", "independ", "confraterniza", "ano novo", "trabalh", "aparecida", "finados",
		"proclama", "consciência negra", "consciencia negra", "paixão", "paixao", "sexta-feira santa", "sexta feira santa"} {
		if strings.Contains(n, p) {
			return "nacional"
		}
	}
	return "municipal"
}

// datasDoFeriado são os dias do feriado, deste ano em diante: o perpétuo
// vale neste ano e no próximo.
func datasDoFeriado(f Feriado, hoje time.Time) []string {
	ini, err1 := time.Parse(time.DateOnly, strings.TrimSpace(f.Inicio))
	fim, err2 := time.Parse(time.DateOnly, strings.TrimSpace(f.Fim))
	if err1 != nil {
		return nil
	}
	if err2 != nil || fim.Before(ini) {
		fim = ini
	}
	var out []string
	add := func(de, ate time.Time) {
		for d := de; !d.After(ate) && len(out) < 60; d = d.AddDate(0, 0, 1) {
			if d.Year() >= hoje.Year() {
				out = append(out, d.Format(time.DateOnly))
			}
		}
	}
	if f.Perpetuo == 1 {
		for _, ano := range []int{hoje.Year(), hoje.Year() + 1} {
			delta := ano - ini.Year()
			add(ini.AddDate(delta, 0, 0), fim.AddDate(delta, 0, 0))
		}
		return out
	}
	add(ini, fim)
	return out
}

func hhmm(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

// aplicarExpediente monta, a partir do GLPI, a política institucional de
// prazos (sem sobrescrever a configurada), os feriados e as equipes das
// unidades. Na simulação, só conta. Cada parte é idempotente.
func (im *Importador) aplicarExpediente(ctx context.Context, slas []SLA, rel *Relatorio) error {
	exp, ok := im.Fonte.(Expediente)
	if !ok {
		return nil
	}
	r := &RelatorioExpediente{SemUnidade: []string{}}
	rel.Expediente = r
	cals, err := exp.Calendarios(ctx)
	if err != nil {
		return fmt.Errorf("calendários: %w", err)
	}
	cal, achou := calendarioDosSLAs(slas, cals)
	metas := metasDosSLAs(slas)
	switch {
	case !achou || len(metas) == 0:
		r.Politica = "sem calendário"
	default:
		r.Calendario = cal.Nome
		for _, m := range metas {
			r.Metas = append(r.Metas, fmt.Sprintf("%s: resposta %d min, solução %d min", m.Prioridade, m.RespostaMin, m.SolucaoMin))
		}
		if err := im.politica(ctx, cal, metas, r); err != nil {
			return err
		}
	}
	grupos, err := exp.Grupos(ctx)
	if err != nil {
		return fmt.Errorf("grupos: %w", err)
	}
	return im.equipes(ctx, grupos, r)
}

func (im *Importador) politica(ctx context.Context, cal Calendario, metas []domain.Meta, r *RelatorioExpediente) error {
	_, err := im.Repo.GetPolitica(ctx, im.Pool, nil)
	switch {
	case err == nil:
		r.Politica = "existente"
	case !errors.Is(err, domain.ErrNaoEncontrado):
		return err
	default:
		r.Politica = "criada"
	}
	turnos := []domain.Turno{}
	for _, t := range cal.Turnos {
		turnos = append(turnos, domain.Turno{Dia: t.Dia, Inicio: hhmm(t.Inicio), Fim: hhmm(t.Fim)})
	}
	p, err := domain.NovaPolitica(nil, cmp.Or(im.Fuso.String(), "America/Cuiaba"), turnos, matrizUsual(), metas)
	if err != nil {
		return fmt.Errorf("política do calendário %q: %w", cal.Nome, err)
	}
	existentes, err := im.Repo.ListFeriados(ctx, im.Pool, nil)
	if err != nil {
		return err
	}
	vistos := map[string]bool{}
	for _, f := range existentes {
		vistos[f.Data] = true
	}
	novos := []domain.Feriado{}
	for _, f := range cal.Feriados {
		for _, d := range datasDoFeriado(f, time.Now().In(im.Fuso)) {
			if vistos[d] {
				continue
			}
			vistos[d] = true
			n, err := domain.NovoFeriado(nil, d, Cortar(f.Nome, 160), tipoDoFeriado(f.Nome))
			if err != nil {
				continue
			}
			novos = append(novos, n)
		}
	}
	r.Feriados = len(novos)
	if !im.Gravar {
		return nil
	}
	return database.WithTx(ctx, im.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if r.Politica == "criada" {
			if _, err := im.Repo.SalvarPolitica(ctx, tx, p); err != nil {
				return err
			}
		}
		for _, f := range novos {
			if _, err := im.Repo.InsertFeriado(ctx, tx, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// equipes cria, na unidade da secretaria de cada grupo que já atendeu
// chamados, a equipe com as pessoas que casaram; a que já existe com o
// mesmo nome ganha quem falta.
func (im *Importador) equipes(ctx context.Context, grupos []Grupo, r *RelatorioExpediente) error {
	for _, g := range grupos {
		if g.Chamados == 0 {
			r.GruposSemChamado++
			continue
		}
		if len(g.Membros) == 0 {
			continue
		}
		sec := im.secretarias[g.EntidadeID]
		d, ok := im.Mapa.Destino(sec, CategoriaGeral)
		if !ok || d.Unidade == nil {
			r.SemUnidade = append(r.SemUnidade, g.Nome)
			continue
		}
		membros := []uuid.UUID{}
		for _, id := range g.Membros {
			p, err := im.pessoa(ctx, id)
			if err != nil {
				return err
			}
			if p == nil || p.id == nil {
				r.MembrosSemPar++
				continue
			}
			membros = append(membros, *p.id)
		}
		ativos, err := im.ativos(ctx, membros)
		if err != nil {
			return err
		}
		r.Equipes++
		r.Membros += len(ativos)
		if !im.Gravar {
			continue
		}
		if err := im.gravarEquipe(ctx, Cortar(g.Nome, domain.NomeMax), d.Unidade, ativos); err != nil {
			return fmt.Errorf("equipe %q: %w", g.Nome, err)
		}
	}
	return nil
}

// ativos são as contas ativas, em ordem (só elas assumem chamados).
func (im *Importador) ativos(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := im.Pool.Query(ctx, `SELECT id FROM users WHERE id = ANY($1) AND active ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (im *Importador) gravarEquipe(ctx context.Context, nome string, unidade *uuid.UUID, membros []uuid.UUID) error {
	return database.WithTx(ctx, im.Pool, func(ctx context.Context, tx pgx.Tx) error {
		var id uuid.UUID
		var atuais []uuid.UUID
		err := tx.QueryRow(ctx, `SELECT e.id, COALESCE((SELECT array_agg(m.user_id) FROM atendimento_equipe_membros m WHERE m.equipe_id = e.id), '{}')
			FROM atendimento_equipes e WHERE e.unidade_id = $1 AND lower(e.nome) = lower($2) LIMIT 1`, *unidade, nome).Scan(&id, &atuais)
		if database.IsNoRows(err) {
			e, err := domain.NovaEquipe(nome, unidade, membros)
			if err != nil {
				return err
			}
			if e, err = im.Repo.InsertEquipe(ctx, tx, e); err != nil {
				return err
			}
			id, err = e.ID, nil
		} else if err != nil {
			return err
		}
		for _, m := range membros {
			if !slices.Contains(atuais, m) {
				atuais = append(atuais, m)
			}
		}
		return im.Repo.SubstituirMembros(ctx, tx, id, atuais)
	})
}

// CategoriaGeral é a linha "*" da tabela: o destino padrão da secretaria.
const CategoriaGeral = -1
