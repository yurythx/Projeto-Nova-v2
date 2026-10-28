package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento"
	"github.com/yurythx/projeto-nova-v2/internal/platform/canais"
)

// portaWhatsApp entrega ao atendimento a resposta que chegou pelo WhatsApp
// (ADR 078). Com o plug-in desligado, a mensagem não entra.
type portaWhatsApp struct {
	mod   *atendimento.Module
	ativo func() bool
}

var _ canais.Recebedor = portaWhatsApp{}

func (p portaWhatsApp) ReceberWhatsApp(ctx context.Context, usuario uuid.UUID, texto, mensagemID string) (string, error) {
	if !p.ativo() {
		return "", nil
	}
	return p.mod.ReceberWhatsApp(ctx, usuario, texto, mensagemID)
}
