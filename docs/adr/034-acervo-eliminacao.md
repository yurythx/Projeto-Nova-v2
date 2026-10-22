# 034 — Acervo, Fase 3: comissões e o rito da eliminação (Anexos I a IV)

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 3 do [roadmap](../ROADMAP_SISMARQ.md); ritos 35/36 e cópia do termo no [ADR 041](041-acervo-ritos-35-36-signum-e-copia.md); aprovação da TTDD de atividades-fim continua fora)
- **Data:** 2026-10-15
- **Relacionado:** [ADR 032](032-acervo-unidades-de-arquivamento.md) (travas da eliminação), [ADR 033](033-acervo-emprestimo-guia-e-importacao.md) (empréstimo e guia)

## Contexto

Eliminar documento público sem o rito da lei é crime (Lei 8.159/1991, art.
25; Lei 10.301/2019, art. 66). A Lei Municipal 10.301/2019 descreve o rito
nos arts. 31 a 40 e dá os modelos dos documentos nos Anexos I a IV:

- a **Comissão Setorial Permanente de Avaliação de Documentos (CSPAD)** de
  cada órgão seleciona e aprova; a **CCPAD** coordena (art. 31). A
  composição mínima está nos §§ 6º e 7º;
- a **Relação de Eliminação** (Anexo I) precisa do parecer jurídico (art. 32,
  § 4º), da ata da CSPAD (Anexo IV) e do "Autorizo" do titular da Divisão de
  Arquivo Público (arts. 19 e 65);
- o **Edital de Ciência** (Anexo III) dá 30 dias aos interessados para
  pedir desentranhamento ou cópia (art. 37);
- uma **amostra** do que seria eliminado fica para sempre (art. 39);
- o **Termo de Eliminação** (Anexo II) fecha o processo; o papel vai à
  fragmentação ou à doação (art. 40).

As travas da Fase 1 (`PodeEliminar`: prazo cumprido, destinação eliminação,
série vigente, sem bloqueio judicial) já impediam a eliminação avulsa. A Fase
3 abre o único caminho permitido.

## Decisão

1. **Comissões** (`acervo_comissoes`, `acervo_comissao_membros`):
   - tipo `SETORIAL` (de uma entidade) ou `CENTRAL`, com nome, portaria de
     designação e situação ativa;
   - o domínio exige um único presidente e as áreas do art. 31: na CSPAD,
     arquivologia ou história, jurídico, administração ou finanças e a
     unidade produtora (§ 6º); na CCPAD, também o Arquivo Público (§ 7º).
     Número par de membros vira aviso, não erro (a lei recomenda ímpar);
   - uma comissão ativa por órgão (índice único no banco);
   - só o Arquivo Público (`acervo:central`, global) cadastra e altera.
2. **Permissão nova `acervo:avaliacao`**, escopada por unidade (ADR 013):
   quem integra a CSPAD monta a relação, registra a ata, o edital, os
   pedidos, a amostragem e o termo na sua unidade. A autorização e a
   devolução são do `acervo:central`.
3. **Relação de eliminação** (`acervo_listagens`, número `LE-AAAA-NNNNN`),
   com a máquina de estados
   `RASCUNHO → APROVADA → AUTORIZADA → EM_EDITAL → ELIMINADA` (ou
   `CANCELADA`):
   - só abre com CSPAD ativa no órgão da unidade;
   - cada unidade é **conferida de novo em toda etapa** que depende dela
     (inclusão, aprovação, autorização e termo): mesma unidade, elegível
     por `PodeEliminar`, fora de empréstimo e, na inclusão, fora de outra
     relação em andamento. Um bloqueio judicial que chega depois da ata
     trava a autorização (art. 44);
   - **aprovação (Anexo IV)** exige parecer jurídico e quórum: presidente
     presente e maioria dos membros. A ata guarda número, data e hora,
     local, manifestações e a lista de presença;
   - o Arquivo Público **autoriza** ou **devolve** com motivo (a relação
     volta a rascunho e a ata é descartada);
   - **edital**: número, data de publicação e edição do Diário. O prazo
     final é a publicação + 30 dias;
   - **pedidos** dos interessados só dentro do prazo, decididos com
     fundamentação; deferido, pode retirar a unidade da relação;
   - **amostragem** (art. 39) depois do edital: a CSPAD informa os
     critérios e o percentual; o sistema sorteia com `crypto/rand` (ao menos
     uma unidade). A amostra vira `PERMANENTE` com destinação
     `GUARDA_PERMANENTE`, e um gatilho no banco impede que ela vá a
     `ELIMINADO`;
   - **termo** (Anexo II) só depois do 30º dia, com a amostragem feita e
     sem pedido pendente. As unidades restantes passam a `ELIMINADO` com
     movimentação própria. Registra quem acompanhou e a destinação do papel
     (art. 40);
   - cancelamento possível até a amostragem.
4. **CHECKs no banco** conferem a ordem do rito (ata antes da autorização,
   autorização antes do edital, termo ≥ edital + 30 dias). Mesmo uma escrita
   direta no banco não pula etapa.
5. **Diário Oficial do Município**. O modelo do Anexo III fala em "Diário
   Oficial do Estado", mas o art. 37 manda publicar no do Município. O
   sistema segue o artigo: o texto gerado diz "Diário Oficial do
   Município" e o veículo confirmado é o **DIORONDON-e** (ADR 043).
   O titular que autoriza nos Anexos é a **Superintendência de Arquivo
   Público** (nome atual; a lei ainda fala em Divisão).
6. **Documentos**: a tela gera os Anexos I a IV preenchidos, com o texto dos
   modelos da lei, para imprimir e assinar. Na ata, a frase "aprovada por
   unanimidade" do modelo virou "aprovada pelos membros presentes": o
   sistema aceita aprovação por maioria e não afirma o que não registrou.
7. Toda etapa grava auditoria e evento (`acervo.listagem.*`,
   `acervo.comissao.salva`) na mesma transação (ADR 016).

## Consequências

- O painel "Avaliar eliminação" do Acervo alimenta a nova relação.
- Fica para depois, com o roadmap: a aprovação da TTDD de atividades-fim
  pelo sistema (art. 32, §§ 4º a 8º). Os ritos 35/36, a cópia do termo e
  o estado `AUSENTE` da ata/termo no Signum estão no ADR 041.
- A amostragem acontece durante o edital, não antes: o edital anuncia o
  conjunto relacionado e a amostra sai dele antes da eliminação.
- Os documentos gerados precisam ser conferidos pela Divisão de Arquivo
  contra os Anexos antes do uso real (critério "pronto" da Fase 3).
