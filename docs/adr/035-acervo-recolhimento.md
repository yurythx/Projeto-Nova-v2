# 035 — Acervo, Fase 4: recolhimento ao Arquivo Público e registro de entrada

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (primeira parte da Fase 4 do [roadmap](../ROADMAP_SISMARQ.md); arquivos privados e instruções normativas ficam para a segunda parte — ver Consequências)
- **Data:** 2026-10-15
- **Relacionado:** [ADR 032](032-acervo-unidades-de-arquivamento.md) (fases e travas), [ADR 033](033-acervo-emprestimo-guia-e-importacao.md) (guia com aceite), [ADR 034](034-acervo-eliminacao.md) (amostra permanente), [ADR 027](027-notificacoes.md) (avisos)

## Contexto

A destinação final de uma série é a eliminação (Fase 3) ou a guarda
permanente. O que é permanente vai ao Arquivo Público, e a Lei Municipal
10.301/2019 diz como:

- o conjunto chega **classificado, avaliado, organizado, higienizado,
  acondicionado e com instrumento descritivo** que permita a identificação,
  o acesso e o controle (art. 53). O preparo e o transporte são custeados
  pelo órgão (§ 2º);
- o Arquivo Público **registra a entrada e encaminha cópia do registro à
  unidade de origem** (art. 5º, VII);
- a extinção do órgão (arts. 17 e 50) e a desestatização (arts. 16 e 51)
  obrigam o recolhimento da guarda permanente, sem esperar os prazos;
- o permanente pode ficar com o órgão produtor (art. 42) — é o caso da
  amostra da eliminação (art. 39), que já nasce permanente;
- o recolhimento segue o cronograma do Regimento Interno (art. 47), que
  ainda não existe (decisão 0.2 do roadmap).

## Decisão

1. **Guarda do Arquivo Público na unidade**: `acervo_unidades` ganha
   `recolhida_em` e `recolhimento_id`. A situação calculada passa a ser
   `RECOLHIDA` ("No Arquivo Público"). A unidade produtora (`unidade_id`)
   **segue como proveniência** e não muda mais. Um gatilho no banco impede
   devolver a recolhida ao órgão ou trocar a proveniência; CHECKs exigem fase
   permanente e os dois campos juntos.
2. **Quem gere** a unidade recolhida é o Arquivo Público (`acervo:central`,
   concessão global); a origem continua consultando. Antes da entrada, nada
   muda.
3. **Recolhimento** (`acervo_recolhimentos`, número `RC-AAAA-NNNNN`):
   - a origem (`acervo:manage` no escopo) relaciona até 2.000 unidades,
     escolhe o **motivo** — `REGULAR`, `EXTINCAO` ou `DESESTATIZACAO`; os
     especiais exigem o ato ou o edital — e **declara o preparo do art. 53**
     (cinco itens, todos obrigatórios; o banco também confere);
   - o **instrumento descritivo** é gerado pelo sistema a partir das
     unidades (código, assunto, datas-limite, quantificação, acesso), com a
     história administrativa do conjunto informada pela origem;
   - condições da unidade (`PodeRecolher`): nem recolhida nem eliminada;
     destinação guarda permanente (vale a gravada na unidade quando ela já é
     permanente, como a amostra); no regular, no intermediário com o prazo
     cumprido ou permanente no órgão; nos especiais, em qualquer fase. Fora
     de empréstimo, de guia de transferência e de outro recolhimento;
   - situações `ENVIADO → RECEBIDO | DEVOLVIDO | CANCELADO`. A origem cancela;
     o Arquivo Público **devolve** com o que falta ou **registra a entrada**.
4. **Entrada** (só `acervo:central`): cada unidade é conferida de novo,
   passa à fase permanente sob a guarda do Arquivo, no local informado, e o
   recolhimento recebe o **número do registro de entrada** (`RE-AAAA-NNNNN`).
   Na mesma transação, o aviso "registro de entrada" é gravado para quem
   gere o acervo da origem por lotação (na unidade, numa unidade acima ou na
   entidade) e para quem enviou — é a **cópia do registro** do art. 5º, VII
   —, entregue em tempo real depois do commit (ADR 027). A devolução também
   avisa a origem.
5. A ficha do recolhimento é o documento para imprimir e assinar: antes da
   entrada, o instrumento descritivo; depois, o registro de entrada.
6. Correção da Fase 3 descoberta aqui: a permissão `acervo:avaliacao` não
   estava no manifesto do plug-in e por isso não aparecia no catálogo de
   permissões dos perfis. Entrou, junto com a descrição atualizada de
   `acervo:central`.

## Consequências

- O painel do Acervo ganha a situação "No Arquivo Público"; "Permanente no
  órgão" passa a indicar o que ainda pode ser recolhido.
- O perfil do Arquivo Público precisa de `acervo:central` **e**
  `acervo:manage` (as rotas de gestão da unidade exigem a segunda).
- O cronograma do Regimento Interno (art. 47) não é imposto pelo sistema
  enquanto o regimento não existir.
- Ficam para a segunda parte da Fase 4: os **arquivos privados de interesse
  público e social** (arts. 55 a 58: cadastro, parecer da Comissão Especial,
  decreto, doação ou depósito, direito de preferência de 60 dias) e as
  **instruções normativas** do Arquivo (art. 54) publicadas no sistema. A
  transferência à instituição sucessora na extinção (art. 17, parte final)
  também.
