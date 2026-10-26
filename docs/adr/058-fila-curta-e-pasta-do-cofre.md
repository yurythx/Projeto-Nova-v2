# 058 — Fila curta e pasta do cofre

- **Status:** Aceito e implementado
- **Data:** 2026-10-25
- **Relacionado:** [ADR 047](047-avaliacao-governanca-e-cascata.md), [ADR 052](052-programas-e-credencial-de-agente.md), [ADR 057](057-rotacao-do-cofre.md)

## Contexto

A fila já classifica, avisa a unidade e mede prazo. Faltava o hábito
curto de triagem, o texto que a equipe cola, o chamado que volta todo
mês, a conta honesta da licença contra o que o agente viu, a pasta que
esconde a senha dentro da unidade e o número de quem viu um artigo e
não abriu pedido.

## Decisão

1. A regra da unidade casa oferta, palavra no título ou lotação de quem
   pede. A primeira, pela ordem, vale. Sem regra, o chamado segue como
   hoje. A prioridade vira impacto e urgência pela matriz já gravada;
   se a matriz não tem aquela prioridade, ela não muda. A equipe da
   regra só é avisada. Ninguém é marcado responsável no lugar da pessoa.
2. Cada oferta tem uma resposta pública e uma nota interna. A equipe
   cola o texto. O envio continua manual.
3. A recorrência abre um chamado no dia 1–28, no fuso da política da
   unidade (ou Brasília), uma vez naquele mês. Quem criou a marca é o
   solicitante. O mês já aberto não abre de novo.
4. A licença pode nomear o programa. A conta lê os ativos em uso ou
   manutenção da unidade. Se nenhuma máquina informou programas, a tela
   diz isso. Se o agente cortou a lista em 100, a conta vem marcada
   como parcial. Sem o nome, não há comparação.
5. A pasta do cofre não é role do Keycloak. Quem não está nela não vê o
   segredo, mesmo com leitura da unidade. Revelar ainda pede a senha de
   acesso e a permissão `cofre:reveal` da unidade, além da marca de
   revelar na pasta. Quem administra a unidade continua vendo. Apagar a
   pasta devolve o segredo à unidade. Mover de unidade tira da pasta.
6. A deflexão conta, nos últimos 30 dias, pessoas que viram artigo e
   não abriram pedido no mesmo dia. Sem vista, o campo não vem. Zero
   com amostra é zero; zero sem amostra não aparece.

## Consequências

As migrations `000014_fila_de_atendimento`, `000015_licenca_programa` e
`000016_cofre_pasta` acrescentam as tabelas. O worker `recorrencias`
só corre com o atendimento ligado. A comparação da licença só roda com
o módulo de ativos ligado. A rotação, a retirada e o endereço do cofre
permanecem.
