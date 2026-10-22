-- +goose Up
-- Ritos especiais da eliminação (arts. 35 e 36), autorização da SEMAD,
-- cópia do Termo (art. 38, parágrafo único) e estado da assinatura Signum
-- nas atas/termo (sempre AUSENTE até a política 0.6 — ADR 041).

ALTER TABLE acervo_listagens
    ADD COLUMN rito TEXT NOT NULL DEFAULT 'ORDINARIO'
        CHECK (rito IN ('ORDINARIO', 'ART_35', 'ART_36')),
    ADD COLUMN ccpad_autorizada_por UUID,
    ADD COLUMN ccpad_autorizada_em TIMESTAMPTZ,
    ADD COLUMN semad_autorizada_por UUID,
    ADD COLUMN semad_autorizada_em TIMESTAMPTZ,
    ADD COLUMN ata_signum TEXT NOT NULL DEFAULT 'AUSENTE' CHECK (ata_signum = 'AUSENTE'),
    ADD COLUMN termo_signum TEXT NOT NULL DEFAULT 'AUSENTE' CHECK (termo_signum = 'AUSENTE'),
    ADD COLUMN termo_copia_em TIMESTAMPTZ,
    ADD COLUMN termo_copia_arquivo INTEGER NOT NULL DEFAULT 0 CHECK (termo_copia_arquivo >= 0),
    ADD COLUMN termo_copia_semad INTEGER NOT NULL DEFAULT 0 CHECK (termo_copia_semad >= 0);

ALTER TABLE acervo_listagens DROP CONSTRAINT acervo_listagens_situacao_check;
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_situacao_check
    CHECK (situacao IN ('RASCUNHO', 'APROVADA', 'AUTORIZADA', 'AGUARDANDO_SEMAD', 'EM_EDITAL', 'ELIMINADA', 'CANCELADA'));

ALTER TABLE acervo_listagens DROP CONSTRAINT acervo_listagens_check1;
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_check1
    CHECK (situacao NOT IN ('AUTORIZADA', 'AGUARDANDO_SEMAD', 'EM_EDITAL', 'ELIMINADA') OR autorizada_em IS NOT NULL);

ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_semad_check
    CHECK (situacao NOT IN ('EM_EDITAL', 'ELIMINADA') OR rito = 'ORDINARIO' OR semad_autorizada_em IS NOT NULL);
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_rito_semad_check
    CHECK (situacao <> 'AGUARDANDO_SEMAD' OR rito IN ('ART_35', 'ART_36'));
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_art35_ccpad_check
    CHECK (rito <> 'ART_35' OR situacao NOT IN ('AUTORIZADA', 'AGUARDANDO_SEMAD', 'EM_EDITAL', 'ELIMINADA') OR ccpad_autorizada_em IS NOT NULL);

-- +goose Down
UPDATE acervo_listagens SET situacao = 'AUTORIZADA' WHERE situacao = 'AGUARDANDO_SEMAD';

ALTER TABLE acervo_listagens DROP CONSTRAINT IF EXISTS acervo_listagens_art35_ccpad_check;
ALTER TABLE acervo_listagens DROP CONSTRAINT IF EXISTS acervo_listagens_rito_semad_check;
ALTER TABLE acervo_listagens DROP CONSTRAINT IF EXISTS acervo_listagens_semad_check;
ALTER TABLE acervo_listagens DROP CONSTRAINT acervo_listagens_check1;
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_check1
    CHECK (situacao NOT IN ('AUTORIZADA', 'EM_EDITAL', 'ELIMINADA') OR autorizada_em IS NOT NULL);
ALTER TABLE acervo_listagens DROP CONSTRAINT acervo_listagens_situacao_check;
ALTER TABLE acervo_listagens ADD CONSTRAINT acervo_listagens_situacao_check
    CHECK (situacao IN ('RASCUNHO', 'APROVADA', 'AUTORIZADA', 'EM_EDITAL', 'ELIMINADA', 'CANCELADA'));

ALTER TABLE acervo_listagens
    DROP COLUMN IF EXISTS termo_copia_semad,
    DROP COLUMN IF EXISTS termo_copia_arquivo,
    DROP COLUMN IF EXISTS termo_copia_em,
    DROP COLUMN IF EXISTS termo_signum,
    DROP COLUMN IF EXISTS ata_signum,
    DROP COLUMN IF EXISTS semad_autorizada_em,
    DROP COLUMN IF EXISTS semad_autorizada_por,
    DROP COLUMN IF EXISTS ccpad_autorizada_em,
    DROP COLUMN IF EXISTS ccpad_autorizada_por,
    DROP COLUMN IF EXISTS rito;
