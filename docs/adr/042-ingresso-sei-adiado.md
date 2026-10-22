# 042 — Ingresso do SEI adiado até a decisão 0.7

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito (adiado de propósito; não implementar um cliente de mentira)
- **Data:** 2026-10-16
- **Relacionado:** [ADR 039](039-acervo-ingresso-do-tramite.md) (ponte do Trâmite), Fase 7 do [roadmap](../ROADMAP_SISMARQ.md)

## Contexto

A Fase 7 pede recolhimento automático do que nasce digital no SEI
(pacote AIP: documentos, metadados, hashes e trilha). A decisão 0.7
([ADR 043](043-fase0-decisoes-institucionais.md)) registrou: o envelope
fica pronto; **ainda não** há versão, API, autorização nem ambiente de
homologação. Sem isso não se liga cliente.

Inventar um cliente REST, um webhook ou um “SEI de laboratório” afirmaria
uma integração que o município não tem.

O Trâmite já ingressa por evento (ADR 039), no mesmo processo: envelope
mínimo, série da TTDD obrigatória, sem FK entre plug-ins.

## Decisão

1. **`ProcessoSEI` e `UnidadeDoSEI` descrevem o envelope** (número SEI ou
   protocolo, unidade produtora, série). A função **recusa** o ingresso
   com `ErrSEIAdiado` depois de validar o mínimo.
2. **Não há rota HTTP, consumidor de fila nem DependsOn** de um módulo
   SEI. Não há URL, token nem mapeamento de órgãos.
3. **Quando a 0.7 existir**, um ADR posterior liga um adaptador (porta
   no Acervo, cliente na composição) no mesmo espírito do Trâmite:
   evento ou pacote conferido, idempotente, hash na entrada.

## Consequências

- A API pública da TTDD (art. 45) já existe no Atlas; documentar o
  contrato com quem mantém o SEI continua na Fase 7.
- O dossiê não afirma integração com o SEI.
