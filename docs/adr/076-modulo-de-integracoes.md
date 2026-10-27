# 076 — Módulo de integrações: migrar de outra ferramenta pela tela, com um conector na rede do cliente

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 072](072-importador-do-glpi.md) (importador), [ADR 075](075-de-para-do-glpi.md) (tabela de-para), [ADR 077](077-usuarios-do-sso.md) (usuários do SSO)

## Contexto

A primeira migração real, do GLPI da prefeitura, foi feita por comando de linha e scripts. Ela mostrou o que o comando não resolve:

- **Rede.** O servidor do Projeto Nova não alcança o GLPI, que só existe na rede interna da prefeitura. Isso vai se repetir em quase toda organização.
- **Revisão.** A tabela de-para foi revisada num CSV no Excel, e o inventário (entidades, categorias, SLAs, volume) saiu de scripts avulsos.
- **Duração.** A leitura completa leva cerca de 40 minutos. Num comando, ela não tem progresso nem retomada.
- **Convivência.** Enquanto o GLPI ainda recebe chamados, a carga precisa ser repetida até a virada.

A maior parte das organizações que vão adotar o Projeto Nova vem do GLPI. Migrar precisa ser uma função do produto, não um trabalho de quem implanta.

## Decisão

1. **Um plug-in `integracoes`**, desligado por padrão, com a permissão `integracoes:manage`. Ele é só global: a migração mexe em todas as unidades.
2. **Extrair, preparar, carregar.** A migração tem três passos separados:
   - **Extração.** Lê a origem e grava os dados brutos numa **área de preparo** (`integracoes_itens`): um registro por chamado, já com participantes, acompanhamentos e satisfação, mais as pessoas, as categorias, as entidades e os SLAs. A extração é incremental: só traz o que mudou desde a última.
   - **Preparo.** O inventário e as sugestões da tabela de-para saem da área de preparo. A tabela fica no banco e é editada na tela.
   - **Carga.** Usa o `Importador` do ADR 072 e 075, com uma `Fonte` que lê a área de preparo. Simular e gravar são tarefas.
3. **A extração roda onde a origem estiver ao alcance:**
   - **direta:** o worker do Projeto Nova lê a origem, quando a rede deixa;
   - **pelo conector:** um programa (`conector`) roda numa máquina da rede do cliente, lê a origem e envia os lotes à API por HTTPS. É o modelo do agente de inventário e do MID Server do ServiceNow. O conector se autentica com um **token de máquina** da conexão, que só envia dados e lê as tarefas daquela conexão. As credenciais da origem ficam com o conector, e o servidor nunca as recebe.
4. **Credenciais cifradas.** No modo direto, as credenciais da origem ficam cifradas com a chave de configuração (`secretcrypto`), como as conexões de IA. A API nunca devolve o segredo.
5. **Tarefas no worker**, com estado (`pendente`, `em_andamento`, `concluida`, `falhou`, `cancelada`), progresso e relatório. Uma tarefa por conexão de cada vez. Se o worker cair, a tarefa volta para `pendente` e recomeça: a carga é idempotente.
6. **Origens plugáveis.** A conexão tem um tipo. Esta versão traz o GLPI. A área de preparo e a `Fonte` são genéricas, e outra origem (OTRS, Zammad, planilha) entra como outro extrator.
7. **Auditoria.** Criar conexão, emitir ou revogar token, editar a tabela e executar tarefa ficam na auditoria. A carga registra uma linha com os totais, como no ADR 072.

## Consequências

Migrar passa a ser um assistente: conexão, extração, inventário, tabela de-para, simulação, carga, e de novo até a virada. O comando `importarglpi` continua, para quem implanta sem a tela.

A área de preparo guarda dados da origem, inclusive nomes e e-mails. Ela é apagada quando a conexão é encerrada, e o tamanho é limitado: um chamado e as suas mensagens ficam num registro, com o texto cortado no limite do atendimento.
