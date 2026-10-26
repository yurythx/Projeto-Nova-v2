-- +goose Up
-- O endereço e o tipo ficam à vista. A senha continua no envelope.
-- O painel aponta máquinas pelo id. Não copia a senha delas.

ALTER TABLE cofre_segredos
    ADD COLUMN tipo TEXT NOT NULL DEFAULT 'outro',
    ADD COLUMN endereco TEXT NOT NULL DEFAULT '';

ALTER TABLE cofre_segredos
    ADD CONSTRAINT cofre_segredos_tipo_chk
    CHECK (tipo IN ('maquina', 'aplicacao', 'painel', 'outro'));

CREATE TABLE cofre_administra (
    painel_id  UUID NOT NULL REFERENCES cofre_segredos (id) ON DELETE CASCADE,
    maquina_id UUID NOT NULL REFERENCES cofre_segredos (id) ON DELETE CASCADE,
    PRIMARY KEY (painel_id, maquina_id),
    CHECK (painel_id <> maquina_id)
);

CREATE INDEX cofre_administra_maquina_id_idx ON cofre_administra (maquina_id);

-- +goose Down
DROP TABLE cofre_administra;

ALTER TABLE cofre_segredos
    DROP CONSTRAINT cofre_segredos_tipo_chk;

ALTER TABLE cofre_segredos
    DROP COLUMN endereco,
    DROP COLUMN tipo;
