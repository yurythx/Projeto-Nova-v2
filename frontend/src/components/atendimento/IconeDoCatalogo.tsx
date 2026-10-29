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

/**
 * Inferência automática de ícone por nome/palavra-chave da categoria ou oferta.
 * Evita repetição indevida de ícones genéricos quando o campo vem vazio do banco.
 */
export function inferirIconePorTexto(texto?: string): string | null {
  if (!texto) return null;
  const n = texto
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "");

  if (/computador|desktop|maquina|pc|estacao|hardware/.test(n)) return "monitor";
  if (/notebook|laptop/.test(n)) return "laptop";
  if (/impressora|impressao|toner|scanner|cartucho/.test(n)) return "printer";
  if (/wifi|wi-fi|sem fio|wireless/.test(n)) return "wifi";
  if (/rede|internet|conexao|switch|roteador|cabo|fibra|ip/.test(n)) return "network";
  if (/servidor|banco de dados|storage|hospedagem|datacenter/.test(n)) return "server";
  if (/nuvem|cloud|drive/.test(n)) return "cloud";
  if (/disco|armazenamento|hd|ssd/.test(n)) return "hard-drive";
  if (/telefone|ramal|telefonia|voip|ligacao/.test(n)) return "phone";
  if (/celular|smartphone|tablet|movel/.test(n)) return "smartphone";
  if (/senha|acesso|login|credencial|usuario|permissao|reset|perfil/.test(n)) return "key-round";
  if (/sistema|software|programa|aplicativo|portal|erp|app|site|web/.test(n)) return "app-window";
  if (/email|e-mail|correio|mensagem|outlook/.test(n)) return "mail";
  if (/seguranca|antivirus|firewall|lgpd|vazamento/.test(n)) return "shield-alert";
  if (/camera|cftv|videomonitoramento/.test(n)) return "camera";
  if (/documento|certidao|declaracao|oficio|formulario|relatorio/.test(n)) return "file-text";
  if (/manutencao|reparo|conserto|oficina|tecnico|ajuste/.test(n)) return "wrench";
  if (/predial|eletrica|hidraulica|sala|ar-condicionado|ar condicionado|estrutura/.test(n))
    return "building-2";
  if (/pessoa|rh|recursos humanos|servidor|colaborador|equipe/.test(n)) return "users";
  if (/entrega|transporte|veiculo|logistica/.test(n)) return "truck";
  if (/compra|aquisicao|suprimento|licitacao/.test(n)) return "shopping-cart";
  if (/duvida|informacao|orientacao|ajuda|tutorial/.test(n)) return "circle-help";
  return null;
}

/** Ícone do bloco: o escolhido, o inferido pelo nome ou, sem eles, o padrão. */
export function IconeDoCatalogo({
  nome,
  contextoNome,
  padrao = "headset",
  ...props
}: { nome?: string; contextoNome?: string; padrao?: string } & LucideProps) {
  const inferido = !nome && contextoNome ? inferirIconePorTexto(contextoNome) : null;
  const iconeAlvo = nome || inferido || padrao;
  const Escolhido = PORNOME[iconeAlvo] ?? PORNOME[padrao] ?? CircleHelp;
  return <Escolhido aria-hidden="true" {...props} />;
}

/** Opções do seletor de ícone, com "padrão" no topo. */
export const OPCOES_DE_ICONE = [
  { value: "", label: "Padrão" },
  ...ICONES_DO_CATALOGO.map((i) => ({ value: i.valor, label: i.rotulo })),
];
