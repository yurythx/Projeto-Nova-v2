-- +goose Up
-- Carga inicial de serviços especializados de TI (PaperMoon + Terraform + Ansible).
-- Podem ser ativados (published), despublicados (draft) ou arquivados (archived)
-- individualmente em Configurações > Gestão de Serviços (/gestao/servicos).

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'whatsapp-api',
    'WhatsApp Business API',
    'Atendimento profissional no número oficial da sua empresa.',
    '## Sobre o Serviço

A WhatsApp Business API foi lançada pela Meta em 2018 para atender empresas que precisavam de escala no atendimento via WhatsApp sem depender do aplicativo convencional — que suporta apenas um dispositivo por vez. Antes dela, muitas empresas recorriam a soluções não-oficiais baseadas em automação do WhatsApp Web, tecnicamente proibidas pelos termos de uso e sujeitas a ban permanente.

Com a API oficial, o número da empresa é verificado diretamente pela Meta, recebe o selo verde de empresa verificada e passa a ter acesso à infraestrutura de mensageria de nível enterprise: entrega garantida por SLA, sem limite de conversas simultâneas, suporte a templates aprovados (HSM) para campanhas ativas e integração nativa com ferramentas de atendimento como o Chatwoot.

A PaperMoon conduz todo o processo: desde a criação e verificação da conta no Meta Business Manager até a configuração do webhook entre a API Meta e o Chatwoot instalado na sua VPS. Nossa equipe conhece os requisitos de aprovação da Meta e reduz significativamente o tempo e os erros no processo de onboarding.

## Diferenciais

- Número verificado diretamente pela Meta — sem risco de ban
- A PaperMoon guia todo o processo de aprovação no Meta Business Manager, passo a passo
- Integração imediata com Chatwoot: equipe atende assim que o número é aprovado
- Monitoramento contínuo da conectividade entre a API Meta e a VPS do cliente
- Suporte na aprovação dos primeiros templates HSM — maior taxa de sucesso na primeira tentativa

## O que nós entregamos

- Guia o processo de registro do número no Meta Business Manager
- Conecta a API Meta ao Chatwoot instalado na sua VPS
- Configura o webhook de recebimento de mensagens
- Auxilia na aprovação dos primeiros templates HSM
- Monitora a conectividade do número com a API Meta

## Responsabilidades do Cliente

- Criar e manter a conta no Meta Business Manager
- Registrar o número WABA (WhatsApp Business Account)
- Cadastrar cartão de crédito válido na conta Meta
- Arcar com os custos de conversas e disparos cobrados pela Meta
- Obter opt-in dos destinatários antes de enviar mensagens ativas
- Manter conformidade com as políticas de uso da Meta

## Recursos e Funcionalidades

### Confiabilidade

- Número verificado diretamente pela Meta
- Zero risco de ban — API oficial, não app clone
- SLA de entrega de mensagens garantido pela Meta
- Uptime de 99,9% na infraestrutura de mensagens Meta

### Capacidade

- Múltiplos agentes no mesmo número simultaneamente
- Sem limite de conversas simultâneas
- 1.000 conversas de serviço gratuitas por mês
- Volume escalonável sem troca de número ou plano

### Mensagens ativas

- Templates HSM aprovados pela Meta para disparos
- Categorias: Marketing, Utility, Authentication
- Envio de mídia: imagens, documentos, vídeos, localização
- Botões interativos e listas de opções

## Perguntas Frequentes (FAQ)

### Qual a diferença entre API oficial e apps não-oficiais?

Apps não-oficiais (como versões modificadas) simulam o WhatsApp e violam os termos de uso — o número pode ser banido permanentemente sem aviso. A API oficial é a única forma autorizada pela Meta para uso empresarial. Com ela, seu número é verificado, protegido e escalável.

### Preciso de cartão cadastrado na Meta?

Sim. Um cartão de crédito ou débito válido deve estar na conta Meta Business Manager para ativar o número e enviar mensagens ativas. As conversas iniciadas pela empresa (notificações, campanhas) são cobradas pela Meta direto no cartão — não pela PaperMoon.

### Posso usar meu número atual?

Sim, desde que ele não esteja em uso no WhatsApp pessoal ou no WhatsApp Business app. O número precisa ser migrado para a API, o que requer desinstalar o app atual. A PaperMoon orienta todo o processo de portabilidade.

### Quanto tempo leva para o número ser aprovado?

A aprovação depende da análise da Meta e da consistência da documentação enviada. A PaperMoon acompanha esse processo, orienta os ajustes necessários e só agenda a integração final quando a conta estiver liberada para operação.

### Posso fazer disparos em massa com a API oficial da Meta?

A API oficial da Meta permite enviar templates de mensagem (HSM) para listas de contatos — notificações, cobranças, confirmações. O volume é limitado pelos níveis de qualidade do número e pelo plano de mensagens da Meta. A PaperMoon configura a integração com o n8n para automação dos disparos e orienta sobre os limites e boas práticas para manter o número com alta reputação.',
    'Comunicação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Conta Meta Business Manager criada e verificada com CNPJ', 'Número de telefone dedicado não vinculado ao WhatsApp pessoal ou Business app', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo', 'Cartão de crédito/débito cadastrado na conta Meta para cobranças de mensagens']::text[],
    ARRAY['01. Crie sua conta Meta Business: Acesse business.whatsapp.com e registre sua empresa. Você precisará de um CNPJ e um número de telefone dedicado ao WhatsApp corporativo. A PaperMoon guia todo o processo.', '02. Verificação e aprovação pela Meta: A Meta verifica sua empresa e aprova o número. A duração dessa etapa varia conforme a documentação e a análise da conta. Após aprovado, seu número aparece como verificado para todos os contatos.', '03. Conectamos ao Chatwoot na sua VPS: Com o número aprovado, a PaperMoon configura o webhook entre a API Meta e o Chatwoot instalado na sua VPS. Em minutos sua equipe já recebe e responde mensagens no painel.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'shield',
    'published',
    10,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'chatwoot',
    'Chatwoot',
    'Toda a sua equipe no mesmo número, com histórico completo.',
    '## Sobre o Serviço

O Chatwoot surgiu em 2019 como um projeto open-source criado por Pranav Raj S e equipe, com o objetivo de oferecer uma alternativa gratuita e auto-hospedada a ferramentas como Intercom e Zendesk — que cobram por agente e mantêm os dados do cliente em servidores de terceiros. Hoje conta com mais de 20 mil estrelas no GitHub e é adotado por empresas de todos os portes ao redor do mundo.

A proposta central do Chatwoot é unificar todos os canais de atendimento — WhatsApp, e-mail, Instagram, Telegram — em um único painel, com histórico completo por contato, atribuição de conversas entre agentes, criação de etiquetas e métricas de SLA em tempo real. Diferente de SaaS pagos, o Chatwoot instalado na sua VPS significa que seus dados de atendimento ficam inteiramente sob controle da sua empresa.

A PaperMoon realiza a instalação completa do Chatwoot com banco de dados PostgreSQL, Redis, Nginx com SSL e todas as configurações de produção. Configuramos as inboxes por canal, criamos os perfis de agente e conectamos ao WhatsApp Business API e ao n8n — entregando o ambiente pronto para sua equipe operar com segurança e contexto completo.

## Diferenciais

- Open-source e auto-hospedado: seus dados de atendimento nunca saem da sua VPS
- Instalação production-ready com PostgreSQL, Redis, Nginx e SSL — sem etapas manuais
- Integração imediata com WhatsApp Business API Meta e n8n configurada pela PaperMoon
- Atualizações de versão gerenciadas pela nossa equipe — sem downtime ou perda de dados
- Custo zero de licença por agente — escale a equipe sem aumentar a fatura de software

## O que nós entregamos

- Instala e configura o Chatwoot na sua VPS
- Integra com a API Meta e o n8n
- Cria as caixas de entrada (inboxes) por canal
- Configura agentes, equipes e permissões iniciais
- Mantém o Chatwoot atualizado para novas versões
- Monitora a disponibilidade do serviço

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o Chatwoot é instalado
- Gerenciar agentes, permissões e equipes no dia a dia
- Exportar ou fazer backup dos históricos de conversa
- Configurar horários de atendimento e respostas padrão

## Recursos e Funcionalidades

### Gestão de conversas

- Múltiplos agentes no mesmo número WhatsApp
- Atribuição manual ou automática de conversas
- Transferência entre agentes e equipes
- Notas internas visíveis só para a equipe
- Respostas rápidas (canned responses) configuráveis
- Histórico completo por contato

### Organização

- Etiquetas personalizáveis por conversa ou contato
- Filtros avançados: por agente, status, canal, etiqueta
- Inbox unificado para WhatsApp, e-mail e outros canais
- Busca em todo o histórico de mensagens

### Métricas e SLA

- Dashboard de métricas por agente e equipe
- Tempo médio de resposta e de resolução
- Relatório de conversas por período
- Alertas de SLA configuráveis por urgência

## Perguntas Frequentes (FAQ)

### O Chatwoot é open-source. Por que pagar a PaperMoon?

O Chatwoot é gratuito para instalar, mas exige infraestrutura (VPS, banco PostgreSQL, Redis, S3 para arquivos), configuração técnica e manutenção contínua (atualizações, backups, monitoramento). A PaperMoon cuida de tudo isso para que sua equipe foque no atendimento, não na infraestrutura.

### Quantos agentes posso ter?

Não há limite de agentes — o Chatwoot é instalado na sua VPS, então o limite real é a capacidade do servidor. Para a maioria das operações, um VPS básico (2 vCPUs, 4 GB RAM) suporta confortavelmente 20 a 50 agentes simultâneos.

### Posso ter múltiplos canais além do WhatsApp?

Sim. O Chatwoot suporta e-mail, Instagram, Telegram, Facebook Messenger e API própria. A PaperMoon configura os canais que você precisar na mesma instalação.

### Os dados ficam onde?

Todos os históricos de conversa ficam armazenados exclusivamente na sua VPS — você tem controle total. A PaperMoon não armazena dados de conversas em nenhum servidor próprio.

### É possível configurar chatbot ou respostas automáticas no Chatwoot?

Sim. O Chatwoot suporta respostas rápidas (canned responses) para agilizar o atendimento, e integra nativamente com o n8n para automações mais avançadas — bot de FAQ, coleta de dados e roteamento automático de conversas. A PaperMoon configura a integração entre Chatwoot e n8n como parte da entrega.',
    'Comunicação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo', 'SMTP configurado ou conta de e-mail para envio de notificações (opcional)']::text[],
    ARRAY['01. Instalação na sua VPS: A PaperMoon instala o Chatwoot na VPS que você contratar (Hostinger, HostGator, AWS...). Todo o setup de banco de dados, Redis e configurações de produção é feito por nós.', '02. Configuração da caixa de entrada: Conectamos a inbox do Chatwoot ao seu número WhatsApp via API Meta. Criamos os perfis dos agentes, equipes e definimos as regras de atribuição conforme sua operação.', '03. Sua equipe começa a atender: Cada agente acessa com login próprio. Conversas chegam distribuídas, com histórico completo do contato, etiquetas e notas internas. Métricas de SLA disponíveis em tempo real.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'message-square',
    'published',
    20,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'n8n',
    'n8n',
    'Filtre, automatize e conecte antes de chegar ao agente.',
    '## Sobre o Serviço

O n8n foi criado em 2019 pelo alemão Jan Oberhauser como alternativa self-hosted ao Zapier — com uma diferença crucial: os fluxos e os dados ficam inteiramente no servidor do cliente, não em infraestrutura da plataforma. Em pouco tempo chegou a mais de 40 mil estrelas no GitHub e hoje conta com mais de 400 integrações nativas com ferramentas de negócios, desenvolvimento e operações.

Sua arquitetura de nós visuais — onde cada bloco representa uma etapa de automação — permite criar desde simples notificações automáticas até pipelines complexos com lógica condicional, loops e transformação de dados, sem necessidade de programação. Empresas de atendimento o utilizam principalmente como camada de triagem entre o WhatsApp e os agentes humanos: o n8n responde as dúvidas mais comuns automaticamente, coleta dados do cliente e encaminha para o agente certo com o contexto já preenchido.

A PaperMoon instala o n8n na VPS do cliente já integrado ao Chatwoot e ao webhook da Meta, e configura os fluxos iniciais conforme o processo de atendimento da empresa: bot de FAQ, mensagem automática fora do horário, roteamento por departamento e coleta de dados cadastrais. Documentamos cada fluxo para que sua equipe possa ajustá-los com autonomia.

## Diferenciais

- Auto-hospedado: seus fluxos de automação e dados de clientes ficam na sua VPS
- Mais de 400 integrações nativas — conecta qualquer sistema com API sem código
- A PaperMoon entrega os fluxos iniciais prontos e documentados, não apenas a instalação
- Zero dependência de plano SaaS: não paga por operação nem por usuário
- Integração nativa com Chatwoot e WhatsApp API já configurada na entrega

## O que nós entregamos

- Instala e configura o n8n na sua VPS
- Cria os fluxos iniciais de qualificação e roteamento
- Integra o n8n com o Chatwoot e a API Meta
- Configura o bot de perguntas frequentes (FAQ)
- Mantém o n8n atualizado e monitorado
- Ajuda na criação de novos fluxos conforme a operação cresce

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o n8n é instalado
- Fornecer credenciais de APIs externas a integrar (CRM, ERP, etc.)
- Aprovar e validar os fluxos criados
- Solicitar ajustes nos fluxos via suporte

## Recursos e Funcionalidades

### Qualificação automática

- Bot de perguntas frequentes (FAQ) 24 horas por dia
- Coleta de nome, CPF, número do pedido ou qualquer dado
- Mensagem automática fora do horário de atendimento
- Confirmação automática de recebimento da mensagem

### Roteamento inteligente

- Roteamento por palavra-chave ou intenção
- Atribuição automática ao departamento correto
- Roteamento por horário (comercial, pós-horário, feriado)
- Encaminhamento por agente disponível ou especialista

### Integrações

- Conexão com planilhas Google Sheets ou Excel
- Consulta de dados em CRM (HubSpot, Pipedrive, RD Station...)
- Integração com sistemas ERP via API REST ou webhook
- Notificações automáticas para Slack, e-mail ou Teams
- Abertura automática de chamados no GLPI

## Perguntas Frequentes (FAQ)

### Preciso saber programar para usar o n8n?

Não. O n8n tem um editor visual onde você cria fluxos arrastando e conectando blocos. A PaperMoon cria os fluxos iniciais para você e treina sua equipe para ajustes simples. Para automações mais complexas, nossa equipe cuida do desenvolvimento.

### O n8n substitui um agente humano?

Não completamente, mas reduz drasticamente o volume de perguntas repetitivas. O n8n resolve as dúvidas mais comuns automaticamente (horário, endereço, status de pedido) e encaminha para agentes apenas o que realmente precisa de atenção humana.

### Consigo integrar com meu sistema atual?

Muito provavelmente sim. O n8n possui mais de 400 integrações nativas (HubSpot, Salesforce, Google Sheets, Slack, Trello, RD Station...) e suporta qualquer sistema com API REST ou webhook. Consulte nossa equipe para avaliar seu caso específico.

### E se o n8n ficar fora do ar?

A PaperMoon monitora a disponibilidade do n8n na sua VPS. Em caso de queda, as mensagens do WhatsApp continuam chegando no Chatwoot — apenas sem passar pelos fluxos automáticos. Nossa equipe é notificada automaticamente e age para restaurar o serviço.

### O n8n salva os dados das automações no meu servidor?

Sim. Com o n8n self-hosted, todos os dados de execução dos fluxos — logs, payloads de entrada e saída, histórico de erros — ficam armazenados no banco de dados da sua VPS. Nenhuma informação dos seus clientes ou fluxos passa pelos servidores da n8n GmbH.',
    'Automação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 1 vCPU, 2 GB RAM, 20 GB SSD)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo']::text[],
    ARRAY['01. Instalação e integração: A PaperMoon instala o n8n na sua VPS e conecta ao webhook do WhatsApp. Toda mensagem que chegar passará pelos seus fluxos antes de aparecer para um agente.', '02. Configuração dos fluxos: Mapeamos sua operação: horários de atendimento, departamentos, perguntas frequentes e regras de roteamento. Criamos os primeiros fluxos no editor visual do n8n.', '03. Filtro em produção: Mensagens chegam, passam pelo n8n — que triaga, responde automaticamente ou encaminha — e só então chegam ao agente certo no Chatwoot, já qualificadas e organizadas.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'filter',
    'published',
    30,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'glpi',
    'GLPI',
    'Central de helpdesk e gestão de chamados de TI instalada na sua VPS.',
    '## Sobre o Serviço

O GLPI (Gestionnaire Libre de Parc Informatique) é desenvolvido desde 2003 pelo projeto francês INDEPNET e hoje mantido pela empresa Teclib''. Com mais de 330 mil instalações registradas em todo o mundo, é o sistema de helpdesk open-source mais utilizado em ambientes corporativos de TI no Brasil — com adoção forte em órgãos públicos, universidades e empresas de médio porte.

Sua proposta vai além do simples registro de chamados: o GLPI integra helpdesk, inventário de ativos (computadores, licenças, contratos), base de conhecimento, gestão de mudanças e relatórios de SLA em uma única plataforma. O módulo de inventário pode ser alimentado automaticamente via agente FusionInventory ou OCS Inventory, eliminando planilhas de controle manuais.

A PaperMoon entrega o GLPI instalado, configurado e pronto para uso — não apenas a instalação técnica, mas a parametrização completa do processo de TI do cliente: criamos as categorias de chamado, definimos os SLAs por urgência e impacto, configuramos as filas de atendimento por equipe, ativamos as notificações automáticas por e-mail e realizamos o treinamento dos técnicos. Cobrança única, sem mensalidade.

## Diferenciais

- Entrega com parametrização completa: SLAs, categorias e filas configuradas conforme seu processo real
- Inventário de TI integrado — nenhuma planilha separada para controlar equipamentos
- Treinamento da equipe técnica incluído no escopo de entrega
- Cobrança única, sem mensalidade — pagou uma vez, é seu para sempre
- Open-source maduro com 20+ anos de desenvolvimento e comunidade ativa

## O que nós entregamos

- Instala e configura o GLPI na sua VPS
- Parametriza categorias, SLAs e filas de atendimento conforme seu negócio
- Importa base de usuários e equipamentos
- Configura notificações automáticas por e-mail
- Documenta toda a configuração e realiza treinamento da equipe de TI

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o GLPI é instalado
- Gerenciar técnicos, categorias e SLAs no dia a dia
- Gerenciar usuários e permissões após o treinamento
- Fazer backup dos dados e histórico de chamados

## Recursos e Funcionalidades

### Gestão de chamados

- Abertura de tickets via e-mail ou portal web
- Categorização por tipo de solicitação e urgência
- SLA configurável por categoria com alertas de prazo
- Notificações automáticas por e-mail a cada atualização
- Histórico completo de todas as interações

### Inventário de TI

- Controle de equipamentos, licenças e contratos
- Vinculação de chamados a ativos do inventário
- Relatórios de uso e manutenção por ativo
- Importação via OCS Inventory ou agente próprio

### Base de conhecimento

- Artigos de solução para problemas recorrentes
- Busca integrada ao abrir chamados
- Histórico de resoluções para referência da equipe
- Redução do tempo de resposta com soluções documentadas

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do GLPI é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, parametrização, configuração de SLAs e treinamento. Suporte técnico posterior (ajustes, atualizações, novos usuários) é contratado sob demanda, via invoice avulsa.

### Posso integrar o GLPI com outros sistemas depois?

Sim. O GLPI possui API REST e suporta integrações com diversas ferramentas. Integrações adicionais — como abertura de chamados via outros canais ou conexão com sistemas de monitoramento — podem ser contratadas separadamente como projetos complementares.

### Posso contratar GLPI e Zabbix juntos?

Sim. O GLPI e o Zabbix podem ser implantados em conjunto na mesma VPS ou em VPSs separadas, conforme a preferência da sua equipe. A contratação conjunta é comum em equipes de TI que querem helpdesk e monitoramento na mesma plataforma.

### Qual a diferença entre o GLPI e outras ferramentas como Freshdesk ou Jira Service?

Freshdesk e Jira Service são SaaS pagos por agente — cada novo técnico aumenta a mensalidade indefinidamente. O GLPI é open-source instalado na sua VPS: sem limite de agentes, sem mensalidade, com dados inteiramente sob controle da empresa. Para equipes de TI que precisam de ITIL, inventário de ativos e base de conhecimento sem custo recorrente, o GLPI é a escolha natural.

### É possível fazer inventário automático de equipamentos com o GLPI?

Sim. O GLPI integra com o FusionInventory ou o agente GLPI Agent, que coletam automaticamente o inventário de hardware e software das estações — processador, memória, disco, aplicativos instalados, número de série. O resultado alimenta o módulo de ativos do GLPI, eliminando planilhas manuais de controle de inventário.',
    'Gestão de TI',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS ou servidor com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo', 'Lista de categorias de chamados, SLAs e grupos de atendimento desejados']::text[],
    ARRAY['01. Instalação na VPS: A PaperMoon instala o GLPI com banco MariaDB na sua VPS e configura o ambiente de produção com SSL e backups automáticos.', '02. Parametrização e SLA: Mapeamos sua operação de TI e configuramos categorias de chamados, SLAs por urgência, filas de atendimento, perfis de técnicos e regras de escalonamento — tudo alinhado ao seu processo.', '03. Equipe de TI em produção: Técnicos recebem chamados organizados por categoria e SLA, com histórico completo de cada solicitação. Treinamos toda a equipe para operar o sistema com autonomia e clareza sobre o processo.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'wrench',
    'published',
    40,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'zabbix',
    'Zabbix',
    'Monitoramento de infraestrutura instalado na sua VPS.',
    '## Sobre o Serviço

O Zabbix foi criado por Alexei Vladishev em 2001 como projeto interno de monitoramento e liberado como open-source em 2004. Hoje é desenvolvido e mantido pela empresa letã Zabbix LLC e é considerado a plataforma de monitoramento de infraestrutura open-source mais robusta e madura do mercado, com adoção em grandes empresas e operadoras de telecomunicações ao redor do mundo.

O Zabbix monitora tudo: servidores físicos e virtuais, switches e roteadores, containers Docker, aplicações web, bancos de dados, arquivos de log e muito mais — via agente instalado no host, SNMP, ICMP ou checks HTTP. Quando uma métrica ultrapassa um limite definido (trigger), o sistema envia alertas imediatos por e-mail, Telegram, Slack ou SMS, com escalonamento automático conforme o nível de severidade.

A PaperMoon instala o Zabbix na VPS do cliente, descobre automaticamente todos os hosts da rede, configura os templates de monitoramento para os serviços em produção (Nginx, PostgreSQL, Docker, etc.) e ajusta os thresholds ideais para o ambiente específico. Criamos dashboards customizados por servidor e serviço, e configuramos as notificações para chegarem ao canal certo da equipe de TI.

## Diferenciais

- Auto-descoberta de hosts na rede: inventário de infraestrutura atualizado automaticamente
- Templates prontos para todos os serviços comuns: Nginx, PostgreSQL, Docker, Linux, VMware
- Alertas com escalonamento: notifica técnico júnior, sênior e gestor conforme o tempo sem resolução
- Dashboards customizados por servidor, serviço ou cliente — não apenas a visão padrão
- Histórico de performance para análise de tendências e planejamento de capacidade

## O que nós entregamos

- Instala e configura o Zabbix na sua VPS
- Configura os agentes Zabbix nos servidores monitorados
- Cria dashboards e triggers para os itens críticos
- Configura alertas por e-mail com escalação e agrupamento
- Documenta a configuração e realiza treinamento do time de TI

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o Zabbix é instalado
- Definir quais servidores e métricas devem ser monitorados
- Configurar os limiares de alerta conforme o negócio
- Gerenciar os acessos dos administradores de TI

## Recursos e Funcionalidades

### Monitoramento

- Servidores Linux, Windows e macOS
- Serviços web (HTTP/S, APIs, endpoints)
- Banco de dados (PostgreSQL, MySQL, Redis)
- Rede: latência, perda de pacotes, interfaces
- Contêineres Docker e clusters Kubernetes

### Alertas

- Alertas por e-mail com detalhes do problema e gravidade
- Escalação: nível 1 → nível 2 se não resolvido
- Agrupamento de alertas para evitar flood
- Alertas de recuperação quando o problema é resolvido

### Visualização

- Dashboards customizáveis por equipe
- Gráficos históricos de qualquer métrica
- Mapas de topologia de rede
- Relatórios de disponibilidade e SLA

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do Zabbix é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração dos agentes, triggers e dashboards. Adição de novos hosts e ajustes futuros são contratados sob demanda, via invoice avulsa.

### Posso monitorar servidores em qualquer provedor?

Sim. O agente Zabbix roda em qualquer servidor Linux ou Windows, independente do provedor (AWS, Azure, GCP, Hostinger, servidores físicos). O único requisito é conectividade de rede entre o agente e o servidor Zabbix.

### Posso contratar Zabbix e GLPI juntos?

Sim. O Zabbix e o GLPI podem ser implantados em conjunto na mesma VPS ou em VPSs separadas. A contratação conjunta é comum em equipes de TI que querem monitoramento e helpdesk na mesma plataforma. Integrações adicionais entre os sistemas podem ser contratadas separadamente.

### Quantos hosts o Zabbix consegue monitorar?

O Zabbix é altamente escalável. A mesma instalação que monitora 10 servidores pode ser expandida para monitorar centenas ou milhares sem mudar a licença — é open-source. Para grandes ambientes (500+ hosts), recomendamos VPS mais robusto (8+ vCPU, 16+ GB RAM) e banco de dados otimizado. A PaperMoon dimensiona a infraestrutura conforme o volume do cliente.

### Posso receber alertas do Zabbix no WhatsApp ou Telegram?

Sim. O Zabbix suporta alertas via e-mail, Telegram, Slack, Teams e qualquer endpoint HTTP (webhook). Para WhatsApp, integramos via n8n: quando o Zabbix dispara um alerta, o n8n recebe o webhook e envia a mensagem no WhatsApp do plantão. A PaperMoon configura o canal de alerta preferido da sua equipe.',
    'Gestão de TI',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS dedicada para o servidor Zabbix com Ubuntu 22.04 (mínimo 2 vCPU, 4 GB RAM, 60 GB SSD)', 'Acesso SSH a todos os servidores e hosts a serem monitorados', 'Lista de hosts, serviços e métricas críticas a monitorar', 'Contatos de alerta (e-mail, WhatsApp ou Telegram)', 'Portas 10050/10051 liberadas no firewall entre agentes e servidor Zabbix']::text[],
    ARRAY['01. Instalação do servidor Zabbix: A PaperMoon instala o Zabbix Server na sua VPS com banco PostgreSQL e interface web configurada. Definimos as credenciais e o acesso HTTPS.', '02. Agentes nos servidores: Instalamos o agente Zabbix nos servidores que você quer monitorar — físicos, VMs ou cloud. Configuramos as métricas: CPU, memória, disco, serviços HTTP/S, latência e outros.', '03. Alertas e dashboards: Quando qualquer métrica cruza o limiar configurado, o Zabbix dispara alertas por e-mail com detalhes do problema, gravidade e identificação do host. Dashboards configuráveis deixam tudo visível em tempo real.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'eye',
    'published',
    50,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'proxmox',
    'Proxmox VE',
    'Virtualização enterprise open-source instalada no seu servidor.',
    '## Sobre o Serviço

O Proxmox VE (Virtual Environment) é desenvolvido desde 2008 pela empresa austríaca Proxmox Server Solutions e se tornou a plataforma de virtualização open-source mais adotada em pequenas e médias empresas. Combina dois motores de virtualização em uma única interface web: o KVM (Kernel-based Virtual Machine) para virtualização completa de qualquer sistema operacional, e o LXC (Linux Containers) para workloads mais leves que não precisam de uma VM dedicada.

O que torna o Proxmox relevante é o que ele oferece sem custo de licença: alta disponibilidade (HA) com failover automático entre nodes, storage distribuído via Ceph, live migration de VMs sem downtime, backup integrado com o Proxmox Backup Server e uma API REST completa para automação via Terraform ou Ansible. Ferramentas equivalentes da VMware chegam a custar 5 vezes mais em licenciamento.

A PaperMoon realiza a instalação do Proxmox de forma presencial — um técnico vai ao local do servidor, configura o ambiente de boot, a rede de gerenciamento, o storage ZFS e as VMs iniciais. Treinamos a equipe de TI para operar com autonomia: criação de VMs, snapshots, restore e expansão do cluster.

## Diferenciais

- Virtualização enterprise sem custo de licença — VMware custa até 5× mais pelo mesmo recurso
- KVM e LXC em uma interface única: VMs completas e containers no mesmo painel
- Alta disponibilidade (HA) e live migration incluídos, sem licença adicional
- Instalação presencial com configuração de ZFS, VLANs e VMs iniciais já no dia da entrega
- Treinamento completo da equipe de TI para operar sem depender da PaperMoon no dia a dia

## O que nós entregamos

- Instala e configura o Proxmox VE no servidor dedicado
- Configura pools de storage (ZFS, LVM ou Ceph conforme hardware)
- Cria as VMs e containers LXC iniciais conforme a necessidade
- Configura rede interna (SDN), VLANs e acesso seguro ao painel
- Implementa rotina de backup agendado (Proxmox Backup Server ou remoto)
- Documenta a infraestrutura e realiza treinamento da equipe de TI

## Responsabilidades do Cliente

- Fornecer servidor dedicado ou bare-metal e agendar visita presencial para a instalação
- Definir quais VMs e serviços serão criados inicialmente
- Gerenciar VMs e containers no dia a dia após o treinamento
- Contratar storage adicional se necessário (NFS, SAN, etc.)

## Recursos e Funcionalidades

### Virtualização

- VMs KVM com suporte a Windows, Linux e qualquer SO
- Containers LXC para serviços mais leves e rápidos
- Snapshots e clones em segundos
- Live migration entre nodes do cluster sem downtime
- CPU e memória configuráveis por VM individualmente

### Storage e backup

- ZFS com RAID-Z para redundância e integridade de dados
- Ceph para storage distribuído em clusters maiores
- Backup agendado com retenção configurável
- Restore granular: VM inteira ou arquivo específico
- Replicação para servidor remoto (disaster recovery)

### Gestão e rede

- Painel web completo — sem linha de comando para operações rotineiras
- API REST para automação e integração com IaC (Terraform)
- SDN para redes virtuais e VLANs isoladas
- Alta disponibilidade (HA) com failover automático
- Controle de acesso por usuário e permissão granular

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do Proxmox é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração e treinamento. Expansões futuras (novos nodes, VMs, integrações) são contratadas sob demanda.

### Que tipo de servidor preciso ter?

O Proxmox VE roda em qualquer servidor x86-64 com suporte a virtualização (Intel VT-x / AMD-V habilitado). Para uso produtivo, recomendamos no mínimo 32 GB RAM e discos SSD. Servidores dedicados (Hetzner, OVH, Contabo) funcionam muito bem. Consulte nossa equipe para avaliar o hardware disponível.

### Posso montar um cluster com múltiplos servidores?

Sim. O Proxmox suporta clusters de até centenas de nodes com alta disponibilidade (HA). A PaperMoon configura o cluster, a rede de corosync e as políticas de failover automático. Para storage compartilhado em cluster, utilizamos Ceph ou NFS.

### Vocês oferecem treinamento após a implantação?

Sim. O treinamento está incluído no escopo. Após a entrega, realizamos sessões com a equipe de TI cobrindo: criação e gerenciamento de VMs, snapshots, restore de backup, migração live e monitoramento de recursos. O número de sessões é definido no momento da contratação.

### O Proxmox VE tem custo de licença?

O Proxmox VE é open-source e gratuito — pode ser instalado e usado sem pagar nenhuma licença. A Proxmox Server Solutions oferece assinaturas de suporte empresarial opcional com atualizações de repositório Enterprise, mas elas não são obrigatórias. A PaperMoon instala o Proxmox com repositório community (gratuito) ou Enterprise conforme a preferência do cliente.',
    'Infraestrutura',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Servidor bare-metal dedicado com processador x86-64 compatível com virtualização (Intel VT-x / AMD-V)', 'Mínimo 32 GB RAM e 2 discos SSD (um para o sistema, um ou mais para VMs)', 'Acesso físico ou KVM-over-IP ao servidor para a instalação', 'Rede local com switch gerenciável para configuração de VLANs (recomendado)']::text[],
    ARRAY['01. Instalação do Proxmox VE: A instalação do Proxmox VE é realizada de forma presencial: um técnico da PaperMoon vai até o local onde o servidor está instalado, realiza o boot pelo pendrive, instala o Proxmox VE, configura a rede de gerenciamento e o painel web com HTTPS. Definimos senhas, usuários e políticas de acesso.', '02. Storage, rede e VMs: Configuramos os pools de storage (ZFS para redundância, LVM para performance), criamos as VMs e containers LXC conforme seu inventário, e configuramos as VLANs e redes internas necessárias.', '03. Backup, HA e treinamento: Implementamos a rotina de backup agendado, configuramos alertas de utilização e realizamos o treinamento da equipe: criar VMs, snapshots, migração live, restore e gestão do cluster.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'server',
    'published',
    60,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'truenas',
    'TrueNAS',
    'Storage centralizado open-source com ZFS na sua rede.',
    '## Sobre o Serviço

O TrueNAS é descendente direto do FreeNAS, lançado em 2005 como o primeiro sistema operacional open-source focado exclusivamente em armazenamento NAS baseado em FreeBSD. Desde 2011 é desenvolvido e mantido pela iXsystems, empresa californiana especializada em storage enterprise. Hoje o projeto possui duas versões: TrueNAS CORE (FreeBSD, mais maduro) e TrueNAS SCALE (baseado em Debian Linux, com suporte a containers e apps).

O grande diferencial do TrueNAS é o sistema de arquivos ZFS — desenvolvido pela Sun Microsystems e considerado o sistema de arquivos mais confiável para armazenamento de dados. O ZFS usa checksums automáticos em todos os blocos de dados, detectando e corrigindo silenciosamente erros de hardware antes que causem corrupção. Diferente do RAID tradicional, o ZFS garante a integridade dos dados mesmo após falhas de disco simultâneas.

A PaperMoon implanta o TrueNAS no hardware dedicado do cliente — um servidor com múltiplos discos ou um NAS dedicado — configura os pools de armazenamento com a redundância ideal (RAID-Z1, Z2 ou Z3 conforme o nível de risco aceitável), cria os datasets com quotas por departamento e ativa o compartilhamento NFS para Linux e SMB para Windows e macOS.

## Diferenciais

- ZFS com checksums automáticos: proteção contra corrupção silenciosa de dados (bit rot)
- Snapshots instantâneos e agendados sem impacto de performance — volte ao estado de ontem em segundos
- Replicação para site remoto: backup offsite automático via ZFS send/receive
- Integração nativa com Nextcloud para transformar o NAS em nuvem privada de arquivos
- Hardware neutro: roda em qualquer servidor com múltiplos discos, sem lock-in de fabricante

## O que nós entregamos

- Instala e configura o TrueNAS no hardware ou servidor dedicado
- Configura pools ZFS com RAID-Z para redundância e integridade de dados
- Cria shares NFS/SMB para acesso da rede local
- Configura snapshots agendados com retenção configurável
- Implementa replicação para servidor remoto (disaster recovery)
- Documenta o ambiente e realiza treinamento da equipe

## Responsabilidades do Cliente

- Fornecer hardware dedicado ou servidor com discos para o TrueNAS
- Definir estrutura de compartilhamentos e permissões por departamento
- Gerenciar usuários e permissões de acesso no dia a dia
- Monitorar o espaço em disco e saúde dos discos

## Recursos e Funcionalidades

### Storage e confiabilidade

- ZFS com RAID-Z — proteção contra corrupção de dados
- Snapshots automáticos agendados com retenção configurável
- Replicação para servidor remoto (disaster recovery)
- Compressão e deduplicação automática de dados
- Scrub agendado para verificação de integridade

### Compartilhamento de arquivos

- SMB/CIFS para Windows — integração nativa com Explorador de Arquivos
- NFS para Linux/macOS e containers
- iSCSI para volumes de bloco (VMs, bancos de dados)
- Permissões granulares por usuário, grupo ou IP
- ACLs compatíveis com Active Directory

### Gestão e monitoramento

- Painel web completo — sem linha de comando para operações rotineiras
- Alertas de saúde dos discos (S.M.A.R.T.) por e-mail
- Estatísticas de uso por dataset em tempo real
- API REST para integração com sistemas de monitoramento
- Suporte a plugins e VMs no TrueNAS Scale

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do TrueNAS é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração dos pools ZFS, shares, snapshots e treinamento. Adição de discos, novos datasets e integrações futuras são contratadas sob demanda.

### Que hardware preciso ter?

O TrueNAS roda em servidores x86-64 com múltiplos discos. Para uso produtivo, recomendamos no mínimo 16 GB RAM (o ZFS usa RAM para cache) e discos dedicados (não compartilhados com o sistema operacional). A PaperMoon avalia o hardware disponível e indica a configuração ideal de RAID-Z.

### Posso usar junto com o Proxmox?

Sim. A combinação TrueNAS + Proxmox é muito comum: o TrueNAS fornece o storage compartilhado via NFS ou iSCSI para o cluster Proxmox, que usa o espaço para armazenar discos de VMs e containers. A PaperMoon configura a integração entre os dois sistemas.

### O TrueNAS substitui um NAS de marca (Synology, QNAP)?

Para a maioria dos casos de uso empresarial, sim. O TrueNAS oferece funcionalidades equivalentes ou superiores (ZFS, replicação, iSCSI) sem custo de licença de software — você paga apenas pelo hardware. A principal diferença é que a configuração inicial é mais técnica, por isso a PaperMoon cuida de toda a implantação.

### Qual a diferença entre TrueNAS CORE e TrueNAS SCALE?

TrueNAS CORE é baseado em FreeBSD — mais maduro e estável, ideal para NAS dedicado. TrueNAS SCALE é baseado em Debian Linux — suporta containers Docker e VMs além do storage, ideal para quem quer consolidar NAS e aplicações no mesmo hardware. A PaperMoon recomenda a versão conforme o caso de uso: CORE para storage puro, SCALE para ambientes que precisam de aplicações rodando junto.',
    'Infraestrutura',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Servidor ou hardware dedicado x86-64 com suporte a ECC RAM (recomendado)', 'Mínimo 16 GB RAM (ZFS usa memória como cache) e discos dedicados separados do SO', 'Acesso físico ou remoto (IPMI/iDRAC) para a instalação', 'Rede local estruturada com capacidade para tráfego de armazenamento (1 Gbps mínimo)']::text[],
    ARRAY['01. Instalação e configuração ZFS: A PaperMoon instala o TrueNAS no hardware definido e configura os pools ZFS com RAID-Z conforme o número de discos disponíveis. Definimos nome de host, rede de gerenciamento e acesso HTTPS.', '02. Shares, snapshots e replicação: Criamos os datasets por departamento, configuramos as shares NFS/SMB com permissões por usuário ou grupo, habilitamos snapshots automáticos com retenção e configuramos replicação para backup remoto.', '03. Integração de rede e treinamento: Integramos o TrueNAS com a rede existente (Active Directory/LDAP se necessário) e treinamos a equipe para gerenciar datasets, shares, snapshots e restaurações de backup pelo painel web.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'hard-drive',
    'published',
    70,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'nextcloud',
    'Nextcloud',
    'Nuvem privada e colaboração instalada na sua VPS.',
    '## Sobre o Serviço

O Nextcloud nasceu em 2016 como um fork do ownCloud, criado pelo próprio fundador original do projeto, Frank Karlitschek, após divergências sobre o modelo de negócios. Desde então cresceu para se tornar a maior plataforma de colaboração auto-hospedada do mundo, com mais de 400 mil instalações conhecidas e suporte ativo de empresas como Deutsche Telekom, Red Hat e Siemens.

Mais do que um simples substituto do Google Drive, o Nextcloud é uma suíte completa: arquivos com versionamento e links de compartilhamento, calendário e contatos sincronizados com qualquer dispositivo, videoconferência via Nextcloud Talk, edição colaborativa de documentos com Nextcloud Office (baseado no LibreOffice), e integração com Active Directory e LDAP para grandes organizações. Para empresas que precisam de conformidade com a LGPD, ter os dados na própria infraestrutura — e não em servidores da Google ou Microsoft — é um requisito não negociável.

A PaperMoon instala o Nextcloud com configuração de produção completa (Nginx, PHP-FPM, Redis para cache de sessão, Let''s Encrypt SSL), integra com o armazenamento local ou com o TrueNAS instalado na rede, e configura as contas de usuário com os grupos e permissões conforme a estrutura da empresa.

## Diferenciais

- Alternativa completa ao Google Workspace e Microsoft 365 — sem mensalidade por usuário
- Seus arquivos ficam na sua VPS: conformidade com LGPD sem depender de terceiros
- Videoconferência (Talk), edição de documentos (Office) e calendário incluídos na mesma plataforma
- Integração com Active Directory/LDAP: login com a mesma credencial de rede
- Conexão nativa com TrueNAS para usar o storage NAS como backend de arquivos

## O que nós entregamos

- Instala e configura o Nextcloud na sua VPS (Docker ou nativo)
- Configura armazenamento local ou integrado (S3, Minio, NFS)
- Instala e parametriza os apps: Talk, Calendar, Contacts, OnlyOffice
- Configura SSL, e-mail de notificação e políticas de segurança
- Migra arquivos existentes (Google Drive, OneDrive, pasta local)
- Documenta o ambiente e realiza treinamento da equipe

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o Nextcloud é instalado
- Definir políticas de compartilhamento e acesso por usuário
- Gerenciar usuários e grupos no dia a dia após o treinamento
- Contratar storage adicional conforme o crescimento dos dados

## Recursos e Funcionalidades

### Arquivos e sincronização

- Sync automático para desktop (Windows, macOS, Linux)
- App mobile (iOS e Android) para acesso offline
- Compartilhamento com link, senha e prazo de validade
- Versionamento de arquivos com histórico de alterações
- Suporte a WebDAV para integração com ferramentas externas

### Colaboração

- Nextcloud Talk — videoconferência e chat interno
- Calendário compartilhado por equipe com sincronização CalDAV
- Agenda de contatos com CardDAV
- Edição colaborativa de documentos (OnlyOffice ou Collabora)
- Kanban e formulários internos

### Segurança e controle

- Todos os dados ficam na sua VPS — sem Google, sem Microsoft
- Autenticação em dois fatores (TOTP, WebAuthn)
- Integração opcional com Active Directory / LDAP
- Logs de auditoria: quem acessou, baixou ou compartilhou o quê
- Criptografia ponta a ponta para pastas sensíveis

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do Nextcloud é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração, migração de dados e treinamento. Suporte técnico posterior é contratado sob demanda.

### Quantos usuários posso ter?

Não há limite de usuários — o Nextcloud é instalado na sua VPS e o limite real é a capacidade do servidor. Para equipes de até 50 usuários, um VPS de 4 vCPUs e 8 GB RAM com SSD é suficiente. Para mais usuários ou volumes maiores, orientamos a configuração ideal.

### É possível migrar meus arquivos do Google Drive?

Sim. A PaperMoon realiza a migração dos arquivos existentes (Google Drive, OneDrive, Dropbox ou servidor local) durante a implantação. A migração mantém a estrutura de pastas e, quando possível, o histórico de versões.

### Vocês oferecem treinamento após a implantação?

Sim. O treinamento está incluído no escopo. Cobrimos: uso do cliente desktop e mobile, compartilhamento de arquivos, videoconferência com o Talk, calendário compartilhado e edição de documentos online. O número de sessões é definido na contratação.

### O Nextcloud funciona no celular?

Sim. O Nextcloud tem apps oficiais para iOS e Android que sincronizam arquivos offline, acessam o calendário, contatos e câmera do Nextcloud Talk para videoconferência. Os apps são gratuitos e se conectam diretamente ao seu servidor — sem passar por nuvem de terceiros.',
    'Infraestrutura',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 80 GB SSD — escalar conforme volume de arquivos)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo']::text[],
    ARRAY['01. Instalação e configuração base: A PaperMoon instala o Nextcloud na sua VPS com banco PostgreSQL, Redis para cache e HTTPS configurado. Definimos cotas de armazenamento, políticas de senha e domínio personalizado.', '02. Apps e integrações: Instalamos e configuramos os apps conforme sua necessidade: Nextcloud Talk (videoconferência), Calendar, Contacts, OnlyOffice ou Collabora para edição de documentos online, e integração com armazenamento externo.', '03. Migração e treinamento: Migramos seus arquivos do Drive, OneDrive ou servidor atual. Após a entrega, treinamos a equipe no uso do Nextcloud desktop, mobile e web — garantindo autonomia completa.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'cloud',
    'published',
    80,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'aapanel',
    'AAPanel',
    'Painel web para hospedar sites e gerenciar servidores com facilidade.',
    '## Sobre o Serviço

O aaPanel (baota.com no mercado chinês) foi desenvolvido pela Baota Internet Technology e lançado internacionalmente em 2018 como alternativa gratuita ao cPanel e ao Plesk — ferramentas tradicionais de hospedagem que cobram licenças anuais de R$ 600 a R$ 3.000. Hoje gerencia mais de 6 milhões de servidores em todo o mundo e é amplamente adotado por agências digitais, desenvolvedores freelancers e empresas que precisam hospedar múltiplos sites em um único servidor sem custo de licença.

O aaPanel resolve o principal problema do gerenciamento de servidores web: a complexidade de instalar e manter manualmente Nginx, Apache, PHP, MySQL, SSL e FTP — tarefas que exigem conhecimento avançado de Linux. Com o aaPanel, todas essas operações são realizadas via interface web com poucos cliques, incluindo instalação de aplicações como WordPress, Joomla ou Laravel com um único botão.

A PaperMoon instala o aaPanel na VPS do cliente, configura a stack inicial (Nginx + PHP + MySQL ou MariaDB), implanta os primeiros sites, emite os certificados SSL e configura as rotinas de backup automático para armazenamento local ou remoto (FTP, S3). Realizamos treinamento para que a equipe do cliente gerencie novos sites e usuários com autonomia.

## Diferenciais

- Zero custo de licença — nenhuma anuidade versus R$ 600+/ano do cPanel ou Plesk
- Múltiplos sites no mesmo servidor com domínios, PHPs e bancos de dados isolados
- Instalação com 1 clique: WordPress, Laravel, PrestaShop e mais de 30 aplicações
- SSL automático via Let''s Encrypt com renovação sem intervenção manual
- Interface em português com painel FTP, banco de dados e monitor de recursos no mesmo lugar

## O que nós entregamos

- Instala e configura o AAPanel na sua VPS (Ubuntu ou CentOS)
- Configura a stack web: Nginx + PHP (multi-versão) + MySQL/MariaDB
- Cria os sites e vhosts iniciais com SSL (Let''s Encrypt automático)
- Aplica hardening de segurança: firewall, fail2ban e WAF
- Configura rotina de backup automático dos sites e banco de dados
- Documenta o ambiente e realiza treinamento da equipe

## Responsabilidades do Cliente

- Contratar e manter a VPS onde o AAPanel é instalado
- Fornecer domínios e credenciais de DNS para configuração
- Gerenciar sites, bancos de dados e atualizações no dia a dia
- Monitorar o espaço em disco e recursos do servidor

## Recursos e Funcionalidades

### Hospedagem de sites

- Múltiplos sites no mesmo servidor com vhosts isolados
- PHP 7.x, 8.x e múltiplas versões simultâneas por site
- SSL automático com Let''s Encrypt e renovação automática
- Redirecionamentos, regras Nginx e configurações por site
- Suporte a WordPress, Laravel, CodeIgniter e qualquer app PHP

### Banco de dados e arquivos

- MySQL/MariaDB com phpMyAdmin integrado
- Gerenciador de arquivos visual no painel
- FTP com usuários isolados por site
- Gerenciamento de cron jobs pela interface

### Segurança e backup

- Firewall de portas gerenciado pelo painel
- Fail2ban integrado para proteção contra brute-force
- WAF básico (ModSecurity) para proteção das aplicações
- Backup agendado de sites e bancos para armazenamento remoto
- Monitoramento de recursos: CPU, memória, disco e rede

## Perguntas Frequentes (FAQ)

### A cobrança é mensal ou única?

A implantação do AAPanel é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração, hardening de segurança e treinamento. Suporte técnico posterior (ajustes, novos sites, atualizações) é contratado sob demanda.

### AAPanel é gratuito. Por que contratar a PaperMoon?

O AAPanel é gratuito para instalar, mas a configuração segura e profissional exige experiência: hardening do servidor, configuração correta de Nginx e PHP, fail2ban, WAF, backup remoto e SSL. Erros de configuração podem deixar o servidor vulnerável ou sites fora do ar. A PaperMoon entrega o ambiente configurado corretamente e documentado para a sua operação.

### Consigo migrar sites de outro servidor?

Sim. A PaperMoon faz a migração completa: arquivos, bancos de dados, configurações de e-mail e DNS. O processo é feito com zero ou mínimo downtime usando snapshot/sync antes do corte final.

### Vocês oferecem treinamento após a implantação?

Sim. O treinamento está incluído no escopo. Ensinamos como criar e gerenciar sites, bancos de dados, contas FTP, instalar extensões PHP, configurar cron jobs e restaurar backups pelo painel do AAPanel.

### O AAPanel suporta múltiplos domínios no mesmo servidor?

Sim. Um dos principais pontos fortes do AAPanel é justamente hospedar múltiplos sites no mesmo servidor — cada domínio tem seu próprio vhost, versão de PHP, banco de dados e certificado SSL, completamente isolados uns dos outros. Não há limite de domínios além da capacidade do servidor.',
    'Infraestrutura',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS ou CentOS 7/8 (mínimo 1 vCPU, 1 GB RAM, 20 GB SSD)', 'Acesso SSH à VPS com usuário root', 'IP estático ou domínio para acesso ao painel de controle']::text[],
    ARRAY['01. Instalação e stack web: A PaperMoon instala o AAPanel na sua VPS e configura a stack completa: Nginx (ou Apache), PHP nas versões que você precisar, MySQL/MariaDB e Redis. O painel fica acessível em HTTPS com autenticação segura.', '02. Sites, SSL e segurança: Criamos os vhosts para os domínios existentes, ativamos SSL automático via Let''s Encrypt, configuramos o firewall interno, fail2ban contra brute-force e WAF básico para proteger as aplicações.', '03. Backup e treinamento: Configuramos backups automáticos (diários ou semanais) para armazenamento local ou FTP/S3 remoto. Treinamos sua equipe para gerenciar sites, criar bancos, instalar extensões PHP e restaurar backups pelo painel.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'layout-dashboard',
    'published',
    90,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'whatsapp-evolution',
    'WhatsApp via Evolution API',
    'Chatwoot multiagente + n8n + Evolution API self-hosted na sua VPS.',
    '## Sobre o Serviço

Para empresas que não se qualificam imediatamente para a API oficial da Meta — como números pré-pagos, contas ainda não verificadas ou equipes que precisam de uma alternativa self-hosted sem depender da aprovação inicial da Meta — a Evolution API conectada ao WhatsApp Web é uma das soluções mais completas disponíveis no mercado brasileiro.

Diferente do WhatsApp Business app (que limita a um dispositivo), a Evolution API permite múltiplas instâncias WhatsApp em um único servidor, cada uma com seu próprio número, todas integradas ao Chatwoot para atendimento multiagente e ao n8n para automação de fluxos. A solução roda inteiramente na VPS do cliente — seus dados de atendimento nunca passam por servidores de terceiros.

A PaperMoon entrega o stack completo na sua VPS: Evolution API + Chatwoot + n8n, todos configurados e integrados entre si. Fluxos de qualificação, bot de FAQ, roteamento por departamento e histórico completo de conversas por contato — preparados conforme a operação do cliente. E quando sua empresa se qualificar para a API oficial da Meta, a PaperMoon realiza a migração sem perda de histórico.

## Diferenciais

- Stack completo entregue pronto: Evolution API + Chatwoot + n8n já integrados
- Múltiplos números WhatsApp em um único servidor, cada um com fluxos independentes
- Independente da aprovação inicial da Meta Business Manager para começar a estruturar a operação
- Caminho de migração garantido para a API oficial Meta quando a empresa se qualificar
- Sem custo por mensagem cobrado pela PaperMoon — a cobrança é pela implantação

## O que nós entregamos

- Instala e configura Evolution API, Chatwoot e n8n na sua VPS
- Conecta as instâncias WhatsApp via QR Code e configura reconexão automática
- Cria os fluxos iniciais de triagem e automação no n8n
- Integra a Evolution API com o Chatwoot para atendimento multiagente
- Documenta o ambiente e treina a equipe
- Suporte mensal incluso durante toda a assinatura

## Responsabilidades do Cliente

- Contratar e manter a VPS onde os serviços são instalados
- Fornecer os números de WhatsApp que serão conectados
- Respeitar os limites de mensagens para evitar bloqueio do número
- Gerenciar instâncias e reconexões no dia a dia

## Recursos e Funcionalidades

### WhatsApp e mensagens

- Múltiplas instâncias WhatsApp em um único servidor
- Envio de texto, imagens, documentos, áudio e vídeo
- Reconexão automática em caso de queda
- Sem custo por mensagem (diferente da API oficial Meta)
- Grupos, listas de transmissão e reações

### Automação com n8n

- Fluxos de triagem antes de chegar ao agente
- Bot de perguntas frequentes 24/7
- Roteamento por palavra-chave, horário ou departamento
- Integração com CRM, planilhas e sistemas internos
- Configuração personalizada por cliente

### Atendimento com Chatwoot

- Múltiplos agentes no mesmo número WhatsApp
- Transferência e atribuição de conversas
- Etiquetas, notas internas e histórico completo
- Métricas de SLA e desempenho por agente
- Canned responses para respostas rápidas

## Perguntas Frequentes (FAQ)

### Qual a diferença do plano WhatsApp via API Meta?

A Evolution API usa o protocolo WhatsApp Web (não oficial), sem necessidade de aprovação da Meta e sem custo por mensagem. Mais agilidade para automações internas e menor custo operacional. A contrapartida é o risco de bloqueio do número em uso intenso. Para atendimento comercial em maior escala com total segurança, recomendamos o plano WhatsApp via API Meta (oficial).

### A cobrança é mensal?

Sim. A assinatura mensal inclui instalação, configuração, manutenção e suporte do ambiente completo (Evolution API + Chatwoot + n8n) na sua VPS. Não há custo por mensagem cobrado pela PaperMoon.

### Posso usar vários números no mesmo servidor?

Sim. A Evolution API suporta múltiplas instâncias WhatsApp no mesmo servidor. A PaperMoon configura o número de instâncias acordado na contratação e pode ampliar conforme a demanda cresce.

### O que acontece se o número for bloqueado?

A PaperMoon orienta sobre limites seguros de volume e boas práticas para minimizar o risco. Em caso de bloqueio, auxiliamos na reconexão e na revisão dos fluxos. A responsabilidade pelo uso adequado do número é do cliente.

### Posso usar vários atendentes com a Evolution API no Chatwoot?

Sim. Toda a stack entregue pela PaperMoon (Evolution API + Chatwoot + n8n) suporta múltiplos agentes atendendo os números conectados. O Chatwoot distribui as conversas por agente ou equipe, com atribuição manual ou automática. Não há limite de agentes — o limite real é a capacidade do servidor.',
    'Comunicação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo', 'Números de WhatsApp disponíveis para escanear o QR Code de ativação']::text[],
    ARRAY['01. Contrate uma VPS: Você contrata um servidor (Hostinger, HostGator, AWS...) e nos passa o acesso SSH. Indicamos as especificações mínimas conforme o volume de mensagens esperado.', '02. Instalamos tudo na sua VPS: A PaperMoon instala e configura Evolution API, Chatwoot e n8n. Conectamos suas instâncias WhatsApp via QR Code, configuramos reconexão automática e criamos os primeiros fluxos no n8n.', '03. Treinamento e suporte contínuo: Treinamos sua equipe para gerenciar instâncias, criar fluxos no n8n e atender no Chatwoot. O suporte mensal incluso garante que o ambiente continue funcionando.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'message-circle',
    'published',
    100,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'evolution-api',
    'Evolution API',
    'API WhatsApp self-hosted para automação via n8n e Chatwoot.',
    '## Sobre o Serviço

A Evolution API é um projeto open-source brasileiro criado em 2022 pelo desenvolvedor Davidson Gomes e mantido por uma comunidade ativa no GitHub. Surgiu como resposta à falta de uma solução nacional robusta para quem precisava de um gateway WhatsApp self-hosted com recursos enterprise: múltiplas instâncias, webhooks granulares por evento, suporte a TypeBot e integração nativa com as principais ferramentas de automação do mercado.

O que torna a Evolution API relevante é sua profundidade de integração: cada instância WhatsApp pode ter webhooks independentes configurados por tipo de evento (mensagem recebida, status de entrega, mudança de grupo), o que a torna ideal como camada de backend para bots, sistemas de notificação e plataformas de atendimento. A API RESTful bem documentada facilita integrações customizadas com sistemas internos das empresas.

A PaperMoon instala a Evolution API em servidor dedicado na VPS do cliente, configura as instâncias necessárias e realiza a integração completa com o n8n (para fluxos de automação) e o Chatwoot (para handoff ao atendimento humano). A combinação das três ferramentas na mesma VPS forma um stack de atendimento completo e auto-hospedado.

## Diferenciais

- Projeto 100% brasileiro — documentação, comunidade e suporte em português
- Webhooks granulares por tipo de evento: configure reações diferentes para cada ação do WhatsApp
- Integração nativa com n8n e Chatwoot — stack completo de atendimento em uma única VPS
- TypeBot integrado para bots conversacionais avançados com menus, perguntas e condições
- Código aberto e auditável: nenhuma caixa-preta entre sua empresa e o gateway WhatsApp

## O que nós entregamos

- Instala e configura a Evolution API na sua VPS
- Conecta as instâncias WhatsApp via QR Code e configura reconexão automática
- Integra com o n8n para automação de fluxos de mensagens
- Integra com o Chatwoot para atendimento multiagente
- Configura webhooks para recebimento de eventos em tempo real
- Documenta o ambiente e realiza treinamento da equipe

## Responsabilidades do Cliente

- Contratar e manter a VPS onde a Evolution API é instalada
- Fornecer os números de WhatsApp que serão conectados
- Gerenciar as instâncias e reconexões no dia a dia
- Respeitar os limites de mensagens para evitar bloqueio do número

## Recursos e Funcionalidades

### Instâncias e mensagens

- Múltiplas instâncias WhatsApp em um único servidor
- Envio de texto, imagens, documentos, áudio e vídeo
- Envio de mensagens para grupos e listas de transmissão
- Reações, edição e exclusão de mensagens
- Reconexão automática em caso de queda

### Automação

- Webhooks para cada evento (mensagem, status, conexão)
- Integração nativa com n8n para fluxos sem código
- Integração com Chatwoot para handoff ao atendimento humano
- TypeBot para bots conversacionais avançados
- RabbitMQ para filas de alta disponibilidade

### Gestão e API

- API REST completa com documentação Swagger
- Painel de administração web para gerenciar instâncias
- Logs de mensagens enviadas e recebidas
- Suporte a proxy por instância para isolamento de IPs
- Controle de sessões e autenticação por API Key

## Perguntas Frequentes (FAQ)

### Qual a diferença da API oficial da Meta?

A Evolution API usa o protocolo WhatsApp Web (não oficial), sem necessidade de aprovação da Meta e sem custo por mensagem. Isso traz mais agilidade e liberdade para automações internas. A contrapartida é o risco de bloqueio do número pelo WhatsApp se o uso for identificado como automação em massa. Para atendimento comercial em escala, nossa solução de WhatsApp API (Meta Oficial) é mais segura e estável.

### A cobrança é mensal ou única?

A implantação da Evolution API é cobrada como projeto único — sem mensalidade. Você paga uma vez pela instalação, configuração e treinamento. Novas instâncias, fluxos ou integrações adicionais são contratadas sob demanda.

### Posso usar junto com o Chatwoot e n8n?

Sim, essa é a configuração mais comum. A Evolution API recebe as mensagens do WhatsApp, o n8n processa e filtra automaticamente, e as conversas que precisam de atenção humana são encaminhadas ao Chatwoot. A PaperMoon configura toda essa integração.

### Vocês oferecem treinamento após a implantação?

Sim. O treinamento está incluído no escopo. Ensinamos como criar e gerenciar instâncias, verificar o status de conexão, criar fluxos básicos no n8n integrado à Evolution API e monitorar logs de mensagens.

### Posso usar a Evolution API para disparos de marketing em massa?

A Evolution API não é recomendada para campanhas em massa — o WhatsApp detecta padrões de envio em volume e pode bloquear o número. Para disparos de marketing ou notificações em escala, recomendamos a API oficial da Meta (nosso plano WhatsApp via API Meta), que tem limites maiores e número verificado. A Evolution API é ideal para automação de atendimento, bots e integrações de baixo volume.',
    'Comunicação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 1 vCPU, 2 GB RAM, 20 GB SSD)', 'Domínio próprio com acesso ao painel DNS para apontar subdomínio', 'Acesso SSH à VPS com usuário root ou sudo', 'Números de WhatsApp para ativação das sessões']::text[],
    ARRAY['01. Instalação e configuração: A PaperMoon instala a Evolution API na sua VPS com banco de dados, Redis para filas e painel de administração protegido por senha. Configuramos o endpoint da API e as chaves de acesso.', '02. Instâncias e integrações: Criamos as instâncias WhatsApp, escaneamos o QR Code para conexão e configuramos reconexão automática. Integramos com o n8n para automações e com o Chatwoot para atendimento humano quando necessário.', '03. Webhooks e treinamento: Configuramos os webhooks para eventos (mensagem recebida, status de entrega, conexão) e treinamos sua equipe para gerenciar instâncias, criar fluxos no n8n e monitorar o status das conexões.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'message-circle',
    'published',
    110,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'redes',
    'Redes corporativas',
    'Projeto, implantação e configuração de rede para empresas.',
    '## Sobre o Serviço

A infraestrutura de rede é a fundação de toda a operação de TI de uma empresa. Redes corporativas bem projetadas garantem que sistemas críticos — ERP, VoIP, câmeras de segurança e servidores — tenham a largura de banda e o isolamento corretos, enquanto redes domésticas improvisadas causam lentidão, falhas de segurança e gargalos que paralisam a operação.

A PaperMoon atua desde o projeto até a entrega certificada: levantamos os requisitos de cada setor (escritório, depósito, sala de reunião, CFTV), desenhamos a topologia com segmentação por VLANs para isolar tráfego crítico do tráfego geral, dimensionamos o hardware de switching e roteamento, configuramos o firewall com políticas de acesso granulares e implantamos o Wi-Fi empresarial com autenticação 802.1X quando necessário.

Entregamos o projeto com documentação completa: diagrama de rede em formato editável, inventário de equipamentos e configurações, e relatório de certificação de cada ponto de rede — essencial para auditoria e futuras expansões. O monitoramento do ambiente pode ser configurado em conjunto com o Zabbix.

## Diferenciais

- Projeto de rede com cálculo de capacidade por setor — não apenas a ligação dos cabos
- Segmentação por VLANs: câmeras, VoIP e servidores isolados do tráfego de usuários
- Firewall gerenciado com políticas de acesso entre VLANs e para a internet
- Documentação completa: diagrama editável + inventário + certificação de todos os pontos
- Integração com Zabbix para monitoramento pró-ativo da rede após a entrega

## O que nós entregamos

- Realiza levantamento técnico do ambiente e propõe o diagrama de rede
- Configura switches gerenciáveis (VLANs, STP, LACP)
- Implanta e configura roteadores e firewalls (pfSense, MikroTik, Fortinet)
- Configura Wi-Fi empresarial com autenticação WPA2-Enterprise ou RADIUS
- Implementa VPN site-to-site e acesso remoto seguro
- Entrega documentação completa e diagrama lógico/físico da rede

## Responsabilidades do Cliente

- Fornecer acesso ao local e à infraestrutura elétrica existente
- Aprovar o projeto de rede antes do início da execução
- Adquirir os equipamentos indicados no projeto (ou solicitar fornecimento à PaperMoon)
- Definir VLANs, segmentações e políticas de acesso desejadas

## Recursos e Funcionalidades

### Switching e roteamento

- VLANs por departamento, dispositivo ou função
- Spanning Tree Protocol (STP/RSTP) para redundância
- Link Aggregation (LACP) para maior largura de banda
- Roteamento estático e dinâmico (OSPF, BGP quando necessário)
- QoS para priorização de VoIP e videoconferência

### Segurança e acesso

- Firewall com regras por VLAN e por usuário
- Wi-Fi empresarial WPA2-Enterprise com autenticação RADIUS
- VPN SSL e site-to-site para acesso remoto seguro
- Segmentação de rede de visitantes (guest network isolada)
- Monitoramento de tráfego e alertas de anomalias

### Documentação e suporte

- Diagrama lógico e físico da rede entregue em PDF e Visio
- Credenciais e acesso documentados em cofre de senhas
- Suporte técnico por 30 dias após a entrega
- Treinamento da equipe de TI interna
- Expansão e ajustes contratados sob demanda

## Perguntas Frequentes (FAQ)

### O serviço é presencial?

Sim. Redes corporativas exigem presença física no local — levantamento, configuração de equipamentos e certificação são realizados presencialmente pela equipe da PaperMoon. Configurações remotas de equipamentos já instalados podem ser realizadas remotamente.

### Vocês fornecem os equipamentos (switches, roteadores)?

O padrão é o cliente adquirir os equipamentos — indicamos marca e modelo conforme o orçamento e a necessidade. Mas podemos incluir o fornecimento no escopo do projeto mediante cotação.

### Como funciona a cobrança?

Projetos de rede são cobrados por escopo — um valor único pelo levantamento, execução e documentação. Não há mensalidade. Expansões e ajustes futuros são cotados separadamente.

### Quanto tempo leva a implantação?

O cronograma depende do tamanho da rede, da quantidade de pontos, da infraestrutura existente e das janelas de execução disponíveis no cliente. A PaperMoon detalha o plano por fases antes do início da implantação.

### A PaperMoon instala Wi-Fi empresarial?

Sim. Configuramos redes Wi-Fi corporativas com controladora, autenticação WPA2-Enterprise (802.1X + RADIUS), rede de visitantes isolada e cobertura dimensionada por planta baixa. Para grandes áreas, desenhamos o projeto de APs com mapas de calor para garantir sinal uniforme. Trabalhamos com Ubiquiti, TP-Link Omada, Aruba e Mikrotik.',
    'Redes e Física',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Planta baixa ou croqui do local com indicação dos pontos de rede desejados', 'Acesso presencial ao espaço físico para levantamento técnico', 'Definição dos equipamentos a serem utilizados (switches, roteadores, APs) ou orçamento para aquisição', 'Contato técnico no local para acompanhar a instalação']::text[],
    ARRAY['01. Levantamento e projeto: Um técnico da PaperMoon visita o local, mapeia o ambiente, conta pontos de rede existentes e elabora o diagrama lógico e físico da rede. O projeto é apresentado e aprovado antes de qualquer execução.', '02. Implantação e configuração: Com o projeto aprovado, nossa equipe executa a configuração dos switches, roteadores e firewall — VLANs, QoS, rotas, regras de firewall, Wi-Fi e VPN — tudo devidamente testado antes da entrega.', '03. Certificação e documentação: Ao final, certificamos a rede, medimos latência e throughput por segmento e entregamos o diagrama atualizado, credenciais e manual de operação para a equipe de TI do cliente.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'network',
    'published',
    120,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'cabeamento',
    'Cabeamento estruturado',
    'Instalação certificada Cat5e, Cat6 e fibra óptica.',
    '## Sobre o Serviço

O cabeamento estruturado é a espinha dorsal física de qualquer rede corporativa. Diferente do cabeamento improvisado — onde cabos percorrem o ambiente sem identificação, fixação adequada ou margem para expansão — o cabeamento estruturado segue as normas ABNT NBR 14565 (Brasil) e TIA/EIA-568 (padrão internacional), garantindo desempenho, organização e facilidade de manutenção por décadas.

A categoria do cabo define o limite de velocidade e distância: Cat5e suporta até 1 Gbps em 100 metros, Cat6 chega a 10 Gbps em distâncias de até 55 metros, e Cat6A mantém os 10 Gbps em 100 metros com melhor blindagem contra interferências. Para infraestrutura de câmeras de segurança ou conexão entre andares e prédios, a fibra óptica é a escolha certa — imune a interferências eletromagnéticas e com alcance de quilômetros.

A PaperMoon realiza o projeto, a instalação e a certificação do cabeamento com certificador homologado, gerando o laudo técnico por ponto de rede — documento obrigatório para auditorias e necessário para qualquer garantia dos materiais. Cada cabo recebe etiqueta de identificação nas duas extremidades e é organizado no patch panel com numeração padronizada.

## Diferenciais

- Certificação com laudo técnico por ponto — o padrão exigido por auditorias de TI e seguradoras
- Cat6A para 10 Gbps em 100 metros: rede preparada para os próximos 10 anos
- Etiquetagem bilateral e numeração padronizada: qualquer técnico encontra qualquer ponto
- Patch panel organizado com espaço calculado para expansão planejada
- Fibra óptica para interligação de andares e prédios — sem interferência eletromagnética

## O que nós entregamos

- Realiza o projeto de passagem de cabos com planta baixa do local
- Instala cabos Cat5e, Cat6 ou Cat6A em conduítes ou canaletas
- Instala e organiza patch panels em rack ou parede
- Identifica todos os pontos com etiquetas padronizadas
- Realiza a certificação com testador homologado e emite laudo
- Organiza e documenta o rack ao final da instalação

## Responsabilidades do Cliente

- Fornecer acesso ao local, planta baixa e autorização para passagem de cabos
- Aprovar o projeto de passagem antes da execução
- Definir quantidade e localização dos pontos de rede desejados
- Viabilizar acesso a forros, paredes e áreas técnicas conforme necessário

## Recursos e Funcionalidades

### Tipos de cabeamento

- Cat5e — até 1 Gbps, ideal para redes legadas e expansões
- Cat6 — até 10 Gbps em curtas distâncias, padrão atual
- Cat6A — 10 Gbps em até 100m, recomendado para novos projetos
- Fibra óptica monomodo e multimodo para backbone entre andares
- Patch panels e keystones de marcas homologadas

### Organização e identificação

- Etiquetagem padronizada em todos os pontos e patch panels
- Organização com guias de cabos e amarrações no rack
- Documentação fotográfica de toda a instalação
- Mapa de pontos entregue em planilha e PDF
- Rack organizado com espaço reservado para expansão

### Normas e certificação

- Instalação conforme ABNT NBR 14565 e EIA/TIA 568
- Certificação com testador Fluke ou equivalente homologado
- Laudo de certificação entregue para cada ponto
- Atenuação, NEXT, FEXT e Return Loss medidos
- Rastreabilidade completa: número de série do cabo por ponto

## Perguntas Frequentes (FAQ)

### Vocês instalam em forro rebaixado e laje?

Sim. Nossa equipe está preparada para passagem em forro de gesso, forro metálico, laje, parede de alvenaria e drywall. O projeto define a melhor trajetória para cada ambiente.

### Qual a diferença entre Cat5e, Cat6 e Cat6A?

Cat5e suporta até 1 Gbps e é adequado para redes legadas. Cat6 suporta 10 Gbps em curtas distâncias e é o padrão atual. Cat6A suporta 10 Gbps plenos até 100m e é recomendado para instalações novas. A fibra é indicada para backbone entre andares ou prédios.

### A certificação é obrigatória?

Para garantia da instalação e conformidade com normas, sim. A certificação comprova que cada ponto atende aos parâmetros da categoria instalada. Sem ela, não há como garantir o desempenho da rede.

### Como funciona a cobrança?

Cobrado por ponto instalado ou por escopo fechado de projeto. Inclui material (cabos, keystones, patch panel) e mão de obra. O laudo de certificação está incluso. Sem mensalidade.

### Vocês fazem instalação de fibra óptica?

Sim. Instalamos fibra óptica monomodo e multimodo para backbone entre andares, conexão entre prédios e links de alta velocidade no datacenter. Inclui fusão ou conectorização, testes de atenuação e laudo de certificação. Para ambientes com interferência eletromagnética (próximo a motores, geradores ou cabos de alta tensão), a fibra é a única opção confiável.',
    'Redes e Física',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Planta baixa ou croqui do local com indicação dos pontos de rede desejados', 'Acesso presencial ao espaço físico para levantamento técnico', 'Decisão sobre categoria do cabeamento (Cat5e, Cat6 ou Cat6A)', 'Acesso aos eletrodutos, forros ou paredes para passagem dos cabos']::text[],
    ARRAY['01. Levantamento e projeto: Visitamos o local, medimos distâncias, identificamos trajetórias de passagem e elaboramos o projeto de cabeamento com localização de cada ponto. O projeto é aprovado antes de qualquer execução.', '02. Instalação e organização: Nossa equipe passa os cabos em conduítes ou canaletas, instala as tomadas RJ-45, monta e organiza o patch panel no rack, identifica cada ponto com etiquetas padronizadas.', '03. Certificação e laudo: Certificamos cada ponto com testador homologado, verificamos atenuação, NEXT e FEXT conforme a norma, e emitimos o laudo de certificação. A entrega inclui o mapa completo da instalação.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'layers',
    'published',
    130,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'manutencao',
    'Manutenção de servidores',
    'Diagnóstico, limpeza e manutenção preventiva e corretiva.',
    '## Sobre o Serviço

Servidores físicos e equipamentos de TI degradam com o tempo — acúmulo de poeira nos dissipadores eleva a temperatura dos processadores, capacitores das fontes de alimentação perdem capacidade, discos HDD acumulam setores defeituosos e firmwares desatualizados introduzem vulnerabilidades de segurança. A manutenção preventiva é a diferença entre um equipamento que funciona por 10 anos e outro que falha em 4.

A PaperMoon executa manutenção preventiva e corretiva em servidores físicos (Dell, HP, IBM, Supermicro), computadores corporativos e notebooks. O processo começa com diagnóstico completo: temperatura, velocidade de leitura/escrita dos discos (via S.M.A.R.T.), tensão da memória RAM, estado dos capacitores e logs de eventos do sistema. O relatório é apresentado antes de qualquer intervenção, com orçamento aprovado pelo cliente.

Todo o histórico de intervenções é registrado no GLPI (se já implantado pelo cliente), criando um ciclo virtuoso de manutenção rastreável — quando um técnico abre um chamado, já sabe o que foi feito no servidor anteriormente. Para ambientes com SLA definido, oferecemos contratos de manutenção mensal com visita presencial agendada.

## Diferenciais

- Diagnóstico completo antes de qualquer intervenção — cliente aprova o orçamento previamente
- Registro de todas as intervenções no GLPI para rastreabilidade do histórico técnico
- Manutenção preventiva agendada: evitar falhas antes que elas paralisem a operação
- Atendimento presencial para servidores físicos e remoto para diagnóstico de software
- Relatório técnico entregue ao final de cada intervenção preventiva ou corretiva

## O que nós entregamos

- Realiza diagnóstico completo de hardware e software (logs, S.M.A.R.T., memória)
- Executa limpeza interna com soprador e produtos antiestáticos
- Troca de peças defeituosas: HDs, SSDs, memórias, fontes, ventoinhas
- Atualiza BIOS/UEFI e firmware de controladoras RAID e iDRAC/iLO
- Reinstala ou migra sistema operacional preservando dados
- Documenta o inventário de hardware e o resultado do serviço

## Responsabilidades do Cliente

- Fornecer acesso ao equipamento (presencial ou via frete para assistência)
- Informar os sintomas observados e o histórico de falhas
- Aprovar o orçamento de peças antes da substituição
- Realizar backup dos dados críticos antes do serviço (ou contratar o serviço de backup PaperMoon)

## Recursos e Funcionalidades

### Manutenção preventiva

- Limpeza interna com soprador industrial e antiestático
- Verificação e reposição de pasta térmica
- Teste de memória (Memtest) e saúde de discos (S.M.A.R.T.)
- Atualização de BIOS/UEFI e firmware de componentes
- Verificação de temperatura e funcionamento dos coolers

### Manutenção corretiva

- Diagnóstico de falhas de hardware e software
- Troca de HDs, SSDs, memórias, fontes e ventoinhas
- Substituição de baterias de CMOS e supercapacitores de RAID
- Reinstalação de sistema operacional (Windows Server, Linux)
- Migração de dados de disco defeituoso para disco novo

### Servidores suportados

- Dell PowerEdge (R-series, T-series)
- HP ProLiant (DL-series, ML-series) com iLO
- Supermicro e servidores whitebox
- Workstations e computadores corporativos
- NAS de marcas diversas (Synology, QNAP, TrueNAS)

## Perguntas Frequentes (FAQ)

### O atendimento é presencial ou posso enviar o equipamento?

Ambos. Para servidores em rack instalados no local, realizamos o atendimento presencial. Para computadores, workstations e servidores tower, o cliente pode enviar o equipamento para nossa bancada ou agendamos visita técnica.

### Vocês garantem a recuperação de dados em disco defeituoso?

Realizamos clonagem de disco com setores defeituosos usando ferramentas especializadas (ddrescue, Clonezilla). A taxa de sucesso depende do nível de dano físico. Para casos críticos, recomendamos laboratório especializado em recuperação de dados.

### Com que frequência devo fazer manutenção preventiva?

Recomendamos manutenção preventiva anual para servidores em produção — incluindo limpeza, verificação de discos, atualização de firmware e relatório de saúde. Servidores em ambientes com muita poeira podem exigir semestral.

### Como funciona a cobrança?

Cobrada por visita técnica + hora de trabalho, ou por escopo fechado para projetos maiores. Peças substituídas são cotadas separadamente com aprovação prévia do cliente.

### A manutenção pode ser contratada de forma periódica ou mensal?

Sim. Para empresas que querem garantir a saúde dos servidores de forma contínua, oferecemos contratos de manutenção preventiva mensal ou trimestral com visita agendada. O contrato inclui relatório técnico após cada visita, registro das intervenções no GLPI e priority response em caso de chamado corretivo urgente.',
    'Redes e Física',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Acesso SSH ou presencial aos servidores a serem mantidos', 'Inventário ou lista dos servidores, SOs e serviços em execução', 'Janela de manutenção combinada com antecedência para serviços críticos', 'Credenciais de acesso (root/sudo) ou autorização para criação de usuário técnico']::text[],
    ARRAY['01. Diagnóstico: Analisamos logs de sistema, saúde dos discos (S.M.A.R.T.), temperatura, memória e componentes. Identificamos a causa raiz da falha ou degradação e apresentamos o relatório antes de qualquer intervenção.', '02. Manutenção e reparos: Com o diagnóstico aprovado, executamos a manutenção: limpeza interna, troca de componentes defeituosos, atualização de firmware e reinstalação/migração do sistema operacional conforme necessário.', '03. Testes e entrega: Após a manutenção, realizamos testes de estabilidade (burn-in, stress test, verificação de memória) e entregamos o relatório completo com o estado do equipamento e recomendações preventivas.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'cpu',
    'published',
    140,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'backup',
    'Backup corporativo',
    'Política de backup e recuperação de desastres para empresas.',
    '## Sobre o Serviço

A ausência de backup é uma das causas mais frequentes de falência de pequenas e médias empresas após incidentes de TI — ransomware, falha de disco, exclusão acidental ou falha de data center. Estudos da indústria mostram que 60% das empresas que perdem dados críticos encerram atividades dentro de 6 meses. O backup bem feito não é apenas cópia de arquivos: é uma política com objetivos claros de recuperação.

Dois conceitos definem uma política de backup robusta: o RPO (Recovery Point Objective) — quanto de dado pode ser perdido, medido em tempo — e o RTO (Recovery Time Objective) — em quanto tempo o sistema precisa estar operacional após a falha. Uma empresa que não pode perder mais de 4 horas de dados e precisa estar online em 2 horas tem requisitos completamente diferentes de uma que aceita perder 1 dia e pode ficar 24h fora.

A PaperMoon conduz todo o processo: levantamento dos dados críticos, definição de RPO e RTO com o gestor, escolha da ferramenta de backup adequada ao ambiente (Veeam para ambientes VMware/Windows, BorgBackup ou Restic para Linux, Proxmox Backup Server para VMs Proxmox), implementação da regra 3-2-1 (3 cópias, 2 mídias diferentes, 1 offsite) e testes de restauração documentados. O Plano de Recuperação de Desastres (DRP) é entregue ao final.

## Diferenciais

- Definição de RPO e RTO antes de qualquer configuração — backup alinhado ao risco real do negócio
- Regra 3-2-1 implementada: cópia local + cópia offsite (nuvem ou site remoto)
- Testes de restore documentados: backup que não é testado não existe
- DRP (Plano de Recuperação de Desastres) entregue como documento operacional ao fim do projeto
- Monitoramento de jobs com alerta imediato por e-mail quando um backup falha

## O que nós entregamos

- Levanta os dados críticos e define RPO/RTO junto com o cliente
- Seleciona e configura a ferramenta de backup adequada ao ambiente
- Configura jobs de backup com retenção, rotação e criptografia
- Implementa backup 3-2-1: 3 cópias, 2 mídias, 1 offsite
- Realiza e documenta testes de restore para validação
- Entrega a documentação do plano de recuperação de desastres (DRP)

## Responsabilidades do Cliente

- Definir quais dados e sistemas são críticos para o negócio
- Fornecer servidor de destino para backup local (ou contratar cloud)
- Validar os testes de restore apresentados pela PaperMoon
- Manter o ambiente de backup ativo e monitorado após a implantação

## Recursos e Funcionalidades

### Cobertura de dados

- Arquivos e pastas compartilhadas (servidores Windows e Linux)
- Bancos de dados (PostgreSQL, MySQL, SQL Server, MongoDB)
- Máquinas virtuais (Proxmox, VMware, Hyper-V)
- E-mail (Exchange, Zimbra, Google Workspace)
- Configurações de rede e firewalls

### Ferramentas e destinos

- BorgBackup e Restic — backup incremental deduplidado
- Veeam Community Edition — backup de VMs
- Bacula — backup corporativo multi-cliente
- Destino local: NAS TrueNAS ou servidor dedicado
- Destino offsite: S3 (AWS, Cloudflare R2, Backblaze B2)

### Confiabilidade e compliance

- Criptografia AES-256 em trânsito e em repouso
- Retenção configurável: diário, semanal, mensal, anual
- Alertas de falha de job por e-mail e/ou Telegram
- Testes de restore periódicos documentados
- DRP com RTO e RPO definidos e validados

## Perguntas Frequentes (FAQ)

### Qual a diferença entre backup local e offsite?

Backup local (NAS na empresa) permite restore rápido mas não protege contra incêndio, roubo ou desastre físico. Backup offsite (nuvem ou localização remota) protege contra esses cenários mas o restore é mais lento. A regra 3-2-1 combina os dois para máxima proteção.

### Com que frequência os backups são feitos?

Depende do RPO definido. Dados críticos como banco de dados podem ser backupeados a cada hora. Arquivos podem ser diários. A política é definida em conjunto com o cliente no levantamento inicial.

### Como sei que o backup está funcionando?

Configuramos alertas por e-mail para sucesso e falha de cada job. Além disso, realizamos testes de restore periódicos (mensais ou trimestrais) e documentamos o resultado — assim você tem evidência de que os backups são restauráveis.

### Como funciona a cobrança?

A implantação é cobrada como projeto único — levantamento, configuração, testes e documentação DRP. Não há mensalidade da PaperMoon. Os custos de armazenamento offsite (nuvem) são pagos diretamente ao provedor de storage.

### O que é a regra 3-2-1 e por que ela é importante?

A regra 3-2-1 é o padrão-ouro de backup: 3 cópias dos dados, em 2 tipos de mídia diferentes, com 1 cópia offsite. Exemplo: dados no servidor principal (1), backup no NAS local (2), backup na nuvem como Backblaze B2 (3). Essa estratégia garante que nenhum evento isolado — falha de disco, incêndio, ransomware — consiga destruir todas as cópias ao mesmo tempo. A PaperMoon implementa essa regra como padrão em todos os projetos de backup.',
    'Redes e Física',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Acesso SSH aos servidores e sistemas de origem do backup', 'Destino de backup definido: servidor local, NAS, nuvem (S3, Backblaze, Wasabi) ou misto', 'Volume de dados estimado para dimensionamento do storage de destino', 'Janela de execução preferida para os jobs de backup (horário de menor uso)']::text[],
    ARRAY['01. Levantamento e política: Identificamos todos os dados críticos do cliente — arquivos, banco de dados, VMs, e-mail —, definimos RPO (quanto de dado pode ser perdido) e RTO (em quanto tempo o sistema deve voltar), e projetamos a arquitetura de backup.', '02. Implantação e automação: Configuramos os jobs de backup com criptografia, compressão e retenção. Implementamos a regra 3-2-1: cópia local no servidor NAS, cópia offsite em nuvem ou localização remota. Monitoramento e alertas por e-mail incluídos.', '03. Teste de restore e DRP: Realizamos testes de restauração de cada tipo de dado para validar a integridade dos backups. Documentamos o procedimento passo a passo no Plano de Recuperação de Desastres — para que qualquer técnico consiga executar sem depender da PaperMoon.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'archive',
    'published',
    150,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'rustdesk',
    'RustDesk',
    'Acesso remoto seguro e auto-hospedado, sem custo por dispositivo.',
    '## Sobre o Serviço

O RustDesk foi criado em 2021 por um desenvolvedor de Hong Kong que publicou o projeto como open-source no GitHub como resposta direta aos preços crescentes do TeamViewer e AnyDesk — ferramentas que chegam a cobrar R$ 200–500 por mês por técnico com um número limitado de dispositivos gerenciados. Escrito em Rust (linguagem de programação conhecida por performance e segurança de memória), o RustDesk se tornou rapidamente um dos projetos open-source de crescimento mais rápido na área de suporte remoto.

A diferença fundamental do RustDesk é a arquitetura self-hosted: toda a comunicação entre o técnico e o dispositivo controlado passa pelo servidor relay instalado na VPS do cliente — não pelos servidores da empresa do software. Isso significa que o tráfego de acesso remoto nunca toca infraestrutura de terceiros, eliminando um vetor de espionagem corporativa e atendendo requisitos de compliance de setores como financeiro, saúde e jurídico.

A PaperMoon instala o servidor relay (hbbs + hbbr) na VPS do cliente, configura o DNS para o endereço do relay, e distribui o cliente RustDesk pré-configurado para os dispositivos da empresa. O técnico de suporte acessa qualquer dispositivo autorizado com a mesma experiência do TeamViewer — sem login externo, sem dependência de conta SaaS.

## Diferenciais

- Servidor relay próprio na sua VPS: tráfego de acesso remoto nunca sai da sua infraestrutura
- Zero custo por dispositivo ou por técnico — sem mensalidade independente do tamanho da equipe
- Clientes para Windows, macOS, Linux, Android e iOS em uma única solução
- Criptografia end-to-end: comunicação protegida mesmo com relay próprio
- Alternativa direta ao TeamViewer (R$200+/mês/técnico) sem abrir mão da experiência de uso

## O que nós entregamos

- Instalação dos dois componentes de servidor (hbbs + hbbr) via Docker na sua VPS
- Configuração de SSL/TLS, firewall e portas dedicadas ao RustDesk
- Geração e distribuição de chaves criptográficas da sua organização
- Build de cliente customizado com logo e nome da empresa (opcional)
- Configuração de acesso não supervisionado para servidores e estações fixas
- Monitoramento de disponibilidade dos servidores RustDesk via Zabbix

## Responsabilidades do Cliente

- Instalar o cliente RustDesk nos dispositivos que deseja acessar (Windows, macOS, Linux)
- Configurar senha de acesso não supervisionado nas máquinas permanentes
- Gerenciar quais colaboradores têm permissão de acesso a cada dispositivo
- Instalar o app RustDesk Mobile (iOS/Android) para acesso pelo celular

## Recursos e Funcionalidades

### Conectividade

- Conexão P2P direta sempre que possível (menor latência)
- Relay automático quando P2P não é viável (firewalls, CGNAT)
- Suporte a múltiplos monitores com troca rápida
- Resolução adaptativa conforme a largura de banda disponível
- Acesso não supervisionado por senha permanente por dispositivo

### Segurança

- Criptografia ponta a ponta — Curve25519 + AES-256-GCM
- Chave privada gerada e armazenada só na sua VPS
- Sem dados de conexão em servidores de terceiros
- Autenticação por senha por sessão ou permanente por dispositivo
- Compatível com políticas LGPD e ISO 27001

### Compatibilidade

- Windows 7+ (32 e 64 bits)
- macOS 10.14+ (Intel e Apple Silicon)
- Linux (Ubuntu, Debian, CentOS, Fedora, openSUSE)
- iOS — visualização de tela
- Android — controle completo

### Recursos avançados

- Transferência de arquivos bidirecional via drag-and-drop
- Chat de texto integrado à sessão
- Área de transferência (clipboard) sincronizada entre máquinas
- Gravação de sessão local para auditoria
- API REST para integração com GLPI e sistemas de ITSM

## Perguntas Frequentes (FAQ)

### Qual a diferença para TeamViewer ou AnyDesk?

TeamViewer e AnyDesk são serviços proprietários cobrados por usuário ou dispositivo, com dados de conexão passando pelos servidores deles. O RustDesk self-hosted é open-source, sem custo por dispositivo e com toda a infraestrutura na sua VPS — nenhum dado de sessão sai da sua rede.

### É possível personalizar o cliente com a marca da empresa?

Sim. O RustDesk permite compilar um cliente customizado com nome, logo e servidor pré-configurado. A PaperMoon realiza o build e distribui os instaladores para Windows, macOS e Linux com a identidade visual da sua empresa.

### Quantos dispositivos e usuários posso ter?

Não há limite de dispositivos ou usuários no RustDesk self-hosted. Uma VPS com 2 vCPU e 2 GB RAM comporta centenas de sessões simultâneas. O único limite é a capacidade do servidor relay que você escolher.

### Como funciona o acesso não supervisionado?

Cada máquina pode ter uma senha permanente configurada. Com ela, o técnico acessa o equipamento mesmo sem ninguém presente — útil para servidores e máquinas de produção. A senha pode ser rotacionada a qualquer momento.

### Como funciona a cobrança?

A implantação é cobrada como projeto único — instalação, configuração, SSL, chaves criptográficas, treinamento e documentação. Não há mensalidade da PaperMoon. Você paga apenas a VPS onde o servidor roda.',
    'Acesso Remoto',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 1 vCPU, 1 GB RAM, 20 GB SSD) para o servidor relay', 'Acesso SSH à VPS com usuário root ou sudo', 'Porta 21115–21119 liberadas no firewall da VPS', 'Computadores clientes com Windows, Linux ou macOS para instalar o RustDesk Client']::text[],
    ARRAY['01. Servidor RustDesk na sua VPS: Implantamos os dois componentes do servidor RustDesk — hbbs (registro de IDs) e hbbr (relay de tráfego) — via Docker na sua VPS, com SSL e firewall configurados.', '02. Clientes configurados e distribuídos: Fornecemos o cliente pré-configurado com o endereço do seu servidor e a chave criptográfica. Opcionalmente, compilamos um cliente com a identidade visual da empresa para distribuição interna.', '03. Treinamento do time de TI: Treinamos o time de suporte para usar o RustDesk: acesso supervisionado e não supervisionado, transferência de arquivos, chat integrado e boas práticas de segurança.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'monitor',
    'published',
    160,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'windows-server',
    'Windows Server',
    'Active Directory, servidor de arquivos e políticas de grupo para ambientes Microsoft corporativos.',
    '## Sobre o Serviço

O Windows Server é a plataforma de servidor da Microsoft com mais de 40 anos de história corporativa. Para ambientes que dependem de Active Directory, Group Policies, servidores de arquivos com permissões NTFS, SQL Server, Exchange ou aplicações LOB (Line of Business) legadas, o Windows Server é a única escolha tecnicamente viável — não há equivalente open-source para o conjunto completo de serviços de diretório do Active Directory com a mesma maturidade e interoperabilidade.

Um ambiente Windows Server bem estruturado começa pelo Active Directory: a hierarquia de Unidades Organizacionais (UOs) define como os usuários, computadores e grupos são organizados, e as GPOs (Group Policies) controlam centenas de comportamentos — desde política de senha e bloqueio de tela até instalação de software e acesso a unidades de rede. Sem uma estrutura de OU bem planejada, a gestão do AD se torna impossível de manter conforme a empresa cresce.

A PaperMoon projeta a estrutura do Active Directory conforme o organograma da empresa, cria as UOs e GPOs alinhadas à política de segurança do cliente, configura o servidor de arquivos com compartilhamentos e permissões NTFS granulares, e integra com o Microsoft 365 via Entra ID Connect para que os usuários usem a mesma credencial nos sistemas locais e na nuvem Microsoft.

## Diferenciais

- Estrutura de OU e GPO projetada conforme o organograma real — não uma estrutura genérica
- Integração com Microsoft 365 via Entra ID Connect: credencial única para local e nuvem
- Servidor de arquivos com permissões NTFS granulares por pasta e grupo de segurança
- WSUS configurado para controle de updates corporativos — sem atualizações surpresa
- Hyper-V aproveitado como hypervisor nativo sem custo adicional de licença

## O que nós entregamos

- Instalação e configuração do Windows Server 2019/2022 (Standard ou Datacenter)
- Promoção do Domain Controller e configuração do Active Directory Domain Services (AD DS)
- Estrutura de Unidades Organizacionais (UOs), grupos de segurança e delegação de administração
- Políticas de Grupo (GPOs): senhas, mapeamento de drives, restrições de software e segurança
- Servidor de arquivos com compartilhamentos NTFS, permissões por grupo, quotas e Shadow Copies
- Integração com Microsoft 365 / Entra ID via Microsoft Entra Connect (AD Sync)
- Configuração de DNS interno, DHCP com reservas e WSUS para patches centralizados
- Backup do AD (System State) e dos dados com agendamento automático
- Monitoramento de disponibilidade do servidor e dos serviços via Zabbix

## Responsabilidades do Cliente

- Fornecer a licença do Windows Server (SPLA via CSP ou licença perpétua)
- Definir a política de grupos de segurança e permissões de acesso por departamento
- Aprovar alterações de GPO antes da aplicação em produção
- Gerenciar a criação e desativação de contas de usuário (ou delegar à PaperMoon por contrato)

## Recursos e Funcionalidades

### Active Directory

- Domínio Windows com estrutura de UOs por departamento
- Grupos de segurança para controle de acesso a recursos
- Delegação de administração por área sem dar acesso de Domain Admin
- Integração com Microsoft 365 / Entra ID (Single Sign-On unificado)
- Fine-Grained Password Policies — políticas de senha por grupo

### Servidor de Arquivos

- Compartilhamentos NTFS com permissões por grupo de segurança do AD
- Quotas de disco por usuário ou pasta compartilhada
- Shadow Copies automáticas — recuperação de versões anteriores em segundos
- DFS (Distributed File System) para namespace unificado entre servidores
- Auditoria de acesso a arquivos — quem abriu, modificou ou excluiu

### Gerenciamento Centralizado

- GPOs para padronizar todas as estações e servidores do domínio
- WSUS — patches e atualizações Windows aprovadas e distribuídas centralmente
- DHCP com reservas por MAC e escopos segmentados por VLAN
- DNS interno com zones de pesquisa direta e reversa

### Segurança

- Políticas de senha: complexidade, validade e histórico por GPO
- BitLocker via GPO para criptografia de disco em laptops
- Account Lockout automático após tentativas falhas
- MFA com Microsoft Entra ID para acesso a aplicações cloud
- Monitoramento de eventos de segurança via Zabbix

## Perguntas Frequentes (FAQ)

### Qual versão do Windows Server é implantada?

Trabalhamos com Windows Server 2019 e 2022 (Standard ou Datacenter, conforme a necessidade de VMs). A versão recomendada para novos projetos é o Windows Server 2022, com suporte estendido até 2031.

### Preciso comprar a licença? Como funciona?

Sim, a licença do Windows Server é responsabilidade do cliente. As opções mais comuns são: licença perpétua (OEM/Retail), SPLA via provedores CSP (pagamento mensal por servidor) ou Microsoft 365 Business Premium para ambientes híbridos. A PaperMoon orienta sobre a melhor opção para o seu caso.

### É possível integrar com o Microsoft 365 (Office 365)?

Sim. A PaperMoon configura o Microsoft Entra Connect (antigo Azure AD Connect) para sincronizar usuários e senhas entre o Active Directory local e o Entra ID. Com isso, o colaborador usa a mesma credencial para logar na estação e acessar Teams, Outlook e SharePoint.

### O que acontece se o servidor cair? O AD deixa de funcionar?

Recomendamos dois Domain Controllers para alta disponibilidade. Com dois DCs, a autenticação no domínio continua funcionando mesmo com um servidor fora do ar. Também configuramos backup automatizado do System State para restauração rápida.

### Como funciona a cobrança?

O projeto de implantação (instalação, AD, GPOs, servidor de arquivos, treinamento e documentação) é cobrado como projeto único. Suporte contínuo e gerenciamento mensal são contratados separadamente conforme a necessidade.',
    'Acesso Remoto',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Licença do Windows Server (2019, 2022 ou superior) e CALs necessárias', 'Hardware ou VM dedicada compatível (mínimo 4 vCPU, 8 GB RAM, 80 GB SSD para o SO)', 'Acesso ao console da VM ou KVM-over-IP para instalação', 'Definição do papel do servidor: AD DS, File Server, IIS, RDS ou combinação']::text[],
    ARRAY['01. Active Directory e domínio: Instalamos o Windows Server, promovemos o Domain Controller, definimos a estrutura de UOs e importamos ou criamos todos os usuários e grupos da organização. DNS e DHCP são configurados para o domínio.', '02. Servidor de arquivos e permissões: Criamos os shares de rede com permissões NTFS granulares por grupo de segurança, quotas por usuário, Shadow Copies para recuperação de versões anteriores e mapeamento automático via GPO nas estações do domínio.', '03. GPOs, segurança e integração M365: Aplicamos políticas de grupo corporativas: bloqueio de mídia removível, políticas de senha, padronização de estações e restrições de software. Integramos ao Microsoft 365 via Entra Connect para SSO unificado.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'users',
    'published',
    170,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'samba',
    'Samba — Servidor de Arquivos',
    'Compartilhamento de arquivos em Linux, compatível nativamente com Windows, macOS e Linux.',
    '## Sobre o Serviço

O Samba existe desde 1992, criado pelo australiano Andrew Tridgell como implementação reverse-engineered do protocolo SMB da Microsoft para Linux. Hoje é mantido como projeto open-source pela Samba Team e é a solução mais amplamente adotada para ambientes de rede heterogêneos onde máquinas Windows, macOS e Linux precisam compartilhar arquivos e impressoras pelo mesmo protocolo.

A proposta do Samba é clara: transformar qualquer servidor Linux em um file server compatível nativo com o Windows Explorer — sem instalar nenhum software adicional nas estações. Máquinas Windows mapeiam compartilhamentos Samba exatamente como fazem com Windows Server, com a mesma experiência de usuário, mas com o servidor rodando em Ubuntu Server em hardware que custa a metade do equivalente Windows.

A PaperMoon instala o Samba no servidor Linux do cliente, configura os compartilhamentos por departamento com controle de acesso baseado em grupos, habilita a auditoria de acessos (quem abriu, modificou ou deletou qual arquivo e quando), define quotas de disco por usuário e integra com o Active Directory existente via Winbind para que as credenciais de rede do Windows funcionem transparentemente no Samba.

## Diferenciais

- Sem custo de licença de Windows Server — o servidor de arquivos roda em Ubuntu Linux
- Integração com Active Directory via Winbind: mesma credencial do Windows para o file server
- Auditoria de acessos por arquivo: registra quem abriu, modificou e deletou cada documento
- Quotas de disco configuráveis por usuário ou grupo — controle de uso de armazenamento
- Performance superior ao Windows Server em hardware equivalente para workloads de arquivos

## O que nós entregamos

- Instalação do Samba em Linux (Ubuntu Server / Debian / RHEL) na sua VPS ou servidor físico
- Configuração de compartilhamentos (shares) por departamento, projeto ou perfil de acesso
- Controle de permissões POSIX + ACLs estendidas por usuário e grupo
- Autenticação local com usuários Samba ou integrada ao Active Directory Windows (domain member)
- Quotas de disco por usuário ou share com alertas automáticos de capacidade
- Auditoria de acessos: log de quem abriu, modificou e excluiu arquivos
- Criptografia de tráfego SMB3 para segurança em redes sem fio e WAN
- Backup dos dados via Rsync ou integração com TrueNAS/Proxmox Backup Server
- Monitoramento de disponibilidade e capacidade de disco via Zabbix

## Responsabilidades do Cliente

- Mapear os drives de rede nas estações Windows (ou via GPO se tiver Active Directory)
- Definir estrutura de pastas e política de acesso por departamento
- Gerenciar usuários Samba (ou integrar ao AD existente para credenciais unificadas)

## Recursos e Funcionalidades

### Compartilhamento

- Protocolo SMB/CIFS — acesso nativo do Windows sem instalar cliente adicional
- Compatível com macOS Finder (AFP via SMB) e Linux via CIFS-utils
- Múltiplos shares com configuração independente de permissões e quotas
- DFS (Distributed File System) para namespace unificado entre múltiplos servidores
- Suporte a links simbólicos e atributos estendidos de arquivo

### Autenticação

- Usuários locais Samba com senha independente
- Integração como domain member de Active Directory Windows
- Suporte a LDAP para autenticação centralizada
- Winbind para mapeamento de usuários AD para UIDs Linux
- Suporte a grupos aninhados do Active Directory nas permissões de share

### Segurança e Auditoria

- Permissões POSIX + ACLs estendidas por usuário e grupo
- Criptografia de tráfego SMB3 (obrigatória ou opcional por share)
- Log de auditoria: abertura, escrita, exclusão e renomeação de arquivos
- Bloqueio de acesso por IP ou sub-rede por share
- Integração com fail2ban para proteção contra força bruta

### Performance e Disponibilidade

- Rsync para replicação assíncrona entre servidores
- Backup incremental agendado com retenção configurável
- Cache de metadados para melhor performance em redes de alta latência
- Monitoramento de IOPS, espaço e disponibilidade via Zabbix
- Notificações automáticas quando o disco ultrapassa 80% de uso

## Perguntas Frequentes (FAQ)

### Qual a diferença entre Samba e Windows Server para arquivos?

Funcionalmente equivalentes para a maioria das empresas: ambos falam SMB/CIFS e integram ao Active Directory. A diferença principal é custo e plataforma — o Samba roda em Linux (sem licença de SO) e é open-source, enquanto o Windows File Server exige licença do Windows Server. Para empresas sem AD existente e sem necessidade de GPOs, o Samba é a escolha mais econômica.

### Máquinas Windows acessam o Samba sem instalar nada?

Sim. O Samba fala SMB/CIFS, o mesmo protocolo nativo do Windows. O usuário simplesmente mapeia o drive de rede (\\servidor\share) ou acessa via explorador de arquivos — sem nenhum software adicional na estação.

### É possível integrar ao Active Directory Windows existente?

Sim. O Samba pode operar como domain member do AD Windows. Com isso, os usuários usam as mesmas credenciais de domínio para acessar os shares Linux — sem criar contas duplicadas. A sincronização de grupos de segurança do AD nas permissões do Samba também é suportada.

### Qual sistema operacional Linux é usado?

Recomendamos Ubuntu Server LTS (22.04 ou 24.04) para novos projetos, por ter suporte de longo prazo e boa integração com o Samba. Também suportamos Debian, Rocky Linux e RHEL conforme preferência do cliente.

### Como funciona a cobrança?

O projeto de implantação (instalação, configuração de shares, permissões, autenticação, quotas, backup e treinamento) é cobrado como projeto único. Não há custo de licença de SO. O cliente paga apenas o servidor (VPS ou físico) onde o Samba roda.',
    'Acesso Remoto',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Servidor Linux com Ubuntu 22.04 LTS (mínimo 1 vCPU, 2 GB RAM, storage conforme volume de dados)', 'Acesso SSH ao servidor com usuário root ou sudo', 'Lista de compartilhamentos (shares), usuários e permissões desejados', 'Integração com Active Directory existente se aplicável (opcional)']::text[],
    ARRAY['01. Instalação e estrutura de shares: Instalamos o Samba no servidor Linux, criamos os compartilhamentos por departamento com permissões POSIX + ACL e integramos ao DNS da rede para que as máquinas localizem os shares pelo nome.', '02. Autenticação e permissões: Configuramos autenticação local com usuários Samba ou integramos ao Active Directory Windows existente (Samba como domain member) — as credenciais de domínio passam a funcionar diretamente nos shares Linux.', '03. Quotas, auditoria e backup: Ativamos quotas de disco por usuário, log de auditoria de acessos e rotina de backup incremental dos dados compartilhados. Monitoramento de espaço e disponibilidade via Zabbix com alertas configuráveis.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'folder-open',
    'published',
    180,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'plone',
    'Plone CMS',
    'Portal corporativo e gestão de conteúdo enterprise com total controle dos seus dados.',
    '## Sobre o Serviço

O Plone é um dos sistemas de gerenciamento de conteúdo mais antigos e seguros do mercado, desenvolvido desde 2001 sobre o framework Zope/Python. Seu histórico de segurança é notável: não há vulnerabilidades CVE críticas registradas nos últimos anos, o que levou à sua adoção por organizações governamentais, militares e acadêmicas nos Estados Unidos e na Europa como plataforma de intranet e gestão documental.

O que diferencia o Plone dos CMSs mais populares como WordPress ou Drupal é sua arquitetura voltada para controle rigoroso de conteúdo: fluxo de publicação configurável (rascunho → revisão → aprovação → publicação), versionamento automático de todas as edições com comparação de versões, controle de acesso por conteúdo individual (não apenas por pasta) e suporte nativo a fluxos de trabalho editoriais complexos — ideal para intranets corporativas onde diferentes equipes têm permissões diferentes sobre diferentes seções.

A PaperMoon instala o Plone com a versão mais recente da linha 6.x, configura o tema institucional da empresa, define os tipos de conteúdo necessários (documentos, notícias, procedimentos, FAQs), configura os fluxos de publicação e treina as equipes de conteúdo e administração.

## Diferenciais

- Histórico de segurança excepcional: adotado por governos e órgãos militares europeus e americanos
- Fluxo de publicação configurável: nenhum conteúdo publicado sem aprovação do responsável
- Versionamento automático de todo conteúdo com comparação entre versões
- Controle de acesso por conteúdo individual — não apenas por pasta ou categoria
- Integração com LDAP/Active Directory para login corporativo unificado

## O que nós entregamos

- Instalação do Plone 6 em servidor Linux (Ubuntu/Debian) via Docker ou buildout
- Configuração de portal corporativo com temas responsivos e identidade visual da empresa
- Estrutura de tipos de conteúdo: documentos, notícias, eventos, base de conhecimento
- Controle de acesso por papel (roles): leitor, redator, editor, gerente e administrador
- Workflow de publicação configurável: rascunho → revisão → publicado
- Integração com LDAP / Active Directory para login único
- Configuração de SSL, proxy reverso Nginx e backups automáticos
- Treinamento da equipe editorial e documentação de operação

## Responsabilidades do Cliente

- Definir a estrutura de seções e tipos de conteúdo desejados
- Fornecer identidade visual (cores, logo, tipografia) para o tema
- Definir papéis de usuário e fluxo de aprovação de conteúdo
- Gerenciar criação de conteúdo e usuários após a entrega

## Recursos e Funcionalidades

### Gestão de conteúdo

- Versionamento completo com histórico e rollback
- Workflow de publicação customizável por tipo de conteúdo
- Agendamento de publicação e expiração automática
- Editor rich-text (TinyMCE) e suporte a Markdown
- Gerenciamento de mídia: imagens, vídeos e documentos

### Controle de acesso

- Papéis granulares por seção do portal
- Integração com LDAP / Active Directory
- SSO via plugins (CAS, OAuth2, SAML)
- Auditoria de acessos e alterações de conteúdo
- Grupos de usuário com permissões herdadas por hierarquia

### Infraestrutura e segurança

- 100% self-hosted — dados nunca saem do seu servidor
- Backup automático do ZODB e blobs de mídia
- Proxy Nginx com HTTPS/TLS (Let''s Encrypt ou certificado corporativo)
- Cache de páginas com Varnish para alta performance
- Suporte a Docker Compose para deploys reproduzíveis

## Perguntas Frequentes (FAQ)

### Plone é adequado para que tipo de empresa?

Plone é ideal para empresas e instituições que precisam de portal corporativo, intranet ou gestão documental com controle de acesso rigoroso — especialmente em setores regulados como saúde, educação, governo e jurídico. É a escolha quando privacidade e controle sobre os dados são prioritários.

### Qual a diferença entre Plone e WordPress?

Plone foi projetado para gestão documental e portais corporativos com controle de acesso granular, workflow de publicação e versionamento nativo. WordPress foca em blogs e sites de conteúdo simples. Para intranets com múltiplos papéis de usuário e fluxos de aprovação, Plone é significativamente mais maduro.

### É possível integrar o Plone ao Active Directory da empresa?

Sim. O Plone possui plugins LDAP e SAML para autenticação federada. Com isso, os colaboradores usam as credenciais do domínio Windows para acessar o portal — sem contas duplicadas e com desativação imediata ao sair da empresa.

### Como funciona a cobrança?

O projeto de implantação (instalação, configuração, tema, integrações LDAP, treinamento e documentação) é cobrado como projeto único. Não há licença — Plone é open-source. O cliente paga apenas o servidor onde o Plone roda.

### O Plone suporta múltiplos idiomas no mesmo portal?

Sim. O Plone tem suporte nativo a multilinguismo (via pacote plone.app.multilingual) — cada conteúdo pode ter versões em diferentes idiomas com navegação por idioma do visitante. É especialmente útil para empresas com operações em múltiplos países ou portais com audiência internacional.',
    'Produtividade e CRM',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Acesso SSH ao servidor com usuário root ou sudo', 'Domínio configurado com DNS apontando para o servidor', 'Definição prévia da estrutura de seções e papéis de usuário', 'Credenciais LDAP/AD se integração de autenticação for desejada (opcional)']::text[],
    ARRAY['01. Planejamento da arquitetura de conteúdo: Mapeamos os tipos de conteúdo, hierarquia de seções, papéis de usuário e fluxo de aprovação. Essa etapa define toda a estrutura do portal antes do primeiro deploy.', '02. Instalação e configuração do Plone: Instalamos o Plone 6 na sua VPS, configuramos o banco de dados ZODB, proxy Nginx com SSL e integrações com LDAP/AD. O ambiente é preparado conforme a arquitetura de conteúdo e as integrações definidas para o projeto.', '03. Tema e identidade visual: Aplicamos um tema responsivo com as cores e logo da empresa. Configuramos layouts de listagem, páginas internas e componentes reutilizáveis.', '04. Entrega, treinamento e documentação: Realizamos treinamento presencial ou remoto com a equipe editorial e entregamos documentação operacional cobrindo criação de conteúdo, gestão de usuários e backups.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'globe',
    'published',
    190,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'keycloak',
    'Keycloak — IAM e SSO',
    'Login único para todos os sistemas da empresa com segurança enterprise.',
    '## Sobre o Serviço

O Keycloak é desenvolvido pela Red Hat desde 2013 e doado à CNCF (Cloud Native Computing Foundation) em 2023, tornando-se o padrão open-source da indústria para gestão de identidade e acesso (IAM). Implementa os protocolos OAuth 2.0, OpenID Connect e SAML 2.0 — os mesmos usados por Google, Microsoft, GitHub e Okta — e é adotado por gigantes como Bosch, Deutsche Telekom e pelo governo da Alemanha para centralizar a autenticação de centenas de sistemas.

O problema que o Keycloak resolve é fundamental: à medida que uma empresa cresce e adota mais softwares (ERP, CRM, helpdesk, colaboração, monitoramento), cada sistema exige um login separado — diferentes senhas, diferentes políticas, diferentes sessões. O SSO (Single Sign-On) soluciona isso: o usuário autentica uma vez no Keycloak e acessa todos os sistemas integrados sem fazer login novamente. Além do SSO, o Keycloak centraliza o MFA (autenticação multifator), controle de sessão, bloqueio de contas e federação com Active Directory e LDAP.

A PaperMoon instala o Keycloak na VPS do cliente, configura os realms por domínio da empresa, integra com o Active Directory existente via Winbind LDAP, e conecta os sistemas da empresa ao Keycloak usando OIDC ou SAML. A alternativa SaaS mais próxima é o Auth0 ou Okta, que cobram a partir de R$ 200/mês para um número limitado de usuários ativos.

## Diferenciais

- SSO para todos os sistemas da empresa: um login, todos os acessos — sem múltiplas senhas
- Alternativa self-hosted ao Auth0 e Okta — sem mensalidade baseada em usuários ativos
- Federação com Active Directory/LDAP: as credenciais corporativas funcionam em todos os sistemas
- MFA nativo: TOTP (Google Authenticator), WebAuthn (chave física) e e-mail OTP
- Padrões abertos OAuth2, OIDC e SAML: conecta qualquer aplicação moderna sem lock-in

## O que nós entregamos

- Instalação do Keycloak em servidor Linux via Docker Compose ou Helm (Kubernetes)
- Configuração de Realm, clientes OAuth2/OIDC e providers SAML 2.0
- Integração com LDAP / Active Directory para sincronização de usuários e grupos
- Configuração de MFA (TOTP via Google Authenticator / Authy) por papel de usuário
- Fluxos de autenticação customizados: login por e-mail, captcha, termos de uso
- Integração de aplicações existentes (N8N, Nextcloud, Chatwoot, GLPI, Zabbix)
- Configuração de SSL, proxy reverso Nginx e alta disponibilidade
- Treinamento da equipe de TI e documentação dos fluxos configurados

## Responsabilidades do Cliente

- Mapear quais aplicações devem ser integradas ao SSO
- Definir políticas de senha e regras de MFA por grupo de usuário
- Fornecer credenciais LDAP/AD se integração de diretório for necessária
- Gerenciar criação e desativação de usuários e grupos no Keycloak

## Recursos e Funcionalidades

### Autenticação e SSO

- Single Sign-On (SSO) para todos os sistemas integrados
- Single Logout — sair de uma app desconecta de todas
- Suporte a OAuth2, OpenID Connect (OIDC) e SAML 2.0
- Social login: Google, Microsoft, GitHub (configurável)
- Impersonation — admin assume sessão de usuário para suporte

### Segurança e MFA

- MFA com TOTP (Google Authenticator, Authy, FreeOTP)
- Política de senhas: complexidade, expiração e histórico
- Bloqueio de conta após tentativas incorretas
- Detecção de força bruta com backoff exponencial
- Auditoria de login: quem acessou, quando e de onde

### Gestão de identidades

- Sincronização bidirecional com LDAP / Active Directory
- User Federation: usuários gerenciados no AD, autenticados no Keycloak
- Grupos e papéis mapeados por regras automáticas do LDAP
- Self-service: usuário altera senha e perfil no portal próprio
- Convite por e-mail com fluxo de primeiro acesso controlado

## Perguntas Frequentes (FAQ)

### Quais aplicações podem ser integradas ao Keycloak?

Qualquer aplicação que suporte OAuth2, OIDC ou SAML pode ser integrada: Nextcloud, GLPI, Zabbix, N8N, Chatwoot, Grafana, Jenkins, Gitea, WordPress, aplicações Django/Node.js/Java e muitas outras. A PaperMoon cuida das configurações de todos os sistemas implantados por nós.

### Keycloak é difícil de manter?

O console web do Keycloak é intuitivo para tarefas do dia a dia — adicionar usuário, resetar senha, bloquear conta. Operações avançadas (novos clientes OIDC, fluxos de autenticação) requerem treinamento, que fazemos parte da entrega. Atualizações são simples via Docker Compose.

### É possível manter o Active Directory e usar o Keycloak ao mesmo tempo?

Sim — é o cenário mais comum. O Keycloak sincroniza usuários do AD via User Federation LDAP. O AD continua sendo a fonte de identidade para estações Windows e o Keycloak centraliza o SSO das aplicações web e cloud.

### Como funciona a cobrança?

Instalação, configuração do Realm, MFA, LDAP e integração das aplicações contratadas são cobrados como projeto único. Keycloak é open-source — sem licença. O cliente paga apenas o servidor onde roda.

### Keycloak suporta login social (Google, Microsoft, GitHub)?

Sim. O Keycloak tem Identity Providers pré-configurados para Google, Microsoft, GitHub, Facebook, Apple e qualquer provedor que suporte OIDC ou SAML. A configuração permite que usuários façam login com conta Google ou Microsoft existente, eliminando a necessidade de criar e gerenciar senha no Keycloak.',
    'Segurança e Identidade',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 30 GB SSD)', 'Acesso SSH ao servidor com usuário root ou sudo', 'Domínio configurado com DNS apontando para o servidor', 'Lista das aplicações a integrar com SSO e seus endpoints de callback', 'Credenciais LDAP/AD se integração de diretório for necessária (opcional)']::text[],
    ARRAY['01. Mapeamento de aplicações e identidades: Levantamos todas as aplicações que precisam de SSO, os grupos de usuário, papéis e políticas de acesso. Esse mapeamento define a estrutura de Realms e clientes no Keycloak.', '02. Instalação e configuração do Keycloak: Instalamos o Keycloak com banco de dados PostgreSQL, proxy Nginx e SSL. Configuramos o Realm principal, fluxos de autenticação, MFA e integração LDAP/AD.', '03. Integração das aplicações: Configuramos cada aplicação como cliente OAuth2/OIDC ou SAML. O login passa a ser centralizado — o usuário autentica uma vez e acessa todos os sistemas sem logar novamente.', '04. Treinamento e entrega: Treinamos a equipe de TI na gestão de usuários, grupos e clientes no console do Keycloak. Entregamos documentação de operação e guia de integração de novas aplicações.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'key-round',
    'published',
    200,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'tailscale',
    'Tailscale',
    'Rede privada mesh com acesso remoto seguro, simples e sem abrir portas no firewall.',
    '## Sobre o Serviço

O Tailscale foi lançado em 2019 por ex-engenheiros do Google e rapidamente se tornou uma das formas mais práticas de construir redes privadas modernas sem a complexidade de VPNs tradicionais baseadas em IPsec ou OpenVPN. Sua base técnica é o WireGuard, protocolo reconhecido por simplicidade, alta performance e criptografia moderna, mas com uma camada adicional de controle central, descoberta automática de peers, ACLs e autenticação via identidade.

Na prática, o Tailscale resolve um problema muito comum nas empresas: conectar pessoas, notebooks, servidores, filiais e ambientes cloud de forma segura sem depender de redirecionamento de portas, IP fixo, regras complexas de NAT ou túneis frágeis. Em vez de expor serviços na internet, cada dispositivo autorizado entra em uma tailnet privada e só conversa com os recursos permitidos pelas políticas de acesso.

A PaperMoon implanta o Tailscale na operação do cliente com desenho de rede, política de grupos, autenticação com Google ou Microsoft, roteadores de sub-rede para acesso a redes locais e documentação operacional. O resultado é acesso remoto rápido e seguro para equipe de TI, usuários e fornecedores, sem sacrificar visibilidade nem governança.

## Diferenciais

- Acesso remoto seguro sem expor portas nem depender de IP fixo
- Baseado em WireGuard: criptografia moderna, baixa latência e excelente desempenho
- ACLs por usuário, grupo, dispositivo e serviço com governança centralizada
- Integração com Google Workspace, Microsoft Entra ID e outros provedores SSO
- Subnets, exit nodes e compartilhamento controlado para filiais, clouds e parceiros

## O que nós entregamos

- Desenha a topologia da tailnet conforme filiais, times e ambientes
- Instala o agente Tailscale em servidores, notebooks e jump hosts necessários
- Configura autenticação com Google, Microsoft ou outro provedor suportado
- Define ACLs, tags e grupos de acesso por perfil de usuário e ambiente
- Configura subnet routers e exit nodes quando houver redes locais a publicar
- Documenta o acesso remoto e treina a equipe para operação diária

## Responsabilidades do Cliente

- Definir quais usuários, dispositivos e redes devem entrar na tailnet
- Aprovar políticas de acesso entre times, fornecedores e ambientes
- Fornecer acesso administrativo aos dispositivos ou servidores a integrar
- Manter a gestão de usuários no provedor de identidade escolhido

## Recursos e Funcionalidades

### Acesso remoto moderno

- Rede privada mesh entre notebooks, servidores e clouds
- Conexão ponto a ponto sempre que possível para menor latência
- Sem abrir portas de RDP, SSH ou painéis na internet
- Cliente leve para Windows, macOS, Linux, iOS e Android

### Governança e segurança

- ACLs por usuário, grupo, tag e porta
- Integração com SSO corporativo e revogação central de acesso
- Logs de acesso e visibilidade dos dispositivos conectados
- Aprovação de dispositivos e políticas de menor privilégio

### Redes e operação

- Subnet routers para acessar redes locais sem instalar agente em tudo
- Exit nodes para navegação segura por rede controlada
- Acesso entre matriz, filial, cloud e home office
- Compartilhamento controlado para terceiros e suporte externo

## Perguntas Frequentes (FAQ)

### Qual a diferença entre Tailscale e uma VPN tradicional?

VPNs tradicionais costumam exigir concentrador, regras de firewall, NAT, IP fixo e configuração manual de túneis. O Tailscale simplifica isso com WireGuard, descoberta automática de peers, autenticação por identidade e políticas centralizadas. O resultado é implantação muito mais rápida e manutenção mais simples.

### Preciso abrir portas no roteador ou firewall?

Na maioria dos cenários, não. O Tailscale usa NAT traversal para estabelecer conexão direta entre os nós. Quando isso não é possível, ele usa relays DERP sem expor seus serviços diretamente na internet.

### Consigo acessar uma rede inteira da empresa sem instalar em todos os equipamentos?

Sim. Com subnet routers, a PaperMoon publica redes locais inteiras dentro da tailnet. Assim, um notebook autorizado consegue acessar servidores e dispositivos internos mesmo que eles não tenham o agente instalado individualmente.

### É uma alternativa ao TeamViewer ou AnyDesk?

Em muitos cenários, sim. O Tailscale não substitui a ferramenta de controle remoto em si, mas cria a conectividade privada e segura entre os dispositivos. Ele combina muito bem com RDP, SSH, RustDesk e painéis web internos sem precisar expor nada publicamente.

### Como funciona a cobrança?

A implantação, desenho da rede, políticas de ACL, subnet routers e treinamento são cobrados como projeto. Se houver gestão contínua do ambiente, isso pode entrar em contrato mensal de operação. O custo recorrente principal depende do plano escolhido diretamente com o Tailscale, quando aplicável.',
    'Segurança e Identidade',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Lista dos usuários, notebooks, servidores e redes que participarão da tailnet', 'Acesso administrativo aos dispositivos ou servidores a integrar', 'Provedor de identidade definido (Google, Microsoft ou outro suportado)', 'Mapeamento de quais redes locais precisam ser publicadas via subnet router']::text[],
    ARRAY['01. Mapeamento de acessos: Levantamos quem precisa acessar o quê: servidores, redes locais, notebooks administrativos, filiais e parceiros. Isso define grupos, tags e políticas de ACL.', '02. Implantação da tailnet: Instalamos o Tailscale nos dispositivos e servidores necessários, conectamos tudo à tailnet e configuramos autenticação centralizada com o provedor de identidade da empresa.', '03. Políticas e rotas privadas: Publicamos sub-redes locais via subnet router, configuramos exit nodes quando necessário e aplicamos regras de acesso mínimo para cada time ou fornecedor.', '04. Validação e handoff: Validamos o acesso remoto em produção, documentamos o fluxo operacional e treinamos a equipe para inclusão de novos dispositivos e troubleshooting básico.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'network',
    'published',
    210,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'twenty-crm',
    'Twenty CRM',
    'CRM open-source e self-hosted — alternativa ao Salesforce com zero custo de licença.',
    '## Sobre o Serviço

O Twenty CRM surgiu em 2023 como uma das iniciativas open-source de crescimento mais rápido no espaço de CRM. Fundado pela empresa francesa Twenty e construído com tecnologias modernas — React no frontend, NestJS no backend, GraphQL para a API — o projeto ultrapassou 10 mil estrelas no GitHub em menos de um ano após o lançamento, o que demonstra o nível de insatisfação com as alternativas existentes.

O problema que o Twenty resolve é o custo proibitivo dos CRMs tradicionais: o Salesforce cobra entre R$ 300 e R$ 1.500 por usuário por mês, o HubSpot tem um plano gratuito muito limitado e planos pagos a partir de R$ 450/mês. Para equipes comerciais de 5 a 20 pessoas, isso representa R$ 30.000 a R$ 180.000 por ano apenas em software de CRM. Com o Twenty auto-hospedado, o custo se limita à VPS e ao serviço de implantação.

A PaperMoon instala o Twenty CRM na VPS do cliente com configuração de produção completa (PostgreSQL, Redis, MinIO para armazenamento de arquivos), configura o pipeline de vendas conforme o processo comercial da empresa, importa a base de contatos existente e realiza treinamento da equipe. A sincronização de e-mail e a integração com Google Calendar ou Outlook Calendar são configuradas no mesmo escopo.

## Diferenciais

- Zero mensalidade por usuário — economize R$ 30 mil+/ano versus Salesforce ou HubSpot
- Interface moderna similar ao Notion: curva de aprendizado mínima para equipes comerciais
- Pipeline de vendas configurável conforme o processo real da empresa, não um template genérico
- API GraphQL para integrações com ERPs, planilhas e automações via n8n
- Seus dados de clientes e oportunidades ficam na sua VPS — nunca em servidor de terceiros

## O que nós entregamos

- Instalação do Twenty CRM via Docker Compose em servidor Linux
- Configuração de banco de dados PostgreSQL, Redis e storage de arquivos
- Customização de objetos: campos adicionais em contatos, empresas e deals
- Configuração de pipelines de vendas com etapas personalizadas por equipe
- Integração de e-mail corporativo (Gmail/Outlook) para sincronização de conversas
- Configuração de SSL, proxy Nginx e backups automáticos
- Treinamento da equipe comercial e documentação de operação

## Responsabilidades do Cliente

- Definir as etapas do pipeline de vendas e campos personalizados por objeto
- Configurar credenciais OAuth da conta de e-mail para sincronização
- Migrar dados de CRM anterior (planilhas ou exportação do CRM atual)
- Gerenciar usuários, permissões e criação de pipelines após entrega

## Recursos e Funcionalidades

### Pipeline e vendas

- Pipelines configuráveis com etapas personalizadas
- Kanban de deals com drag-and-drop entre etapas
- Campos customizados por objeto (contato, empresa, deal)
- Filtros e visões salvas por equipe ou vendedor
- Relatórios de funil: taxa de conversão por etapa

### Comunicação e atividades

- Sincronização bidirecional de e-mails (Gmail e Outlook)
- Log de chamadas, reuniões e tarefas por contato
- Notas com menção de membros da equipe
- Integração com Google Calendar e Outlook Calendar
- Timeline completa de interações por empresa e contato

### Dados e integração

- API GraphQL nativa para integração com outros sistemas
- Importação via CSV de qualquer CRM ou planilha
- Webhooks para automação com N8N ou Zapier
- 100% self-hosted — dados ficam no seu servidor
- Modelo de dados extensível via metadados (sem código)

## Perguntas Frequentes (FAQ)

### Twenty CRM é adequado para que tamanho de empresa?

O Twenty é ideal para startups, PMEs e equipes comerciais de até ~200 pessoas. É especialmente atraente para quem está saindo de planilhas ou quer uma alternativa self-hosted ao HubSpot ou Pipedrive sem custo por usuário.

### É possível migrar dados do HubSpot, Salesforce ou Pipedrive?

Sim. O Twenty aceita importação via CSV e tem API GraphQL que permite migração programática. A PaperMoon cuida da migração como parte da entrega — mapeamos campos do CRM atual para os objetos do Twenty e importamos contatos, empresas e histórico de deals.

### O Twenty CRM tem aplicativo mobile?

O Twenty tem interface web responsiva que funciona bem em dispositivos móveis. Um app nativo (iOS/Android) está no roadmap do projeto open-source. Para equipes que precisam de app mobile robusto agora, avalie se a interface web atende ou considere complementar com um app PWA.

### Como funciona a cobrança?

A implantação (instalação, configuração, customização de campos e pipelines, migração de dados e treinamento) é cobrada como projeto único. Não há custo de licença — Twenty é open-source com licença AGPL. O cliente paga apenas o servidor.

### O Twenty CRM tem integração com e-mail e Google Calendar?

Sim. O Twenty integra com Gmail e Google Calendar via OAuth2, e com qualquer servidor de e-mail via IMAP. Mensagens e reuniões com contatos do CRM aparecem automaticamente no histórico do contato ou empresa — sem copiar e colar manualmente. A PaperMoon configura as integrações como parte da entrega.',
    'Produtividade e CRM',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 2 vCPU, 4 GB RAM, 40 GB SSD)', 'Acesso SSH ao servidor com usuário root ou sudo', 'Domínio configurado com DNS apontando para o servidor', 'Exportação de dados do CRM atual (CSV ou API) se migração for necessária', 'Credenciais OAuth2 de conta Google ou Microsoft para sincronização de e-mail (opcional)']::text[],
    ARRAY['01. Mapeamento do processo comercial: Levantamos as etapas do pipeline, campos customizados de contatos e empresas, e fluxo de atividades da equipe. Esse mapeamento garante que o Twenty reflita o processo real de vendas.', '02. Instalação e configuração: Instalamos o Twenty via Docker Compose com PostgreSQL, Redis e storage. Configuramos SSL, proxy Nginx e integração de e-mail corporativo para sincronização automática de conversas.', '03. Customização e migração de dados: Criamos campos personalizados, pipelines e etapas. Importamos os dados do CRM anterior (HubSpot, Salesforce, Pipedrive ou planilhas) via API ou CSV.', '04. Treinamento e entrega: Treinamos a equipe comercial no uso do Twenty: criação de deals, atividades, filtros e relatórios. Entregamos documentação operacional e guia de administração.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'building',
    'published',
    220,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'papermark',
    'Papermark',
    'Compartilhe propostas e documentos com rastreamento total — alternativa ao DocSend.',
    '## Sobre o Serviço

O Papermark surgiu em 2023 como alternativa open-source ao DocSend — ferramenta da Dropbox que cobra entre US$ 15 e US$ 65 por usuário por mês para rastrear o engajamento com propostas e documentos compartilhados. Construído em Next.js com banco PostgreSQL, o projeto foi criado como resposta direta ao aumento de preços do DocSend após sua aquisição pela Dropbox e rapidamente se tornou uma das ferramentas mais adotadas por equipes comerciais que precisam de inteligência sobre suas propostas.

O Papermark resolve um problema específico e valioso: quando você envia uma proposta por e-mail ou PDF, não sabe se o prospect chegou a abrir, quanto tempo passou lendo, quais páginas mais interessaram e se encaminhou para outra pessoa. Com o Papermark, cada documento é compartilhado por um link único com rastreamento: você recebe notificação em tempo real quando o documento é aberto e tem acesso a um dashboard com tempo por página, porcentagem lida e número de visualizações.

A PaperMoon instala o Papermark na VPS do cliente com configuração completa de produção (PostgreSQL, S3-compatible storage para os documentos, SSL), configura as notificações por e-mail e Slack, e orienta a equipe comercial no uso da ferramenta. Documentos confidenciais — propostas financeiras, contratos, memorandos — nunca transitam por servidores de terceiros.

## Diferenciais

- Self-hosted: propostas e documentos confidenciais ficam na sua infraestrutura, nunca no DocSend/Dropbox
- Rastreamento por página: saiba em qual parte da proposta o cliente passou mais tempo
- Notificação em tempo real quando o prospect abre o documento — momento ideal para follow-up
- Link protegido por senha e com validade configurável — controle total sobre quem acessa
- Zero custo de mensalidade versus US$ 15–65/usuário/mês do DocSend

## O que nós entregamos

- Instalação do Papermark via Docker Compose em servidor Linux
- Configuração de banco de dados PostgreSQL e storage de documentos (S3 ou local)
- Configuração de domínio customizado para os links de compartilhamento
- Integração de e-mail transacional para notificações de visualização
- Configuração de SSL, proxy Nginx e backups automáticos
- Treinamento da equipe comercial e documentação de operação

## Responsabilidades do Cliente

- Definir domínio customizado para os links de documentos
- Fornecer credenciais de e-mail transacional (SendGrid, Resend ou SMTP próprio)
- Gerenciar upload de documentos e criação de espaços de trabalho após entrega

## Recursos e Funcionalidades

### Compartilhamento seguro

- Links únicos por destinatário com rastreamento individual
- Proteção por e-mail verificado ou senha por link
- Expiração de link por data ou número de visualizações
- Desativação imediata de link sem alterar o documento
- Suporte a PDF, DOCX, PPTX e arquivos de imagem

### Analytics de engajamento

- Tempo de leitura por página com gráfico de atenção
- Total de visualizações por link e por documento
- Localização geográfica aproximada do leitor
- Notificação em tempo real por e-mail na abertura do link
- Histórico completo de sessões por destinatário

### Controle e privacidade

- 100% self-hosted — documentos no seu servidor
- Sem marca d''água de terceiros nos documentos
- Domínio customizado nos links de compartilhamento
- Sem limite de documentos ou visualizações (limitado só pelo storage)
- Logs de acesso para conformidade e auditoria interna

## Perguntas Frequentes (FAQ)

### Qual a diferença do Papermark para o DocSend?

DocSend é um SaaS pago por documento ou usuário com dados armazenados nos servidores deles. O Papermark self-hosted é open-source, sem custo de licença e com documentos no seu servidor — ideal para empresas que compartilham propostas confidenciais e precisam de total controle sobre os dados.

### Que tipos de arquivo posso compartilhar?

PDFs são o formato principal (com analytics por página). O Papermark também suporta DOCX, PPTX e imagens — esses são convertidos para visualização no browser. Para melhores analytics de tempo por página, recomendamos exportar documentos como PDF antes de subir.

### Os destinatários precisam criar conta para abrir o documento?

Não. O destinatário recebe um link e abre o documento diretamente no browser — sem instalar nada nem criar conta. Você pode pedir o e-mail dele antes de liberar o acesso (verificação de e-mail) ou manter o link aberto sem autenticação.

### Como funciona a cobrança?

A implantação (instalação, configuração de storage, domínio customizado, e-mail transacional e treinamento) é cobrada como projeto único. Papermark é open-source — sem licença. O cliente paga apenas o servidor e o storage.

### Posso criar apresentações no Papermark ou apenas subir documentos?

O Papermark trabalha com documentos existentes — você sobe PDFs, DOCX ou PPTX e o sistema gera o link rastreável. Para criar apresentações dentro da plataforma, recomendamos criar o deck no PowerPoint, Keynote ou Canva, exportar como PDF e subir no Papermark para o compartilhamento rastreável.',
    'Produtividade e CRM',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['VPS com Ubuntu 22.04 LTS (mínimo 1 vCPU, 2 GB RAM, 20 GB SSD + storage para documentos)', 'Acesso SSH ao servidor com usuário root ou sudo', 'Domínio configurado com DNS apontando para o servidor', 'Credenciais de provedor de e-mail transacional (Resend, SendGrid ou SMTP próprio)']::text[],
    ARRAY['01. Instalação e configuração: Instalamos o Papermark via Docker Compose com PostgreSQL e storage. Configuramos SSL, domínio customizado e integração de e-mail para notificações de abertura de documentos.', '02. Configuração de storage de documentos: Configuramos o armazenamento dos documentos: S3 compatível (MinIO self-hosted ou S3 da AWS) ou volume local — conforme sua preferência de custo e controle.', '03. Treinamento e entrega: Treinamos a equipe no upload de documentos, criação de links rastreáveis, proteção por senha ou e-mail, e interpretação dos analytics de leitura por página.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'file-text',
    'published',
    230,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'crowdsec',
    'CrowdSec — Segurança Colaborativa',
    'Proteção automática contra ataques com inteligência de ameaças compartilhada globalmente.',
    '## Sobre o Serviço

O CrowdSec foi fundado em 2020 por uma equipe de ex-pesquisadores de segurança franceses e representa uma evolução do paradigma de proteção de servidores: enquanto ferramentas como o Fail2ban analisam os logs localmente e bloqueiam IPs apenas com base no comportamento observado no próprio servidor, o CrowdSec adiciona uma camada de inteligência coletiva — uma rede global de mais de 100 mil servidores que compartilham informações sobre IPs maliciosos em tempo real.

A distinção técnica é importante: o CrowdSec é um IDS/IPS comportamental, não um firewall. O firewall (iptables, UFW, security groups) bloqueia portas e protocolos de forma estática. O CrowdSec analisa os logs das aplicações em tempo real, identifica padrões de comportamento malicioso (tentativas de brute force em SSH, scans de vulnerabilidades em Nginx, exploits em WordPress) e bloqueia o IP atacante dinamicamente antes que o ataque seja concluído — mesmo que o IP nunca tenha atacado especificamente o seu servidor antes, se ele já atacou outros servidores na rede colaborativa, você já está protegido.

A PaperMoon instala o agente CrowdSec em todos os servidores da infraestrutura do cliente, configura os cenários de detecção para os serviços em produção (SSH, Nginx, API, painel admin), implanta os bouncers de bloqueio (Nginx bouncer, Cloudflare bouncer, iptables bouncer) e configura o dashboard web para visualização de ataques bloqueados em tempo real.

## Diferenciais

- Inteligência coletiva de +100 mil servidores: IPs maliciosos bloqueados antes do primeiro ataque ao seu servidor
- Análise comportamental em camada de aplicação — detecta o que firewalls estáticos não conseguem ver
- Bouncers para Nginx, Traefik, Cloudflare e iptables: o bloqueio acontece no ponto certo da pilha
- Dashboard web com histórico completo de ataques bloqueados e mapa geográfico de origem
- Evolução do Fail2ban: mesma filosofia, com coordenação entre servidores e inteligência global

## O que nós entregamos

- Instalação do CrowdSec Agent em todos os servidores Linux da sua infraestrutura
- Configuração de cenários de detecção: brute-force SSH, ataques web, exploração de CVEs
- Configuração de bouncers: Nginx (HTTP 403), iptables (bloqueio de IP), Cloudflare
- Integração de logs: SSH, Nginx, Apache, Traefik, aplicações custom
- Dashboard local via CrowdSec Console ou integração com Grafana
- Configuração de alertas e relatórios de ameaças por e-mail ou Slack
- Documentação de operação e treinamento da equipe de TI

## Responsabilidades do Cliente

- Fornecer acesso SSH aos servidores onde o CrowdSec será instalado
- Definir política de bloqueio: duração de ban, whitelist de IPs internos
- Aprovar integração com a rede colaborativa CrowdSec para compartilhamento de IPs
- Gerenciar whitelists e decisões de bloqueio manual após entrega

## Recursos e Funcionalidades

### Detecção de ameaças

- Brute-force SSH, RDP, FTP e painéis admin web
- Scan de portas e fingerprinting de serviços
- Exploração de CVEs conhecidos em tempo real
- Flood de requisições HTTP (DDoS de camada 7)
- Crawlers maliciosos, bots de spam e scrapers agressivos

### Bloqueio e resposta

- Bloqueio automático de IP via iptables ou nftables
- Integração com Nginx/Traefik para retorno 403 sem consumo de recursos
- Bouncer Cloudflare para bloqueio na borda CDN
- Decisões com tempo configurável: 1h, 24h, permanente
- Whitelist de IPs internos, escritórios e VPNs

### Inteligência colaborativa

- Acesso à blocklist da rede global CrowdSec (milhões de IPs)
- Seu servidor contribui com IPs detectados (opt-in configurável)
- Reputação de IP em tempo real antes do primeiro ataque chegar
- Listas de terceiros: Tor exit nodes, VPNs conhecidas, hosting abusivo
- API REST para consultar reputação de IPs programaticamente

## Perguntas Frequentes (FAQ)

### Qual a diferença entre CrowdSec e Fail2ban?

Fail2ban analisa logs localmente e bloqueia por IP — simples e eficaz, mas sem contexto global. CrowdSec adiciona detecção comportamental mais sofisticada, múltiplos bouncers simultâneos e acesso à inteligência coletiva de milhões de sensores globais — o IP de um atacante identificado em outro país já chega bloqueado na sua rede antes do primeiro ataque.

### O CrowdSec envia dados da minha rede para fora?

Por padrão, o CrowdSec compartilha com a rede colaborativa apenas os IPs atacantes detectados (não dados de conteúdo, usuários ou logs). Esse compartilhamento pode ser desativado — o produto continua funcionando normalmente só com detecção local. A PaperMoon documenta e configura conforme a política de privacidade da empresa.

### CrowdSec substitui o firewall?

Não — é complementar. O firewall (iptables, UFW, security groups) bloqueia portas e protocolos. O CrowdSec analisa o comportamento em nível de aplicação e bloqueia IPs dinamicamente com base em padrões de ataque — algo que firewalls estáticos não fazem. Os dois devem existir juntos.

### Funciona com Nginx, Traefik e outros proxies reversos?

Sim. O CrowdSec tem bouncers para Nginx, Traefik, Caddy, Apache, HAProxy e Cloudflare. O bouncer HTTP retorna 403 ou redirect para captcha antes que a requisição chegue à aplicação — sem consumir recursos do backend.

### Como funciona a cobrança?

A instalação, configuração de cenários e bouncers, dashboard e treinamento são cobrados como projeto único. CrowdSec é open-source e a rede colaborativa tem tier gratuito. O cliente paga apenas os servidores onde o agente roda.',
    'Segurança e Identidade',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Servidores Linux com Ubuntu 20.04+ ou Debian 10+ (agente roda em qualquer serviço exposto à internet)', 'Acesso SSH a todos os servidores onde o CrowdSec será instalado', 'Lista de serviços expostos: SSH, Nginx, APIs, painéis admin', 'Definição de política de bloqueio: duração de ban e IPs a colocar em whitelist']::text[],
    ARRAY['01. Auditoria da infraestrutura: Mapeamos todos os servidores, serviços expostos e fontes de log disponíveis. Identificamos os vetores de ataque mais relevantes para definir os cenários de detecção prioritários.', '02. Instalação do agente e cenários: Instalamos o CrowdSec Agent em cada servidor e configuramos os parsers de log e cenários de detecção: SSH, Nginx/Apache, APIs e CVEs conhecidos. Em horas, os primeiros IPs maliciosos já são detectados.', '03. Configuração dos bouncers: Instalamos os bouncers que executam o bloqueio: iptables (bloqueio de IP na camada de rede), Nginx/Traefik (retorno 403 na camada HTTP) ou integração com Cloudflare para bloquear na borda.', '04. Dashboard e alertas: Configuramos dashboard de métricas (CrowdSec Console ou Grafana) e alertas em tempo real para ataques detectados. Entregamos documentação e treinamos a equipe na leitura das decisões.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'shield-check',
    'published',
    240,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'terraform',
    'Terraform',
    'Infraestrutura como Código (IaC) para ambientes multi-cloud e locais.',
    '## Sobre o Serviço

O Terraform é a ferramenta padrão de mercado para Infraestrutura como Código (IaC), permitindo descrever topologias inteiras de servidores, redes, firewalls e storages em arquivos de configuração declarativos (HCL).

Em vez de criar máquinas virtuais e configurar portas manualmente via painéis gráficos — processo lento e propenso a falhas humanas —, o Terraform documenta o estado desejado da infraestrutura no Git. Qualquer alteração passa por revisão de código, é pré-visualizada com precisão (terraform plan) e aplicada de forma atômica e previsível (terraform apply).

A equipe configura os módulos de infraestrutura sob medida para Proxmox, AWS, Azure, Google Cloud ou provedores locais, estabelecendo repositório Git com controle de versão, estado remoto seguro e travamento contra concorrência.

## Diferenciais

- Provisionamento declarativo e reprodutível: elimine configurações manuais e divergências
- Suporte multi-provedor: compatível com Proxmox, AWS, Azure, GCP, Cloudflare e bare-metal
- Histórico e governança: toda mudança na infraestrutura é auditável e versionada no Git
- Prevenção de falhas com plano de execução: saiba exatamente o que será alterado antes de aplicar
- Modularidade: componentes reutilizáveis para criação ágil de novos ambientes (Dev, Homologação, Produção)

## O que nós entregamos

- Mapeia a arquitetura e define a estrutura de módulos Terraform
- Escreve os manifests HCL para provisionamento de VMs, redes e recursos
- Configura backend remoto seguro (MinIO/S3) com travamento de estado (state locking)
- Implementa pipeline de CI/CD para validação automática (plan/apply)
- Treina a equipe técnica nas operações diárias do Terraform

## Responsabilidades do Cliente

- Fornecer credenciais de acesso ou APIs com as permissões adequadas
- Disponibilizar repositório Git para hospedagem dos manifests
- Validar a topologia e dimensionamento dos servidores sugeridos
- Aprovar as mudanças de infraestrutura antes da execução em produção

## Recursos e Funcionalidades

### Automação e Confiabilidade

- Criação automática de máquinas virtuais e containers LXC no Proxmox
- Provisionamento de instâncias, VPCs, discos e rotas em provedores cloud
- Execução idempotente: apenas as alterações necessárias são aplicadas
- Destruição e recriação controlada de ambientes descartáveis

### Segurança e Estado

- State file armazenado em storage seguro e criptografado
- Travamento de concorrência para evitar execuções simultâneas conflitantes
- Gerenciamento seguro de segredos e chaves de acesso via variáveis de ambiente
- Detecção de desvio (drift detection) entre a infraestrutura real e o código

## Perguntas Frequentes (FAQ)

### O Terraform substitui ferramentas como o Ansible?

Não, eles são complementares. O Terraform é ideal para provisionar a infraestrutura (criar a VM, configurar a rede e os discos). O Ansible é ideal para configurar o sistema operacional e instalar os softwares dentro dessa VM.

### Funciona com servidores locais ou somente em nuvem?

Funciona perfeitamente em servidores locais com Proxmox VE, VMware ou OpenStack, assim como em nuvens públicas (AWS, Azure, GCP, Oracle Cloud) e híbridas.

### O que acontece se alguém alterar uma máquina manualmente?

O Terraform detecta a divergência (drift) na próxima execução e permite restaurar o estado padronizado ou atualizar o código para refletir a nova realidade.',
    'Infraestrutura',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Acesso com privilégios de API ao Proxmox VE ou provedor de nuvem contratado', 'Repositório Git institucional (GitHub, GitLab ou Gitea)', 'Bucket S3 ou MinIO compatível para armazenamento remoto do estado do Terraform']::text[],
    ARRAY['01. Levantamento e Arquitetura: Mapeamos todos os servidores, redes, sub-redes e regras de firewall atuais e planejadas para a organização.', '02. Desenvolvimento dos Módulos IaC: Codificamos a infraestrutura em linguagem declarativa HCL, parametrizando recursos e definindo variáveis de ambiente seguras.', '03. Validação e Execução Controlada: Executamos o plano de provisionamento em ambiente de testes, conferimos a conformidade e realizamos a entrega em produção com estado travado e monitorado.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'cloud',
    'published',
    250,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

INSERT INTO catalog_services (
    slug,
    title,
    summary,
    description,
    category,
    audience,
    requirements,
    steps,
    channels,
    sla,
    cost,
    icon,
    status,
    position,
    published_at
) VALUES (
    'ansible',
    'Ansible',
    'Automação de configurações, orquestração e padronização sem agentes.',
    '## Sobre o Serviço

O Ansible é uma plataforma de automação de TI open-source criada pela Red Hat que simplifica radicalmente a administração de sistemas. Diferente de soluções legadas que exigem a instalação de softwares pesados (agentes) em cada servidor, o Ansible opera de forma agentless, conectando-se diretamente via SSH em servidores Linux ou WinRM em servidores Windows.

Com o Ansible, rotinas operacionais críticas — como aplicar atualizações de segurança em lote, instalar pacotes padronizados, criar usuários corporativos, configurar regras de firewall e auditar conformidade — deixam de ser feitas manualmente em cada servidor. Tudo é descrito em playbooks claros e declarativos em formato YAML.

A equipe desenvolve playbooks e roles customizados para a infraestrutura da organização, implementando inventários dinâmicos, hardening de segurança e rotinas de automação com total garantia de idempotência: a execução só altera o que realmente precisa ser corrigido.

## Diferenciais

- Arquitetura Agentless: não requer instalação nem manutenção de agentes nos servidores gerenciados
- Idempotência garantida: executar um playbook várias vezes só altera o que estiver fora do padrão
- Playbooks legíveis em YAML: documentação viva e compreensível de todas as configurações do parque
- Hardening e conformidade em lote: aplique políticas de segurança em dezenas de servidores em minutos
- Inventários dinâmicos integrados ao Proxmox, Active Directory e nuvem

## O que nós entregamos

- Estrutura o inventário de servidores e grupos de máquinas da organização
- Desenvolve playbooks e roles modulares para as demandas operacionais
- Implementa rotinas automatizadas de atualização e hardening de segurança
- Configura o ambiente de execução centralizado (Ansible Runner / Semaphore)
- Capacita a equipe interna para execução e manutenção dos playbooks

## Responsabilidades do Cliente

- Disponibilizar acesso SSH com chave pública ou WinRM nos servidores-alvo
- Definir a lista de servidores a serem gerenciados e suas funções
- Informar as regras de negócio e softwares necessários para cada grupo de máquinas
- Aprovar as janelas de manutenção para execução de atualizações em lote

## Recursos e Funcionalidades

### Gerenciamento de Configuração

- Instalação e atualização padronizada de pacotes de software
- Configuração de serviços de rede, Nginx, Docker, PostgreSQL e bancos de dados
- Gerenciamento centralizado de usuários, grupos e chaves SSH autorizadas
- Distribuição e sincronização de arquivos de configuração com templates Jinja2

### Segurança e Hardening

- Aplicação de regras de firewall UFW / iptables em massa
- Desativação de serviços desnecessários e portas vulneráveis
- Aplicação de patches críticos de segurança do sistema operacional em minutos
- Auditoria automatizada de conformidade com diretrizes governamentais e CIS Benchmarks

## Perguntas Frequentes (FAQ)

### Preciso instalar algum software nos servidores gerenciados?

Não. O Ansible é agentless. Ele precisa apenas do serviço SSH padrão em servidores Linux ou WinRM em servidores Windows, além do interpretador Python padrão do sistema.

### Existe risco de um playbook quebrar um servidor em produção?

Os playbooks são desenvolvidos com o princípio da idempotência e testados previamente. Além disso, o Ansible possui o modo de simulação (--check / dry-run), que mostra exatamente o que seria alterado antes de qualquer comando ser executado.

### É possível agendar execuções automáticas?

Sim. Podemos configurar execuções agendadas via cron, pipelines de CI/CD ou interfaces web como Semaphore / AWX para auditoria e aplicação contínua de configurações.',
    'Automação',
    'Empresas, órgãos públicos e organizações que buscam infraestrutura e serviços de TI dedicados, sob medida e com controle total dos seus dados.',
    ARRAY['Acesso SSH com privilégios sudo/root nos servidores Linux ou WinRM nos servidores Windows', 'Servidores na mesma rede privada ou acessíveis via VPN corporativa (ex.: Tailscale)', 'Inventário básico dos servidores a serem gerenciados']::text[],
    ARRAY['01. Mapeamento do Parque e Acessos: Cadastramos os servidores no inventário, organizando-os por grupos funcionais (banco de dados, web, proxies, aplicações) e configuramos as chaves de comunicação segura.', '02. Criação dos Playbooks e Políticas: Codificamos as rotinas de configuração, regras de firewall, pacotes e hardening em roles e playbooks testados em ambiente de homologação.', '03. Automação em Produção e Monitoramento: Executamos a padronização no parque de servidores, gerando relatórios de conformidade e habilitando a execução periódica de rotinas automatizadas.']::text[],
    '[{"type":"online","label":"Portal Institucional","value":"https://papermoon.cloud"},{"type":"email","label":"Atendimento e Suporte","value":"contato@papermoon.cloud"},{"type":"telefone","label":"WhatsApp Oficial","value":"+55 66 99216-3988"}]'::jsonb,
    'Implantação em até 5 a 10 dias úteis; suporte contínuo.',
    'Sob consulta / Implementação e serviços gerenciados',
    'cpu',
    'published',
    260,
    now()
)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    audience = EXCLUDED.audience,
    requirements = EXCLUDED.requirements,
    steps = EXCLUDED.steps,
    channels = EXCLUDED.channels,
    sla = EXCLUDED.sla,
    cost = EXCLUDED.cost,
    icon = EXCLUDED.icon,
    position = EXCLUDED.position,
    updated_at = now();

-- +goose Down
DELETE FROM catalog_services WHERE slug IN ('whatsapp-api', 'chatwoot', 'n8n', 'glpi', 'zabbix', 'proxmox', 'truenas', 'nextcloud', 'aapanel', 'whatsapp-evolution', 'evolution-api', 'redes', 'cabeamento', 'manutencao', 'backup', 'rustdesk', 'windows-server', 'samba', 'plone', 'keycloak', 'tailscale', 'twenty-crm', 'papermark', 'crowdsec', 'terraform', 'ansible');
