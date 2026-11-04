-- +goose Up
-- Abertura de chamado pelo WhatsApp (ADR 078). O portal, com login,
-- continua sendo a outra porta. O roteiro fica com quem escreveu até o
-- chamado nascer pelas mesmas regras. A mensagem entra uma vez só, mesmo
-- antes de existir chamado.
CREATE TABLE atendimento_whatsapp_roteiro (
    usuario_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    etapa TEXT NOT NULL CHECK (etapa IN ('oferta', 'titulo', 'descricao', 'campo', 'feito')),
    oferta_id UUID REFERENCES atendimento_ofertas (id) ON DELETE CASCADE,
    titulo TEXT NOT NULL DEFAULT '' CHECK (char_length(titulo) <= 160),
    descricao TEXT NOT NULL DEFAULT '' CHECK (char_length(descricao) <= 4000),
    respostas JSONB NOT NULL DEFAULT '{}',
    campo_indice INT NOT NULL DEFAULT 0 CHECK (campo_indice >= 0),
    pagina INT NOT NULL DEFAULT 0 CHECK (pagina >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE atendimento_whatsapp_mensagens (
    message_id TEXT PRIMARY KEY CHECK (char_length(message_id) BETWEEN 1 AND 200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_whatsapp_mensagens;
DROP TABLE IF EXISTS atendimento_whatsapp_roteiro;
