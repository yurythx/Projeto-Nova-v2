# 052 — Programas da máquina e credencial que não é de pessoa

- **Status:** Aceito e implementado
- **Data:** 2026-10-24
- **Relacionado:** [ADR 051](051-fuso-da-serie-e-inventario-local.md), [ADR 013](013-permissao-com-escopo-e-heranca.md)

## Contexto

O agente media a capacidade da máquina e autenticava com o token de uma
conta. O correio e o cadastro de contratos tinham pouco teste no caminho
que grava. Continua de fora: portal separado, varredura de rede,
analytics e chatbot.

## Decisão

1. **Programas.** O inventário aceita até 100 pares de nome e versão.
   Caminho é recusado. Passou de 100, a lista é parcial e o resumo diz
   isso. No Linux lê o cadastro de pacotes. No Windows lê o registro da
   máquina, não o da pessoa, e ignora componente de sistema.
2. **Credencial.** `ativos:manage` emite um segredo `nova_agt_` para uma
   unidade. Só o hash fica gravado. O bearer recebe `ativos:reportar`
   nessa unidade e não passa pelo cadastro de pessoas. Revogar impede
   novo uso e conserva a linha.

## Consequências

A migration `000008_credencial_de_agente` cria `ativos_credenciais`.
Sem a credencial, o token de uma conta com `ativos:reportar` continua
válido.
