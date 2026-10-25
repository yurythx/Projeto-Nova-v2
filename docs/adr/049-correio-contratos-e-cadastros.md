# 049 — Correio, contratos e os cadastros da terceira leva

- **Status:** Aceito e implementado
- **Data:** 2026-10-23
- **Relacionado:** [ADR 048](048-roteiro-itsm.md) (roteiro), [ADR 027](027-notificacoes.md) (sino antes do e-mail), [ADR 016](016-camada-de-aplicacao-e-transacoes.md) (transação), [ADR 031](031-typesense-na-busca-global.md) (chamado fora do índice público)

## Contexto

A [leva 1](048-roteiro-itsm.md) colocou a wiki na abertura, o chamado na
busca com o mesmo recorte de quem atende e o aviso do prazo para a
equipe. Faltavam o correio, o cadastro de contrato e os vínculos que o
GLPI já oferece. A plataforma de analytics e o atendente virtual
continuam de fora.

## Decisão

1. **Saída.** Se `SMTP_HOST` está definido, o aviso do módulo
   atendimento (passagem de estado e pedido de avaliação) sai por e-mail
   depois do sino. Falha de SMTP não desfaz a transação. Os outros
   módulos não mandam e-mail.
2. **Entrada.** Se `IMAP_HOST` está definido, um worker lê a caixa. A
   mensagem vira comentário quando o assunto traz `#N` ou `[#N]` e o
   remetente é o e-mail de quem abriu ou de quem atende. A reserva por
   `Message-ID` impede a segunda gravação. Remetente estranho, chamado
   encerrado ou número ausente não geram comentário e não são erro.
3. **Contratos.** Plug-in `contratos`, permissão `contratos:manage`.
   `GET/POST/DELETE /api/v1/contratos/fornecedores`,
   `/api/v1/contratos/contratos` e `/api/v1/contratos/licencas`. A
   licença não tem campo de chave. O worker avisa uma vez em cada marco
   (30, 7, 0 e vencido) para quem administra ou gere contratos no
   alcance da unidade.
4. **Relações de ativo.** `GET/POST /api/v1/ativos/{id}/relacoes` e
   `DELETE /api/v1/ativos/{id}/relacoes/{relacao}`. Tipos `depende`,
   `composto` e `conectado`. Dependência em ciclo é recusada. Não há
   descoberta de rede.
5. **Perguntas da oferta.** `campos` na oferta e `respostas` na
   abertura. Até 20. Tipos texto, número e lista. A resposta não é
   reescrita na edição da solicitação.
6. **Risco e congelamento.** `risco_score` opcional, de 5 a 15, na
   faixa do risco gravado. `GET/POST /api/v1/mudancas/congelamentos` e
   `DELETE /api/v1/mudancas/congelamentos/{id}`. A janela bloqueia
   criação, edição e o envio para avaliação. Não cancela o que já foi
   agendado.

## Consequências

A migration `000005_roteiro_itsm` acrescenta as colunas e as tabelas.
Sem SMTP e sem IMAP o comportamento anterior do sino permanece. O
índice público continua sem chamado.
