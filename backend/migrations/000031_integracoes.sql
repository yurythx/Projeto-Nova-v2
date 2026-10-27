-- +goose Up
-- Módulo de integrações (ADR 076): migrar de outra ferramenta pela tela.
-- Extrair para a área de preparo, revisar a tabela de-para e carregar.

-- A conexão com a origem. No modo direto, as credenciais ficam cifradas
-- (secretcrypto); no modo conector, ficam com o conector, e aqui só o hash
-- do token de máquina que ele usa.
CREATE TABLE integracoes_conexoes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo TEXT NOT NULL CHECK (tipo IN ('glpi')),
    nome TEXT NOT NULL CHECK (length(btrim(nome)) BETWEEN 1 AND 160),
    modo TEXT NOT NULL CHECK (modo IN ('direto', 'conector')),
    url TEXT NOT NULL DEFAULT '' CHECK (length(url) <= 500),
    credenciais TEXT NOT NULL DEFAULT '',
    perfil TEXT NOT NULL DEFAULT '' CHECK (length(perfil) <= 80),
    fuso TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    token_hash TEXT UNIQUE,
    token_em TIMESTAMPTZ,
    extraido_em TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL DEFAULT ''
);

-- A área de preparo: o dado bruto da origem, um registro por item. Os
-- campos fora do JSON são o que o inventário agrupa sem abrir o JSON.
CREATE TABLE integracoes_itens (
    conexao_id UUID NOT NULL REFERENCES integracoes_conexoes(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL CHECK (tipo IN ('chamado', 'pessoa', 'categoria', 'entidade', 'sla')),
    id_origem INTEGER NOT NULL,
    versao TEXT NOT NULL DEFAULT '',
    nome TEXT NOT NULL DEFAULT '',
    entidade INTEGER,
    categoria INTEGER,
    sla INTEGER,
    minutos INTEGER,
    classe TEXT NOT NULL DEFAULT '',
    data TEXT NOT NULL DEFAULT '',
    dados JSONB NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (conexao_id, tipo, id_origem)
);

-- A tabela de-para (ADR 075) de cada conexão. categoria -1 é o "*".
CREATE TABLE integracoes_depara (
    conexao_id UUID NOT NULL REFERENCES integracoes_conexoes(id) ON DELETE CASCADE,
    secretaria TEXT NOT NULL CHECK (length(btrim(secretaria)) BETWEEN 1 AND 160),
    categoria INTEGER NOT NULL CHECK (categoria >= -1),
    categoria_nome TEXT NOT NULL DEFAULT '',
    chamados INTEGER NOT NULL DEFAULT 0,
    com_prazo INTEGER NOT NULL DEFAULT 0,
    nivel TEXT NOT NULL DEFAULT '',
    unidade_id UUID REFERENCES unidades(id) ON DELETE SET NULL,
    oferta TEXT NOT NULL CHECK (length(btrim(oferta)) BETWEEN 1 AND 160),
    tipo TEXT NOT NULL CHECK (tipo IN ('incidente', 'requisicao')),
    prioridade TEXT NOT NULL CHECK (prioridade IN ('critica', 'alta', 'media', 'baixa')),
    PRIMARY KEY (conexao_id, secretaria, categoria)
);

-- As tarefas: extrair, simular e carregar, uma por conexão de cada vez.
-- batida_em é o sinal de vida de quem executa: parada há muito, a tarefa
-- volta a pendente e recomeça (a carga é idempotente).
CREATE TABLE integracoes_tarefas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conexao_id UUID NOT NULL REFERENCES integracoes_conexoes(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL CHECK (tipo IN ('extracao', 'simulacao', 'carga')),
    estado TEXT NOT NULL DEFAULT 'pendente' CHECK (estado IN ('pendente', 'em_andamento', 'concluida', 'falhou', 'cancelada')),
    feitos INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    relatorio JSONB,
    erro TEXT NOT NULL DEFAULT '',
    pedida_por TEXT NOT NULL DEFAULT '',
    criada_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    iniciada_em TIMESTAMPTZ,
    batida_em TIMESTAMPTZ,
    terminada_em TIMESTAMPTZ
);
CREATE UNIQUE INDEX integracoes_tarefa_ativa ON integracoes_tarefas (conexao_id) WHERE estado IN ('pendente', 'em_andamento');
CREATE INDEX integracoes_tarefas_conexao ON integracoes_tarefas (conexao_id, criada_em DESC);

-- +goose Down
DROP TABLE IF EXISTS integracoes_tarefas;
DROP TABLE IF EXISTS integracoes_depara;
DROP TABLE IF EXISTS integracoes_itens;
DROP TABLE IF EXISTS integracoes_conexoes;
