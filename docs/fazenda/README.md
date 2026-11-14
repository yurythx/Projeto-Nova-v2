# Protocolo da Fazenda

Base de revisão de toda entrega do módulo `fazenda`. O código só fecha um requisito quando o teste citado na [matriz](MATRIZ.md) passa e o texto abaixo não foi contrariado.

## Documentos de aceite

- [Nota técnica](Nota_Tecnica_TI_Decretos_13653_13654.docx) — requisitos RC, RA, RR e RG dos Decretos 13.653/2026 e 13.654/2026.
- [Guia operacional](Decretos_13653_13654_Guia_Operacional_Bandeira.pptx) — como a equipe aplica os decretos.
- Jornada de referência, fora deste repositório: o protocolo eletrônico da prefeitura. Entra o comportamento do contribuinte (entrar, protocolar, acompanhar, ser avisado). Não entra o prazo de 30 dias, o formulário livre nem o restante daquele sistema.

Decisão: [ADR 084](../adr/084-protocolo-da-fazenda.md). Arquitetura e regras de negócio: [ARQUITETURA.md](ARQUITETURA.md).

## O que não se faz numa entrega

- Portaria, tela ou configuração não criam requisito, prazo ou benefício.
- WhatsApp não substitui o domicílio (área restrita e e-mail). A ciência é a consulta ou o 10º dia corrido.
- Débito e CPF não saem no aviso.
- Débito sem a primeira baixa não se chama parcelado.
- Módulos de TI permanecem. A separação é perfil e unidade.
