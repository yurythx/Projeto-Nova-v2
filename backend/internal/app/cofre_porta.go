package app

import (
	"context"
	"errors"

	"github.com/google/uuid"

	atendimentodomain "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	cofreapp "github.com/yurythx/projeto-nova-v2/internal/modules/cofre/application"
	cofredomain "github.com/yurythx/projeto-nova-v2/internal/modules/cofre/domain"
	signumdomain "github.com/yurythx/projeto-nova-v2/internal/modules/signum/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// confirmaSenha traduz a reautenticação do Signum para o cofre.
// A senha da pessoa não é gravada.
type confirmaSenha struct {
	reauth signumdomain.Reauthenticator
}

func (c confirmaSenha) Confirmar(ctx context.Context, userID uuid.UUID, username string, federada bool, senha string) error {
	if c.reauth == nil {
		return cofredomain.ErrSemConfirmacao
	}
	_, err := c.reauth.Reauthenticate(ctx, userID, username, federada, senha)
	if errors.Is(err, signumdomain.ErrReauth) {
		return cofredomain.ErrSenhaRecusada
	}
	return err
}

// portaSegredo deixa o atendimento perguntar o nome de um segredo.
// O cofre desligado ou fora do alcance não entrega senha — nem o nome.
type portaSegredo struct {
	svc   *cofreapp.Service
	ativo func() bool
}

func (p portaSegredo) Nome(ctx context.Context, identity auth.Identity, id uuid.UUID) (string, error) {
	if p.svc == nil || p.ativo == nil || !p.ativo() {
		return "", atendimentodomain.ErrCofreDesligado
	}
	return p.svc.Nome(ctx, identity, id)
}
