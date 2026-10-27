package app

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"

	atendimentodomain "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	contratosapp "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	contratosdomain "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// maxRotuloContrato é o teto do rótulo copiado no chamado.
const maxRotuloContrato = 400

// portaContratos deixa o atendimento apontar o contrato de apoio (ADR 071):
// os vigentes que servem às unidades que a pessoa atende, e os
// institucionais. Com o plug-in Contratos desligado, não há contrato.
type portaContratos struct {
	svc   *contratosapp.Service
	ativo func() bool
}

func (p portaContratos) Listar(ctx context.Context, identity auth.Identity) ([]atendimentodomain.ContratoDeApoio, error) {
	if p.svc == nil || p.ativo == nil || !p.ativo() {
		return nil, atendimentodomain.ErrContratosDesligado
	}
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	lista, err := p.svc.Vigentes(ctx, cob.Unidades, cob.All)
	if err != nil {
		return nil, err
	}
	out := make([]atendimentodomain.ContratoDeApoio, 0, len(lista))
	for _, c := range lista {
		out = append(out, atendimentodomain.ContratoDeApoio{ID: c.ID, Rotulo: rotuloContrato(c), Fornecedor: c.FornecedorNome, PrazoHoras: c.PrazoHoras})
	}
	return out, nil
}

func (p portaContratos) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (atendimentodomain.ContratoDeApoio, error) {
	lista, err := p.Listar(ctx, identity)
	if err != nil {
		return atendimentodomain.ContratoDeApoio{}, err
	}
	for _, c := range lista {
		if c.ID == id {
			return c, nil
		}
	}
	return atendimentodomain.ContratoDeApoio{}, atendimentodomain.ErrContratoAusente
}

// rotuloContrato é o que o chamado mostra e guarda: número, título e
// fornecedor, cortado no teto da coluna.
func rotuloContrato(c contratosdomain.Contrato) string {
	r := c.Titulo
	if c.Numero != "" {
		r = c.Numero + " — " + r
	}
	r += " (" + c.FornecedorNome + ")"
	if utf8.RuneCountInString(r) > maxRotuloContrato {
		r = string([]rune(r)[:maxRotuloContrato])
	}
	return r
}
