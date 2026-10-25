# 054 — Revelação confirmada e segredo no chamado

- **Status:** Aceito e implementado
- **Data:** 2026-10-24
- **Relacionado:** [ADR 053](053-cofre-de-senhas.md), [ADR 011](011-borda-unica-ip-confiavel-e-busca-por-prefixo.md), [ADR 013](013-permissao-com-escopo-e-heranca.md)

## Contexto

A primeira versão do cofre revelava a senha para quem já tinha sessão e
`cofre:reveal`. No mercado, abrir um segredo pede a senha de novo. Um
chamado que precise da senha não pode copiá-la para o texto.

## Decisão

1. **Confirmação.** `POST /api/v1/cofre/segredos/{id}/revelar` exige
   `senha_de_acesso`, a senha da pessoa, conferida pelo mesmo
   reautenticador do Signum. Conta local confere o hash. Conta do
   Keycloak confere no diretório. Senha errada conta no bloqueio do
   login (`user:` + nome). A auditoria registra a recusa sem o valor.
   Credencial de máquina não revela.
2. **Chamado.** `PUT /api/v1/atendimento/solicitacoes/{id}/segredo`
   grava `cofre_segredo_id`. A coluna não tem chave estrangeira e não
   recebe a senha. Quem abre o pedido não vê o id. O cofre desligado
   recusa o vínculo. Solicitação encerrada também.
3. **Fora desta leva.** Retirada com prazo, rotação automática e pasta
   com acesso próprio.

## Consequências

A migration `000010_segredo_no_chamado` acrescenta a coluna. O
atendimento pergunta o nome ao cofre por uma porta em `internal/app`.
Nenhum plugin importa o outro.
