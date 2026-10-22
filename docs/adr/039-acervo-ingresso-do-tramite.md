# 039 — Acervo: o processo encerrado do Trâmite vira unidade de arquivamento

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 2 do [roadmap](../ROADMAP_SISMARQ.md), item da ponte)
- **Data:** 2026-10-16
- **Relacionado:** [ADR 026](026-tramite-e-atlas-publico.md) (classificação e guarda), [ADR 033](033-acervo-emprestimo-guia-e-importacao.md) (a ponte ficou para depois), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

O ADR 033 deixou o ingresso do processo encerrado para a decisão 0.7 e o
item 7 da seção 6 do roadmap: o SEI é a fase corrente no artigo; o
Trâmite da plataforma é ponte para o que **não** passa pelo SEI. Enquanto
a Prefeitura não confirmar o SEI, o Trâmite já classifica pela TTDD e
lista a guarda (ADR 026). Sem ingresso, essa guarda não vira acervo.

## Decisão

1. **Evento, não import cruzado.** O Trâmite já emite `tramite.processo.concluido`.
   O Acervo consome a fila `sismarq.acervo.tramite`. Não há `DependsOn`:
   o Acervo funciona sem o Trâmite (cadastro pela tela, importação, SEI
   na Fase 7).
2. **Referência por valor**, como o ADR 026: `tramite_processo_id` na
   unidade, `UNIQUE`, **sem FK** entre plug-ins.
3. **Só com série da TTDD.** Sem `codigo_ttdd` o consumer ignora (não é
   falha, não vai para a DLQ). A listagem de guarda do Trâmite continua
   pedindo classificação.
4. **Idempotente.** Concluir de novo (depois de reabrir) não duplica.
   Reabrir **não apaga** a unidade: o ingresso é fato arquivístico.
5. **Proveniência e sigilo.** Unidade produtora = origem do processo.
   Tipo `PROCESSO`. Datas-limite = abertura → conclusão. Público /
   restrito / sigiloso seguem o sigilo do Trâmite, com hipótese legal
   explícita no restrito e no sigiloso. O **assunto** só viaja no evento
   se o processo for público (o barramento não vaza conteúdo).
6. **Sem IAM da tela.** O consumer não usa lotação: o processo já existia
   na unidade. Cadastro e importação continuam exigindo `acervo:manage`.
7. **Arquivar** não gera segundo ingresso (só `concluido`).

O envelope do evento passa a trazer `codigo_ttdd`, `unidade_origem_id`,
`created_at` e `concluido_at` — metadados, não conteúdo.

## Consequências

- A listagem de guarda do Trâmite (ADR 026) **orienta**; a eliminação
  continua na esteira da CCPAD, agora sobre a unidade de arquivamento.
- O SEI (Fase 7, decisão 0.7) entra pelo mesmo padrão: evento → ingresso,
  sem o Acervo importar o cliente SEI.
- A ficha da unidade aponta para `/tramite/{id}` quando veio do Trâmite.
