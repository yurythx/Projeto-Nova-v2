# Registros de Decisão de Arquitetura (ADR)

Toda mudança de contrato de API, RBAC, auditoria ou middleware exige um ADR
(ver checklist no template de PR).

| ADR | Decisão | Status |
|---|---|---|
| [002](002-enterprise-resilience-and-governance.md) | Resiliência: idempotência, circuit breaker, DLQ | Aceito (contexto histórico: parte dos módulos citados foi substituída pelo Microkernel) |
| [003](003-local-auth-rsa-hardening.md) | Login local RS256 e bloqueio de conta | Aceito (o bloqueio vigente é o progressivo no Redis — ver 011) |
| [004](004-outbox-listen-notify-e-rbac-modular.md) | Outbox por LISTEN/NOTIFY e RBAC por módulo | Aceito (o exemplo `demands` é histórico; o RBAC modular vive hoje nos Manifests do Kernel) |
| [005](005-roadmap-conformidade-governamental.md) | Roadmap de conformidade SGD/MGI | Em execução |
| [006](006-rfc7807-problem-details.md) | Erros RFC 7807 por negociação de conteúdo | Aceito |
| [007](007-excecao-csp-style-src-vlibras.md) | Exceção de CSP `style-src` (VLibras) | Aceito, revisão trimestral |
| [008](008-kernel-grafo-de-modulos-e-estrategia-de-testes.md) | Grafo de módulos, invariantes do IAM, proveniência da auditoria, LGPD federado e estratégia de testes | Aceito |
| [009](009-revisao-dos-plugins-e-meta-de-cobertura.md) | Revisão das regras de negócio de cada plug-in e meta de 100% de cobertura por módulo no CI | Aceito |
| [010](010-revisao-da-plataforma-e-cobertura-por-pacote.md) | Revisão de cada pacote da plataforma e da composição; RabbitMQ e MinIO reais no CI; meta de 100% por pacote | Aceito |
| [011](011-borda-unica-ip-confiavel-e-busca-por-prefixo.md) | Proxy HTTPS como borda única, IP do visitante confiável ponta a ponta, bloqueio de login por IP com limite próprio, busca por prefixo, reautenticação do Signum | Aceito |
| [012](012-estrutura-desativada-nao-concede.md) | Estrutura desativada deixa de conceder perfis e de receber processos; unidade-mãe com subunidades não é excluída | Aceito |
| [013](013-permissao-com-escopo-e-heranca.md) | Permissão com escopo e herança: as permissões de gestão valem onde foram concedidas (e abaixo); unidade dona do conteúdo; administração delegada do IAM | Aceito e implementado |
| [014](014-publico-alvo.md) | Público-alvo: Blog, Wiki e Agenda publicados só para secretarias e/ou unidades | Aceito |
| [016](016-camada-de-aplicacao-e-transacoes.md) | A camada de aplicação delimita a transação com pgx (escrita + outbox + auditoria); domínio nunca importa o driver | Aceito |
| [020](020-conexoes-de-ia.md) | Conexões de IA pela tela (Configurações → Inteligência artificial): fornecedores compatíveis com a API da OpenAI, chave cifrada e só de escrita, teste antes de salvar, principal e reserva, autorização e mascaramento para fornecedor externo (LGPD); IA local como serviço `ia-local` | Aceito e implementado |
| [027](027-notificacoes.md) | Notificações persistentes da plataforma: caixa, leitura, opt-out por módulo, idempotência, gravação na mesma transação e entrega em tempo real depois do commit, retenção | Aceito e implementado |
| [030](030-blog-autoria-dedicada-publico-alvo-e-estrutura-rondonopolis.md) | Autoria do Blog em páginas dedicadas (novo/editar com TipTap WYSIWYG e MinIO); seletor de público-alvo recolhido por padrão ("Todos"); estrutura corporativa exclusiva de Rondonópolis (MT) e unificação do nome Projeto-nova-v2 | Aceito e implementado |
| [031](031-typesense-na-busca-global.md) | Typesense como índice da Busca Global: cliente com coleção garantida, filtro escapado e chave obrigatória; só conteúdo público indexado; carga no worker por geração, recarga por evento e volta ao Postgres se o índice falhar | Aceito e implementado |
| [046](046-produto-novo.md) | Produto novo: o sistema anterior (SISMARQ, arquivo municipal) sai por inteiro; migrations num esquema base para banco novo; nomes do produto trocados | Aceito e implementado |
| [047](047-avaliacao-governanca-e-cascata.md) | Avaliação persistida, indicadores da central a partir dos relógios, aviso de 2 horas e conclusão dos vínculos da mudança na mesma transação | Aceito e implementado |
| [048](048-roteiro-itsm.md) | O que a gestão de serviços copia do mercado, o que não copia, e a ordem das levas | Aceito; as três levas estão implementadas |
| [049](049-correio-contratos-e-cadastros.md) | Correio do atendimento, contratos, relações de ativo, perguntas da oferta, nota de risco e congelamento | Aceito e implementado |
| [050](050-portal-serie-e-agente.md) | Portal de quem pede, série diária dos indicadores e agente da própria máquina | Aceito e implementado |
| [051](051-fuso-da-serie-e-inventario-local.md) | Dia da série no fuso da unidade e inventário da própria máquina | Aceito e implementado |
| [052](052-programas-e-credencial-de-agente.md) | Programas da máquina e credencial que não é de pessoa | Aceito e implementado |
| [053](053-cofre-de-senhas.md) | Cofre de senhas no escopo da unidade, com revelação auditada | Aceito e implementado |
| [054](054-revelacao-e-segredo-no-chamado.md) | O chamado guarda só o id do segredo; a confirmação ao revelar foi substituída pelo 059 | Aceito e implementado |
| [055](055-endereco-e-painel-no-cofre.md) | Endereço à vista e painel que aponta as máquinas que administra | Aceito e implementado |
| [056](056-retirada-com-prazo.md) | Retirada exclusiva com prazo; devolver não troca a senha | Aceito e implementado |
| [057](057-rotacao-do-cofre.md) | Rotação da senha guardada ao devolver, ao vencer e a cada 30 dias | Aceito e implementado |
| [058](058-fila-curta-e-pasta-do-cofre.md) | Triagem curta, resposta pronta, chamado mensal, licença contra programa, pasta do cofre e deflexão | Aceito e implementado |
| [059](059-revelar-sem-senha-de-acesso.md) | Revelar usa a sessão; retirar continua pedindo a senha de acesso | Aceito e implementado |
| [060](060-pasta-vazia-para-apagar.md) | Pasta com senha não apaga; exclusão de pasta e de senha pede confirmação | Aceito e implementado |
| [061](061-catalogo-de-colunas.md) | Nome, cor e ordem das colunas no banco; a chave continua o estado | Aceito e implementado |
| [062](062-coluna-nova-no-quadro.md) | Coluna nova nasce só no quadro pedido, com chave gerada e passagens próprias | Aceito e implementado |
| [063](063-imagens-do-ci-no-registro.md) | Imagens geradas pelo CI no GHCR (assinadas, com SBOM); o servidor só baixa a tag de um commit testado | Aceito e implementado |
| [064](064-catraca-de-cobertura.md) | Catraca de cobertura: piso por pacote abaixo de 100%, que só sobe; pacote novo continua exigindo 100% | Aceito e implementado |
| [065](065-ia-assistiva-no-atendimento.md) | IA assistiva no atendimento: sugere classificação e resumo a quem atende; cada sugestão e decisão é registrada e auditada; nada no chamado muda sem a pessoa | Aceito e implementado |
| [066](066-ocorrencias-e-pagina-de-status.md) | Incidente grave: plug-in Ocorrências com coordenação, linha do tempo e revisão; página de status pública em /status, só com o que é público | Aceito e implementado |
| [067](067-eventos-do-monitoramento.md) | Alertas do Alertmanager viram incidente: um chamado por fingerprint, reenvio e volta ao normal viram nota; token de máquina; conta de sistema monitoramento | Aceito e implementado |
| [068](068-aprovacao-em-etapas-e-tarefas.md) | Requisição com aprovação em etapas (a chefia decide sem permissão do atendimento; recusa com motivo cancela) e tarefas por equipe | Aceito e implementado |
| [069](069-experiencia-de-quem-pede-xla.md) | Experiência de quem pede (XLA): facilidade na avaliação, reabertura, idas e voltas e espera em horas corridas, com metas da instituição | Aceito e implementado |
| [070](070-relatorio-por-secretaria-e-ola.md) | Relatório por secretaria (unidade de topo da árvore) com prazo, primeiro contato e experiência; OLA das equipes medido nas tarefas | Aceito e implementado |
| [071](071-contrato-de-apoio.md) | Contrato de apoio: prazo do fornecedor no contrato, o chamado aponta o contrato (cópia do combinado) e o tempo em "aguardando terceiro" é medido no relatório | Aceito e implementado |
| [072](072-importador-do-glpi.md) | Importador do GLPI: comando que lê a API REST, simula por padrão, casa pessoas por e-mail e login, e pode rodar de novo sem duplicar | Aceito e implementado |
| [073](073-ola-no-chamado.md) | OLA no chamado: histórico de atribuição por equipe, tempo sem as esperas pelo servidor e pelo fornecedor, reconstruído para o passado | Aceito e implementado |
