# 041 — Ritos dos arts. 35 e 36, cópia do Termo (art. 38) e Signum nas atas

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (ritos e cópia); Signum nas atas adiado de propósito
- **Data:** 2026-10-16
- **Relacionado:** [ADR 034](034-acervo-eliminacao.md), [ADR 040](040-pades-icp-brasil-adiado.md)

## Contexto

O rito ordinário (Anexos I a IV) não cobre:

- documento que **não consta** da TTDD (art. 35): CCPAD + Gerente da
  Divisão + SEMAD;
- órgão **sem TTDD de atividades-fim** oficializada (art. 36): a Relação
  vai à CCPAD e ao Gerente, depois à SEMAD;
- a **cópia do Termo** à Superintendência de Arquivo Público e à SEMAD
  (art. 38, parágrafo único);
- a **assinatura da ata e do termo no Signum** (artigo, item 4.6).

Fingir ICP-Brasil ou um Signum “já assinado” seria o mesmo erro do ADR
040. A permissão `acervo:semad` não inventa a lotação: o IAM (Keycloak /
LDAP) mapeia o grupo da SEMAD para um perfil com essa permissão **global**
(ADR 043). Sem destinatário, `termo_copia_semad` fica 0.

## Decisão

1. **`rito` na Relação:** `ORDINARIO` (padrão), `ART_35` ou `ART_36`.
2. **Art. 35:** só entra o que `PodeEliminar` recusaria por não estar
   previsto na TTDD (série inexistente ou destinação indefinida). Guarda
   permanente e prazo da série já destinada à eliminação continuam
   travados. Depois da ata da CSPAD: membro da CCPAD (`autorizar-ccpad`)
   → Arquivo Público (`autorizar`) → SEMAD (`autorizar-semad`).
3. **Art. 36:** a Relação usa a CCPAD ativa; depois do Arquivo Público
   falta a SEMAD. Não detectamos sozinhos quais órgãos estão no 0.5 —
   quem abre a Relação declara o rito.
4. **Cópia do Termo:** na mesma transação do registro, aviso na caixa de
   quem tem `acervo:central` e `acervo:semad` globais. Zero destinatários
   SEMAD não bloqueia o Termo: a permissão ainda não foi concedida. O
   JSON grava `termo_copia_arquivo` e `termo_copia_semad`.
5. **Signum:** `ata_signum` e `termo_signum` valem `AUSENTE` (CHECK no
   banco). Quando a 0.6 existir, um ADR liga o Signum; até lá o JSON
   declara a ausência.

## Consequências

- Situação nova `AGUARDANDO_SEMAD`. O edital só abre em `AUTORIZADA`.
- Homologação com a Divisão passa a incluir um piloto art. 35 ou 36 e a
  conferência das cópias na caixa.
- Aprovação da TTDD de atividades-fim (art. 32, §§ 4º a 8º) continua
  fora: é outro processo, no Atlas.
