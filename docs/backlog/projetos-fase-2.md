# Projetos — correções depois da carta 5W2H

## Feito

### 1. Outbox engole erro

As escritas de projeto, tarefa, vínculo e apontamento devolvem o erro do
outbox. A falha desfaz a transação, no mesmo caminho da carta.

### 3. Responsável é texto livre

Gerente e responsável da tarefa passam a ser uma pessoa ativa. Com unidade
no projeto, a lotação exibida (perfil do diretório ou escopo principal)
precisa ser essa unidade. O nome gravado é o da conta. Nome novo sem id é
recusado. Nome antigo, já gravado sem id, permanece até alguém escolher uma
pessoa ou limpar o campo.

### 4. Cronograma de 90 dias

A carteira abre em 90 dias e oferece 30, 180 e 365.

### 2. Projeto concluído volta para em andamento

`concluido` não sai pela passagem comum. A reabertura é `POST /projetos/{id}/reabertura`
com o motivo, feita por um superior da unidade ou por um administrador. O
registro fica em `projetos_reaberturas`. Ao concluir, quem pode reabrir
recebe o aviso na caixa, com o link da ficha.
