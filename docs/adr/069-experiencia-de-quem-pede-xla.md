# 069 — Experiência de quem pede (XLA)

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 047](047-avaliacao-governanca-e-cascata.md) (avaliação e governança), [ADR 068](068-aprovacao-em-etapas-e-tarefas.md)

## Contexto

Os indicadores da central (MTTR, MTBF, prazo cumprido, primeiro contato) medem o processo: se a equipe fez o combinado. O ITIL 4 e o ITIL 5 pedem também os acordos de experiência (XLA): como foi para a pessoa. Um chamado pode cumprir o prazo e, mesmo assim, ter feito o servidor responder três vezes, reabrir o pedido e esperar dias de relógio, porque o prazo conta só o expediente e para enquanto se espera por ele. A nota de satisfação sozinha não mostra isso, e pouca gente responde.

## Decisão

1. **Facilidade.** A avaliação ganha uma pergunta opcional ao lado da nota: "Foi fácil conseguir o que você precisava?", de 1 a 5. É o índice de esforço (CES), na escala da nota. As avaliações antigas ficam sem resposta e não entram na média.
2. **Medidas tiradas do histórico**, sem perguntar nada a ninguém, sobre os pedidos concluídos do escopo de quem consulta:
   - **Reabertura:** % dos concluídos com histórico que voltaram de concluído para atendimento.
   - **Idas e voltas:** quantas vezes, em média, o pedido foi para "aguardando o servidor".
   - **Espera:** mediana, em horas corridas, da abertura à conclusão. É o relógio de quem pediu: inclui noite, fim de semana e o tempo de aprovação.
3. **Metas da instituição.** Uma linha única (`atendimento_xla`) com nota mínima, facilidade mínima, reabertura máxima, idas e voltas máximas e espera máxima. Meta nula = medida sem meta. Só a gestão do atendimento em toda a instituição as define, e cada troca vai para a auditoria. Quem atende lê.
4. **Sem amostra, sem afirmação.** Cada medida sai com o valor, a meta, a amostra e "atingida". Sem amostra ou sem meta, "atingida" fica nula: o painel não pinta de verde o que não mediu.
5. As medidas saem junto da governança (`GET /atendimento/governanca`, campo `experiencia`), com a mesma amostra e o mesmo recorte de escopo dos outros indicadores. O painel da assistência ganha o bloco "Experiência de quem pede".

## Consequências

A central passa a enxergar o atrito que o prazo esconde: pedido devolvido para o servidor, reaberto ou parado no relógio dele. As metas são poucas e explícitas, o que evita um painel de números sem dono.

A espera inclui a aprovação de propósito: para quem pediu, esperar a chefia também é esperar. Se a central quiser separar, a aprovação tem histórico próprio (ADR 068).

A facilidade depende de resposta voluntária. Por isso a amostra aparece ao lado do valor, e a meta só é julgada quando há resposta.
