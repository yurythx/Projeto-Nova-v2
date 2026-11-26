# Seeds desta pasta — só para demonstração

Os arquivos `seed_*_rondonopolis.sql` e `update_destaque_editorial_proxmox.sql`
são **dados fictícios da “Nova Corporation”** (personas, wiki de
onboarding, agenda, arquivos, trâmite). Servem só para demonstrar os
plug-ins em ambiente de teste.

`seed_protocolos_demo.sql` cria seis serviços fictícios do Protocolo Digital
(formulários, responsáveis técnicos e 46 pedidos em todos os estados) nas
unidades PREFEITURA, SMS e SEMECEL; `remover_protocolos_demo.sql` apaga tudo
o que ele criou e os pedidos de teste abertos nesses serviços.

**Não aplicar** em homologação nem em produção. A estrutura organizacional
real entra por `deploy/estrutura` e pelo mapeamento do AD (`deploy/iam`).
