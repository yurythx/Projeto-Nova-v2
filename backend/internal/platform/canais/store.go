package canais

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/secretcrypto"
)

// Store persiste a configuração do e-mail e o contato das pessoas.
type Store interface {
	// Email devolve a configuração da tela (ok false: nunca configurada).
	Email(ctx context.Context) (Email, bool, error)
	// SalvarEmail grava. Senha vazia mantém a gravada, a menos que removerSenha.
	SalvarEmail(ctx context.Context, e Email, removerSenha bool, por string) (Email, error)
	Contato(ctx context.Context, usuario uuid.UUID) (Contato, error)
	SalvarContato(ctx context.Context, usuario uuid.UUID, c Contato) (Contato, error)
}

// PostgresStore implementa Store.
type PostgresStore struct {
	db     database.DBTX
	cipher *secretcrypto.Cipher
}

// NewPostgresStore cria o store.
func NewPostgresStore(db database.DBTX, cipher *secretcrypto.Cipher) *PostgresStore {
	return &PostgresStore{db: db, cipher: cipher}
}

var _ Store = (*PostgresStore)(nil)

const colunasEmail = `ativo, host, porta, usuario, senha_cifrada, remetente, updated_at, updated_by`

func (s *PostgresStore) Email(ctx context.Context) (Email, bool, error) {
	e, err := s.scanEmail(s.db.QueryRow(ctx, `SELECT `+colunasEmail+` FROM canais_email WHERE id = 1`))
	if database.IsNoRows(err) {
		return Email{Origem: OrigemNenhuma}, false, nil
	}
	return e, err == nil, err
}

func (s *PostgresStore) scanEmail(row interface{ Scan(...any) error }) (Email, error) {
	var e Email
	var cifrada string
	if err := row.Scan(&e.Ativo, &e.Host, &e.Porta, &e.Usuario, &cifrada, &e.Remetente, &e.UpdatedAt, &e.UpdatedBy); err != nil {
		return e, err
	}
	senha, err := s.cipher.Decrypt(cifrada)
	if err != nil {
		return e, fmt.Errorf("canais: senha do SMTP ilegível (CONFIG_ENCRYPTION_KEY mudou?): %w", err)
	}
	e.Senha, e.TemSenha, e.Origem = senha, senha != "", OrigemTela
	return e, nil
}

func (s *PostgresStore) SalvarEmail(ctx context.Context, e Email, removerSenha bool, por string) (Email, error) {
	nova := ""
	if e.Senha != "" {
		nova = s.cipher.Encrypt(e.Senha)
	}
	out, err := s.scanEmail(s.db.QueryRow(ctx, `INSERT INTO canais_email (id, ativo, host, porta, usuario, senha_cifrada, remetente, updated_at, updated_by)
		VALUES (1, $1, $2, $3, $4, $5, $6, now(), $8)
		ON CONFLICT (id) DO UPDATE SET ativo = $1, host = $2, porta = $3, usuario = $4, remetente = $6, updated_at = now(), updated_by = $8,
			senha_cifrada = CASE WHEN $7 THEN '' WHEN $5 <> '' THEN $5 ELSE canais_email.senha_cifrada END
		RETURNING `+colunasEmail, e.Ativo, e.Host, e.Porta, e.Usuario, nova, e.Remetente, removerSenha, por))
	if err != nil {
		return Email{}, fmt.Errorf("canais: %w", err)
	}
	return out, nil
}

// Contato devolve o contato da pessoa e o e-mail do login (quando é real).
func (s *PostgresStore) Contato(ctx context.Context, usuario uuid.UUID) (Contato, error) {
	var c Contato
	var sso string
	err := s.db.QueryRow(ctx, `SELECT contato_email, contato_whatsapp, contato_respondido_em, email FROM users WHERE id = $1`, usuario).
		Scan(&c.Email, &c.WhatsApp, &c.RespondidoEm, &sso)
	if database.IsNoRows(err) {
		return c, ErrSemConta
	}
	if err != nil {
		return c, fmt.Errorf("canais: %w", err)
	}
	c.Respondido = c.RespondidoEm != nil
	if !strings.HasSuffix(sso, ".invalid") {
		c.EmailDoSSO = sso
	}
	return c, nil
}

func (s *PostgresStore) SalvarContato(ctx context.Context, usuario uuid.UUID, c Contato) (Contato, error) {
	tag, err := s.db.Exec(ctx, `UPDATE users SET contato_email = $2, contato_whatsapp = $3, contato_respondido_em = now() WHERE id = $1`,
		usuario, c.Email, c.WhatsApp)
	if err != nil {
		return Contato{}, fmt.Errorf("canais: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Contato{}, ErrSemConta
	}
	return s.Contato(ctx, usuario)
}
