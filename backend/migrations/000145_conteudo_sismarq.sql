-- +goose Up
-- Conteúdo público do SISMARQ (docs/ROADMAP_SISMARQ.md): a carta de serviços e
-- as publicações passam a tratar só do Arquivo Público Municipal, com base em
-- docs/SISMARQ (Lei 10.301/2019, TTDD publicada e artigo técnico). O conteúdo
-- anterior (serviços de TI e artigos sobre eles) é ARQUIVADO, não apagado: o
-- estado anterior fica em sismarq_conteudo_anterior e o Down o restaura.

CREATE TABLE sismarq_conteudo_anterior (
    tabela TEXT NOT NULL,
    id     UUID NOT NULL,
    status TEXT NOT NULL,
    pinned BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (tabela, id)
);

INSERT INTO sismarq_conteudo_anterior (tabela, id, status)
SELECT 'catalog_services', id, status FROM catalog_services
WHERE status <> 'archived' AND slug NOT IN ('consulta-documentos-publicos', 'consulta-tabela-temporalidade', 'pedido-desentranhamento-edital', 'autorizacao-eliminacao-documentos', 'eliminacao-fora-da-ttdd', 'aprovacao-plano-classificacao-ttdd', 'transferencia-arquivo-intermediario', 'emprestimo-documentos', 'recolhimento-arquivo-publico', 'orientacao-tecnica-gestao-documental', 'capacitacao-gestao-documentos', 'declaracao-arquivo-privado', 'assistencia-arquivos-privados', 'alienacao-arquivo-privado');

UPDATE catalog_services SET status = 'archived'
WHERE id IN (SELECT id FROM sismarq_conteudo_anterior WHERE tabela = 'catalog_services');

INSERT INTO sismarq_conteudo_anterior (tabela, id, status, pinned)
SELECT 'blog_posts', id, status, pinned FROM blog_posts
WHERE status <> 'archived' AND slug NOT IN ('lei-10301-cria-arquivo-publico-e-sismarq', 'tabelas-de-temporalidade-publicadas', 'as-tres-idades-dos-documentos', 'como-funciona-a-eliminacao-de-documentos', 'comissoes-de-avaliacao-ccpad-e-cspad', 'classificacao-pelo-metodo-duplex', 'sismarq-e-sei-papeis-complementares', 'digitalizar-nao-e-descartar', 'arquivos-privados-de-interesse-publico', 'direito-de-acesso-aos-documentos-publicos');

UPDATE blog_posts SET status = 'archived', pinned = false
WHERE id IN (SELECT id FROM sismarq_conteudo_anterior WHERE tabela = 'blog_posts');

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('consulta-documentos-publicos', 'Consulta a documentos públicos municipais', 'Acesso pleno aos documentos públicos municipais, de forma ágil e transparente, ressalvados o sigilo legal e a proteção da intimidade.', '## Sobre o serviço

A Lei Municipal nº 10.301/2019 assegura ao cidadão o **direito de acesso pleno aos documentos públicos municipais**, com consulta franqueada de forma ágil e transparente (art. 2º). Garantir esse acesso, de forma segura e resguardados o sigilo e as restrições legais, é a primeira finalidade do SISMARQ (art. 7º, I).

## Restrições

Ficam ressalvados os documentos cujo sigilo seja imprescindível à segurança da sociedade e do Estado e os que envolvem a inviolabilidade da intimidade, da vida privada, da honra e da imagem das pessoas (art. 2º).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 2º, 5º, X, e 7º, I
- Constituição Federal, art. 216, § 2º', 'Acesso à informação', 'Cidadãos, pesquisadores e órgãos da administração municipal',
    ARRAY['Identificação do documento, do assunto ou do período de interesse']::text[], ARRAY['Solicitar a consulta ao Arquivo Público Municipal', 'Análise de eventual restrição de acesso (art. 2º)', 'Disponibilização do documento para consulta']::text[], '[]'::jsonb, '', 'Gratuito', 'book-open', 'published', 1, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('consulta-tabela-temporalidade', 'Consulta à Tabela de Temporalidade (TTDD)', 'Consulta pública às Tabelas de Temporalidade e Destinação de Documentos publicadas no DIORONDON-e: prazos de guarda e destinação de cada série.', '## Sobre o serviço

A Tabela de Temporalidade e Destinação de Documentos (TTDD) é o instrumento, aprovado pela autoridade competente, que **define os prazos de guarda e a destinação** dos documentos (art. 27). Para cada série, indica quantos anos o documento fica no arquivo corrente e no intermediário (art. 29) e se ele será eliminado ou guardado permanentemente (art. 28).

As tabelas das secretarias foram aprovadas pela Comissão Central Permanente de Avaliação de Documentos (CCPAD) e publicadas no Diário Oficial do Município (DIORONDON-e), nas edições nº 6.017 (25/08/2025), nº 6.111 (13/01/2026), nº 6.275 (11/09/2026) e nº 6.281 (21/09/2026).

## Como consultar

A consulta é pública e não exige login, na página **Temporalidade** deste portal (/temporalidade), com busca por código ou assunto.

## Base legal

- Lei Municipal nº 10.301/2019, arts. 22 e 27 a 30', 'Acesso à informação', 'Cidadãos e servidores municipais',
    '{}'::text[], ARRAY['Acessar a página Temporalidade (/temporalidade)', 'Buscar pelo código de classificação ou pelo assunto', 'Consultar os prazos, a destinação e as observações da série']::text[], '[]'::jsonb, 'Imediato (consulta on-line)', 'Gratuito', 'calendar-clock', 'published', 2, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('pedido-desentranhamento-edital', 'Pedido de desentranhamento ou cópia (Edital de Ciência de Eliminação)', 'No prazo de 30 dias da publicação do edital, o interessado pode requerer o desentranhamento de documentos ou cópias de peças de processos.', '## Sobre o serviço

Antes de qualquer eliminação, a Comissão Setorial Permanente de Avaliação de Documentos publica no Diário Oficial do Município o **Edital de Ciência de Eliminação de Documentos** (art. 37 e Anexo III). O edital dá publicidade ao ato e abre **30 dias** para manifestações.

Nesse prazo, os interessados podem requerer, às suas expensas, o **desentranhamento de documentos ou cópias de peças** do processo (art. 37, § 2º).

## Base legal

- Lei Municipal nº 10.301/2019, art. 37 e Anexo III', 'Avaliação e eliminação', 'Partes interessadas nos documentos relacionados num Edital de Ciência de Eliminação',
    ARRAY['Qualificação do requerente', 'Demonstração da legitimidade do pedido', 'Indicação do número do edital e dos documentos de interesse']::text[], ARRAY['Apresentar petição à Comissão Permanente de Avaliação do órgão indicado no edital', 'Protocolar dentro de 30 dias da publicação do edital', 'Retirar os documentos ou as cópias deferidos']::text[], '[]'::jsonb, '30 dias a partir da publicação do edital', 'Às expensas do requerente (Anexo III)', 'file-x', 'published', 3, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('autorizacao-eliminacao-documentos', 'Autorização para eliminação de documentos', 'Nenhum documento é eliminado sem autorização prévia do Arquivo Público Municipal e sem o rito da Lei 10.301/2019, com os modelos dos Anexos I a IV.', '## Sobre o serviço

É proibida a eliminação de documentos da administração municipal **sem autorização prévia do Arquivo Público Municipal** (art. 65). Só pode ser eliminado o que está **previsto na Tabela de Temporalidade**, depois de cumprido o prazo de guarda (arts. 19 e 30, parágrafo único).

## O rito

1. A CSPAD avalia os documentos e confere a existência de ações judiciais (arts. 32, II, e 44).
2. A **Relação de Eliminação de Documentos** (Anexo I) é preenchida e assinada (art. 34).
3. A CSPAD aprova a listagem em reunião registrada na **Ata** (Anexo IV, art. 33).
4. O titular do Arquivo Público **autoriza** a eliminação (Anexo I).
5. O **Edital de Ciência** (Anexo III) é publicado no Diário Oficial do Município, com prazo de 30 dias (art. 37).
6. Uma **amostragem** dos documentos passa à guarda permanente (art. 39).
7. A eliminação é feita por fragmentação, e o papel é doado (art. 40). O **Termo de Eliminação** (Anexo II) registra o ato, e uma cópia vai à Divisão de Arquivo Público e à Secretaria de Administração (art. 38).

Documentos de guarda permanente **nunca** são eliminados, nem depois de digitalizados (arts. 20 e 42).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 19, 33 a 40, 42 e 65 e Anexos I a IV
- Resolução CONARQ nº 7/1997', 'Avaliação e eliminação', 'Órgãos e entidades da administração municipal, por meio da sua CSPAD',
    ARRAY['Relação de Eliminação de Documentos (Anexo I) assinada pelo responsável pela seleção, pelo presidente da CSPAD e pelo gestor do órgão', 'Ata da reunião da CSPAD (Anexo IV) com a lista de presença', 'Parecer da assessoria jurídica do órgão (art. 32, § 4º)', 'Comprovação da aprovação das contas pelo Tribunal de Contas, quando exigida (Anexo I)']::text[], ARRAY['Avaliação pela CSPAD', 'Relação de Eliminação (Anexo I)', 'Ata da CSPAD (Anexo IV)', 'Autorização do Arquivo Público', 'Edital de Ciência (Anexo III) e prazo de 30 dias', 'Amostragem (art. 39)', 'Eliminação e Termo de Eliminação (Anexo II)']::text[], '[]'::jsonb, '', 'Gratuito', 'scale', 'published', 4, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('eliminacao-fora-da-ttdd', 'Eliminação de documentos não previstos na TTDD', 'Rito especial para documentos que não constam da Tabela de Temporalidade ou de órgãos que ainda não oficializaram a sua.', '## Sobre o serviço

Toda eliminação de documentos que **não constem da Tabela de Temporalidade** das atividades-meio ou das atividades-fim depende de autorização da **Comissão Central de Avaliação de Documentos**, do **Gerente da Divisão de Arquivo Público** e da **Secretaria Municipal de Administração** (art. 35).

O órgão que **ainda não oficializou** a sua TTDD de atividades-fim encaminha a Relação de Eliminação à Comissão Central e ao Gerente da Divisão de Arquivo Público. Depois da análise, ela segue à Secretaria Municipal de Administração para aprovação (art. 36).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 35 e 36', 'Avaliação e eliminação', 'Órgãos e entidades da administração municipal',
    ARRAY['Relação de Eliminação de Documentos (Anexo I)']::text[], ARRAY['Encaminhar a Relação de Eliminação à Comissão Central e à Divisão de Arquivo Público', 'Análise da Comissão Central e da Divisão', 'Aprovação pela Secretaria Municipal de Administração', 'Seguir o rito do edital, da amostragem e do termo']::text[], '[]'::jsonb, '', 'Gratuito', 'file-search', 'published', 5, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('aprovacao-plano-classificacao-ttdd', 'Aprovação de Plano de Classificação e Tabela de Temporalidade', 'Análise pela Comissão Central e aprovação dos Planos de Classificação e das Tabelas de Temporalidade das atividades-fim de cada órgão.', '## Sobre o serviço

Cada secretaria e órgão deve elaborar o seu **Plano de Classificação** e a sua **Tabela de Temporalidade** das atividades-fim, com os órgãos produtores, as espécies documentais, os prazos de guarda, a destinação e a fundamentação legal ou administrativa (art. 28).

As CSPADs encaminham os instrumentos ao Arquivo Público antes da oficialização, para análise da **Comissão Central Permanente de Avaliação de Documentos** (art. 31, § 4º). A Comissão Central examina a **consistência técnico-arquivística** do trabalho, não os prazos estipulados (art. 32, § 6º). Ela devolve um parecer conclusivo com sugestões ou correções (§ 7º). Depois de aprovados, os instrumentos são submetidos ao titular da Secretaria Municipal de Administração e publicados no Diário Oficial do Município (§ 8º).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 23 a 28, 31 e 32', 'Gestão de documentos', 'Secretarias e órgãos municipais, por meio da sua CSPAD',
    ARRAY['Plano de Classificação e Tabela de Temporalidade das atividades-fim', 'Parecer da assessoria jurídica do órgão sobre a temporalidade e as referências legais (art. 32, § 4º)', 'Encaminhamento por ofício ou memorando (art. 32, § 5º)']::text[], ARRAY['Elaboração pela CSPAD', 'Encaminhamento ao Arquivo Público', 'Análise da Comissão Central e parecer conclusivo', 'Aprovação pela Secretaria Municipal de Administração', 'Publicação no Diário Oficial do Município']::text[], '[]'::jsonb, '', 'Gratuito', 'list-tree', 'published', 6, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('transferencia-arquivo-intermediario', 'Transferência ao arquivo intermediário', 'Envio dos documentos que cumpriram o prazo no arquivo corrente para a unidade com atribuições de arquivo intermediário do órgão.', '## Sobre o serviço

A Tabela de Temporalidade indica quantos anos o documento fica na **unidade produtora** (arquivo corrente). Cumprido esse prazo, ele segue para a **unidade com atribuições de arquivo intermediário** do órgão, onde cumpre os prazos prescricionais ou precaucionais (art. 29).

A execução das determinações da Tabela de Temporalidade cabe aos núcleos de Protocolo e Arquivo Corrente de cada secretaria (art. 49), sob a coordenação da CSPAD (art. 32, VII).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 18, 29, 32, VII, e 49', 'Gestão de documentos', 'Unidades produtoras dos órgãos municipais',
    ARRAY['Documentos classificados conforme a Tabela de Temporalidade', 'Prazo da fase corrente cumprido']::text[], ARRAY['Identificar os documentos com prazo corrente cumprido', 'Preparar e relacionar os documentos', 'Transferir para o arquivo intermediário do órgão']::text[], '[]'::jsonb, '', 'Gratuito', 'arrow-right-left', 'published', 7, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('emprestimo-documentos', 'Empréstimo e devolução de documentos', 'Empréstimo controlado de documentos arquivados, com registro de saída e devolução.', '## Sobre o serviço

A padronização dos procedimentos de **empréstimo e consulta** de documentos é uma das rotinas que os órgãos setoriais do SISMARQ devem implementar (art. 12, II). A TTDD da Secretaria Municipal de Administração reserva uma subfunção para o registro de protocolo, empréstimo e devolução (subfunção 2.0.03.00).

## Base legal

- Lei Municipal nº 10.301/2019, art. 12, II', 'Gestão de documentos', 'Unidades da administração municipal',
    ARRAY['Identificação do documento e da unidade solicitante']::text[], ARRAY['Solicitar o empréstimo', 'Registro da saída', 'Devolução e registro do retorno']::text[], '[]'::jsonb, '', 'Gratuito', 'package', 'published', 8, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('recolhimento-arquivo-publico', 'Recolhimento de documentos permanentes ao Arquivo Público', 'Entrada no Arquivo Público Municipal dos documentos de valor permanente, com registro de entrada e cópia à unidade de origem.', '## Sobre o serviço

Os documentos de **valor permanente** são inalienáveis e imprescritíveis (art. 20) e são recolhidos ao Arquivo Público Municipal. Cabe a ele acompanhar o recolhimento, **registrar a entrada** e **encaminhar cópia do registro** à unidade de origem (art. 5º, VII).

O recolhimento segue o cronograma do Regimento Interno do Arquivo Público (art. 47). Os órgãos podem pedir orientação técnica para preparar os documentos (art. 53, § 1º).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 5º, VII, 20, 41, 42, 47 e 53', 'Recolhimento e guarda permanente', 'Órgãos e entidades da administração municipal',
    ARRAY['Documentos classificados e avaliados', 'Documentos organizados, higienizados e acondicionados', 'Instrumento descritivo que permita a identificação, o acesso e o controle (art. 53)']::text[], ARRAY['Solicitar orientação técnica, se necessário', 'Preparar os documentos conforme o art. 53', 'Recolher conforme o cronograma do Regimento Interno', 'Registro de entrada e envio de cópia à unidade de origem']::text[], '[]'::jsonb, '', 'Despesas de preparo, acondicionamento e transporte custeadas pelo órgão produtor (art. 53, § 2º)', 'archive', 'published', 9, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('orientacao-tecnica-gestao-documental', 'Orientação técnica em gestão de documentos', 'Orientação do Arquivo Público às comissões e aos órgãos setoriais na classificação, na avaliação e na aplicação da Tabela de Temporalidade.', '## Sobre o serviço

Como órgão central do SISMARQ, o Arquivo Público orienta as Comissões Permanentes de Avaliação de Documentos na **elaboração e aplicação** dos Planos de Classificação e das Tabelas de Temporalidade (art. 46). Também orienta e acompanha os órgãos setoriais nas normas e rotinas de gestão de documentos (art. 11, V) e na preparação do recolhimento (art. 53, § 1º).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 5º, I e II, 11, V, 46 e 53, § 1º', 'Orientação e capacitação', 'Secretarias, órgãos setoriais e comissões de avaliação',
    ARRAY['Indicação do órgão e do tema da orientação']::text[], ARRAY['Solicitar a orientação ao Arquivo Público', 'Atendimento técnico', 'Acompanhamento da aplicação']::text[], '[]'::jsonb, '', 'Gratuito', 'users', 'published', 10, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('capacitacao-gestao-documentos', 'Capacitação em gestão de documentos', 'Treinamento, aperfeiçoamento e reciclagem dos servidores que atuam na gestão de documentos de arquivo.', '## Sobre o serviço

Compete ao Arquivo Público promover o **treinamento e a orientação técnica** dos profissionais responsáveis pelas atividades arquivísticas das unidades do SISMARQ (art. 5º, VIII) e estimular a capacitação, o aperfeiçoamento e a reciclagem dos servidores (art. 11, VIII). Os órgãos setoriais devem proporcionar essa capacitação aos seus servidores (art. 12, IV).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 5º, VIII, 11, VIII, e 12, IV', 'Orientação e capacitação', 'Servidores que atuam em protocolo e arquivo',
    '{}'::text[], ARRAY['Inscrição do servidor pela unidade', 'Participação na capacitação']::text[], '[]'::jsonb, '', 'Gratuito', 'graduation-cap', 'published', 11, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('declaracao-arquivo-privado', 'Declaração de arquivo privado de interesse público e social', 'Arquivos privados relevantes para a história e a cultura de Rondonópolis podem ser declarados de interesse público e social por decreto.', '## Sobre o serviço

Arquivos privados de pessoas físicas ou jurídicas com conjuntos de documentos **relevantes para a história, a cultura e o desenvolvimento científico e tecnológico** do município podem ser declarados de interesse público e social por **decreto** (art. 56).

A declaração é precedida de parecer com avaliação técnica de uma **Comissão Especial** de especialistas constituída pelo Arquivo Público (§ 1º). Ela **não transfere** o acervo para o Arquivo Público nem retira do detentor a responsabilidade pela guarda (§ 4º). O acervo pode ser doado ao Arquivo Público ou nele depositado a título revogável (§ 5º). Os arquivos das entidades privadas encarregadas de serviços públicos municipais já são considerados de interesse público e social (§ 3º).

## Base legal

- Lei Municipal nº 10.301/2019, arts. 55 e 56', 'Arquivos privados', 'Proprietários ou detentores de arquivos privados',
    ARRAY['Identificação do proprietário ou detentor', 'Descrição do acervo']::text[], ARRAY['Requerimento ao Arquivo Público', 'Avaliação técnica pela Comissão Especial', 'Parecer', 'Decreto de declaração']::text[], '[]'::jsonb, '', 'Gratuito', 'landmark', 'published', 12, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('assistencia-arquivos-privados', 'Assistência técnica a arquivos privados de interesse público', 'Apoio técnico, mediante convênio, para organizar, preservar e divulgar acervos privados de interesse público.', '## Sobre o serviço

Os proprietários ou detentores de arquivos privados declarados de interesse público e social podem receber **assistência técnica** do Arquivo Público Municipal, ou de outras instituições arquivísticas, **mediante convênio**. O objetivo é apoiar a organização, a preservação e a divulgação do acervo (art. 57).

## Base legal

- Lei Municipal nº 10.301/2019, art. 57', 'Arquivos privados', 'Proprietários ou detentores de arquivos privados declarados de interesse público e social',
    ARRAY['Arquivo declarado de interesse público e social']::text[], ARRAY['Solicitação ao Arquivo Público', 'Celebração de convênio', 'Execução da assistência técnica']::text[], '[]'::jsonb, '', 'Conforme o convênio', 'hand-heart', 'published', 13, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO catalog_services (slug, title, summary, description, category, audience, requirements, steps, channels, sla, cost, icon, status, position, published_at)
VALUES ('alienacao-arquivo-privado', 'Notificação de alienação de arquivo privado de interesse público', 'Quem pretende alienar um arquivo privado de interesse público deve notificar o Município, que tem preferência na aquisição.', '## Sobre o serviço

A alienação de arquivos privados declarados de interesse público e social deve ser precedida de **notificação ao Município**, titular do **direito de preferência**. O Município tem prazo máximo de **60 dias** para manifestar interesse na aquisição (art. 58).

## Base legal

- Lei Municipal nº 10.301/2019, art. 58', 'Arquivos privados', 'Proprietários de arquivos privados declarados de interesse público e social',
    ARRAY['Identificação do acervo e das condições da alienação']::text[], ARRAY['Notificar o Município', 'Aguardar a manifestação do Município em até 60 dias']::text[], '[]'::jsonb, 'Manifestação do Município em até 60 dias', 'Gratuito', 'file-text', 'published', 14, now())
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, description = EXCLUDED.description,
    category = EXCLUDED.category, audience = EXCLUDED.audience, requirements = EXCLUDED.requirements, steps = EXCLUDED.steps,
    sla = EXCLUDED.sla, cost = EXCLUDED.cost, icon = EXCLUDED.icon, status = 'published', position = EXCLUDED.position,
    published_at = COALESCE(catalog_services.published_at, EXCLUDED.published_at);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('lei-10301-cria-arquivo-publico-e-sismarq', 'Lei nº 10.301/2019: o Arquivo Público Municipal e o SISMARQ', 'A lei municipal criou o Arquivo Público, definiu a política municipal de arquivos e organizou a gestão de documentos da Prefeitura no Sistema Municipal de Arquivos.', 'A **Lei Municipal nº 10.301, de 10 de junho de 2019**, criou o **Arquivo Público Municipal de Rondonópolis**, definiu as diretrizes da Política Municipal de Arquivos Públicos e Privados e instituiu o **Sistema Municipal de Arquivos (SISMARQ)**.

## Por que a lei existe

A lei parte do dever do Poder Público de gerir os documentos de arquivo e assegurar o acesso às informações neles contidas (Constituição Federal, art. 216, § 2º; Lei Federal nº 8.159/1991). Ela também reconhece a necessidade de reduzir ao essencial os documentos acumulados, sem prejuízo dos atos administrativos, dos direitos e da memória institucional.

## O que ela organiza

- **O Arquivo Público Municipal**, subordinado ao Gabinete da Secretaria Municipal de Administração, com a missão de implementar a política de arquivos, recolher os documentos permanentes e garantir o acesso a eles (art. 5º).
- **O SISMARQ**, que reúne a gestão de documentos de todos os órgãos. O Arquivo Público é o órgão central, e as unidades de protocolo e arquivo das secretarias são os órgãos setoriais (arts. 6º a 12).
- **Os instrumentos de gestão**: o Plano de Classificação e a Tabela de Temporalidade (arts. 22 a 30).
- **As comissões de avaliação** e o rito de eliminação, com os modelos dos Anexos I a IV (arts. 31 a 40).

## O direito do cidadão

A lei assegura o **acesso pleno aos documentos públicos municipais**, ressalvados o sigilo imprescindível à segurança da sociedade e do Estado e a proteção da intimidade, da vida privada, da honra e da imagem das pessoas (art. 2º).', 'comunicado', 'published', true, '2026-10-15 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('tabelas-de-temporalidade-publicadas', 'Tabelas de Temporalidade das secretarias publicadas no DIORONDON-e', 'A Comissão Central Permanente de Avaliação de Documentos aprovou as Tabelas de Temporalidade e Destinação de Documentos das secretarias, publicadas no Diário Oficial do Município.', 'As **Tabelas de Temporalidade e Destinação de Documentos (TTDD)** das secretarias municipais foram aprovadas pela **Comissão Central Permanente de Avaliação de Documentos (CCPAD)** e publicadas no Diário Oficial Eletrônico de Rondonópolis (DIORONDON-e). As publicações são:

- edição nº 6.017, de 25 de agosto de 2025;
- edição nº 6.111, de 13 de janeiro de 2026 (suplementar);
- edição nº 6.275, de 11 de setembro de 2026;
- edição nº 6.281, de 21 de setembro de 2026.

## O que a tabela diz

Cada tabela organiza os documentos da secretaria por **função**, **subfunção** e **série documental**, com um código de classificação (por exemplo, 2.0.01.00.01). Para cada série, ela informa:

- o prazo no **arquivo corrente** (unidade produtora);
- o prazo no **arquivo intermediário**;
- a **destinação final**: eliminação ou guarda permanente;
- as **observações**, com o fundamento legal ou administrativo dos prazos.

## Por que importa

A Tabela de Temporalidade é o instrumento que define os prazos de guarda e a destinação dos documentos (art. 27 da Lei nº 10.301/2019). Sem ela, nada pode ser eliminado: a eliminação só ocorre se estiver prevista na tabela e for autorizada pelo Arquivo Público (art. 19).

## Onde consultar

As séries publicadas podem ser consultadas, sem login, na página **Temporalidade** deste portal.', 'noticia', 'published', false, '2026-10-14 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('as-tres-idades-dos-documentos', 'As três idades dos documentos públicos', 'Corrente, intermediária e permanente: como a lei municipal classifica os documentos ao longo do seu ciclo de vida.', 'A Lei nº 10.301/2019 identifica os documentos públicos municipais em **três idades** (art. 18).

## Corrente

Documentos **em curso** ou que, mesmo sem movimentação, são objeto de **consultas frequentes**. Ficam na **unidade produtora**, no arquivo corrente, pelo número de anos indicado na Tabela de Temporalidade (art. 29, I).

## Intermediária

Documentos que não são mais de uso corrente e, por interesse administrativo, **aguardam a eliminação ou o recolhimento** para a guarda permanente. Ficam na **unidade com atribuições de arquivo intermediário** do órgão, cumprindo prazos prescricionais ou precaucionais (art. 29, II).

## Permanente

Conjuntos de documentos de **valor histórico, probatório e informativo**, que devem ser **definitivamente preservados**. São inalienáveis e imprescritíveis (art. 20) e não podem ser eliminados nem depois de microfilmados ou digitalizados (art. 42).

## Como se define o tempo de guarda

Os prazos da Tabela de Temporalidade consideram três tempos (art. 30):

- **vigência**: o tempo em que o documento produz efeitos administrativos e legais plenos;
- **prescrição**: o tempo em que se pode recorrer ao Judiciário. A guarda é dilatada se a prescrição for interrompida ou suspensa;
- **precaução**: a guarda preventiva antes de eliminar ou recolher.', 'noticia', 'published', false, '2026-10-13 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('como-funciona-a-eliminacao-de-documentos', 'Como funciona a eliminação de documentos: os Anexos I a IV', 'Relação, ata, edital de 30 dias, amostragem e termo: o rito que a lei municipal exige antes de qualquer descarte.', 'Na administração municipal, **nenhum documento é eliminado sem autorização prévia do Arquivo Público** (art. 65 da Lei nº 10.301/2019). Só pode ser eliminado o que está previsto na Tabela de Temporalidade, com o prazo de guarda cumprido (art. 19). O caminho tem etapas e modelos definidos em lei.

## 1. Avaliação pela comissão

A **Comissão Setorial Permanente de Avaliação de Documentos (CSPAD)** do órgão analisa os documentos e confere se há **ações judiciais** encerradas ou em curso nas quais o Município figure como autor ou réu (arts. 32, II, e 44).

## 2. Relação de Eliminação (Anexo I)

Os documentos são listados com o **código de classificação**, o **assunto**, as **datas-limite**, a **unidade de arquivamento** (quantificação e especificação) e as observações (art. 34). Quando necessário, a relação indica as contas aprovadas pelo Tribunal de Contas. Ela é assinada pelo responsável pela seleção, pelo presidente da comissão e pelo gestor do órgão, e recebe o **"Autorizo"** do titular da Divisão de Arquivo Público.

## 3. Ata da comissão (Anexo IV)

A CSPAD aprova a listagem em reunião registrada em **ata**, com as manifestações dos presentes e a lista de presença (art. 33).

## 4. Edital de Ciência (Anexo III)

O edital é publicado no Diário Oficial do Município e abre **30 dias** para manifestações. Nesse prazo, os interessados podem pedir o **desentranhamento** de documentos ou cópias de peças (art. 37).

## 5. Amostragem

Dos documentos destinados à eliminação, uma **amostra representativa**, escolhida por critérios qualitativos e quantitativos, vai para a **guarda permanente** (art. 39).

## 6. Eliminação e Termo (Anexo II)

A eliminação é feita por **fragmentação** manual ou mecânica, e o papel é doado nos termos do Decreto Federal nº 5.940/2006 (art. 40). O **Termo de Eliminação** registra o ato, e uma cópia vai à Divisão de Arquivo Público e à Secretaria Municipal de Administração (art. 38).

## Responsabilidade

Quem desfigurar ou destruir documento de valor permanente ou de interesse público e social fica sujeito à **responsabilidade penal, civil e administrativa** (art. 66).', 'noticia', 'published', false, '2026-10-12 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('comissoes-de-avaliacao-ccpad-e-cspad', 'CCPAD e CSPAD: quem avalia os documentos do Município', 'A lei municipal prevê uma Comissão Central junto ao Arquivo Público e uma Comissão Setorial em cada órgão, com composição multidisciplinar.', 'A avaliação documental, que define o que se guarda e por quanto tempo, é feita por **comissões permanentes** (art. 31 da Lei nº 10.301/2019).

## Comissão Setorial (CSPAD)

Cada órgão e entidade da administração municipal tem uma **Comissão Setorial Permanente de Avaliação de Documentos**, vinculada ao gabinete da autoridade máxima do órgão (§ 2º). A comissão é composta preferencialmente em **número ímpar** (§ 6º), com:

- servidor com formação em **Arquivologia ou História**;
- servidor da **assessoria jurídica**, que analisa o valor legal dos documentos;
- servidor da área de **administração e finanças**;
- servidores das **unidades** às quais os documentos se referem;
- outros profissionais ligados ao acervo, como médicos, engenheiros e historiadores.

## Comissão Central (CCPAD)

A **Comissão Central Permanente de Avaliação de Documentos** fica junto ao Arquivo Público. Ela estabelece os procedimentos para elaborar e aprovar os Planos de Classificação e as Tabelas de Temporalidade das atividades-fim (§ 3º). É formada por servidores efetivos de Arquivologia ou História, da assessoria jurídica, de administração e finanças e do Arquivo Público (§ 7º).

## O que as comissões fazem

- Identificam, analisam, avaliam e selecionam os documentos (art. 32, I).
- Analisam a existência de ações judiciais (art. 32, II).
- Elaboram e atualizam os Planos de Classificação e as Tabelas de Temporalidade e propõem os critérios de amostragem (art. 32, III).
- Coordenam a transferência e o recolhimento de documentos ao Arquivo Público (art. 32, VII).

O trabalho nas comissões não é remunerado e é considerado serviço público relevante (art. 32, § 3º).', 'noticia', 'published', false, '2026-10-11 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('classificacao-pelo-metodo-duplex', 'Classificação de documentos pelo método duplex', 'Como o código de classificação organiza os documentos do geral para o particular, do órgão produtor à espécie documental.', 'O **Plano de Classificação de Documentos** é o instrumento usado para classificar todo documento de arquivo (art. 23 da Lei nº 10.301/2019). Classificar é agrupar os documentos pelo mesmo assunto, classe e subclasse do órgão produtor.

## O método duplex

Os planos das atividades-meio e das atividades-fim adotam o **método duplex**, que distribui os documentos em classes e subclasses de assunto, **do geral para o particular**, conforme as atividades do órgão (art. 24).

## O código

O código de classificação da espécie documental é composto por cinco unidades de informação (art. 25):

1. órgão produtor;
2. classe;
3. subclasse;
4. assunto;
5. espécie documental.

Nas Tabelas de Temporalidade publicadas, o código aparece como **órgão, função, subfunção e série** (por exemplo, 2.0.01.00.01, "Organogramas", da Secretaria Municipal de Administração, Gestão de Pessoas e Inovação).

## Atividades-meio

Para as atividades-meio, comuns a todos os órgãos, a lei remete às classes do plano do CONARQ (art. 26): organização administrativa, comunicação institucional, gestão de recursos humanos, gestão de bens materiais e patrimoniais, gestão orçamentária e financeira, gestão de documentos e informações e gestão de atividades complementares.', 'noticia', 'published', false, '2026-10-10 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('sismarq-e-sei-papeis-complementares', 'SISMARQ e SEI: papéis complementares', 'O SEI tramita os processos em curso; o SISMARQ cuida da guarda intermediária e permanente e do rito de eliminação.', 'A chegada do processo eletrônico (SEI) às prefeituras leva, às vezes, à ideia de que o arquivamento dele resolve todo o acervo do município. O artigo técnico do SISMARQ separa as responsabilidades.

## SEI: a fase corrente

O SEI cuida dos documentos **nato-digitais** da rotina diária. Nele ficam a **tramitação** dos processos entre as secretarias, os despachos e as assinaturas, a instrução processual e a autuação dos processos das comissões de avaliação. É a **fase corrente** (art. 18, § 1º, da Lei nº 10.301/2019).

## SISMARQ: as fases intermediária e permanente

O SISMARQ cuida do **passivo físico e digitalizado** e das **fases intermediária e permanente** (art. 18, §§ 2º e 3º). Isso inclui:

- a localização física dos documentos (depósito, estante e caixa);
- a digitalização com valor probatório;
- a aplicação da Tabela de Temporalidade, com bloqueio dos documentos envolvidos em ações judiciais;
- o rito de eliminação com os modelos da lei municipal.

## Como se ligam

Os processos concluídos no SEI são **recolhidos** ao SISMARQ, em conformidade com os arts. 5º, VII, e 53 da lei municipal. O SEI instrui as decisões das comissões, e o SISMARQ guarda o acervo.', 'noticia', 'published', false, '2026-10-09 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('digitalizar-nao-e-descartar', 'Digitalizar não é descartar: documentos permanentes e digitalização', 'A lei municipal proíbe eliminar documentos permanentes depois de digitalizados. Veja o que a digitalização com valor probatório exige.', 'Digitalizar o acervo melhora o acesso e protege o original do manuseio, mas **não autoriza o descarte** dos documentos permanentes.

## O que diz a lei municipal

Os documentos de guarda permanente **não podem ser eliminados** após microfilmagem, digitalização ou qualquer outra forma de reprodução. Eles devem ser preservados pelo órgão produtor ou recolhidos ao Arquivo Público (art. 42 da Lei nº 10.301/2019). As disposições da lei valem também para os documentos **digitais, digitalizados e híbridos** (art. 48).

## Digitalização com valor probatório

Para que a cópia digitalizada tenha valor de prova, o artigo técnico do SISMARQ segue o **Decreto Federal nº 10.278/2020**:

- arquivo no formato **PDF/A** (ISO 19005);
- resolução mínima de **300 DPI** para textos e **600 DPI** para plantas e imagens de alta densidade;
- **metadados técnicos**: identificador único, responsável pela digitalização, data e hora, equipamento, resolução, código de classificação e o **hash SHA-256** do arquivo;
- reconhecimento de texto (**OCR**) para tornar o conteúdo pesquisável.

## E os documentos que podem ser eliminados?

Para os documentos com destinação de eliminação, valem a Tabela de Temporalidade e o rito completo: avaliação, relação, ata, edital de 30 dias, amostragem e termo (arts. 33 a 40).', 'noticia', 'published', false, '2026-10-08 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('arquivos-privados-de-interesse-publico', 'Arquivos privados de interesse público e social', 'Acervos particulares relevantes para a história de Rondonópolis podem ser protegidos por decreto, sem deixar de pertencer aos seus donos.', 'Além dos documentos públicos, a Lei nº 10.301/2019 protege os **arquivos privados** relevantes para o município (arts. 55 a 58).

## O que é um arquivo privado

É o conjunto de documentos produzidos ou recebidos por pessoas físicas ou jurídicas em decorrência de suas atividades (art. 55).

## Declaração de interesse público e social

Arquivos com conjuntos de documentos relevantes para a **história, a cultura e o desenvolvimento científico e tecnológico** de Rondonópolis podem ser declarados de interesse público e social por **decreto** (art. 56). Antes, uma **Comissão Especial** de especialistas, constituída pelo Arquivo Público, faz a avaliação técnica (§ 1º).

## O que muda para o dono

- O acervo **continua com o proprietário**, que segue responsável pela guarda e pela preservação (§ 4º).
- O acesso público depende de **autorização** do proprietário ou possuidor (§ 2º).
- O acervo pode ser **doado** ao Arquivo Público ou nele **depositado** a título revogável (§ 5º).
- O proprietário pode receber **assistência técnica** mediante convênio (art. 57).
- Antes de vender, o proprietário deve **notificar o Município**, que tem preferência e **60 dias** para manifestar interesse (art. 58).

Os arquivos das entidades privadas encarregadas de serviços públicos municipais já são considerados de interesse público e social (art. 56, § 3º).', 'noticia', 'published', false, '2026-10-07 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO blog_posts (slug, title, summary, body, kind, status, pinned, published_at)
VALUES ('direito-de-acesso-aos-documentos-publicos', 'O direito de acesso aos documentos públicos municipais', 'A lei municipal garante o acesso pleno aos documentos públicos, de forma ágil e transparente, com o sigilo como exceção.', 'O primeiro direito que a Lei nº 10.301/2019 assegura é o **acesso do cidadão aos documentos públicos municipais** (art. 2º).

## A regra é o acesso

A consulta deve ser franqueada **de forma ágil e transparente** pelo Poder Público. Garantir esse acesso, a cidadãos e órgãos, de forma segura, é a primeira finalidade do SISMARQ (art. 7º, I). Também cabe ao Arquivo Público difundir informações sobre o seu acervo e garantir o acesso aos documentos (art. 5º, X).

## As exceções

Ficam ressalvados os documentos cujo sigilo seja **imprescindível à segurança da sociedade e do Estado** e os que envolvem a **inviolabilidade da intimidade, da vida privada, da honra e da imagem das pessoas** (art. 2º).

## O que é documento público

São públicos os documentos produzidos e recebidos pelos órgãos e entidades municipais no exercício de suas funções (art. 14). Também são públicos os produzidos por agentes públicos no exercício do cargo e por empresas, concessionárias e permissionárias quando exercem funções delegadas pelo Município (parágrafo único).

## Como consultar

Veja os serviços **Consulta a documentos públicos municipais** e **Consulta à Tabela de Temporalidade** na carta de serviços do Arquivo Público.', 'noticia', 'published', false, '2026-10-06 09:00:00-04')
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, body = EXCLUDED.body,
    kind = EXCLUDED.kind, status = 'published', pinned = EXCLUDED.pinned;
-- +goose StatementEnd

UPDATE branding_settings SET app_description = 'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis'
WHERE app_description = 'Plataforma corporativa modular';

ALTER TABLE branding_settings ALTER COLUMN app_description SET DEFAULT 'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis';

-- +goose Down
ALTER TABLE branding_settings ALTER COLUMN app_description SET DEFAULT 'Plataforma corporativa modular';
UPDATE branding_settings SET app_description = 'Plataforma corporativa modular'
WHERE app_description = 'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis';

DELETE FROM blog_posts WHERE slug IN ('lei-10301-cria-arquivo-publico-e-sismarq', 'tabelas-de-temporalidade-publicadas', 'as-tres-idades-dos-documentos', 'como-funciona-a-eliminacao-de-documentos', 'comissoes-de-avaliacao-ccpad-e-cspad', 'classificacao-pelo-metodo-duplex', 'sismarq-e-sei-papeis-complementares', 'digitalizar-nao-e-descartar', 'arquivos-privados-de-interesse-publico', 'direito-de-acesso-aos-documentos-publicos');

DELETE FROM catalog_services WHERE slug IN ('consulta-documentos-publicos', 'consulta-tabela-temporalidade', 'pedido-desentranhamento-edital', 'autorizacao-eliminacao-documentos', 'eliminacao-fora-da-ttdd', 'aprovacao-plano-classificacao-ttdd', 'transferencia-arquivo-intermediario', 'emprestimo-documentos', 'recolhimento-arquivo-publico', 'orientacao-tecnica-gestao-documental', 'capacitacao-gestao-documentos', 'declaracao-arquivo-privado', 'assistencia-arquivos-privados', 'alienacao-arquivo-privado');

UPDATE catalog_services c SET status = a.status FROM sismarq_conteudo_anterior a WHERE a.tabela = 'catalog_services' AND a.id = c.id;
UPDATE blog_posts b SET status = a.status, pinned = a.pinned FROM sismarq_conteudo_anterior a WHERE a.tabela = 'blog_posts' AND a.id = b.id;
DROP TABLE sismarq_conteudo_anterior;
