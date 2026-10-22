import {
  Archive,
  BookOpen,
  Box,
  Briefcase,
  Bug,
  ClipboardList,
  Calendar,
  Contact,
  Folder,
  FolderKanban,
  GitPullRequest,
  LayoutGrid,
  Mail,
  MessagesSquare,
  Newspaper,
  Package,
  PenTool,
  Search,
  Send,
  Server,
  ShieldCheck,
  Users,
  Wallet,
  type LucideProps,
} from "lucide-react";
import type { ComponentType } from "react";

// Ícone de cada módulo a partir do Manifest.Icon (nome lucide) do Kernel.
const ICONS: Record<string, ComponentType<LucideProps>> = {
  users: Users,
  archive: Archive,
  "shield-check": ShieldCheck,
  "messages-square": MessagesSquare,
  send: Send,
  newspaper: Newspaper,
  "layout-grid": LayoutGrid,
  mail: Mail,
  contact: Contact,
  calendar: Calendar,
  folder: Folder,
  "book-open": BookOpen,
  search: Search,
  "pen-tool": PenTool,
  "folder-kanban": FolderKanban,
  box: Box,
  briefcase: Briefcase,
  "clipboard-list": ClipboardList,
  package: Package,
  server: Server,
  bug: Bug,
  "git-pull-request": GitPullRequest,
  wallet: Wallet,
};

export function ModuleIcon({ name, ...props }: { name: string } & LucideProps) {
  const Icon = ICONS[name] ?? Box;
  return <Icon {...props} />;
}
