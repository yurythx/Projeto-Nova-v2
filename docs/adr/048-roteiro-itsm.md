# 048 — O que a gestão de serviços copia do mercado

- **Status:** Aceito. As três levas e o que ficou para depois estão implementados. O que não se constrói continua no [roteiro](../ROADMAP_GESTAO_DE_SERVICOS.md).
- **Data:** 2026-10-23
- **Relacionado:** [ADR 031](031-typesense-na-busca-global.md) (índice só de conteúdo público), [ADR 027](027-notificacoes.md) (aviso)

## Contexto

A comparação com o GLPI (referência do setor público) e com o ServiceNow
(referência de mercado) apontou lacunas. Copiar a plataforma de
analytics ou o atendente virtual do ServiceNow, ou publicar chamado no
índice público, pioraria o produto. Não há correio na plataforma: um
coletor de e-mail sem porta de envio seria um atalho.

## Decisão

1. **Não se constrói** a plataforma de analytics nem o atendente virtual.
2. **Chamado na busca** entra pela Busca Global, com a mesma visibilidade
   de quem abriu e de quem atende. O módulo não implementa
   `search.Indexer`. O trecho do resultado não leva a descrição.
3. **Artigo antes do chamado** usa a wiki já publicada, pela busca do
   módulo `wiki`. A lista fixa de tutoriais no navegador sai.
4. **Escala do prazo** aos 90%, nas duas horas finais e no vencimento
   avisa a equipe da unidade junto com o responsável. O responsável não
   muda: trocar sozinho tira o chamado de quem já atende.
5. **Correio, contrato, licença, fornecedor, relação entre ativos,
   campos por oferta, nota por e-mail, nota numérica da mudança e janela
   de congelamento** estão na [ADR 049](049-correio-contratos-e-cadastros.md).
6. **Portal do solicitante, série histórica e agente de inventário**
   estão na [ADR 050](050-portal-serie-e-agente.md).

## Consequências

A Busca Global passa a consultar o atendimento quando o módulo está
ligado. Quem não abriu e não atende a unidade não vê o chamado. O índice
Typesense continua sem dado de solicitação.
