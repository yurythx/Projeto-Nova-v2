# 064 — Catraca de cobertura enquanto a gestão de serviços não chega a 100%

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 009](009-revisao-dos-plugins-e-meta-de-cobertura.md), [ADR 010](010-revisao-da-plataforma-e-cobertura-por-pacote.md), [ADR 063](063-imagens-do-ci-no-registro.md)

## Contexto

Os ADRs 009 e 010 exigem 100% de cobertura por pacote, conferida pelo CI. Quando o histórico recomeçou (ADR 046), o `.github/workflows` não veio junto. Por duas semanas nada conferiu a meta, e a gestão de serviços entrou sem ela.

Com o CI de volta e todos os serviços de teste no ar (Postgres, RabbitMQ, MinIO, Typesense), 13 pacotes ficam abaixo de 100%. São cerca de 1.300 trechos, quase todos de atendimento, cofre, ativos, quadros, projetos e contratos.

Escrever esses testes antes de religar o CI deixaria o projeto mais tempo sem gate nenhum. Baixar a meta apagaria a regra que os outros 30 pacotes já cumprem.

## Decisão

1. A meta continua 100% por pacote.
2. `scripts/coverage-floor.txt` dá a cada pacote abaixo da meta um piso: a cobertura medida em 2026-10-27, menos 0,2 ponto de margem para testes que variam com o tempo. O pacote listado não pode cair abaixo do piso. O que não está listado, inclusive todo pacote novo, precisa de 100%.
3. O piso só sobe. O gate avisa quando a cobertura passou do piso em meio ponto e quando o pacote chegou a 100%; aí a linha sai do arquivo. Remover uma linha sem chegar a 100%, ou baixar um piso, é mudança deste ADR.
4. O arquivo de pisos é lido pelo nome, nunca pela posição na linha de comando. Arquivo vazio, ausente ou só com comentários não aprova nada. Perfil vazio também reprova. `scripts/coverage-gate.test.sh` confere esses casos no CI.

O item 4 existe porque a primeira versão desta catraca lia um arquivo de pisos vazio como se fosse o perfil e aprovava tudo. Por alguns minutos isso pareceu provar que a cobertura estava em 100%.

## Consequências

O CI volta a barrar regressão em todos os pacotes desde já. A dívida fica visível num arquivo só, com o número de cada pacote. A meta de 100% dos ADRs 009 e 010 continua de pé; esta é a rampa até ela.
