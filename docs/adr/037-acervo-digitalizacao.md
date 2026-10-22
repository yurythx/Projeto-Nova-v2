# 037 — Acervo, Fase 5: digitalização com valor probatório e preservação digital

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (Fase 5 do [roadmap](../ROADMAP_SISMARQ.md), salvo o que está em Consequências)
- **Data:** 2026-10-16
- **Relacionado:** [ADR 032](032-acervo-unidades-de-arquivamento.md) (a unidade de arquivamento), ADR da cópia WORM da auditoria (object lock), [ADR 027](027-notificacoes.md) (avisos)

## Contexto

O Decreto Federal 10.278/2020 dá ao documento digitalizado o mesmo valor
do original em papel quando ele cumpre requisitos técnicos. A Lei
Municipal 10.301/2019 aplica as suas regras aos documentos digitais e
digitalizados (art. 48). O artigo técnico em `docs/SISMARQ` descreve o
pipeline (itens 4.1 e 4.3; Quadro 2, módulo SISMARQ_PIPELINE):

- PDF/A (ISO 19005), com OCR, alinhamento (*deskew*) e limpeza;
- resolução mínima: 300 DPI para texto e 600 DPI para plantas;
- metadados do art. 9º do Decreto no XMP e no banco (identificador,
  responsável, data e hora em UTC, equipamento, resolução, hash, código de
  classificação);
- SHA-256, guarda imutável e verificação periódica de integridade
  (*fixity*);
- visualização em fluxo, com meta de primeira página em menos de 300 ms.

## Decisão

1. **Entrada pela tela, ligada à unidade de arquivamento**. O PDF do
   scanner vai do navegador direto ao MinIO por URL pré-assinada (o mesmo
   padrão dos anexos), até 2 GB (`ACERVO_DIGITAL_MAX_MB`). O registro
   guarda o SHA-256 do original e os metadados do Decreto informados:
   equipamento, resolução declarada, tipo (texto ou planta), parte/volume e
   o responsável — usuário do sistema e **CPF só mascarado**
   (`***.456.789-**`, minimização da LGPD). Só quem gere a unidade envia
   (depois de recolhida, o Arquivo Público); a eliminada não se digitaliza.
2. **Pipeline no worker** (`acervo_digitais`, fila com `FOR UPDATE SKIP
   LOCKED`; o que ficar `PROCESSANDO` por mais de 3 h volta à fila):
   - `pdfinfo` e `pdfimages` medem as páginas e a **menor resolução** das
     imagens do original. Abaixo do mínimo do tipo (`ACERVO_DPI_TEXTO`,
     `ACERVO_DPI_PLANTA`) ou sem imagens (PDF nato-digital), o envio é
     **reprovado** — é preciso digitalizar de novo;
   - `ocrmypdf --output-type pdfa-2 --deskew --clean --skip-text -l por`
     gera o **PDF/A-2b** com OCR em português;
   - um script com **pikepdf** grava no XMP os metadados do Decreto, no
     namespace do artigo (`govbr:`), **com o schema de extensão
     (`pdfaExtension`)** que o PDF/A exige para propriedades próprias — sem
     ele o arquivo deixaria de ser PDF/A válido — e salva **linearizado**;
   - `pdftotext` extrai o texto (tabela `acervo_digital_textos`, para a
     busca da Fase 6);
   - confere que o PDF/A tem as mesmas páginas do original;
   - calcula o SHA-256 do PDF/A e grava no bucket dedicado
     (`ACERVO_DIGITAL_BUCKET`) com **object lock em modo Compliance**:
     permanente (ou série sem destinação definida) por 100 anos; destinação
     eliminação, até um ano depois do fim do prazo intermediário (o
     representante segue a TTDD do papel), no mínimo um ano;
   - apaga o original (o PDF/A passa a ser a cópia oficial); falha do
     pipeline volta à fila e, na 3ª tentativa, fica `FALHOU` com o original
     guardado para o Arquivo Público reprocessar.
   O hash gravado no XMP é o do **original**: o do PDF/A muda ao gravar o
   próprio XMP e fica no banco.
3. **O pronto é imutável no banco**: gatilho impede apagar o
   representante e mudar hash, objeto ou reduzir a retenção.
4. **Integridade (fixity)**: um worker relê cada PDF/A guardado (padrão
   uma vez por dia, `ACERVO_FIXITY_INTERVALO`) e compara o SHA-256. Cada
   verificação fica registrada (`acervo_digital_verificacoes`); se o hash
   não bater ou o arquivo sumir, a falha é auditada, sai no outbox e a
   equipe com `acervo:central` (concessão global) é avisada. O Arquivo
   Público também verifica na hora e tem o painel de falhas.
5. **Visualizador em fluxo**: `GET /acervo/digitais/{id}/arquivo` atende
   `Range` (HTTP 206, `Content-Range`, `ETag` = SHA-256, 416 fora do
   tamanho) direto do MinIO (`storage.RangeReader`); o proxy do frontend
   passa a repassar `Range`/`If-Range` e os cabeçalhos do 206. O pdf.js
   (`pdfjs-dist` 5.4.296) pede trechos de 64 KB, sem baixar o arquivo, e o
   `PDFViewer` renderiza só as páginas visíveis; a tela mostra o tempo até
   a primeira página.
6. **Art. 42 no domínio**: digitalizar não muda a fase nem a destinação da
   unidade — `PodeEliminar` segue recusando o permanente.
7. **A imagem do worker** passa a ter `ocrmypdf`, `tesseract-ocr-por`,
   `unpaper`, `ghostscript`, `poppler-utils` e `python3-pikepdf`. O teste
   `TestPipelineReal` roda o pipeline de verdade nessa imagem (`make
   pipeline-real`, também no CI): gera uma página "escaneada" a 300 DPI e
   torta, e confere o OCR, o PDF/A-2b, os metadados do Decreto no XMP, o
   schema de extensão e a linearização (≈1,5 s por página).

### Correções de passagem

8. `golang.org/x/net` 0.58 → 0.60 (quatro vulnerabilidades alcançáveis no
   govulncheck) e `next` 16.3.6 → 16.3.8 (seis alertas, um alto). O `npm
   audit` bloqueante do CI passa a olhar as dependências de produção; o
   relatório completo continua, informativo, porque o `braces` 3.0.3 (via
   `eslint-config-next`, só no lint) não tem versão corrigida publicada.

## Consequências

- **Assinatura ICP-Brasil (PAdES)** do representante: [ADR 040](040-pades-icp-brasil-adiado.md)
  — adiado até a decisão 0.6; o JSON declara `pades=AUSENTE`.
- **Pasta monitorada (SMB/FTP)**: fica para quando houver o scanner de
  produção definido (decisão 0.6); a entrada hoje é pela tela.
- **SCAN vs nato-digital e TIFF:** [ADR 044](044-ingresso-scan-nato-digital.md)
  — PDF sem imagens deixa de ser reprovado quando a origem é
  `NATO_DIGITAL`; TIFF entra no mesmo pipeline.
- A conformidade PDF/A é a gerada pelo ocrmypdf/Ghostscript; uma validação
  formal (veraPDF) pode entrar no pipeline depois.
- Eliminar o papel de uma série de eliminação não apaga sozinho o
  representante: o object lock segura o PDF/A até a retenção; a remoção
  acompanha o Termo de Eliminação numa fase seguinte.
- A busca no texto do OCR (com o controle de acesso do documento) é a
  Fase 6 ([ADR 038](038-acervo-acesso-busca-e-difusao.md)).
