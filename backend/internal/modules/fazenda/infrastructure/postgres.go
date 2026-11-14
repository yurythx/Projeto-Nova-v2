// Package infrastructure persiste o protocolo da Fazenda.
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const colunas = `id, numero, ano, seq, programa, entrada, titular, situacao, recibo_em, codigo,
	certidao, aliquota, execucao, exigencia_aberta, exigencia_prazo, exigencia_prorrogada, despacho_em, alerta_limite, destinatario, consulta_em,
	tqc_assinado_em, selo_criterios, isencao_fim, cassacao_ciencia`

// PostgresRepository implementa domain.Repositorio.
type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func scanProcesso(row pgx.Row) (domain.Processo, error) {
	var p domain.Processo
	var aberta, prazo, despacho, alerta, consulta, tqc, cassacao *time.Time
	var destinatario *uuid.UUID
	err := row.Scan(&p.ID, &p.Numero, &p.Ano, &p.Seq, &p.Programa, &p.Entrada, &p.Titular, &p.Situacao,
		&p.ReciboEm, &p.Codigo, &p.Certidao, &p.Aliquota, &p.Execucao, &aberta, &prazo, &p.ExigenciaProrrogada, &despacho, &alerta, &destinatario, &consulta,
		&tqc, &p.SeloCriterios, &p.IsencaoFim, &cassacao)
	if aberta != nil {
		p.ExigenciaAberta = *aberta
	}
	if prazo != nil {
		p.ExigenciaPrazo = *prazo
	}
	if despacho != nil {
		p.DespachoEm = *despacho
	}
	if alerta != nil {
		p.AlertaLimite = *alerta
	}
	if destinatario != nil {
		p.Destinatario = *destinatario
	}
	if consulta != nil {
		p.ConsultaEm = *consulta
	}
	if tqc != nil {
		p.TQCAssinadoEm = *tqc
	}
	if cassacao != nil {
		p.CassacaoCiencia = *cassacao
	}
	return p, err
}

func idOuNulo(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func tempo(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (r *PostgresRepository) Inserir(ctx context.Context, db database.DBTX, p domain.Processo) error {
	_, err := db.Exec(ctx, `INSERT INTO fazenda_processos (`+colunas+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		p.ID, p.Numero, p.Ano, p.Seq, p.Programa, p.Entrada, p.Titular, p.Situacao, p.ReciboEm, p.Codigo,
		p.Certidao, p.Aliquota, p.Execucao, tempo(p.ExigenciaAberta), tempo(p.ExigenciaPrazo), p.ExigenciaProrrogada,
		tempo(p.DespachoEm), tempo(p.AlertaLimite), idOuNulo(p.Destinatario), tempo(p.ConsultaEm),
		tempo(p.TQCAssinadoEm), p.SeloCriterios, p.IsencaoFim, tempo(p.CassacaoCiencia))
	if err != nil {
		return fmt.Errorf("fazenda.Inserir: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Atualizar(ctx context.Context, db database.DBTX, p domain.Processo) error {
	tag, err := db.Exec(ctx, `UPDATE fazenda_processos SET situacao = $2, certidao = $3, aliquota = $4, execucao = $5,
		exigencia_aberta = $6, exigencia_prazo = $7, exigencia_prorrogada = $8, despacho_em = $9, alerta_limite = $10, destinatario = $11, consulta_em = $12,
		tqc_assinado_em = $13, selo_criterios = $14, isencao_fim = $15, cassacao_ciencia = $16 WHERE id = $1`,
		p.ID, p.Situacao, p.Certidao, p.Aliquota, p.Execucao, tempo(p.ExigenciaAberta), tempo(p.ExigenciaPrazo), p.ExigenciaProrrogada,
		tempo(p.DespachoEm), tempo(p.AlertaLimite), idOuNulo(p.Destinatario), tempo(p.ConsultaEm),
		tempo(p.TQCAssinadoEm), p.SeloCriterios, p.IsencaoFim, tempo(p.CassacaoCiencia))
	if err != nil {
		return fmt.Errorf("fazenda.Atualizar: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNaoEncontrado
	}
	return nil
}

func (r *PostgresRepository) Buscar(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Processo, error) {
	p, err := scanProcesso(db.QueryRow(ctx, `SELECT `+colunas+` FROM fazenda_processos WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Processo{}, domain.ErrNaoEncontrado
	}
	if err != nil {
		return domain.Processo{}, fmt.Errorf("fazenda.Buscar: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) PorCodigo(ctx context.Context, db database.DBTX, codigo string) (domain.Processo, error) {
	p, err := scanProcesso(db.QueryRow(ctx, `SELECT `+colunas+` FROM fazenda_processos WHERE codigo = $1`, codigo))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Processo{}, domain.ErrNaoEncontrado
	}
	if err != nil {
		return domain.Processo{}, fmt.Errorf("fazenda.PorCodigo: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) Listar(ctx context.Context, db database.DBTX, titular string) ([]domain.Processo, error) {
	rows, err := db.Query(ctx, `SELECT `+colunas+` FROM fazenda_processos WHERE ($1 = '' OR titular = $1) ORDER BY recibo_em, id`, titular)
	if err != nil {
		return nil, fmt.Errorf("fazenda.Listar: %w", err)
	}
	defer rows.Close()
	out := []domain.Processo{}
	for rows.Next() {
		p, err := scanProcesso(rows)
		if err != nil {
			return nil, fmt.Errorf("fazenda.Listar scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fazenda.Listar rows: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) GuardarPeca(ctx context.Context, db database.DBTX, processo uuid.UUID, tipo, objectKey string) error {
	_, err := db.Exec(ctx, `INSERT INTO fazenda_pecas (processo_id, tipo, object_key) VALUES ($1, $2, $3)
		ON CONFLICT (processo_id, tipo) DO UPDATE SET object_key = EXCLUDED.object_key`, processo, tipo, objectKey)
	if err != nil {
		return fmt.Errorf("fazenda.GuardarPeca: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListarPecas(ctx context.Context, db database.DBTX, processo uuid.UUID) ([]domain.PecaArquivo, error) {
	rows, err := db.Query(ctx, `SELECT tipo, object_key FROM fazenda_pecas WHERE processo_id = $1 ORDER BY tipo`, processo)
	if err != nil {
		return nil, fmt.Errorf("fazenda.ListarPecas: %w", err)
	}
	defer rows.Close()
	out := []domain.PecaArquivo{}
	for rows.Next() {
		var peca domain.PecaArquivo
		var chave string
		if err := rows.Scan(&peca.Tipo, &chave); err != nil {
			return nil, fmt.Errorf("fazenda.ListarPecas scan: %w", err)
		}
		peca.Chave = chave
		out = append(out, peca)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fazenda.ListarPecas rows: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) Proximo(ctx context.Context, db database.DBTX, ano int) (int, error) {
	var n int
	err := db.QueryRow(ctx, `INSERT INTO fazenda_sequencia (ano, ultimo) VALUES ($1, 1)
		ON CONFLICT (ano) DO UPDATE SET ultimo = fazenda_sequencia.ultimo + 1 RETURNING ultimo`, ano).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("fazenda.Proximo: %w", err)
	}
	return n, nil
}
