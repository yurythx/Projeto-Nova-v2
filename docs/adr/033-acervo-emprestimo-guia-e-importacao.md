# 033 — Acervo, Fase 2: empréstimo, guia de transferência e importação do acervo existente

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 2 do [roadmap](../ROADMAP_SISMARQ.md))
- **Data:** 2026-10-15
- **Relacionado:** [ADR 032](032-acervo-unidades-de-arquivamento.md) (Acervo), [ADR 025](025-atlas-em-escala.md) (importação simular → aplicar)

## Contexto

A Fase 2 põe o acervo físico que já existe dentro do sistema e controla o dia
a dia dos arquivos corrente e intermediário. Os documentos de `docs/SISMARQ`
dizem como:

- **Empréstimo e consulta** são rotinas que os órgãos setoriais padronizam
  (Lei 10.301/2019, art. 12, II). A TTDD da SEMAD tem uma subfunção para
  isso: "Registro de Protocolo, Registro, Empréstimo e Devolução"
  (2.0.03.00).
- **Transferência** do corrente ao intermediário acontece cumprido o prazo
  da TTDD (art. 29). A própria TTDD classifica a "Transferência de documentos
  intermediários" como série (2.0.03.01.00). A remessa, portanto, produz um
  documento: a guia.

## Decisão

1. **Empréstimo** (`acervo_emprestimos`):
   - registra quem retira (um usuário), a finalidade, o prazo de devolução
     (até um ano), quem entregou e quem recebeu de volta;
   - cada unidade tem no máximo um empréstimo em aberto, garantido por um
     índice único no banco;
   - unidade eliminada não sai;
   - o painel conta as unidades emprestadas e as atrasadas, e há uma lista
     de empréstimos em aberto.
2. **Guia de transferência** (`acervo_guias`, número `GT-AAAA-NNNNN`):
   - relaciona até 500 unidades de uma origem. Cada uma precisa estar no
     corrente, com o prazo corrente cumprido, fora de empréstimo e de outra
     guia aguardando recebimento;
   - as unidades **só mudam de fase no aceite** ("Registrar recebimento"),
     quando são conferidas de novo;
   - guia ainda não recebida pode ser cancelada;
   - a guia é impressa para assinatura.
3. **A transferência avulsa** (`POST /unidades/{id}/transferir`) passa a
   gerar a guia de uma unidade e a receber na mesma transação. Toda
   transferência deixa o seu documento.
4. **Importação do acervo existente** (`/acervo/importacao/simular` e
   `/aplicar`):
   - aceita o CSV que o Excel exporta, com `;` ou `,`, datas em `dd/mm/aaaa`
     e até 5.000 linhas;
   - a unidade produtora vem pela sigla e o local pelo caminho da tela de
     locais;
   - cada linha passa pelas **mesmas regras do cadastro** (mesma função
     `criarTx`);
   - **tudo ou nada**: só a planilha simulada (mesmo hash) e sem nenhum erro
     é gravada. Para dados com efeito legal, não se grava metade de uma
     planilha;
   - a planilha-modelo é servida pela API.
5. **Etiqueta** com QR para a lombada da caixa ou a capa do livro, que abre a
   ficha no sistema.
6. As movimentações passam a usar `clock_timestamp()`. Várias na mesma
   transação (relação na guia e transferência) ficam em ordem no histórico.

## Consequências

- A situação `TRANSFERIR` alimenta a tela de nova guia.
- A ponte com o Trâmite está no [ADR 039](039-acervo-ingresso-do-tramite.md):
  o processo encerrado e classificado vira unidade de arquivamento. O SEI
  (decisão 0.7) entra depois pelo mesmo padrão de evento.
- Cobrança automática de empréstimo atrasado, por notificação, também fica
  para depois. Hoje o atraso aparece no painel e na lista.
