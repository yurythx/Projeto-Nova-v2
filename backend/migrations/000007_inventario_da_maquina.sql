-- +goose Up
-- Medida da própria máquina. Nulo significa que ninguém mediu: o
-- cadastro manual permanece sem inventário inventado.

ALTER TABLE ativos ADD COLUMN inventario JSONB;

-- +goose Down
ALTER TABLE ativos DROP COLUMN inventario;
