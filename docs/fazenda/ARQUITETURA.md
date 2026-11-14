# Protocolo da Fazenda — arquitetura e regras

Documento de trabalho da equipe. A lista de aceite continua sendo a [nota técnica](Nota_Tecnica_TI_Decretos_13653_13654.docx) e o [guia operacional](Decretos_13653_13654_Guia_Operacional_Bandeira.pptx). Cada requisito aponta o teste na [matriz](MATRIZ.md). A decisão de arquitetura é o [ADR 084](../adr/084-protocolo-da-fazenda.md).

Conferido em 13/11/2026 contra esses dois documentos. O módulo nasce desligado.

## 1. Arquitetura

O protocolo é o plug-in `fazenda`, no mesmo núcleo da TI. Quem não tem permissão não vê o menu e a API recusa. IAM e Auditoria permanecem ligados. Atendimento e Trâmite não calculam tributo.

```text
Contribuinte (gov.br) ──┐
Servidor (login atual) ─┼── módulo fazenda ── canais (e-mail e um WhatsApp)
                         │                  ├── egress (evento para fora)
                         │                  └── integrações (cadastro, quando houver contrato)
                         └── página pública do código de autenticidade
```

Camadas, como os outros módulos:

| Camada | Onde | O que guarda |
|---|---|---|
| Domínio | `backend/internal/modules/fazenda/domain` | Prazos, bloqueios, alíquotas, estados |
| Aplicação | `.../application` | Quem pode ver e quem pode agir; processo, evento e auditoria na mesma transação |
| Banco | `.../infrastructure` e migrations `000062`, `000063` | Processo, sequência anual, segredo da Gove, dono da conexão |
| Rotas | `.../transport` | Simulador, gov.br, autenticidade, protocolo e atos da equipe |
| Tela | `frontend/src/app/regulariza`, `frontend/src/app/govbr` e `frontend/src/app/(protected)/fazenda` | Simulador público, Login Único e lista da equipe |

O número do processo é `NNNNNN/AAAA`. A data e a hora do recibo, no fuso de Mato Grosso (UTC−4), são a data legal do protocolo.

Egress publica o fato. Canais entrega o aviso. Integrações é o único catálogo de conexões, cada uma com dono `ti` ou `sefaz`. A regra da alíquota fica na Fazenda. O tipo `consulta_imobiliaria` não extrai chamado e não chama produção até a SEFAZ confirmar o contrato.

## 2. Quem entra e o que vê

| Pessoa | Entrada | O que vê |
|---|---|---|
| Contribuinte | gov.br | Só os próprios processos |
| Procurador | gov.br, com a procuração no processo | O processo em que representa |
| Atendente | Login do servidor, permissão `fazenda:protocolar` | Protocola no balcão, sem papel |
| Analista | `fazenda:analisar` | Exigência, parcelamento, baixa, execução, aviso |
| Autoridade | `fazenda:decidir` | Decisão, recurso e redução de alíquota |
| Cartório e público | Página do código | Só se o código confere. Sem débito e sem CPF |

No Avança, decide o Secretário de Fazenda. No Regulariza, o Secretário Adjunto de Receita ou o fiscal designado por portaria. O sistema não inventa essa designação: a permissão `fazenda:decidir` é concedida a quem a portaria nomear.

O servidor assina por dentro com o Signum. Procuração e despacho com certificado ICP-Brasil entram como prova anexada. O domicílio eletrônico é a área restrita mais o e-mail cadastrado. WhatsApp só avisa, com um provedor ativo por vez. A Gove é a terceira opção, ao lado de Meta e Evolution, e não recebe resposta.

## 3. Jornada

1. No Regulariza, a pessoa simula pela inscrição, sem abrir processo.
2. Entra pelo gov.br, pelo balcão ou por procurador.
3. Anexa um PDF por tipo do Anexo II. Faltando obrigatório, o envio não segue. Documento fora da lista legal não entra.
4. O recibo grava número, data e hora. Isso é o protocolo.
5. A ciência vale na consulta ou no 10º dia corrido, o que ocorrer primeiro.
6. A exigência é uma só. Enquanto estiver aberta, a análise para. Sem resposta, o processo arquiva. Outra exigência só por fato ou documento novo.
7. A equipe decide. O recurso fica no próprio processo. As etapas da Lei 1.800/1990 entram quando a SEFAZ entregar o rito.
8. Certidão e despacho levam código. A página pública confirma o código e nada mais.

O aviso que sai por e-mail, WhatsApp ou Egress leva o número do processo e a orientação. Débito e CPF ficam na área restrita.

## 4. Regras comuns

| Regra | Como o sistema aplica |
|---|---|
| RC-01 a RC-02 | Três entradas. Procurador, inclusive loteadora, incorporadora, imobiliária e tabelionato, anexa a procuração |
| RC-03 | Checklist do Anexo II, um arquivo por tipo |
| RC-04 | Recibo imutável em horário de Mato Grosso |
| RC-05 | Ciência na consulta ou no 10º dia |
| RC-06 | Avança: 15 dias, uma prorrogação. Regulariza: 5 dias úteis. Sem prorrogação |
| RC-07 | Dado que já está na base não vira campo obrigatório de tela |
| RC-08 | Situação, prazo e pendência no processo |
| RC-09 | Código de autenticidade e consulta pública |
| RC-10 | Auditoria do núcleo, sem edição |
| RC-11 | Recurso no processo, sem inventar as etapas da Lei 1.800/1990 |

Portaria não cria requisito, prazo nem benefício.

## 5. Regulariza Rondonópolis

Adesão até 31/12/2026, 23h59, horário de Mato Grosso. O requerimento tem três módulos: A obrigatório, B opcional, C só com débito quitado ou parcelado e com contrato assinado até 31/12/2025.

| Regra | Valor no código |
|---|---|
| Remissão e anistia | 100% dos juros e das multas moratórias de IPTU e taxas de 2021 a 2026. Principal e correção permanecem. Dolo, fraude ou simulação ficam de fora |
| Parcelamento | À vista ou até 12 parcelas, mínimo R$ 100. A primeira guia vence em até 10 dias. O estado fica “aguardando pagamento” até a primeira baixa. “Parcelado” só depois dela |
| Atraso | Alerta aos 30 dias. Acima de 60, rescisão e encargos proporcionais ao saldo |
| Módulo C | Cinco condições: contrato até 31/12/2025, pedido no prazo, cadastro atualizado, débitos quitados ou parcelados, registro até 31/03/2027. A alíquota cai 75% (2% vira 0,5%) |
| Prova da data | Firma, declaração de IR, declaração do vendedor, quitação ou documento com data certa. Este último exige decisão fundamentada. A data é até 31/12/2025 |
| Matrícula | Até 30 dias na data do protocolo |
| Fila | 10 dias úteis, em ordem de protocolo. Rito prioritário a partir de 01/12/2026 |
| Certidão | Anexo V: adquirente, transmitente, inscrição, matrícula, data do instrumento, alíquota e limite de 31/03/2027 |
| Guia | Texto “Alíquota reduzida – Lei nº 15.068/2026 – Certidão nº … – registro até 31/03/2027”. Depois dessa data, cobra-se a diferença |
| Revogação | Defesa de 15 dias |
| DUA | Anexo IV, linha a linha. Primeira entrega até 28/12/2026. Semestral a partir de 30/06/2027: 31/07 e 31/01. Multa de R$ 200 por unidade omitida ou errada |
| Contribuinte do IPTU | 15 dias para o adquirente. Efeito no lançamento de 2027. O titular registral segue responsável solidário |
| Comunicação de alienação | Pessoa física. Com o preço quitado e validado, o transmitente sai no exercício seguinte e responde pelos anteriores |
| Execução fiscal | O processo só sinaliza a Procuradoria. Suspender ou extinguir é ato dela |
| Ofício | Atualizar o cadastro de ofício não concede o benefício |

Pedido protocolado até 31/12/2026 conserva o direito à decisão, mesmo que a decisão saia depois.

## 6. Avança Rondonópolis

Requerimento até 31/12/2030, 23h59, horário de Mato Grosso, antes do início das obras. O benefício não nasce tácito, presumido nem automático.

| Regra | Valor no código |
|---|---|
| Investimento mínimo | 100% do valor venal de referência na data do protocolo. Contam obra e infraestrutura. Não contam compra do imóvel, tributo, venda, publicidade, corretagem, juros nem despesa administrativa |
| Etapas | Triagem em 5 dias úteis, parecer em 30 dias, decisão em 10 dias. A exigência aberta suspende esses prazos |
| TQC | Assinatura em 10 dias da ciência. Sem assinatura, o despacho não produz efeito |
| ITBI | 1% só no imóvel matriz, com as três condições, até 31/12/2030. A guia cita a Lei nº 15.067/2026 e o número da Certidão de Reconhecimento |
| Isenção de IPTU | Três exercícios a partir do seguinte ao despacho. Despacho em 2027 isenta 2028, 2029 e 2030. Venda, cessão, posse ou TVEO não movem o fim |
| Individualização | 30 dias após o TVEO. A unidade herda só o prazo restante |
| Marcos | Início em 90 dias, 30%, 70% e conclusão em até 24 meses. Uma prorrogação de até 12 meses, pedida antes do vencimento, não estende o benefício |
| Relatórios | Semestrais até 31/01 e 31/07. Por marco, até 15 dias depois. Alerta 30 dias antes. Relatório anual até 31/03 |
| Cassação | Defesa de 20 dias e lançamento do que foi dispensado |
| Selo | Mínimo de 2 critérios, no máximo 5. Desconto de 2%, 3%, 5% ou 8% por três exercícios depois da isenção. Sem cumulação: vale o mais favorável. Se um critério deixar de funcionar, o percentual cai |

A SEFAZ examina o efeito fiscal. O documento do órgão urbanístico não é reanalisado aqui.

## 7. Relatórios e sigilo

A divulgação externa só sai em dado agregado (CTN, art. 198). O Avança presta contas da renúncia por exercício e do relatório anual. O Regulariza presta o relatório final até 30/06/2027. O painel interno mostra etapa, prazo, exigência e relatório atrasado. Nome de autoridade não entra na divulgação.

## 8. O que já está no código e o que espera contrato

| Já está no módulo | Espera a SEFAZ ou outro sistema |
|---|---|
| Regras dos dois decretos, com teste | Credencial do gov.br, homologação e produção |
| Simulador público em `/regulariza` | Leitura real de inscrição, valor venal e débitos |
| Protocolo, exigência, decisão, recurso, parcelamento e baixa no estado | Emissão da guia e arquivo de retorno da arrecadação |
| Gove como terceiro WhatsApp, com segredo cifrado | Índice de correção da parcela e da multa da DUA |
| Conexão com dono SEFAZ ou TI | Texto das etapas da Lei 1.800/1990 |
| Aviso sem débito e sem CPF | Portarias da unidade, da autoridade, do manual e dos modelos |
| Lista da equipe em `/fazenda`, com pendência e prazo | Destino de contabilidade, diário e cartório, quando existir |
| Protocolo do próprio processo, acompanhamento e resposta da exigência | Login real do gov.br no lugar da sessão de desenvolvimento |
| Página pública `/autenticidade`, com QR gerado no navegador | Publicação no portal, com o módulo ligado |
| Entrega da DUA em `/fazenda/dua`, com recibo sem documento fiscal | Planilha oficial e baixa na arrecadação |
| Modelos de aviso, sino e saída por e-mail e WhatsApp quando o canal está ligado; alerta de prazo a cada hora | Provedor e credencial do canal na homologação |

Enquanto o cadastro e a arrecadação não tiverem contrato, a contingência é o processo eletrônico que a SEFAZ já usa. Os prazos da lei correm do mesmo jeito.

## 9. O que segue sem terceiro

As frentes que não dependiam de credencial nem de contrato estão no código:

1. Em `/fazenda` a entrada segue o desenho do protocolo de referência, com os componentes deste sistema: faixa de abertura, cartão de cada serviço com detalhe e solicitação, e a consulta pelo número e pela situação. O detalhe informa o prazo de análise do decreto. O formulário abre em `/fazenda/novo?programa=…`, com entrada gov.br, balcão ou procurador e as peças do Anexo II daquele serviço. A ficha em `/fazenda/{id}` mostra o serviço, a situação, o código de autenticidade, a ciência e o andamento com as datas gravadas. Quem analisa calcula o parcelamento, registra o retorno da primeira parcela, anota a redução e sinaliza a execução fiscal. A exigência aberta traz o prazo e a resposta. Quem analisa abre a exigência. Quem decide registra a decisão e, com o rito descrito pela SEFAZ, o recurso.
2. `/autenticidade` confirma o código e desenha o QR no navegador. O detalhe do Regulariza aponta o simulador em `/regulariza`. Os valores saem em reais. A entrada do gov.br e o retorno, com nome e e-mail, usam o mesmo cartão.
3. `/fazenda/dua` recebe a linha, devolve as inconsistências e um recibo sem documento fiscal. A multa fica na resposta autenticada.
4. A fila em `/fazenda` mostra pendência, prazo da exigência e, para a equipe, o relatório semestral, o anual e o final: no prazo, em alerta ou atrasado.
5. Recibo, exigência e conclusão usam o mesmo modelo de aviso. A decisão do Avança grava o despacho. A fila mostra o marco de 90 dias, o de 24 meses e o relatório de cada um. Um processo em segundo plano, e o botão Publicar alertas, gravam o evento `fazenda.aviso.externo` uma vez por marco. Quem protocolou o próprio processo também recebe o texto no sino. Com o canal ligado, o mesmo texto sai por e-mail e WhatsApp, sem dado fiscal.
6. `/fazenda/dua` recebe uma linha avulsa e também a planilha, uma unidade por linha. O recibo externo não repete documento nem valor.

Não se antecipa: o índice de correção, as etapas da Lei 1.800/1990, a chamada ao cadastro em produção e a guia real da arrecadação. O evento segue para o Egress. O sino, o e-mail e o WhatsApp entregam o mesmo texto quando o titular é usuário da plataforma e o canal está ligado.

## 10. Como testar

O módulo fica desligado até alguém com permissão de administração ativá-lo. Com ele ligado, em homologação:

- `GET /api/v1/fazenda/govbr` inicia o login do contribuinte. Sem credencial, a resposta diz que falta a credencial. Não cria sessão.
- `POST /api/v1/fazenda/simulador` com a inscrição devolve devido, remitido e ITBI estimado, sem abrir processo.
- `GET /api/v1/fazenda/autenticidade/{codigo}` responde se o código confere.
- A tela pública do simulador é `/regulariza`. A da autenticidade é `/autenticidade`. A entrada do contribuinte é `/govbr` e o retorno é `/govbr/retorno`. O verificador fica só naquele navegador. A tela mostra nome e e-mail. O CPF não aparece. Sem credencial, a mensagem da API fica na tela.
- A lista, o novo protocolo, o processo e a DUA ficam em `/fazenda`, atrás do login. Qualquer pessoa autenticada protocola o próprio processo. Protocolar para outro titular continua exigindo `fazenda:protocolar`. Os prazos e o recibo aparecem no horário de Cuiabá. Data zero de exigência não é exibida.

Situação em 13/11/2026, nesta máquina: a stack de desenvolvimento está no ar. A API responde em `http://localhost:8012` e a tela em `http://localhost:3012`. O módulo Fazenda foi ligado neste ambiente. A porta 8080 pertence a outro sistema. O código ainda não foi publicado para produção.
