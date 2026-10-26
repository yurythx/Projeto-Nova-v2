// Package infrastructure persiste o catálogo e conta, só para leitura, os
// registros que ocupam cada coluna. Não importa os outros plug-ins.
package infrastructure

import (
	"context"
	"fmt"

	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) Listar(ctx context.Context, db database.DBTX, quadro string) ([]domain.Coluna, error) {
	rows, err := db.Query(ctx, `
		SELECT quadro, chave, nome, cor, tom, ordem, entrada, ativa
		FROM quadro_colunas
		WHERE quadro = $1
		ORDER BY ativa DESC, ordem, chave`, quadro)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Coluna{}
	for rows.Next() {
		var c domain.Coluna
		if err := rows.Scan(&c.Quadro, &c.Chave, &c.Nome, &c.Cor, &c.Tom, &c.Ordem, &c.Entrada, &c.Ativa); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) SoltarOrdens(ctx context.Context, db database.DBTX, quadro string) error {
	_, err := db.Exec(ctx, `UPDATE quadro_colunas SET ordem = ordem + 1000 WHERE quadro = $1 AND ordem < 1000`, quadro)
	return err
}

func (r *Repository) Inserir(ctx context.Context, db database.DBTX, c domain.Coluna) error {
	_, err := db.Exec(ctx, `
		INSERT INTO quadro_colunas (quadro, chave, ordem, nome, cor, tom, entrada, ativa)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.Quadro, c.Chave, c.Ordem, c.Nome, c.Cor, c.Tom, c.Entrada, c.Ativa)
	return err
}

func (r *Repository) Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error) {
	rows, err := db.Query(ctx, `
		SELECT p.de, p.para
		FROM quadro_passagens p
		JOIN quadro_colunas destino ON destino.quadro = p.quadro AND destino.chave = p.para
		WHERE p.quadro = $1 AND destino.ativa
		ORDER BY p.de, p.para`, quadro)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var de, para string
		if err := rows.Scan(&de, &para); err != nil {
			return nil, err
		}
		out[de] = append(out[de], para)
	}
	return out, rows.Err()
}

func (r *Repository) SubstituirPassagens(ctx context.Context, db database.DBTX, quadro, chave string, itens []domain.Passagem) error {
	if _, err := db.Exec(ctx, `DELETE FROM quadro_passagens WHERE quadro = $1 AND (de = $2 OR para = $2)`, quadro, chave); err != nil {
		return err
	}
	for _, item := range itens {
		if _, err := db.Exec(ctx, `INSERT INTO quadro_passagens (quadro, de, para) VALUES ($1, $2, $3)`, quadro, item.De, item.Para); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Salvar(ctx context.Context, db database.DBTX, cols []domain.Coluna) error {
	for _, c := range cols {
		tag, err := db.Exec(ctx, `
			UPDATE quadro_colunas
			SET nome = $3, cor = $4, ordem = $5, ativa = $6, updated_at = now()
			WHERE quadro = $1 AND chave = $2`,
			c.Quadro, c.Chave, c.Nome, c.Cor, c.Ordem, c.Ativa)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNaoEncontrada
		}
	}
	return nil
}

// Contar lê a tabela do módulo. fechada junta rejeitada, falhou e
// cancelada. concluida junta resolvido e atendido. O estado do registro
// não é reescrito.
func (r *Repository) Contar(ctx context.Context, db database.DBTX, quadro, chave string) (int, error) {
	q, args, ok := contagem(quadro, chave)
	if !ok {
		return 0, domain.ErrQuadro
	}
	var n int
	if err := db.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("quadros: contar %s/%s: %w", quadro, chave, err)
	}
	return n, nil
}

func contagem(quadro, chave string) (string, []any, bool) {
	switch quadro {
	case "problemas":
		return `SELECT count(*) FROM problemas WHERE estado = $1`, []any{chave}, true
	case "mudancas":
		if chave == "fechada" {
			return `SELECT count(*) FROM mudancas WHERE estado IN ('rejeitada', 'falhou', 'cancelada')`, nil, true
		}
		return `SELECT count(*) FROM mudancas WHERE estado = $1`, []any{chave}, true
	case "projetos":
		return `SELECT count(*) FROM projetos WHERE estado = $1`, []any{chave}, true
	case "tarefas":
		return `SELECT count(*) FROM projetos_tarefas WHERE estado = $1`, []any{chave}, true
	case "ativos":
		return `SELECT count(*) FROM ativos WHERE estado = $1`, []any{chave}, true
	case "fila":
		if chave == "concluida" {
			return `SELECT count(*) FROM atendimento_solicitacoes WHERE estado IN ('resolvido', 'atendido')`, nil, true
		}
		return `SELECT count(*) FROM atendimento_solicitacoes WHERE estado = $1`, []any{chave}, true
	default:
		return "", nil, false
	}
}
