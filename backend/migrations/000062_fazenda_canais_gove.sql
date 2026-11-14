-- +goose Up
-- Protocolo da Fazenda (ADR 084).
-- Gove entra como terceiro provedor de WhatsApp, um ativo por vez.
-- A conexão de integrações ganha dono (ti ou sefaz) e o tipo da consulta
-- imobiliária, que não extrai chamado.

ALTER TABLE canais_whatsapp DROP CONSTRAINT IF EXISTS canais_whatsapp_provedor_check;
ALTER TABLE canais_whatsapp ADD CONSTRAINT canais_whatsapp_provedor_check
    CHECK (provedor IN ('meta', 'evolution', 'gove'));
ALTER TABLE canais_whatsapp
    ADD COLUMN gove_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN gove_client_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN gove_segredo_cifrado TEXT NOT NULL DEFAULT '',
    ADD COLUMN gove_modelo TEXT NOT NULL DEFAULT '';

ALTER TABLE integracoes_conexoes DROP CONSTRAINT IF EXISTS integracoes_conexoes_tipo_check;
ALTER TABLE integracoes_conexoes ADD CONSTRAINT integracoes_conexoes_tipo_check
    CHECK (tipo IN ('glpi', 'consulta_imobiliaria'));
ALTER TABLE integracoes_conexoes
    ADD COLUMN dono TEXT NOT NULL DEFAULT 'ti' CHECK (dono IN ('ti', 'sefaz'));

-- +goose Down
ALTER TABLE integracoes_conexoes DROP COLUMN IF EXISTS dono;
ALTER TABLE integracoes_conexoes DROP CONSTRAINT IF EXISTS integracoes_conexoes_tipo_check;
ALTER TABLE integracoes_conexoes ADD CONSTRAINT integracoes_conexoes_tipo_check
    CHECK (tipo IN ('glpi'));

ALTER TABLE canais_whatsapp
    DROP COLUMN IF EXISTS gove_modelo,
    DROP COLUMN IF EXISTS gove_segredo_cifrado,
    DROP COLUMN IF EXISTS gove_client_id,
    DROP COLUMN IF EXISTS gove_url;
ALTER TABLE canais_whatsapp DROP CONSTRAINT IF EXISTS canais_whatsapp_provedor_check;
ALTER TABLE canais_whatsapp ADD CONSTRAINT canais_whatsapp_provedor_check
    CHECK (provedor IN ('meta', 'evolution'));
