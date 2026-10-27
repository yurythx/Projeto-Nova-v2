package app

import (
	"context"
	"errors"

	iamapp "github.com/yurythx/projeto-nova-v2/internal/modules/iam/application"
	iamdomain "github.com/yurythx/projeto-nova-v2/internal/modules/iam/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/keycloakadmin"
)

// portaSSO liga a sincronização do IAM (ADR 077) à API de administração do
// Keycloak, com o client e o segredo da configuração do SSO (a da tela,
// ou o .env).
type portaSSO struct {
	cli         *keycloakadmin.Cliente
	credenciais func(ctx context.Context) (issuer, client, secret string)
}

var _ iamapp.DiretorioSSO = portaSSO{}

func (p portaSSO) Usuarios(ctx context.Context) ([]iamdomain.UsuarioSSO, error) {
	issuer, client, secret := p.credenciais(ctx)
	pessoas, err := p.cli.Usuarios(ctx, keycloakadmin.Config{IssuerURL: issuer, ClientID: client, ClientSecret: secret})
	if errors.Is(err, keycloakadmin.ErrNaoConfigurado) {
		return nil, iamapp.ErrSemSSO
	}
	if err != nil {
		return nil, err
	}
	out := make([]iamdomain.UsuarioSSO, len(pessoas))
	for i, u := range pessoas {
		out[i] = iamdomain.UsuarioSSO{Subject: u.ID, Username: u.Username, Email: u.Email, Nome: u.Nome(), Ativo: u.Enabled, Grupos: u.Grupos}
	}
	return out, nil
}
