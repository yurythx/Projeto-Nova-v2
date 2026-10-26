# 060 — Pasta com senha não apaga

- **Status:** Aceito e implementado
- **Data:** 2026-10-26
- **Relacionado:** [ADR 058](058-fila-curta-e-pasta-do-cofre.md)

## Contexto

Apagar a pasta devolvia os segredos à unidade, sem perguntar. Quem
administra podia alargar o alcance da senha sem perceber: o item saía
da lista da pasta e voltava a quem tem `cofre:read` na unidade.

## Decisão

`DELETE /api/v1/cofre/pastas/{id}` recusa a pasta que ainda tem senha.
A mensagem pede para esvaziar antes. A senha não é apagada junto. A
pasta vazia continua podendo ser apagada. A tela pede confirmação antes
de apagar pasta vazia e antes de apagar senha.

## Consequências

Quem quiser desfazer a pasta move ou apaga as senhas primeiro. A chave
estrangeira ainda zera `pasta_id` se a linha sumir por outro caminho; o
serviço não usa mais esse caminho enquanto houver senha.
