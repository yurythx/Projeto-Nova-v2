# 044 — Ingresso SCAN versus nato-digital e TIFF no pipeline

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado
- **Data:** 2026-10-16
- **Relacionado:** [ADR 037](037-acervo-digitalizacao.md), [ADR 040](040-pades-icp-brasil-adiado.md), [FASE0](../acervo/FASE0.md), [DIGITALIZACAO.md](../acervo/DIGITALIZACAO.md)

## Contexto

O pipeline da Fase 5 (ADR 037) media o DPI pelas imagens do PDF e
**reprovava** o arquivo sem imagens, com a mensagem de que o Decreto
10.278 trata da digitalização do papel. Isso está certo para o scan. Está
errado para o **PDF nato-digital** (já eletrônico): ele não tem (nem deve
ter) uma imagem de scanner por página. Na prática, ofícios e PDFs de
sistema eram recusados como se fossem scan ruim.

Os scanners de produção também geram **TIFF** multipágina. O ADR 037
deixou TIFF para depois.

Não há pasta SMB/FTP homologada (Fase 0, decisão 0.6) nem certificado
ICP-Brasil (ADR 040).

## Decisão

1. **Origem obrigatória no registro** (`SCAN` | `NATO_DIGITAL`, omissão =
   `SCAN` para compatibilidade).
2. **SCAN:** PDF ou TIFF; DPI declarado 72–4800; o worker exige imagens e
   o mínimo de 300/600 DPI conforme o tipo. Abaixo disso: `REPROVADO`.
3. **NATO_DIGITAL:** PDF já eletrônico; DPI declarado gravado como 0; o
   worker **não** reprova por ausência de imagens nem por DPI. Gera
   PDF/A-2b, OCR (`--skip-text`), XMP, SHA-256 e object lock como no scan.
4. **TIFF** entra no mesmo pipeline (`ocrmypdf` aceita TIFF; `tiffinfo`
   mede páginas e resolução). A imagem do worker instala `libtiff-tools`.
5. **Entrada continua pela tela** da unidade de arquivamento. Pasta
   monitorada, veraPDF e PAdES ficam como no ADR 037/040.

## Consequências

- A ficha da unidade mostra os dois caminhos; a página inicial pública
  explica o fluxo (`#digitalizacao`).
- Homologação: `deploy/homolog/seed-digitais.sql` (metadados) e o envio
  real do arquivo pela tela, porque o PDF/A precisa do objeto no MinIO.
- Nato-digital **não** equivale a valor probatório do Decreto 10.278
  (aquele decreto é da digitalização do papel). O SISMARQ preserva o
  arquivo em PDF/A com proveniência `NATO_DIGITAL`.
