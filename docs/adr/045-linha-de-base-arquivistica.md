# 045 — Linha de base do domínio arquivístico e do espelho federal

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito
- **Data:** 2026-10-18
- **Relacionado:** [ROADMAP_SISMARQ](../ROADMAP_SISMARQ.md), ADRs [017](017-ttdd-oficial.md)–[044](044-ingresso-scan-nato-digital.md), [DOMINIO.md](../acervo/DOMINIO.md), [EARQ_RDC.md](../acervo/EARQ_RDC.md), [TTDD_API.md](../acervo/TTDD_API.md)

## Contexto

As Fases 1 a 6 do acervo e o Atlas (TTDD e procedimentos) estão no código.
Quem for alterar digitalização, busca, prazos, eliminação ou classificação
precisa do mesmo mapa: o que a Lei 10.301 trava, o que o Decreto 10.278
exige, o que o CONARQ chama de SIGAD e de RDC-Arq, e o que o município
ainda não tem (certificado, SEI, portarias, RIPD).

Sem essa linha de base, um PR tende a misturar plug-ins, inventar PAdES,
expor o Typesense no navegador ou “completar” a TTDD com códigos que a
CCPAD não publicou.

## Decisão

1. **Documento de domínio.** A análise viva está em
   [`docs/acervo/DOMINIO.md`](../acervo/DOMINIO.md). Mudança de regra de
   negócio do Acervo, da TTDD ou dos procedimentos **cita esse texto e o
   ADR da fase**; se divergir, atualiza o domínio no mesmo commit.
2. **Espelho federal.** O checklist de trabalho rumo ao e-ARQ Brasil v2
   (Resolução CONARQ 50/2022, art. 13 da lei municipal) e ao RDC-Arq
   (Resolução CONARQ 51/2023) está em
   [`docs/acervo/EARQ_RDC.md`](../acervo/EARQ_RDC.md). Não é certificação
   CONARQ: é o mapa para a Fase 8 e para PRs de preservação.
3. **Papéis dos plug-ins.** Atlas = norma (TTDD, fluxos). Acervo = documento
   de arquivo. Trâmite = corrente do que não passa pelo SEI. Signum =
   assinatura eletrônica (não ICP). Arquivos = pasta de trabalho, não
   repositório permanente. Colaboração só por porta em `internal/app`.
4. **Ordem do trabalho que ainda depende de gente.** Homologação C1–C4 e
   C6 na Superintendência/SEMAD; carga CONARQ 01–07 só com tabela oficial;
   PAdES só com certificado (ADR 040); SEI só com API (ADR 042). Código
   novo **não** afirma conformidade que esses itens não fecharam.
5. **SIGAD antes de RDC certificado.** O SISMARQ opera como SIGAD
   municipal com guarda reforçada (PDF/A, hash, object lock, fixity). Pacote
   AIP, veraPDF, PREMIS e autoavaliação ISO 16363 entram por ADR próprio,
   depois da linha de base e sem fingir repositório certificado.

## Consequências

- PRs de acervo/Atlas/TTDD passam a apontar `docs/acervo/` no template
  quando tocam regra de negócio.
- A coluna “Hoje” do e-ARQ no roadmap deixa de ser só “Falta”: a lacuna
  está inventariada; a certificação continua Fase 8.
- Homologação C1–C4 e atos (arts. 54, 64) continuam fora do git.
