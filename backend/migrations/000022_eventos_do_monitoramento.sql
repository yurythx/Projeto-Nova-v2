-- +goose Up
-- Alertas do monitoramento viram incidente (ADR 067).

-- Conta de sistema que aparece como quem abriu. Inativa e sem senha: não
-- entra por login local, e o "subject" não é um id do Keycloak (que são
-- UUIDs), então nenhum token a representa. O id é o de
-- domain.ContaMonitoramento.
INSERT INTO users (id, keycloak_subject, username, email, display_name, active)
VALUES ('6d6f6e69-746f-4072-a000-6d656e746f00', 'sistema:monitoramento', 'sistema.monitoramento',
        'monitoramento@sistema.invalid', 'Monitoramento', false)
ON CONFLICT (id) DO NOTHING;

-- A oferta em que os alertas abrem incidente (uma linha só).
CREATE TABLE atendimento_eventos_config (
    unico BOOLEAN PRIMARY KEY DEFAULT true CHECK (unico),
    oferta_id UUID REFERENCES atendimento_ofertas(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Um alerta (fingerprint) aponta para o chamado aberto por ele.
CREATE TABLE atendimento_eventos (
    fingerprint TEXT PRIMARY KEY CHECK (length(fingerprint) BETWEEN 1 AND 128),
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    alerta TEXT NOT NULL,
    ultimo_disparo TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_eventos;
DROP TABLE IF EXISTS atendimento_eventos_config;
-- A conta fica se já abriu algum chamado: apagar levaria o histórico.
DELETE FROM users u WHERE u.id = '6d6f6e69-746f-4072-a000-6d656e746f00'
    AND NOT EXISTS (SELECT 1 FROM atendimento_solicitacoes s WHERE s.solicitante_id = u.id);
