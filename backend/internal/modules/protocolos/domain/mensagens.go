package domain

import (
	"errors"
	"time"
)

// Ações da conversa. As duas aparecem na consulta pública com o texto.
const (
	AcaoMensagemCidadao = "mensagem_cidadao"
	AcaoMensagemMesa    = "mensagem_mesa"
)

// MensagensSemResposta é quantas mensagens o cidadão manda seguidas antes de a mesa responder.
const MensagensSemResposta = 3

// ErrMuitasMensagens recusa a quarta mensagem seguida do cidadão.
var ErrMuitasMensagens = errors.New("protocolos: aguarde a resposta da prefeitura")

// SeguidasDoCidadao conta as mensagens do cidadão no fim da trilha, desde a última da mesa.
// As notas internas e as ações sem texto ao cidadão não interrompem a contagem.
func SeguidasDoCidadao(historico []Historico) int {
	n := 0
	for i := len(historico) - 1; i >= 0; i-- {
		switch historico[i].Acao {
		case AcaoMensagemCidadao:
			n++
		case AcaoMensagemMesa:
			return n
		}
	}
	return n
}

// MensagemDoCidadao marca a novidade para a mesa. Pedido encerrado ou em
// complementação não aceita mensagem: a complementação tem a sua resposta.
func (p Pedido) MensagemDoCidadao(texto string, seguidas int, agora time.Time) (Pedido, string, error) {
	if EstadoFinal(p.Status) {
		return Pedido{}, "", ErrEstadoFinal
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, "", ErrComplemento
	}
	texto, err := TextoInterno(texto, 1000)
	if err != nil {
		return Pedido{}, "", err
	}
	if seguidas >= MensagensSemResposta {
		return Pedido{}, "", ErrMuitasMensagens
	}
	p.MensagemNova = true
	p.UpdatedAt = agora
	return p, texto, nil
}

// MensagemDaMesa responde ao cidadão e apaga a novidade.
func (p Pedido) MensagemDaMesa(texto string, agora time.Time) (Pedido, string, error) {
	if EstadoFinal(p.Status) {
		return Pedido{}, "", ErrEstadoFinal
	}
	texto, err := TextoInterno(texto, 2000)
	if err != nil {
		return Pedido{}, "", err
	}
	p.MensagemNova = false
	p.UpdatedAt = agora
	return p, texto, nil
}
