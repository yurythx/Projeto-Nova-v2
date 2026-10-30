-- +goose Up
-- Oferta oficial de incidente para alertas do monitoramento (Prometheus Alertmanager - ADR 067).
INSERT INTO atendimento_ofertas (id, nome, descricao, tipo, ativa, icone, ordem, unidade_id, categoria_id)
VALUES (
    '6d6f6e69-746f-4072-a000-6f6665727461'::uuid,
    'Alerta do Monitoramento',
    'Abertura automática de incidente gerado por alertas de infraestrutura e serviços (Alertmanager / Prometheus).',
    'incidente',
    true,
    'alert-triangle',
    999,
    COALESCE(
        (SELECT id FROM unidades WHERE sigla = 'PREFEITURA' LIMIT 1),
        (SELECT id FROM unidades LIMIT 1)
    ),
    (SELECT id FROM atendimento_categorias LIMIT 1)
)
ON CONFLICT (id) DO UPDATE
SET tipo = 'incidente', ativa = true;

INSERT INTO atendimento_eventos_config (unico, oferta_id, updated_at)
VALUES (
    true,
    '6d6f6e69-746f-4072-a000-6f6665727461'::uuid,
    now()
)
ON CONFLICT (unico) DO UPDATE
SET oferta_id = EXCLUDED.oferta_id, updated_at = now();

-- +goose Down
UPDATE atendimento_eventos_config SET oferta_id = NULL WHERE unico = true;
DELETE FROM atendimento_ofertas WHERE id = '6d6f6e69-746f-4072-a000-6f6665727461'::uuid;
