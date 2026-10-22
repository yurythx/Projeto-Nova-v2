import {
  GitPullRequest,
  Headset,
  Megaphone,
  ArrowRightLeft,
  CalendarClock,
  FileSearch,
  GraduationCap,
  Package,
  Baby,
  BookOpen,
  Boxes,
  Briefcase,
  Building2,
  Cloud,
  Code,
  Cpu,
  Database,
  Eye,
  FileCode,
  FileText,
  Filter,
  FolderOpen,
  Globe,
  HandHeart,
  HardDrive,
  HeartPulse,
  Home,
  Info,
  KeyRound,
  Layers,
  LayoutDashboard,
  LayoutGrid,
  MessageCircle,
  MessageSquare,
  Monitor,
  Network,
  Scale,
  Server,
  Shield,
  ShieldCheck,
  Terminal,
  Users,
  Workflow,
  Wrench,
  type LucideProps,
} from "lucide-react";
import type { ComponentType } from "react";

// Ícone de um serviço do catálogo (campo "icon", nome lucide livre
// escolhido pelo gestor). Nome desconhecido cai no ícone genérico.
const ICONS: Record<string, ComponentType<LucideProps>> = {
  "arrow-right-left": ArrowRightLeft,
  "calendar-clock": CalendarClock,
  "file-search": FileSearch,
  "git-pull-request": GitPullRequest,
  "graduation-cap": GraduationCap,
  headset: Headset,
  megaphone: Megaphone,
  package: Package,
  baby: Baby,
  "book-open": BookOpen,
  boxes: Boxes,
  briefcase: Briefcase,
  building: Building2,
  cloud: Cloud,
  code: Code,
  cpu: Cpu,
  database: Database,
  eye: Eye,
  "file-code": FileCode,
  "file-text": FileText,
  filter: Filter,
  "folder-open": FolderOpen,
  globe: Globe,
  "hand-heart": HandHeart,
  "hard-drive": HardDrive,
  "heart-pulse": HeartPulse,
  home: Home,
  info: Info,
  "key-round": KeyRound,
  layers: Layers,
  "layout-dashboard": LayoutDashboard,
  "message-circle": MessageCircle,
  "message-square": MessageSquare,
  monitor: Monitor,
  network: Network,
  scale: Scale,
  server: Server,
  shield: Shield,
  "shield-check": ShieldCheck,
  terminal: Terminal,
  users: Users,
  workflow: Workflow,
  wrench: Wrench,
};

export const SERVICE_ICON_NAMES = Object.keys(ICONS);

export function ServiceIcon({ name, ...props }: { name?: string } & LucideProps) {
  const Icon = (name && ICONS[name]) || LayoutGrid;
  return <Icon {...props} />;
}

interface EstiloCategoria {
  badge: string;
  icon: string;
  tag: string;
}

// Classes escritas por extenso: o Tailwind só gera as que aparecem no código.
const PALETA: EstiloCategoria[] = [
  {
    badge:
      "bg-blue-500/15 border border-blue-600/30 text-blue-900 dark:bg-blue-500/20 dark:border-blue-400/50 dark:text-blue-300 shadow-xs",
    icon: "text-blue-700 dark:text-blue-300",
    tag: "bg-blue-50 text-blue-900 border-blue-300 dark:bg-blue-950/80 dark:text-blue-300 dark:border-blue-500/50 font-bold",
  },
  {
    badge:
      "bg-emerald-500/15 border border-emerald-600/30 text-emerald-800 dark:bg-emerald-500/20 dark:border-emerald-400/50 dark:text-emerald-300 shadow-xs",
    icon: "text-emerald-700 dark:text-emerald-300",
    tag: "bg-emerald-50 text-emerald-900 border-emerald-300 dark:bg-emerald-950/80 dark:text-emerald-300 dark:border-emerald-500/50 font-bold",
  },
  {
    badge:
      "bg-violet-500/15 border border-violet-600/30 text-violet-800 dark:bg-violet-500/20 dark:border-violet-400/50 dark:text-violet-300 shadow-xs",
    icon: "text-violet-700 dark:text-violet-300",
    tag: "bg-violet-50 text-violet-900 border-violet-300 dark:bg-violet-950/80 dark:text-violet-300 dark:border-violet-500/50 font-bold",
  },
  {
    badge:
      "bg-amber-500/15 border border-amber-600/30 text-amber-900 dark:bg-amber-500/20 dark:border-amber-400/50 dark:text-amber-300 shadow-xs",
    icon: "text-amber-800 dark:text-amber-300",
    tag: "bg-amber-50 text-amber-950 border-amber-300 dark:bg-amber-950/80 dark:text-amber-300 dark:border-amber-500/50 font-bold",
  },
  {
    badge:
      "bg-cyan-500/15 border border-cyan-600/30 text-cyan-800 dark:bg-cyan-500/20 dark:border-cyan-400/50 dark:text-cyan-300 shadow-xs",
    icon: "text-cyan-700 dark:text-cyan-300",
    tag: "bg-cyan-50 text-cyan-900 border-cyan-300 dark:bg-cyan-950/80 dark:text-cyan-300 dark:border-cyan-500/50 font-bold",
  },
  {
    badge:
      "bg-rose-500/15 border border-rose-600/30 text-rose-800 dark:bg-rose-500/20 dark:border-rose-400/50 dark:text-rose-300 shadow-xs",
    icon: "text-rose-700 dark:text-rose-300",
    tag: "bg-rose-50 text-rose-900 border-rose-300 dark:bg-rose-950/80 dark:text-rose-300 dark:border-rose-500/50 font-bold",
  },
];

/** Cor estável da categoria: a mesma categoria tem sempre a mesma cor, e
 * serviço sem categoria fica com a primeira. */
export function getCategoryIconStyles(category?: string | null): EstiloCategoria {
  const nome = (category || "").trim().toLowerCase();
  let soma = 0;
  for (const c of nome) soma = (soma * 31 + c.charCodeAt(0)) % 9973;
  return PALETA[nome ? soma % PALETA.length : 0]!;
}
