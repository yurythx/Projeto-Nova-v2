-- +goose Up
-- A rotação troca a senha guardada no cofre. Não entra na máquina.
-- O aviso pendente fica à vista até alguém aplicar no endereço.

ALTER TABLE cofre_segredos
    ADD COLUMN rotaciona BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN rotacao_pendente BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN proxima_rotacao TIMESTAMPTZ;

CREATE INDEX cofre_segredos_proxima_rotacao_idx
    ON cofre_segredos (proxima_rotacao)
    WHERE rotaciona AND NOT rotacao_pendente AND proxima_rotacao IS NOT NULL;

-- +goose Down
DROP INDEX cofre_segredos_proxima_rotacao_idx;

ALTER TABLE cofre_segredos
    DROP COLUMN proxima_rotacao,
    DROP COLUMN rotacao_pendente,
    DROP COLUMN rotaciona;
