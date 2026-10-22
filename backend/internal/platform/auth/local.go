package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
)

// LocalIssuer é a identidade em nome da qual o backend assina seus
// próprios tokens locais (§ Sistema de Login Local), usada tanto como
// "iss" quanto como "aud" das claims — os dois únicos participantes deste
// protocolo são "este mesmo processo assina" e "este mesmo processo
// verifica", então issuer e audience coincidem deliberadamente. Também
// serve para diferenciar nos logs/depuração um token local de um token do
// Keycloak.
const LocalIssuer = "projeto-nexus-local"

// minRSAKeyBits é o tamanho mínimo de chave aceito por NewLocalSigner —
// abaixo disso a assinatura RSA não é mais considerada segura pelos
// padrões atuais (NIST SP 800-57 recomenda 2048 bits como piso).
const minRSAKeyBits = 2048

// LocalAccount é o subconjunto de dados de um usuário local necessário
// para montar um token — deliberadamente sem PasswordHash: quem monta um
// LocalAccount já verificou a senha antes (ver internal/platform/localauth),
// este tipo nunca deveria conseguir carregar o hash por engano.
type LocalAccount struct {
	ID       string
	Name     string
	Username string
	Email    string
	Roles    []string
	Groups   []string
}

// localClaims é o formato do JWT que o backend assina para logins locais.
// Espelha deliberadamente o suficiente de accessTokenClaims para que
// LocalSigner.verifyToken produza uma Identity idêntica em forma à que sai
// de um token do Keycloak — o resto da aplicação (RBAC, auditoria,
// handlers) nunca precisa saber qual dos dois caminhos autenticou o
// chamador.
type localClaims struct {
	jwt.RegisteredClaims
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	Roles             []string `json:"roles"`
	Groups            []string `json:"groups"`
	// AuthTime é o instante do login com senha (claim OIDC "auth_time"),
	// preservado a cada renovação — é dele que se mede o teto absoluto da
	// sessão (LOCAL_AUTH_SESSION_MAX).
	AuthTime *jwt.NumericDate `json:"auth_time,omitempty"`
}

// ErrSessionTooOld recusa renovar um token cuja sessão (contada do login
// com senha) já passou do teto absoluto: a renovação estende a sessão
// enquanto há uso, mas não para sempre.
var ErrSessionTooOld = errors.New("auth: local session exceeded its maximum lifetime")

// LocalSigner assina e verifica os tokens do login local com um par de
// chaves RSA (RS256) PRÓPRIO — nunca compartilhado com o Keycloak/SSO, nem
// com nenhum outro segredo da aplicação (§ Sistema de Login Local: "um
// login separado do SSO"). Construído uma única vez no bootstrap
// (internal/app/dependencies.go) a partir de config.LocalAuthConfig; um
// *LocalSigner nil em qualquer ponto downstream (Verifier, localauth.Handlers)
// significa "login local desligado" — a checagem de nil substitui um campo
// booleano "Enabled" espalhado por múltiplas camadas.
type LocalSigner struct {
	privateKey *rsa.PrivateKey
	tokenTTL   time.Duration
	sessionMax time.Duration // 0 = sem teto absoluto
}

// NewLocalSigner analisa cfg.PrivateKeyPEM e monta um LocalSigner pronto
// para assinar/verificar. Retorna (nil, nil) — não um erro — quando
// cfg.Enabled é false, já que não há nada para configurar nesse caso; a
// validação de que PrivateKeyPEM está presente quando Enabled=true já
// aconteceu em config.Load (fail fast na configuração, antes de chegar
// aqui). Falha (erro, não nil) para qualquer PEM inválido ou chave menor
// que minRSAKeyBits — a aplicação deve recusar subir com uma chave fraca
// em vez de aceitar silenciosamente.
func NewLocalSigner(cfg config.LocalAuthConfig) (*LocalSigner, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	key, err := parseRSAPrivateKeyPEM(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth: parse LOCAL_AUTH_PRIVATE_KEY: %w", err)
	}
	if bits := key.N.BitLen(); bits < minRSAKeyBits {
		return nil, fmt.Errorf("auth: LOCAL_AUTH_PRIVATE_KEY is %d bits, minimum is %d", bits, minRSAKeyBits)
	}

	// Sem normalização de TokenTTL aqui: config.Load já garante um
	// default de 1h quando LOCAL_AUTH_TOKEN_TTL não é definida
	// (l.durationVal), e um valor negativo/zero passado deliberadamente
	// (como um teste construindo um token já expirado) deve ser
	// respeitado, não silenciosamente substituído.
	return &LocalSigner{privateKey: key, tokenTTL: cfg.TokenTTL, sessionMax: cfg.SessionMax}, nil
}

// parseRSAPrivateKeyPEM aceita tanto PKCS1 ("BEGIN RSA PRIVATE KEY") quanto
// PKCS8 ("BEGIN PRIVATE KEY") — os dois formatos que `openssl genrsa`/`openssl
// genpkey` produzem, dependendo da versão/flags usadas, para não forçar
// quem está gerando a chave a acertar o formato exato.
func parseRSAPrivateKeyPEM(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not a valid PKCS1 or PKCS8 RSA private key: %w", err)
	}
	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("PEM key is not an RSA private key (got %T)", keyAny)
	}
	return key, nil
}

// IssueToken assina um token RS256 de curta duração para account. Chamado
// pelo handler de login local depois que a senha já foi conferida — este
// tipo nunca verifica senha, só emite e valida o token resultante.
func (s *LocalSigner) IssueToken(account LocalAccount) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	return s.issue(account, now, now)
}

// RenewToken emite um token novo para uma sessão local ainda válida
// (POST /auth/refresh), mantendo o authTime do login original. O
// vencimento nunca passa de authTime+sessionMax; uma sessão que já chegou
// ao teto recebe ErrSessionTooOld e o usuário volta a digitar a senha.
func (s *LocalSigner) RenewToken(account LocalAccount, authTime time.Time) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	if s.sessionMax > 0 && !now.Before(authTime.Add(s.sessionMax)) {
		return "", time.Time{}, ErrSessionTooOld
	}
	return s.issue(account, now, authTime)
}

func (s *LocalSigner) issue(account LocalAccount, now, authTime time.Time) (token string, expiresAt time.Time, err error) {
	expiresAt = now.Add(s.tokenTTL)
	if limit := authTime.Add(s.sessionMax); s.sessionMax > 0 && expiresAt.After(limit) {
		expiresAt = limit
	}
	claims := localClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    LocalIssuer,
			Audience:  jwt.ClaimStrings{LocalIssuer},
			Subject:   account.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		PreferredUsername: account.Username,
		Email:             account.Email,
		Name:              account.Name,
		Roles:             account.Roles,
		Groups:            account.Groups,
		AuthTime:          jwt.NewNumericDate(authTime),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign local token: %w", err)
	}
	return signed, expiresAt, nil
}

// verifyToken confere a assinatura RS256 de rawToken contra a chave
// pública derivada de s.privateKey, e a issuer/audience/expiração das
// claims, extraindo a Identity resultante. Retorna erro para qualquer
// token que não tenha sido assinado por este mesmo processo com esta
// mesma chave — incluindo, deliberadamente, um token de verdade do
// Keycloak (assinado por uma chave RSA completamente diferente, a do
// realm, nunca a local).
func (s *LocalSigner) verifyToken(rawToken string) (Identity, error) {
	var claims localClaims
	_, err := jwt.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		return &s.privateKey.PublicKey, nil
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(LocalIssuer),
		jwt.WithAudience(LocalIssuer),
	)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: local token verification failed: %w", err)
	}
	// Tokens emitidos antes do auth_time existir: o iat é o melhor
	// substituto (subestima a idade da sessão em no máximo um TTL).
	authTime := claims.AuthTime
	if authTime == nil {
		authTime = claims.IssuedAt
	}

	return Identity{
		Subject:  claims.Subject,
		Username: claims.PreferredUsername,
		Email:    claims.Email,
		Name:     claims.Name,
		Roles:    claims.Roles,
		Groups:   claims.Groups,
		Source:   SourceLocal,
		AuthTime: numericTime(authTime),
	}, nil
}

func numericTime(d *jwt.NumericDate) time.Time {
	if d == nil {
		return time.Time{}
	}
	return d.Time
}
