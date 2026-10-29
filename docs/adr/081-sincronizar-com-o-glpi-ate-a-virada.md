# ADR 081 — Sincronizar com o GLPI até a virada

## Contexto

A carga ([ADR 076](076-modulo-de-integracoes.md)) trouxe 14.998 chamados, mas o GLPI segue em uso até a virada. A carga incremental só pegava os chamados novos: o que mudava lá depois de importado (estado, técnico, solução, respostas, avaliação) ficava velho aqui. Extrair e carregar dependia de alguém pedir pela tela.

## Decisão

1. **Chave na conexão.** `sincronizar` liga e desliga pela tela ("Sincronizar até a virada", `PUT /integracoes/conexoes/{id}/sincronizacao`), com auditoria.
   - Para ligar, a conexão precisa poder extrair (token do conector ou credenciais) e carregar (tabela de-para completa, com unidade em todas as linhas).
2. **Agendamento no worker das integrações.** A cada `IntervaloSincronizacao` (15 min), o worker pede a extração das conexões que sincronizam, com `pedida_por = "sincronização"`.
   - O intervalo conta da última extração **pedida**, não da concluída. Com o conector fora do ar, a sincronização espera em vez de repetir.
   - Uma tarefa pedida pela tela no meio tempo faz a vez dela.
   - Quando a extração termina bem, a carga entra na fila na mesma transação.
3. **O importador atualiza o já importado que mudou.**
   - O registro guarda a versão do chamado no GLPI (`date_mod`) e quando a sincronização o tocou por último (`sincronizado_em`).
   - Com a sincronização ligada, o chamado com outra versão é trazido de novo:
     - estado, com a passagem na data da mudança no GLPI;
     - técnico;
     - solução e fechamento;
     - acompanhamentos novos;
     - avaliação;
     - última atualização.
   - Os acompanhamentos entram no registro e não se repetem. Para os importados antes deste ADR, a comparação é pela mesma hora no mesmo chamado.
   - Versão vazia (carga anterior) faz a primeira sincronização trazer todos de novo, sem repetir nada.
4. **O importado é espelho, só leitura.** Com alguma conexão sincronizando, o atendimento recusa mudar o chamado importado (`ErrEspelho`, 409 "atendido no GLPI até a virada"):
   - comentário;
   - estado;
   - atribuição;
   - classificação;
   - apontamento;
   - IA;
   - visitas;
   - cofre e contrato;
   - conclusão por mudança.

   O detalhe vem com `espelho: true` e sem próximos passos, e a tela mostra o aviso. A resposta por e-mail não entra, e a do WhatsApp recebe a explicação. O chamado aberto no Nova segue normal.
5. **Na virada, desliga-se.** Os importados passam a ser atendidos aqui, sem nenhuma migração. Depois, revoga-se o token do conector e trocam-se o App-Token e a senha do GLPI.
6. **O desfazer** ([ADR 079](079-desfazer-a-carga.md)) compara com a última sincronização, e não com a carga. O que a sincronização trouxe não conta como mexido aqui.

## Consequências

- Cada ciclo lê a área de preparo inteira (15 mil itens, alguns segundos) e só grava o que mudou. Se ficar pesado, o próximo passo é filtrar pela versão já na leitura.
- O conector precisa rodar em modo `--vigiar` numa máquina da rede do GLPI, como serviço. Fora do ar, nada quebra: a sincronização só para de trazer mudanças.
- Durante a sincronização, a equipe atende os chamados vindos do GLPI no GLPI. O Nova serve de consulta, com a busca e a lista do [ADR 080](080-lista-de-chamados-como-o-glpi.md).
- Os testes dos pacotes que mexem nas mesmas tabelas globais (registro da importação, política institucional) devem rodar em sequência (`-p 1`, como no CI).

## Implementação

- **Migração 42:** `versao` e `sincronizado_em` no registro, o tipo `acompanhamento`, e `sincronizar` na conexão.
- `atendimento/glpi/sincronizar.go` (`atualizar`, `registrados`, `ler`, `marcos`, `gravarAvaliacao`) e o laço em `importar.go`.
- Atendimento:
  - `WithEspelho`, `espelhado` e `recusarEspelho`;
  - `Repository.Importado`;
  - `Detalhe.Espelho`.
- Integrações:
  - `Sincronizar` e `SincronizacaoAtiva`;
  - o agendamento (`agendar`, `ParaSincronizar`);
  - a carga depois da extração, em `terminar`.
- App: `espelhoGLPI`, e `Importador.Sincronizar` a partir da conexão.
- Telas: o bloco "Sincronizar até a virada" na conexão e o aviso no chamado.
