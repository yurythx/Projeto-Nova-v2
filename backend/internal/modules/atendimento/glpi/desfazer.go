package glpi

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Desfeito é o relatório do desfazer da carga (ADR 079). Na simulação os
// números são os mesmos da execução: o trabalho é feito e a transação,
// desfeita no fim.
type Desfeito struct {
	Gravou     bool `json:"gravou"`
	Chamados   int  `json:"chamados"`
	Ofertas    int  `json:"ofertas"`
	Categorias int  `json:"categorias"`
	Equipes    int  `json:"equipes"`
	Politicas  int  `json:"politicas"`
	Feriados   int  `json:"feriados"`
	// Mantidos são os chamados importados que alguém mexeu aqui depois da
	// carga; Exemplos traz até MaxExemplos números deles.
	Mantidos         int     `json:"chamados_mantidos"`
	Exemplos         []int64 `json:"exemplos_mantidos"`
	OfertasMantidas  int     `json:"ofertas_mantidas"`
	CadastrosMantido int     `json:"cadastros_mantidos"`
}

// MaxExemplos limita os números de chamados mantidos no relatório.
const MaxExemplos = 20

// errSimulacao desfaz a transação da simulação.
var errSimulacao = errors.New("glpi: simulação do desfazer")

// mexidos marca os chamados importados que receberam alguma ação aqui
// depois da carga. O registro da importação nasce na transação do
// chamado, e o que a carga e a sincronização (ADR 081) gravam leva a data
// do GLPI: o que é posterior à última passagem delas foi feito aqui. Na
// dúvida o chamado fica, que é o lado seguro.
const mexidos = `
CREATE TEMP TABLE desfazer_chamados ON COMMIT DROP AS
SELECT i.destino AS id, i.em, s.numero,
       (EXISTS (SELECT 1 FROM audit_logs l WHERE l.resource_type = 'solicitacao' AND l.resource_id = i.destino::text AND l.created_at > i.created_at)
     OR EXISTS (SELECT 1 FROM atendimento_comentarios x WHERE x.solicitacao_id = i.destino AND x.created_at > i.created_at)
     OR EXISTS (SELECT 1 FROM atendimento_passagens x WHERE x.solicitacao_id = i.destino AND x.em > i.created_at)
     OR EXISTS (SELECT 1 FROM atendimento_avaliacoes x WHERE x.solicitacao_id = i.destino AND x.created_at > i.created_at)
     OR EXISTS (SELECT 1 FROM atendimento_atribuicoes x WHERE x.solicitacao_id = i.destino AND x.inicio > i.created_at)
     OR EXISTS (SELECT 1 FROM atendimento_apontamentos x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM atendimento_tarefas x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM atendimento_aprovacoes x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM atendimento_visitas x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM atendimento_ia_sugestoes x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM correio_recebidos x WHERE x.solicitacao_id = i.destino)
     OR EXISTS (SELECT 1 FROM ocorrencias x WHERE x.chamado_id = i.destino)
     OR EXISTS (SELECT 1 FROM problemas_incidentes x WHERE x.solicitacao_id = i.destino)) AS mexido
  FROM (SELECT destino, GREATEST(created_at, COALESCE(sincronizado_em, created_at)) AS created_at,
               GREATEST(created_at, COALESCE(sincronizado_em, created_at)) AS em
          FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao') i
  JOIN atendimento_solicitacoes s ON s.id = i.destino`

// cadastros são os cadastros que a carga criou, na ordem de remoção. Um
// cadastro fica quando alguém o alterou aqui depois da carga ou quando o
// que fica ainda o usa.
var cadastros = []struct {
	tipo, tabela, recurso, emUso string
}{
	{"oferta", "atendimento_ofertas", "oferta",
		`EXISTS (SELECT 1 FROM atendimento_solicitacoes s WHERE s.oferta_id = c.id)`},
	{"categoria", "atendimento_categorias", "categoria",
		`EXISTS (SELECT 1 FROM atendimento_ofertas o WHERE o.categoria_id = c.id)`},
	{"equipe", "atendimento_equipes", "equipe",
		`EXISTS (SELECT 1 FROM atendimento_regras x WHERE x.equipe_id = c.id AND x.nome NOT LIKE 'GLPI: %')
		 OR EXISTS (SELECT 1 FROM atendimento_oferta_tarefas x WHERE x.equipe_id = c.id)
		 OR EXISTS (SELECT 1 FROM atendimento_tarefas x WHERE x.equipe_id = c.id)
		 OR EXISTS (SELECT 1 FROM atendimento_atribuicoes x WHERE x.equipe_id = c.id AND x.inicio > k.em)`},
	{"politica", "atendimento_politicas", "politica", `false`},
	{"feriado", "atendimento_feriados", "feriado", `false`},
}

// Desfazer remove o que a carga do GLPI trouxe e ninguém mexeu aqui
// depois: os chamados (com o histórico, os comentários e as avaliações)
// e os cadastros que a carga criou. gravar falso só simula.
func Desfazer(ctx context.Context, pool *pgxpool.Pool, gravar bool, progresso func(feitos, total int)) (Desfeito, error) {
	r := Desfeito{Gravou: gravar, Exemplos: []int64{}}
	total := 2 + len(cadastros)
	if progresso == nil {
		progresso = func(int, int) {}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return r, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	// Uma carga ao mesmo tempo gravaria no registro que está sendo lido.
	if _, err := tx.Exec(ctx, `LOCK TABLE atendimento_importacao IN EXCLUSIVE MODE`); err != nil {
		return r, err
	}
	if _, err := tx.Exec(ctx, mexidos); err != nil {
		return r, err
	}
	progresso(1, total)
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE mexido), count(*) FILTER (WHERE NOT mexido) FROM desfazer_chamados`).
		Scan(&r.Mantidos, &r.Chamados); err != nil {
		return r, err
	}
	rows, err := tx.Query(ctx, `SELECT numero FROM desfazer_chamados WHERE mexido ORDER BY numero LIMIT $1`, MaxExemplos)
	if err != nil {
		return r, err
	}
	if r.Exemplos, err = pgx.CollectRows(rows, pgx.RowTo[int64]); err != nil {
		return r, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM atendimento_solicitacoes s USING desfazer_chamados d WHERE s.id = d.id AND NOT d.mexido`); err != nil {
		return r, fmt.Errorf("chamados: %w", err)
	}
	progresso(2, total)
	contas := []*int{&r.Ofertas, &r.Categorias, &r.Equipes, &r.Politicas, &r.Feriados}
	for i, c := range cadastros {
		removidos, mantidos, err := desfazerCadastro(ctx, tx, c.tipo, c.tabela, c.recurso, c.emUso)
		if err != nil {
			return r, fmt.Errorf("%s: %w", c.tipo, err)
		}
		*contas[i] = removidos
		if c.tipo == "oferta" {
			r.OfertasMantidas = mantidos
		} else {
			r.CadastrosMantido += mantidos
		}
		progresso(3+i, total)
	}
	// Sai do registro o que não existe mais: uma carga nova traz de novo.
	if _, err := tx.Exec(ctx, `
		DELETE FROM atendimento_importacao i WHERE i.origem = 'glpi' AND NOT CASE i.tipo
			WHEN 'solicitacao' THEN EXISTS (SELECT 1 FROM atendimento_solicitacoes x WHERE x.id = i.destino)
			WHEN 'oferta' THEN EXISTS (SELECT 1 FROM atendimento_ofertas x WHERE x.id = i.destino)
			WHEN 'categoria' THEN EXISTS (SELECT 1 FROM atendimento_categorias x WHERE x.id = i.destino)
			WHEN 'equipe' THEN EXISTS (SELECT 1 FROM atendimento_equipes x WHERE x.id = i.destino)
			WHEN 'politica' THEN EXISTS (SELECT 1 FROM atendimento_politicas x WHERE x.id = i.destino)
			WHEN 'feriado' THEN EXISTS (SELECT 1 FROM atendimento_feriados x WHERE x.id = i.destino)
			WHEN 'acompanhamento' THEN EXISTS (SELECT 1 FROM atendimento_comentarios x WHERE x.id = i.destino)
			ELSE true END`); err != nil {
		return r, fmt.Errorf("registro: %w", err)
	}
	if !gravar {
		return r, nil
	}
	return r, tx.Commit(ctx)
}

// desfazerCadastro remove os cadastros de um tipo que a carga criou, e
// devolve quantos saíram e quantos ficaram.
func desfazerCadastro(ctx context.Context, tx pgx.Tx, tipo, tabela, recurso, emUso string) (int, int, error) {
	// O mesmo destino pode estar em várias linhas do registro: vale a
	// primeira, a da criação.
	q := fmt.Sprintf(`
		WITH criados AS (
			SELECT destino AS id, min(created_at) AS em FROM atendimento_importacao
			 WHERE origem = 'glpi' AND tipo = $1 AND criado GROUP BY destino
		), alvo AS (
			SELECT c.id, (EXISTS (SELECT 1 FROM audit_logs l WHERE l.resource_type = $2 AND l.resource_id = c.id::text AND l.created_at > k.em)
			              OR %s) AS fica
			  FROM criados k JOIN %s c ON c.id = k.id
		), removidos AS (
			DELETE FROM %s t USING alvo a WHERE t.id = a.id AND NOT a.fica RETURNING t.id
		)
		SELECT (SELECT count(*) FROM removidos), (SELECT count(*) FROM alvo WHERE fica)`, emUso, tabela, tabela)
	var removidos, mantidos int
	err := tx.QueryRow(ctx, q, tipo, recurso).Scan(&removidos, &mantidos)
	return removidos, mantidos, err
}
