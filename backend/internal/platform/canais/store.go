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
	// WhatsApp devolve a configuração (ok false: nunca configurada).
	WhatsApp(ctx context.Context) (WhatsApp, bool, error)
	// SalvarWhatsApp grava. Token e chave vazios mantêm os gravados, a
	// menos que removerToken ou removerChave.
	SalvarWhatsApp(ctx context.Context, w WhatsApp, removerToken, removerChave bool, por string) (WhatsApp, error)
	// UsuarioPorWhatsApp acha a conta ativa dona de um dos números.
	UsuarioPorWhatsApp(ctx context.Context, numeros []string) (uuid.UUID, bool, error)
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

const colunasWhatsApp = `ativo, provedor, meta_numero_id, meta_token_cifrado, meta_modelo, meta_idioma,
	evolution_url, evolution_instancia, evolution_chave_cifrada, receber, webhook_token_cifrado, meta_app_secret_cifrado,
	updated_at, updated_by`

func (s *PostgresStore) WhatsApp(ctx context.Context) (WhatsApp, bool, error) {
	w, err := s.scanWhatsApp(s.db.QueryRow(ctx, `SELECT `+colunasWhatsApp+` FROM canais_whatsapp WHERE id = 1`))
	if database.IsNoRows(err) {
		return WhatsApp{Provedor: ProvedorMeta, MetaIdioma: "pt_BR"}, false, nil
	}
	return w, err == nil, err
}

func (s *PostgresStore) scanWhatsApp(row interface{ Scan(...any) error }) (WhatsApp, error) {
	var w WhatsApp
	var token, chave, webhook, appSecret string
	if err := row.Scan(&w.Ativo, &w.Provedor, &w.MetaNumeroID, &token, &w.MetaModelo, &w.MetaIdioma,
		&w.EvolutionURL, &w.EvolutionInstancia, &chave, &w.Receber, &webhook, &appSecret, &w.UpdatedAt, &w.UpdatedBy); err != nil {
		return w, err
	}
	for _, c := range []struct {
		cifrado string
		claro   *string
	}{{token, &w.MetaToken}, {chave, &w.EvolutionChave}, {webhook, &w.WebhookToken}, {appSecret, &w.MetaAppSecret}} {
		claro, err := s.cipher.Decrypt(c.cifrado)
		if err != nil {
			return w, fmt.Errorf("canais: credencial do WhatsApp ilegível (CONFIG_ENCRYPTION_KEY mudou?): %w", err)
		}
		*c.claro = claro
	}
	w.TemToken, w.TemChave, w.TemAppSecret = w.MetaToken != "", w.EvolutionChave != "", w.MetaAppSecret != ""
	return w, nil
}

func (s *PostgresStore) SalvarWhatsApp(ctx context.Context, w WhatsApp, removerToken, removerChave bool, por string) (WhatsApp, error) {
	cifrar := func(s2 string) string {
		if s2 == "" {
			return ""
		}
		return s.cipher.Encrypt(s2)
	}
	// O token do webhook vem pronto do serviço só quando é novo; vazio
	// mantém o gravado.
	out, err := s.scanWhatsApp(s.db.QueryRow(ctx, `INSERT INTO canais_whatsapp (id, ativo, provedor, meta_numero_id, meta_token_cifrado,
			meta_modelo, meta_idioma, evolution_url, evolution_instancia, evolution_chave_cifrada, receber, webhook_token_cifrado,
			meta_app_secret_cifrado, updated_at, updated_by)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $13, $14, $15, now(), $12)
		ON CONFLICT (id) DO UPDATE SET ativo = $1, provedor = $2, meta_numero_id = $3, meta_modelo = $5, meta_idioma = $6,
			evolution_url = $7, evolution_instancia = $8, receber = $13, updated_at = now(), updated_by = $12,
			meta_token_cifrado = CASE WHEN $10 THEN '' WHEN $4 <> '' THEN $4 ELSE canais_whatsapp.meta_token_cifrado END,
			evolution_chave_cifrada = CASE WHEN $11 THEN '' WHEN $9 <> '' THEN $9 ELSE canais_whatsapp.evolution_chave_cifrada END,
			webhook_token_cifrado = CASE WHEN $14 <> '' THEN $14 ELSE canais_whatsapp.webhook_token_cifrado END,
			meta_app_secret_cifrado = CASE WHEN $15 <> '' THEN $15 ELSE canais_whatsapp.meta_app_secret_cifrado END
		RETURNING `+colunasWhatsApp, w.Ativo, w.Provedor, w.MetaNumeroID, cifrar(w.MetaToken), w.MetaModelo, w.MetaIdioma,
		w.EvolutionURL, w.EvolutionInstancia, cifrar(w.EvolutionChave), removerToken, removerChave, por, w.Receber,
		cifrar(w.WebhookToken), cifrar(w.MetaAppSecret)))
	if err != nil {
		return WhatsApp{}, fmt.Errorf("canais: %w", err)
	}
	return out, nil
}

// UsuarioPorWhatsApp acha a conta ativa dona de um dos números. Dois donos
// do mesmo número é ambíguo: ninguém responde por ele.
func (s *PostgresStore) UsuarioPorWhatsApp(ctx context.Context, numeros []string) (uuid.UUID, bool, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM users WHERE active AND contato_whatsapp = ANY($1::text[]) LIMIT 2`, numeros)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("canais: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return uuid.Nil, false, fmt.Errorf("canais: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return uuid.Nil, false, fmt.Errorf("canais: %w", err)
	}
	if len(ids) != 1 {
		return uuid.Nil, false, nil
	}
	return ids[0], true, nil
}
