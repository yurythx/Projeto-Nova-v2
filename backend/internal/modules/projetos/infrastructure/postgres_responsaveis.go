package infrastructure

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Quem pode gerir projetos na entidade da unidade, ou em escopo global.
// A lotação do diretório não entra: muita conta de gestão não tem unidade gravada.
const ondePodeGerir = `
	u.active
	AND u.username NOT LIKE 'anon\_%'
	AND (
		'nova-admin' = ANY (u.roles)
		OR EXISTS (
			SELECT 1
			FROM user_scopes s
			JOIN perfis pf ON pf.id = s.perfil_id AND pf.ativo
			WHERE s.user_id = u.id
			  AND ('projetos:manage' = ANY (pf.permissoes) OR '*' = ANY (pf.permissoes))
			  AND nova_scope_active(s.entidade_id, s.unidade_id, s.departamento_id)
			  AND (
			    (s.entidade_id IS NULL AND s.unidade_id IS NULL AND s.departamento_id IS NULL)
			    OR s.unidade_id = $UNIDADE
			    OR s.entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE)
			    OR s.unidade_id IN (SELECT id FROM unidades WHERE entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE))
			    OR s.departamento_id IN (
			      SELECT d.id FROM departamentos d
			      JOIN unidades un ON un.id = d.unidade_id
			      WHERE un.id = $UNIDADE OR un.entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE)
			    )
			  )
		)
		OR EXISTS (
			SELECT 1
			FROM ad_group_mappings m
			JOIN perfis pf ON pf.id = m.perfil_id AND pf.ativo
			WHERE lower(m.ad_group) = ANY (SELECT lower(g) FROM unnest(u.groups) g)
			  AND ('projetos:manage' = ANY (pf.permissoes) OR '*' = ANY (pf.permissoes))
			  AND nova_scope_active(m.entidade_id, m.unidade_id, m.departamento_id)
			  AND (
			    (m.entidade_id IS NULL AND m.unidade_id IS NULL AND m.departamento_id IS NULL)
			    OR m.unidade_id = $UNIDADE
			    OR m.entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE)
			    OR m.unidade_id IN (SELECT id FROM unidades WHERE entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE))
			    OR m.departamento_id IN (
			      SELECT d.id FROM departamentos d
			      JOIN unidades un ON un.id = d.unidade_id
			      WHERE un.id = $UNIDADE OR un.entidade_id = (SELECT entidade_id FROM unidades WHERE id = $UNIDADE)
			    )
			  )
		)
	)`

func sqlPodeGerir(arg string) string {
	return strings.ReplaceAll(ondePodeGerir, "$UNIDADE", arg)
}

func (r *Repository) ListarResponsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Responsavel, error) {
	q := `SELECT u.id, COALESCE(NULLIF(u.display_name, ''), u.username)
		FROM users u
		WHERE ` + sqlPodeGerir("$1") + `
		ORDER BY lower(COALESCE(NULLIF(u.display_name, ''), u.username))`
	rows, err := db.Query(ctx, q, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	lista := []domain.Responsavel{}
	for rows.Next() {
		var item domain.Responsavel
		if err := rows.Scan(&item.ID, &item.Nome); err != nil {
			return nil, mapDB(err)
		}
		lista = append(lista, item)
	}
	return lista, mapDB(rows.Err())
}

func (r *Repository) NomeDeResponsavel(ctx context.Context, db database.DBTX, pessoa uuid.UUID, unidade *uuid.UUID) (string, error) {
	q := `
		SELECT COALESCE(NULLIF(u.display_name, ''), u.username)
		FROM users u
		WHERE u.id = $1 AND ` + sqlPodeGerir("$2")
	var nome string
	err := db.QueryRow(ctx, q, pessoa, unidade).Scan(&nome)
	if err != nil {
		if database.IsNoRows(err) {
			return "", domain.ErrPessoaInvalida
		}
		return "", mapDB(err)
	}
	return nome, nil
}
