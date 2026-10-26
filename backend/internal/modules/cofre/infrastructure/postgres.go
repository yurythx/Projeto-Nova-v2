package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

func mapDB(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNaoEncontrado
	case database.IsForeignKeyViolation(err):
		return domain.EntradaInvalida{Msg: "a unidade não existe"}
	case database.IsCheckViolation(err):
		return domain.EntradaInvalida{Msg: "dados fora do cadastro"}
	default:
		return fmt.Errorf("cofre: %w", err)
	}
}

func idsOrEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

func clausula(inicio int) string {
	a, b, c, d, e, f := inicio, inicio+1, inicio+2, inicio+3, inicio+4, inicio+5
	return fmt.Sprintf(`($%d = false OR unidade_id = ANY($%d::uuid[])) AND (
		$%d = false OR pasta_id IS NULL OR $%d = true
		OR pasta_id IN (SELECT pasta_id FROM cofre_pasta_membros WHERE usuario_id = $%d)
		OR unidade_id = ANY($%d::uuid[]))`, a, b, c, d, e, f)
}

func filtroVals(f domain.Filtro) []any {
	return []any{f.Restrito, idsOrEmpty(f.Unidades), f.FiltrarPastas, f.GereTudo, f.Usuario, idsOrEmpty(f.GereUnidades)}
}

const colunas = `id, nome, usuario, tipo, endereco, rotaciona, rotacao_pendente, proxima_rotacao, unidade_id, pasta_id, versao, dek_embrulhada, segredo_cifrado, created_at, updated_at`

func scanSegredo(scan func(dest ...any) error) (domain.Selado, error) {
	var item domain.Selado
	err := scan(
		&item.ID, &item.Nome, &item.Usuario, &item.Tipo, &item.Endereco,
		&item.Rotaciona, &item.RotacaoPendente, &item.ProximaRotacao,
		&item.UnidadeID, &item.PastaID, &item.Versao, &item.Dek, &item.Cifra, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filtro) ([]domain.Selado, error) {
	rows, err := db.Query(ctx, `SELECT `+colunas+`
		FROM cofre_segredos WHERE `+clausula(1)+` ORDER BY nome, id`, filtroVals(f)...)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Selado{}
	for rows.Next() {
		item, err := scanSegredo(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) (domain.Selado, error) {
	args := append([]any{id}, filtroVals(f)...)
	row := db.QueryRow(ctx, `SELECT `+colunas+`
		FROM cofre_segredos WHERE id = $1 AND `+clausula(2), args...)
	item, err := scanSegredo(row.Scan)
	return item, mapDB(err)
}

func (r *Repository) Insert(ctx context.Context, db database.DBTX, item domain.Selado) error {
	_, err := db.Exec(ctx, `INSERT INTO cofre_segredos
		(id, nome, usuario, tipo, endereco, rotaciona, rotacao_pendente, proxima_rotacao, unidade_id, pasta_id, versao, dek_embrulhada, segredo_cifrado, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		item.ID, item.Nome, item.Usuario, item.Tipo, item.Endereco, item.Rotaciona, item.RotacaoPendente, item.ProximaRotacao,
		item.UnidadeID, item.PastaID, item.Versao, item.Dek, item.Cifra, item.CreatedAt, item.UpdatedAt)
	return mapDB(err)
}

func (r *Repository) Update(ctx context.Context, db database.DBTX, item domain.Selado, f domain.Filtro) error {
	args := append([]any{
		item.ID, item.Nome, item.Usuario, item.Tipo, item.Endereco, item.Rotaciona, item.RotacaoPendente, item.ProximaRotacao,
		item.UnidadeID, item.PastaID, item.Versao, item.Dek, item.Cifra, item.UpdatedAt,
	}, filtroVals(f)...)
	tag, err := db.Exec(ctx, `UPDATE cofre_segredos
		SET nome = $2, usuario = $3, tipo = $4, endereco = $5, rotaciona = $6, rotacao_pendente = $7, proxima_rotacao = $8,
		    unidade_id = $9, pasta_id = $10, versao = $11, dek_embrulhada = $12, segredo_cifrado = $13, updated_at = $14
		WHERE id = $1 AND `+clausula(15), args...)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, db database.DBTX, id uuid.UUID, f domain.Filtro) error {
	args := append([]any{id}, filtroVals(f)...)
	tag, err := db.Exec(ctx, `DELETE FROM cofre_segredos WHERE id = $1 AND `+clausula(2), args...)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ListarAdministracao(ctx context.Context, db database.DBTX, f domain.Filtro) (map[uuid.UUID][]uuid.UUID, error) {
	rows, err := db.Query(ctx, `SELECT a.painel_id, a.maquina_id
		FROM cofre_administra a
		JOIN cofre_segredos p ON p.id = a.painel_id
		JOIN cofre_segredos m ON m.id = a.maquina_id
		WHERE ($1 = false OR p.unidade_id = ANY($2::uuid[]))
		  AND ($1 = false OR m.unidade_id = ANY($2::uuid[]))
		ORDER BY a.painel_id, a.maquina_id`, f.Restrito, idsOrEmpty(f.Unidades))
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[uuid.UUID][]uuid.UUID{}
	for rows.Next() {
		var painel, maquina uuid.UUID
		if err := rows.Scan(&painel, &maquina); err != nil {
			return nil, mapDB(err)
		}
		out[painel] = append(out[painel], maquina)
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) SubstituirAdministracao(ctx context.Context, db database.DBTX, painel uuid.UUID, maquinas []uuid.UUID, f domain.Filtro) error {
	if _, err := db.Exec(ctx, `DELETE FROM cofre_administra a
		USING cofre_segredos m
		WHERE a.painel_id = $1 AND a.maquina_id = m.id
		  AND ($2 = false OR m.unidade_id = ANY($3::uuid[]))`,
		painel, f.Restrito, idsOrEmpty(f.Unidades)); err != nil {
		return mapDB(err)
	}
	for _, maquina := range maquinas {
		if _, err := db.Exec(ctx, `INSERT INTO cofre_administra (painel_id, maquina_id) VALUES ($1, $2)`, painel, maquina); err != nil {
			return mapDB(err)
		}
	}
	return nil
}

func (r *Repository) Travar(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	var got uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM cofre_segredos WHERE id = $1 FOR UPDATE`, id).Scan(&got)
	return mapDB(err)
}

func (r *Repository) RetiradaAberta(ctx context.Context, db database.DBTX, segredo uuid.UUID) (domain.RetiradaGravada, error) {
	var item domain.RetiradaGravada
	err := db.QueryRow(ctx, `SELECT id, segredo_id, usuario_id, nome, ate
		FROM cofre_retiradas WHERE segredo_id = $1 AND devolvida_em IS NULL`, segredo).
		Scan(&item.ID, &item.SegredoID, &item.UsuarioID, &item.Nome, &item.Ate)
	return item, mapDB(err)
}

func (r *Repository) InserirRetirada(ctx context.Context, db database.DBTX, item domain.RetiradaGravada) error {
	_, err := db.Exec(ctx, `INSERT INTO cofre_retiradas
		(id, segredo_id, usuario_id, nome, ate, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.SegredoID, item.UsuarioID, item.Nome, item.Ate, time.Now().UTC())
	if database.IsUniqueViolation(err) {
		return domain.ErrOcupado
	}
	return mapDB(err)
}

func (r *Repository) RenovarRetirada(ctx context.Context, db database.DBTX, id uuid.UUID, ate time.Time, nome string) error {
	tag, err := db.Exec(ctx, `UPDATE cofre_retiradas SET ate = $2, nome = $3
		WHERE id = $1 AND devolvida_em IS NULL`, id, ate, nome)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) EncerrarRetirada(ctx context.Context, db database.DBTX, id uuid.UUID, quando time.Time) error {
	tag, err := db.Exec(ctx, `UPDATE cofre_retiradas SET devolvida_em = $2
		WHERE id = $1 AND devolvida_em IS NULL`, id, quando)
	if err != nil {
		return mapDB(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *Repository) ListarRetiradas(ctx context.Context, db database.DBTX, f domain.Filtro, agora time.Time) (map[uuid.UUID]domain.Retirada, error) {
	rows, err := db.Query(ctx, `SELECT r.segredo_id, r.usuario_id, r.nome, r.ate
		FROM cofre_retiradas r
		JOIN cofre_segredos s ON s.id = r.segredo_id
		WHERE r.devolvida_em IS NULL AND r.ate > $3
		  AND ($1 = false OR s.unidade_id = ANY($2::uuid[]))`,
		f.Restrito, idsOrEmpty(f.Unidades), agora)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]domain.Retirada{}
	for rows.Next() {
		var segredo uuid.UUID
		var item domain.Retirada
		if err := rows.Scan(&segredo, &item.UsuarioID, &item.Nome, &item.Ate); err != nil {
			return nil, mapDB(err)
		}
		out[segredo] = item
	}
	return out, mapDB(rows.Err())
}

func (r *Repository) ListarParaRotacionar(ctx context.Context, db database.DBTX, agora time.Time) ([]domain.Selado, error) {
	rows, err := db.Query(ctx, `SELECT `+colunas+`
		FROM cofre_segredos
		WHERE rotaciona AND NOT rotacao_pendente AND proxima_rotacao IS NOT NULL AND proxima_rotacao <= $1
		ORDER BY proxima_rotacao, id
		LIMIT 50`, agora)
	if err != nil {
		return nil, mapDB(err)
	}
	defer rows.Close()
	out := []domain.Selado{}
	for rows.Next() {
		item, err := scanSegredo(rows.Scan)
		if err != nil {
			return nil, mapDB(err)
		}
		out = append(out, item)
	}
	return out, mapDB(rows.Err())
}
