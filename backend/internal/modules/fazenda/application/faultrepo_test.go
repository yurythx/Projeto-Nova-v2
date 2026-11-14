package application_test

// Segue scripts/genfault.py. A interface do domínio se chama Repositorio.

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

type innerRepo = domain.Repositorio

var errBoom = errors.New("falha simulada no repositório")

// faultRepo envolve o repositório real: na chamada failAt devolve erro; depois
// da chamada poisonAt envenena a transação (a instrução SQL seguinte — do
// repositório, do outbox ou da auditoria — falha). noTx marca veneno em
// leitura fora de transação, onde não há instrução seguinte na mesma conexão.
type faultRepo struct {
	inner                   innerRepo
	calls, failAt, poisonAt int
	trace                   []string
	noTx                    bool
}

func (fr *faultRepo) hook(name string) error {
	fr.calls++
	fr.trace = append(fr.trace, name)
	if fr.calls == fr.failAt {
		return errBoom
	}
	return nil
}

func (fr *faultRepo) post(ctx context.Context, db database.DBTX) {
	if fr.calls != fr.poisonAt {
		return
	}
	if _, inTx := db.(pgx.Tx); !inTx {
		fr.noTx = true
		return
	}
	_, _ = db.Exec(ctx, `SELECT 1/0`)
}

func (fr *faultRepo) Inserir(ctx context.Context, db database.DBTX, p domain.Processo) error {
	if err := fr.hook("Inserir"); err != nil {
		return err
	}
	defer fr.post(ctx, db)
	return fr.inner.Inserir(ctx, db, p)
}

func (fr *faultRepo) Atualizar(ctx context.Context, db database.DBTX, p domain.Processo) error {
	if err := fr.hook("Atualizar"); err != nil {
		return err
	}
	defer fr.post(ctx, db)
	return fr.inner.Atualizar(ctx, db, p)
}

func (fr *faultRepo) Buscar(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Processo, error) {
	if err := fr.hook("Buscar"); err != nil {
		var z0 domain.Processo
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Buscar(ctx, db, id)
}

func (fr *faultRepo) Listar(ctx context.Context, db database.DBTX, titular string) ([]domain.Processo, error) {
	if err := fr.hook("Listar"); err != nil {
		var z0 []domain.Processo
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Listar(ctx, db, titular)
}

func (fr *faultRepo) Proximo(ctx context.Context, db database.DBTX, ano int) (int, error) {
	if err := fr.hook("Proximo"); err != nil {
		var z0 int
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Proximo(ctx, db, ano)
}

func (fr *faultRepo) GuardarPeca(ctx context.Context, db database.DBTX, processo uuid.UUID, tipo, objectKey string) error {
	if err := fr.hook("GuardarPeca"); err != nil {
		return err
	}
	defer fr.post(ctx, db)
	return fr.inner.GuardarPeca(ctx, db, processo, tipo, objectKey)
}

func (fr *faultRepo) ListarPecas(ctx context.Context, db database.DBTX, processo uuid.UUID) ([]domain.PecaArquivo, error) {
	if err := fr.hook("ListarPecas"); err != nil {
		return nil, err
	}
	defer fr.post(ctx, db)
	return fr.inner.ListarPecas(ctx, db, processo)
}

func (fr *faultRepo) PorCodigo(ctx context.Context, db database.DBTX, codigo string) (domain.Processo, error) {
	if err := fr.hook("PorCodigo"); err != nil {
		var z0 domain.Processo
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.PorCodigo(ctx, db, codigo)
}
