# 085 — Protocolo de serviços: um plug-in para a carta e o pedido

- **Status:** Aceito e em implementação
- **Data:** 2026-11-16
- **Relacionado:** [ADR 008](008-kernel-grafo-de-modulos-e-estrategia-de-testes.md), [ADR 013](013-permissao-com-escopo-e-heranca.md), [ADR 016](016-camada-de-aplicacao-e-transacoes.md), [ADR 078](078-central-do-solicitante-e-canais.md), [ADR 084](084-protocolo-da-fazenda.md)

## Contexto

A prefeitura já opera um protocolo eletrônico em que a secretaria publica um serviço, desenha o formulário e o cidadão abre um pedido com número, status, etapa e consulta. Esse comportamento passa a viver no Projeto Nova. O Trâmite continua o processo interno. O Atendimento continua o chamado. A Fazenda continua o rito dos decretos.

## Decisão

1. O plug-in `protocolos` nasce desligado. Serviço, formulário, pedido, agendamento, responsável técnico e consulta pública ficam nele. Nenhum outro plug-in é importado.
2. Secretaria é a unidade do IAM. O recurso tem unidade dona. As permissões `protocolos:gerir`, `protocolos:tratar` e `protocolos:abrir` valem no escopo em que foram concedidas.
3. O pedido público não exige sessão. A consulta pelo número também não. CPF, telefone, respostas e a chave de arquivo não saem na consulta. O nome sai, salvo quando o campo de anonimato do serviço está marcado: aí o nome é denúncia anônima. A resposta à complementação também é pública: exige o CPF do pedido e só vale em `em_complementacao`.
4. O número junta a sigla da unidade (ou as iniciais do nome), o ano e cinco dígitos. Há nova tentativa se o número já existe.
5. O aviso ao cidadão usa o sino e os Canais da plataforma (e-mail e um WhatsApp: Meta, Evolution ou Gove). O e-mail e o WhatsApp vão para o contato informado no pedido quando o canal está ligado. O módulo não guarda SMTP nem segredo. O texto leva número, fato e orientação. O prazo conta dia útil em Cuiabá e pula sábado e domingo. O Egress recebe o evento de domínio para quem assinar `protocolos.*`.
6. O arquivo do formulário sobe pelo ticket do MinIO, direto do navegador, no prefixo do serviço. O pedido guarda a chave. Sem armazenamento configurado, a resposta continua o nome informado.

## Consequências

A carta e a mesa têm rota própria. Desligar o módulo tira as duas e não altera Trâmite, Atendimento nem Fazenda. A regra nova entra em `docs/REGRAS_DE_NEGOCIO.md` no mesmo conjunto de mudanças.
