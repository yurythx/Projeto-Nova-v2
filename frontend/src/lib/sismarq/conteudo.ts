// Conteúdo institucional do Projeto-nova-v2: o microkernel e os plug-ins.

export const PRODUTO = {
  nome: "Projeto-nova-v2",
  finalidade:
    "Plataforma microkernel. O núcleo entrega identidade, auditoria e o ciclo de vida dos plug-ins. Cada módulo de negócio nasce desligado e só entra quando alguém o ativa.",
} as const;

export interface Pilar {
  nome: string;
  descricao: string;
}

export const PILARES: Pilar[] = [
  {
    nome: "Núcleo",
    descricao: "IAM e Auditoria ficam sempre ativos. Eles não se desligam em Configurações → Módulos.",
  },
  {
    nome: "Plug-ins",
    descricao:
      "Blog, Wiki, Agenda, Catálogo, Contato, Diretório, Arquivos, Trâmite, Signum, Busca, Mercúrio, Egress e o módulo modelo existem no código e começam desativados.",
  },
  {
    nome: "Isolamento",
    descricao:
      "Cada plug-in tem domínio, aplicação, infraestrutura e transporte. Um plug-in não importa o outro: a ligação passa pelo núcleo.",
  },
];

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
        q: "O que é o Projeto-nova-v2?",
        a: "É o microkernel desta plataforma: um núcleo estável e plug-ins que se ligam e desligam sem recompilar o sistema.",
        fonte: "Núcleo da plataforma",
      },
      {
        q: "O que fica ligado desde a instalação?",
        a: "Identidade e acesso (IAM) e a trilha de auditoria. O restante dos módulos nasce desativado.",
        fonte: "Configurações → Módulos",
      },
      {
        q: "Como ligo um módulo?",
        a: "Quem administra a plataforma ativa o plug-in em Configurações → Módulos. Se ele depender de outro, esse outro precisa estar ligado antes.",
        fonte: "Ciclo de vida do Kernel",
      },
    ],
  },
];
