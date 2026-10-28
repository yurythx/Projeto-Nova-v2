-- +goose Up
-- Andamento da sincronização com o SSO (ADR 077). Num realm federado ao AD
-- ela leva horas: cada página lida atualiza a linha, e a tela da API
-- acompanha a que roda no worker. Sem batida há 20 minutos, a linha é de
-- um processo que caiu e não segura a próxima.
CREATE TABLE iam_sso_sincronizacoes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iniciada_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    batida_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminada_em TIMESTAMPTZ,
    relatorio JSONB NOT NULL DEFAULT '{}',
    erro TEXT NOT NULL DEFAULT ''
);
CREATE INDEX iam_sso_sincronizacoes_recentes ON iam_sso_sincronizacoes (iniciada_em DESC);

-- +goose Down
DROP TABLE IF EXISTS iam_sso_sincronizacoes;
