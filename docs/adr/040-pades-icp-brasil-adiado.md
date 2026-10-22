# 040 — PAdES ICP-Brasil adiado até a política municipal (decisão 0.6)

- **Status:** **Parcialmente superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (a parte do Atlas/Acervo saiu do código). Antes: Aceito (adiado de propósito; não implementar um certificado de mentira)
- **Data:** 2026-10-16
- **Relacionado:** [ADR 037](037-acervo-digitalizacao.md) (pipeline PDF/A), Signum (assinatura eletrônica com reautenticação)

## Contexto

O Decreto 10.278/2020 e o artigo científico pedem assinatura ICP-Brasil
(PAdES) no representante digital. O Signum hoje é assinatura **eletrônica**
com reautenticação e SHA-256, sem certificado A1/A3. A decisão 0.6
([ADR 043](043-fase0-decisoes-institucionais.md)) registrou: quem
digitaliza é o **Arquivo Municipal**; pasta SMB/FTP ainda será definida;
**não há** certificado A1/A3/HSM. O padrão a seguir, quando o certificado
existir, é o Decreto 10.278 — não o Signum atual.

Implementar um “PAdES” com chave local, autoassinada ou com o Signum
atual **fingiria** valor de prova que o município não tem. Isso é pior do
que declarar a ausência.

## Decisão

1. **Não assinar com ICP-Brasil enquanto não houver certificado municipal.**
   O pipeline da Fase 5 permanece PDF/A-2b + OCR + XMP + SHA-256 + object
   lock + fixity. Isso já dá integridade e metadados do Decreto; não dá
   autoria ICP-Brasil.
2. **O JSON declara a ausência.** O campo `pades` do representante vale
   `AUSENTE`. A tela autenticada mostra que o certificado ainda não
   aplica. Nada no disco muda. `AfirmaAssinaturaICP` permanece falso.
3. **O Signum não vira PAdES.** Continua nos envelopes de processo. Ata e
   termo de eliminação também declaram `AUSENTE` (ADR 041).
4. **A porta `AssinadorICP` existe no domínio e não é composta** em
   `internal/app`. Quando o certificado existir, um ADR liga o adaptador,
   assina o PDF/A **depois** do pipeline (o hash muda; o banco e o XMP
   acompanham) e grava o dicionário PAdES.

## Consequências

- O valor probatório **completo** do Decreto 10.278 fica pendente do
  certificado. O dossiê e o VPAT não afirmam PAdES.
- Pasta monitorada SMB/FTP espera a definição operacional (0.6: “ainda
  será decidido como”).
- Testes travam qualquer regressão que passe a afirmar ICP sem política.
