# 027 — Notificações persistentes

- **Status:** Aceito e implementado
- **Data:** 2026-10-04
- **Relacionado:** [ADR 004](004-outbox-listen-notify-e-rbac-modular.md) (outbox e barramento)

## Contexto

Os módulos precisam avisar pessoas do que muda (hoje, o atendimento avisa
sobre a solicitação e sobre o prazo perto de vencer). O sino só
guardaria, em memória, os toasts da sessão: recarregar a página apagaria
tudo, sem caixa de entrada, leitura, preferência nem garantia contra aviso
repetido.

## Decisão

1. **Caixa de notificações da plataforma** (`platform/notificacoes`,
   tabelas `notificacoes` e `notificacoes_preferencias`), disponível a todos
   os módulos:
   - **Persistência:** cada aviso é gravado para o usuário, com título,
     mensagem, link e lido ou não.
   - **Idempotência:** a chave é única por usuário (`UNIQUE (user_id,
     chave)`). Repetir o envio, ou a reentrega de um evento, não duplica
     o aviso.
   - **Consistência:** o aviso é gravado **na mesma transação** da
     mudança (`Registrar`), então só existe se a mudança existir. Ele é
     **entregue em tempo real depois do commit** (`Entregar`, WebSocket
     no tópico pessoal `user:<id>`). A entrega é de melhor esforço: quem
     está fora vê o aviso ao abrir o sino.
   - **Opt-out por módulo:** em `/notificacoes`, a pessoa desliga os
     avisos de um módulo, e a escolha vale no envio.
   - **Segurança:** o link só aceita caminho interno (CHECK no banco e
     validação no código), sem redirecionamento externo. Cada pessoa só lê
     e marca a própria caixa (`GET /notificacoes`,
     `POST /notificacoes/{id}/lida`, `POST /notificacoes/lidas`,
     `GET`/`PUT /notificacoes/preferencias`).
   - **Retenção (LGPD, minimização):** um aviso lido há mais de 180 dias é
     apagado quando chega um aviso novo para a mesma pessoa.
## Consequências

- O sino mostra a caixa persistida e os avisos novos na hora.
  `/notificacoes` traz a lista completa e as preferências.
- Novos módulos podem avisar com `notificacoes.Registrar` dentro da
  própria transação, sem infraestrutura nova.
- As notificações são dado pessoal (destinatário e conteúdo): ficam
  sujeitas à retenção acima e são apagadas com o usuário
  (`ON DELETE CASCADE`).
- **Fora do escopo:** e-mail e resumo diário. Os avisos ficam no sino e na
  caixa, e a entrega por e-mail pode ser acrescentada depois sobre a mesma
  tabela.
