# 061 — Catálogo das colunas dos quadros

- **Status:** Aceito e implementado
- **Data:** 2026-10-26
- **Relacionado:** [ADR 016](016-camada-de-aplicacao-e-transacoes.md), [ADR 060](060-pasta-vazia-para-apagar.md)

## Contexto

O nome e a cor da coluna ficavam no navegador. Duas pessoas viam quadros diferentes. A ordem numérica, a partir de zero, já era a posição do fluxo, mas também nascia no código de cada tela.

A coluna é o estado do registro. Apagar a coluna apagaria a história. Inventar uma chave nova na tela criaria um estado que o fluxo do módulo não conhece.

## Decisão

O módulo `quadros` guarda, por quadro, a chave, a ordem, o nome, a cor, o tom, se a coluna é a inicial e se ela está no quadro. Problemas, mudanças, projetos, tarefas, ativos e a fila leem o mesmo catálogo no quadro e na lista.

A chave não muda. Nome, cor, ordem e retirada ficam nesta API. Uma coluna nova, com chave gerada e passagens próprias, nasce pela API descrita em [ADR 062](062-coluna-nova-no-quadro.md). Mover o card continua na API de estado de cada módulo.

Quem lê o quadro lê o catálogo, com a permissão que já abre a tela. Quem gere o módulo renomeia, recolore, reordena, retira e devolve na página Colunas daquele quadro. A barra da coluna não oferece essas ações. Não há permissão nova.

Retirar exige coluna vazia. A coluna inicial não sai. A contagem é SQL somente leitura nas tabelas dos outros módulos (`problemas`, `mudancas`, `projetos`, `projetos_tarefas`, `ativos`, `atendimento_solicitacoes`), sem importar esses plug-ins. Na mudança, `fechada` conta `rejeitada`, `falhou` e `cancelada`. Na fila, `concluida` conta `resolvido` e `atendido`.

Cada alteração grava o outbox e a auditoria na mesma transação: quem, a chave, a ordem anterior e a nova. O nome não substitui o estado.

A migração semeia o catálogo de fábrica. Nome e cor que estavam só no navegador não migram. Se o catálogo não responde, a tela usa de novo o quadro de fábrica.

Devolver uma coluna retirada coloca ela no fim e renumera as ativas a partir de zero.

## Consequências

O nome oficial é o mesmo para quem abre o quadro. Recolher a coluna e a cor de um card continuam neste navegador. Desligar o módulo `quadros` devolve as telas ao quadro de fábrica até ele voltar.
