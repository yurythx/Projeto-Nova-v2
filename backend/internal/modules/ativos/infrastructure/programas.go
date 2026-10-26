package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Contagem é o que o agente viu. Parcial significa que alguma máquina
// cortou a lista em 100, então a conta pode estar menor.
type Contagem struct {
	Instalados    int
	ComInventario int
	Parcial       bool
}

// ContarPrograma conta ativos em uso ou manutenção cujo inventário traz o nome.
func ContarPrograma(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, nome string) (Contagem, error) {
	if unidades == nil {
		unidades = []uuid.UUID{}
	}
	var c Contagem
	err := db.QueryRow(ctx, `SELECT
			COUNT(*) FILTER (WHERE inventario IS NOT NULL)::int,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1
				FROM jsonb_array_elements(
					CASE WHEN jsonb_typeof(inventario->'programas') = 'array'
					THEN inventario->'programas' ELSE '[]'::jsonb END
				) elem
				WHERE lower(elem->>'nome') = lower($1)
			))::int,
			COALESCE(bool_or(COALESCE((inventario->>'programas_truncado')::boolean, false)), false)
		FROM ativos
		WHERE estado IN ('em_uso', 'manutencao')
		  AND ($2 = false OR unidade_id = ANY($3::uuid[]))`, nome, restrito, unidades).
		Scan(&c.ComInventario, &c.Instalados, &c.Parcial)
	return c, err
}
