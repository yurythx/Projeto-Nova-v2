package infrastructure

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) ListarPastas(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Pasta, error) {
	rows, err := db.Query(ctx, `SELECT p.id, p.nome, p.unidade_id, p.created_at, p.updated_at
		FROM cofre_pastas p
		WHERE (
			($1 = true AND ($2 = false OR p.unidade_id = ANY($3::uuid[])))
			OR p.unidade_id = ANY($4::uuid[])
			OR EXISTS (SELECT 1 FROM cofre_pasta_membros m WHERE m.pasta_id = p.id AND m.usuario_id = $5)
		)
		ORDER BY p.nome, p.id`, f.GereTudo, f.Restrito, idsOrEmpty(f.Unidades), idsOrEmpty(f.GereUnidades), f.Usuario)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Pasta{}
	for rows.Next() {
		var p domain.Pasta
		if err := rows.Scan(&p.ID, &p.Nome, &p.UnidadeID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDB(err)
	}
	for i := range out {
		membros, err := r.membros(ctx, db, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Membros = membros
	}
	return out, nil
}

func (r *Repository) membros(ctx context.Context, db database.DBTX, pasta uuid.UUID) ([]domain.Membro, error) {
	rows, err := db.Query(ctx, `SELECT usuario_id, pode_revelar FROM cofre_pasta_membros WHERE pasta_id = $1 ORDER BY usuario_id`, pasta)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Membro{}
	for rows.Next() {
		var m domain.Membro
		if err := rows.Scan(&m.UsuarioID, &m.PodeRevelar); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, m)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ObterPasta(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Pasta, error) {
	var p domain.Pasta
	err := db.QueryRow(ctx, `SELECT id, nome, unidade_id, created_at, updated_at FROM cofre_pastas WHERE id = $1`, id).
		Scan(&p.ID, &p.Nome, &p.UnidadeID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Pasta{}, mapDB(err)
	}
	p.Membros, err = r.membros(ctx, db, id)
	return p, err
}

func (r *Repository) InserirPasta(ctx context.Context, db database.DBTX, p domain.Pasta) error {
	_, err := db.Exec(ctx, `INSERT INTO cofre_pastas (id, nome, unidade_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)`,
		p.ID, p.Nome, p.UnidadeID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return mapDB(err)
	}
	return r.SubstituirMembros(ctx, db, p.ID, p.Membros)
}

func (r *Repository) ContarNaPasta(ctx context.Context, db database.DBTX, pasta uuid.UUID) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM cofre_segredos WHERE pasta_id = $1`, pasta).Scan(&n)
	return n, err
}

func (r *Repository) ExcluirPasta(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM cofre_pastas WHERE id = $1`, id)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) SubstituirMembros(ctx context.Context, db database.DBTX, pasta uuid.UUID, membros []domain.Membro) error {
	if _, err := db.Exec(ctx, `DELETE FROM cofre_pasta_membros WHERE pasta_id = $1`, pasta); err != nil {
		return mapDB(err)
	}
	for _, m := range membros {
		if _, err := db.Exec(ctx, `INSERT INTO cofre_pasta_membros (pasta_id, usuario_id, pode_revelar) VALUES ($1,$2,$3)`,
			pasta, m.UsuarioID, m.PodeRevelar); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

func (r *Repository) Membro(ctx context.Context, db database.DBTX, pasta, usuario uuid.UUID) (domain.Membro, error) {
	var m domain.Membro
	err := db.QueryRow(ctx, `SELECT usuario_id, pode_revelar FROM cofre_pasta_membros WHERE pasta_id = $1 AND usuario_id = $2`, pasta, usuario).
		Scan(&m.UsuarioID, &m.PodeRevelar)
	return m, mapDB(err)
}

func (r *Repository) UsuariosExistem(ctx context.Context, db database.DBTX, ids []uuid.UUID) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM users WHERE id = ANY($1::uuid[])`, ids).Scan(&n)
	return n == len(ids), mapDB(err)
}

func (r *Repository) Pessoas(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]domain.Pessoa, error) {
	rows, err := db.Query(ctx, `SELECT u.id, COALESCE(NULLIF(btrim(u.display_name), ''), u.username)
		FROM users u
		WHERE u.active
		  AND ($1::uuid IS NULL OR EXISTS (SELECT 1 FROM user_scopes s WHERE s.user_id = u.id AND s.unidade_id = $1))
		ORDER BY 2, u.id
		LIMIT 100`, unidade)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Pessoa{}
	for rows.Next() {
		var p domain.Pessoa
		if err := rows.Scan(&p.ID, &p.Nome); err != nil {
			return nil, mapDB(err)
		}
		out = append(out, p)
	}
	return out, mapDB(rows.Err())
}
