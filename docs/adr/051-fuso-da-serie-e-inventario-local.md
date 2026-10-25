# 051 — Dia da série no fuso da unidade e inventário da própria máquina

- **Status:** Aceito e implementado
- **Data:** 2026-10-24
- **Relacionado:** [ADR 050](050-portal-serie-e-agente.md)

## Contexto

A série cortava o dia em UTC. O expediente já usa o fuso da política da
unidade, em geral Brasília. O agente só enviava o nome do sistema. O
portal continua o mesmo: não há uma segunda interface. Não há varredura
de rede nem plataforma de analytics.

## Decisão

1. **Dia civil.** Cada unidade grava o dia no fuso da própria política.
   Sem política, vale `America/Sao_Paulo`. A linha institucional e a
   janela de quem cobre várias unidades usam a política institucional,
   com o mesmo padrão. A data entra no banco como `DATE`, para o fuso
   da sessão não deslocar o dia. Dias já gravados não são reescritos.
2. **Inventário.** `POST /api/v1/ativos/inventario` aceita sistema,
   processadores, memória e até 16 discos locais. A medida fica no
   ativo e a lista mostra o resumo. Memória ausente não vira zero.
   Edição manual conserva a medida. O programa lê o sistema, a memória
   e os volumes locais. Disco de rede e compartilhamento ficam de fora.

## Consequências

A migration `000007_inventario_da_maquina` acrescenta `ativos.inventario`,
nulo quando ninguém mediu.
