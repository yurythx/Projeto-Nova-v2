-- +goose Up
-- Importação do GLPI (ADR 072).

-- Conta de sistema dos chamados importados cuja pessoa não existe aqui.
-- Inativa: ninguém entra com ela.
INSERT INTO users (id, keycloak_subject, username, email, display_name, active)
VALUES ('696d706f-7274-4163-a000-676c70690000', 'sistema:importacao', 'sistema.importacao',
        'importacao@sistema.invalid', 'Importação', false)
ON CONFLICT (id) DO NOTHING;

-- De onde veio cada coisa importada: rodar de novo pula o que já entrou.
CREATE TABLE atendimento_importacao (
    origem TEXT NOT NULL CHECK (origem IN ('glpi')),
    tipo TEXT NOT NULL CHECK (tipo IN ('oferta', 'solicitacao')),
    id_origem TEXT NOT NULL CHECK (length(id_origem) BETWEEN 1 AND 80),
    destino UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (origem, tipo, id_origem)
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_importacao;
-- A conta fica se ainda assina algum chamado; senão, sai.
DELETE FROM users WHERE id = '696d706f-7274-4163-a000-676c70690000'
    AND NOT EXISTS (SELECT 1 FROM atendimento_solicitacoes WHERE solicitante_id = '696d706f-7274-4163-a000-676c70690000');
