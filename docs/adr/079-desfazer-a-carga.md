# ADR 079 — Desfazer a carga do GLPI

## Contexto

A carga do GLPI ([ADR 076](076-modulo-de-integracoes.md)) gravou na produção 14.998 chamados, 51 ofertas, 4 categorias do catálogo, 2 equipes e a política de prazos institucional. Não havia como voltar atrás:

- não existe rota que apague chamados;
- o backup do banco volta tudo, inclusive o que foi feito depois da carga;
- apagar à mão esbarra nas chaves: a oferta não sai enquanto houver chamado (`RESTRICT`), e `problemas_incidentes` e `atendimento_recorrencia_aberturas` apontam para o chamado sem chave nenhuma.

O registro da importação (`atendimento_importacao`) guardava só os chamados e as ofertas. As ofertas incluíam as que a tabela de-para só reaproveitou. Categorias, equipes, política e feriados criados pela carga não ficavam registrados.

## Decisão

1. **Duas tarefas novas na conexão:**
   - `desfazer_simulacao` faz todo o trabalho numa transação e a desfaz no fim, então os números são os da execução;
   - `desfazer` grava.

   As duas rodam no worker das integrações, uma de cada vez por conexão, como a carga.
2. **Confirmação:** o `desfazer` só é aceito com `confirmacao` igual ao nome da conexão. A tela pede que se digite o nome e sugere simular antes. Só quem tem `integracoes:manage` global pede.
3. **O que sai:**
   - os chamados importados, com o que cai em cascata (histórico, comentários, avaliações, eventos);
   - os cadastros que a carga criou: ofertas (com regras de fila e etapas), categorias, equipes, a política institucional e os feriados.
4. **O que fica, na dúvida a favor de ficar:**
   - **o chamado em que alguém mexeu aqui depois da carga.** Conta como mexido:
     - auditoria da solicitação posterior ao registro;
     - comentário, passagem ou avaliação com data posterior ao registro;
     - qualquer atribuição, apontamento, tarefa, aprovação, visita, sugestão da IA ou e-mail recebido;
     - vínculo com ocorrência ou problema.

     A carga grava as datas do GLPI e o registro nasce na transação do chamado. O que é posterior ao registro foi feito aqui. Um falso positivo só deixa o chamado ficar.
   - **o cadastro alterado aqui depois da carga** (auditoria do recurso), **ou ainda em uso** pelo que fica:
     - a oferta, por um chamado;
     - a categoria, por uma oferta;
     - a equipe, por regra, modelo de tarefa, tarefa ou atribuição.
   - **a oferta que já existia** e que a tabela só reaproveitou. O registro ganha a coluna `criado`.
5. **O registro** passa a guardar categorias, equipes, política e feriados (`marcarCriado`, sem repetir). No fim do desfazer, sai do registro o que já não existe, e uma carga nova traz de novo. O que ficou continua registrado e não se repete.
6. **O relatório** traz:
   - quantos de cada tipo saíram;
   - quantos chamados ficaram, com até 20 números;
   - quantos cadastros ficaram.

## Consequências

- O GLPI não é tocado: desfazer e carregar de novo é seguro até a virada.
- O registro é por origem (`glpi`), não por conexão. Com uma conexão GLPI, como hoje, é o mesmo. Uma segunda conexão GLPI exigiria o desfazer por conexão.
- A categoria que a carga pôs numa oferta que já existia (quando ela não tinha categoria) fica, porque continua em uso. O relatório a conta entre os cadastros que ficaram.
- A remoção é uma transação só. Para 15 mil chamados, leva segundos. Uma carga ao mesmo tempo espera a trava do registro.
- Depois da virada, o desfazer perde o sentido: a conexão é apagada junto com o token do conector.
- O backup antes de desfazer continua recomendado. A tela lembra disso.

## Implementação

- **Migração 40:**
  - tipos novos nas tarefas e no registro;
  - a coluna `criado`, falsa para a oferta reaproveitada (a que não tem a descrição da carga);
  - o registro das categorias já importadas.
- `atendimento/glpi/desfazer.go` (`Desfazer`) e o registro dos cadastros criados em `importar.go` e `expediente.go`.
- Integrações:
  - `Carregador.Desfazer`;
  - `PedirTarefa(..., confirmacao)`;
  - o worker pega os tipos novos;
  - `TarefaRequest.confirmacao`.
- Tela da conexão: o bloco "Desfazer a carga" e o resumo legível do relatório.
- Os testes que mexem no registro, que é global, se revezam por uma trava do banco (`dbtest.Exclusivo`).
