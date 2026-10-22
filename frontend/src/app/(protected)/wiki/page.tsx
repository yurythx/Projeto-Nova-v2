import { BookOpen } from "lucide-react";

import { PageHeader } from "@/components/sismarq/PageHeader";

export default function WikiHome() {
  return (
    <div>
      <PageHeader eyebrow="Wiki" title="Base de conhecimento" description="Manuais internos em Markdown, em árvore, com histórico de versões. O módulo nasce desligado." />
      <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-surface-border p-12 text-center text-muted">
        <BookOpen size={32} aria-hidden="true" />
        <p>Escolha uma página na árvore ou crie uma nova com o botão +.</p>
      </div>
    </div>
  );
}
