# 084 — Protocolo da Fazenda

- **Status:** Aceito e em implementação
- **Data:** 2026-11-13
- **Relacionado:** [013](013-permissao-com-escopo-e-heranca.md), [016](016-camada-de-aplicacao-e-transacoes.md), [076](076-modulo-de-integracoes.md), [078](078-central-do-solicitante-e-canais.md)

## Contexto

Os Decretos 13.653/2026 (Avança Rondonópolis) e 13.654/2026 (Regulariza Rondonópolis) exigem protocolo eletrônico, com o contribuinte no canal, ciência, exigência única e benefícios de ITBI e IPTU. A prefeitura já opera um Protocolo Digital em outro repositório. A nota técnica e o guia operacional são a lista de aceite.

## Decisão

1. O protocolo novo é o plug-in `fazenda`, desligado por padrão, nas camadas do núcleo. A regra do tributo mora nele. Atendimento e Trâmite não calculam IPTU.
2. TI e Fazenda convivem. Quem não tem permissão do módulo não vê o menu. Projetos, se a SEFAZ precisar, é o módulo já existente, na unidade dela.
3. A jornada do Protocolo Digital é a referência de tela e de aviso. O prazo de 30 dias e o formulário livre daquele sistema não entram.
4. Egress continua a saída de eventos. Integrações continua o catálogo, agora com dono `ti` ou `sefaz` e com o tipo `consulta_imobiliaria`, que não extrai chamado.
5. O servidor entra pelo login atual. O contribuinte entra pelo gov.br (PKCE), porta em `platform/govbr`. O Signum permanece a assinatura interna. O certificado ICP-Brasil entra como prova anexada.
6. O domicílio é a área restrita e o e-mail. A ciência vale na consulta ou no 10º dia corrido. O WhatsApp tem um provedor ativo. A Gove é a terceira opção, ao lado de Meta e Evolution. O aviso não leva débito nem CPF.
7. Toda entrega é conferida pela matriz em `docs/fazenda/MATRIZ.md`.

## Consequências

Enquanto cadastro e arrecadação não tiverem contrato, a SEFAZ segue no processo eletrônico atual, como os decretos já permitem. O estado do parcelamento só vira `parcelado` depois da primeira baixa.
