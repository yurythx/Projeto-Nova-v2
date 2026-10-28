-- +goose Up
-- A categoria do catálogo de cada linha da tabela de-para (ADR 078): no
-- GLPI a categoria é uma árvore, e a raiz dela agrupa as ofertas na
-- central do solicitante. Vazio: a oferta fica em "Outros".
ALTER TABLE integracoes_depara
    ADD COLUMN catalogo TEXT NOT NULL DEFAULT '' CHECK (char_length(catalogo) <= 80);

-- +goose Down
ALTER TABLE integracoes_depara DROP COLUMN IF EXISTS catalogo;
