# 072 — Importador do GLPI

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 067](067-eventos-do-monitoramento.md) (conta de sistema), [ADR 069](069-experiencia-de-quem-pede-xla.md) (avaliação)

## Contexto

A maior parte das prefeituras que vão adotar o Projeto Nova já atende pelo GLPI. Começar do zero apaga o histórico: o servidor perde os pedidos que fez, a central perde a série dos indicadores e a reincidência que sustenta a gestão de problemas. Sem migração, a adoção vira uma troca de ferramenta com perda, e quem decide adia.

## Decisão

1. **Comando de linha, fora da API.** `go run ./cmd/importarglpi` lê o GLPI e grava direto no banco do Projeto Nova, com a mesma configuração de banco dos outros comandos. A importação é uma operação de implantação, feita por quem administra, uma vez ou poucas, e não uma tela.
2. **Lê a API REST do GLPI** (`apirest.php`, GLPI 9.5 a 10), com App-Token e token de usuário, e não o banco MySQL. A API respeita as permissões do GLPI e não depende do esquema interno, que muda entre versões.
3. **O que entra:**
   - **Categorias** viram **ofertas inativas**: requisição ou incidente, conforme o tipo do chamado no GLPI. Todas entram na unidade escolhida no comando (`--unidade`). A organização revisa o catálogo e ativa as que valem.
   - **Chamados** viram **solicitações**, com título, descrição, estado, prioridade (urgência e impacto do GLPI, de 1 a 5, convertidos para a escala de 1 a 3) e as datas originais de abertura, solução e fechamento.
   - **Acompanhamentos** viram **comentários**. Os privados viram nota interna.
   - **Satisfação** vira **avaliação**: a nota de 0 a 5 do GLPI fica entre 1 e 5.
4. **Pessoas não são criadas.** Solicitante e técnico são casados com os usuários que já existem, pelo e-mail e depois pelo login, porque as contas vêm do AD/SSO. Quem não for achado fica com a conta de sistema **importação**, e o nome original vai no começo da descrição ou do comentário.
5. **Estados.** Novo entra aberto; pendente entra "aguardando terceiro", porque no GLPI ele é a espera genérica. Em atendimento entra em atendimento. Solucionado entra resolvido ou atendido, conforme o tipo. Fechado entra encerrado. Os abertos continuam na fila; os encerrados ficam para consulta e para os indicadores.
6. **Idempotente.** Cada item gravado guarda a origem (`glpi`, tipo e id) em `atendimento_importacao`. Rodar de novo pula o que já entrou e traz só o novo, o que permite uma carga grande antes da virada e uma final no dia.
7. **Simulação por padrão.** Sem `--gravar`, o comando só lê, casa as pessoas e mostra o que faria: quantos chamados, quantas pessoas sem par, quantas ofertas novas. Cada chamado é gravado na sua própria transação, junto com a sua origem. Um erro num chamado não desfaz os outros e aparece no relatório final.
8. **Sem eventos de negócio.** A importação não avisa ninguém nem publica no outbox. O passado não deve tocar o sino de quem atende. A auditoria registra uma linha por execução, com os totais.

## Consequências

A prefeitura migra com histórico, e os relatórios (ADR 069, 070) começam com dados reais. O casamento por e-mail depende de o AD estar sincronizado. O relatório da simulação mostra quem não casou, para corrigir antes de gravar.

Anexos do GLPI não entram nesta versão: pedem o armazenamento de objetos e o controle de tamanho. Ativos do GLPI também ficam de fora: o inventário do Projeto Nova vem do agente.

O histórico de cada chamado importado é o mínimo: aberto, em atendimento e a conclusão, com as datas do GLPI. Por isso a reabertura e as idas e voltas (ADR 069) só passam a valer para os chamados abertos aqui.
