-- +goose Up
-- Canais de comunicação (ADR 078). O e-mail sai pelo SMTP configurado na
-- tela (uma linha só); a variável de ambiente vale só enquanto ninguém o
-- configurou. A senha fica cifrada (secretcrypto).
CREATE TABLE canais_email (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    ativo BOOLEAN NOT NULL DEFAULT false,
    host TEXT NOT NULL DEFAULT '' CHECK (char_length(host) <= 255),
    porta INTEGER NOT NULL DEFAULT 587 CHECK (porta BETWEEN 1 AND 65535),
    usuario TEXT NOT NULL DEFAULT '' CHECK (char_length(usuario) <= 255),
    senha_cifrada TEXT NOT NULL DEFAULT '',
    remetente TEXT NOT NULL DEFAULT '' CHECK (char_length(remetente) <= 255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL DEFAULT ''
);

-- O contato que a pessoa deixa para receber os avisos (no SSO quase
-- ninguém tem e-mail). Deixar o contato é o consentimento (LGPD); a data
-- de resposta também guarda quem preferiu não deixar, para não perguntar
-- de novo.
ALTER TABLE users
    ADD COLUMN contato_email TEXT NOT NULL DEFAULT '' CHECK (char_length(contato_email) <= 254),
    ADD COLUMN contato_whatsapp TEXT NOT NULL DEFAULT '' CHECK (contato_whatsapp ~ '^(\+[1-9][0-9]{7,14})?$'),
    ADD COLUMN contato_respondido_em TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS contato_respondido_em,
    DROP COLUMN IF EXISTS contato_whatsapp,
    DROP COLUMN IF EXISTS contato_email;
DROP TABLE IF EXISTS canais_email;
