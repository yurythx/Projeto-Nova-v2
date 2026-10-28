-- +goose Up
-- WhatsApp dos avisos (ADR 078): a API oficial da Meta ou a Evolution
-- API, uma ativa de cada vez. As credenciais de cada uma ficam cifradas à
-- parte: trocar de provedor não apaga a do outro.
CREATE TABLE canais_whatsapp (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    ativo BOOLEAN NOT NULL DEFAULT false,
    provedor TEXT NOT NULL DEFAULT 'meta' CHECK (provedor IN ('meta', 'evolution')),
    meta_numero_id TEXT NOT NULL DEFAULT '' CHECK (meta_numero_id ~ '^[0-9]{0,32}$'),
    meta_token_cifrado TEXT NOT NULL DEFAULT '',
    meta_modelo TEXT NOT NULL DEFAULT '' CHECK (char_length(meta_modelo) <= 512),
    meta_idioma TEXT NOT NULL DEFAULT 'pt_BR' CHECK (char_length(meta_idioma) <= 10),
    evolution_url TEXT NOT NULL DEFAULT '' CHECK (char_length(evolution_url) <= 500),
    evolution_instancia TEXT NOT NULL DEFAULT '' CHECK (char_length(evolution_instancia) <= 100),
    evolution_chave_cifrada TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE IF EXISTS canais_whatsapp;
