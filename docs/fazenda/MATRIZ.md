# Matriz de requisitos — Decretos 13.653 e 13.654

Cada linha fecha quando o teste existe e passa. Fase 0 registra a matriz. As fases seguintes implementam.

| Requisito | Fase | Onde a regra mora | Teste |
|---|---|---|---|
| RC-01 gov.br, balcão sem papel | 1 e 2 | `platform/govbr`, `domain.Abrir` | `govbr/porta_test.go`, `domain/regras_test.go` |
| RC-02 procurador e prova | 2 | `domain.Checklist` com entrada procurador | `domain/regras_test.go` |
| RC-03 checklist bloqueia o envio | 2 | `domain.Checklist` | `domain/regras_test.go` |
| RC-04 recibo com data e hora | 2 | `domain.Abrir` | `domain/regras_test.go` |
| RC-05 ciência na consulta ou no 10º dia | 2 | `domain.Ciencia` | `domain/regras_test.go` |
| RC-06 exigência única e prazos | 2 | `domain.NovaExigencia`, `Prorrogar`, `Resolver` | `domain/regras_test.go` |
| RC-07 não pedir o que a prefeitura já tem | 2 | `domain.Checklist` com `JaNaBase` | `domain/regras_test.go` |
| RC-08 painel | 2 | situação, prazo e pendência em `Processo` | `domain/regras_test.go` |
| RC-09 QR e consulta pública | 2 | `domain.Certidao`, `ConfirmarAutenticidade` | `domain/regras_test.go` |
| RC-10 trilha | 2 | auditoria na mesma transação do ato | `application/service_test.go` |
| RC-11 recurso no processo | 2 | `domain.PodeRecorrer`, `Processo.Recorrer` | `domain/regras_test.go` |
| RR-01 simulador sem processo | 3 | `domain.Simular` | `domain/regras_test.go` |
| RR-02 requerimento A/B/C até 31/12/2026 23h59 MT | 3 | `domain.ValidarRequerimento`, `DentroDaAdesao` | `domain/regras_test.go` |
| RR-03 remissão e anistia | 3 | `domain.Remissao` | `domain/regras_test.go` |
| RR-04 parcelamento e termo | 3 | `domain.Parcelar` | `domain/regras_test.go` |
| RR-05 alerta, rescisão, estado parcelado só após a baixa | 3 e 4 | `domain.BaixarRetorno`, `SituacaoAtraso` | `domain/regras_test.go` |
| RR-06 execução fiscal | 3 | `domain.SinalizarExecucao` | `domain/regras_test.go` |
| RR-07 fila, 10 dias úteis, módulo C bloqueado | 3 | `domain.PrazoFila`, `ValidarRequerimento` | `domain/regras_test.go` |
| RR-08 certidão Anexo V | 3 | `domain.CertidaoAnexoV` | `domain/regras_test.go` |
| RR-09 alíquota −75% e registro até 31/03/2027, com a anotação da Lei 15.068 | 3 e 4 | `domain.ReducaoITBI`, `RegistroNoPrazo`, `NovaGuiaITBI` | `domain/regras_test.go` |
| RR-10 revogação, defesa 15 dias | 4 | `domain.PrazoDefesaRevogacao` | `domain/regras_test.go` |
| RR-11 alteração de contribuinte | 4 | `domain.PrazoAdquirente`, `EfeitoNoLancamento` | `domain/regras_test.go` |
| RR-12 DUA e validação | 4 | `domain.ValidarLinha` | `domain/regras_test.go` |
| RR-13 prazos e multa de R$ 200 | 4 | `domain.MultaDUA`, `PrimeiraDUA` | `domain/regras_test.go` |
| RR-14 comunicação de alienação | 4 | `domain.PodeComunicarAlienacao` | `domain/regras_test.go` |
| RR-15 cartório informa o registro | 5 | `domain.AvisoCartorio` | `domain/regras_test.go` |
| RA-01 a RA-04 reconhecimento e TQC | 5 | `domain.InvestimentoSuficiente` | `domain/regras_test.go` |
| RA-05 ITBI a 1% | 5 | `domain.ITBIMatriz` | `domain/regras_test.go` |
| RA-06 e RA-07 isenção de 3 exercícios a partir do seguinte ao despacho | 5 | `domain.NovaIsencao`, `Heranca`, `AnotacaoIsencao` | `domain/regras_test.go` |
| RA-08 e RA-09 marcos de 90 dias e 24 meses, relatórios em 31/01, 31/07 e 15 dias do marco | 5 | `domain.NovoCronograma`, `PrazoRelatorioSemestral`, `AlertarMarco` | `domain/regras_test.go` |
| RA-10 cassação, defesa 20 dias | 5 | `domain.PrazoDefesaCassacao`, `LancarDispensado` | `domain/regras_test.go` |
| RA-11 e RA-12 selo e desconto | 5 | `domain.DescontoSelo`, `MaisFavoravel` | `domain/regras_test.go` |
| RG-01 a RG-04 renúncia agregada | 5 | `domain.DivulgacaoExterna` | `domain/regras_test.go` |

O aviso externo (e-mail, WhatsApp, Egress) usa `domain.AvisoExterno`, que só carrega número, fato e orientação. Teste: `domain/regras_test.go`.

Conferência de 13/11/2026 contra a nota técnica e o guia operacional:

- Requerimento do Avança até 31/12/2030: `DentroDoAvanca`.
- Contrato do módulo C até 31/12/2025 e matrícula de até 30 dias: `ContratoNoLimite`, `MatriculaRecente`.
- Prova da data do art. 13: `ProvaDaData`. Módulo C pede `prova_data` e `matricula`, um arquivo cada.
- Isenção a partir do exercício seguinte ao despacho, com a anotação da Lei 15.067. Selo nos três exercícios seguintes: `ExerciciosDoSelo`.
- Marcos: início em 90 dias, conclusão em 24 meses, uma prorrogação pedida antes do vencimento, sem estender o benefício. TQC em 10 dias da ciência: `PrazoAssinaturaTQC`.
- Triagem em 5 dias úteis, parecer em 30 dias e decisão em 10 dias. A exigência aberta suspende a análise.
- Guia do Regulariza com o texto “Alíquota reduzida – Lei nº 15.068/2026”. Guia do Avança com a Lei nº 15.067.
- DUA do Anexo IV com matrícula, adquirente e endereço. Prazos: `EntregaDUA`.
- Benefício só com pedido e decisão expressa. Atualização de ofício não concede benefício.

Jornada sem terceiro, em 13/11/2026:

- O contribuinte autenticado abre o próprio processo. A permissão `fazenda:protocolar` continua obrigatória só quando o titular é outra pessoa.
- Recibo, exigência, conclusão e alerta saem de `domain.ModeloAviso`. A fila calcula `domain.Pendencia`.
- A entrega da planilha é `domain.ReceberDUA` em `POST /fazenda/processos/dua`. O recibo não repete o documento.
- A página pública `/autenticidade` gera o QR no navegador.
- A página `/govbr` inicia o Login Único e guarda o verificador neste navegador. `/govbr/retorno` troca o código e mostra nome e e-mail.
- A fila da equipe lê `domain.PrazosDoPainel`: o último relatório vencido fica atrasado e o próximo entra em alerta nos 30 dias anteriores.
- A decisão do Avança grava o despacho. `domain.MarcosDoProcesso` mostra o marco de 90 dias, o de 24 meses e o relatório de cada um. `Alertar` publica o aviso uma vez.
- Quem protocola o próprio processo fica em `destinatario`. O mesmo texto entra no sino e, com o canal ligado, sai por e-mail e WhatsApp. O processo `alertas-de-prazo` repete a publicação a cada hora. Protocolar para outra pessoa não usa o usuário da equipe como destinatário.
