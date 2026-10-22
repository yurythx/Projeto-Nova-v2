import { ServiceIcon } from "@/lib/catalog/icons";

interface BlogCoverImageProps {
  coverUrl?: string | null;
  title: string;
  slug: string;
  aspect?: "hero" | "card" | "article";
}

// Ícone da capa padrão pelo tema da publicação (palavra no slug).
const TEMA_ICONE: [string, string][] = [
  ["temporalidade", "calendar-clock"],
  ["eliminacao", "file-x"],
  ["digitalizacao", "scan-line"],
  ["recolhimento", "archive"],
  ["comiss", "users"],
  ["classificacao", "list-tree"],
  ["privad", "landmark"],
  ["acesso", "book-open"],
];

export function BlogCoverImage({ coverUrl, title, slug, aspect = "card" }: BlogCoverImageProps) {
  const iconName = TEMA_ICONE.find(([tema]) => slug.includes(tema))?.[1] ?? "archive";

  const aspectClass =
    aspect === "hero"
      ? "h-64 sm:h-80 w-full"
      : aspect === "article"
        ? "h-72 sm:h-96 w-full"
        : "h-48 w-full";

  // Se houver imagem do MinIO, renderiza a imagem real
  if (coverUrl) {
    return (
      <div
        className={`relative overflow-hidden rounded-xl border border-surface-border bg-surface-2 ${aspectClass}`}
      >
        {/* URL assinada do MinIO, que expira: o otimizador do next/image a guardaria em cache. */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={coverUrl}
          alt={`Capa do artigo: ${title}`}
          loading="lazy"
          className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
        />
      </div>
    );
  }

  // Fallback visual com a marca do Projeto-nova-v2
  return (
    <div
      className={`relative flex flex-col justify-between overflow-hidden rounded-xl border border-surface-border bg-gradient-to-br from-surface-2 via-surface-1 to-surface-3 p-6 text-foreground shadow-xs transition-all duration-300 group-hover:border-accent/40 ${aspectClass}`}
    >
      <div className="absolute -top-12 -right-12 h-44 w-44 rounded-full bg-accent/10 blur-2xl pointer-events-none" />
      <div className="absolute -bottom-10 -left-10 h-36 w-36 rounded-full bg-primary/10 blur-2xl pointer-events-none" />

      <div className="relative z-10 flex items-center justify-between">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-surface-border bg-surface-1/80 shadow-xs backdrop-blur-xs">
          <ServiceIcon name={iconName} size={24} className="text-accent" />
        </div>
        <span className="rounded-md border border-surface-border/60 bg-surface/80 px-2.5 py-1 text-[11px] font-mono uppercase tracking-wider text-muted">
          NOVA V2
        </span>
      </div>

      <div className="relative z-10 mt-auto flex flex-col gap-1">
        <span className="text-xs font-mono font-medium text-muted">PROJETO-NOVA-V2</span>
        <h3 className="line-clamp-2 text-base font-bold tracking-tight text-foreground sm:text-lg">
          {title}
        </h3>
      </div>
    </div>
  );
}
