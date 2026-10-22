-- +goose Up
-- Remove o catálogo e o blog da PaperMoon / Nexus Corporation, que a
-- 000145 só arquivou. O SISMARQ é o sistema do Arquivo Público
-- (Lei 10.301/2019): ficam só os serviços e as notícias institucionais.

DELETE FROM catalog_services
WHERE slug NOT IN (
    'consulta-documentos-publicos',
    'consulta-tabela-temporalidade',
    'pedido-desentranhamento-edital',
    'autorizacao-eliminacao-documentos',
    'eliminacao-fora-da-ttdd',
    'aprovacao-plano-classificacao-ttdd',
    'transferencia-arquivo-intermediario',
    'emprestimo-documentos',
    'recolhimento-arquivo-publico',
    'orientacao-tecnica-gestao-documental',
    'capacitacao-gestao-documentos',
    'declaracao-arquivo-privado',
    'assistencia-arquivos-privados',
    'alienacao-arquivo-privado'
);

DELETE FROM blog_posts
WHERE slug NOT IN (
    'lei-10301-cria-arquivo-publico-e-sismarq',
    'tabelas-de-temporalidade-publicadas',
    'as-tres-idades-dos-documentos',
    'como-funciona-a-eliminacao-de-documentos',
    'comissoes-de-avaliacao-ccpad-e-cspad',
    'classificacao-pelo-metodo-duplex',
    'sismarq-e-sei-papeis-complementares',
    'digitalizar-nao-e-descartar',
    'arquivos-privados-de-interesse-publico',
    'direito-de-acesso-aos-documentos-publicos'
);

DELETE FROM sismarq_conteudo_anterior;

-- +goose Down
-- O material PaperMoon não volta: não é conteúdo do Arquivo Público.
SELECT 1;
