import {
  AppWindow,
  Building2,
  Camera,
  CircleHelp,
  Cloud,
  Cpu,
  FileText,
  HardDrive,
  Headset,
  KeyRound,
  Laptop,
  Lock,
  Mail,
  Monitor,
  Network,
  Phone,
  Printer,
  Server,
  ShieldAlert,
  ShoppingCart,
  Smartphone,
  TriangleAlert,
  Truck,
  UserPlus,
  Users,
  Wifi,
  Wrench,
  type LucideProps,
} from "lucide-react";
import type { ComponentType } from "react";

type Icone = ComponentType<LucideProps>;

/** Os ícones que a gestão escolhe para categorias e ofertas (ADR 078). O
 * banco guarda só o nome; um nome desconhecido cai no padrão do tipo. */
export const ICONES_DO_CATALOGO: { valor: string; rotulo: string; Icone: Icone }[] = [
  { valor: "monitor", rotulo: "Computador", Icone: Monitor },
  { valor: "laptop", rotulo: "Notebook", Icone: Laptop },
  { valor: "printer", rotulo: "Impressora", Icone: Printer },
  { valor: "smartphone", rotulo: "Celular", Icone: Smartphone },
  { valor: "phone", rotulo: "Telefone", Icone: Phone },
  { valor: "wifi", rotulo: "Wi-Fi", Icone: Wifi },
  { valor: "network", rotulo: "Rede", Icone: Network },
  { valor: "server", rotulo: "Servidor", Icone: Server },
  { valor: "hard-drive", rotulo: "Armazenamento", Icone: HardDrive },
  { valor: "cloud", rotulo: "Nuvem", Icone: Cloud },
  { valor: "cpu", rotulo: "Equipamento", Icone: Cpu },
  { valor: "app-window", rotulo: "Sistema", Icone: AppWindow },
  { valor: "mail", rotulo: "E-mail", Icone: Mail },
  { valor: "key-round", rotulo: "Senha", Icone: KeyRound },
  { valor: "lock", rotulo: "Acesso", Icone: Lock },
  { valor: "user-plus", rotulo: "Novo usuário", Icone: UserPlus },
  { valor: "users", rotulo: "Pessoas", Icone: Users },
  { valor: "shield-alert", rotulo: "Segurança", Icone: ShieldAlert },
  { valor: "camera", rotulo: "Câmera", Icone: Camera },
  { valor: "file-text", rotulo: "Documento", Icone: FileText },
  { valor: "shopping-cart", rotulo: "Compra", Icone: ShoppingCart },
  { valor: "truck", rotulo: "Entrega", Icone: Truck },
  { valor: "building-2", rotulo: "Prédio", Icone: Building2 },
  { valor: "wrench", rotulo: "Manutenção", Icone: Wrench },
  { valor: "headset", rotulo: "Suporte", Icone: Headset },
  { valor: "triangle-alert", rotulo: "Problema", Icone: TriangleAlert },
  { valor: "circle-help", rotulo: "Dúvida", Icone: CircleHelp },
];

const PORNOME: Record<string, Icone> = Object.fromEntries(
  ICONES_DO_CATALOGO.map((i) => [i.valor, i.Icone]),
);

/** Ícone do bloco: o escolhido ou, sem ele, o padrão. */
export function IconeDoCatalogo({
  nome,
  padrao = "circle-help",
  ...props
}: { nome?: string; padrao?: string } & LucideProps) {
  const Escolhido = PORNOME[nome ?? ""] ?? PORNOME[padrao] ?? CircleHelp;
  return <Escolhido aria-hidden="true" {...props} />;
}

/** Opções do seletor de ícone, com "padrão" no topo. */
export const OPCOES_DE_ICONE = [
  { value: "", label: "Padrão" },
  ...ICONES_DO_CATALOGO.map((i) => ({ value: i.valor, label: i.rotulo })),
];
