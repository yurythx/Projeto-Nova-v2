# 036 — Acervo, Fase 4 (segunda parte): instruções normativas versionadas e arquivos privados

- **Status:** **Superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (módulo removido do código). Antes: Aceito e implementado (conclui a Fase 4 do [roadmap](../ROADMAP_SISMARQ.md), salvo o que está em Consequências)
- **Data:** 2026-10-16
- **Relacionado:** [ADR 035](035-acervo-recolhimento.md) (recolhimento), [ADR 026](026-tramite-e-atlas-publico.md) (consulta pública sem login)

## Contexto

Dois temas da Lei Municipal 10.301/2019 completam a guarda permanente:

- **Instruções normativas** (art. 54): o Arquivo Público publica as normas
  que os órgãos seguem. Elas **mudam com o tempo** — prazos, cronograma do
  Regimento (art. 47), procedimentos. Quem consulta precisa do texto em vigor
  e, numa auditoria ou num processo, do texto que valia numa data passada.
- **Arquivos privados de interesse público e social** (arts. 55 a 58): o
  acervo de uma pessoa física ou jurídica relevante para a história, a
  cultura ou o desenvolvimento do município é declarado por decreto,
  precedido do parecer de uma Comissão Especial (art. 56, § 1º). A declaração
  não transfere a guarda (§ 4º); o titular pode doar ou depositar a título
  revogável (§ 5º); o acesso público depende da autorização do titular
  (§ 2º); a entidade privada de serviço público municipal já é de interesse
  por força da lei (§ 3º); na venda, o município tem preferência por 60 dias
  (art. 58).

## Decisão

### Instruções normativas — cada alteração é uma nova redação

1. `acervo_normas` (número `N/AAAA`, situação `RASCUNHO`, `VIGENTE` ou
   `REVOGADA`) e `acervo_norma_redacoes` (versão, ementa, texto em
   Markdown, **motivo** — "Redação original" ou o ato que altera —, data e
   Diário Oficial da publicação).
2. **A redação publicada nunca muda nem é apagada**: um gatilho no banco
   recusa `UPDATE` e `DELETE` nela. Alterar a norma é abrir uma nova redação
   em rascunho (no máximo uma por norma, por índice único), que só vale
   depois de publicada. A data de publicação não pode ser futura nem
   anterior à da redação em vigor.
3. Cada redação publicada **vigora até a publicação da seguinte**; a última,
   até a revogação ou até hoje. O sistema calcula esse período e o mostra —
   é o "texto compilado" das leis.
4. **Revogar** encerra a vigência com o ato e a data, descarta um rascunho
   pendente e mantém todo o texto consultável. Descartar o rascunho de uma
   norma nunca publicada apaga a norma (nada dela chegou ao público).
5. **Consulta pública sem login** (`/api/v1/acervo/publico/normas`, página
   `/normas`): só as normas já publicadas, com a redação em vigor e as
   anteriores, cada uma com o período e o motivo; busca no número, na ementa
   e no texto publicado. O rascunho nunca aparece. O plug-in Acervo passa a
   ser público no Kernel (aparece no menu do site e no sitemap) e a
   allowlist do proxy anônimo libera só o prefixo `v1/acervo/publico/`.
6. Só o Arquivo Público (`acervo:central`, global) escreve.

### Arquivos privados

7. `acervo_privados` (número `AP-AAAA-NNNNN`) com titular, descrição e
   relevância, datas-limite, mensuração, local, acesso autorizado,
   convênio (art. 57), e as etapas:
   `CADASTRADO → COM_PARECER → DECLARADO`, ou `NAO_DECLARADO` com parecer
   contrário. A entidade de serviço público (`servico_publico`, só pessoa
   jurídica) nasce `DECLARADO` (§ 3º). CHECKs no banco exigem o parecer
   favorável e o decreto antes da declaração, salvo o § 3º.
8. **Guarda**: `TITULAR`, `DEPOSITADO` (revogável — volta ao titular) ou
   `DOADO` (definitiva), sempre com termo e data, só no declarado.
9. **Direito de preferência** (`acervo_privado_alienacoes`): a notificação
   do titular abre 60 dias; a manifestação (interesse ou não, com
   fundamento) só vale dentro do prazo; vencido, a preferência caducou e o
   sistema mostra isso. Uma notificação pendente por vez.
10. Histórico próprio (`acervo_privado_eventos`) de cada passo, além da
    auditoria e do evento de outbox — sem os dados pessoais no evento.
11. Leitura e escrita **só do Arquivo Público**: o cadastro tem CPF/CNPJ e
    contato do titular (LGPD).

### Correções de passagem

12. O cliente da API tratava qualquer resposta sem corpo como "resposta
    ilegível". O 204 agora é sucesso — corrigiu também o "marcar como lida"
    do Mercúrio, que respondia 204 e nunca atualizava a tela.
13. `/acervo` entrou nas rotas protegidas do cliente: a sessão expirada nas
    telas do Acervo volta ao login.

## Consequências

- O Arquivo Público pode publicar o Regimento Interno e o cronograma de
  recolhimento (art. 47) como instruções normativas e mudá-los com o tempo
  sem perder o que valia antes.
- Não há anexos (PDF) nas normas: o texto é Markdown. Se o Diário Oficial
  for a fonte oficial, o número e a página ficam registrados em cada
  redação.
- A transferência do acervo à instituição sucessora na extinção (art. 17,
  parte final) e a assinatura digital das normas no Signum ficam para depois.
