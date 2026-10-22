# 038 — Acervo, Fase 6: acesso, busca e difusão

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 6 do [roadmap](../ROADMAP_SISMARQ.md))
- **Data:** 2026-10-16
- **Relacionado:** [ADR 031](031-sismarq-e-typesense.md) (Typesense só com conteúdo público), [ADR 032](032-acervo-unidades-de-arquivamento.md) (nível de acesso), [ADR 037](037-acervo-digitalizacao.md) (texto do OCR)

## Contexto

A Lei 10.301/2019 garante o acesso (arts. 2º, 5º X/XI, 7º I) e trata o
sigilo como exceção. O artigo científico propõe consultar o Typesense
**direto do navegador**, com `tenant_id`. Isso conflita com o ADR 031
(índice só do que é público) e com o acervo, que tem restrito, sigiloso e
dados pessoais. A seção 6 do roadmap pedia esta decisão antes de indexar.

## Decisão

1. **Busca só pela API.** O navegador não fala com o Typesense. A API
   aplica o filtro: consulta anônima = só ostensivo; autenticada = unidades
   do escopo, com a faceta `nivel_acesso`. Sem `TYPESENSE_URL`, a busca
   usa o `tsvector` do Postgres (texto do OCR, sem facetas).
2. **Duas coleções no índice**, nunca uma chave “filtro embutido” no
   browser: pública (ostensivo) e interna. A recarga é pelo worker, a
   partir da fila `acervo_indice_pendentes` (gatilhos no banco cobrem o
   recálculo diário e o SQL).
3. **Publicação do representante** (`publicado_em`) só se estiver `PRONTO`
   e a unidade for `PUBLICO` e não eliminada. O banco recusa o resto
   (gatilho). Mudança de nível ou eliminação despublica na hora.
4. **Consulta pública** sem login: ficha da unidade (sem localização, sem
   observações, sem quem cadastrou), PDF com HTTP 206 quando liberado, e
   o pedido do que não é ostensivo pelo SIC (série `2.0.03.02`).
5. **Exposições virtuais** (art. 5º, X e XI): o Arquivo Público (`acervo:central`)
   monta, publica e recolhe. A visita pública só vê documentos já
   liberados.
6. **Indicador SEMAD:** contadores diários de busca, ficha, documento e
   exposição (`acervo_consultas`).
7. **Sem `tenant_id`:** um município por implantação (ADR 015).

## Consequências

- Páginas `/arquivo`, `/arquivo/busca`, `/arquivo/unidades/{id}`,
  `/arquivo/exposicoes/{slug}` e, autenticadas, `/acervo/busca` e
  `/acervo/exposicoes`.
- O proxy anônimo libera `v1/acervo/publico/*` (com `Range` no PDF).
- A ponte com o Trâmite é o [ADR 039](039-acervo-ingresso-do-tramite.md).
- PAdES ICP-Brasil continua o [ADR 040](040-pades-icp-brasil-adiado.md).
