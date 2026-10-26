// Conteúdo institucional do Projeto Nova. O texto explica o que o produto faz
// no enquadramento do ITIL (Version 5). Não reproduz o manual oficial.

export const PRODUTO = {
  nome: "Projeto Nova",
  descricaoCurta:
    "Gestão de produtos e serviços no ITIL 5: incidente, requisição, problema, mudança e projeto.",
  finalidade:
    "O Projeto Nova gere produtos e serviços digitais. O objetivo, no ITIL (Version 5), é criar valor com quem usa o serviço: restaurar o que parou, atender o pedido, remover a causa, mudar com controle e conduzir a iniciativa até o resultado.",
} as const;

export interface Pilar {
  nome: string;
  descricao: string;
}

/** As quatro práticas que o produto opera de ponta a ponta. */
export const PRATICAS: Pilar[] = [
  {
    nome: "Incidente e requisição",
    descricao:
      "O incidente é uma interrupção ou uma degradação: a meta é restaurar o serviço. A requisição é um pedido previsto, como acesso ou informação. Os dois entram na mesma fila, com tipo, impacto, urgência e prazo em horário de expediente.",
  },
  {
    nome: "Problema",
    descricao:
      "O problema investiga a causa que o incidente só contorna. Quando a causa é conhecida e ainda não sai de vez, o erro conhecido guarda o contorno para o próximo atendimento.",
  },
  {
    nome: "Mudança",
    descricao:
      "A mudança altera um serviço com avaliação, risco, janela e plano. Sem esse controle, a correção de hoje vira o incidente de amanhã.",
  },
  {
    nome: "Projeto",
    descricao:
      "O projeto conduz uma iniciativa com tarefas e prazo. Ele não substitui o chamado do dia nem a mudança: cada um tem o seu registro e o seu estado.",
  },
];

export const FUNDAMENTOS: Pilar[] = [
  {
    nome: "Objetivo",
    descricao:
      "O objetivo do ITIL é co-criar valor. No Projeto Nova isso aparece como serviço restabelecido, pedido atendido, causa registrada, mudança controlada e iniciativa acompanhada. Indicador sem amostra fica vazio: um zero inventado não é valor.",
  },
  {
    nome: "Princípios",
    descricao:
      "A operação segue os princípios que o ITIL (Version 5) mantém: foco no valor, começar de onde se está, avançar em passos com retorno, colaborar com visibilidade, olhar o serviço por inteiro, manter simples e automatizar o que já é repetitivo e seguro.",
  },
  {
    nome: "Ciclo do produto e do serviço",
    descricao:
      "O ciclo vai do entendimento e do desenho até a operação e o suporte, e volta na melhoria. Descobrir o que o serviço precisa, desenhar, construir e transicionar, operar e sustentar. O incidente vive na operação. A correção duradoura volta pela mudança.",
  },
  {
    nome: "Quatro dimensões",
    descricao:
      "Pessoas e organização (quem pede, quem atende, em qual unidade), fluxos e processos (fila, problema, mudança, projeto), informação e tecnologia (registro, auditoria, cofre) e parceiros (terceiro e fornecedor no chamado e no contrato).",
  },
];

export const REGRAS_PUBLICAS: Pilar[] = [
  {
    nome: "Quem vê o quê",
    descricao:
      "A permissão vale na unidade e nas unidades abaixo dela. Fora do alcance, o registro não aparece. Quem abre acompanha o próprio pedido. Quem atende trabalha a fila da sua unidade.",
  },
  {
    nome: "O que cada prática não faz",
    descricao:
      "O incidente não espera a causa raiz para encerrar. O problema não é um chamado renomeado. A mudança não se executa sem avaliação. O projeto não absorve a fila do dia.",
  },
  {
    nome: "O que fica registrado",
    descricao:
      "Cada ação relevante entra na auditoria. A auditoria não guarda senha, segredo do cofre nem coordenada. Métrica de prazo, resolução e satisfação só aparece quando houve amostra.",
  },
  {
    nome: "O que nasce desligado",
    descricao:
      "Identidade e auditoria ficam ligadas. Os demais módulos entram quando a instalação os ativa. Um módulo não lê o banco do outro: a ligação passa pelo núcleo.",
  },
];

export const PILARES: Pilar[] = [
  {
    nome: "Núcleo",
    descricao: "IAM e Auditoria ficam sempre ativos. Eles não se desligam em Configurações → Módulos.",
  },
  {
    nome: "Módulos de serviço",
    descricao:
      "Atendimento, problemas, mudanças e projetos formam a operação. Catálogo, wiki, ativos, contratos e cofre sustentam o serviço. Cada um nasce desligado e só entra quando a instalação liga.",
  },
  {
    nome: "Isolamento",
    descricao:
      "Cada módulo tem domínio, aplicação, infraestrutura e transporte. Um módulo não importa o outro: a ligação passa pelo núcleo, com auditoria na mesma transação do fato.",
  },
];

export const ACESSO = [
  "Incidente restaura o serviço. Requisição atende um pedido previsto.",
  "Problema trata a causa e guarda o erro conhecido.",
  "Mudança segue com avaliação, risco, janela e plano.",
  "Projeto conduz a iniciativa, separado da fila do dia.",
] as const;

export interface ItemFAQ {
  q: string;
  a: string;
  fonte: string;
}

export interface SecaoFAQ {
  categoria: string;
  itens: ItemFAQ[];
}

export const FAQ: SecaoFAQ[] = [
  {
    categoria: "O projeto",
    itens: [
      {
        q: "O que é o Projeto Nova?",
        a: "É a gestão de produtos e serviços desta instalação. O nome do produto é Projeto Nova. A operação segue o ITIL (Version 5) nas práticas que o sistema realmente executa: incidente, requisição, problema, mudança e projeto.",
        fonte: "Identidade do produto",
      },
      {
        q: "O que o ITIL pede e o que o sistema faz?",
        a: "O ITIL orienta a criar valor com quem usa o serviço, ao longo do ciclo do produto e do serviço. O Projeto Nova registra esse ciclo na fila, no problema, na mudança e no projeto. Não promete práticas que não estão ligadas nesta instalação.",
        fonte: "ITIL (Version 5), aplicado ao produto",
      },
      {
        q: "O que fica ligado desde a instalação?",
        a: "Identidade e acesso (IAM) e a trilha de auditoria. O restante dos módulos nasce desativado e entra em Configurações → Módulos.",
        fonte: "Configurações → Módulos",
      },
    ],
  },
  {
    categoria: "Objetivos e fundamentos",
    itens: [
      {
        q: "Qual é o objetivo?",
        a: "Co-criar valor: o serviço volta, o pedido é atendido, a causa fica registrada, a mudança acontece com controle e a iniciativa tem dono e prazo. Um número sem amostra não entra no painel.",
        fonte: "Objetivo do ITIL e regra de indicador",
      },
      {
        q: "Quais princípios orientam o uso?",
        a: "Foco no valor, começar do que já existe, avançar em passos com retorno, colaborar com visibilidade, olhar o serviço inteiro, manter simples e automatizar só o que já é repetitivo e seguro. A triagem sugere equipe e prioridade. Ela não escolhe uma pessoa ao acaso e não envia resposta sozinha.",
        fonte: "Princípios do ITIL (Version 5)",
      },
      {
        q: "O que são as quatro dimensões aqui?",
        a: "Pessoas e unidades, os fluxos de fila, problema, mudança e projeto, a informação (registro, auditoria e cofre) e os parceiros (terceiro no chamado e fornecedor no contrato).",
        fonte: "Dimensões da gestão de produto e serviço",
      },
    ],
  },
  {
    categoria: "Incidente e requisição",
    itens: [
      {
        q: "Qual a diferença entre incidente e requisição?",
        a: "Incidente: algo quebrou ou piorou e precisa voltar ao normal. Requisição: um pedido previsto, como um acesso ou uma informação. O tipo nasce da oferta do catálogo. Os dois compartilham a fila, os estados e o prazo.",
        fonte: "Atendimento",
      },
      {
        q: "Como a prioridade é definida?",
        a: "Pelo impacto e pela urgência. Sem regra específica, a classificação média usa impacto e urgência no nível 2. Uma regra da unidade pode ajustar equipe e prioridade. Ela não inventa um responsável.",
        fonte: "Política de atendimento",
      },
    ],
  },
  {
    categoria: "Problema, mudança e projeto",
    itens: [
      {
        q: "Quando abrir um problema?",
        a: "Quando a mesma falha se repete ou a causa ainda não foi removida. O erro conhecido descreve a falha e o contorno enquanto a correção definitiva não entra por uma mudança.",
        fonte: "Problemas e erros conhecidos",
      },
      {
        q: "O que uma mudança exige?",
        a: "Avaliação, risco, janela e plano. A mudança pode ser rejeitada, falhar ou ser concluída. O estado registra isso. Executar sem avaliação foge da política.",
        fonte: "Mudanças",
      },
      {
        q: "Para que serve o projeto?",
        a: "Para uma iniciativa com tarefas, com estado de planejado a concluído ou cancelado. O projeto não vira a fila de chamados nem substitui a mudança do serviço.",
        fonte: "Projetos",
      },
    ],
  },
  {
    categoria: "Regras e políticas",
    itens: [
      {
        q: "Quem pode ver um registro?",
        a: "Quem tem a permissão na unidade dona, ou numa unidade acima dela. Fora do alcance, a consulta responde como se o registro não existisse. Criar fora do alcance é recusado.",
        fonte: "Escopo por unidade",
      },
      {
        q: "O que a auditoria não guarda?",
        a: "Senha, segredo do cofre e coordenada geográfica. O fato da ação fica. O segredo, não.",
        fonte: "Auditoria e cofre",
      },
      {
        q: "Por que um indicador aparece vazio?",
        a: "Porque não houve amostra no período. Vazio significa sem amostra. Não é zero e não é meta batida.",
        fonte: "Indicadores do atendimento",
      },
    ],
  },
];
