package app

import (
	"context"
	"errors"

	atendimentodomain "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/iaconfig"
)

// portaIA liga a IA de quem atende à função atendimento.assistente
// (ADR 065). O roteador escolhe a conexão e mascara os dados pessoais
// quando o fornecedor é externo.
type portaIA struct {
	roteador *iaconfig.Roteador
}

func (p portaIA) Disponivel(ctx context.Context) (bool, error) {
	return p.roteador.Disponivel(ctx, iaconfig.FuncaoAtendimento)
}

// Perguntar usa temperatura baixa: classificação e resumo pedem a mesma
// resposta para o mesmo chamado, não criatividade.
func (p portaIA) Perguntar(ctx context.Context, sistema, pergunta string, maxTokens int) (string, string, error) {
	r, err := p.roteador.Conversar(ctx, iaconfig.FuncaoAtendimento, iaconfig.Pedido{
		Sistema: sistema, Pergunta: pergunta, Montar: func(q string) string { return q },
		Temperatura: 0.1, MaxTokens: maxTokens,
	})
	if errors.Is(err, iaconfig.ErrSemConexao) {
		return "", "", atendimentodomain.ErrSemIA
	}
	return r.Texto, r.Conexao, err
}
