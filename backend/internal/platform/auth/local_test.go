package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
)

// testRSAKeyPEM generates a throwaway RSA private key (PKCS1 PEM) of the
// given bit size for use in a single test — never reused across tests, so
// no test accidentally depends on another's key material.
func testRSAKeyPEM(t *testing.T, bits int) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	return string(pem.EncodeToMemory(block))
}

func testSigner(t *testing.T, ttl time.Duration) *LocalSigner {
	t.Helper()
	cfg := config.LocalAuthConfig{Enabled: true, PrivateKeyPEM: testRSAKeyPEM(t, 2048), TokenTTL: ttl}
	signer, err := NewLocalSigner(cfg)
	if err != nil {
		t.Fatalf("NewLocalSigner: %v", err)
	}
	return signer
}

func TestIssueToken_ThenVerifyToken_RoundTrips(t *testing.T) {
	signer := testSigner(t, time.Hour)
	account := LocalAccount{ID: "user-1", Username: "admin", Email: "admin@projeto-nova.local", Roles: []string{"nexus-admin", "nexus-user"}}

	token, expiresAt, err := signer.IssueToken(account)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt = %v, want in the future", expiresAt)
	}

	identity, err := signer.verifyToken(token)
	if err != nil {
		t.Fatalf("verifyToken: %v", err)
	}
	if identity.Subject != "user-1" {
		t.Errorf("Subject = %q, want user-1", identity.Subject)
	}
	if identity.Username != "admin" {
		t.Errorf("Username = %q, want admin", identity.Username)
	}
	if identity.Source != SourceLocal {
		t.Errorf("Source = %q, want %q", identity.Source, SourceLocal)
	}
	if !identity.HasRole("nexus-admin") || !identity.HasRole("nexus-user") {
		t.Errorf("Roles = %v, want to include nexus-admin and nexus-user", identity.Roles)
	}
}

func TestVerifyToken_WrongKeyIsRejected(t *testing.T) {
	signerA := testSigner(t, time.Hour)
	signerB := testSigner(t, time.Hour)

	token, _, err := signerA.IssueToken(LocalAccount{ID: "user-1", Username: "admin"})
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	// signerB tem uma chave RSA completamente diferente — precisa rejeitar
	// um token assinado por signerA, exatamente como rejeitaria um token
	// de verdade do Keycloak (chave RSA de outra origem qualquer).
	if _, err := signerB.verifyToken(token); err == nil {
		t.Fatal("expected verification to fail with a different key pair")
	}
}

func TestVerifyToken_ExpiredTokenIsRejected(t *testing.T) {
	signer := testSigner(t, -time.Hour) // já expirado
	token, _, err := signer.IssueToken(LocalAccount{ID: "user-1", Username: "admin"})
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	if _, err := signer.verifyToken(token); err == nil {
		t.Fatal("expected verification to reject an expired token")
	}
}

// A renovação mantém o auth_time do login com senha, nunca vence depois
// do teto absoluto e é recusada quando a sessão já o atingiu.
func TestRenewToken_KeepsAuthTimeAndHonorsSessionMax(t *testing.T) {
	cfg := config.LocalAuthConfig{Enabled: true, PrivateKeyPEM: testRSAKeyPEM(t, 2048), TokenTTL: time.Hour, SessionMax: 8 * time.Hour}
	signer, err := NewLocalSigner(cfg)
	if err != nil {
		t.Fatal(err)
	}
	account := LocalAccount{ID: "user-1", Username: "admin"}

	token, _, err := signer.IssueToken(account)
	if err != nil {
		t.Fatal(err)
	}
	login, err := signer.verifyToken(token)
	if err != nil || time.Since(login.AuthTime) > time.Minute {
		t.Fatalf("login carimba auth_time = agora: %v %v", login.AuthTime, err)
	}

	authTime := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	renewed, exp, err := signer.RenewToken(account, authTime)
	if err != nil || time.Until(exp) < 59*time.Minute {
		t.Fatalf("renovação no meio da sessão: exp em %v, %v", time.Until(exp), err)
	}
	if id, err := signer.verifyToken(renewed); err != nil || !id.AuthTime.Equal(authTime) {
		t.Fatalf("auth_time preservado: %v (want %v) %v", id.AuthTime, authTime, err)
	}

	// A 30 min do teto: o token novo vence no teto, não 1h depois.
	_, exp, err = signer.RenewToken(account, time.Now().Add(-7*time.Hour-30*time.Minute))
	if err != nil || time.Until(exp) > 31*time.Minute {
		t.Fatalf("vencimento limitado ao teto: %v %v", time.Until(exp), err)
	}

	if _, _, err := signer.RenewToken(account, time.Now().Add(-8*time.Hour)); !errors.Is(err, ErrSessionTooOld) {
		t.Fatalf("sessão no teto: %v", err)
	}
}

func TestNewLocalSigner_DisabledReturnsNilWithoutError(t *testing.T) {
	signer, err := NewLocalSigner(config.LocalAuthConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewLocalSigner: unexpected error %v", err)
	}
	if signer != nil {
		t.Fatal("expected a nil signer when local auth is disabled")
	}
}

func TestNewLocalSigner_RejectsKeySmallerThanMinimum(t *testing.T) {
	_, err := NewLocalSigner(config.LocalAuthConfig{Enabled: true, PrivateKeyPEM: testRSAKeyPEM(t, 1024)})
	if err == nil {
		t.Fatal("expected an error for a key smaller than the minimum size")
	}
}

func TestNewLocalSigner_RejectsInvalidPEM(t *testing.T) {
	_, err := NewLocalSigner(config.LocalAuthConfig{Enabled: true, PrivateKeyPEM: "not a pem"})
	if err == nil {
		t.Fatal("expected an error for an invalid PEM value")
	}
}

// Tokens emitidos antes do auth_time existir: o iat faz as vezes dele; sem
// os dois, a idade da sessão fica desconhecida (instante zero).
func TestLocalTokenLegadoSemAuthTime(t *testing.T) {
	signer := testSigner(t, time.Hour)
	assinar := func(claims jwt.MapClaims) string {
		t.Helper()
		raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(signer.privateKey)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	emitido := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"iss": LocalIssuer, "aud": LocalIssuer, "sub": "u1", "exp": time.Now().Add(time.Hour).Unix()}
	}
	comIat := base()
	comIat["iat"] = emitido.Unix()
	id, err := signer.verifyToken(assinar(comIat))
	if err != nil || !id.AuthTime.Equal(emitido) {
		t.Fatalf("iat no lugar do auth_time: %v %v", id.AuthTime, err)
	}
	id, err = signer.verifyToken(assinar(base()))
	if err != nil || !id.AuthTime.IsZero() {
		t.Fatalf("sem iat nem auth_time: %v %v", id.AuthTime, err)
	}
}
