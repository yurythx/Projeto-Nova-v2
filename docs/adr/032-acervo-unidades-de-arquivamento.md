# 032 — Acervo: unidades de arquivamento, prazos de guarda e as travas da lei

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 1 do [roadmap](../ROADMAP_SISMARQ.md)); o critério de contagem dos prazos aguarda a validação da Divisão de Arquivo
- **Data:** 2026-10-15
- **Relacionado:** [ADR 013](013-permissao-com-escopo-e-heranca.md) (escopo), [ADR 017](017-ttdd-oficial.md) e [019](019-vigencia-da-ttdd-e-versoes.md) (TTDD), [ADR 031](031-sismarq-e-typesense.md)

## Contexto

O SISMARQ existe para gerir os documentos da administração municipal nas
três idades (Lei 10.301/2019, art. 18) até a eliminação ou a guarda
permanente. A plataforma já tinha a TTDD oficial no Atlas, mas nada que
representasse o documento físico: a caixa, o livro, o processo. A Fase 1 do
roadmap pede esse núcleo, com as travas da lei no domínio.

## Decisão

1. **Plug-in Acervo** (`internal/modules/acervo`):
   - depende do Atlas (`DependsOn: atlas`);
   - lê a TTDD por uma porta própria (`domain.TTDD`), ligada ao Atlas em
     `internal/app` (`ttddPort`). Os plug-ins continuam sem se importar.
2. **Unidade de arquivamento** é o que os Anexos I a III listam. Ela tem:
   - tipo: caixa, pasta, livro, processo, dossiê, documento, mapa ou outro;
   - código de classificação (série da TTDD);
   - assunto, unidade produtora (dona, para o escopo) e datas-limite;
   - quantificação e especificação, observações;
   - fase, destinação e nível de acesso com hipótese legal;
   - localização física.

   O número `UA-AAAA-NNNNNN` é gerado pelo banco.
3. **Destinação** vem da série no cadastro e na reclassificação (art. 28).
   - Série sem destinação na TTDD (6 casos) fica `INDEFINIDA`: nada sai do
     intermediário até a Comissão Central defini-la.
   - Série revogada não recebe cadastro. Unidade já classificada numa série
     revogada fica `SERIE_REVOGADA` até ser reclassificada (art. 52).
4. **Prazos (arts. 29 e 30).**
   - A fase **corrente** conta da **data-limite final** da unidade.
   - A **intermediária** conta do fim da corrente.
   - Condições:
     - "Enquanto estiver vigorando" (727 séries) termina na **data do
       evento** (fim da vigência), informada no cadastro;
     - "NN meses" e "NN dias" contam do início da fase;
     - "NN dias após a data do evento" conta do evento;
     - condição que o sistema não interpreta vale a data do evento.
   - Sem a data, a unidade fica `AGUARDA_EVENTO`.
   - **Este critério de contagem é uma premissa** (roadmap, seção 6, item
     5): a Divisão de Arquivo deve confirmá-lo numa amostra antes da Fase 3.
5. **Resultado gravado.** O fim de cada fase e a **situação** (no corrente,
   transferir, no intermediário, avaliar eliminação, recolher, aguardando
   evento ou destinação, bloqueada, permanente, eliminada) ficam na tabela,
   para filtrar e contar sem refazer a conta. Eles são recalculados:
   - a cada escrita;
   - diariamente (worker `prazos`);
   - quando a TTDD muda pela tela (novo evento `atlas.ttdd.carga.aplicada`,
     fila `sismarq.acervo.ttdd`).
6. **Travas da lei**, no domínio (`PodeTransferir`, `PodeEliminar`) e
   **também no banco** (gatilho `acervo_travas`). Nenhuma permissão as
   contorna:
   - permanente nunca muda de fase (arts. 20 e 42);
   - eliminada não volta nem é reclassificada (art. 38);
   - só se elimina a série destinada à eliminação (art. 19);
   - nada é eliminado com bloqueio judicial ativo (art. 44);
   - unidade de arquivamento nunca é apagada.
7. **Transferência** do corrente ao intermediário só com o prazo corrente
   cumprido (art. 29). **Bloqueio judicial** com o número do processo e
   liberação com motivo (arts. 32, II, e 44).
8. **Permissões:**
   - `acervo:read`, `acervo:manage` (agente setorial, art. 9º) e
     `acervo:bloqueio` valem **no escopo** (ADR 013): a secretaria cuida do
     que é dela, e fora do escopo a leitura é 404 e a ação é 403;
   - `acervo:central` (os locais físicos do Arquivo Público, órgão central)
     só com concessão global.
9. Toda escrita grava, na mesma transação:
   - a unidade;
   - o **histórico** (`acervo_movimentacoes`);
   - o **evento** no outbox (`acervo.unidade.*`);
   - a **auditoria**.

## Consequências

- `PodeEliminar` já existe e está testado. A esteira de eliminação (Fase 3)
  só o chama e acrescenta o rito: relação, ata, edital, amostragem e termo.
- O recolhimento à guarda permanente (Fase 4) usa a situação `RECOLHER`.
- O nível de acesso já é gravado. A consulta pública e a busca do acervo
  (Fase 6) partem dele.
- A carga da TTDD pelo `make ttdd-aplicar` (SQL direto) não emite o evento.
  O recálculo diário cobre esse caminho.
